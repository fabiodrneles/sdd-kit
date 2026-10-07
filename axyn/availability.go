package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Models go down (#344): free models hit token or request limits, providers go offline.
// The engine reads the agent's output; a provider error, with no file changed, means the
// model is down: the attempt does not count, the model leaves the queue for this run, and
// the same work goes to the next model able to do it. Both conditions are needed, because
// the word "timeout" may well be the ticket itself.

var downRe = regexp.MustCompile(`(?i)(\b429\b|too many requests|rate[ _-]?limit|quota|insufficient[_ ]?(quota|credits|balance)|limit (exceeded|reached)|exceeded your|tokens? per (minute|day)|\b50[234]\b|service unavailable|bad gateway|overloaded|econnrefused|econnreset|enotfound|fetch failed|network error|connection (refused|reset)|model (not found|is not available|unavailable)|provider (error|unavailable))`)

// downReason is why the model looks down, or "" when it just worked (well or badly).
func downReason(out string, changed bool) string {
	if changed {
		return ""
	}
	m := downRe.FindString(out)
	if m == "" {
		return ""
	}
	l := strings.ToLower(m)
	switch {
	case strings.Contains(l, "429") || strings.Contains(l, "rate") || strings.Contains(l, "quota") || strings.Contains(l, "limit") || strings.Contains(l, "token") || strings.Contains(l, "insufficient") || strings.Contains(l, "exceeded") || strings.Contains(l, "too many"):
		return "limite de uso ou de tokens atingido (" + m + ")"
	}
	return "fora do ar (" + m + ")"
}

// tailWriter keeps the last max bytes written to it.
type tailWriter struct {
	mu  sync.Mutex
	max int
	b   []byte
}

func (t *tailWriter) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.b = append(t.b, p...)
	if len(t.b) > t.max {
		t.b = t.b[len(t.b)-t.max:]
	}
	return len(p), nil
}

func (t *tailWriter) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.b)
}

// markDown takes a model out of the queue for this run and says what happens next.
func (s *mcpServer) markDown(model, why string) {
	if s.down == nil {
		s.down = map[string]string{}
	}
	s.down[model] = why
}

func allDownMessage(down map[string]string) string {
	var b strings.Builder
	b.WriteString("Todos os modelos disponíveis ficaram fora de uso nesta execução, e por isso o axyn parou sem gastar tentativas:\n")
	for m, why := range down {
		fmt.Fprintf(&b, "  %s: %s\n", m, why)
	}
	b.WriteString("Limites de uso dos modelos gratuitos costumam voltar em minutos ou horas. Para seguir de onde parou, mais tarde, no terminal, na raiz do projeto: axyn run --resume\n")
	b.WriteString("Ou acrescente outro modelo à escada agora (axyn model) e rode axyn run --resume.")
	return b.String()
}
