package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// The model bench (axyn bench, #344) tests the models the user has in opencode on a fixed
// golden set, one task per step of the axyn process (plan, code, tests, fix), scores each
// answer only with deterministic checks (no model judges a model), classifies each model
// per step and routes it to the steps it does best. The engine then uses that routing:
// the planner gets the best planner, a code ticket the best coder, a tests ticket the best
// test writer.
//
// Lessons applied from the QA material the owner sent:
//   - a green build is not quality: a weakened or edited test vetoes the model for that
//     step, whatever its average (a grave case never dilutes into the mean);
//   - one success of a stochastic system proves little: every task runs --runs times and
//     the model is ranked by its success rate, with the continuous score kept as well;
//   - test the tester: the tests a model writes must kill injected bugs (mutants);
//   - bad-test signals count against the score: fixed sleeps, skips, no value assertion;
//   - a plan needs error and edge paths, not volume;
//   - realistic data (accents, empty input) and stable task ids;
//   - every round is a snapshot: a score drop from the previous round is flagged as drift,
//     and the profile goes stale when opencode, the axyn or the model list changes.

//go:embed all:bench
var benchFS embed.FS

const (
	roleplan  = "plan"
	roleCode  = "code"
	roleTests = "tests"
	roleFix   = "fix"
)

var roleNames = map[string]string{roleplan: "plano", roleCode: "código", roleTests: "testes", roleFix: "conserto"}

type benchTask struct {
	ID, Role, Fixture, Agent, Prompt string
}

var benchTasks = []benchTask{
	{"plan-001", roleplan, "plan", "axyn-plan",
		"Pedido do usuário: adicionar o comando `notas apagar ID`, que apaga a nota com aquele ID depois de pedir confirmação (s/N). Escreva a spec com FR-n e AC-n, cobrindo também os casos de erro e de borda, e os tickets, cada um citando os ACs que cumpre; grave tudo com a ferramenta axyn_plan."},
	{"code-001", roleCode, "code", "axyn-code",
		"Ticket: implemente a função Make em slug.go para o teste slug_test.go passar. Rode `go test ./...` até passar. Não altere o arquivo de teste."},
	{"tests-001", roleTests, "tests", "axyn-code",
		"Ticket: escreva testes em Go (price_test.go) para o pacote price: cubra todas as categorias de idade, as bordas (11, 12, 17, 18, 59, 60), a idade negativa e o desconto de estudante, comparando os valores exatos. Não altere price.go. Rode `go test ./...` até passar."},
	{"fix-001", roleFix, "fix", "axyn-code",
		"Ticket: o CI do projeto falha. Saída do `go vet ./... && go test ./...`:\n./cart.go:24:9: fmt.Sprintf format %d has arg fmt.Sprint(Total(items)) of wrong type string\nCorrija o código de produção para o vet e os testes passarem. Não altere os testes."},
}

// mutants are the bugs injected in the tests task: tests that let one survive missed it.
var mutants = [][2]string{
	{"case age < 12:", "case age <= 12:"},
	{"case age < 18:", "case age <= 18:"},
	{"case age < 60:", "case age <= 60:"},
	{"case age < 0:", "case age < -1:"},
	{"p = base / 2", "p = base / 3"},
	{"p = base * 80 / 100", "p = base * 85 / 100"},
	{"p = base * 60 / 100", "p = base * 65 / 100"},
	{"if student {", "if !student {"},
}

type benchRun struct {
	Score   int      `json:"score"`
	Pass    bool     `json:"pass"`
	Cheat   bool     `json:"cheat,omitempty"`
	Down    bool     `json:"down,omitempty"` // limit reached or offline: not a failure of the model
	Seconds float64  `json:"seconds"`
	Notes   []string `json:"notes,omitempty"`
}

type benchResult struct {
	Model string     `json:"model"`
	Task  string     `json:"task"`
	Role  string     `json:"role"`
	Runs  []benchRun `json:"runs"`
}

func (r benchResult) rate() float64 {
	if len(r.Runs) == 0 {
		return 0
	}
	n := 0
	for _, x := range r.Runs {
		if x.Pass && !x.Cheat {
			n++
		}
	}
	return float64(n) / float64(len(r.Runs))
}

func (r benchResult) score() int {
	if len(r.Runs) == 0 {
		return 0
	}
	t := 0
	for _, x := range r.Runs {
		if x.Cheat {
			return 0 // veto: one weakened test is never averaged away
		}
		t += x.Score
	}
	return t / len(r.Runs)
}

func (r benchResult) allDown() bool {
	for _, x := range r.Runs {
		if !x.Down {
			return false
		}
	}
	return len(r.Runs) > 0
}

func (r benchResult) cheated() bool {
	for _, x := range r.Runs {
		if x.Cheat {
			return true
		}
	}
	return false
}

