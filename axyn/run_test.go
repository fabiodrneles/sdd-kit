package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const planRequest = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"axyn_plan","arguments":{"name":"demo","spec":"- **FR-1** x\n- **AC-1** dado a, então b\n- **AC-2** dado c, então d\n","tickets":[{"title":"Um","body":"primeiro","acs":["AC-1"]},{"title":"Dois","body":"segundo","acs":["AC-2"]}]}}}`

// fakeAgent writes a stand-in for `opencode run --agent NAME ...`. The planner calls the
// real axyn mcp server (so axyn_plan validates the plan); the coder does what `code` says.
func fakeAgent(t *testing.T, code string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "axyn")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	dir := t.TempDir()
	write(t, dir, "plan.json", planRequest+"\n")
	write(t, dir, "fake.sh", `echo "$@" >> "$FAKE_LOG"
case "$3" in
axyn-plan) '`+bin+`' mcp < '`+filepath.Join(dir, "plan.json")+`' > /dev/null ;;
axyn-code)
	n=$(printf '%s' "$*" | sed -n 's/.*Ticket \([0-9]*\):.*/\1/p' | head -n 1)
`+code+`
	;;
esac
`)
	t.Setenv("FAKE_LOG", filepath.Join(dir, "calls.log"))
	return "sh " + filepath.Join(dir, "fake.sh")
}

func runLoopIn(t *testing.T, dir string, extra ...string) (int, string) {
	t.Helper()
	var out, errb bytes.Buffer
	args := append([]string{"--wait", "--dir", dir, "--ci", "true", "--config", ""}, extra...)
	args = append(args, "crie uma coisa")
	code := runRunCmd(args, &out, &errb)
	return code, out.String() + errb.String()
}

func subjects(t *testing.T, dir string) []string {
	t.Helper()
	out, _ := (&mcpServer{dir: dir}).git("log", "--reverse", "--format=%s")
	return strings.Split(strings.TrimSpace(out), "\n")
}

// 021 AC-7: a fake coding agent and a request with two tickets; the engine alone takes
// both through the gates and into commits, in order.
func TestRunTwoTicketsInOrder(t *testing.T) {
	dir := repo(t)
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf 'ticket %s\n' "$n" > "f$n.txt"`))

	code, out := runLoopIn(t, dir)
	if code != exitOK {
		t.Fatalf("code %d\n%s", code, out)
	}
	got := subjects(t, dir)
	want := []string{"init", "docs: spec demo", "feat: Um", "feat: Dois"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("commits %v, want %v", got, want)
	}
	for _, s := range []string{"concluído", "entregue: ticket 1", "entregue: ticket 2"} {
		if !strings.Contains(out, s) {
			t.Errorf("status sem %q:\n%s", s, out)
		}
	}
	calls, _ := os.ReadFile(os.Getenv("FAKE_LOG"))
	if n := strings.Count(string(calls), "run --agent axyn-code"); n != 2 {
		t.Errorf("agente de código chamado %d vez(es), quer 2:\n%s", n, calls)
	}
	if !strings.HasPrefix(strings.Split(string(calls), "\n")[0], "run --agent axyn-plan") {
		t.Errorf("a primeira chamada deveria ser o plano:\n%s", calls)
	}
	if out, _ := (&mcpServer{dir: dir}).git("status", "--porcelain"); out != "" {
		t.Errorf("o estado do motor vazou para o git: %s", out)
	}
}

// 021 AC-7: a fake agent that deletes a test never gets the ticket delivered.
func TestRunAgentDeletingTestIsNotDelivered(t *testing.T) {
	dir := repo(t)
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `rm -f x_test.go`))

	code, out := runLoopIn(t, dir)
	if code != exitFail {
		t.Fatalf("code %d, quer %d\n%s", code, exitFail, out)
	}
	for _, s := range subjects(t, dir) {
		if strings.HasPrefix(s, "feat:") {
			t.Errorf("ticket entregue com um teste apagado: %s", s)
		}
	}
	if !strings.Contains(out, "parado") || !strings.Contains(out, "axyn_decide") || !strings.Contains(out, "arquivo de teste apagado") {
		t.Errorf("status sem o motivo e a pergunta:\n%s", out)
	}
	s := &mcpServer{dir: dir}
	if _, tk := s.openTicket(); tk == nil || tk.ID != 1 || len(tk.Attempts) != maxFailsPerModel-1 {
		t.Errorf("o ticket 1 deveria seguir aberto, com %d tentativas: %+v", maxFailsPerModel-1, tk)
	}
	// every retry carries the engine's verdict back to the agent
	calls, _ := os.ReadFile(os.Getenv("FAKE_LOG"))
	if !strings.Contains(string(calls), "reprovada pelo motor") {
		t.Errorf("a nova tentativa não recebeu o diagnóstico:\n%s", calls)
	}
}

