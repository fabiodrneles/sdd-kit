package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// The engine drives the loop (spec 021 FR-9): plan, pick the ticket, call the coding
// agent without interaction, gates, recovery and delivery. The main model of opencode
// only starts it (axyn_run) and shows the progress (axyn_status); it decides nothing.

const (
	runRunning = "rodando"
	runDone    = "concluído"
	runStopped = "parado"

	// askMarker is how the recovery advice announces the question to the user (FR-8 6).
	askMarker      = "pergunta ao usuário ("
	exhaustedMark  = "escada esgotada"
	defaultTimeout = 20 * time.Minute
	staleAfter     = 30 * time.Minute
	planTries      = 2
	placeholder    = "padrão" // the single rung used when no ladder is configured
)

type runOpts struct {
	CI         string `json:"ci"`
	Base       string `json:"base"`
	MaxLines   int    `json:"max_lines"`
	Config     string `json:"config"`
	Model      string `json:"model,omitempty"` // forces one model instead of the ladder
	TimeoutSec int    `json:"timeout_sec"`
	Resume     bool   `json:"resume,omitempty"` // use the open plan instead of planning again
}

// runState is the file the status reads. It lives in .axyn/ (ignored by git), so the
// work survives the end of the session.
type runState struct {
	ID        string    `json:"id"`
	Request   string    `json:"request"`
	Status    string    `json:"status"`
	Phase     string    `json:"phase"`
	Ticket    int       `json:"ticket,omitempty"`
	Total     int       `json:"total,omitempty"`
	Title     string    `json:"title,omitempty"`
	Model     string    `json:"model,omitempty"`
	Attempts  int       `json:"attempts"`
	Gate      string    `json:"gate,omitempty"`
	Delivered []string  `json:"delivered,omitempty"`
	Message   string    `json:"message,omitempty"`
	Started   time.Time `json:"started"`
	Updated   time.Time `json:"updated"`
	Opts      runOpts   `json:"opts"`
}

func runsDir(dir string) string { return filepath.Join(dir, ".axyn", "runs") }

func statePathFor(dir, id string) string { return filepath.Join(runsDir(dir), id+".json") }

func saveRun(dir string, st *runState) error {
	if err := os.MkdirAll(runsDir(dir), 0o755); err != nil {
		return err
	}
	// .axyn holds only engine state: it must never reach a diff or a commit.
	if err := os.WriteFile(filepath.Join(dir, ".axyn", ".gitignore"), []byte("*\n"), 0o644); err != nil {
		return err
	}
	st.Updated = time.Now()
	b, _ := json.MarshalIndent(st, "", "  ")
	p := statePathFor(dir, st.ID)
	if err := os.WriteFile(p+".tmp", b, 0o644); err != nil {
		return err
	}
	return os.Rename(p+".tmp", p)
}

func loadRun(dir, id string) (*runState, error) {
	if id == "" {
		entries, _ := os.ReadDir(runsDir(dir))
		var names []string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".json") {
				names = append(names, strings.TrimSuffix(e.Name(), ".json"))
			}
		}
		if len(names) == 0 {
			return nil, fmt.Errorf("nenhuma execução: rode axyn_run antes")
		}
		sort.Strings(names)
		id = names[len(names)-1]
	}
	if strings.ContainsAny(id, `/\.`) {
		return nil, fmt.Errorf("id inválido: %s", id)
	}
	b, err := os.ReadFile(statePathFor(dir, id))
	if err != nil {
		return nil, fmt.Errorf("execução %s não encontrada", id)
	}
	var st runState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// renderStatus is the progress shown to the user: ticket, gates, model and attempts.