type benchProfile struct {
	Date     time.Time           `json:"date"`
	Axyn     string              `json:"axyn"`
	Opencode string              `json:"opencode"`
	Models   []string            `json:"models"`
	Results  []benchResult       `json:"results"`
	Routing  map[string][]string `json:"routing"`
	Applied  bool                `json:"applied"`
	Pinned   map[string]string   `json:"pinned,omitempty"` // steps the user fixed by hand (axyn bench --set)
	// Contained are, per step, the models under the bar: still used, inside the guide.
	Contained map[string][]string `json:"contained,omitempty"`

	prevForReport *benchProfile // the round before, for the drift section; not saved
}

func benchDir() string {
	cfg := defaultConfigPath()
	if cfg == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(cfg), "bench")
}

func profilePathFile() string { return filepath.Join(benchDir(), "profile.json") }

func loadProfile() (*benchProfile, error) {
	b, err := os.ReadFile(profilePathFile())
	if err != nil {
		return nil, err
	}
	var p benchProfile
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func saveProfile(p *benchProfile) error {
	if err := os.MkdirAll(benchDir(), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(p, "", "  ")
	return os.WriteFile(profilePathFile(), b, 0o644)
}

// copyFixture writes the task's files into dir (the ".txt" keeps them out of the build).
func copyFixture(name, dir string) error {
	root := path.Join("bench", name)
	return fs.WalkDir(benchFS, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := benchFS.ReadFile(p)
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, filepath.FromSlash(strings.TrimSuffix(strings.TrimPrefix(p, root+"/"), ".txt")))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, b, 0o644)
	})
}

// sandbox is a fresh git repo with the task's fixture and the axyn configured in opencode.
func sandbox(task benchTask) (string, error) {
	dir, err := os.MkdirTemp("", "axyn-bench-"+task.ID+"-")
	if err != nil {
		return "", err
	}
	if err := copyFixture(task.Fixture, dir); err != nil {
		return dir, err
	}
	s := &mcpServer{dir: dir}
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "bench@axyn"}, {"config", "user.name", "axyn bench"}} {
		if out, err := s.git(args...); err != nil {
			return dir, fmt.Errorf("git %s: %s", args[0], out)
		}
	}
	if _, err := installOpencode(dir); err != nil {
		return dir, err
	}
	if out, err := s.git("add", "-A"); err != nil {
		return dir, fmt.Errorf("git add: %s", out)
	}
	if out, err := s.git("commit", "-qm", "base"); err != nil {
		return dir, fmt.Errorf("git commit: %s", out)
	}
	return dir, nil
}

func runIn(dir, command string) (string, error) {
	cmd, err := shellCommand(command)
	if err != nil {
		return "", err
	}
	cmd.Dir = dir
	b, err := cmd.CombinedOutput()
	return string(b), err
}

const benchCI = "go vet ./... && go test -count=1 ./..."

// changed lists the files the attempt changed (tracked and new).
func changed(dir string) []fileDiff {
	d, err := gitDiff(dir, "HEAD")
	if err != nil {
		return nil
	}
	return parseDiff(strings.NewReader(d))
}

func touched(files []fileDiff, name string) bool {
	for _, f := range files {
		if f.path == name {
			return true
		}
	}
	return false
}

func diffLines(files []fileDiff) int {
	n := 0
	for _, f := range files {
		n += len(f.added) + len(f.removed)
	}
	return n
}

func speedBonus(sec float64, max int) int {
	switch {
	case sec <= 60:
		return max
	case sec <= 180:
		return max / 2
	}
	return 0
}

var (
	goSleepRe     = regexp.MustCompile(`time\.Sleep\(`)
	goSkipRe      = regexp.MustCompile(`\bt\.Skip`)
	valueAssertRe = regexp.MustCompile(`(!=|==|reflect\.DeepEqual|cmp\.Diff|assert\.|require\.|errors\.Is)`)
)

// scoreTask judges one attempt with deterministic checks only.
func scoreTask(task benchTask, dir string, sec float64) benchRun {
	run := benchRun{Seconds: sec}
	note := func(f string, a ...any) { run.Notes = append(run.Notes, fmt.Sprintf(f, a...)) }
	files := changed(dir)
	if lf := checkLoosenedTests(files); len(lf) > 0 {
		run.Cheat = true
		for _, f := range lf {
			note("trapaça: %s", f.reason)
		}
		return run
	}
	switch task.Role {
	case roleplan:
		return scorePlan(dir, run)
	case roleCode, roleFix:
		test := map[string]string{roleCode: "slug_test.go", roleFix: "cart_test.go"}[task.Role]
		if touched(files, test) {
			run.Cheat = true
			note("trapaça: alterou o teste %s em vez do código", test)
			return run
		}
		if out, err := runIn(dir, benchCI); err != nil {
			note("o CI não passou: %s", lastLines(out, 3))
			return run
		}
		run.Pass = true
		run.Score = 70 + speedBonus(sec, 20)
		if diffLines(files) <= map[string]int{roleCode: 60, roleFix: 15}[task.Role] {
			run.Score += 10
		} else {
			note("diff maior que o necessário: %d linhas", diffLines(files))
		}
		return run
	case roleTests:
		return scoreTests(dir, files, run)
	}
	return run
}

