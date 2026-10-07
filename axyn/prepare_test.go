package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// emptyRepo is a git repository with only a README, like a freshly created GitHub repo.
func emptyRepo(t *testing.T) string {
	t.Helper()
	for _, k := range []string{"GIT_AUTHOR", "GIT_COMMITTER"} {
		t.Setenv(k+"_NAME", "t")
		t.Setenv(k+"_EMAIL", "t@t")
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	write(t, dir, "README.md", "# demo\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "init")
	return dir
}

// fakeTools makes every tool look installed, and `make` a stand-in that passes, so the
// run loop's gates do not need node or the network.
func fakeTools(t *testing.T) {
	t.Helper()
	old := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/true", nil }
	t.Cleanup(func() { lookPath = old })
	bin := t.TempDir()
	write(t, bin, "make", "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(filepath.Join(bin, "make"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func runRequest(t *testing.T, dir, request string, extra ...string) (int, string) {
	t.Helper()
	var out, errb bytes.Buffer
	args := append([]string{"--wait", "--dir", dir, "--config", ""}, extra...)
	code := runRunCmd(append(args, request), &out, &errb)
	return code, out.String() + errb.String()
}

// 021 FR-4, AC-4: an empty repository gets the web template in its own commit before the
// first ticket, and the tickets then go through the gates.
func TestRunPreparesEmptyRepoWithTheStackTemplate(t *testing.T) {
	dir := emptyRepo(t)
	fakeTools(t)
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf '<!doctype html><title>x</title>\n' > "p$n.html"`))

	code, out := runRequest(t, dir, "crie uma landing page")
	if code != exitOK {
		t.Fatalf("code %d\n%s", code, out)
	}
	subs := subjects(t, dir)
	prep := -1
	for i, s := range subs {
		if s == "chore: prepare project (axyn, stack web)" {
			prep = i
		}
	}
	if prep < 0 || !strings.HasPrefix(subs[len(subs)-1], "feat:") || prep > len(subs)-3 {
		t.Fatalf("o commit de preparo deveria vir antes dos tickets: %q", subs)
	}
	for _, f := range []string{"Makefile", ".github/workflows/ci.yml", ".htmlvalidate.json", "scripts/links.sh", "CONTRIBUTING.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("o preparo não criou %s", f)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "README.md")); string(b) != "# demo\n" {
		t.Errorf("o preparo sobrescreveu o README: %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "CONTRIBUTING.md")); strings.Contains(string(b), "{{") {
		t.Errorf("marcadores não substituídos no CONTRIBUTING.md")
	}
	if !hasCI(dir) {
		t.Errorf("o Makefile preparado não tem o alvo ci")
	}

	// The prepared project is not prepared again.
	head, _ := (&mcpServer{dir: dir}).git("rev-parse", "HEAD")
	r := &runner{s: &mcpServer{dir: dir, ci: defaultCICmd}, st: &runState{Request: "mais uma página"}, log: &bytes.Buffer{}}
	if msg := r.prepare(); msg != "" {
		t.Errorf("projeto preparado deveria seguir: %s", msg)
	}
	if again, _ := (&mcpServer{dir: dir}).git("rev-parse", "HEAD"); again != head {
		t.Errorf("preparou de novo um projeto que já tem make ci")
	}
}

// 021 FR-4: files without a recognizable stack make the engine ask, and the answer, a
// decision in the spec, picks the template on resume.
func TestRunAsksForTheStackAndUsesTheAnswer(t *testing.T) {
	dir := emptyRepo(t)
	write(t, dir, "main.c", "int main(void){return 0;}\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "c")
	fakeTools(t)
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf 'x\n' > "f$n.html"`))

	code, out := runRequest(t, dir, "crie uma coisa")
	if code == exitOK || !strings.Contains(out, askMarker) || !strings.Contains(out, stackQuestion) {
		t.Fatalf("deveria parar com a pergunta da stack, code %d\n%s", code, out)
	}
	if calls, _ := os.ReadFile(os.Getenv("FAKE_LOG")); strings.Contains(string(calls), "axyn-code") {
		t.Errorf("chamou o agente de código antes de preparar o projeto")
	}
	if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
		t.Errorf("aplicou um template sem saber a stack")
	}

	s := &mcpServer{dir: dir}
	if _, err := s.toolDecide("Qual é a stack do projeto?", "web, por enquanto"); err != nil {
		t.Fatal(err)
	}
	if code, out := runRequest(t, dir, "", "--resume"); code != exitOK {
		t.Fatalf("resume: code %d\n%s", code, out)
	}
	if !contains(subjects(t, dir), "chore: prepare project (axyn, stack web)") {
		t.Errorf("a resposta não escolheu o template web: %q", subjects(t, dir))
	}
}

