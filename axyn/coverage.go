package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Coverage without hand edits (#334). Before the first ticket, the axyn measures the
// project's current coverage (with the user's own tests) and keeps it as the floor: the
// project may never go below it, and every delivered ticket raises the floor, in the
// ticket's own commit, up to the goal of the Makefile (COVERAGE_MIN, 80 by default).
// New code always needs tests: a ticket that changes code without any test is rejected,
// and in Go at least patchMin% of the added lines must be covered.

const patchMin = 80

// lastCoverage is the total coverage the last gate run printed, or -1.
var lastCoverage float64 = -1

var (
	coverageOut  = regexp.MustCompile(`cobertura: ([0-9]+(?:\.[0-9]+)?)%`)
	coverageLine = regexp.MustCompile(`(?m)^COVERAGE_MIN \?= (\d+)$`)
	goalNote     = regexp.MustCompile(`# axyn: o mínimo de cobertura sobe sozinho a cada ticket entregue, até a meta de (\d+)%`)
)

func parseCoverage(out string) (float64, bool) {
	m := coverageOut.FindAllStringSubmatch(out, -1)
	if len(m) == 0 {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[len(m)-1][1], 64)
	return v, err == nil
}

// makefileGoal is the COVERAGE_MIN of the project's Makefile.
func makefileGoal(dir string) (int, bool) {
	b, err := os.ReadFile(filepath.Join(dir, "Makefile"))
	if err != nil {
		return 0, false
	}
	if m := goalNote.FindStringSubmatch(string(b)); m != nil {
		n, err := strconv.Atoi(m[1])
		return n, err == nil
	}
	m := coverageLine.FindStringSubmatch(string(b))
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

// coverageFloor is the minimum the gates use instead of the Makefile's, once measured.
func coverageFloor(dir string) (int, bool) {
	pl, _, err := (&mcpServer{dir: dir}).loadPlan()
	if err != nil || !pl.CoverageMeasured || pl.CoverageFloor >= pl.CoverageGoal {
		return 0, false
	}
	return pl.CoverageFloor, true
}

func isTestPath(p string) bool {
	l := strings.ToLower(filepath.ToSlash(p))
	name := l[strings.LastIndex(l, "/")+1:]
	return strings.Contains(name, "test") || strings.Contains(name, ".spec.") ||
		strings.Contains(l, "/tests/") || strings.HasPrefix(l, "tests/") || strings.Contains(l, "__tests__/")
}

// checkCodeHasTests rejects a diff that adds code without adding or changing any test.
func checkCodeHasTests(files []fileDiff) []finding {
	var code []string
	tested := false
	for _, f := range files {
		if f.deleted || len(f.added) == 0 || !sourceExt[filepath.Ext(f.path)] {
			continue
		}
		if isTestPath(f.path) {
			tested = true
		} else {
			code = append(code, f.path)
		}
	}
	if len(code) == 0 || tested {
		return nil
	}
	return []finding{{"tests", fmt.Sprintf("o ticket muda código (%s) sem nenhum teste novo ou alterado; todo código novo precisa de teste", strings.Join(code, ", "))}}
}

// profilePath is where the template's Makefile writes the Go coverage profile.
func profilePath(dir string) string {
	if out, err := (&mcpServer{dir: dir}).git("rev-parse", "--git-path", "sdd-out"); err == nil && out != "" {
		p := out
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		return filepath.Join(p, "coverage.out")
	}
	return filepath.Join(dir, "coverage.out")
}

type block struct {
	file              string
	start, end, n, ct int
}

func readProfile(path string) []block {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	var out []block
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		// file.go:12.34,15.2 3 1
		l := sc.Text()
		colon := strings.LastIndex(l, ":")
		if colon < 0 || strings.HasPrefix(l, "mode:") {
			continue
		}
		fields := strings.Fields(l[colon+1:])
		if len(fields) != 3 {
			continue
		}
		rng := strings.Split(fields[0], ",")
		if len(rng) != 2 {
			continue
		}
		s, _ := strconv.Atoi(strings.Split(rng[0], ".")[0])
		e, _ := strconv.Atoi(strings.Split(rng[1], ".")[0])
		n, _ := strconv.Atoi(fields[1])
		c, _ := strconv.Atoi(fields[2])
		out = append(out, block{l[:colon], s, e, n, c})
	}
	return out
}

// checkPatchCoverage: in Go, the added lines of the ticket must be covered by tests.
func checkPatchCoverage(dir string, files []fileDiff, since time.Time) []finding {
	path := profilePath(dir)
	if st, err := os.Stat(path); err != nil || st.ModTime().Before(since) {
		return nil // no fresh profile: the stack measures coverage its own way
	}
	blocks := readProfile(path)
	covered, total := 0, 0
	missing := map[string][]int{}
	for _, f := range files {
		if f.deleted || filepath.Ext(f.path) != ".go" || isTestPath(f.path) {
			continue
		}
		for _, ln := range f.addedAt {
			seen, hit := false, false
			for _, b := range blocks {
				if (b.file == f.path || strings.HasSuffix(b.file, "/"+f.path)) && ln >= b.start && ln <= b.end {
					seen = true
					hit = hit || b.ct > 0
				}
			}
			if !seen {
				continue // not a statement (comment, declaration, blank line)
			}
			total++
			if hit {
				covered++
			} else {
				missing[f.path] = append(missing[f.path], ln)
			}
		}
	}
	if total == 0 || covered*100 >= patchMin*total {
		return nil
	}
	var where []string
	for p, ls := range missing {
		var s []string
		for _, l := range ls {
			s = append(s, strconv.Itoa(l))
		}
		where = append(where, p+" linhas "+strings.Join(s, ","))
	}
	sort.Strings(where)
	return []finding{{"coverage", fmt.Sprintf("o código novo do ticket tem %d%% das linhas cobertas por testes (mínimo %d%%); sem teste: %s", covered*100/total, patchMin, strings.Join(where, "; "))}}
}

// measureCoverage runs the CI once on the base, before the first ticket, to learn the
// project's coverage with its own tests. It returns what to tell the user.
func (r *runner) measureCoverage() string {
	pl, path, err := r.s.loadPlan()
	// Only a real measure ends this: v1.18.0 marked the plan as checked before a failed
	// measure, and v1.18.1 never measured again (#342).
	if err != nil || pl.CoverageMeasured || strings.TrimSpace(r.s.ci) == "" {
		return ""
	}
	goal, ok := makefileGoal(r.s.dir)
	if !ok {
		pl.CoverageChecked = true // no coverage minimum in this project
		_ = r.s.savePlan(pl, path)
		return ""
	}
	run := func(command string) (string, error) {
		cmd, err := shellCommand(command)
		if err != nil {
			return "", err
		}
		cmd.Dir = r.s.dir
		if cmd.Env == nil {
			cmd.Env = os.Environ()
		}
		cmd.Env = append(cmd.Env, "COVERAGE_MIN=0")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	out, ciErr := run(r.s.ci)
	c, found := parseCoverage(out)
	var notes []string
	if ciErr != nil {
		// The project's CI already fails before any ticket (often lint): the first ticket
		// has to fix it too, and the model is told so.
		pl.BaseErrors = ciErrors(out, 6)
		if len(pl.BaseErrors) > 0 {
			notes = append(notes, "o make ci já falha na branch base, antes de qualquer ticket; o primeiro ticket precisa corrigir também:\n  "+strings.Join(pl.BaseErrors, "\n  "))
		}
	}
	if !found && r.s.ci == defaultCICmd && testTarget(r.s.dir) {
		out, _ = run("make test") // the lint failed first: measure with the tests alone
		c, found = parseCoverage(out)
	}
	if !found {
		_ = r.s.savePlan(pl, path)
		return strings.Join(append(notes, "não consegui medir a cobertura atual do projeto (os testes não rodaram); vou tentar de novo na próxima execução"), "\n")
	}
	pl.CoverageChecked = true
	pl.CoverageMeasured, pl.CoverageGoal, pl.CoverageNow = true, goal, c
	pl.CoverageFloor = min(int(math.Floor(c)), goal)
	pl.CoverageGaps = coverageGaps(r.s.dir, goal)
	_ = r.s.savePlan(pl, path)
	prefix := strings.Join(notes, "\n")
	if prefix != "" {
		prefix += "\n"
	}
	if pl.CoverageFloor >= goal {
		return prefix + fmt.Sprintf("cobertura atual do projeto: %.1f%% (meta %d%%): já está na meta", c, goal)
	}
	return prefix + fmt.Sprintf("cobertura atual do projeto: %.1f%% (meta %d%%). O axyn não deixa cair abaixo disso, exige teste em todo código novo e sobe o mínimo sozinho a cada ticket entregue, até a meta", c, goal)
}

// raiseCoverage runs before a green ticket's commit: the floor goes up to the coverage the
// ticket reached, and the Makefile's COVERAGE_MIN follows, so the CI of the PR enforces it.
func (r *runner) raiseCoverage() string {
	pl, path, err := r.s.loadPlan()
	if err != nil || !pl.CoverageMeasured || lastCoverage < 0 {
		return ""
	}
	nf := min(int(math.Floor(lastCoverage)), pl.CoverageGoal)
	if nf > pl.CoverageFloor {
		pl.CoverageFloor = nf
		_ = r.s.savePlan(pl, path)
	}
	mk := filepath.Join(r.s.dir, "Makefile")
	b, err := os.ReadFile(mk)
	if err != nil {
		return ""
	}
	text := string(b)
	m := coverageLine.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	cur, _ := strconv.Atoi(m[1])
	if cur == pl.CoverageFloor {
		return ""
	}
	note := fmt.Sprintf("# axyn: o mínimo de cobertura sobe sozinho a cada ticket entregue, até a meta de %d%%.\n", pl.CoverageGoal)
	if !strings.Contains(text, "# axyn: o mínimo de cobertura") {
		text = coverageLine.ReplaceAllString(text, strings.TrimSuffix(note, "\n")+"\nCOVERAGE_MIN ?= $1")
	}
	text = coverageLine.ReplaceAllString(text, fmt.Sprintf("COVERAGE_MIN ?= %d", pl.CoverageFloor))
	if err := os.WriteFile(mk, []byte(text), 0o644); err != nil {
		return ""
	}
	return fmt.Sprintf("mínimo de cobertura no Makefile: %d%% (era %d%%; meta %d%%)", pl.CoverageFloor, cur, pl.CoverageGoal)
}

// coverageGaps lists the files under the goal, least covered first (Go profile only).
func coverageGaps(dir string, goal int) []string {
	type fc struct {
		file      string
		cov, stmt int
	}
	per := map[string]*fc{}
	for _, b := range readProfile(profilePath(dir)) {
		name := b.file
		if mod := goModule(dir); mod != "" {
			name = strings.TrimPrefix(name, mod+"/")
		}
		f := per[name]
		if f == nil {
			f = &fc{file: name}
			per[name] = f
		}
		f.stmt += b.n
		if b.ct > 0 {
			f.cov += b.n
		}
	}
	var list []*fc
	for _, f := range per {
		if f.stmt > 0 && f.cov*100 < goal*f.stmt {
			list = append(list, f)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		pi, pj := list[i].cov*1000/list[i].stmt, list[j].cov*1000/list[j].stmt
		if pi != pj {
			return pi < pj
		}
		return list[i].file < list[j].file
	})
	var out []string
	for i, f := range list {
		if i == 8 {
			out = append(out, fmt.Sprintf("e mais %d arquivo(s)", len(list)-8))
			break
		}
		out = append(out, fmt.Sprintf("%s: %d%% (%d de %d instruções sem teste)", f.file, f.cov*100/f.stmt, f.stmt-f.cov, f.stmt))
	}
	return out
}

func goModule(dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(l, "module "))
		}
	}
	return ""
}