func scoreTests(dir string, files []fileDiff, run benchRun) benchRun {
	note := func(f string, a ...any) { run.Notes = append(run.Notes, fmt.Sprintf(f, a...)) }
	if touched(files, "price.go") {
		run.Cheat = true
		note("trapaça: alterou o código de produção (price.go) numa tarefa de testes")
		return run
	}
	var added []string
	for _, f := range files {
		if isTestPath(f.path) {
			added = append(added, f.added...)
		}
	}
	if len(added) == 0 {
		note("nenhum teste escrito")
		return run
	}
	// Run three times: a test that passes once and fails later is flaky.
	out, err := runIn(dir, "go vet ./... && go test -count=3 -coverprofile=cover.out ./...")
	if err != nil {
		note("os testes não passam (em 3 execuções): %s", lastLines(out, 3))
		return run
	}
	run.Pass = true
	cov, _ := parseGoCover(out)
	run.Score = 20 + int(30*min(cov, 100)/100)
	killed := 0
	src := filepath.Join(dir, "price.go")
	orig, _ := os.ReadFile(src)
	for _, m := range mutants {
		mut := strings.Replace(string(orig), m[0], m[1], 1)
		if mut == string(orig) {
			continue
		}
		_ = os.WriteFile(src, []byte(mut), 0o644)
		if _, err := runIn(dir, "go test -count=1 ./..."); err != nil {
			killed++
		} else {
			note("bug que os testes não pegaram: %q virou %q", m[0], m[1])
		}
	}
	_ = os.WriteFile(src, orig, 0o644)
	run.Score += 40 * killed / len(mutants)
	text := strings.Join(added, "\n")
	quality := 10
	if goSleepRe.MatchString(text) {
		quality = 0
		note("sleep fixo no teste")
	}
	if goSkipRe.MatchString(text) {
		quality = 0
		note("teste pulado (t.Skip)")
	}
	if !valueAssertRe.MatchString(text) {
		quality = 0
		note("nenhuma comparação de valor nos testes")
	}
	run.Score += quality
	note("cobertura %.0f%%, %d de %d bugs injetados pegos", cov, killed, len(mutants))
	return run
}

var goCoverRe = regexp.MustCompile(`coverage: ([0-9.]+)% of statements`)

func parseGoCover(out string) (float64, bool) {
	m := goCoverRe.FindAllStringSubmatch(out, -1)
	if len(m) == 0 {
		return 0, false
	}
	var v float64
	_, err := fmt.Sscanf(m[len(m)-1][1], "%f", &v)
	return v, err == nil
}

var (
	specACRe = regexp.MustCompile(`(?m)\bAC-(\d+)\b[^\n]*`)
	errWord  = regexp.MustCompile(`(?i)(erro|inválid|inexistent|não existe|recus|falha|cancel|negad|sem permiss)`)
	edgeWord = regexp.MustCompile(`(?i)(vazi|limite|zero|máxim|mínim|duplic|primeir|últim|borda|nenhuma nota)`)
)

func scorePlan(dir string, run benchRun) benchRun {
	note := func(f string, a ...any) { run.Notes = append(run.Notes, fmt.Sprintf(f, a...)) }
	pl, _, err := (&mcpServer{dir: dir}).loadPlan()
	if err != nil || len(pl.Tickets) == 0 {
		note("o plano não foi gravado com a ferramenta axyn_plan")
		return run
	}
	spec, err := os.ReadFile(filepath.Join(dir, pl.Spec))
	if err != nil {
		note("a spec do plano não existe: %s", pl.Spec)
		return run
	}
	run.Pass, run.Score = true, 30
	acs := map[string]string{}
	for _, m := range specACRe.FindAllStringSubmatch(string(spec), -1) {
		id := "AC-" + m[1]
		if _, ok := acs[id]; !ok {
			acs[id] = strings.ToLower(strings.TrimSpace(m[0]))
		}
	}
	if len(acs) >= 3 {
		run.Score += 15
	} else {
		note("só %d critério(s) de aceite", len(acs))
	}
	errs, edges := 0, 0
	seen := map[string]bool{}
	dup := false
	for _, t := range acs {
		if errWord.MatchString(t) {
			errs++
		}
		if edgeWord.MatchString(t) {
			edges++
		}
		key := strings.Join(strings.Fields(t)[1:], " ")
		if seen[key] {
			dup = true
		}
		seen[key] = true
	}
	if errs >= 2 {
		run.Score += 20
	} else {
		note("só %d critério(s) de caso de erro (precisa de pelo menos 2)", errs)
	}
	if edges >= 1 {
		run.Score += 10
	} else {
		note("nenhum critério de borda (vazio, limite, duplicado)")
	}
	if !dup {
		run.Score += 10
	} else {
		note("critérios de aceite repetidos")
	}
	cites := true
	for _, t := range pl.Tickets {
		if len(t.ACs) == 0 {
			cites = false
		}
		for _, a := range t.ACs {
			if _, ok := acs[a]; !ok {
				cites = false
			}
		}
	}
	if cites && len(pl.Tickets) <= 6 {
		run.Score += 15
	} else {
		note("ticket sem AC, citando AC que não existe, ou plano com mais de 6 tickets")
	}
	return run
}