// 021 FR-4: a missing tool stops the run before the first ticket, with what to install,
// and no attempt is spent.
func TestRunStopsOnMissingToolBeforeAnyAttempt(t *testing.T) {
	dir := emptyRepo(t)
	old := lookPath
	lookPath = func(name string) (string, error) {
		if name == "node" {
			return "", errors.New("not found")
		}
		return "/usr/bin/true", nil
	}
	t.Cleanup(func() { lookPath = old })
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf 'x\n' > "f$n.html"`))

	code, out := runRequest(t, dir, "crie uma landing page")
	if code == exitOK || !strings.Contains(out, "precisa de node") || !strings.Contains(out, "Para instalar, rode:") || !strings.Contains(out, "nenhuma tentativa foi gasta") {
		t.Fatalf("deveria parar pedindo o node, code %d\n%s", code, out)
	}
	if calls, _ := os.ReadFile(os.Getenv("FAKE_LOG")); strings.Contains(string(calls), "axyn-code") {
		t.Errorf("gastou tentativa sem a ferramenta")
	}
	if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
		t.Errorf("aplicou o template sem a ferramenta")
	}
}

// A Makefile of the user's own without the ci target is never replaced: the run stops
// and says what to add.
func TestRunStopsOnMakefileWithoutCITarget(t *testing.T) {
	dir := emptyRepo(t)
	write(t, dir, "Makefile", "build:\n\techo ok\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "mk")
	fakeTools(t)
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf 'x\n' > "f$n.html"`))
	code, out := runRequest(t, dir, "crie uma landing page")
	if code == exitOK || !strings.Contains(out, "não tem o alvo ci") {
		t.Fatalf("deveria parar pedindo o alvo ci, code %d\n%s", code, out)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "Makefile")); string(b) != "build:\n\techo ok\n" {
		t.Errorf("alterou o Makefile do usuário")
	}
}

func TestDetectStack(t *testing.T) {
	for _, c := range []struct {
		files   []string
		request string
		want    string
	}{
		{nil, "crie uma landing page", "web"},
		{nil, "uma API em Python com FastAPI", "python"},
		{nil, "um app React", "node"},
		{[]string{"package.json"}, "", "node"},
		{[]string{"go.mod"}, "", "go"},
		{[]string{"pyproject.toml"}, "", "python"},
		{[]string{"pom.xml"}, "", "java"},
		{[]string{"app/app.csproj"}, "", "dotnet"},
		{[]string{"Cargo.toml"}, "", "rust"},
		{[]string{"index.html", "style.css"}, "crie uma API em python", "web"},
		{[]string{"main.c"}, "", ""},
		{[]string{"README.md", "docs/x.md"}, "", "web"},
	} {
		dir := t.TempDir()
		for _, f := range c.files {
			write(t, dir, f, "x\n")
		}
		if got := detectStack(dir, c.request); got != c.want {
			t.Errorf("arquivos %v, pedido %q: stack %q, quero %q", c.files, c.request, got, c.want)
		}
	}
}

