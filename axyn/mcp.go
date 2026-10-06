package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// The MCP server (spec 021 FR-3): JSON-RPC 2.0 over stdio, one message per line.

type rpcRequest struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type ticket struct {
	ID    int      `json:"id"`
	Title string   `json:"title"`
	Body  string   `json:"body"`
	ACs   []string `json:"acs"`
	Done  bool     `json:"done"`

	// Record of the delivery (spec 021 FR-7).
	Attempts []attempt `json:"attempts,omitempty"`
	Model    string    `json:"model,omitempty"`
	Strategy string    `json:"strategy,omitempty"`
	CostUSD  float64   `json:"cost_usd,omitempty"`
}

type plan struct {
	Spec    string   `json:"spec"`
	Tickets []ticket `json:"tickets"`
}

// mcpServer holds the gate settings. They come from how the engine started the
// server (axyn mcp flags), never from the tool call: the caller is the model whose
// diff is being judged, and it must not be able to pick an empty CI, a huge limit or
// a base that empties the diff (021 FR-5, NFR-2).
type mcpServer struct {
	dir      string
	ci       string
	base     string
	maxLines int
	config   string // model ladder file (FR-6); empty or missing means no ladder
	fallback bool   // the run loop (FR-9) counts attempts even with no ladder configured
}

func obj(props map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": required}
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

var mcpTools = []toolDef{
	{"axyn_plan", "Valida e grava a spec (specs/NNN-nome/spec.md) e os tickets; cada ticket cita ao menos um AC da spec.",
		obj(map[string]any{
			"name": str("nome curto da spec, em minúsculas e hífens"),
			"spec": str("texto da spec em markdown, com FR-N e AC-N"),
			"tickets": map[string]any{"type": "array", "items": obj(map[string]any{
				"title": str("título"), "body": str("o que fazer"),
				"acs": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			}, "title", "acs")},
		}, "name", "spec", "tickets")},
	{"axyn_next", "Devolve o próximo ticket aberto e o pacote dele (spec, ACs citados).",
		obj(map[string]any{})},
	{"axyn_gate", "Roda os portões (CI, teste afrouxado, caminhos protegidos, tamanho) sobre o diff atual.",
		obj(map[string]any{
			"max_lines": map[string]any{"type": "integer", "description": "limite menor que o do projeto (nunca maior)"},
			"plan":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "arquivos que vai alterar (passo do plano antes do código)"},
		})},
	{"axyn_decide", "Grava a resposta do usuário a uma pergunta do axyn como decisão na spec.",
		obj(map[string]any{
			"question": str("a pergunta feita ao usuário"),
			"answer":   str("a resposta do usuário"),
		}, "question", "answer")},
	{"axyn_run", "Inicia em segundo plano o laço inteiro do axyn (planejar, código, portões, recuperação, entrega) e devolve um id. Não decide nada do processo: depois só mostre o andamento com axyn_status.",
		obj(map[string]any{
			"request": str("o pedido do usuário, como foi escrito"),
			"resume":  map[string]any{"type": "boolean", "description": "retoma o plano aberto, sem planejar de novo (depois de axyn_decide)"},
		})},
	{"axyn_status", "Mostra o andamento de uma execução do axyn_run: ticket atual, portões, modelo e tentativas.",
		obj(map[string]any{"id": str("id devolvido pelo axyn_run; vazio é a última execução")})},
	{"axyn_ship", "Commit, push e PR do diff atual, só se o portão estiver verde; recusa se reprovar.",
		obj(map[string]any{
			"message": str("mensagem do commit (Conventional Commits)"),
		}, "message")},
}