// route orders the models for each step: veto first, a minimum score (a test that only
// passes is not enough), then success rate, then score.
func route(results []benchResult, minScore int) (map[string][]string, map[string][]string) {
	routing, contained := map[string][]string{}, map[string][]string{}
	for _, role := range []string{roleplan, roleCode, roleTests, roleFix} {
		var good, rest []benchResult
		for _, r := range results {
			if r.Role != role {
				continue
			}
			if !r.cheated() && r.rate() > 0 && r.score() >= minScore {
				good = append(good, r)
			} else {
				rest = append(rest, r)
			}
		}
		order := func(rs []benchResult) {
			sort.SliceStable(rs, func(i, j int) bool {
				if rs[i].cheated() != rs[j].cheated() {
					return !rs[i].cheated()
				}
				if rs[i].rate() != rs[j].rate() {
					return rs[i].rate() > rs[j].rate()
				}
				return rs[i].score() > rs[j].score()
			})
		}
		order(good)
		order(rest)
		for _, r := range good {
			routing[role] = append(routing[role], r.Model)
		}
		// No model is thrown away (#344): the ones under the bar still work, after the good
		// ones, contained by the guide (see containment.go), so even a machine with only weak
		// models can use the axyn.
		for _, r := range rest {
			routing[role] = append(routing[role], r.Model)
			contained[role] = append(contained[role], r.Model)
		}
	}
	return routing, contained
}

func level(r benchResult) string {
	switch {
	case r.cheated():
		return "contido (vetado por trapaça)"
	case r.rate() >= 0.8 && r.score() >= 80:
		return "forte"
	case r.rate() >= 0.5 && r.score() >= 50:
		return "bom"
	}
	if r.allDown() {
		return "indisponível na avaliação"
	}
	return "contido"
}

func findResult(results []benchResult, model, task string) *benchResult {
	for i := range results {
		if results[i].Model == model && results[i].Task == task {
			return &results[i]
		}
	}
	return nil
}

// report renders the classification table, the routing and the drift against prev.
func report(p, prev *benchProfile) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# Avaliação dos modelos (axyn bench)\n\n")
	w("Rodada de %s, axyn %s, opencode %s. Notas de 0 a 100, só com verificações automáticas (nenhum modelo julga outro); taxa = tentativas aprovadas sobre o total.\n\n", p.Date.Format("2006-01-02 15:04"), p.Axyn, p.Opencode)
	w("| Modelo |")
	for _, t := range benchTasks {
		w(" %s |", roleNames[t.Role])
	}
	w("\n|---|")
	for range benchTasks {
		w("---|")
	}
	w("\n")
	for _, m := range p.Models {
		w("| %s |", m)
		for _, t := range benchTasks {
			r := findResult(p.Results, m, t.ID)
			if r == nil {
				w(" — |")
				continue
			}
			w(" %d, taxa %.0f%%, %s |", r.score(), 100*r.rate(), level(*r))
		}
		w("\n")
	}
	w("\n## Para que o axyn vai usar cada modelo\n\n")
	for _, role := range []string{roleplan, roleCode, roleTests, roleFix} {
		if pin := p.Pinned[role]; pin != "" {
			w("- **%s:** %s (fixado por você)\n", roleNames[role], pin)
		} else if ms := p.Routing[role]; len(ms) > 0 {
			var names []string
			for _, m := range ms {
				if inList(p.Contained[role], m) {
					m += " (contido)"
				}
				names = append(names, m)
			}
			w("- **%s:** %s\n", roleNames[role], strings.Join(names, ", "))
		} else {
			w("- **%s:** nenhum modelo avaliado; o axyn usa a escada do `axyn model`\n", roleNames[role])
		}
	}
	w("\nModelos marcados como contidos continuam sendo usados, depois dos outros, dentro da guia do axyn: escopo travado nos arquivos do ticket, testes existentes travados, diff de no máximo %d linhas e instruções estritas. Nenhum modelo é descartado.\n", containedMaxLines)
	if prev != nil {
		var drift []string
		for _, r := range p.Results {
			if o := findResult(prev.Results, r.Model, r.Task); o != nil && o.score()-r.score() >= 15 {
				drift = append(drift, fmt.Sprintf("%s em %s: %d → %d", r.Model, roleNames[r.Role], o.score(), r.score()))
			}
		}
		if len(drift) > 0 {
			w("\n## Pioraram desde a última rodada\n\n")
			for _, d := range drift {
				w("- %s\n", d)
			}
		}
	}
	w("\n## Detalhes de cada tentativa\n\nO código que o modelo escreveu em cada tentativa fica na pasta `bench-%s/` ao lado deste relatório, um arquivo `.diff` por tentativa; a saída completa dos modelos, em `bench-%s.log`.\n\n", p.Date.Format("20060102-150405"), p.Date.Format("20060102-150405"))
	for _, r := range p.Results {
		for i, x := range r.Runs {
			w("- %s, %s (tentativa %d): nota %d em %.0fs", r.Model, r.Task, i+1, x.Score, x.Seconds)
			if len(x.Notes) > 0 {
				w(": %s", strings.Join(x.Notes, "; "))
			}
			w("\n")
		}
	}
	return b.String()
}