func TestInitIsIdempotentAndNeverOverwrites(t *testing.T) {
	dir := emptyRepo(t)
	write(t, dir, "CONTRIBUTING.md", "meu\n")
	var out, errb bytes.Buffer
	if code := runInitCmd([]string{"--dir", dir, "--stack", "node"}, &out, &errb); code != exitOK {
		t.Fatalf("code %d: %s", code, errb.String())
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "CONTRIBUTING.md")); string(b) != "meu\n" {
		t.Errorf("o init sobrescreveu um arquivo existente")
	}
	if !hasCI(dir) {
		t.Errorf("o init não criou o make ci")
	}
	out.Reset()
	if code := runInitCmd([]string{"--dir", dir, "--stack", "node"}, &out, &errb); code != exitOK || !strings.Contains(out.String(), " 0 arquivo(s)") {
		t.Errorf("o segundo init deveria não criar nada: %d %s", code, out.String())
	}
	if code := runInitCmd([]string{"--dir", t.TempDir(), "--stack", "cobol"}, &out, &errb); code != exitUsage {
		t.Errorf("stack desconhecida deveria dar uso, deu %d", code)
	}
}

// The embedded copy is the kit's template/ (make axyn-templates regenerates it).
func TestEmbeddedTemplatesMatchTheKit(t *testing.T) {
	for _, part := range append([]string{"common", "seed"}, stacks...) {
		src := filepath.Join("..", "template", part)
		seen := map[string]bool{}
		_ = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(src, p)
			seen[filepath.ToSlash(rel)] = true
			want, _ := os.ReadFile(p)
			got, err := templates.ReadFile(path.Join("templates", part, filepath.ToSlash(rel)))
			if err != nil || !bytes.Equal(got, want) {
				t.Errorf("axyn/templates/%s/%s diverge de template/ (rode make axyn-templates)", part, rel)
			}
			return nil
		})
		_ = fs.WalkDir(templates, path.Join("templates", part), func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && !seen[strings.TrimPrefix(p, "templates/"+part+"/")] {
				t.Errorf("%s sobrou em axyn/templates (rode make axyn-templates)", p)
			}
			return err
		})
	}
}

// 021 AC-4: the prepared web project's `make ci` fails an invalid HTML and passes a
// valid one. Needs make and node (npx); skipped without them.
func TestPreparedWebCIRejectsInvalidHTML(t *testing.T) {
	for _, tool := range []string{"make", "npx"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("sem %s", tool)
		}
	}
	if testing.Short() {
		t.Skip("-short")
	}
	dir := emptyRepo(t)
	if _, err := applyTemplate(dir, "web"); err != nil {
		t.Fatal(err)
	}
	ci := func() error {
		cmd := exec.Command("make", "ci")
		cmd.Dir = dir
		return cmd.Run()
	}
	write(t, dir, "index.html", "<html><body><p>sem doctype<div></p></body></html>\n")
	if ci() == nil {
		t.Errorf("make ci aprovou um HTML inválido")
	}
	valid, err := os.ReadFile(filepath.Join("..", "template", "skeleton", "web", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	write(t, dir, "index.html", string(valid))
	if err := ci(); err != nil {
		t.Errorf("make ci reprovou um HTML válido: %v", err)
	}
}

// The hint is a command to paste, per system, with every missing tool in it.
func TestInstallCommands(t *testing.T) {
	old := lookPath
	t.Cleanup(func() { lookPath = old })
	lookPath = func(name string) (string, error) {
		if name == "apt-get" || name == "brew" {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
	for _, c := range []struct {
		goos    string
		missing []string
		want    []string
	}{
		{"linux", []string{"make", "node"}, []string{"sudo apt-get update && sudo apt-get install -y make nodejs npm"}},
		{"linux", []string{"node", "npm"}, []string{"sudo apt-get update && sudo apt-get install -y nodejs npm"}},
		{"darwin", []string{"python3"}, []string{"brew install python"}},
		{"windows", []string{"make", "node"}, []string{"winget install -e --id ezwinports.make", "winget install -e --id OpenJS.NodeJS.LTS"}},
		{"linux", []string{"cargo"}, []string{rustupCmd}},
	} {
		if got := installCommands(c.missing, c.goos); strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("%s %v: %q, quero %q", c.goos, c.missing, got, c.want)
		}
	}
	lookPath = func(string) (string, error) { return "", errors.New("not found") }
	if got := installCommands([]string{"node"}, "darwin"); len(got) != 2 || !strings.Contains(got[0], "Homebrew/install") {
		t.Errorf("sem brew no macOS, o primeiro comando instala o Homebrew: %q", got)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
