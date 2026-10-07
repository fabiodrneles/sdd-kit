package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// A ticket the model could not get through the gates is never a dead end (spec 021 FR-8,
// #324): its last attempt is kept on a WIP branch, published as a draft PR (or written to
// .axyn/ajuda-<ticket>.md without a remote) with the code, every rejection and a text to
// paste into another AI; a fix made on that branch is gated first on the next resume.

const maxHelpDiff = 20000

// wipDiff is the code of the ticket's last attempt, against the plan's base.
func (s *mcpServer) wipDiff(base, wip string) string {
	if base == "" || wip == "" {
		return ""
	}
	d, err := s.git("diff", "--no-color", "--no-ext-diff", base+"..."+wip)
	if err != nil {
		return ""
	}
	if len(d) > maxHelpDiff {
		d = d[:maxHelpDiff] + "\n… (diff cortado; o código completo está na branch " + wip + ")"
	}
	return d
}

func helpText(t *ticket, base, wip, diff string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "O axyn não conseguiu fazer o ticket «%s» passar nos portões (`make ci`, testes, lint), depois de %d tentativa(s). O código da última tentativa está na branch `%s`.\n\n", t.Title, len(t.Attempts), wip)
	b.WriteString("## O ticket\n\n")
	fmt.Fprintf(&b, "%s\n\nCritérios de aceite: %s\n\n", strings.TrimSpace(t.Body), strings.Join(t.ACs, ", "))
	b.WriteString("## Por que cada tentativa foi reprovada\n\n")
	for i, a := range t.Attempts {
		if a.Green {
			continue
		}
		step := ""
		if a.Step != "" {
			step = ", " + a.Step
		}
		fmt.Fprintf(&b, "- Tentativa %d (%s%s): %s\n", i+1, a.Model, step, strings.Join(a.Reason, "; "))
	}
	b.WriteString("\n## Como seguir\n\n")
	fmt.Fprintf(&b, "1. Corrija o código na branch `%s` (você mesmo, ou com outra IA usando o texto abaixo) e faça commit nela.\n", wip)
	b.WriteString("2. Na raiz do projeto, rode `axyn run --resume`: o axyn roda os portões no código corrigido antes de tudo e, se passar, entrega o ticket.\n")
	b.WriteString("3. Ou dê mais chances com outro modelo: `axyn model` (o novo modelo ganha outras 10 tentativas) e `axyn run --resume`.\n\n")
	b.WriteString("## Para pedir ajuda a outra IA (copie tudo abaixo)\n\n````text\n")
	fmt.Fprintf(&b, "Corrija o código abaixo para ele passar no `make ci` do projeto (lint e testes), sem afrouxar, pular ou apagar testes.\n\nTicket: %s\n%s\nCritérios de aceite: %s\n\nErros:\n", t.Title, strings.TrimSpace(t.Body), strings.Join(t.ACs, ", "))
	if n := len(t.Attempts); n > 0 {
		fmt.Fprintf(&b, "%s\n", strings.Join(t.Attempts[n-1].Reason, "\n"))
	}
	fmt.Fprintf(&b, "\nCódigo (diff):\n%s\n````\n", diff)
	return b.String()
}

// publishHelp runs after saveWIP: it records the WIP branch in the plan, publishes the help
// (draft PR or file) and puts the user back on the base branch. It returns what to tell.
func (r *runner) publishHelp(ticketID int) string {
	s := r.s
	wip, _ := s.git("rev-parse", "--abbrev-ref", "HEAD")
	pl, path, err := s.loadPlan()
	if err != nil {
		return ""
	}
	var t *ticket
	for i := range pl.Tickets {
		if pl.Tickets[i].ID == ticketID {
			t = &pl.Tickets[i]
		}
	}
	if t == nil || wip == "" || wip == "HEAD" || wip == pl.Base {
		return ""
	}
	t.WIP = wip
	_ = s.savePlan(pl, path)
	return r.publish(pl, t, wip)
}

