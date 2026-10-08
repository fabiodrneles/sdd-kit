package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// reportTerm is the bench report for the console (#359): the same facts as the Markdown
// file, laid out to be read at a glance. Each score is colored by its level (green strong,
// yellow good, red contained), the routing names the best models first, and only the
// attempts that went wrong are listed, one short line each. Off a console it is plain.
func reportTerm(p *benchProfile, path string, color bool) string {
	c := func(code, s string) string {
		if !color || code == "" {
			return s
		}
		return code + s + cReset
	}
	const (
		green = "\x1b[1;92m"
		red   = "\x1b[1;91m"
		dimRd = "\x1b[91m"
	)
	short := func(m string) string { return strings.TrimPrefix(m, "opencode/") }
	pad := func(s string, n int) string {
		if k := utf8.RuneCountInString(s); k < n {
			return s + strings.Repeat(" ", n-k)
		}
		return s
	}
	lpad := func(s string, n int) string {
		if k := utf8.RuneCountInString(s); k < n {
			return strings.Repeat(" ", n-k) + s
		}
		return s
	}
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }

	w("\n%s  %s\n", c(cYellow, "▌ axyn · avaliação dos modelos"), c(cGray, fmt.Sprintf("%s · axyn %s · opencode %s", p.Date.Format("02/01/2006 15:04"), p.Axyn, p.Opencode)))
	w("%s\n\n", c(cGray, "  notas de 0 a 100, só com verificações automáticas: nenhum modelo julga outro"))

	width := len("modelo")
	for _, m := range p.Models {
		if n := utf8.RuneCountInString(short(m)); n > width {
			width = n
		}
	}
	const col = 10
	w("  %s", c(cWhite, pad("modelo", width)))
	for _, t := range benchTasks {
		w("%s", c(cWhite, lpad(roleNames[t.Role], col)))
	}
	w("\n  %s\n", c(cGray, strings.Repeat("─", width+col*len(benchTasks))))
	anyOld := false
	for _, m := range p.Models {
		w("  %s", pad(short(m), width))
		for _, t := range benchTasks {
			r := findResult(p.Results, m, t.ID)
			switch {
			case r == nil:
				w("%s ", c(cGray, lpad("·", col-1)))
				continue
			case r.cheated():
				w("%s ", c(red, lpad("veto", col-1)))
				continue
			case r.allDown():
				w("%s ", c(cYellow, lpad("fora", col-1)))
				continue
			}
			mark := " "
			if r.old(p.Date) {
				mark, anyOld = "*", true
			}
			code := red
			switch level(*r) {
			case "forte":
				code = green
			case "bom":
				code = cYellow
			}
			w("%s%s", c(code, lpad(fmt.Sprint(r.score()), col-1)), c(cGray, mark))
		}
		w("\n")
	}
	w("\n  %s %s  %s %s  %s %s  %s %s", c(green, "■"), c(cGray, "forte"), c(cYellow, "■"), c(cGray, "bom"), c(red, "■"), c(cGray, "contido (só no modo guiado)"), c(cGray, "·"), c(cGray, "não avaliado  fora = indisponível (não conta)  veto = trapaça"))
	if anyOld {
		w("  %s", c(cGray, "* de uma rodada anterior"))
	}
	w("\n\n%s\n", c(cWhite, "  Quem faz cada etapa"))
	for _, role := range []string{roleplan, roleCode, roleTests, roleFix} {
		label := pad(roleNames[role], 9)
		if pin := p.Pinned[role]; pin != "" {
			w("  %s %s %s %s\n", c(cYellow, glyph("▸", "►")), c(cWhite, label), short(pin), c(cGray, "(fixado por você)"))
			continue
		}
		var top, held []string
		for _, m := range p.Routing[role] {
			if inList(p.Contained[role], m) {
				held = append(held, short(m))
			} else {
				top = append(top, short(m))
			}
		}
		line := c(dimRd, "nenhum modelo passou: todos no modo guiado")
		if len(top) > 0 {
			line = c(green, top[0])
			if len(top) > 1 {
				rest := top[1:]
				if len(rest) > 3 {
					rest = append(rest[:3:3], fmt.Sprintf("+%d", len(top)-4))
				}
				line += c(cGray, " → "+strings.Join(rest, ", "))
			}
		}
		if len(held) > 0 {
			line += c(cGray, fmt.Sprintf("  (no modo guiado: %s)", strings.Join(held, ", ")))
		}
		w("  %s %s %s\n", c(cYellow, glyph("▸", "►")), c(cWhite, label), line)
	}

	var bad []string
	for _, r := range p.Results {
		for _, x := range r.Runs {
			if x.Down || (x.Pass && !x.Cheat && x.Score >= 50) {
				continue
			}
			why := strings.Join(x.Notes, "; ")
			if why == "" {
				why = "reprovado"
			}
			when := ""
			if r.old(p.Date) {
				when = c(cGray, " *")
			}
			bad = append(bad, fmt.Sprintf("  %s %s %s%s %s", c(dimRd, glyph("✘", "×")), pad(short(r.Model), width), c(cWhite, pad(taskNames[r.Task], 9)), when, c(cGray, shorten(oneLine(why), 110))))
		}
	}
	if len(bad) > 0 {
		w("\n%s\n%s\n", c(cWhite, "  O que deu errado"), strings.Join(bad, "\n"))
	}
	if path != "" {
		w("\n  %s %s\n", c(cGray, "relatório completo, com o código de cada tentativa:"), path)
	}
	return b.String()
}
