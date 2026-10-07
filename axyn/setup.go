package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// setup is spec 021 FR-1 for the repository: everything the axyn needs on GitHub (a
// remote, Actions that may open PRs, auto-merge) checked by `axyn doctor` and configured
// by `axyn setup` through the gh CLI, so the user does as little as possible.

type checkLine struct {
	status string // ok, configurado, falta, aviso
	item   string
	detail string
}

type setupRun struct {
	dir     string
	branch  string // the repository's default branch, from the API
	fix     bool
	protect bool
	lines   []checkLine
}

func (r *setupRun) add(status, item, detail string) {
	r.lines = append(r.lines, checkLine{status, item, detail})
}

func (r *setupRun) git(args ...string) (string, error) {
	return (&mcpServer{dir: r.dir}).git(args...)
}

// gh runs the gh CLI in the project; stdin feeds `--input -`.
func (r *setupRun) gh(stdin string, args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Dir = r.dir
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	b, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(b)), err
}

func (r *setupRun) ghJSON(v any, args ...string) error {
	out, err := r.gh("", args...)
	if err != nil {
		return fmt.Errorf("%s", firstLine(out))
	}
	return json.Unmarshal([]byte(out), v)
}

// githubRepo is OWNER/REPO of the origin remote, or "" when it is not on GitHub.
func githubRepo(url string) string {
	u := strings.TrimSuffix(strings.TrimSpace(url), ".git")
	i := strings.Index(u, "github.com")
	if i < 0 {
		return ""
	}
	parts := strings.FieldsFunc(u[i+len("github.com"):], func(c rune) bool { return c == '/' || c == ':' })
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-2] + "/" + parts[len(parts)-1]
}

func (r *setupRun) run() {
	if _, err := lookPath("git"); err != nil {
		r.add("falta", "git", "instale e rode de novo: "+strings.Join(installCommands([]string{"git"}, runtime.GOOS), " && "))
		return
	}
	if _, err := r.git("rev-parse", "--git-dir"); err != nil {
		r.add("falta", "repositório git", "rode dentro do repositório do projeto, ou crie um: git init")
		return
	}
	ghOK := r.checkGH()
	r.checkIdentity(ghOK)
	repo := r.checkRemote(ghOK)
	if ghOK && repo != "" {
		r.checkActions(repo)
		r.checkRepoSettings(repo)
		r.checkProtection(repo)
	}
	r.checkTools()
}

func (r *setupRun) checkGH() bool {
	if _, err := lookPath("gh"); err != nil {
		r.add("falta", "gh (GitHub CLI)", "instale: "+strings.Join(installCommands([]string{"gh"}, runtime.GOOS), " && ")+"; depois: gh auth login")
		return false
	}
	if _, err := r.gh("", "auth", "status"); err != nil {
		r.add("falta", "login no GitHub", "rode: gh auth login (só você pode fazer: abre o navegador)")
		return false
	}
	r.add("ok", "gh (GitHub CLI)", "autenticado")
	return true
}

func (r *setupRun) checkIdentity(ghOK bool) {
	name, _ := r.git("config", "user.name")
	email, _ := r.git("config", "user.email")
	if name != "" && email != "" {
		r.add("ok", "identidade do git", fmt.Sprintf("%s <%s>", name, email))
		return
	}
	if r.fix && ghOK {
		var u struct {
			ID    int    `json:"id"`
			Login string `json:"login"`
			Name  string `json:"name"`
		}
		if err := r.ghJSON(&u, "api", "user"); err == nil && u.Login != "" {
			if name == "" {
				name = u.Name
				if name == "" {
					name = u.Login
				}
			}
			if email == "" {
				email = fmt.Sprintf("%d+%s@users.noreply.github.com", u.ID, u.Login)
			}
			_, e1 := r.git("config", "user.name", name)
			_, e2 := r.git("config", "user.email", email)
			if e1 == nil && e2 == nil {
				r.add("configurado", "identidade do git", fmt.Sprintf("%s <%s>, da sua conta do GitHub (só neste repositório)", name, email))
				return
			}
		}
	}
	r.add("falta", "identidade do git", `rode: git config --global user.name "Seu Nome" && git config --global user.email "voce@exemplo.com" (ou axyn setup, que usa a sua conta do GitHub)`)
}