// stale says why the applied profile should be measured again, or "".
func (p *benchProfile) stale(models []string) string {
	switch {
	case time.Since(p.Date) > 30*24*time.Hour:
		return "a avaliação tem mais de 30 dias"
	case p.Axyn != version:
		return "o axyn mudou de versão desde a avaliação"
	}
	if v := toolVersion("opencode", "--version"); v != "não instalado" && p.Opencode != "" && v != p.Opencode {
		return "o opencode mudou de versão desde a avaliação"
	}
	known := map[string]bool{}
	for _, m := range p.Models {
		known[m] = true
	}
	for _, r := range p.Results {
		if r.allDown() {
			known[r.Model] = false
		}
	}
	for _, m := range models {
		if !known[m] {
			return "há modelos novos que ainda não foram avaliados (" + m + ")"
		}
	}
	return ""
}

func runBenchCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn bench", flag.ContinueOnError)
	fs.SetOutput(stderr)
	only := fs.String("models", "", "modelos a avaliar, separados por vírgula (padrão: os gratuitos do opencode e os da sua escada)")
	all := fs.Bool("all", false, "avalia todos os modelos do opencode, inclusive os pagos")
	tasks := fs.String("tasks", "", "tarefas, separadas por vírgula: plan, code, tests, fix (padrão: todas)")
	runs := fs.Int("runs", 1, "quantas vezes cada tarefa roda por modelo (modelos variam: a taxa de sucesso pesa mais que uma vez)")
	timeout := fs.Duration("timeout", 10*time.Minute, "tempo máximo de cada tentativa")
	ask := fs.Bool("ask", false, "pergunta antes de aplicar o resultado (o padrão é aplicar sozinho)")
	parallel := fs.Int("parallel", 0, "quantos modelos avaliar ao mesmo tempo (padrão: o que a máquina aguenta, 1 com menos de 8 GB de RAM ou até 4 núcleos)")
	minScore := fs.Int("min-score", 50, "nota mínima (0 a 100) para um modelo receber uma etapa; trapaça é veto em qualquer nota")
	show := fs.Bool("show", false, "só mostra a última avaliação")
	applyFlag := fs.Bool("apply", false, "aplica a última avaliação")
	off := fs.Bool("off", false, "deixa de usar a avaliação (volta à escada do axyn model)")
	set := fs.String("set", "", "fixa à mão o modelo de uma etapa, por exemplo plano=MODELO (etapas: plano, codigo, testes, conserto); etapa= sem modelo desfaz")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *set != "" {
		return benchSet(*set, stdout, stderr)
	}
	if *applyFlag || *off {
		p, err := loadProfile()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "axyn bench: nenhuma avaliação ainda; rode: axyn bench")
			return exitFail
		}
		p.Applied = *applyFlag
		if err := saveProfile(p); err != nil {
			_, _ = fmt.Fprintf(stderr, "axyn bench: %v\n", err)
			return exitFail
		}
		if *applyFlag {
			_, _ = fmt.Fprintln(stdout, "aplicada: o axyn escolhe o modelo de cada etapa pela avaliação de "+p.Date.Format("2006-01-02"))
		} else {
			_, _ = fmt.Fprintln(stdout, "desligada: o axyn volta a usar a escada do axyn model")
		}
		return exitOK
	}
	if *show {
		p, err := loadProfile()
		if err != nil {
			_, _ = fmt.Fprintln(stdout, "nenhuma avaliação ainda; rode: axyn bench")
			return exitOK
		}
		_, _ = fmt.Fprintln(stdout, report(p, nil))
		return exitOK
	}
	for _, need := range []string{"go", "git"} {
		if _, err := lookPath(need); err != nil {
			_, _ = fmt.Fprintf(stderr, "axyn bench: as tarefas da avaliação são em Go e precisam de %s; para instalar: %s\n", need, strings.Join(installCommands([]string{need}, hostOS), " && "))
			return exitFail
		}
	}
	models, err := benchModels(*only, *all)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn bench: %v\n", err)
		return exitFail
	}
	chosen := benchTasks
	if *tasks != "" {
		chosen = nil
		for _, t := range benchTasks {
			if strings.Contains(","+*tasks+",", ","+t.Role+",") {
				chosen = append(chosen, t)
			}
		}
	}
	p := benchCore(models, chosen, *runs, *parallel, *timeout, *minScore, stdout, nil)
	prev := p.prevForReport
	text := report(p, prev)
	mdPath := filepath.Join(benchDir(), "bench-"+p.Date.Format("20060102-150405")+".md")
	_ = os.WriteFile(mdPath, []byte(text), 0o644)
	_, _ = fmt.Fprintf(stdout, "\n%s\nrelatório completo: %s\n", text, mdPath)
	apply := true
	if *ask {
		_, _ = fmt.Fprint(stdout, "\nUsar esta recomendação nas próximas execuções do axyn? (s/N): ")
		line, _ := readLine(stdin)
		a := strings.ToLower(strings.TrimSpace(line))
		apply = a == "s" || a == "sim" || a == "y" || a == "yes"
	}
	p.Applied = apply
	if err := saveProfile(p); err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn bench: %v\n", err)
		return exitFail
	}
	if apply {
		_, _ = fmt.Fprintln(stdout, "aplicada: o axyn passa a escolher o modelo de cada etapa por esta avaliação (para voltar à escada do axyn model: axyn bench --off)")
	} else {
		_, _ = fmt.Fprintln(stdout, "não aplicada; para aplicar depois: axyn bench --apply")
	}
	return exitOK
}

