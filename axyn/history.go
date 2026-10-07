package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// history is one file with everything a run did (spec 021 FR-7): the user sends it, and
// whoever reads it knows what happened without reproducing the run. It never carries a
// secret: keys and tokens are redacted from every part.

var secretRes = []*regexp.Regexp{
	regexp.MustCompile(`sk-[A-Za-z0-9_\-]{6,}`),
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{6,}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{10,}`),
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._\-]{8,}`),
	regexp.MustCompile(`AIza[0-9A-Za-z_\-]{20,}`),
	regexp.MustCompile(`(?i)\b([A-Z0-9_]*(KEY|TOKEN|SECRET|PASSWORD))\s*[=:]\s*\S+`),
	regexp.MustCompile(`://[^/\s:@]+:[^/\s@]+@`), // credentials in a URL
}

func redact(s string) string {
	for i, re := range secretRes {
		switch i {
		case 5:
			s = re.ReplaceAllString(s, "$1=[removido]")
		case 6:
			s = re.ReplaceAllString(s, "://[removido]@")
		default:
			s = re.ReplaceAllString(s, "[removido]")
		}
	}
	return s
}

func toolVersion(name string, args ...string) string {
	if _, err := lookPath(name); err != nil {
		return "não instalado"
	}
	b, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "não consegui ler a versão"
	}
	return firstLine(string(b))
}

// buildHistory renders the run as Markdown.
func buildHistory(dir string, st *runState) string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	s := &mcpServer{dir: dir}

	w("# Histórico do axyn: execução %s\n\n", st.ID)
	w("Gerado em %s por `axyn history`. Chaves e tokens foram removidos.\n\n", time.Now().Format(time.RFC3339))
	w("## Resumo\n\n")
	w("- **Pedido:** %s\n- **Status:** %s (fase: %s)\n", oneLine(st.Request), st.Status, st.Phase)
	w("- **Início / última atualização:** %s / %s\n", st.Started.Format(time.RFC3339), st.Updated.Format(time.RFC3339))
	if st.Ticket > 0 {
		w("- **Ticket atual:** %d de %d «%s», %d tentativa(s), modelo %s\n", st.Ticket, st.Total, st.Title, st.Attempts, st.Model)
	}
	if st.Gate != "" {
		w("- **Último portão:** %s\n", st.Gate)
	}
	if st.Message != "" {
		w("\n### Mensagem final\n\n```text\n%s\n```\n", st.Message)
	}
	if len(st.Delivered) > 0 {
		w("\n### Entregas\n\n")
		for _, d := range st.Delivered {
			w("- %s\n", d)
		}
	}
	w("\n### Opções\n\n- CI: `%s`; base do diff: `%s`; limite: %d linhas; modelo forçado: %q; escada: %q\n",
		st.Opts.CI, st.Opts.Base, st.Opts.MaxLines, st.Opts.Model, st.Opts.Config)

	w("\n## Plano\n\n")
	if pl, _, err := s.loadPlan(); err != nil {
		w("Sem plano gravado (%v).\n", err)
	} else {
		w("- **Spec:** `%s`; branch base: `%s`\n\n", pl.Spec, pl.Base)
		for _, t := range pl.Tickets {
			state := "aberto"
			if t.Done {
				state = "entregue"
			}
			w("### Ticket %d: %s (%s)\n\n", t.ID, t.Title, state)
			w("- ACs: %s; modelo: %s; estratégia: %s\n", strings.Join(t.ACs, ", "), t.Model, t.Strategy)
			if strings.TrimSpace(t.Body) != "" {
				w("- O que fazer: %s\n", oneLine(t.Body))
			}
			for _, a := range t.Earlier {
				w("- Antes (não conta mais; %s): %s\n", a.Step, strings.Join(a.Reason, "; "))
			}
			for i, a := range t.Attempts {
				verdict := "reprovado"
				if a.Green {
					verdict = "verde"
				}
				w("- Tentativa %d (%s, %s", i+1, a.Model, verdict)
				if a.Step != "" {
					w(", passo %s", a.Step)
				}
				w(")")
				if len(a.Reason) > 0 {
					w(": %s", strings.Join(a.Reason, "; "))
				}
				if len(a.Plan) > 0 {
					w(" (plano: %s)", strings.Join(a.Plan, ", "))
				}
				w("\n")
			}
			w("\n")
		}
		for _, t := range pl.Tickets {
			if d := s.wipDiff(pl.Base, t.WIP); d != "" {
				w("### Código da última tentativa do ticket %d (branch `%s`)\n\n```diff\n%s\n```\n\n", t.ID, t.WIP, strings.TrimRight(d, "\n"))
			}
		}
		if spec, err := os.ReadFile(filepath.Join(dir, pl.Spec)); err == nil {
			w("### Spec\n\n````markdown\n%s\n````\n", strings.TrimRight(string(spec), "\n"))
		}
	}

	w("\n## Commits e branches\n\n```text\n")
	since := st.Started.Add(-time.Minute).Format(time.RFC3339)
	if out, err := s.git("log", "--all", "--since="+since, "--format=%h %ad %s%d", "--date=iso"); err == nil && out != "" {
		w("%s\n", out)
	} else {
		w("(nenhum commit desde o início da execução)\n")
	}
	if out, err := s.git("status", "--short", "--branch"); err == nil {
		w("\n$ git status\n%s\n", out)
	}
	w("```\n")

	w("\n## Ambiente\n\n")
	w("- axyn %s; %s/%s\n- %s\n- opencode: %s\n", version, runtime.GOOS, runtime.GOARCH,
		toolVersion("git", "--version"), toolVersion("opencode", "--version"))
	d := &setupRun{dir: dir}
	d.run()
	w("\n### axyn doctor\n\n```text\n")
	for _, l := range d.lines {
		w("%-11s  %s: %s\n", l.status, l.item, l.detail)
	}
	w("```\n")

	w("\n## Log completo\n\n")
	if logb, err := os.ReadFile(filepath.Join(runsDir(dir), st.ID+".log")); err == nil {
		w("```text\n%s\n```\n", strings.TrimRight(string(logb), "\n"))
	} else {
		w("Sem log (%v).\n", err)
	}
	return redact(b.String())
}

// writeHistory writes the history of run id (the last one when empty) and returns its path.
func writeHistory(dir, id, out string) (string, error) {
	st, err := loadRun(dir, id)
	if err != nil {
		return "", fmt.Errorf("nenhuma execução para o histórico: %v", err)
	}
	if out == "" {
		// .axyn is out of git: a file in the project root would dirty the tree, and the
		// next axyn_run refuses a dirty tree.
		out = filepath.Join(dir, ".axyn", "axyn_history-"+st.ID+".md")
	}
	if abs, err := filepath.Abs(out); err == nil {
		out = abs
	}
	if err := os.WriteFile(out, []byte(buildHistory(dir, st)), 0o644); err != nil {
		return "", err
	}
	return out, nil
}

func runHistoryCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn history", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "raiz do repositório")
	out := fs.String("out", "", "arquivo de saída, em qualquer pasta (padrão: .axyn/axyn_history-ID.md no repositório)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	path, err := writeHistory(*dir, fs.Arg(0), *out)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn history: %v\n", err)
		return exitFail
	}
	_, _ = fmt.Fprintf(stdout, "histórico gravado em %s; anexe esse arquivo ao pedir ajuda (chaves e tokens foram removidos)\n", path)
	return exitOK
}
