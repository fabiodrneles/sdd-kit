package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const goodPlan = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"axyn_plan","arguments":{"name":"apagar","spec":"- **FR-1** apagar uma nota pelo ID, com confirmação\n- **AC-1** dado uma nota 1, quando apagar 1 e confirmar com s, então a nota some\n- **AC-2** dado nenhuma nota com ID 9, quando apagar 9, então mostra erro de nota inexistente\n- **AC-3** dado a confirmação recusada com N, quando apagar, então nada muda e mostra cancelado\n- **AC-4** dado a lista vazia, quando apagar 1, então mostra erro\n","tickets":[{"title":"Apagar com confirmação","body":"comando apagar","acs":["AC-1","AC-3"]},{"title":"Erros","body":"ID inexistente e lista vazia","acs":["AC-2","AC-4"]}]}}}`

// benchAgent is a stand-in for opencode: model "bom" solves every task, "trapaceiro"
// weakens or edits the tests.
func benchAgent(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "axyn")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	ref, _ := filepath.Abs("testdata/bench")
	dir := t.TempDir()
	write(t, dir, "plan.json", goodPlan+"\n")
	write(t, dir, "fake.sh", `model="$5"; prompt="$6"
case "$3:$model" in
axyn-plan:bom) '`+bin+`' mcp < '`+filepath.Join(dir, "plan.json")+`' > /dev/null ;;
axyn-code:bom)
	case "$prompt" in
	*slug.go*) cp '`+ref+`/slug.go' slug.go ;;
	*price_test.go*) cp '`+ref+`/price_test.go' price_test.go ;;
	*cart.go*) cp '`+ref+`/cart.go' cart.go ;;
	esac ;;
*:semchave) echo x >> '`+filepath.Join(dir, "calls")+`'; echo "Error: 401 Unauthorized: No auth credentials found" ;;
axyn-code:trapaceiro)
	case "$prompt" in
	*slug.go*) printf 'package slug\n' > slug_test.go ;;
	*price_test.go*) printf 'package price\n\nimport "testing"\n\nfunc TestNada(t *testing.T) { _, _ = Category(1) }\n' > price_test.go ;;
	*cart.go*) printf 'package cart\n' > cart_test.go ;;
	esac ;;