// benchCore runs the golden set: each model in its own worker (parallel at a time), each
// task runs times; the profile is saved after every result, so an interruption keeps what ran.
func benchCore(models []string, chosen []benchTask, runs, parallel int, timeout time.Duration, minScore int, out io.Writer, progress func(done, total int, detail string)) *benchProfile {
	prev, _ := loadProfile()
	p := &benchProfile{Date: time.Now(), Axyn: version, Opencode: toolVersion("opencode", "--version"), Models: models, prevForReport: prev}
	if prev != nil {
		p.Pinned = prev.Pinned // a choice by hand survives every new evaluation
		// Evaluating only some tasks or models keeps the other results of the last round:
		// axyn bench --tasks plan redoes the plan step, not the whole battery.
		run := map[string]bool{}
		for _, m := range models {
			for _, t := range chosen {
				run[m+"|"+t.ID] = true
			}
		}
		for _, r := range prev.Results {
			if !run[r.Model+"|"+r.Task] {
				p.Results = append(p.Results, r)
			}
		}
		for _, m := range prev.Models {
			if !inList(p.Models, m) {
				p.Models = append(p.Models, m)
			}
		}
		p.Routing, p.Contained = route(p.Results, minScore)
	}
	if runs < 1 {
		runs = 1
	}
	if parallel < 1 {
		parallel = autoParallel()
	}
	total := len(models) * len(chosen) * runs
	benchRoundDir = filepath.Join(benchDir(), "bench-"+p.Date.Format("20060102-150405"))
	defer func() { benchRoundDir = "" }()
	est := time.Duration(total) * 3 * time.Minute / time.Duration(parallel) // about 3 minutes per attempt
	_, _ = fmt.Fprintf(out, "avaliando %d modelo(s) em %d tarefa(s), %d vez(es) cada: %d tentativas, %d modelo(s) por vez, por volta de %s (depende da máquina e da velocidade dos modelos; o que já rodou fica salvo)\n", len(models), len(chosen), runs, total, parallel, clock(est))
	_ = os.MkdirAll(benchDir(), 0o755)
	var log io.Writer = io.Discard
	if logf, err := os.Create(filepath.Join(benchDir(), "bench-"+p.Date.Format("20060102-150405")+".log")); err == nil {
		defer func() { _ = logf.Close() }()
		log = &syncWriter{w: logf}
	}
	var mu sync.Mutex
	n := 0
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup
	for _, m := range models {
		wg.Add(1)
		sem <- struct{}{}
		go func(m string) {
			defer wg.Done()
			defer func() { <-sem }()
			for _, t := range chosen {
				res := benchResult{Model: m, Task: t.ID, Role: t.Role}
				for i := 0; i < runs; i++ {
					if progress != nil {
						mu.Lock()
						progress(n, total, fmt.Sprintf("%s, %s", m, roleNames[t.Role]))
						mu.Unlock()
					}
					run := benchOnce(t, m, timeout, log)
					res.Runs = append(res.Runs, run)
					verdict := "reprovado"
					switch {
					case run.Down:
						verdict = "indisponível (limite ou fora do ar; avaliado de novo na próxima rodada)"
					case run.Cheat:
						verdict = "trapaça (veto)"
					case run.Pass:
						verdict = fmt.Sprintf("nota %d", run.Score)
					}
					mu.Lock()
					n++
					_, _ = fmt.Fprintf(out, "[%d/%d] %s, %s: %s (%.0fs)\n", n, total, m, roleNames[t.Role], verdict, run.Seconds)
					if progress != nil {
						progress(n, total, fmt.Sprintf("%s, %s: %s", m, roleNames[t.Role], verdict))
					}
					mu.Unlock()
				}
				mu.Lock()
				p.Results = append(p.Results, res)
				p.Routing, p.Contained = route(p.Results, minScore)
				_ = saveProfile(&benchProfile{Date: p.Date, Axyn: p.Axyn, Opencode: p.Opencode, Models: p.Models, Results: p.Results, Routing: p.Routing, Contained: p.Contained, Pinned: p.Pinned, Applied: prev != nil && prev.Applied})
				mu.Unlock()
			}
		}(m)
	}
	wg.Wait()
	sort.SliceStable(p.Results, func(i, j int) bool { return p.Results[i].Model < p.Results[j].Model })
	return p
}

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(b)
}

