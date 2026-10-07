package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Recovery without changing the model (spec 021 FR-8, D18). A rejected diff goes through
// these steps, in order, before the ladder may move to another model. Each step has a
// cap of attempts and a cap on how much text it sends back (a stand-in for tokens).

type recStep struct {
	name  string
	tries int
}

var recSteps = []recStep{
	{"diagnóstico exato", 1},
	{"mais contexto", 1},
	{"várias tentativas", 2},
	{"ticket dividido", 3},
	{"plano antes do código", 1},
	{"pergunta ao usuário", 1},
}

const (
	stepAsk       = "pergunta ao usuário"
	stepPlan      = "plano antes do código"
	stepRecheck   = "código salvo conferido de novo"
	maxStepLines  = 30
	maxStepChars  = 3000
	maxSnippets   = 4
	snippetRadius = 2
)

// maxFailsPerModel is how many rejected diffs a model gets before the ladder moves on:
// the first try plus every attempt of every recovery step.
var maxFailsPerModel = 1 + func() int {
	n := 0
	for _, s := range recSteps {
		n += s.tries
	}
	return n
}()

// stepFor says which recovery step follows `fails` rejected diffs on a model: the index
// into recSteps and the attempt inside it (1-based). Index -1 means no failure yet.
func stepFor(fails int) (idx, try int) {
	if fails <= 0 {
		return -1, 0
	}
	off := fails - 1
	for i, s := range recSteps {
		if off < s.tries {
			return i, off + 1
		}
		off -= s.tries
	}
	return len(recSteps), 0
}

var (
	failTestRe = regexp.MustCompile(`--- FAIL: (\w+)`)
	fileLineRe = regexp.MustCompile(`^\s*([\w./-]+\.\w{1,5}):(\d+)(?::\d+)?:?\s*(.*)$`)
	errLineRe  = regexp.MustCompile(`(?i)^\s*(FAIL\b|error\b|.*\b(expected|want|got)\b|.*\b(assertion|panic)\b)`)
	ticketFile = regexp.MustCompile(`[\w./-]+\.\w{1,5}\b`)
)

type fileRef struct {
	path string
	line int
}

// diagnose is step 1, without an LLM: the failing test, the lint line and the code around
// it, instead of the whole log.
func diagnose(ciLog string) (lines []string, refs []fileRef) {
	seen := map[string]bool{}
	add := func(l string) {
		l = strings.TrimSpace(l)
		if l != "" && !seen[l] && len(lines) < 12 {
			seen[l] = true
			lines = append(lines, l)
		}
	}
	for _, l := range strings.Split(ciLog, "\n") {
		if m := fileLineRe.FindStringSubmatch(l); m != nil {
			n, _ := strconv.Atoi(m[2])
			if len(refs) < maxSnippets && !seen[m[1]+":"+m[2]] {
				seen[m[1]+":"+m[2]] = true
				refs = append(refs, fileRef{m[1], n})
			}
			add(l)
		} else if failTestRe.MatchString(l) || errLineRe.MatchString(l) {
			add(l)
		}
	}
	return lines, refs
}

// snippet returns the lines around ref, or nothing if the file is missing or outside dir.
func snippet(dir string, ref fileRef, radius int) []string {
	if filepath.IsAbs(ref.path) || strings.Contains(filepath.ToSlash(filepath.Clean(ref.path)), "..") {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(dir, ref.path))
	if err != nil {
		return nil
	}
	src := strings.Split(string(b), "\n")
	lo, hi := max(ref.line-1-radius, 0), min(ref.line+radius, len(src))
	var out []string
	for i := lo; i < hi; i++ {
		out = append(out, fmt.Sprintf("%s:%d| %s", ref.path, i+1, src[i]))
	}
	return out
}

// capText keeps a step's text under its cap of lines and characters.
func capText(lines []string) string {
	if len(lines) > maxStepLines {
		lines = append(lines[:maxStepLines], "... (cortado no teto do passo)")
	}
	s := strings.Join(lines, "\n")
	if len(s) > maxStepChars {
		s = s[:maxStepChars] + "\n... (cortado no teto do passo)"
	}
	return s
}

// ticketPaths are the files a ticket names; the plan step checks against them.
func ticketPaths(t *ticket) []string { return ticketFile.FindAllString(t.Title+" "+t.Body, -1) }

func inTicket(p string, allowed []string) bool {
	for _, a := range allowed {
		if p == a || strings.HasSuffix(p, "/"+a) || strings.HasSuffix(a, "/"+p) {
			return true
		}
	}
	return false
}

// checkPlan is step 5, done by the engine: the plan may only name the ticket's files.
func checkPlan(t *ticket, files []string) []finding {
	allowed := ticketPaths(t)
	if len(files) == 0 || len(allowed) == 0 {
		return nil
	}
	var out []finding
	for _, f := range files {
		if !inTicket(filepath.ToSlash(filepath.Clean(f)), allowed) {
			out = append(out, finding{"plan", "o plano cita arquivo fora do ticket: " + f})
		}
	}
	return out
}

// confusion looks for the signs that the model does not understand the ticket (FR-8 6):
// plans that contradict each other, or files outside the ticket.
func confusion(t *ticket) string {
	var plans [][]string
	for _, a := range t.Attempts {
		for _, r := range a.Reason {
			if strings.HasPrefix(r, "[plan]") {
				return "o plano cita arquivos fora do ticket"
			}
		}
		if len(a.Plan) > 0 {
			plans = append(plans, a.Plan)
		}
	}
	for i := 1; i < len(plans); i++ {
		if !overlap(plans[i-1], plans[i]) {
			return "os planos enviados se contradizem"
		}
	}
	return ""
}