esac
`)
	return "sh " + filepath.Join(dir, "fake.sh")
}

// #344: the bench scores each model per step with deterministic checks, vetoes the one
// that weakens tests, routes each step to the best model, and the engine follows it.
func TestBenchClassifiesAndRoutes(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("sem go")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("AXYN_OPENCODE", benchAgent(t))
	var o, e strings.Builder
	if code := runBenchCmd([]string{"--models", "bom,trapaceiro", "--runs", "1", "--timeout", "2m"}, &o, &e); code != exitOK {
		t.Fatalf("bench: %d %s\n%s", code, e.String(), o.String())
	}
	out := o.String()
	for _, want := range []string{"bom, plano: nota 100", "trapaça (veto)", "contido (vetado por trapaça)", "## Para que o axyn vai usar cada modelo", "aplicada"} {
		if !strings.Contains(out, want) {
			t.Errorf("a saída não traz %q:\n%s", want, out)
		}
	}
	p, err := loadProfile()
	if err != nil || !p.Applied {
		t.Fatalf("perfil: %v %+v", err, p)
	}
	for _, role := range []string{roleplan, roleCode, roleTests, roleFix} {
		if got := p.Routing[role]; len(got) != 2 || got[0] != "bom" || got[1] != "trapaceiro" {
			t.Errorf("%s: o bom primeiro, e o trapaceiro depois, nunca descartado: %v", role, got)
		}
		if got := p.Contained[role]; len(got) != 1 || got[0] != "trapaceiro" {
			t.Errorf("%s: o trapaceiro roda contido: %v", role, got)
		}
	}
	if !isContained(&ticket{Title: "x"}, "trapaceiro") || isContained(&ticket{Title: "x"}, "bom") {
		t.Error("só o trapaceiro roda no modo guiado")
	}
	tests := findResult(p.Results, "bom", "tests-001")
	if tests == nil || tests.score() < 90 || !strings.Contains(strings.Join(tests.Runs[0].Notes, " "), "8 de 8 bugs injetados pegos") {
		t.Errorf("testes do modelo bom: %+v", tests)
	}
	if plan := findResult(p.Results, "bom", "plan-001"); plan == nil || plan.score() != 100 {
		t.Errorf("plano completo deveria ter 100: %+v", plan)
	}
	// The engine follows the applied routing.
	r := &runner{s: &mcpServer{dir: t.TempDir(), fallback: true}, st: &runState{}}
	if got := r.planModel(); got != "bom" {
		t.Errorf("planejador: %q", got)
	}
	if ms := r.ladderFor(&ticket{Title: "Testes para a cobertura chegar a 80%"}); len(ms) == 0 || ms[0].ID != "bom" {
		t.Errorf("escada do ticket de testes: %+v", ms)
	}
	md, _ := filepath.Glob(filepath.Join(benchDir(), "bench-*.md"))
	if len(md) == 0 {
		t.Error("o relatório deveria ficar gravado para auditoria")
	}
	if diffs, _ := filepath.Glob(filepath.Join(benchDir(), "bench-*", "bom-code-001-*.diff")); len(diffs) == 0 {
		t.Error("o código de cada tentativa deveria ficar gravado para auditoria")
	} else if b, _ := os.ReadFile(diffs[0]); !strings.Contains(string(b), "func Make") {
		t.Errorf("o diff da tentativa: %s", b)
	}
	// Lifecycle: an old evaluation, or a new model, makes the profile stale.
	if why := p.stale([]string{"bom", "novo"}); !strings.Contains(why, "novo") {
		t.Errorf("modelo novo: %q", why)
	}
	p.Date = time.Now().Add(-40 * 24 * time.Hour)
	if why := p.stale(nil); !strings.Contains(why, "30 dias") {
		t.Errorf("avaliação velha: %q", why)
	}
	if code := runBenchCmd([]string{"--set", "plano=meu-modelo"}, &o, &e); code != exitOK {
		t.Fatal("--set")
	}
	if got := r.planModel(); got != "meu-modelo" {
		t.Errorf("etapa fixada à mão: %q", got)
	}
	if code := runBenchCmd([]string{"--off"}, &o, &e); code != exitOK {
		t.Fatal("--off")
	}
	if got := r.planModel(); got == "bom" {
		t.Error("desligada, o planejador volta à escada")
	}
	_ = os.Remove(profilePathFile())
}

// #344: the guide rejects what a contained model does outside its leash, whatever it is.
func TestContainmentGuide(t *testing.T) {
	dir := repo(t)
	s := &mcpServer{dir: dir, contained: true}
	tk := &ticket{Title: "Ajustar", Body: "mude só o um.txt"}
	write(t, dir, "um.txt", "novo\n")
	if f := s.containFindings(tk); len(f) != 0 {
		t.Errorf("dentro do escopo, nada: %v", f)
	}
	write(t, dir, "outro.go", "package x\n")
	write(t, dir, "x_test.go", "package x\n\n// mexido\n")
	write(t, dir, "novo_test.go", "package x\n")
	f := s.containFindings(tk)
	got := ""
	for _, x := range f {
		got += x.reason + "\n"
	}
	if !strings.Contains(got, "fora do ticket: outro.go") || !strings.Contains(got, "teste que já existe: x_test.go") || strings.Contains(got, "novo_test.go") {
		t.Errorf("a guia: %s", got)
	}
}

func TestAutoParallel(t *testing.T) {
	if n := autoParallel(); n < 1 || n > 3 {
		t.Errorf("paralelismo: %d", n)
	}
	if totalMemory() == 0 {
		t.Log("memória desconhecida nesta máquina")
	}
}

// Evaluating one task again keeps the other tasks' results of the last round.
func TestBenchOneTaskKeepsTheRest(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("sem go")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("AXYN_OPENCODE", benchAgent(t))
	var o, e strings.Builder
	if code := runBenchCmd([]string{"--models", "bom", "--timeout", "2m"}, &o, &e); code != exitOK {
		t.Fatalf("bench: %d %s", code, e.String())
	}
	// #359: a model without a key, left from an older round, goes away.
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	old, _ := loadProfile()
	old.Models = append(old.Models, "openrouter/x:free")
	old.Results = append(old.Results, benchResult{Model: "openrouter/x:free", Task: "code-001", Role: roleCode, Runs: []benchRun{{}}})
	_ = saveProfile(old)
	o.Reset()
	if code := runBenchCmd([]string{"--models", "bom", "--tasks", "plan", "--timeout", "2m"}, &o, &e); code != exitOK {
		t.Fatalf("bench --tasks plan: %d %s", code, e.String())
	}
	if strings.Contains(o.String(), "openrouter/x:free") {
		t.Error("modelo sem chave de uma rodada anterior continua no relatório")
	}
	if !strings.Contains(o.String(), "forte *") || !strings.Contains(o.String(), "code-001 (tentativa 1, rodada de ") {
		t.Errorf("o resultado mantido da rodada anterior precisa estar marcado:\n%s", o.String())
	}
	p, _ := loadProfile()
	for _, task := range []string{"plan-001", "code-001", "tests-001", "fix-001"} {
		if findResult(p.Results, "bom", task) == nil {
			t.Errorf("%s sumiu ao refazer só o plano", task)
		}
	}
}

// On Windows a locked test binary fails Go's cleanup although the tests passed: that is
// the machine, not the model.
func TestOnlyCleanupFailed(t *testing.T) {
	win := "ok  \tbenchtests\t3.294s\tcoverage: 100.0% of statements\ngo: unlinkat C:\\Users\\G\\AppData\\Local\\Temp\\go-build1\\b001\\benchtests.test.exe: O arquivo já está sendo usado por outro processo."
	if !onlyCleanupFailed(win) {
		t.Error("só a limpeza falhou: os testes passaram")
	}
	real := "--- FAIL: TestMake (0.00s)\nFAIL\nFAIL\tbenchcode\t1.2s\ngo: unlinkat x.exe: O arquivo já está sendo usado por outro processo."
	if onlyCleanupFailed(real) {
		t.Error("um teste que falhou continua reprovado")
	}
	if onlyCleanupFailed("ok  \tx\t1s") {
		t.Error("sem erro de limpeza, nada a relevar")
	}
}

func TestKeyAvailable(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENROUTER_API_KEY", "")
	if keyAvailable("openrouter/x:free") {
		t.Error("sem a chave, o modelo do OpenRouter fica de fora")
	}
	if !keyAvailable("opencode/nemotron-free") {
		t.Error("modelo do opencode não precisa de chave")
	}
	t.Setenv("OPENROUTER_API_KEY", "x")
	if !keyAvailable("openrouter/x:free") {
		t.Error("com a chave na variável, entra")
	}
	t.Setenv("OPENROUTER_API_KEY", "")
	write(t, filepath.Join(os.Getenv("XDG_DATA_HOME"), "opencode"), "auth.json", `{"openrouter":{"type":"api","key":"k"}}`)
	if !keyAvailable("openrouter/x:free") {
		t.Error("com a chave do opencode auth login, entra")
	}
}

// #361: a model without a valid key (or down) answers the first task with the provider's
// error; it is unavailable, not a zero, and its other tasks are skipped in this round.
func TestBenchDownModelIsSkipped(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("sem go")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	agent := benchAgent(t)
	t.Setenv("AXYN_OPENCODE", agent)
	var o, e strings.Builder
	if code := runBenchCmd([]string{"--models", "semchave", "--timeout", "2m"}, &o, &e); code != exitOK {
		t.Fatalf("bench: %d %s", code, e.String())
	}
	calls, _ := os.ReadFile(filepath.Join(filepath.Dir(strings.TrimPrefix(agent, "sh ")), "calls"))
	if n := strings.Count(string(calls), "x"); n != 1 {
		t.Errorf("o modelo sem chave foi chamado %d vezes; devia ser 1", n)
	}
	if !strings.Contains(o.String(), "pulado") || strings.Contains(o.String(), "semchave, plano: nota") {
		t.Errorf("as outras tarefas ficam como indisponíveis:\n%s", o.String())
	}
}
