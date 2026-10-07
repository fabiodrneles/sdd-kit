package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const twoTests = `package x

import "testing"

func TestA(t *testing.T) {
	if 1 != 1 {
		t.Fatal("a")
	}
}

func TestB(t *testing.T) {
	if 2 != 2 {
		t.Fatal("b")
	}
}
`

func repo(t *testing.T) string {
	// the server under test commits too, and CI runners have no git identity
	for _, k := range []string{"GIT_AUTHOR", "GIT_COMMITTER"} {
		t.Setenv(k+"_NAME", "t")
		t.Setenv(k+"_EMAIL", "t@t")
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	write(t, dir, "x_test.go", twoTests)
	write(t, dir, "x.go", "package x\n")
	write(t, dir, "specs/spec.md", "spec\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "init")
	return dir
}

func gate(dir string, extra ...string) (string, int) {
	out, _, code := runCLI(append([]string{"gate", "--dir", dir, "--ci", ""}, extra...)...)
	return out, code
}

// 021 FR-5: a clean diff passes the gates.
func TestGateGreen(t *testing.T) {
	dir := repo(t)
	write(t, dir, "x.go", "package x\n\nfunc F() {}\n")
	write(t, dir, "x_new_test.go", "package x\n") // new code comes with a test (#334)
	out, code := gate(dir)
	if code != exitOK || !strings.Contains(out, "verde") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

// 021 AC-1: a fake agent that deletes a test fails the gate with the reason.
func TestGateDeletedTest(t *testing.T) {
	dir := repo(t)
	write(t, dir, "x_test.go", strings.Replace(twoTests, "func TestB", "func helperB", 1))
	out, code := gate(dir)
	if code != exitGateFailed || !strings.Contains(out, "teste(s) a menos") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

// 021 AC-1: a deleted test file fails too.
func TestGateDeletedTestFile(t *testing.T) {
	dir := repo(t)
	if err := os.Remove(filepath.Join(dir, "x_test.go")); err != nil {
		t.Fatal(err)
	}
	out, code := gate(dir)
	if code != exitGateFailed || !strings.Contains(out, "arquivo de teste apagado: x_test.go") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

// 021 FR-5: fewer assertions and skipped tests fail the gate.
func TestGateLoosenedAssertionAndSkip(t *testing.T) {
	dir := repo(t)
	body := strings.Replace(twoTests, `t.Fatal("b")`, "_ = 0", 1)
	body = strings.Replace(body, "func TestA(t *testing.T) {\n", "func TestA(t *testing.T) {\n\tt.Skip(\"later\")\n", 1)
	write(t, dir, "x_test.go", body)
	out, code := gate(dir)
	if code != exitGateFailed || !strings.Contains(out, "asserção(ões) a menos") || !strings.Contains(out, "pulado") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

// 021 FR-5: protected paths cannot change, including new files.
func TestGateProtectedPaths(t *testing.T) {
	dir := repo(t)
	write(t, dir, "specs/spec.md", "changed\n")
	write(t, dir, ".github/workflows/ci.yml", "on: push\n")
	out, code := gate(dir)
	if code != exitGateFailed || !strings.Contains(out, "protegido alterado: specs/spec.md") ||
		!strings.Contains(out, "protegido alterado: .github/workflows/ci.yml") {
		t.Fatalf("code %d, out %q", code, out)
	}
	// The CI definition: editing it could make "make ci" skip the tests.
	write(t, dir, "Makefile", "ci:\n\ttrue\n")
	write(t, dir, "web/eslint.config.js", "export default []\n")
	out, _ = gate(dir)
	if !strings.Contains(out, "protegido alterado: Makefile") || !strings.Contains(out, "protegido alterado: web/eslint.config.js") {
		t.Fatalf("CI definition not protected: %q", out)
	}
	write(t, dir, "docs/a.md", "a\n")
	out, _ = gate(dir, "--protect", "docs")
	if !strings.Contains(out, "protegido alterado: docs/a.md") {
		t.Fatalf("out %q", out)
	}
}

// 021 FR-5: the diff size limit.
func TestGateSizeLimit(t *testing.T) {
	dir := repo(t)
	write(t, dir, "x.go", "package x\n"+strings.Repeat("// c\n", 10))
	out, code := gate(dir, "--max-lines", "5")
	if code != exitGateFailed || !strings.Contains(out, "passa do limite de 5") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

// 021 FR-5: a failing project CI fails the gate.
func TestGateCI(t *testing.T) {
	dir := repo(t)
	var o, e strings.Builder
	if code := run([]string{"gate", "--dir", dir, "--ci", "exit 1"}, &o, &e); code != exitGateFailed || !strings.Contains(o.String(), "[ci]") {
		t.Fatalf("code %d, out %q", code, o.String())
	}
	o.Reset()
	if code := run([]string{"gate", "--dir", dir, "--ci", "true"}, &o, &e); code != exitOK {
		t.Fatalf("code %d, out %q", code, o.String())
	}
}
