package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// explainStop is what the user reads when a ticket stops (#330): what was asked, what the
// axyn tried in each attempt, the real error of the gates, what it means and the concrete
// ways forward, each with the command to run and where.

var stepPlain = map[string]string{
	"":                  "primeira tentativa, só com o ticket",
	"diagnóstico exato": "o axyn devolveu ao modelo o erro exato da tentativa anterior",
	"mais contexto":     "o axyn deu ao modelo mais contexto (os arquivos e testes ligados ao erro)",
	"várias tentativas": "o axyn pediu uma versão diferente do código",
	"ticket dividido":   "o axyn dividiu o ticket em passos menores (primeiro o teste, depois o código)",
	stepPlan:            "o axyn pediu ao modelo um plano dos arquivos antes do código",
	stepAsk:             "o axyn parou para perguntar a você",
	stepRecheck:         "o axyn conferiu de novo o código salvo, sem chamar o modelo",
}

var errLine = regexp.MustCompile(`(?i)(\.[a-z]{1,4}:\d+|error|erro|fail|panic|undefined|not found|cannot|denied|abaixo do mínimo|cobertura:)`)

// ciErrors are the lines of the CI output that name an error, at most max, else its tail.
func ciErrors(out string, max int) []string {
	out = fixMojibake(out)
	var hits, all []string
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "make: ***") || strings.HasPrefix(l, "make[") {
			continue
		}
		all = append(all, l)
		if errLine.MatchString(l) && len(hits) < max {
			hits = append(hits, l)
		}
	}
	if len(hits) > 0 {
		return hits
	}
	if len(all) > max {
		all = all[len(all)-max:]
	}
	return all
}

// decisions are the user's decisions recorded in the spec.
func decisions(spec string) string {
	var b strings.Builder
	for _, l := range strings.Split(spec, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "- **D-") {
			b.WriteString(l + "\n")
		}
	}
	return b.String()
}

// meaning says in plain words what the error is, and an instruction that would fix it.
func meaning(reason string) (what, hint string) {
	r := strings.ToLower(reason)
	switch {
	case strings.Contains(r, "[escopo]") || strings.Contains(r, "só acrescenta testes"):
		return "o ticket era só de testes, mas o modelo mudou código de produção (fez trabalho de outro ticket).",
			"desfaça a mudança no código de produção e escreva só testes; as funcionalidades ficam para os outros tickets"
	case strings.Contains(r, "sem nenhum teste novo"):
		return "o ticket mudou código sem escrever nenhum teste; todo código novo precisa de teste.",
			"escreva um teste para cada critério de aceite do ticket, citando o AC no nome ou num comentário"
	case strings.Contains(r, "[protected]") || strings.Contains(r, "[tests]") || strings.Contains(r, "[plan]"):
		return "o modelo mexeu no que não podia (testes, specs, workflows ou o plano do axyn).",
			"não mexa em testes existentes, specs nem workflows"
	case strings.Contains(r, "[coverage]"):
		return "o código novo do ticket não está coberto por testes: as linhas citadas acima não rodam em nenhum teste.",
			"escreva testes que passem pelas linhas sem teste citadas acima"
	case strings.Contains(r, "cobertura abaixo do mínimo") || strings.Contains(r, "coverage"):
		return "a cobertura de testes do projeto caiu abaixo do mínimo: o código novo entrou com poucos testes.",
			"escreva testes para o código deste ticket e para as funções que ele toca"
	case strings.Contains(r, "errcheck"):
		return "o lint (golangci-lint) exige que todo erro devolvido por uma função seja tratado, e o código ignorou algum.",
			"trate todo erro que o lint apontou: if err != nil { return err }, ou _ = f() quando o erro pode ser ignorado"
	case strings.Contains(r, "undefined") || strings.Contains(r, "cannot use") || strings.Contains(r, "declared and not used"):
		return "o código não compila: usa algo que não existe ou com o tipo errado.",
			"faça o código compilar (go build ./...) sem mudar os testes"
	case strings.Contains(r, "--- fail") || strings.Contains(r, "fail\t") || strings.Contains(r, "test"):
		return "um teste falhou: o código não faz o que o critério de aceite pede.",
			"faça o teste que falhou passar, sem mudar o teste"
	case strings.Contains(r, "gofmt") || strings.Contains(r, "goimports") || strings.Contains(r, "not properly formatted"):
		return "o código não está formatado como o projeto exige.",
			"formate o código com gofmt -w ."
	case strings.Contains(r, "[size]"):
		return "a mudança ficou grande demais para um ticket.",
			"faça só o mínimo que o ticket pede"
	case strings.Contains(r, "[ci]"):
		return "o make ci do projeto (lint e testes) falhou com o código do modelo.",
			"corrija os erros do make ci acima sem afrouxar os testes"
	}
	return "o código não passou nas verificações do axyn.", "corrija os erros acima"
}

