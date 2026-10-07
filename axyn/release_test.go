package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fakeReleaseScript = `#!/bin/sh
echo "$@" >> .release-calls
case "$1" in
--dry-run) printf 'sdd-release: versão v1.2.0 calculada pelo go-release-manager\n  | ### Adicionado\n  | - flag --timeout no check (#4)\n' ;;
--tag) echo "sdd-release: tag v$2 disparada" ;;
*) echo "sdd-pr: PR #9 aberto: https://github.com/o/r/pull/9" ;;
esac
`

func releaseRepo(t *testing.T) string {
	t.Helper()
	dir := repo(t)
	write(t, filepath.Join(dir, "scripts"), "sdd-release.sh", fakeReleaseScript)
	old := lookPath
	lookPath = func(name string) (string, error) {
		if name == "gh" || name == "go" {
			return "/usr/bin/" + name, nil
		}
		return old(name)
	}
	t.Cleanup(func() { lookPath = old })
	return dir
}

func calls(t *testing.T, dir string) string {
	b, _ := os.ReadFile(filepath.Join(dir, ".release-calls"))
	return string(b)
}

// #340: axyn release shows the version and what goes in it, and opens the PR only on a yes.
func TestReleaseAsksBeforeOpening(t *testing.T) {
	dir := releaseRepo(t)
	oldIn := stdin
	t.Cleanup(func() { stdin = oldIn })
	stdin = strings.NewReader("n\n")
	var o, e strings.Builder
	if code := runReleaseCmd([]string{"--dir", dir}, &o, &e); code != exitOK {
		t.Fatalf("release: %d %s", code, e.String())
	}
	for _, want := range []string{"A próxima versão é v1.2.0", "go-release-manager", "- flag --timeout no check (#4)", "Abrir o PR de fechamento da v1.2.0? (s/N)", "nada foi feito"} {
		if !strings.Contains(o.String(), want) {
			t.Errorf("a saída não traz %q:\n%s", want, o.String())
		}
	}
	if c := calls(t, dir); strings.TrimSpace(c) != "--dry-run" {
		t.Errorf("sem o sim, só a simulação: %q", c)
	}
	stdin = strings.NewReader("s\n")
	o.Reset()
	if code := runReleaseCmd([]string{"--dir", dir}, &o, &e); code != exitOK || !strings.Contains(o.String(), "PR de fechamento aberto: https://github.com/o/r/pull/9") {
		t.Errorf("com o sim, abre o PR: %d\n%s", code, o.String())
	}
	o.Reset()
	if code := runReleaseCmd([]string{"--dir", dir, "--tag", "v1.2.0"}, &o, &e); code != exitOK || !strings.Contains(calls(t, dir), "--tag 1.2.0") {
		t.Errorf("--tag: %d %s", code, calls(t, dir))
	}
}

// #340: without the script or the tools, the axyn says what to do.
func TestReleaseMissing(t *testing.T) {
	dir := repo(t)
	if msg := releaseMissing(dir); !strings.Contains(msg, "axyn init") {
		t.Errorf("sem o script: %q", msg)
	}
	write(t, filepath.Join(dir, "scripts"), "sdd-release.sh", fakeReleaseScript)
	old := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("não") }
	t.Cleanup(func() { lookPath = old })
	if msg := releaseMissing(dir); !strings.Contains(msg, "gh, go") {
		t.Errorf("sem gh e go: %q", msg)
	}
}

// #340: the /axyn tool previews without confirm and opens the PR with it.
func TestReleaseTool(t *testing.T) {
	dir := releaseRepo(t)
	s := &mcpServer{dir: dir}
	out, err := s.toolRelease(nil)
	if err != nil || !strings.Contains(out, "v1.2.0") || !strings.Contains(out, "Pergunte ao usuário") {
		t.Errorf("prévia: %v %q", err, out)
	}
	raw, _ := json.Marshal(map[string]any{"confirm": true})
	if out, err := s.toolRelease(raw); err != nil || !strings.Contains(out, "pull/9") {
		t.Errorf("com confirm: %v %q", err, out)
	}
}
