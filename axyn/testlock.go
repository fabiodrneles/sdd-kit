package main

import (
	"fmt"
	"strings"
)

// The lock on the tests a project already has (#381): a ticket that writes code may add
// tests, but not change or remove the ones the base has. Those tests are the contract the
// code is judged by, and a model that cannot make them pass otherwise loosens them. After
// each attempt the engine puts them back; when the model insists, the owner decides.

// testLockAsk is how many attempts may touch the same locked test before the owner is asked.
const testLockAsk = 2

const testLockMark = "trava dos testes"

// lockedTestEdits are the test files the base has that the open ticket changed or removed,
// but for the ones the owner released. Tests only added to an existing file are fine.
func (s *mcpServer) lockedTestEdits(t *ticket) []string {
	d, err := gitDiff(s.dir, s.baseRef())
	if err != nil {
		return nil
	}
	var out []string
	for _, f := range parseDiff(strings.NewReader(d)) {
		if !isTestPath(f.path) || !isCodePath(f.path) || inList(t.TestsUnlocked, f.path) {
			continue
		}
		if !f.deleted && len(f.removed) == 0 {
			continue
		}
		if _, err := s.git("cat-file", "-e", s.baseRef()+":"+f.path); err != nil {
			continue
		}
		out = append(out, f.path)
	}
	return out
}

// freezeTests puts the locked tests back from the base and counts, per file, the attempts
// that touched them. It returns the files it put back and the ones to ask the owner about.
func (s *mcpServer) freezeTests(id int) (undone, ask []string) {
	pl, path, err := s.loadPlan()
	if err != nil {
		return nil, nil
	}
	var t *ticket
	for i := range pl.Tickets {
		if pl.Tickets[i].ID == id {
			t = &pl.Tickets[i]
		}
	}
	if t == nil || t.testsOnly() {
		return nil, nil
	}
	undone = s.lockedTestEdits(t)
	if len(undone) == 0 {
		return nil, nil
	}
	if t.LockHits == nil {
		t.LockHits = map[string]int{}
	}
	for _, p := range undone {
		_, _ = s.git("checkout", s.baseRef(), "--", p)
		t.LockHits[p]++
		if t.LockHits[p] >= testLockAsk {
			ask = append(ask, p)
		}
	}
	t.LockAsk = ask
	_ = s.savePlan(pl, path)
	return undone, ask
}

// testLockQuestion is the question the run stops with, in the shape axyn decide answers.
func testLockQuestion(t *ticket, files []string) string {
	return strings.Join([]string{
		fmt.Sprintf("%s (%s): o ticket «%s» tentou mudar, em %d tentativas, testes que já existem: %s. O axyn desfez as mudanças. Qual destes é o que você quer?", askMarker[:len(askMarker)-2], testLockMark, t.Title, testLockAsk, strings.Join(files, ", ")),
		"  A) liberar esses testes para este ticket (o comportamento mudou de propósito e os testes precisam acompanhar)",
		"  B) manter travados (o ticket precisa passar sem mudar esses testes)",
		"para responder, no terminal, na raiz do projeto: axyn decide A (ou B); a resposta vira uma decisão na spec e o axyn continua sozinho.",
	}, "\n")
}

// testLockPrompt tells the coding model, up front, which tests are locked.
func testLockPrompt(t *ticket) string {
	s := "\n\nTrava dos testes: os testes que o projeto já tem estão travados. Acrescente testes novos (num arquivo novo ou no fim de um existente), mas não mude nem apague linhas dos testes existentes: o axyn desfaz essas mudanças."
	if len(t.TestsUnlocked) > 0 {
		s += " O dono liberou para este ticket: " + strings.Join(t.TestsUnlocked, ", ") + "."
	}
	return s
}

// answerTestLock applies the owner's answer to the lock question to the open ticket.
func answerTestLock(t *ticket, answer string) {
	if strings.HasPrefix(answer, "A)") {
		for _, p := range t.LockAsk {
			if !inList(t.TestsUnlocked, p) {
				t.TestsUnlocked = append(t.TestsUnlocked, p)
			}
		}
	}
	t.LockAsk, t.LockHits = nil, nil
}
