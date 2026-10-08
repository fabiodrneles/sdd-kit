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
	CI         string     `json:"ci"`
	Base       string     `json:"base"`
	MaxLines   int        `json:"max_lines"`
	Config     string     `json:"config"`
	Model      string     `json:"model,omitempty"` // forces one model instead of the ladder
	TimeoutSec int        `json:"timeout_sec"`
	Resume     bool       `json:"resume,omitempty"` // use the open plan instead of planning again
	Bench      *benchSpec `json:"bench,omitempty"`  // this run is a model bench, not a request (#356)
}

// runState is the file the status reads. It lives in .axyn/ (ignored by git), so the
// work survives the end of the session.
type runState struct {
	PID       int       `json:"pid,omitempty"` // the worker process; a dead one means the run was interrupted
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

	// Live progress for the watch (#348): when the phase began and, for long phases such
	// as the model bench, how far it is.
	PhaseSince time.Time `json:"phase_since,omitempty"`
	Done       int       `json:"done,omitempty"`
	Of         int       `json:"of,omitempty"`
	Detail     string    `json:"detail,omitempty"`
}

func runsDir(dir string) string { return filepath.Join(dir, ".axyn", "runs") }

// The model benches keep their own list (#363): the project's work and the evaluations
// never share a file, so neither hides the other from resume, decide, status or history.
func benchRunsDir(dir string) string { return filepath.Join(dir, ".axyn", "bench-runs") }

const benchIDPrefix = "bench-"

func runDirFor(dir, id string) string {
	if strings.HasPrefix(id, benchIDPrefix) {
		return benchRunsDir(dir)
	}
	return runsDir(dir)
}

func statePathFor(dir, id string) string { return filepath.Join(runDirFor(dir, id), id+".json") }

