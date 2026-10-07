package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

// templates is a copy of the kit's template/ (make axyn-templates), so a project is
// prepared offline and in the axyn's own version (spec 021 FR-4).
//
//go:embed all:templates
var templates embed.FS

var stacks = []string{"web", "node", "python", "go", "java", "dotnet", "rust"}

// stackQuestion is how the engine asks for the stack; the answer comes back as a decision.
const stackQuestion = "qual é a stack do projeto?"

// lookPath is replaced in tests.
var lookPath = exec.LookPath

// stackTools are the tools the stack's `make ci` runs, beyond git.
var stackTools = map[string][]string{
	"web": {"make", "node"}, "node": {"make", "node", "npm"}, "python": {"make", "python3"},
	"go": {"make", "go"}, "java": {"make", "java"}, "dotnet": {"make", "dotnet"}, "rust": {"make", "cargo"},
}

// pkgs is the package of each tool per package manager, for a copy-and-paste install
// command (the user runs what axyn prints, like a framework's own hint).
var pkgs = map[string]map[string]string{
	"git":     {"apt": "git", "dnf": "git", "pacman": "git", "brew": "git", "winget": "Git.Git"},
	"make":    {"apt": "make", "dnf": "make", "pacman": "make", "brew": "make", "winget": "ezwinports.make"},
	"node":    {"apt": "nodejs npm", "dnf": "nodejs npm", "pacman": "nodejs npm", "brew": "node", "winget": "OpenJS.NodeJS.LTS"},
	"npm":     {"apt": "nodejs npm", "dnf": "nodejs npm", "pacman": "nodejs npm", "brew": "node", "winget": "OpenJS.NodeJS.LTS"},
	"python3": {"apt": "python3 python3-pip python3-venv", "dnf": "python3 python3-pip", "pacman": "python python-pip", "brew": "python", "winget": "Python.Python.3.12"},
	"go":      {"apt": "golang-go", "dnf": "golang", "pacman": "go", "brew": "go", "winget": "GoLang.Go"},
	"java":    {"apt": "openjdk-21-jdk maven", "dnf": "java-21-openjdk-devel maven", "pacman": "jdk21-openjdk maven", "brew": "openjdk@21 maven", "winget": "Microsoft.OpenJDK.21"},
	"dotnet":  {"apt": "dotnet-sdk-8.0", "dnf": "dotnet-sdk-8.0", "pacman": "dotnet-sdk", "brew": "dotnet-sdk", "winget": "Microsoft.DotNet.SDK.8"},
}

const rustupCmd = "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y"

// packageManager is the one the install command uses on this system ("" when none known).
func packageManager(goos string) string {
	switch goos {
	case "windows":
		return "winget"
	case "darwin":
		return "brew"
	}
	for _, m := range []struct{ bin, name string }{{"apt-get", "apt"}, {"dnf", "dnf"}, {"pacman", "pacman"}, {"brew", "brew"}} {
		if _, err := lookPath(m.bin); err == nil {
			return m.name
		}
	}
	return ""
}

// installCommands are the lines to paste to install the missing tools, deduplicated.
func installCommands(missing []string, goos string) []string {
	mgr := packageManager(goos)
	var names, out []string
	seen := map[string]bool{}
	rust := false
	for _, t := range missing {
		if t == "cargo" {
			rust = true
			continue
		}
		for _, n := range strings.Fields(pkgs[t][mgr]) {
			if !seen[n] {
				seen[n] = true
				names = append(names, n)
			}
		}
	}
	if len(names) > 0 {
		switch mgr {
		case "apt":
			out = append(out, "sudo apt-get update && sudo apt-get install -y "+strings.Join(names, " "))
		case "dnf":
			out = append(out, "sudo dnf install -y "+strings.Join(names, " "))
		case "pacman":
			out = append(out, "sudo pacman -S --needed "+strings.Join(names, " "))
		case "brew":
			if _, err := lookPath("brew"); err != nil {
				out = append(out, `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`)
			}
			out = append(out, "brew install "+strings.Join(names, " "))
		case "winget":
			for _, n := range names {
				out = append(out, "winget install -e --id "+n)
			}
		}
	}
	if rust {
		if goos == "windows" {
			out = append(out, "winget install -e --id Rustlang.Rustup")
		} else {
			out = append(out, rustupCmd)
		}
	}
	if len(out) == 0 && len(missing) > 0 {
		out = append(out, "instale pelo gerenciador de pacotes do sistema: "+strings.Join(missing, ", "))
	}
	return out
}

var ciTargetRe = regexp.MustCompile(`(?m)^ci\s*:`)

