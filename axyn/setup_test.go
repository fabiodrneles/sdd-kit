package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGH is a gh stand-in whose GitHub lives in files under the returned directory:
// repo.json, actions.json, wf.json, protection (exists = protected), noauth, plan403.
// Every write to the API is logged in writes.
const fakeGHScript = `#!/bin/sh
S="$GHS"
if [ "$1" = auth ]; then [ ! -e "$S/noauth" ]; exit $?; fi
if [ "$1" = repo ] && [ "$2" = create ]; then
	echo "repo create $*" >> "$S/writes"
	git remote add origin "https://github.com/ana/$3.git"; exit 0
fi
shift # api
m=GET
if [ "$1" = -X ]; then m="$2"; shift 2; fi
p="$1"
[ "$m" = GET ] || echo "$m $*" >> "$S/writes"
case "$m $p" in
"GET user") echo '{"id":7,"login":"ana","name":"Ana"}' ;;
"GET repos/o/r") cat "$S/repo.json" ;;
"PATCH repos/o/r") echo '{"allow_auto_merge":true,"delete_branch_on_merge":true}' > "$S/repo.json" ;;
"GET repos/o/r/actions/permissions") cat "$S/actions.json" ;;
"PUT repos/o/r/actions/permissions") echo '{"enabled":true}' > "$S/actions.json" ;;
"GET repos/o/r/actions/permissions/workflow") cat "$S/wf.json" ;;
"PUT repos/o/r/actions/permissions/workflow") echo '{"default_workflow_permissions":"write","can_approve_pull_request_reviews":true}' > "$S/wf.json" ;;
"GET repos/o/r/branches/main/protection") [ -e "$S/protection" ] || { echo "gh: Not Found (HTTP 404)"; exit 1; } ;;
"PUT repos/o/r/branches/main/protection")
	if [ -e "$S/plan403" ]; then echo "gh: Upgrade to GitHub Pro or make this repository public (HTTP 403)"; exit 1; fi
	cat > "$S/protection" ;;
*) echo "gh falso: $m $p" >&2; exit 1 ;;
esac
`