// coverageQuestion is asked once, before the first ticket, when the project is under the
// goal: who writes the missing tests. AXYN_COVERAGE=auto|manual answers it in advance.
func (r *runner) coverageQuestion() string {
	pl, path, err := r.s.loadPlan()
	if err != nil || !pl.CoverageMeasured || pl.CoverageFloor >= pl.CoverageGoal || pl.CoverageChoice != "" {
		return ""
	}
	if c := os.Getenv("AXYN_COVERAGE"); c == "auto" || c == "manual" {
		if err := applyCoverageChoice(r.s, c); err == nil {
			return ""
		}
	}
	_ = path
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	if pl.CoverageNow == 0 {
		w("O projeto não tem testes que cubram o código (cobertura 0%%), e a meta é %d%%.\n", pl.CoverageGoal)
	} else {
		w("Os testes do projeto cobrem %.1f%% do código; a meta é %d%% (faltam %.1f pontos).\n", pl.CoverageNow, pl.CoverageGoal, float64(pl.CoverageGoal)-pl.CoverageNow)
	}
	if len(pl.CoverageGaps) > 0 {
		w("\nOnde falta cobertura:\n")
		for _, g := range pl.CoverageGaps {
			w("  %s\n", g)
		}
	}
	w("\nAntes de começar o seu pedido, escolha quem escreve os testes que faltam (no terminal, na raiz do projeto):\n")
	w("  1. O axyn escreve, num ticket antes dos outros, e depois segue com o seu pedido:\n       axyn coverage auto\n")
	w("  2. Você escreve, quando quiser; o axyn segue já com o seu pedido, sem deixar a cobertura cair, exigindo teste em todo código novo e subindo o mínimo a cada ticket:\n       axyn coverage manual\n")
	w("\nPara não ver esta pergunta nos próximos projetos, defina AXYN_COVERAGE=auto (ou manual).")
	return b.String()
}