// hasCI reports whether the project already has the `make ci` the gates run.
func hasCI(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "Makefile"))
	return err == nil && ciTargetRe.Match(b)
}

// detectStack picks the stack from the project's files, then from the request when the
// project has no code yet. It returns "" when it cannot tell (the engine then asks).
func detectStack(dir, request string) string {
	has := func(names ...string) bool {
		for _, n := range names {
			if m, _ := filepath.Glob(filepath.Join(dir, n)); len(m) > 0 {
				return true
			}
		}
		return false
	}
	switch {
	case has("package.json"):
		return "node"
	case has("go.mod"):
		return "go"
	case has("pyproject.toml", "requirements.txt", "setup.py"):
		return "python"
	case has("pom.xml", "build.gradle", "build.gradle.kts"):
		return "java"
	case has("*.csproj", "*.sln", "*/*.csproj"):
		return "dotnet"
	case has("Cargo.toml"):
		return "rust"
	}
	if code := codeFiles(dir); len(code) > 0 {
		for _, f := range code {
			if ext := strings.ToLower(filepath.Ext(f)); ext != ".html" && ext != ".css" && ext != ".js" {
				return ""
			}
		}
		return "web"
	}
	// No code yet: the request decides; a page or a site is web.
	words := " " + strings.ToLower(request) + " "
	for _, k := range []struct{ word, stack string }{
		{" python", "python"}, {" django", "python"}, {" flask", "python"}, {" fastapi", "python"},
		{" node", "node"}, {" react", "node"}, {" next.js", "node"}, {" express", "node"}, {" typescript", "node"},
		{" golang", "go"}, {" em go ", "go"}, {" java ", "java"}, {" spring", "java"},
		{" c#", "dotnet"}, {" .net", "dotnet"}, {" dotnet", "dotnet"}, {" rust", "rust"},
	} {
		if strings.Contains(words, k.word) {
			return k.stack
		}
	}
	return "web"
}

// codeFiles lists the project's files that are not documentation or configuration.
func codeFiles(dir string) []string {
	var out []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() {
			if rel != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "specs" || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(rel)) {
		case ".md", ".txt", ".json", ".yml", ".yaml", ".toml", ".lock", "":
			return nil
		}
		if d.Name() == "LICENSE" {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	return out
}

// decidedStack reads the user's answer to stackQuestion from the spec's decisions.
func decidedStack(specText string) string {
	for _, l := range strings.Split(specText, "\n") {
		if !strings.Contains(l, "(decisão do usuário)") || !strings.Contains(strings.ToLower(l), "stack") {
			continue // the model may reword the question; any user decision about the stack counts
		}
		_, ans, ok := strings.Cut(l, "→")
		if !ok {
			continue
		}
		ans = " " + strings.ToLower(ans) + " "
		for _, s := range stacks {
			if strings.Contains(ans, " "+s+" ") || strings.Contains(ans, " "+s+",") || strings.Contains(ans, " "+s+".") {
				return s
			}
		}
	}
	return ""
}

// missingTools returns the tools the stack's `make ci` needs and the PATH lacks.
func missingTools(stack string) []string {
	var out []string
	for _, t := range append([]string{"git"}, stackTools[stack]...) {
		if _, err := lookPath(t); err != nil {
			out = append(out, t)
		}
	}
	return out
}

// missingMessage says what is missing and the exact commands to install it.
func missingMessage(stack string, miss []string) string {
	return fmt.Sprintf("o projeto %s precisa de %s para o `make ci` rodar. Para instalar, rode:\n\n  %s\n\ne depois rode axyn_run de novo com resume (nenhuma tentativa foi gasta).",
		stack, strings.Join(miss, ", "), strings.Join(installCommands(miss, runtime.GOOS), "\n  "))
}

// applyTemplate writes template/common, template/<stack> and template/seed into dir,
// never over an existing file, and returns the files it created.
func applyTemplate(dir, stack string) ([]string, error) {
	project, owner, repo := templateValues(dir)
	repl := strings.NewReplacer("{{PROJECT}}", project, "{{OWNER}}", owner, "{{REPO}}", repo)
	var created []string
	for _, part := range []string{"common", stack, "seed"} {
		root := path.Join("templates", part)
		err := fs.WalkDir(templates, root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel := strings.TrimPrefix(p, root+"/")
			dst := filepath.Join(dir, filepath.FromSlash(rel))
			if _, err := os.Stat(dst); err == nil {
				return nil
			}
			b, err := templates.ReadFile(p)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if strings.HasSuffix(rel, ".sh") {
				mode = 0o755
			}
			if err := os.WriteFile(dst, []byte(repl.Replace(string(b))), mode); err != nil {
				return err
			}
			created = append(created, rel)
			return nil
		})
		if err != nil {
			return created, err
		}
	}
	sort.Strings(created)
	return created, nil
}

