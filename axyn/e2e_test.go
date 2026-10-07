package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// cloned returns a clone of a bare "remote" holding one commit, the user's starting point.
func cloned(t *testing.T) string {
	t.Helper()
	src := repo(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	git(t, src, "clone", "-q", "--bare", ".", remote)
	dir := filepath.Join(t.TempDir(), "meu-site")
	git(t, filepath.Dir(dir), "clone", "-q", remote, dir)
	return dir
}

func runLanding(t *testing.T, dir string) (int, string) {
	t.Helper()
	var out, errb strings.Builder
	args := []string{"--wait", "--dir", dir, "--ci", "true", "--config", "", "crie uma landing page"}
	code := runRunCmd(args, &out, &errb)
	return code, out.String() + errb.String()
}

// 021 AC-5 (CI side): in a cloned repository with axyn installed, "crie uma landing page"
// ends in a spec and one commit per ticket, run by the fake agent.
func TestE2ELandingPage(t *testing.T) {
	dir := cloned(t)
	if out, _, code := runCLI("install", "--dir", dir); code != exitOK {
		t.Fatalf("install: %d %s", code, out)
	}
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf '<h1>ticket %s</h1>\n' "$n" > "index$n.html"`))

	if code, out := runLanding(t, dir); code != exitOK {
		t.Fatalf("code %d\n%s", code, out)
	}
	for branch, f := range map[string]string{"feat/demo-1-um": "index1.html", "feat/demo-2-dois": "index2.html"} {
		if got := strings.Join(branchSubjects(t, dir, branch), "|"); !strings.HasSuffix(got, "|docs: spec demo|feat: "+map[string]string{"index1.html": "Um", "index2.html": "Dois"}[f]) {
			t.Errorf("%s: commits %s", branch, got)
		}
		if _, err := (&mcpServer{dir: dir}).git("cat-file", "-e", branch+":"+f); err != nil {
			t.Errorf("falta %s em %s", f, branch)
		}
	}
}

// 021 AC-5: when the agent breaks the project the engine stops with the reason and the
// project's tests are still there.
func TestE2EStopsWithReason(t *testing.T) {
	dir := cloned(t)
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `rm -f x_test.go`))

	code, out := runLanding(t, dir)
	if code != exitFail {
		t.Fatalf("code %d, want %d\n%s", code, exitFail, out)
	}
	if !strings.Contains(out, "teste") {
		t.Errorf("a parada não diz o motivo:\n%s", out)
	}
	// the broken attempt is parked on a WIP branch; the commit the user cloned is intact
	if out, err := (&mcpServer{dir: dir}).git("show", "origin/HEAD:x_test.go"); err != nil || out == "" {
		t.Errorf("o teste do projeto sumiu: %v", err)
	}
	if out, _ := (&mcpServer{dir: dir}).git("branch", "--list", "feat/*-wip"); out == "" {
		t.Errorf("o trabalho a meio não foi guardado numa branch WIP")
	}
}