func (r *runner) explainStop(ticketID int, help string) string {
	pl, _, err := r.s.loadPlan()
	if err != nil {
		return "o axyn parou: " + err.Error()
	}
	var t *ticket
	for i := range pl.Tickets {
		if pl.Tickets[i].ID == ticketID {
			t = &pl.Tickets[i]
		}
	}
	if t == nil {
		return "o axyn parou: o ticket sumiu do plano"
	}
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("O axyn parou no ticket %d de %d, «%s», e precisa de você.\n\n", t.ID, len(pl.Tickets), t.Title)
	w("O que aconteceu: o modelo escreveu o código %d vez(es) e nenhuma passou nas verificações do axyn (o make ci do projeto, com lint e testes, mais as regras do axyn). O axyn não entrega código que não passa, então parou para você decidir.\n\n", len(t.Attempts))
	w("O que foi tentado:\n")
	if n := len(t.Earlier); n > 0 {
		w("  (antes destas, %d tentativa(s) ou conferência(s) que não contam mais, de antes de uma decisão sua ou da escolha da cobertura; estão no axyn history)\n", n)
	}
	last := ""
	for i, a := range t.Attempts {
		why := strings.Join(a.Reason, "; ")
		if why == "" {
			why = "reprovado"
		}
		plain, ok := stepPlain[a.Step]
		if !ok {
			plain = a.Step
		}
		w("  %d. %s (modelo %s): %s\n", i+1, plain, a.Model, shorten(why, 220))
		if !a.Green {
			last = why
		}
	}
	what, hint := meaning(last)
	w("\nO erro da última tentativa:\n  %s\n", strings.ReplaceAll(last, " | ", "\n  "))
	w("O que isso quer dizer: %s\n\n", what)
	w("Como resolver (escolha um; todos no terminal, na raiz do projeto, com o opencode fechado):\n")
	w("  1. Tentar de novo com o mesmo modelo: mais %d tentativas, e o modelo recebe o que já falhou para fazer diferente:\n       axyn retry\n", maxFailsPerModel)
	w("  2. Dar ao modelo uma instrução mais clara, com as suas palavras; o axyn grava como decisão na spec e tenta de novo, com novas tentativas:\n")
	w("       axyn decide \"%s\"\n", hint)
	w("  3. Trocar de modelo (o novo modelo ganha outras %d tentativas; para saber qual modelo resolve melhor cada etapa, avalie os seus com axyn bench):\n       axyn model\n       axyn run --resume\n", maxFailsPerModel)
	if t.WIP != "" {
		w("  4. Corrigir o código você mesmo, ou com outra IA, na branch %s:\n       git checkout %s\n       (corrija e faça commit)\n       git checkout %s\n       axyn run --resume\n", t.WIP, t.WIP, pl.Base)
	}
	if strings.TrimSpace(help) != "" {
		w("\n%s\n", strings.TrimSpace(help))
	}
	w("\nPara ver tudo o que aconteceu num arquivo: axyn history")
	return b.String()
}

func shorten(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// fixMojibake undoes UTF-8 text that was read as Latin-1 on the way ("mÃ­nimo" for
// "mínimo"), as the make output arrives on some Windows consoles.
func fixMojibake(s string) string {
	if !strings.ContainsAny(s, "ÃÂ") {
		return s
	}
	b := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 0xff {
			return s
		}
		b = append(b, byte(r))
	}
	if !utf8.Valid(b) {
		return s
	}
	return string(b)
}