// templateValues fills {{PROJECT}}, {{OWNER}} and {{REPO}} from the origin remote, like
// adopt.sh; with no remote yet, the directory and the local user stand in.
func templateValues(dir string) (project, owner, repo string) {
	abs, _ := filepath.Abs(dir)
	project, repo = filepath.Base(abs), filepath.Base(abs)
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	if b, err := cmd.Output(); err == nil {
		u := strings.TrimSuffix(strings.TrimSpace(string(b)), ".git")
		u = strings.ReplaceAll(u, ":", "/")
		if parts := strings.Split(u, "/"); len(parts) >= 2 {
			owner, repo = parts[len(parts)-2], parts[len(parts)-1]
		}
	}
	if owner == "" {
		owner = "owner"
		if u, err := user.Current(); err == nil && u.Username != "" {
			owner = filepath.Base(u.Username)
		}
	}
	return project, owner, repo
}

// prepare is FR-4 in the run loop: before the first ticket, a project without `make ci`
// gets the stack's template in its own commit. It returns a message when the run must
// stop (a question for the user, a missing tool, a Makefile without the ci target).
func (r *runner) prepare() string {
	dir := r.s.dir
	if r.s.ci != defaultCICmd || hasCI(dir) {
		return ""
	}
	if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
		return "o Makefile do projeto não tem o alvo ci, que os portões rodam (`make ci`); adicione um alvo ci com o lint e os testes do projeto e rode axyn_run de novo com resume"
	}
	stack := ""
	if pl, _, err := r.s.loadPlan(); err == nil {
		if b, err := os.ReadFile(filepath.Join(dir, pl.Spec)); err == nil {
			stack = decidedStack(string(b))
		}
	}
	if stack == "" {
		stack = detectStack(dir, r.st.Request)
	}
	if stack == "" {
		return "o axyn precisa de uma resposta sua; grave com axyn_decide e rode axyn_run de novo com resume:\n" +
			askMarker + "o projeto não tem CI e não deu para saber a stack) " + stackQuestion +
			" Responda com uma destas: " + strings.Join(stacks, ", ") + "."
	}
	if miss := missingTools(stack); len(miss) > 0 {
		return missingMessage(stack, miss)
	}
	r.set("preparando o projeto")
	created, err := applyTemplate(dir, stack)
	if err != nil {
		return "não consegui preparar o projeto: " + err.Error()
	}
	if len(created) > 0 {
		if out, err := r.s.git("add", "-A"); err != nil {
			return "git add falhou: " + out
		}
		if out, err := r.s.git("commit", "-m", "chore: prepare project (axyn, stack "+stack+")"); err != nil {
			return "git commit falhou: " + out
		}
	}
	_, _ = fmt.Fprintf(r.log, "projeto preparado com o template %s: %d arquivo(s)\n", stack, len(created))
	return ""
}

func runInitCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	stack := fs.String("stack", "", "stack do projeto: "+strings.Join(stacks, ", ")+" (padrão: detectada pelos arquivos)")
	dir := fs.String("dir", ".", "raiz do repositório")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	s := *stack
	if s == "" {
		s = detectStack(*dir, "")
	}
	known := false
	for _, k := range stacks {
		known = known || k == s
	}
	if !known {
		_, _ = fmt.Fprintf(stderr, "axyn init: não deu para saber a stack; use --stack com uma destas: %s\n", strings.Join(stacks, ", "))
		return exitUsage
	}
	created, err := applyTemplate(*dir, s)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn init: %v\n", err)
		return exitFail
	}
	_, _ = fmt.Fprintf(stdout, "axyn init: template %s, %d arquivo(s) criado(s) (nenhum existente foi alterado)\n", s, len(created))
	for _, f := range created {
		_, _ = fmt.Fprintf(stdout, "  %s\n", f)
	}
	if !hasCI(*dir) {
		_, _ = fmt.Fprintln(stdout, "aviso: o Makefile já existia sem o alvo ci; adicione um alvo ci para os portões do axyn")
	}
	if miss := missingTools(s); len(miss) > 0 {
		_, _ = fmt.Fprintf(stdout, "falta %s para o make ci. Para instalar, rode:\n\n  %s\n", strings.Join(miss, ", "), strings.Join(installCommands(miss, runtime.GOOS), "\n  "))
	}
	return exitOK
}