func fakeGH(t *testing.T, configured bool) string {
	t.Helper()
	bin, state := t.TempDir(), t.TempDir()
	write(t, bin, "gh", fakeGHScript)
	if err := os.Chmod(filepath.Join(bin, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GHS", state)
	if configured {
		write(t, state, "repo.json", `{"allow_auto_merge":true,"delete_branch_on_merge":true}`)
		write(t, state, "actions.json", `{"enabled":true}`)
		write(t, state, "wf.json", `{"default_workflow_permissions":"write","can_approve_pull_request_reviews":true}`)
	} else {
		write(t, state, "repo.json", `{"allow_auto_merge":false,"delete_branch_on_merge":false}`)
		write(t, state, "actions.json", `{"enabled":false}`)
		write(t, state, "wf.json", `{"default_workflow_permissions":"read","can_approve_pull_request_reviews":false}`)
	}
	old := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/true", nil }
	t.Cleanup(func() { lookPath = old })
	return state
}

// githubRepoDir is a repository with an identity and origin on github.com/o/r.
func githubRepoDir(t *testing.T) string {
	t.Helper()
	dir := emptyRepo(t)
	git(t, dir, "config", "user.name", "t")
	git(t, dir, "config", "user.email", "t@t")
	git(t, dir, "remote", "add", "origin", "git@github.com:o/r.git")
	return dir
}

func setupCmd(t *testing.T, fix bool, args ...string) (int, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := runSetupCmd(args, &out, &errb, fix)
	return code, out.String() + errb.String()
}

func writes(state string) string {
	b, _ := os.ReadFile(filepath.Join(state, "writes"))
	return string(b)
}

// 021 FR-1: a repository already configured passes the doctor, and the setup writes nothing.
func TestDoctorOnConfiguredRepo(t *testing.T) {
	state := fakeGH(t, true)
	dir := githubRepoDir(t)
	if code, out := setupCmd(t, false, "--dir", dir); code != exitOK || !strings.Contains(out, "tudo certo") {
		t.Fatalf("doctor: code %d\n%s", code, out)
	}
	if code, out := setupCmd(t, true, "--dir", dir); code != exitOK {
		t.Fatalf("setup: code %d\n%s", code, out)
	}
	if w := writes(state); w != "" {
		t.Errorf("o setup escreveu num repositório já configurado:\n%s", w)
	}
}

// 021 FR-1: Actions without write and auto-merge off: the doctor says so, the setup makes
// the right calls, and the doctor passes afterwards.
func TestSetupConfiguresActionsAndAutoMerge(t *testing.T) {
	state := fakeGH(t, false)
	dir := githubRepoDir(t)
	code, out := setupCmd(t, false, "--dir", dir)
	if code == exitOK || !strings.Contains(out, "GitHub Actions: precisa") || !strings.Contains(out, "merge automático: permitir") {
		t.Fatalf("doctor deveria apontar o que falta: code %d\n%s", code, out)
	}
	if w := writes(state); w != "" {
		t.Errorf("o doctor não pode escrever:\n%s", w)
	}
	if code, out := setupCmd(t, true, "--dir", dir); code != exitOK {
		t.Fatalf("setup: code %d\n%s", code, out)
	}
	w := writes(state)
	for _, want := range []string{
		"PUT repos/o/r/actions/permissions -F enabled=true",
		"PUT repos/o/r/actions/permissions/workflow -f default_workflow_permissions=write -F can_approve_pull_request_reviews=true",
		"PATCH repos/o/r -F allow_auto_merge=true -F delete_branch_on_merge=true",
	} {
		if !strings.Contains(w, want) {
			t.Errorf("faltou a chamada %q:\n%s", want, w)
		}
	}
	if strings.Contains(w, "protection") {
		t.Errorf("protegeu a main sem --protect-main:\n%s", w)
	}
	if code, out := setupCmd(t, false, "--dir", dir); code != exitOK {
		t.Errorf("doctor depois do setup: code %d\n%s", code, out)
	}
}

func TestSetupWithoutGHOrLogin(t *testing.T) {
	state := fakeGH(t, true)
	dir := githubRepoDir(t)
	write(t, state, "noauth", "")
	if code, out := setupCmd(t, true, "--dir", dir); code == exitOK || !strings.Contains(out, "gh auth login") {
		t.Errorf("sem login: code %d\n%s", code, out)
	}
	lookPath = func(name string) (string, error) {
		if name == "gh" {
			return "", errors.New("not found")
		}
		return "/usr/bin/true", nil
	}
	if code, out := setupCmd(t, false, "--dir", dir); code == exitOK || !strings.Contains(out, "gh (GitHub CLI): instale:") {
		t.Errorf("sem gh: code %d\n%s", code, out)
	}
}

// The free plan has no protection on private repositories: warn and go on.
func TestSetupProtectMainOnPlanWithout(t *testing.T) {
	state := fakeGH(t, true)
	dir := githubRepoDir(t)
	write(t, state, "plan403", "")
	code, out := setupCmd(t, true, "--dir", dir, "--protect-main")
	if code != exitOK || !strings.Contains(out, "aviso        proteção da main: o plano") {
		t.Fatalf("403 do plano deveria ser aviso: code %d\n%s", code, out)
	}
	_ = os.Remove(filepath.Join(state, "plan403"))
	if code, out := setupCmd(t, true, "--dir", dir, "--protect-main"); code != exitOK || !strings.Contains(out, "configurado  proteção da main") {
		t.Fatalf("proteção: code %d\n%s", code, out)
	}
	if b, _ := os.ReadFile(filepath.Join(state, "protection")); !strings.Contains(string(b), `"contexts":["make ci"]`) {
		t.Errorf("a proteção deveria exigir o make ci: %s", b)
	}
}

// No remote and no identity: the setup creates the private repo and takes the identity
// from the GitHub account.
func TestSetupCreatesRepoAndIdentity(t *testing.T) {
	state := fakeGH(t, true)
	dir := emptyRepo(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "none"))
	code, out := setupCmd(t, false, "--dir", dir)
	if code == exitOK || !strings.Contains(out, "gh repo create") || !strings.Contains(out, "identidade do git") {
		t.Fatalf("doctor deveria pedir o repositório e a identidade: %d\n%s", code, out)
	}
	code, out = setupCmd(t, true, "--dir", dir)
	if !strings.Contains(writes(state), "repo create repo create "+filepath.Base(dir)+" --private --source . --remote origin --push") {
		t.Errorf("gh repo create errado:\n%s\n%s", writes(state), out)
	}
	if name, _ := (&mcpServer{dir: dir}).git("config", "user.email"); name != "7+ana@users.noreply.github.com" {
		t.Errorf("identidade: %q\n%s", name, out)
	}
	if !strings.Contains(out, "configurado  repositório no GitHub: ana/") {
		t.Errorf("sem o repositório criado:\n%s (code %d)", out, code)
	}
}

func TestGithubRepo(t *testing.T) {
	for in, want := range map[string]string{
		"https://github.com/o/r.git":              "o/r",
		"git@github.com:o/r.git":                  "o/r",
		"http://local_proxy@127.0.0.1:1/git/o/r":  "",
		"https://x-access-token:t@github.com/o/r": "o/r",
		"ssh://git@github.com/o/r":                "o/r",
	} {
		if got := githubRepo(in); got != want {
			t.Errorf("githubRepo(%q) = %q, quero %q", in, got, want)
		}
	}
}
