package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #330: a stopped ticket explains what was tried, the real error and the ways forward;
// axyn decide records the instruction, gives the ticket new attempts and the coder sees it.
func TestStopExplainsAndDecideResumes(t *testing.T) {
	dir := repo(t)
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `rm -f x_test.go`))
	code, out := runLoopIn(t, dir)
	if code == exitOK {
		t.Fatalf("deveria parar:\n%s", out)
	}
	st, _ := loadRun(dir, "")
	for _, want := range []string{"O axyn parou no ticket 1 de", "O que foi tentado:", "1. primeira tentativa", "O erro da última tentativa:", "O que isso quer dizer:", "axyn decide \"", "axyn model", "git checkout", "axyn history"} {
		if !strings.Contains(st.Message, want) {
			t.Errorf("a mensagem não traz %q:\n%s", want, st.Message)
		}
	}
	if strings.Contains(st.Message, "ambíguo") {
		t.Errorf("a mensagem não deveria falar em ticket ambíguo:\n%s", st.Message)
	}
	st.Request = "" // a run from an older axyn kept no request
	_ = saveRun(dir, st)

	old := spawnWorker
	spawnWorker = func(_, _ string) error { return nil }
	t.Cleanup(func() { spawnWorker = old })
	var o, e strings.Builder
	if code := runDecideCmd([]string{"--dir", dir, "não apague o x_test.go"}, &o, &e); code != exitOK {
		t.Fatalf("decide: %d %s", code, e.String())
	}
	s := &mcpServer{dir: dir}
	pl, _, _ := s.loadPlan()
	spec, _ := os.ReadFile(filepath.Join(dir, pl.Spec))
	if !strings.Contains(string(spec), "não apague o x_test.go") {
		t.Errorf("a decisão deveria estar na spec:\n%s", spec)
	}
	if tk := pl.Tickets[0]; len(tk.Attempts) != 0 || len(tk.Earlier) == 0 {
		t.Errorf("a decisão deveria dar novas tentativas e guardar as antigas: %d %d", len(tk.Attempts), len(tk.Earlier))
	}
	if prompt, _ := s.toolNext(); !strings.Contains(prompt, "Decisões do usuário") || !strings.Contains(prompt, "não apague o x_test.go") {
		t.Errorf("o modelo deveria receber a decisão:\n%s", prompt)
	}
	if !strings.Contains(o.String(), "retomada em segundo plano") {
		t.Errorf("a execução deveria retomar:\n%s", o.String())
	}
	if last, _ := loadRun(dir, ""); !strings.Contains(last.Request, "retomada do plano") {
		t.Errorf("sem o pedido antigo, o status deveria mostrar o plano: %q", last.Request)
	}
}

func TestCIErrorsAndMeaning(t *testing.T) {
	out := "go vet ./...\ncmd/check.go:42:12: Error return value of `resp.Body.Close` is not checked (errcheck)\nmake: *** [ci] Error 1\n"
	got := ciErrors(out, 6)
	if len(got) != 1 || !strings.Contains(got[0], "cmd/check.go:42") {
		t.Errorf("as linhas de erro do CI: %q", got)
	}
	if what, hint := meaning("[ci] `make ci` falhou; erros: " + got[0]); !strings.Contains(what, "erro devolvido") || !strings.Contains(hint, "if err != nil") {
		t.Errorf("errcheck em palavras simples: %q %q", what, hint)
	}
}

// #330: a project with code and no tests starts the coverage minimum at 0, and a
// coverage failure is explained as such.
func TestCoverageForUntestedProject(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	if !codeWithoutTests(dir) {
		t.Fatal("código sem testes")
	}
	if _, err := applyTemplate(dir, "go"); err != nil {
		t.Fatal(err)
	}
	mk, _ := os.ReadFile(filepath.Join(dir, "Makefile"))
	if !strings.Contains(string(mk), "COVERAGE_MIN ?= 0\n") || strings.Contains(string(mk), "COVERAGE_MIN ?= 80") {
		t.Errorf("o mínimo deveria começar em 0:\n%s", mk)
	}
	write(t, dir, "main_test.go", "package main\n")
	if codeWithoutTests(dir) {
		t.Error("com um teste, o projeto não está sem testes")
	}
	got := ciErrors("cobertura: 0.0% (mínimo 80%)\ncobertura abaixo do mínimo; suba os testes ou ajuste COVERAGE_MIN no Makefile\nmake: *** [Makefile:26: test] Error 1\n", 6)
	if len(got) != 2 {
		t.Errorf("as linhas da cobertura: %q", got)
	}
	if what, _ := meaning(strings.Join(got, " | ")); !strings.Contains(what, "coberta por testes") {
		t.Errorf("a cobertura em palavras simples: %q", what)
	}
}
