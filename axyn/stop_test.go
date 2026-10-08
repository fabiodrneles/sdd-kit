//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// #363: axyn stop ends a run in the background, with what it started, and says how to go on.
func TestStopEndsTheBackgroundRun(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("sh", "-c", "sleep 60 & sleep 60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = cmd.Wait() }()
	now := time.Now()
	st := &runState{ID: "bench-1", Request: "avaliação dos modelos", Status: runRunning, PID: cmd.Process.Pid, Started: now, Opts: runOpts{Bench: &benchSpec{}}}
	if err := saveRun(dir, st); err != nil {
		t.Fatal(err)
	}
	var o, e strings.Builder
	if code := runStopCmd([]string{"--dir", dir}, &o, &e); code != exitOK {
		t.Fatalf("stop: %d %s", code, e.String())
	}
	time.Sleep(200 * time.Millisecond)
	if processAlive(cmd.Process.Pid) {
		t.Error("o processo continua vivo")
	}
	got, _ := loadRun(dir, "bench-1")
	if got.Status != runStopped || !strings.Contains(got.Message, "axyn bench") || !strings.Contains(o.String(), "parei a execução bench-1") {
		t.Errorf("estado: %+v\n%s", got, o.String())
	}
	o.Reset()
	if runStopCmd([]string{"--dir", dir}, &o, &e); !strings.Contains(o.String(), "nada rodando") {
		t.Errorf("de novo: %s", o.String())
	}
}

// #363: the work and the benches keep separate lists: a bench after the work does not
// hide it from status, history, decide and --resume.
func TestBenchHasItsOwnList(t *testing.T) {
	dir := t.TempDir()
	work := &runState{ID: "20261007-100000", Request: "notas apagar", Status: runStopped, Message: "o ticket 2 precisa de uma resposta sua", Started: time.Now()}
	legacy := &runState{ID: "20261007-200000", Request: "avaliação dos modelos", Status: runDone, Started: time.Now(), Opts: runOpts{Bench: &benchSpec{}}}
	bench := &runState{ID: "bench-20261007-230000", Request: "avaliação dos modelos", Status: runDone, Started: time.Now(), Opts: runOpts{Bench: &benchSpec{}}}
	for _, st := range []*runState{work, legacy, bench} {
		if err := saveRun(dir, st); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(benchRunsDir(dir), bench.ID+".json")); err != nil {
		t.Errorf("a avaliação grava na lista dela: %v", err)
	}
	if st, _ := loadRun(dir, ""); st.ID != work.ID {
		t.Errorf("o trabalho do projeto continua sendo o último: %s", st.ID)
	}
	if b, _ := loadBenchRun(dir); b.ID != bench.ID {
		t.Errorf("última avaliação: %s", b.ID)
	}
	if busy(dir) != nil {
		t.Error("nada rodando")
	}
	bench.Status, bench.PID = runRunning, 0
	_ = saveRun(dir, bench)
	if b := busy(dir); b == nil || b.ID != bench.ID {
		t.Errorf("a avaliação rodando ocupa a pasta: %+v", b)
	}
}

// #365: an untracked name with accents, and a nested repository git refuses to add, do
// not stop the gate with "exit status 128".
func TestGitDiffOddUntracked(t *testing.T) {
	dir := t.TempDir()
	run := func(d string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = d
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run(dir, "init", "-q")
	run(dir, "-c", "user.email=a@b", "-c", "user.name=a", "commit", "-q", "--allow-empty", "-m", "base")
	_ = os.WriteFile(filepath.Join(dir, "relatório.go"), []byte("package x\n"), 0o644)
	nested := filepath.Join(dir, "sub")
	_ = os.MkdirAll(nested, 0o755)
	run(nested, "init", "-q")
	_ = os.WriteFile(filepath.Join(nested, "a.txt"), []byte("x\n"), 0o644)
	diff, err := gitDiff(dir, "HEAD")
	if err != nil {
		t.Fatalf("gitDiff: %v", err)
	}
	if !strings.Contains(diff, "relatório.go") {
		t.Errorf("o arquivo com acento entra no diff:\n%s", diff)
	}
	if _, err := gitDiff(dir, "nao-existe"); err == nil || !strings.Contains(err.Error(), "git diff:") {
		t.Errorf("o erro traz a mensagem do git: %v", err)
	}
}

// #366: the sh and bash of Git for Windows, found from git --exec-path.
func TestGitShellDirs(t *testing.T) {
	got := gitShellDirs(`C:\Program Files\Git\mingw64\libexec\git-core` + "\n")
	if len(got) != 2 || got[0] != `C:\Program Files\Git\usr\bin` || got[1] != `C:\Program Files\Git\bin` {
		t.Errorf("Git for Windows: %v", got)
	}
	if got := gitShellDirs("/usr/lib/git-core"); got != nil {
		t.Errorf("fora do Windows, nada: %v", got)
	}
}

// #366: coverage profiles a model left in the tree go away before the gates; code stays.
func TestCleanCoverageProfiles(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	_ = cmd.Run()
	_ = os.WriteFile(filepath.Join(dir, "full_coverage"), []byte("mode: atomic\nx.go:1.1,2.2 1 1\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "coverage.out"), []byte("mode: set\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("mode: of thinking\n"), 0o644)
	cleanCoverageProfiles(dir)
	for _, gone := range []string{"full_coverage", "coverage.out"} {
		if _, err := os.Stat(filepath.Join(dir, gone)); err == nil {
			t.Errorf("%s ficou", gone)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Error("um arquivo que não é perfil de cobertura fica")
	}
}

// #368: new test lines do not count against the size of a ticket; production code does.
func TestCheckSizeIgnoresNewTests(t *testing.T) {
	many := make([]string, 1180)
	tests := fileDiff{path: "cmd/check_test.go", added: many}
	if f := checkSize([]fileDiff{tests}, 400); len(f) != 0 {
		t.Errorf("1180 linhas de teste novas não passam do limite: %v", f)
	}
	code := fileDiff{path: "cmd/check.go", added: make([]string, 401)}
	if f := checkSize([]fileDiff{tests, code}, 400); len(f) != 1 || !strings.Contains(f[0].reason, "1180 linhas novas de teste") {
		t.Errorf("código de produção conta: %v", f)
	}
	gone := fileDiff{path: "cmd/check_test.go", removed: make([]string, 401)}
	if f := checkSize([]fileDiff{gone}, 400); len(f) != 1 {
		t.Errorf("linhas de teste apagadas contam: %v", f)
	}
}

// #371: a make ci that failed only on the Windows cleanup is run again; a real failure is not.
func TestGateRetriesCleanupOnlyFailure(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.email=a@b", "-c", "user.name=a", "commit", "-q", "--allow-empty", "-m", "base"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	flaky := `if [ -f .tried ]; then echo "ok  x 1s"; else touch .tried; echo "ok  x 1s"; echo "go: unlinkat x.exe: being used by another process"; exit 1; fi`
	var log strings.Builder
	_, findings, err := evalGate(dir, "HEAD", 400, flaky, nil, &log)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.gate == "ci" {
			t.Errorf("a falha só de limpeza devia ser repetida e passar: %v\n%s", f, log.String())
		}
	}
	_ = os.Remove(filepath.Join(dir, ".tried"))
	_, findings, _ = evalGate(dir, "HEAD", 400, `echo "--- FAIL: TestX"; exit 1`, nil, &log)
	if len(findings) == 0 || findings[len(findings)-1].gate != "ci" {
		t.Errorf("falha de verdade continua reprovada: %v", findings)
	}
}

// #372: a tests-only ticket may not change production code, except the files the base
// errors name (the first ticket fixes them).
func TestTestsOnlyFindings(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "user.email=a@b", "-c", "user.name=a"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-q")
	_ = os.MkdirAll(filepath.Join(dir, "checker"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "cmd"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "checker", "checker.go"), []byte("package checker\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "cmd", "check.go"), []byte("package cmd\n"), 0o644)
	run("add", "-A")
	run("commit", "-q", "-m", "base")
	_ = os.WriteFile(filepath.Join(dir, "checker", "checker.go"), []byte("package checker\n// fix\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "cmd", "check.go"), []byte("package cmd\nvar timeout = 30\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "cmd", "check_test.go"), []byte("package cmd\n"), 0o644)
	s := &mcpServer{dir: dir, base: "HEAD"}
	pl := &plan{BaseErrors: []string{`checker\checker.go:71:23: Error return value of resp.Body.Close is not checked (errcheck)`}}
	tk := &ticket{Title: "Testes para a cobertura chegar a 80%"}
	got := s.testsOnlyFindings(pl, tk)
	if len(got) != 1 || !strings.Contains(got[0].reason, "cmd/check.go") {
		t.Errorf("só cmd/check.go é reprovado (checker.go corrige o erro da base; o teste é livre): %v", got)
	}
}

// #375: before an attempt of a tests-only ticket, production files go back to the base;
// the tests and the files of the base errors stay.
func TestRestoreProduction(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-c", "user.email=a@b", "-c", "user.name=a"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-q")
	_ = os.MkdirAll(filepath.Join(dir, "cmd"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "cmd", "check.go"), []byte("package cmd\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	run("add", "-A")
	run("commit", "-q", "-m", "base")
	_ = os.WriteFile(filepath.Join(dir, "cmd", "check.go"), []byte("package cmd\nvar timeout = 30\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n// fixed\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "cmd", "timeout.go"), []byte("package cmd\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "cmd", "check_test.go"), []byte("package cmd\n"), 0o644)
	s := &mcpServer{dir: dir, base: "HEAD"}
	pl := &plan{BaseErrors: []string{"main.go:1:1: something (errcheck)"}}
	put := s.restoreProduction(pl)
	if len(put) != 2 {
		t.Errorf("voltam check.go e timeout.go: %v", put)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "cmd", "check.go")); string(b) != "package cmd\n" {
		t.Errorf("check.go volta à base: %q", b)
	}
	if _, err := os.Stat(filepath.Join(dir, "cmd", "timeout.go")); err == nil {
		t.Error("arquivo novo de produção sai")
	}
	if _, err := os.Stat(filepath.Join(dir, "cmd", "check_test.go")); err != nil {
		t.Error("o teste fica")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "main.go")); !strings.Contains(string(b), "fixed") {
		t.Error("o arquivo do erro da base fica")
	}
}
