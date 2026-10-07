package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"Página inicial com título e botão": "pagina-inicial-com-titulo-e-botao",
		"Ação: São João — Ñandú!":           "acao-sao-joao-nandu",
		"  Um  ":                            "um",
	} {
		if got := slugify(in, 40); got != want {
			t.Errorf("slugify(%q) = %q, quero %q", in, got, want)
		}
	}
	if got := specSlug("specs/003-landing-page/spec.md"); got != "landing-page" {
		t.Errorf("specSlug = %q", got)
	}
}

// 021 FR-9: a second request numbers its tickets from 1 again; with the same titles the
// branches still differ and both requests deliver.
func TestRunSecondRequestWithSameTitlesDelivers(t *testing.T) {
	dir := repo(t)
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf 'a %s\n' "$n" > "a$n.txt"`))
	if code, out := runLoopIn(t, dir); code != exitOK {
		t.Fatalf("primeiro pedido: code %d\n%s", code, out)
	}
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf 'b %s\n' "$n" > "b$n.txt"`))
	if code, out := runLoopIn(t, dir); code != exitOK {
		t.Fatalf("segundo pedido: code %d\n%s", code, out)
	}
	out, _ := (&mcpServer{dir: dir}).git("branch", "--list", "feat/*", "--format=%(refname:short)")
	branches := strings.Fields(out)
	if len(branches) != 4 {
		t.Fatalf("quero 4 branches de ticket, tenho %v", branches)
	}
	for _, b := range branches {
		if got := branchSubjects(t, dir, b); strings.HasPrefix(got[len(got)-2], "feat:") {
			t.Errorf("%s leva o commit de outro ticket: %v", b, got)
		}
	}
}

// The PR targets the branch the run started on.
func TestDeliverOpensThePRAgainstTheBase(t *testing.T) {
	dir := cloned(t)
	base, _ := (&mcpServer{dir: dir}).git("rev-parse", "--abbrev-ref", "HEAD")
	bin := t.TempDir()
	log := filepath.Join(bin, "gh.log")
	write(t, bin, "gh", "#!/bin/sh\necho \"$@\" >> '"+log+"'\necho https://github.com/o/r/pull/1\n")
	if err := os.Chmod(filepath.Join(bin, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf 'x %s\n' "$n" > "x$n.txt"`))
	if code, out := runLanding(t, dir); code != exitOK {
		t.Fatalf("code %d\n%s", code, out)
	}
	calls, _ := os.ReadFile(log)
	for _, l := range strings.Split(strings.TrimSpace(string(calls)), "\n") {
		if !strings.Contains(l, "--base "+base) || !strings.Contains(l, "--head feat/demo-") {
			t.Errorf("pr create sem --base %s/--head do ticket: %s", base, l)
		}
	}
	if n := strings.Count(string(calls), "pr create"); n != 2 {
		t.Errorf("quero 2 PRs, tenho %d:\n%s", n, calls)
	}
}