func readLine(r io.Reader) (string, error) {
	var b []byte
	one := make([]byte, 1)
	for {
		n, err := r.Read(one)
		if n == 1 {
			if one[0] == '\n' {
				return string(b), nil
			}
			b = append(b, one[0])
		}
		if err != nil {
			return string(b), err
		}
	}
}

// benchOnce runs one attempt of a task on a model, in a fresh sandbox.
func benchOnce(t benchTask, model string, timeout time.Duration, log io.Writer) benchRun {
	dir, err := sandbox(t)
	if dir != "" {
		defer func() { _ = os.RemoveAll(dir) }()
	}
	// What the model wrote stays for the audit (was the score fair? should the bench
	// change?): one .diff per attempt in the round's folder, named in the report.
	defer func() {
		if benchRoundDir == "" || dir == "" {
			return
		}
		if d, err := gitDiff(dir, "HEAD"); err == nil {
			_ = os.MkdirAll(benchRoundDir, 0o755)
			name := fmt.Sprintf("%s-%s-%d.diff", strings.NewReplacer("/", "_", ":", "_", "\\", "_").Replace(model), t.ID, time.Now().UnixNano()%1e6)
			_ = os.WriteFile(filepath.Join(benchRoundDir, name), []byte(d), 0o644)
		}
	}()
	if err != nil {
		return benchRun{Notes: []string{"não consegui preparar a tarefa: " + err.Error()}}
	}
	_, _ = fmt.Fprintf(log, "\n=== %s em %s (%s)\n", model, t.ID, dir)
	start := time.Now()
	out, err := callAgent(dir, t.Agent, model, t.Prompt, timeout, log)
	sec := time.Since(start).Seconds()
	var fe fatalErr
	if errors.As(err, &fe) {
		return benchRun{Seconds: sec, Notes: []string{err.Error()}}
	}
	if why := downReason(out, len(changed(dir)) > 0); why != "" {
		// Down is not bad: the model is measured again on the next round.
		return benchRun{Seconds: sec, Down: true, Notes: []string{"indisponível durante a avaliação: " + why}}
	}
	run := scoreTask(t, dir, sec)
	_, _ = fmt.Fprintf(log, "--- nota %d, passou %v, trapaça %v: %s\n", run.Score, run.Pass, run.Cheat, strings.Join(run.Notes, "; "))
	return run
}

// benchModels is the list to evaluate: --models, else the free ones (or all) plus the ladder.
func benchModels(only string, all bool) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	add := func(m string) {
		m = strings.TrimSpace(m)
		if m != "" && m != placeholder && !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	if only != "" {
		for _, m := range strings.Split(only, ",") {
			add(m)
		}
		return out, nil
	}
	if ms, err := loadModels(defaultConfigPath()); err == nil {
		for _, m := range ms {
			add(m.ID)
		}
	}
	list, err := opencodeModels(all)
	if err != nil && len(out) == 0 {
		return nil, err
	}
	for _, m := range list {
		add(m)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("nenhum modelo para avaliar; veja os modelos com: opencode models")
	}
	return out, nil
}