func renderStatus(st *runState) string {
	status := st.Status
	if st.Status == runRunning && time.Since(st.Updated) > staleAfter {
		status = "interrompida (sem sinal há " + time.Since(st.Updated).Round(time.Minute).String() + ")"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "execução %s: %s\npedido: %s\n", st.ID, status, oneLine(st.Request))
	if st.Ticket > 0 {
		fmt.Fprintf(&b, "ticket: %d de %d «%s»\n", st.Ticket, st.Total, st.Title)
	}
	fmt.Fprintf(&b, "fase: %s\n", st.Phase)
	if st.Model != "" {
		fmt.Fprintf(&b, "modelo: %s\n", st.Model)
	}
	fmt.Fprintf(&b, "tentativas no ticket: %d\n", st.Attempts)
	if st.Gate != "" {
		fmt.Fprintf(&b, "portões: %s\n", st.Gate)
	}
	for _, d := range st.Delivered {
		fmt.Fprintf(&b, "entregue: %s\n", d)
	}
	if st.Message != "" {
		fmt.Fprintf(&b, "%s\n", st.Message)
	}
	return strings.TrimRight(b.String(), "\n")
}

// startRun records the run and starts the worker in the background; it returns the id.
func (s *mcpServer) startRun(request string, resume bool) (string, error) {
	if strings.TrimSpace(request) == "" && !resume {
		return "", fmt.Errorf("axyn_run recusado: o pedido está vazio")
	}
	abs, err := filepath.Abs(s.dir)
	if err != nil {
		return "", err
	}
	if last, err := loadRun(abs, ""); err == nil && last.Status == runRunning && time.Since(last.Updated) < staleAfter {
		return "", fmt.Errorf("já há a execução %s em andamento; veja com axyn_status", last.ID)
	}
	st := &runState{
		ID: time.Now().Format("20060102-150405"), Request: request, Status: runRunning, Phase: "iniciando",
		Started: time.Now(),
		Opts: runOpts{CI: s.ci, Base: s.base, MaxLines: s.maxLines, Config: s.config,
			TimeoutSec: int(defaultTimeout / time.Second), Resume: resume},
	}
	if err := saveRun(abs, st); err != nil {
		return "", err
	}
	if err := spawnWorker(abs, st.ID); err != nil {
		st.Status, st.Message = runStopped, "não consegui iniciar o trabalho: "+err.Error()
		_ = saveRun(abs, st)
		return "", err
	}
	return st.ID, nil
}