func TestRunAgentCommandMissing(t *testing.T) {
	dir := repo(t)
	t.Setenv("AXYN_OPENCODE", "/nonexistent/opencode")
	code, out := runLoopIn(t, dir)
	if code != exitFail || !strings.Contains(out, "não consegui chamar") {
		t.Fatalf("code %d\n%s", code, out)
	}
}

func TestRunRefusesDirtyTree(t *testing.T) {
	dir := repo(t)
	write(t, dir, "x.go", "package x\n// mexido\n")
	t.Setenv("AXYN_OPENCODE", "/nonexistent/opencode")
	code, out := runLoopIn(t, dir)
	if code != exitFail || !strings.Contains(out, "alterações") {
		t.Fatalf("code %d\n%s", code, out)
	}
}

// 021 FR-9: axyn_run only starts the work and returns an id; axyn_status shows it.
func TestMCPRunStartsInBackgroundAndStatusShowsIt(t *testing.T) {
	dir := repo(t)
	var started string
	old := spawnWorker
	spawnWorker = func(_, id string) error { started = id; return nil }
	defer func() { spawnWorker = old }()

	text, isErr := callTool(t, dir, "axyn_run", map[string]any{"request": "crie uma landing page"})
	if isErr || started == "" || !strings.Contains(text, started) {
		t.Fatalf("axyn_run: isErr %v, %q, id %q", isErr, text, started)
	}
	text, isErr = callTool(t, dir, "axyn_status", map[string]any{"id": started})
	if isErr || !strings.Contains(text, "rodando") || !strings.Contains(text, "crie uma landing page") {
		t.Fatalf("axyn_status: isErr %v, %q", isErr, text)
	}
	if text, _ = callTool(t, dir, "axyn_status", map[string]any{}); !strings.Contains(text, started) {
		t.Errorf("sem id, o status deveria mostrar a última execução: %q", text)
	}
	if text, isErr = callTool(t, dir, "axyn_run", map[string]any{"request": "outro"}); !isErr || !strings.Contains(text, "em andamento") {
		t.Errorf("duas execuções ao mesmo tempo: isErr %v, %q", isErr, text)
	}
	if out, _ := (&mcpServer{dir: dir}).git("status", "--porcelain"); out != "" {
		t.Errorf(".axyn deveria ficar fora do git: %s", out)
	}
}

func TestMCPRunRefusesEmptyRequestAndBadID(t *testing.T) {
	dir := repo(t)
	if text, isErr := callTool(t, dir, "axyn_run", map[string]any{"request": "  "}); !isErr || !strings.Contains(text, "vazio") {
		t.Errorf("pedido vazio: isErr %v, %q", isErr, text)
	}
	if _, isErr := callTool(t, dir, "axyn_status", map[string]any{"id": "../x"}); !isErr {
		t.Error("id com caminho deveria ser recusado")
	}
}

func TestCommandIsASingleInstruction(t *testing.T) {
	for _, bad := range []string{"axyn_next", "axyn_gate", "axyn_ship", "axyn-code"} {
		if strings.Contains(commandTemplate, bad) {
			t.Errorf("o comando /axyn não deve citar %s: o motor conduz o laço", bad)
		}
	}
	if !strings.Contains(commandTemplate, "axyn_run") || !strings.Contains(commandTemplate, "$ARGUMENTS") {
		t.Error("o comando /axyn deve chamar axyn_run com $ARGUMENTS")
	}
}