// applyCoverageChoice records the choice; auto puts a tests ticket before the others.
func applyCoverageChoice(s *mcpServer, choice string) error {
	pl, path, err := s.loadPlan()
	if err != nil {
		return err
	}
	pl.CoverageChoice = choice
	// Attempts made before the coverage was sorted out fought a goal no ticket could meet:
	// they stay in the history (earlier) and the open tickets start the ladder again.
	for i := range pl.Tickets {
		if !pl.Tickets[i].Done {
			pl.Tickets[i].Earlier = append(pl.Tickets[i].Earlier, pl.Tickets[i].Attempts...)
			pl.Tickets[i].Attempts = nil
		}
	}
	if choice == "auto" {
		id := 0
		for _, t := range pl.Tickets {
			id = max(id, t.ID)
		}
		body := fmt.Sprintf("Escreva testes para o código que já existe, até a cobertura do projeto chegar a %d%% (hoje: %.1f%%). Não mude o código de produção, só acrescente testes. Comece pelos arquivos menos cobertos:\n- %s",
			pl.CoverageGoal, pl.CoverageNow, strings.Join(pl.CoverageGaps, "\n- "))
		t := ticket{ID: id + 1, Title: fmt.Sprintf("Testes para a cobertura chegar a %d%%", pl.CoverageGoal), Body: body}
		pl.Tickets = append([]ticket{t}, pl.Tickets...)
	}
	return s.savePlan(pl, path)
}

func runCoverageCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn coverage", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "raiz do repositório")
	noResume := fs.Bool("no-resume", false, "só grava a escolha, sem retomar a execução")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	s := &mcpServer{dir: *dir}
	pl, _, err := s.loadPlan()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn coverage: %v\n", err)
		return exitFail
	}
	choice := fs.Arg(0)
	if choice == "" {
		if !pl.CoverageMeasured {
			_, _ = fmt.Fprintln(stdout, "a cobertura ainda não foi medida; o axyn mede antes do primeiro ticket")
			return exitOK
		}
		_, _ = fmt.Fprintf(stdout, "cobertura medida: %.1f%%; mínimo atual: %d%%; meta: %d%%; quem escreve os testes que faltam: %s\n", pl.CoverageNow, pl.CoverageFloor, pl.CoverageGoal, map[string]string{"": "ainda não escolhido", "auto": "o axyn", "manual": "você"}[pl.CoverageChoice])
		for _, g := range pl.CoverageGaps {
			_, _ = fmt.Fprintf(stdout, "  %s\n", g)
		}
		return exitOK
	}
	if choice != "auto" && choice != "manual" {
		_, _ = fmt.Fprintln(stderr, "axyn coverage: use auto (o axyn escreve os testes) ou manual (você escreve)")
		return exitUsage
	}
	if err := applyCoverageChoice(s, choice); err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn coverage: %v\n", err)
		return exitFail
	}
	if choice == "auto" {
		_, _ = fmt.Fprintln(stdout, "combinado: o axyn escreve os testes que faltam num ticket antes dos outros")
	} else {
		_, _ = fmt.Fprintln(stdout, "combinado: você escreve os testes que faltam; o axyn não deixa a cobertura cair e exige teste em todo código novo")
	}
	if *noResume {
		_, _ = fmt.Fprintln(stdout, "para continuar: axyn run --resume")
		return exitOK
	}
	id, err := s.startRun("", true)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn coverage: a escolha foi gravada, mas não consegui retomar: %v; rode: axyn run --resume\n", err)
		return exitFail
	}
	_, _ = fmt.Fprintf(stdout, "execução %s retomada em segundo plano; para acompanhar: axyn status --watch\n", id)
	return exitOK
}

var testTargetRe = regexp.MustCompile(`(?m)^test\s*:`)

func testTarget(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "Makefile"))
	return err == nil && testTargetRe.Match(b)
}