func saveRun(dir string, st *runState) error {
	if err := os.MkdirAll(runDirFor(dir, st.ID), 0o755); err != nil {
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
		// The last run of the project's work; a bench saved here by axyn v1.22 is skipped.
		names := runIDs(runsDir(dir))
		for i := len(names) - 1; i >= 0; i-- {
			if st, err := loadRun(dir, names[i]); err == nil && st.Opts.Bench == nil {
				return st, nil
			}
		}
		return nil, fmt.Errorf("nenhuma execução ainda: peça uma tarefa com /axyn no opencode, ou rode axyn run \"seu pedido\" na raiz do projeto")
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

// loadBenchRun is the last model bench of this folder.
func loadBenchRun(dir string) (*runState, error) {
	names := runIDs(benchRunsDir(dir))
	if len(names) == 0 {
		return nil, fmt.Errorf("nenhuma avaliação dos modelos ainda nesta pasta; para começar: axyn bench")
	}
	return loadRun(dir, names[len(names)-1])
}

func runIDs(d string) []string {
	entries, _ := os.ReadDir(d)
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			names = append(names, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	sort.Strings(names)
	return names
}

// busy is what runs in this folder now: the work or a bench (one at a time, so a small
// machine runs one opencode at a time).
func busy(dir string) *runState {
	if st, err := loadRun(dir, ""); err == nil && alive(st) {
		return st
	}
	if st, err := loadBenchRun(dir); err == nil && alive(st) {
		return st
	}
	return nil
}

// alive says whether a run marked "rodando" still has its worker (spec 021 FR-8): a
// worker that died (the computer froze, the terminal was killed) leaves the state behind,
// and the run must be resumable at once, not after staleAfter.
func alive(st *runState) bool {
	if st.Status != runRunning || time.Since(st.Updated) > staleAfter {
		return false
	}
	return st.PID == 0 || processAlive(st.PID)
}

// renderStatus is the progress shown to the user: ticket, gates, model and attempts.
func renderStatus(st *runState) string {
	status := st.Status
	switch {
	case st.Status == runRunning && time.Since(st.Updated) > staleAfter:
		status = "interrompida (sem sinal há " + time.Since(st.Updated).Round(time.Minute).String() + "); para continuar: axyn run --resume"
	case st.Status == runRunning && !alive(st):
		status = "interrompida (o processo do axyn parou, talvez o computador travou ou o terminal fechou); para continuar: axyn run --resume"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "execução %s: %s\npedido: %s\n", st.ID, status, requestLabel(st))
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
	// #365: outside a repository (the owner ran it in his home folder) the run would only
	// start to stop; say so at once, with where to go.
	if _, err := os.Stat(filepath.Join(abs, ".git")); err != nil {
		example := "cd ~/caminho/do/projeto"
		if hostOS == "windows" {
			example = "cd E:\\caminho\\do\\projeto"
		}
		return "", fmt.Errorf("esta pasta (%s) não é a raiz de um projeto git; entre na pasta do projeto (%s) e rode de novo", abs, example)
	}
	if b := busy(abs); b != nil {
		return "", errRunning{b}
	}
	if last, err := loadRun(abs, ""); resume && strings.TrimSpace(request) == "" && err == nil && last.Request != "avaliação dos modelos" {
		request = last.Request // the status keeps showing what was asked (v1.22 copied the bench's name)
	}
	if resume && strings.TrimSpace(request) == "" {
		if pl, _, err := s.loadPlan(); err == nil {
			request = "retomada do plano " + pl.Spec // an older run kept no request
		}
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
	logf, err := openLog(filepath.Join(runDirFor(dir, id), id+".log"))
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

// errRunning is a run already going on: not an error for the user, who gets its progress.
type errRunning struct{ st *runState }

func (e errRunning) Error() string {
	if e.st.Opts.Bench != nil {
		return "Agora há uma avaliação dos modelos rodando nesta pasta (execução " + e.st.ID + "), e o trabalho do projeto espera ela terminar. Andamento:\n" + renderStatus(e.st) +
			"\nPara acompanhar: axyn bench --watch. Para não esperar: axyn stop (o que já foi avaliado fica salvo) e, em seguida, axyn run --resume para continuar o trabalho do projeto."
	}
	return "Já estou trabalhando nisso (execução " + e.st.ID + "). Andamento agora:\n" + renderStatus(e.st) +
		"\nPara acompanhar sem rodar nada à mão: axyn status --watch. Para parar: axyn stop (depois, axyn run --resume continua de onde parou)."
}

func (s *mcpServer) toolRun(request string, resume bool) (string, error) {
	id, err := s.startRun(request, resume)
	var running errRunning
	if errors.As(err, &running) {
		return running.Error(), nil
	}
	if err != nil {
		return "", err
	}
	return withUpdate(fmt.Sprintf("Comecei a trabalhar no seu pedido (execução %s). Vou mostrando o andamento por aqui; num terminal, axyn status --watch avisa a cada mudança.", id)), nil
}

func (s *mcpServer) toolStatus(id string) (string, error) {
	st, err := loadRun(s.dir, id)
	if err != nil {
		return "", err
	}
	return withUpdate(renderStatus(st)), nil
}

type runner struct {
	planAgent string // axyn-plan, or axyn-plan-open after the free tier refused it
	lastOut   string // the tail of the last agent's output (model down detection)
	s         *mcpServer
	st        *runState
	log       io.Writer
}

func (r *runner) set(phase string) {
	if phase != r.st.Phase {
		r.st.PhaseSince = time.Now()
		r.st.Done, r.st.Of, r.st.Detail = 0, 0, ""
	}
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

// ladderFor is the ladder of one ticket (see mcpServer.ticketLadder).
func (r *runner) ladderFor(t *ticket) []model {
	if ms := r.s.ticketLadder(t); len(ms) > 0 {
		return ms
	}
	if len(r.s.down) > 0 {
		return nil // every model is down
	}
	return r.ladder()
}

type fatalErr struct{ error }

// agent calls `opencode run --agent NAME [--model M] PROMPT` without interaction.
func (r *runner) agent(name, modelID, prompt string) error {
	timeout := time.Duration(r.st.Opts.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	tail, err := callAgent(r.s.dir, name, modelID, prompt, timeout, r.log)
	r.lastOut = tail
	return err
}

// callAgent runs one opencode agent in dir; the engine and the model bench share it.
func callAgent(dir, name, modelID, prompt string, timeout time.Duration, log io.Writer) (string, error) {
	argv := strings.Fields(os.Getenv("AXYN_OPENCODE"))
	if len(argv) == 0 {
		argv = []string{"opencode"}
	}
	args := append(append([]string{}, argv[1:]...), "run", "--agent", name)
	if modelID != "" {
		args = append(args, "--model", modelID)
	}
	exe := argv[0]
	if p, err := exec.LookPath(argv[0]); err == nil {
		exe = p
	}
	args = append(args, cmdSafe(exe, windowsShellNote()+prompt))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], args...)
	cmd.Dir = dir
	// opencode's shell tool stops a command after 2 minutes; make ci on a small machine
	// takes longer (#380). Its default goes to 15 minutes; the call itself keeps its own cap.
	if os.Getenv("OPENCODE_EXPERIMENTAL_BASH_DEFAULT_TIMEOUT_MS") == "" {
		cmd.Env = append(os.Environ(), "OPENCODE_EXPERIMENTAL_BASH_DEFAULT_TIMEOUT_MS=900000")
	}
	tail := &tailWriter{max: 16 << 10}
	w := io.MultiWriter(log, tail)
	cmd.Stdout, cmd.Stderr = w, w
	_, _ = fmt.Fprintf(log, "== %s %s\n", argv[0], name)
	err := cmd.Run()
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) && ctx.Err() == nil {
		return tail.String(), fatalErr{fmt.Errorf("não consegui chamar o agente %s: %w", name, err)} // the command did not even start
	}
	return tail.String(), nil // a failing or timed-out agent is just a bad attempt: the gates judge the diff
}

func (r *runner) plan() error {
	if r.planAgent == "" {
		r.planAgent = "axyn-plan"
	}
	oldSpec := ""
	if pl, _, err := r.s.loadPlan(); err == nil {
		oldSpec = pl.Spec
	}
	prompt := "Pedido do usuário: " + r.st.Request
	for try := 1; try <= planTries; try++ {
		r.set("planejando")
		m := r.planModel()
		if err := r.agent(r.planAgent, m, prompt); err != nil {
			return err
		}
		r.undoPlannerEdits()
		if pl, _, err := r.s.loadPlan(); err == nil && pl.Spec != oldSpec && len(pl.Tickets) > 0 {
			return nil
		}
		if freeTierRefused(r.lastOut) && r.planAgent != planOpenAgent {
			_, _ = fmt.Fprintf(r.log, "o opencode recusou o planejador restrito no plano gratuito; refazendo com o %s (as edições dele são desfeitas)\n", planOpenAgent)
			r.planAgent = planOpenAgent
			try-- // the refusal is opencode's, not a planning attempt
			continue
		}
		if why := downReason(r.lastOut, false); why != "" && m != "" {
			r.s.markDown(m, why)
			if r.planModel() == "" {
				return fmt.Errorf("%s", allDownMessage(r.s.down))
			}
			_, _ = fmt.Fprintf(r.log, "modelo %s %s: o plano passa para %s\n", m, why, r.planModel())
			try-- // a model that is down is not a planning attempt
			continue
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
	// The best planner of the applied bench first (#344), then the ladder; a model that
	// went down in this run is skipped.
	cands := routed(roleplan)
	for _, m := range r.ladder() {
		if m.ID != placeholder {
			cands = append(cands, m.ID)
		}
	}
	for _, m := range cands {
		if _, down := r.s.down[m]; !down {
			return m
		}
	}
	return ""
}

// run is the whole loop. It returns when every ticket is delivered or the engine must stop.
func (r *runner) run() {
	start, _ := r.s.git("rev-parse", "--abbrev-ref", "HEAD")
	if out, err := r.s.git("status", "--porcelain"); err != nil {
		r.stop(runStopped, "esta pasta não é um repositório git; rode o axyn na raiz do projeto")
		return
	} else if strings.TrimSpace(out) != "" {
		// A resumed run finds the code of the interrupted attempt: it goes to a WIP commit
		// (nothing is lost) and the ticket starts again from the base (spec 021 FR-8).
		_, t := r.s.openTicket()
		if !r.st.Opts.Resume || t == nil {
			r.stop(runStopped, "a pasta do projeto tem alterações que não são do axyn; faça commit delas (ou guarde com git stash) e peça de novo")
			return
		}
		_, _ = r.s.git("reset", "-q") // intent-to-add left by an interrupted gate
		note := r.s.saveWIP(t, "tentativa interrompida")
		r.recordWIP(t.ID)
		_, _ = fmt.Fprintln(r.log, note)
	}
	r.autoBench()
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
	if msg := r.measureCoverage(); msg != "" {
		_, _ = fmt.Fprintln(r.log, msg)
	}
	if q := r.coverageQuestion(); q != "" {
		r.stop(runStopped, q)
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
			done := fmt.Sprintf("os %d ticket(s) do plano estão entregues", len(pl.Tickets))
			if _, err := os.Stat(filepath.Join(r.s.dir, filepath.FromSlash(releaseScript))); err == nil {
				done += "; depois do merge dos PRs, para fechar uma versão com eles: axyn release"
			}
			r.stop(runDone, done)
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
			if cur, _ := r.s.git("rev-parse", "--abbrev-ref", "HEAD"); base != "" && cur != base {
				if out, _ := r.s.git("status", "--porcelain"); strings.TrimSpace(out) != "" {
					_ = r.s.saveWIP(t, "parada")
				}
				r.recordWIP(t.ID)                      // the stop message and the next resume use this branch
				_, _ = r.s.git("checkout", "-q", base) // the user ends where the run started
			}
			r.stop(status, msg)
			return
		}
	}
}

// ticket drives the open ticket to delivery. It returns a status only when the run must stop.
func (r *runner) ticket() (string, string) {
	feedback := ""
	if _, t := r.s.openTicket(); t != nil {
		if green, report, ok := r.gateWIP(t); ok {
			if green {
				return r.ship(t)
			}
			if strings.HasPrefix(report, "o `make ci` precisa") {
				return runStopped, report
			}
			feedback = report
		}
	}
	bound := maxFailsPerModel*(len(r.ladder())+8) + 2
	for i := 0; i < bound; i++ {
		_, t := r.s.openTicket()
		if t == nil {
			return runStopped, "o ticket sumiu do plano"
		}
		models := r.ladderFor(t)
		idx := ladderState(models, t.Attempts)
		if idx >= len(models) {
			return runStopped, r.explainStop(t.ID, "")
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
		r.s.contained = isContained(t, cur)
		if r.s.contained {
			prompt = containedGuide + prompt
			_, _ = fmt.Fprintf(r.log, "modelo %s no modo guiado (contido pela avaliação)\n", cur)
		}
		if feedback != "" {
			prompt += "\n\nA tentativa anterior foi reprovada pelo motor:\n" + feedback
		}
		if t.testsOnly() {
			if pl, _, err := r.s.loadPlan(); err == nil {
				if put := r.s.restoreProduction(pl); len(put) > 0 {
					_, _ = fmt.Fprintf(r.log, "o ticket «%s» só acrescenta testes, e a tentativa anterior mudou código de produção (%s): o ticket recomeça da base\n", t.Title, strings.Join(put, ", "))
					prompt += "\n\nA tentativa anterior mudou código de produção (" + strings.Join(put, ", ") + ") e foi descartada: o projeto está como na base. Este ticket só acrescenta testes para o código como ele está; não implemente nada da spec."
				}
			}
		}
		r.set("código")
		planPath, planBefore := r.planSnapshot()
		if err := r.agent("axyn-code", modelArg, prompt); err != nil {
			return runStopped, err.Error()
		}
		dirty, _ := r.s.git("status", "--porcelain")
		if why := downReason(r.lastOut, strings.TrimSpace(dirty) != ""); why != "" && cur != placeholder {
			r.s.markDown(cur, why)
			if len(r.ladderFor(t)) == 0 {
				return runStopped, allDownMessage(r.s.down)
			}
			next := r.ladderFor(t)[min(ladderState(r.ladderFor(t), t.Attempts), len(r.ladderFor(t))-1)].ID
			_, _ = fmt.Fprintf(r.log, "modelo %s %s: a tentativa não conta, e o trabalho passa para %s\n", cur, why, next)
			r.st.Gate = "modelo " + cur + " " + why + "; trocando para " + next
			continue
		}
		// The lock of a tests-only ticket (#380): whatever production code the model changed
		// is undone before the gates, so it never reaches a delivery; its tests must pass on
		// the code as it is.
		lockNote := ""
		if t.testsOnly() {
			if pl, _, err := r.s.loadPlan(); err == nil {
				if undone := r.s.revertOffenders(pl); len(undone) > 0 {
					lockNote = "trava do ticket só de testes: desfiz as mudanças em " + strings.Join(undone, ", ") + "; os testes precisam passar com o código como ele está, sem mexer em produção\n"
					_, _ = fmt.Fprint(r.log, lockNote)
				}
			}
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
		report = lockNote + report
		r.st.Gate = firstLine(report)
		if green {
			return r.ship(t)
		}
		_, _ = fmt.Fprintf(r.log, "%s\n", report)
		switch {
		case strings.Contains(report, askMarker):
			_ = r.s.saveWIP(t, "à espera da resposta do usuário")
			help := r.publishHelp(t.ID)
			return runStopped, r.explainStop(t.ID, help)
		case strings.Contains(report, exhaustedMark):
			help := r.publishHelp(t.ID)
			return runStopped, r.explainStop(t.ID, help)
		}
		feedback = report
	}
	return runStopped, "teto de tentativas do ticket " + r.st.Title
}

// ship delivers a ticket whose diff passed the gates.
func (r *runner) ship(t *ticket) (string, string) {
	r.set("entrega")
	if msg := r.raiseCoverage(); msg != "" {
		_, _ = fmt.Fprintln(r.log, msg)
	}
	msg := "feat: " + strings.TrimSpace(t.Title)
	if r.st.Model != "" && r.st.Model != placeholder {
		// The author model in the commit: without it, which model's code survives or is
		// reverted later cannot be measured (the QA material, "registrar a autoria").
		msg += "\n\nAxyn-Model: " + r.st.Model
	}
	out, err := r.s.deliver(msg, true)
	if err != nil {
		return runStopped, err.Error()
	}
	r.st.Delivered = append(r.st.Delivered, fmt.Sprintf("ticket %d «%s» — %s", t.ID, t.Title, strings.ReplaceAll(out, "\n", "; ")))
	return "", ""
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

// runJob is the worker: it runs the loop of a recorded run in this process.
func runJob(dir, id string, out io.Writer) int {
	st, err := loadRun(dir, id)
	if err != nil {
		_, _ = fmt.Fprintf(out, "axyn: %v\n", err)
		return exitFail
	}
	st.PID = os.Getpid()
	_ = saveRun(dir, st)
	if st.Opts.Bench != nil {
		return benchJob(dir, st, out)
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
		var running errRunning
		if errors.As(err, &running) {
			_, _ = fmt.Fprintln(stdout, running.Error())
			return exitOK
		}
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "axyn run: %v\n", err)
			return exitFail
		}
		_, _ = fmt.Fprintf(stdout, "execução %s iniciada em segundo plano; para acompanhar, com aviso quando terminar ou precisar de você: axyn status --watch\n", id)
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
	if logf, err := openLog(filepath.Join(runsDir(*dir), st.ID+".log")); err == nil {
		defer func() { _ = logf.Close() }()
		out = io.MultiWriter(stdout, logf)
	}
	return runJob(*dir, st.ID, out)
}

func runStatusCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "raiz do repositório")
	watch := fs.Bool("watch", false, "fica acompanhando: uma linha a cada mudança, e aviso (bipe e notificação) quando termina, para ou pergunta algo")
	interval := fs.Duration("interval", 2*time.Second, "(--watch) de quanto em quanto tempo conferir")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *watch {
		return watchRun(*dir, fs.Arg(0), *interval, stdout)
	}
	st, err := loadRun(*dir, fs.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn status: %v\n", err)
		return exitFail
	}
	text := withUpdate(renderStatus(st))
	if isTerminal(stdout) {
		enableVT()
		text = colorStatus(text)
	}
	_, _ = fmt.Fprintln(stdout, text)
	return exitOK
}

// requestLabel is what was asked; a resume by axyn v1.22 copied the bench's name into the
// project's work, and that run shows as the work it is (#364).
func requestLabel(st *runState) string {
	if st.Opts.Bench == nil && st.Request == "avaliação dos modelos" {
		return "retomada do trabalho do projeto"
	}
	return oneLine(st.Request)
}
