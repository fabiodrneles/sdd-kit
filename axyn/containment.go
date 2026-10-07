package main

import (
	"fmt"
	"strings"
)

// The guide for contained models (#344). A model the bench put under the bar (or caught
// weakening a test) still works, after the good ones, but on a short leash the engine
// enforces itself, whatever the model does:
//   - scope: when the ticket names files, only those files and tests may change;
//   - tests: existing test files are read-only, new tests may only be added;
//   - size: the diff is limited to containedMaxLines;
//   - prompt: strict, step-by-step instructions, with no room to improvise.
// The usual gates still apply on top: the model has no choice but to do the work it was
// given, or be rejected.

const containedMaxLines = 150

const containedGuide = `MODO GUIADO (o axyn está conduzindo este trabalho passo a passo):
1. Faça somente o que o ticket abaixo pede, no menor número de linhas possível.
2. Mexa somente nos arquivos citados no ticket e em arquivos de teste novos.
3. Não altere nem apague testes que já existem; só acrescente testes novos.
4. Não renomeie, não reorganize e não "melhore" nada fora do ticket.
5. Rode os testes do projeto e só termine quando eles passarem.
Qualquer coisa fora destas regras é desfeita pelo axyn e a tentativa é reprovada.

`

func inList(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// isContained says whether model runs inside the guide for this kind of ticket.
func isContained(t *ticket, modelID string) bool {
	p, err := loadProfile()
	if err != nil || !p.Applied || modelID == "" {
		return false
	}
	for _, m := range p.Pinned {
		if m != modelID {
			continue
		}
		return false // a model the user chose by hand runs free
	}
	role := roleCode
	if t != nil && strings.HasPrefix(t.Title, "Testes para") {
		role = roleTests
	}
	return inList(p.Contained[role], modelID)
}

// containFindings are the extra gate findings of a contained attempt.
func (s *mcpServer) containFindings(t *ticket) []finding {
	base := s.base
	if base == "" {
		base = "HEAD"
	}
	d, err := gitDiff(s.dir, base)
	if err != nil {
		return nil
	}
	allowed := ticketPaths(t)
	var out []finding
	for _, f := range parseDiff(strings.NewReader(d)) {
		existed := false
		if _, err := s.git("cat-file", "-e", base+":"+f.path); err == nil {
			existed = true
		}
		if isTestPath(f.path) {
			if existed {
				out = append(out, finding{"guia", "modelo guiado não pode alterar um teste que já existe: " + f.path})
			}
			continue
		}
		if len(allowed) > 0 && !inTicket(f.path, allowed) {
			out = append(out, finding{"guia", fmt.Sprintf("arquivo fora do ticket: %s (o ticket cita %s)", f.path, strings.Join(allowed, ", "))})
		}
	}
	return out
}