func runMCP(args []string, in io.Reader, out, errw io.Writer) int {
	fs := flag.NewFlagSet("axyn mcp", flag.ContinueOnError)
	fs.SetOutput(errw)
	ci := fs.String("ci", defaultCICmd, "comando do CI do projeto")
	base := fs.String("base", "HEAD", "referência do git contra a qual o diff é medido")
	maxLines := fs.Int("max-lines", defaultMaxLines, "limite de linhas alteradas por ticket")
	config := fs.String("config", defaultConfigPath(), "arquivo com a escada de modelos")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	s := &mcpServer{dir: ".", ci: *ci, base: *base, maxLines: *maxLines, config: *config}
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	enc := json.NewEncoder(out)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32700, "JSON inválido"}})
			continue
		}
		if len(req.ID) == 0 { // notification: no reply
			continue
		}
		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
		switch req.Method {
		case "initialize":
			resp.Result = map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "axyn", "version": version},
			}
		case "ping":
			resp.Result = map[string]any{}
		case "tools/list":
			resp.Result = map[string]any{"tools": mcpTools}
		case "tools/call":
			var p struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				resp.Error = &rpcError{-32602, "parâmetros inválidos"}
				break
			}
			text, err := s.call(p.Name, p.Arguments)
			isErr := err != nil
			if isErr {
				text = err.Error()
			}
			resp.Result = map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}, "isError": isErr}
		default:
			resp.Error = &rpcError{-32601, "método desconhecido: " + req.Method}
		}
		_ = enc.Encode(resp)
	}
	if err := sc.Err(); err != nil {
		_, _ = fmt.Fprintf(errw, "axyn mcp: %v\n", err)
		return exitUsage
	}
	return exitOK
}

type gateArgs struct {
	MaxLines int      `json:"max_lines"`
	Plan     []string `json:"plan"` // files the model says it will touch (FR-8 step 5)
}

// call runs one tool. A gate that fails is a normal answer (text, not an error);
// a tool that cannot do its job returns an error, which the client sees as isError.
func (s *mcpServer) call(name string, raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	switch name {
	case "axyn_plan":
		return s.toolPlan(raw)
	case "axyn_next":
		return s.toolNext()
	case "axyn_gate":
		var a gateArgs
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", err
		}
		_, report, err := s.gate(a)
		return report, err
	case "axyn_decide":
		var a struct{ Question, Answer string }
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", err
		}
		return s.toolDecide(a.Question, a.Answer)
	case "axyn_run":
		var a struct {
			Request string `json:"request"`
			Resume  bool   `json:"resume"`
		}
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", err
		}
		return s.toolRun(a.Request, a.Resume)
	case "axyn_status":
		var a struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", err
		}
		return s.toolStatus(a.ID)
	case "axyn_ship":
		var a struct {
			gateArgs
			Message string `json:"message"`
		}
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", err
		}
		return s.toolShip(a.gateArgs, a.Message)
	}
	return "", fmt.Errorf("ferramenta desconhecida: %s", name)
}

// gate runs the FR-5 gates and renders the report; green is true when nothing failed.
func (s *mcpServer) gate(a gateArgs) (bool, string, error) {
	// The caller may only make the size limit stricter, never looser.
	maxLines := s.maxLines
	if a.MaxLines > 0 && a.MaxLines < maxLines {
		maxLines = a.MaxLines
	}
	var ciLog bytes.Buffer
	n, findings, err := evalGate(s.dir, s.base, maxLines, s.ci, nil, &ciLog)
	if err != nil {
		return false, "", fmt.Errorf("git diff falhou: %v", err)
	}
	if len(a.Plan) > 0 {
		if _, t := s.openTicket(); t != nil {
			findings = append(findings, checkPlan(t, a.Plan)...)
		}
	}
	advice, recovered := s.recordAttempt(len(findings) == 0, findings, a.Plan, ciLog.String())
	if len(findings) == 0 {
		return true, fmt.Sprintf("gate: verde (%d arquivo(s) no diff)", n), nil
	}
	var b strings.Builder
	ciFailed := false
	for _, f := range findings {
		fmt.Fprintf(&b, "gate: reprovado [%s] %s\n", f.gate, f.reason)
		ciFailed = ciFailed || f.gate == "ci"
	}
	if ciFailed && ciLog.Len() > 0 && !recovered {
		b.WriteString("--- fim do log do CI ---\n" + tail(ciLog.String(), 40))
	}
	if advice != "" {
		b.WriteString(advice)
	}
	return false, strings.TrimRight(b.String(), "\n"), nil
}

// openTicket is the plan and its first ticket not yet delivered.
func (s *mcpServer) openTicket() (*plan, *ticket) {
	pl, _, err := s.loadPlan()
	if err != nil {
		return nil, nil
	}
	for i := range pl.Tickets {
		if !pl.Tickets[i].Done {
			return pl, &pl.Tickets[i]
		}
	}
	return pl, nil
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n") + "\n"
}