// recordWIP notes the WIP branch HEAD is on as the ticket's, without publishing it (an
// interrupted attempt is not a failed ticket).
func (r *runner) recordWIP(ticketID int) {
	wip, _ := r.s.git("rev-parse", "--abbrev-ref", "HEAD")
	pl, path, err := r.s.loadPlan()
	if err != nil || wip == "" || wip == "HEAD" || wip == pl.Base {
		return
	}
	for i := range pl.Tickets {
		if pl.Tickets[i].ID == ticketID {
			pl.Tickets[i].WIP = wip
		}
	}
	_ = r.s.savePlan(pl, path)
}

func (r *runner) publish(pl *plan, t *ticket, wip string) string {
	s := r.s
	text := helpText(t, pl.Base, wip, s.wipDiff(pl.Base, wip))

	note := ""
	remotes, _ := s.git("remote")
	_, ghErr := lookPath("gh")
	if strings.Contains(remotes, "origin") && ghErr == nil {
		if out, err := s.git("push", "-u", "origin", wip); err != nil {
			note = "não consegui publicar a branch " + wip + ": " + firstLine(out)
		} else {
			body := filepath.Join(os.TempDir(), "axyn-ajuda-"+r.st.ID+".md")
			_ = os.WriteFile(body, []byte(text), 0o644)
			args := []string{"pr", "create", "--draft", "--title", "[não passou nos portões] " + t.Title, "--body-file", body, "--head", wip}
			if pl.Base != "" {
				args = append(args, "--base", pl.Base)
			}
			cmd := exec.Command("gh", args...)
			cmd.Dir = s.dir
			b, err := cmd.CombinedOutput()
			_ = os.Remove(body)
			if err == nil {
				note = "o código da última tentativa está num PR em rascunho, com os erros e um texto pronto para pedir ajuda a outra IA: " + strings.TrimSpace(string(b))
			} else {
				note = "PR em rascunho não aberto (" + firstLine(string(b)) + ")"
			}
		}
	}
	if !strings.HasPrefix(note, "o código da última tentativa está num PR") {
		file := filepath.Join(s.dir, ".axyn", fmt.Sprintf("ajuda-ticket-%d.md", t.ID))
		if err := os.WriteFile(file, []byte(text), 0o644); err == nil {
			note = strings.TrimSpace(note + "; o código, os erros e um texto pronto para pedir ajuda a outra IA estão em " + file)
			note = strings.TrimPrefix(note, "; ")
		}
	}
	if pl.Base != "" {
		_, _ = s.git("checkout", "-q", pl.Base) // the user ends where the run started
	}
	return note
}

// gateWIP is the first step of a resumed ticket with a WIP branch: that code (maybe fixed
// by the user or another AI) goes through the gates before any agent runs.
func (r *runner) gateWIP(t *ticket) (green bool, report string, ok bool) {
	if !r.st.Opts.Resume || t.WIP == "" {
		return false, "", false
	}
	if _, err := r.s.git("rev-parse", "--verify", "--quiet", "refs/heads/"+t.WIP); err != nil {
		return false, "", false
	}
	if _, err := r.s.git("checkout", t.WIP, "--", "."); err != nil {
		return false, "", false
	}
	_, _ = r.s.git("reset", "-q")
	_, _ = fmt.Fprintf(r.log, "retomando com o código da branch %s: os portões rodam nele primeiro\n", t.WIP)
	r.set("portões")
	green, report, err := r.gate()
	if err != nil {
		return false, err.Error(), true
	}
	if !green {
		// Checking the saved code calls no model: it is history, not an attempt of the
		// ladder (it once showed up as attempt 11, "primeira tentativa").
		if pl, path, err := r.s.loadPlan(); err == nil {
			for i := range pl.Tickets {
				tk := &pl.Tickets[i]
				if tk.ID == t.ID && len(tk.Attempts) > 0 {
					a := tk.Attempts[len(tk.Attempts)-1]
					a.Step = stepRecheck
					tk.Earlier = append(tk.Earlier, a)
					tk.Attempts = tk.Attempts[:len(tk.Attempts)-1]
				}
			}
			_ = r.s.savePlan(pl, path)
		}
	}
	return green, report, true
}
