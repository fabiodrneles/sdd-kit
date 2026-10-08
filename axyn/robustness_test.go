package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// #322: on Windows without sh on the PATH, the CI runs under the sh of Git for Windows.
func TestGitShellOnWindows(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "Git", "cmd"), "git.exe", "")
	write(t, filepath.Join(root, "Git", "bin"), "sh.exe", "")
	oldOS, oldLook := hostOS, lookPath
	t.Cleanup(func() { hostOS, lookPath = oldOS, oldLook })
	hostOS = "windows"
	lookPath = func(name string) (string, error) {
		if name == "git" {
			return filepath.Join(root, "Git", "cmd", "git.exe"), nil
		}
		return "", errors.New("not found")
	}
	sh, dir := gitShell()
	if sh != filepath.Join(root, "Git", "bin", "sh.exe") || dir != filepath.Join(root, "Git", "bin") {
		t.Fatalf("sh do Git: %q %q", sh, dir)
	}
	cmd, err := shellCommand("make ci")
	if err != nil || cmd.Args[0] != sh {
		t.Fatalf("o CI deveria rodar no sh do Git: %v %v", cmd, err)
	}
	if path := strings.Join(cmd.Env, "\n"); !strings.Contains(path, "PATH="+dir) {
		t.Errorf("a pasta do sh deveria ir para o PATH do CI")
	}
	if !toolPresent("sh") {
		t.Errorf("o doctor deveria achar o sh do Git")
	}
	_ = os.Remove(sh)
	if _, err := shellCommand("make ci"); err == nil || !strings.Contains(err.Error(), "winget install -e --id Git.Git") {
		t.Errorf("sem sh, o comando para instalar o Git: %v", err)
	}
}

// #321: the gate's intent-to-add is undone, or `git stash` fails ("not uptodate").
func TestGateUndoesIntentToAdd(t *testing.T) {
	dir := repo(t)
	write(t, dir, "novo.go", "package x\n")
	if _, err := gitDiff(dir, "HEAD"); err != nil {
		t.Fatal(err)
	}
	if out, _ := (&mcpServer{dir: dir}).git("diff", "--cached", "--name-only"); out != "" {
		t.Errorf("o índice ficou com o intent-to-add: %q", out)
	}
	if out, err := (&mcpServer{dir: dir}).git("stash", "push", "-u"); err != nil {
		t.Errorf("git stash falhou depois do portão: %s", out)
	}
}

// #321: a run whose worker died reads as interrupted and can be resumed at once.
func TestDeadRunIsResumable(t *testing.T) {
	dir := repo(t)
	dead := exec.Command("true")
	if err := dead.Run(); err != nil {
		t.Skip("sem o comando true")
	}
	st := &runState{ID: "20260101-000000", Request: "algo", Status: runRunning, Phase: "código", Started: time.Now(), PID: dead.Process.Pid}
	if err := saveRun(dir, st); err != nil {
		t.Fatal(err)
	}
	if alive(st) {
		t.Fatal("o processo terminou: a execução não está viva")
	}
	if !strings.Contains(renderStatus(st), "interrompida") || !strings.Contains(renderStatus(st), "axyn run --resume") {
		t.Errorf("o status deveria dizer interrompida e como seguir: %s", renderStatus(st))
	}
	old := spawnWorker
	spawnWorker = func(_, _ string) error { return nil }
	t.Cleanup(func() { spawnWorker = old })
	text, isErr := callTool(t, dir, "axyn_run", map[string]any{"resume": true})
	if isErr || !strings.Contains(text, "Comecei a trabalhar") {
		t.Fatalf("a retomada deveria começar na hora: %v %q", isErr, text)
	}
	if last, _ := loadRun(dir, ""); last.Request != "algo" {
		t.Errorf("a retomada deveria manter o pedido: %q", last.Request)
	}
}