// statePath is where the engine keeps the plan: inside .git, so it never shows up in a diff.
func (s *mcpServer) statePath() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = s.dir
	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("não é um repositório git: %v", err)
	}
	g := strings.TrimSpace(string(b))
	if !filepath.IsAbs(g) {
		g = filepath.Join(s.dir, g)
	}
	return filepath.Join(g, "axyn", "plan.json"), nil
}

func (s *mcpServer) loadPlan() (*plan, string, error) {
	p, err := s.statePath()
	if err != nil {
		return nil, "", err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, p, fmt.Errorf("nenhum plano gravado: rode axyn_plan antes")
	}
	var pl plan
	if err := json.Unmarshal(b, &pl); err != nil {
		return nil, p, err
	}
	return &pl, p, nil
}

func (s *mcpServer) savePlan(pl *plan, p string) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(pl, "", "  ")
	return os.WriteFile(p, b, 0o644)
}

var (
	nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	acRe   = regexp.MustCompile(`\bAC-\d+\b`)
	specNo = regexp.MustCompile(`^(\d+)-`)
)

func (s *mcpServer) toolPlan(raw json.RawMessage) (string, error) {
	var a struct {
		Name    string `json:"name"`
		Spec    string `json:"spec"`
		Tickets []struct {
			Title string   `json:"title"`
			Body  string   `json:"body"`
			ACs   []string `json:"acs"`
		} `json:"tickets"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return "", err
	}
	if !nameRe.MatchString(a.Name) {
		return "", fmt.Errorf("plano recusado: name deve ser minúsculas, dígitos e hífens")
	}
	declared := map[string]bool{}
	for _, ac := range acRe.FindAllString(a.Spec, -1) {
		declared[ac] = true
	}
	if len(declared) == 0 {
		return "", fmt.Errorf("plano recusado: a spec não declara nenhum AC-N")
	}
	if len(a.Tickets) == 0 {
		return "", fmt.Errorf("plano recusado: nenhum ticket")
	}
	pl := plan{}
	for i, t := range a.Tickets {
		if strings.TrimSpace(t.Title) == "" {
			return "", fmt.Errorf("plano recusado: ticket %d sem título", i+1)
		}
		if len(t.ACs) == 0 {
			return "", fmt.Errorf("plano recusado: ticket %d (%s) não cita nenhum AC", i+1, t.Title)
		}
		for _, ac := range t.ACs {
			if !declared[ac] {
				return "", fmt.Errorf("plano recusado: ticket %d cita %s, que a spec não declara", i+1, ac)
			}
		}
		pl.Tickets = append(pl.Tickets, ticket{ID: i + 1, Title: t.Title, Body: t.Body, ACs: t.ACs})
	}

	statePath, err := s.statePath()
	if err != nil {
		return "", err
	}
	next := 1
	entries, _ := os.ReadDir(filepath.Join(s.dir, "specs"))
	for _, e := range entries {
		if m := specNo.FindStringSubmatch(e.Name()); m != nil {
			var n int
			_, _ = fmt.Sscan(m[1], &n)
			if n >= next {
				next = n + 1
			}
		}
	}
	rel := filepath.Join("specs", fmt.Sprintf("%03d-%s", next, a.Name), "spec.md")
	full := filepath.Join(s.dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(full, []byte(strings.TrimRight(a.Spec, "\n")+"\n"), 0o644); err != nil {
		return "", err
	}
	// The spec is committed on its own, so the ticket diffs that follow never touch it.
	if out, err := s.git("add", "--", rel); err != nil {
		return "", fmt.Errorf("git add falhou: %s", out)
	}
	if out, err := s.git("commit", "-m", "docs: spec "+a.Name, "--", rel); err != nil {
		return "", fmt.Errorf("git commit falhou: %s", out)
	}
	pl.Spec = rel
	if err := s.savePlan(&pl, statePath); err != nil {
		return "", err
	}
	return fmt.Sprintf("plano gravado: %s com %d ticket(s)", rel, len(pl.Tickets)), nil
}

func (s *mcpServer) toolNext() (string, error) {
	pl, _, err := s.loadPlan()
	if err != nil {
		return "", err
	}
	for _, t := range pl.Tickets {
		if t.Done {
			continue
		}
		spec, _ := os.ReadFile(filepath.Join(s.dir, pl.Spec))
		var b strings.Builder
		fmt.Fprintf(&b, "Ticket %d: %s\n\n%s\n\nACs: %s\n\n", t.ID, t.Title, t.Body, strings.Join(t.ACs, ", "))
		for _, l := range strings.Split(string(spec), "\n") {
			for _, ac := range t.ACs {
				if strings.Contains(l, ac) {
					b.WriteString(l + "\n")
					break
				}
			}
		}
		b.WriteString("\nCada AC acima deve ser citado num teste. Não altere specs, workflows nem a configuração do axyn.")
		return b.String(), nil
	}
	return "todos os tickets do plano estão entregues", nil
}

func (s *mcpServer) git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = s.dir
	b, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(b)), err
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// toolShip commits, pushes and opens the PR, but only after the gates pass on this very diff.
func (s *mcpServer) toolShip(a gateArgs, msg string) (string, error) {
	if strings.TrimSpace(msg) == "" {
		return "", fmt.Errorf("ship recusado: mensagem de commit vazia")
	}
	green, report, err := s.gate(a)
	if err != nil {
		return "", err
	}
	if !green {
		return "", fmt.Errorf("ship recusado: o portão reprovou o diff\n%s", report)
	}
	return s.deliver(msg, false)
}

// deliver commits, pushes and opens the PR for a diff that already passed the gates.
// newBranch starts a branch for the ticket even when HEAD is already on one (the run
// loop delivers one ticket after the other).
func (s *mcpServer) deliver(msg string, newBranch bool) (string, error) {

	var t *ticket
	pl, planPath, perr := s.loadPlan()
	if perr == nil {
		for i := range pl.Tickets {
			if !pl.Tickets[i].Done {
				t = &pl.Tickets[i]
				break
			}
		}
	}

	branch, _ := s.git("rev-parse", "--abbrev-ref", "HEAD")
	if newBranch || branch == "main" || branch == "master" || branch == "HEAD" {
		subject := msg
		if i := strings.Index(subject, ":"); i > 0 && i < 12 {
			subject = subject[i+1:]
		}
		slug := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(subject), "-"), "-")
		if len(slug) > 40 {
			slug = strings.Trim(slug[:40], "-")
		}
		if t != nil {
			slug = fmt.Sprintf("%d-%s", t.ID, slug)
		}
		branch = "feat/" + slug
		if out, err := s.git("checkout", "-b", branch); err != nil {
			return "", fmt.Errorf("git checkout falhou: %s", out)
		}
	}
	if out, err := s.git("add", "-A"); err != nil {
		return "", fmt.Errorf("git add falhou: %s", out)
	}
	if out, err := s.git("commit", "-m", msg); err != nil {
		return "", fmt.Errorf("git commit falhou: %s", out)
	}
	notes := []string{"commit em " + branch}

	if remotes, _ := s.git("remote"); strings.Contains(remotes, "origin") {
		if out, err := s.git("push", "-u", "origin", branch); err != nil {
			return "", fmt.Errorf("commit feito, mas o push falhou: %s", out)
		}
		notes = append(notes, "push feito")
		if _, err := exec.LookPath("gh"); err == nil {
			cmd := exec.Command("gh", "pr", "create", "--title", msg, "--body", "Entregue pelo axyn com o portão verde.")
			cmd.Dir = s.dir
			b, err := cmd.CombinedOutput()
			if err != nil {
				notes = append(notes, "PR não aberto: "+strings.TrimSpace(string(b)))
			} else {
				notes = append(notes, "PR: "+strings.TrimSpace(string(b)))
			}
		} else {
			notes = append(notes, "PR não aberto: gh não instalado")
		}
	} else {
		notes = append(notes, "sem remoto origin: push e PR pulados")
	}
	if t != nil {
		t.Done = true
		if t.Strategy == "" {
			t.Strategy = "direto"
			if len(t.Attempts) > 1 {
				t.Strategy = "nova tentativa"
			}
		}
		t.CostUSD, _ = strconv.ParseFloat(os.Getenv("AXYN_COST_USD"), 64)
		if t.Model != "" {
			notes = append(notes, fmt.Sprintf("registro: modelo %s, %d tentativa(s), estratégia %s, custo US$ %.4f", t.Model, len(t.Attempts), t.Strategy, t.CostUSD))
		}
		_ = s.savePlan(pl, planPath)
	}
	return strings.Join(notes, "\n"), nil
}
