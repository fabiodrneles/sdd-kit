package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// axyn release closes a version of the user's project (#340): it runs the project's
// scripts/sdd-release.sh (from the sdd-kit template the axyn prepared), where the version
// comes from the go-release-manager (Conventional Commits). First a dry run shows the
// version and what goes in it; the PR with the CHANGELOG is opened only after a yes.

const releaseScript = "scripts/sdd-release.sh"

var releaseVersionRe = regexp.MustCompile(`v(\d+\.\d+\.\d+)`)

// releaseMissing says what the release needs and is missing, with how to fix it.
func releaseMissing(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(releaseScript))); err != nil {
		return "este projeto não tem o " + releaseScript + "; para trazê-lo, rode na raiz do projeto: axyn init (copia o que falta do template, sem mexer no que existe), faça commit e tente de novo"
	}
	var miss []string
	if _, err := lookPath("gh"); err != nil {
		miss = append(miss, "gh")
	}
	_, goErr := lookPath("go")
	_, grmErr := lookPath("go-release-manager")
	if goErr != nil && grmErr != nil {
		miss = append(miss, "go")
	}
	if len(miss) > 0 {
		return "a release precisa de " + strings.Join(miss, ", ") + " (o go-release-manager calcula a versão e roda com o Go). Para instalar: " +
			strings.Join(installCommands(miss, hostOS), " && ")
	}
	return ""
}

func runRelease(dir string, args ...string) (string, error) {
	cmd, err := shellCommand("sh " + releaseScript + " " + strings.Join(args, " "))
	if err != nil {
		return "", err
	}
	cmd.Dir = dir
	b, err := cmd.CombinedOutput()
	return fixMojibake(strings.TrimSpace(string(b))), err
}

// releasePreview is the dry run told in plain words: version, what goes in it, next step.
func releasePreview(dir string) (string, string, error) {
	out, err := runRelease(dir, "--dry-run")
	if err != nil {
		return "", "", fmt.Errorf("não consegui calcular a release: %s", lastLines(out, 6))
	}
	ver := ""
	var items []string
	for _, l := range strings.Split(out, "\n") {
		if ver == "" && strings.Contains(l, "calculada") {
			if m := releaseVersionRe.FindStringSubmatch(l); m != nil {
				ver = "v" + m[1]
			}
		}
		if t := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "|")); strings.HasPrefix(t, "- ") {
			items = append(items, t)
		}
	}
	if ver == "" {
		return "", "", fmt.Errorf("o go-release-manager não calculou uma versão (sem commits feat ou fix desde a última tag, não há o que lançar): %s", lastLines(out, 4))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "A próxima versão é %s, calculada pelo go-release-manager a partir dos commits (feat sobe a versão do meio, fix a do fim, ! a primeira).\n", ver)
	if len(items) > 0 {
		b.WriteString("O que entra nela (rascunho do CHANGELOG, pelos PRs mesclados):\n")
		for _, it := range items {
			fmt.Fprintf(&b, "  %s\n", it)
		}
	}
	b.WriteString("Ao confirmar, o axyn abre um PR de fechamento (branch chore/release-" + ver + ") com o CHANGELOG e as versões dos arquivos; depois do merge desse PR, a tag e a página da release saem sozinhas pelo workflow do projeto.")
	return ver, b.String(), nil
}

func lastLines(s string, n int) string {
	ls := strings.Split(strings.TrimSpace(s), "\n")
	if len(ls) > n {
		ls = ls[len(ls)-n:]
	}
	return strings.Join(ls, " | ")
}

func openRelease(dir string) (string, error) {
	out, err := runRelease(dir)
	if err != nil {
		return "", fmt.Errorf("o PR de fechamento não foi aberto: %s", lastLines(out, 6))
	}
	pr := ""
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "/pull/") {
			pr = strings.TrimSpace(l[strings.LastIndex(l, " ")+1:])
		}
	}
	msg := "PR de fechamento aberto"
	if pr != "" {
		msg += ": " + pr
	}
	return msg + ". Revise o CHANGELOG no PR e faça o merge; a tag e a release saem sozinhas. Se o projeto não tiver o workflow de release, depois do merge rode na raiz do projeto: axyn release --tag VERSÃO", nil
}

func runReleaseCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn release", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "raiz do repositório")
	yes := fs.Bool("yes", false, "abre o PR de fechamento sem perguntar")
	tag := fs.String("tag", "", "com o PR de fechamento mesclado: cria a tag X.Y.Z e publica a release")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if msg := releaseMissing(*dir); msg != "" {
		_, _ = fmt.Fprintln(stderr, "axyn release: "+msg)
		return exitFail
	}
	if *tag != "" {
		out, err := runRelease(*dir, "--tag", strings.TrimPrefix(*tag, "v"))
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "axyn release: a tag não foi criada: %s\n", lastLines(out, 6))
			return exitFail
		}
		_, _ = fmt.Fprintf(stdout, "%s\nrelease v%s disparada; a página da release aparece em alguns minutos\n", out, strings.TrimPrefix(*tag, "v"))
		return exitOK
	}
	ver, preview, err := releasePreview(*dir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn release: %v\n", err)
		return exitFail
	}
	_, _ = fmt.Fprintln(stdout, preview)
	if !*yes {
		_, _ = fmt.Fprintf(stdout, "\nAbrir o PR de fechamento da %s? (s/N): ", ver)
		line, _ := bufio.NewReader(stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(line)); a != "s" && a != "sim" && a != "y" && a != "yes" {
			_, _ = fmt.Fprintln(stdout, "nada foi feito; quando quiser: axyn release")
			return exitOK
		}
	}
	msg, err := openRelease(*dir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn release: %v\n", err)
		return exitFail
	}
	_, _ = fmt.Fprintln(stdout, msg)
	return exitOK
}

// toolRelease is the MCP side: without confirm it previews; with confirm it opens the PR.
func (s *mcpServer) toolRelease(raw json.RawMessage) (string, error) {
	var a struct {
		Confirm bool `json:"confirm"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", err
		}
	}
	if msg := releaseMissing(s.dir); msg != "" {
		return "", fmt.Errorf("%s", msg)
	}
	if !a.Confirm {
		_, preview, err := releasePreview(s.dir)
		if err != nil {
			return "", err
		}
		return preview + "\nPergunte ao usuário se quer abrir o PR de fechamento; com o sim, chame axyn_release com confirm.", nil
	}
	return openRelease(s.dir)
}