// routed is the model the applied bench picked for a role, the first one not in skip.
func routed(role string) []string {
	p, err := loadProfile()
	if err != nil || !p.Applied {
		return nil
	}
	if m := p.Pinned[role]; m != "" {
		out := []string{m}
		for _, x := range p.Routing[role] {
			if x != m {
				out = append(out, x)
			}
		}
		return out
	}
	return p.Routing[role]
}

var _ = exec.Command // the sandbox runs commands through shellCommand

// autoBench is the default path (#344): with no evaluation yet, or a stale one, the run
// evaluates every free model on the machine before planning and applies the result, so the
// user never has to pick models. AXYN_BENCH=off, a forced --model or axyn bench --off skip it.
func (r *runner) autoBench() {
	if os.Getenv("AXYN_BENCH") == "off" || r.st.Opts.Model != "" || os.Getenv("AXYN_OPENCODE") != "" {
		return
	}
	if _, err := lookPath("go"); err != nil {
		_, _ = fmt.Fprintln(r.log, "avaliação dos modelos pulada: as tarefas são em Go, e o Go não está instalado")
		return
	}
	models, err := benchModels("", false)
	if err != nil || len(models) == 0 {
		return
	}
	why := "primeira execução: ainda não há avaliação"
	if p, err := loadProfile(); err == nil {
		if !p.Applied {
			return // the user turned it off
		}
		if why = p.stale(models); why == "" {
			return
		}
	}
	r.set("avaliando modelos")
	_, _ = fmt.Fprintf(r.log, "avaliando os modelos gratuitos da máquina antes de começar (%s); o axyn escolhe sozinho o melhor modelo para cada etapa\n", why)
	p := benchCore(models, benchTasks, 1, 0, 10*time.Minute, 50, r.log, func(done, total int, detail string) {
		r.st.Done, r.st.Of, r.st.Detail = done, total, detail
		_ = saveRun(r.s.dir, r.st)
	})
	p.Applied = true
	_ = saveProfile(p)
	text := report(p, p.prevForReport)
	_ = os.WriteFile(filepath.Join(benchDir(), "bench-"+p.Date.Format("20060102-150405")+".md"), []byte(text), 0o644)
	_, _ = fmt.Fprintln(r.log, text)
}

var roleAliases = map[string]string{"plano": roleplan, "plan": roleplan, "codigo": roleCode, "código": roleCode, "code": roleCode,
	"testes": roleTests, "tests": roleTests, "conserto": roleFix, "fix": roleFix}

// benchSet pins (or unpins) the model of one step by hand, on top of the evaluation.
func benchSet(arg string, stdout, stderr io.Writer) int {
	k, m, ok := strings.Cut(arg, "=")
	role := roleAliases[strings.ToLower(strings.TrimSpace(k))]
	if !ok || role == "" {
		_, _ = fmt.Fprintln(stderr, "axyn bench: use --set ETAPA=MODELO, com ETAPA plano, codigo, testes ou conserto")
		return exitUsage
	}
	p, err := loadProfile()
	if err != nil {
		p = &benchProfile{Date: time.Now(), Axyn: version, Routing: map[string][]string{}}
	}
	if p.Pinned == nil {
		p.Pinned = map[string]string{}
	}
	m = strings.TrimSpace(m)
	if m == "" {
		delete(p.Pinned, role)
		_, _ = fmt.Fprintf(stdout, "%s: volta a seguir a avaliação\n", roleNames[role])
	} else {
		p.Pinned[role] = m
		_, _ = fmt.Fprintf(stdout, "%s: fixado em %s (vale mesmo depois de novas avaliações; para desfazer: axyn bench --set %s=)\n", roleNames[role], m, strings.ToLower(k))
	}
	p.Applied = true
	if err := saveProfile(p); err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn bench: %v\n", err)
		return exitFail
	}
	return exitOK
}

// benchRoundDir is the folder of the running round, where each attempt's diff is kept.
var benchRoundDir string

// autoParallel is how many models to evaluate at once on this machine: every attempt runs
// opencode and compiles Go, so a small machine (the owner's i3 with 4 GB) does one at a
// time, which is faster there than two fighting for memory.
func autoParallel() int {
	mem := totalMemory()
	cpus := runtime.NumCPU()
	switch {
	case mem > 0 && mem < 8<<30, cpus <= 4:
		return 1
	case mem >= 16<<30 && cpus >= 8:
		return 3
	}
	return 2
}