func (r *setupRun) checkRemote(ghOK bool) string {
	url, err := r.git("remote", "get-url", "origin")
	if err == nil {
		if repo := githubRepo(url); repo != "" {
			r.add("ok", "repositório no GitHub", repo)
			return repo
		}
		r.add("aviso", "repositório no GitHub", "o origin não é do GitHub ("+url+"): push e PR do axyn não são configurados aqui")
		return ""
	}
	abs, _ := filepath.Abs(r.dir)
	name := filepath.Base(abs)
	create := "gh repo create " + name + " --private --source . --remote origin"
	if _, err := r.git("rev-parse", "--verify", "--quiet", "HEAD"); err == nil {
		create += " --push"
	}
	if !r.fix || !ghOK {
		r.add("falta", "repositório no GitHub", "sem o remoto origin; rode: "+create+" (ou axyn setup)")
		return ""
	}
	if out, err := r.gh("", strings.Fields(create)[1:]...); err != nil {
		r.add("falta", "repositório no GitHub", "o gh repo create falhou: "+firstLine(out))
		return ""
	}
	url, _ = r.git("remote", "get-url", "origin")
	repo := githubRepo(url)
	r.add("configurado", "repositório no GitHub", repo+" criado (privado) e ligado como origin")
	return repo
}

func (r *setupRun) checkActions(repo string) {
	var a struct {
		Enabled bool `json:"enabled"`
	}
	var w struct {
		Default    string `json:"default_workflow_permissions"`
		CanApprove bool   `json:"can_approve_pull_request_reviews"`
	}
	errA := r.ghJSON(&a, "api", "repos/"+repo+"/actions/permissions")
	errW := r.ghJSON(&w, "api", "repos/"+repo+"/actions/permissions/workflow")
	if errA != nil || errW != nil {
		r.add("aviso", "GitHub Actions", "não consegui ler as permissões (precisa ser admin do repositório)")
		return
	}
	if a.Enabled && w.Default == "write" && w.CanApprove {
		r.add("ok", "GitHub Actions", "habilitado, com escrita e permissão para abrir PRs")
		return
	}
	if !r.fix {
		r.add("falta", "GitHub Actions", "precisa estar habilitado, com escrita e permissão para abrir PRs (o CI e o motor do sdd-kit usam); rode: axyn setup")
		return
	}
	if !a.Enabled {
		if out, err := r.gh("", "api", "-X", "PUT", "repos/"+repo+"/actions/permissions", "-F", "enabled=true", "-f", "allowed_actions=all"); err != nil {
			r.add("falta", "GitHub Actions", "não consegui habilitar: "+firstLine(out))
			return
		}
	}
	if out, err := r.gh("", "api", "-X", "PUT", "repos/"+repo+"/actions/permissions/workflow",
		"-f", "default_workflow_permissions=write", "-F", "can_approve_pull_request_reviews=true"); err != nil {
		r.add("falta", "GitHub Actions", "não consegui dar escrita ao GITHUB_TOKEN: "+firstLine(out))
		return
	}
	r.add("configurado", "GitHub Actions", "habilitado, com escrita e permissão para abrir PRs")
}

func (r *setupRun) checkRepoSettings(repo string) {
	var s struct {
		AutoMerge    bool   `json:"allow_auto_merge"`
		DeleteBranch bool   `json:"delete_branch_on_merge"`
		Default      string `json:"default_branch"`
	}
	if err := r.ghJSON(&s, "api", "repos/"+repo); err != nil {
		r.add("aviso", "merge automático", "não consegui ler o repositório: "+err.Error())
		return
	}
	r.branch = s.Default
	if s.AutoMerge && s.DeleteBranch {
		r.add("ok", "merge automático", "permitido, e a branch é apagada depois do merge")
		return
	}
	if !r.fix {
		r.add("falta", "merge automático", "permitir o merge automático e apagar a branch depois do merge; rode: axyn setup")
		return
	}
	if out, err := r.gh("", "api", "-X", "PATCH", "repos/"+repo, "-F", "allow_auto_merge=true", "-F", "delete_branch_on_merge=true"); err != nil {
		r.add("falta", "merge automático", "não consegui configurar: "+firstLine(out))
		return
	}
	r.add("configurado", "merge automático", "permitido, e a branch é apagada depois do merge")
}

const protectionBody = `{"required_status_checks":{"strict":false,"contexts":["make ci"]},"enforce_admins":false,"required_pull_request_reviews":null,"restrictions":null}`