// spawnWorker is replaced in tests. It starts the worker through a short-lived middle
// process (`run --detach`), so the worker is not a descendant of the MCP server: when
// opencode exits it kills the server's whole process tree, and a direct child died with
// it, its run left "rodando" forever.
var spawnWorker = func(dir, id string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "run", "--dir", dir, "--detach", id)
	detach(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// startDetached is the middle process: it starts the worker in its own session and exits.
func startDetached(dir, id string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	logf, err := os.OpenFile(filepath.Join(runsDir(dir), id+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = logf.Close() }()
	cmd := exec.Command(exe, "run", "--dir", dir, "--job", id)
	cmd.Stdout, cmd.Stderr = logf, logf
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func (s *mcpServer) toolRun(request string, resume bool) (string, error) {
	id, err := s.startRun(request, resume)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("execução %s iniciada em segundo plano; acompanhe com axyn_status (id %s). Só mostre o andamento: o axyn conduz o resto.", id, id), nil
}

func (s *mcpServer) toolStatus(id string) (string, error) {
	st, err := loadRun(s.dir, id)
	if err != nil {
		return "", err
	}
	return renderStatus(st), nil
}

type runner struct {
	s   *mcpServer
	st  *runState
	log io.Writer
}

func (r *runner) set(phase string) {
	r.st.Phase = phase
	_ = saveRun(r.s.dir, r.st)
}

func (r *runner) stop(status, msg string) {
	r.st.Status, r.st.Message = status, msg
	if status == runDone {
		r.st.Phase = "fim"
	}
	_ = saveRun(r.s.dir, r.st)
}

// ladder is the configured models; with none, a single placeholder lets FR-8 count the
// attempts and cap them, and the agent runs with opencode's own default model.
func (r *runner) ladder() []model {
	if ms, err := loadModels(r.s.config); err == nil && len(ms) > 0 {
		return ms
	}
	return []model{{ID: placeholder}}
}

type fatalErr struct{ error }

// agent calls `opencode run --agent NAME [--model M] PROMPT` without interaction.
func (r *runner) agent(name, modelID, prompt string) error {
	argv := strings.Fields(os.Getenv("AXYN_OPENCODE"))
	if len(argv) == 0 {
		argv = []string{"opencode"}
	}
	args := append(append([]string{}, argv[1:]...), "run", "--agent", name)
	if modelID != "" {
		args = append(args, "--model", modelID)
	}
	args = append(args, prompt)
	timeout := time.Duration(r.st.Opts.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], args...)
	cmd.Dir = r.s.dir
	cmd.Stdout, cmd.Stderr = r.log, r.log
	_, _ = fmt.Fprintf(r.log, "== %s %s\n", argv[0], name)
	err := cmd.Run()
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) && ctx.Err() == nil {
		return fatalErr{fmt.Errorf("não consegui chamar o agente %s: %w", name, err)} // the command did not even start
	}
	return nil // a failing or timed-out agent is just a bad attempt: the gates judge the diff
}

func (r *runner) plan() error {
	oldSpec := ""
	if pl, _, err := r.s.loadPlan(); err == nil {
		oldSpec = pl.Spec
	}
	prompt := "Pedido do usuário: " + r.st.Request
	for try := 1; try <= planTries; try++ {
		r.set("planejando")
		if err := r.agent("axyn-plan", r.planModel(), prompt); err != nil {
			return err
		}
		if pl, _, err := r.s.loadPlan(); err == nil && pl.Spec != oldSpec && len(pl.Tickets) > 0 {
			return nil
		}
		prompt += "\n\nO plano não foi gravado: chame a ferramenta axyn_plan com a spec (com FR-N e AC-N) e os tickets, cada um citando um AC."
	}
	return fmt.Errorf("o agente axyn-plan não gravou um plano válido em %d tentativas", planTries)
}

// planModel is the model the planner runs on: the forced one, else the first of the
// ladder (axyn model), else opencode's default. Before, the planner always took
// opencode's default, which may be a model the user has no key for.
func (r *runner) planModel() string {
	if r.st.Opts.Model != "" {
		return r.st.Opts.Model
	}
	if ms := r.ladder(); len(ms) > 0 && ms[0].ID != placeholder {
		return ms[0].ID
	}
	return ""
}

// run is the whole loop. It returns when every ticket is delivered or the engine must stop.
func (r *runner) run() {
	if out, err := r.s.git("status", "--porcelain"); err != nil || strings.TrimSpace(out) != "" {
		r.stop(runStopped, "a árvore de trabalho tem alterações (ou não é um repositório git); faça commit ou guarde-as antes do axyn_run")
		return
	}
	start, _ := r.s.git("rev-parse", "--abbrev-ref", "HEAD")
	if !r.st.Opts.Resume {
		if err := r.plan(); err != nil {
			r.stop(runStopped, err.Error())
			return
		}
	}
	// Every ticket starts from the plan's base branch (FR-9: one branch and one PR per
	// ticket); before this, ticket 2 branched from ticket 1 and its PR carried both. The
	// base lives in the plan, so a --resume from a WIP branch still finds it.
	base := ""
	if pl, path, err := r.s.loadPlan(); err == nil {
		if pl.Base == "" && start != "HEAD" {
			pl.Base = start
			_ = r.s.savePlan(pl, path)
		}
		base = pl.Base
	}
	if cur, _ := r.s.git("rev-parse", "--abbrev-ref", "HEAD"); base != "" && cur != base {
		if out, err := r.s.git("checkout", base); err != nil {
			r.stop(runStopped, "não consegui voltar à branch base "+base+": "+out)
			return
		}
	}
	if msg := r.prepare(); msg != "" {
		r.stop(runStopped, msg)
		return
	}
	for {
		pl, t := r.s.openTicket()
		switch {
		case pl == nil:
			r.stop(runStopped, "nenhum plano gravado")
			return
		case t == nil:
			if base != "" {
				_, _ = r.s.git("checkout", base) // the user ends where the run started
			}
			r.stop(runDone, fmt.Sprintf("os %d ticket(s) do plano estão entregues", len(pl.Tickets)))
			return
		}
		if cur, _ := r.s.git("rev-parse", "--abbrev-ref", "HEAD"); base != "" && cur != base {
			if out, err := r.s.git("checkout", base); err != nil {
				r.stop(runStopped, "não consegui voltar à branch base "+base+": "+out)
				return
			}
		}
		r.st.Ticket, r.st.Total, r.st.Title, r.st.Attempts, r.st.Gate = t.ID, len(pl.Tickets), t.Title, len(t.Attempts), ""
		if status, msg := r.ticket(); status != "" {
			r.stop(status, msg)
			return
		}
	}
}

// ticket drives the open ticket to delivery. It returns a status only when the run must stop.
func (r *runner) ticket() (string, string) {
	feedback := ""
	bound := maxFailsPerModel*len(r.ladder()) + 2
	for i := 0; i < bound; i++ {
		_, t := r.s.openTicket()
		if t == nil {
			return runStopped, "o ticket sumiu do plano"
		}
		models := r.ladder()
		idx := ladderState(models, t.Attempts)
		if idx >= len(models) {
			return runStopped, exhaustedMark + " no ticket " + t.Title
		}
		cur := models[idx].ID
		modelArg := cur
		if cur == placeholder {
			modelArg = ""
		}
		if r.st.Opts.Model != "" {
			modelArg = r.st.Opts.Model
		}
		r.st.Model, r.st.Attempts = cur, len(t.Attempts)
		prompt, _ := r.s.toolNext()
		if feedback != "" {
			prompt += "\n\nA tentativa anterior foi reprovada pelo motor:\n" + feedback
		}
		r.set("código")
		planPath, planBefore := r.planSnapshot()
		if err := r.agent("axyn-code", modelArg, prompt); err != nil {
			return runStopped, err.Error()
		}
		r.set("portões")
		var green bool
		var report string
		var err error
		if r.planTampered(planPath, planBefore) {
			advice, _ := r.s.recordAttempt(false, []finding{{"plan", "o agente alterou o plano do axyn (restaurado)"}}, nil, "")
			report = strings.TrimSpace("gate: reprovado [plan] o agente alterou o plano do axyn (restaurado)\n" + advice)
		} else if green, report, err = r.gate(); err != nil {
			return runStopped, err.Error()
		}
		r.st.Gate = firstLine(report)
		if green {
			r.set("entrega")
			out, err := r.s.deliver("feat: "+strings.TrimSpace(t.Title), true)
			if err != nil {
				return runStopped, err.Error()
			}
			r.st.Delivered = append(r.st.Delivered, fmt.Sprintf("ticket %d «%s» — %s", t.ID, t.Title, strings.ReplaceAll(out, "\n", "; ")))
			return "", ""
		}
		_, _ = fmt.Fprintf(r.log, "%s\n", report)
		switch {
		case strings.Contains(report, askMarker):
			note := r.s.saveWIP(t, "à espera da resposta do usuário")
			return runStopped, "o axyn precisa de uma resposta sua; grave com axyn_decide e rode axyn_run de novo com resume:\n" + extractAsk(report) + "\n" + note
		case strings.Contains(report, exhaustedMark):
			return runStopped, report
		}
		feedback = report
	}
	return runStopped, "teto de tentativas do ticket " + r.st.Title
}

// planSnapshot keeps the plan as it was before the coding agent runs. The plan lives
// in .git, out of every diff, so the gates cannot see an agent that rewrites it (marks a
// ticket delivered, changes the spec); the engine compares it instead.
func (r *runner) planSnapshot() (string, []byte) {
	p, err := r.s.statePath()
	if err != nil {
		return "", nil
	}
	b, _ := os.ReadFile(p)
	return p, b
}

// planTampered reports whether the agent changed the plan, and puts the original back.
func (r *runner) planTampered(path string, before []byte) bool {
	if path == "" {
		return false
	}
	after, _ := os.ReadFile(path)
	if bytes.Equal(after, before) {
		return false
	}
	_ = os.WriteFile(path, before, 0o644)
	return true
}

// gate runs the gates on the agent's diff. An agent that changed nothing is a failed attempt.
func (r *runner) gate() (bool, string, error) {
	if out, _ := r.s.git("status", "--porcelain"); strings.TrimSpace(out) == "" {
		advice, _ := r.s.recordAttempt(false, []finding{{"diff", "o agente não alterou nada"}}, nil, "")
		return false, strings.TrimSpace("gate: reprovado [diff] o agente não alterou nada\n" + advice), nil
	}
	return r.s.gate(gateArgs{})
}

func firstLine(s string) string {
	l, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return l
}

func extractAsk(report string) string {
	var out []string
	on := false
	for _, l := range strings.Split(report, "\n") {
		if strings.Contains(l, askMarker) {
			on = true
		}
		if on {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

// runJob is the worker: it runs the loop of a recorded run in this process.
func runJob(dir, id string, out io.Writer) int {
	st, err := loadRun(dir, id)
	if err != nil {
		_, _ = fmt.Fprintf(out, "axyn: %v\n", err)
		return exitFail
	}
	s := &mcpServer{dir: dir, ci: st.Opts.CI, base: st.Opts.Base, maxLines: st.Opts.MaxLines, config: st.Opts.Config, fallback: true}
	r := &runner{s: s, st: st, log: out}
	r.run()
	_, _ = fmt.Fprintln(out, renderStatus(st))
	if st.Status != runDone {
		return exitFail
	}
	return exitOK
}

func runRunCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ci := fs.String("ci", defaultCICmd, "comando do CI do projeto")
	base := fs.String("base", "HEAD", "referência do git contra a qual o diff é medido")
	maxLines := fs.Int("max-lines", defaultMaxLines, "limite de linhas alteradas por ticket")
	config := fs.String("config", defaultConfigPath(), "arquivo com a escada de modelos")
	mdl := fs.String("model", "", "força um modelo, em vez da escada")
	timeout := fs.Duration("timeout", defaultTimeout, "tempo limite de cada chamada ao agente")
	wait := fs.Bool("wait", false, "roda no primeiro plano, até o fim")
	resume := fs.Bool("resume", false, "retoma o plano aberto, sem planejar de novo")
	dir := fs.String("dir", ".", "raiz do repositório")
	job := fs.String("job", "", "(interno) id da execução a conduzir")
	detachID := fs.String("detach", "", "(interno) inicia a execução fora da árvore de processos do chamador")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *detachID != "" {
		if err := startDetached(*dir, *detachID); err != nil {
			_, _ = fmt.Fprintf(stderr, "axyn run: %v\n", err)
			return exitFail
		}
		return exitOK
	}
	if *job != "" {
		return runJob(*dir, *job, stdout)
	}
	request := strings.Join(fs.Args(), " ")
	if strings.TrimSpace(request) == "" && !*resume {
		_, _ = fmt.Fprintln(stderr, "axyn run: informe o pedido, por exemplo: axyn run \"crie uma landing page\"")
		return exitUsage
	}
	s := &mcpServer{dir: *dir, ci: *ci, base: *base, maxLines: *maxLines, config: *config}
	if !*wait {
		id, err := s.startRun(request, *resume)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "axyn run: %v\n", err)
			return exitFail
		}
		_, _ = fmt.Fprintf(stdout, "execução %s iniciada em segundo plano; veja com: axyn status %s\n", id, id)
		return exitOK
	}
	st := &runState{ID: time.Now().Format("20060102-150405"), Request: request, Status: runRunning, Phase: "iniciando",
		Started: time.Now(), Opts: runOpts{CI: *ci, Base: *base, MaxLines: *maxLines, Config: *config, Model: *mdl,
			TimeoutSec: int(timeout.Seconds()), Resume: *resume}}
	if err := saveRun(*dir, st); err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn run: %v\n", err)
		return exitFail
	}
	// The log also goes to the run's file, as in the background, for axyn history (FR-7).
	out := stdout
	if logf, err := os.OpenFile(filepath.Join(runsDir(*dir), st.ID+".log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		defer func() { _ = logf.Close() }()
		out = io.MultiWriter(stdout, logf)
	}
	return runJob(*dir, st.ID, out)
}

func runStatusCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "raiz do repositório")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	st, err := loadRun(*dir, fs.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn status: %v\n", err)
		return exitFail
	}
	_, _ = fmt.Fprintln(stdout, renderStatus(st))
	return exitOK
}
