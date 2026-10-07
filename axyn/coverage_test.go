package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// #334: code without any test is rejected; code with a test passes.
func TestCodeNeedsTests(t *testing.T) {
	code := fileDiff{path: "cmd/check.go", added: []string{"x := 1"}}
	if f := checkCodeHasTests([]fileDiff{code}); len(f) != 1 || !strings.Contains(f[0].reason, "sem nenhum teste novo") {
		t.Errorf("código sem teste deveria reprovar: %v", f)
	}
	test := fileDiff{path: "cmd/check_test.go", added: []string{"func TestX(t *testing.T) {}"}}
	if f := checkCodeHasTests([]fileDiff{code, test}); len(f) != 0 {
		t.Errorf("com teste, passa: %v", f)
	}
	if f := checkCodeHasTests([]fileDiff{{path: "README.md", added: []string{"x"}}}); len(f) != 0 {
		t.Errorf("documentação não precisa de teste: %v", f)
	}
}

// #334: the added lines of a Go ticket must be covered; the finding names the lines.
func TestPatchCoverage(t *testing.T) {
	dir := repo(t)
	prof := profilePath(dir)
	if err := os.MkdirAll(filepath.Dir(prof), 0o755); err != nil {
		t.Fatal(err)
	}
	since := time.Now().Add(-time.Second)
	write(t, filepath.Dir(prof), "coverage.out", "mode: atomic\nmod/checker/checker.go:10.1,12.2 2 1\nmod/checker/checker.go:13.1,16.2 3 0\n")
	diff := "diff --git a/checker/checker.go b/checker/checker.go\n--- a/checker/checker.go\n+++ b/checker/checker.go\n@@ -9,2 +10,7 @@\n+a\n+b\n+c\n+d\n+e\n ctx\n"
	files := parseDiff(strings.NewReader(diff))
	if got := files[0].addedAt; len(got) != 5 || got[0] != 10 || got[4] != 14 {
		t.Fatalf("números das linhas adicionadas: %v", got)
	}
	f := checkPatchCoverage(dir, files, since)
	if len(f) != 1 || !strings.Contains(f[0].reason, "60%") || !strings.Contains(f[0].reason, "checker/checker.go linhas 13,14") {
		t.Errorf("cobertura do código novo: %v", f)
	}
	if f := checkPatchCoverage(dir, files, time.Now().Add(time.Hour)); len(f) != 0 {
		t.Errorf("sem perfil novo, nada a medir: %v", f)
	}
}

// #334: the axyn measures the coverage with the project's own tests, the gates use it as
// the floor, and each delivered ticket raises it in the Makefile, up to the goal.
func TestCoverageRatchet(t *testing.T) {
	dir := repo(t)
	write(t, dir, "Makefile", "COVERAGE_MIN ?= 80\n\nci:\n\t@echo ok\n")
	s := &mcpServer{dir: dir, ci: "echo 'cobertura: 35.7% (mínimo 0%)'"}
	pl := &plan{Spec: "specs/x.md", Tickets: []ticket{{ID: 1, Title: "t"}}}
	p, _ := s.statePath()
	if err := s.savePlan(pl, p); err != nil {
		t.Fatal(err)
	}
	r := &runner{s: s, st: &runState{}}
	if msg := r.measureCoverage(); !strings.Contains(msg, "35.7%") || !strings.Contains(msg, "meta 80%") {
		t.Errorf("medição: %q", msg)
	}
	if floor, ok := coverageFloor(dir); !ok || floor != 35 {
		t.Errorf("o piso deveria ser 35: %d %v", floor, ok)
	}
	lastCoverage = 52.3
	if msg := r.raiseCoverage(); !strings.Contains(msg, "52%") {
		t.Errorf("o mínimo deveria subir para 52: %q", msg)
	}
	mk, _ := os.ReadFile(filepath.Join(dir, "Makefile"))
	if !strings.Contains(string(mk), "COVERAGE_MIN ?= 52\n") || !strings.Contains(string(mk), "até a meta de 80%") {
		t.Errorf("Makefile:\n%s", mk)
	}
	if goal, _ := makefileGoal(dir); goal != 80 {
		t.Errorf("a meta continua 80: %d", goal)
	}
	lastCoverage = 40
	_ = r.raiseCoverage()
	if floor, _ := coverageFloor(dir); floor != 52 {
		t.Errorf("o piso nunca desce: %d", floor)
	}
	lastCoverage = 95
	_ = r.raiseCoverage()
	if _, ok := coverageFloor(dir); ok {
		t.Errorf("na meta, vale o Makefile")
	}
	lastCoverage = -1
}