// #324: a ticket that stops keeps its code on a WIP branch and a help file (no remote),
// the user ends on the base, and a fix on that branch is gated first on resume.
func TestStoppedTicketHelpAndFixOnResume(t *testing.T) {
	dir := repo(t)
	base, _ := (&mcpServer{dir: dir}).git("rev-parse", "--abbrev-ref", "HEAD")
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf '\nfunc TestC(t *testing.T) { t.Skip() }\n' >> x_test.go`))
	if code, out := runLoopIn(t, dir); code == exitOK || !strings.Contains(out, "e precisa de você") {
		t.Fatalf("deveria parar com a pergunta: %d\n%s", code, out)
	}
	if cur, _ := (&mcpServer{dir: dir}).git("rev-parse", "--abbrev-ref", "HEAD"); cur != base {
		t.Errorf("o usuário deveria voltar à base, está em %s", cur)
	}
	pl, _, _ := (&mcpServer{dir: dir}).loadPlan()
	wip := pl.Tickets[0].WIP
	if wip == "" {
		t.Fatal("a branch WIP não foi gravada no plano")
	}
	help, err := os.ReadFile(filepath.Join(dir, ".axyn", "ajuda-ticket-1.md"))
	if err != nil {
		t.Fatalf("sem remoto, o arquivo de ajuda: %v", err)
	}
	for _, want := range []string{"Para pedir ajuda a outra IA", "x_test.go", "axyn run --resume", "axyn model"} {
		if !strings.Contains(string(help), want) {
			t.Errorf("a ajuda não traz %q", want)
		}
	}

	// The user fixes the code on the WIP branch (puts the test back, writes the feature).
	s := &mcpServer{dir: dir}
	if out, err := s.git("checkout", "-q", wip); err != nil {
		t.Fatal(out)
	}
	if out, err := s.git("checkout", base, "--", "x_test.go"); err != nil {
		t.Fatal(out)
	}
	write(t, dir, "um.txt", "corrigido à mão\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "fix")
	git(t, dir, "checkout", "-q", base)

	// On resume the fixed code passes the gates before any agent runs.
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf 'ticket %s\n' "$n" > "f$n.txt"`))
	code, out := runLoopIn(t, dir, "--resume")
	if code != exitOK {
		t.Fatalf("resume: %d\n%s", code, out)
	}
	if !strings.Contains(out, "os portões rodam nele primeiro") {
		t.Errorf("o código corrigido deveria passar pelos portões primeiro:\n%s", out)
	}
	if calls, _ := os.ReadFile(os.Getenv("FAKE_LOG")); strings.Count(string(calls), "axyn-code") != 1 {
		t.Errorf("o agente não deveria rodar no ticket 1 corrigido, só no 2:\n%s", calls)
	}
}

// #321: a resume that finds the code of an interrupted attempt keeps it in a WIP commit.
func TestResumeSavesInterruptedCode(t *testing.T) {
	dir := repo(t)
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf '\nfunc TestC(t *testing.T) { t.Skip() }\n' >> x_test.go`))
	if code, _ := runLoopIn(t, dir); code == exitOK {
		t.Fatal("deveria parar")
	}
	write(t, dir, "meio.txt", "tentativa interrompida\n") // the worker died mid-attempt
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf 'ok %s\n' "$n" > "g$n.txt"`))
	code, out := runLoopIn(t, dir, "--resume")
	if strings.Contains(out, "alterações que não são do axyn") {
		t.Fatalf("a retomada não deveria recusar o código interrompido:\n%s", out)
	}
	_ = code
	found := false
	for _, b := range strings.Fields(mustGit(t, dir, "branch", "--list", "feat/*-wip*", "--format=%(refname:short)")) {
		if strings.Contains(mustGit(t, dir, "show", "--stat", b), "meio.txt") {
			found = true
		}
	}
	if !found {
		t.Errorf("o código interrompido deveria estar num commit WIP")
	}
}

func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, _ := (&mcpServer{dir: dir}).git(args...)
	return out
}

// #324: with a remote and gh, the stopped ticket becomes a draft PR.
func TestStoppedTicketBecomesDraftPR(t *testing.T) {
	dir := cloned(t)
	bin := t.TempDir()
	log := filepath.Join(bin, "gh.log")
	write(t, bin, "gh", "#!/bin/sh\necho \"$@\" >> '"+log+"'\necho https://github.com/o/r/pull/9\n")
	if err := os.Chmod(filepath.Join(bin, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf '\nfunc TestC(t *testing.T) { t.Skip() }\n' >> x_test.go`))
	code, out := runLanding(t, dir)
	if code == exitOK {
		t.Fatal("deveria parar")
	}
	calls, _ := os.ReadFile(log)
	if !strings.Contains(string(calls), "pr create --draft --title [não passou nos portões]") {
		t.Errorf("deveria abrir um PR em rascunho:\n%s", calls)
	}
	if !strings.Contains(out, "PR em rascunho") || !strings.Contains(out, "pull/9") {
		t.Errorf("a mensagem deveria apontar o PR:\n%s", out)
	}
}