func (r *setupRun) checkProtection(repo string) {
	branch := r.branch
	if branch == "" {
		branch = "main"
	}
	path := "repos/" + repo + "/branches/" + branch + "/protection"
	if _, err := r.gh("", "api", path); err == nil {
		r.add("ok", "proteção da "+branch, "configurada")
		return
	}
	if !r.fix || !r.protect {
		r.add("aviso", "proteção da "+branch, "opcional: exigir o check \"make ci\" antes do merge; rode: axyn setup --protect-main")
		return
	}
	if out, err := r.gh(protectionBody, "api", "-X", "PUT", path, "--input", "-"); err != nil {
		if strings.Contains(out, "403") || strings.Contains(strings.ToLower(out), "upgrade") {
			r.add("aviso", "proteção da "+branch, "o plano do GitHub não permite proteção em repositório privado; seguindo sem ela")
			return
		}
		r.add("falta", "proteção da "+branch, "não consegui configurar: "+firstLine(out))
		return
	}
	r.add("configurado", "proteção da "+branch, "o check \"make ci\" é exigido antes do merge")
}

func (r *setupRun) checkTools() {
	if _, err := lookPath("opencode"); err != nil {
		r.add("falta", "opencode", "instale: curl -fsSL https://opencode.ai/install | bash (ou npm i -g opencode-ai)")
	} else {
		r.add("ok", "opencode", "instalado")
	}
	r.checkModel()
	stack := detectStack(r.dir, "")
	if stack == "" {
		return
	}
	if miss := missingTools(stack); len(miss) > 0 {
		r.add("falta", "ferramentas do make ci ("+stack+")", "falta "+strings.Join(miss, ", ")+"; rode: "+strings.Join(installCommands(miss, runtime.GOOS), " && "))
		return
	}
	r.add("ok", "ferramentas do make ci ("+stack+")", "instaladas")
}

func runSetupCmd(args []string, stdout, stderr io.Writer, fix bool) int {
	name := "axyn doctor"
	if fix {
		name = "axyn setup"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "raiz do repositório")
	protect := fs.Bool("protect-main", false, "(setup) exige o check make ci antes do merge na branch principal")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if _, err := os.Stat(*dir); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: %v\n", name, err)
		return exitUsage
	}
	r := &setupRun{dir: *dir, fix: fix, protect: *protect}
	r.run()
	missing := 0
	for _, l := range r.lines {
		mark := map[string]string{"ok": "ok         ", "configurado": "configurado", "falta": "falta      ", "aviso": "aviso      "}[l.status]
		_, _ = fmt.Fprintf(stdout, "%s  %s: %s\n", mark, l.item, l.detail)
		if l.status == "falta" {
			missing++
		}
	}
	if n := updateNotice(); n != "" {
		_, _ = fmt.Fprintf(stdout, "\n%s\n", n)
	}
	if missing > 0 {
		if !fix {
			_, _ = fmt.Fprintf(stdout, "\n%d item(ns) faltando; o axyn setup configura o que dá, e os outros mostram o comando a rodar.\n", missing)
		}
		return exitFail
	}
	_, _ = fmt.Fprintln(stdout, "\ntudo certo: o axyn pode abrir um PR por ticket neste repositório.")
	return exitOK
}

// checkModel shows the model the background agents get (spec 021 FR-6).
func (r *setupRun) checkModel() {
	ms, err := loadModels(defaultConfigPath())
	switch {
	case err != nil:
		r.add("falta", "modelo do axyn", err.Error()+"; para regravar: axyn model")
	case len(ms) == 0:
		r.add("aviso", "modelo do axyn", "nenhum escolhido: os agentes usam o modelo padrão do opencode; para escolher: axyn model")
	case ms[0].KeyEnv != "" && os.Getenv(ms[0].KeyEnv) == "":
		r.add("falta", "modelo do axyn", ms[0].ID+": a variável "+ms[0].KeyEnv+" não está definida; "+setEnvHint(ms[0].KeyEnv))
	default:
		extra := ""
		if len(ms) > 1 {
			extra = fmt.Sprintf(" (e mais %d na escada)", len(ms)-1)
		}
		r.add("ok", "modelo do axyn", ms[0].ID+extra+"; para trocar: axyn model")
	}
}