// #334: under the goal, the axyn says how much is missing and where, asks once who writes
// the tests, and auto puts a tests ticket before the others.
func TestCoverageQuestionAndChoice(t *testing.T) {
	dir := repo(t)
	write(t, dir, "Makefile", "COVERAGE_MIN ?= 80\n")
	s := &mcpServer{dir: dir, ci: "echo 'cobertura: 35.7% (mínimo 0%)'"}
	p, _ := s.statePath()
	if err := s.savePlan(&plan{Spec: "specs/x.md", Tickets: []ticket{{ID: 1, Title: "pedido"}}}, p); err != nil {
		t.Fatal(err)
	}
	r := &runner{s: s, st: &runState{}}
	_ = r.measureCoverage()
	t.Setenv("AXYN_COVERAGE", "")
	q := r.coverageQuestion()
	for _, want := range []string{"35.7%", "meta é 80%", "faltam 44.3 pontos", "axyn coverage auto", "axyn coverage manual", "AXYN_COVERAGE"} {
		if !strings.Contains(q, want) {
			t.Errorf("a pergunta não traz %q:\n%s", want, q)
		}
	}
	var o, e strings.Builder
	if code := runCoverageCmd([]string{"--dir", dir, "--no-resume", "auto"}, &o, &e); code != exitOK {
		t.Fatalf("coverage auto: %d %s", code, e.String())
	}
	pl, _, _ := s.loadPlan()
	if len(pl.Tickets) != 2 || !strings.Contains(pl.Tickets[0].Title, "cobertura chegar a 80%") || pl.Tickets[0].ID != 2 {
		t.Errorf("o ticket de testes deveria vir antes: %+v", pl.Tickets)
	}
	if q := r.coverageQuestion(); q != "" {
		t.Errorf("com a escolha feita, não pergunta de novo: %q", q)
	}
	o.Reset()
	runCoverageCmd([]string{"--dir", dir}, &o, &e)
	if !strings.Contains(o.String(), "o axyn") || !strings.Contains(o.String(), "meta: 80%") {
		t.Errorf("axyn coverage mostra o estado: %s", o.String())
	}
}

// #334: AXYN_COVERAGE answers the question in advance.
func TestCoverageChoiceFromEnv(t *testing.T) {
	dir := repo(t)
	write(t, dir, "Makefile", "COVERAGE_MIN ?= 80\n")
	s := &mcpServer{dir: dir, ci: "echo 'cobertura: 0.0% (mínimo 0%)'"}
	p, _ := s.statePath()
	_ = s.savePlan(&plan{Spec: "specs/x.md", Tickets: []ticket{{ID: 1, Title: "pedido"}}}, p)
	r := &runner{s: s, st: &runState{}}
	_ = r.measureCoverage()
	t.Setenv("AXYN_COVERAGE", "manual")
	if q := r.coverageQuestion(); q != "" {
		t.Errorf("AXYN_COVERAGE=manual não deveria perguntar: %q", q)
	}
	if pl, _, _ := s.loadPlan(); pl.CoverageChoice != "manual" || len(pl.Tickets) != 1 {
		t.Errorf("manual não cria ticket: %+v", pl)
	}
}