func overlap(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

func asked(t *ticket) bool {
	for _, a := range t.Attempts {
		if a.Step == stepAsk {
			return true
		}
	}
	return false
}

// recoveryAdvice renders what the model must do next, for the step that follows `fails`
// rejected diffs on its model.
func (s *mcpServer) recoveryAdvice(t *ticket, fails int, ciLog string) string {
	idx, try := stepFor(fails)
	why := ""
	if c := confusion(t); c != "" && !asked(t) && idx < len(recSteps)-1 {
		idx, try, why = len(recSteps)-1, 1, c
	}
	if idx < 0 || idx >= len(recSteps) {
		return ""
	}
	st := recSteps[idx]
	head := fmt.Sprintf("recuperação, passo %d de %d «%s» (tentativa %d de %d, sem trocar de modelo):", idx+1, len(recSteps), st.name, try, st.tries)
	diag, refs := diagnose(ciLog)
	var body []string
	switch st.name {
	case "diagnóstico exato":
		body = append(body, "corrija só o que o diagnóstico aponta:")
		body = append(body, diag...)
		for _, r := range refs {
			body = append(body, snippet(s.dir, r, snippetRadius)...)
		}
		if len(diag) == 0 {
			body = append(body, "(o log não trouxe teste nem linha de erro; fim do log abaixo)")
			body = append(body, strings.Split(strings.TrimRight(tail(ciLog, 10), "\n"), "\n")...)
		}
	case "mais contexto":
		body = append(body, "mais contexto, só o que o erro aponta:")
		for _, r := range refs {
			body = append(body, snippet(s.dir, r, 15)...)
		}
		body = append(body, s.usages(diag)...)
	case "várias tentativas":
		body = append(body, fmt.Sprintf("proponha uma variação diferente do mesmo diff (variação %d de %d); vale a primeira que passa nos portões.", try, st.tries))
		body = append(body, diag...)
	case "ticket dividido":
		parts := []string{"só o teste que falha", "a implementação mínima que o faz passar", "o ajuste final"}
		body = append(body, fmt.Sprintf("divida o ticket; agora faça só %s (passo %d de %d), e rode axyn_gate antes de seguir.", parts[min(try-1, len(parts)-1)], try, st.tries))
	case stepPlan:
		body = append(body, "antes de editar, envie o plano em axyn_gate com `plan` (a lista de arquivos que vai mexer); o motor só aceita os arquivos do ticket:")
		body = append(body, ticketPaths(t)...)
	case stepAsk:
		if why == "" {
			why = "as tentativas anteriores não resolveram"
		}
		body = append(body,
			fmt.Sprintf("pergunta ao usuário (%s): o ticket «%s» ficou ambíguo. Qual destes é o que você quer?", why, t.Title),
			"  A) só os arquivos que o ticket cita, sem mexer em mais nada",
			"  B) dividir o ticket em dois, um por AC",
			"  C) outra instrução (escreva)",
			"para responder, no terminal, na raiz do projeto: axyn decide A (ou B, ou a sua instrução entre aspas); a resposta vira uma decisão na spec e o axyn continua sozinho.")
	}
	return head + "\n" + capText(body)
}

// usages lists where the failing function is used (step 2), via git grep.
func (s *mcpServer) usages(diag []string) []string {
	var out []string
	for _, d := range diag {
		m := failTestRe.FindStringSubmatch(d)
		if m == nil {
			continue
		}
		name := strings.TrimPrefix(m[1], "Test")
		if name == "" {
			continue
		}
		res, _ := s.git("grep", "-n", "-w", "-m", "3", "--", name)
		for i, l := range strings.Split(strings.TrimSpace(res), "\n") {
			if l != "" && i < 6 {
				out = append(out, "uso: "+l)
			}
		}
	}
	return out
}

// toolDecide records the user's answer as a decision in the spec, in its own commit so
// ticket diffs never touch the spec.
func (s *mcpServer) toolDecide(question, answer string) (string, error) {
	if strings.TrimSpace(question) == "" || strings.TrimSpace(answer) == "" {
		return "", fmt.Errorf("decisão recusada: pergunta e resposta são obrigatórias")
	}
	pl, _, err := s.loadPlan()
	if err != nil {
		return "", err
	}
	full := filepath.Join(s.dir, pl.Spec)
	b, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	n := strings.Count(string(b), "- **D-") + 1
	entry := fmt.Sprintf("- **D-%d** (decisão do usuário) %s → %s\n", n, oneLine(question), oneLine(answer))
	text := strings.TrimRight(string(b), "\n") + "\n"
	if !strings.Contains(text, "## Decisões") {
		text += "\n## Decisões\n\n"
	}
	if err := os.WriteFile(full, []byte(text+entry), 0o644); err != nil {
		return "", err
	}
	if out, err := s.git("add", "--", pl.Spec); err != nil {
		return "", fmt.Errorf("git add falhou: %s", out)
	}
	if out, err := s.git("commit", "-m", fmt.Sprintf("docs: decision D-%d in the spec", n), "--", pl.Spec); err != nil {
		return "", fmt.Errorf("git commit falhou: %s", out)
	}
	return fmt.Sprintf("decisão D-%d gravada em %s", n, pl.Spec), nil
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }
