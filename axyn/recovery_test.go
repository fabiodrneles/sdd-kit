package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const failingCIScript = `if [ -f fix.txt ]; then exit 0; fi
echo '=== RUN   TestFoo'
echo 'noise line one'
echo '--- FAIL: TestFoo (0.00s)'
echo '    x_test.go:3: expected 1, got 2'
echo 'noise line two'
exit 1`

const failingCI = "failing-ci"

// oneModelServer is a server with a single model on the ladder and a ticket that names x.go.
func oneModelServer(t *testing.T, ci string) *mcpServer {
	t.Helper()
	if ci == failingCI {
		// a script file keeps the echoed lines out of the "[ci] `cmd` falhou" reason
		p := filepath.Join(t.TempDir(), "ci.sh")
		if err := os.WriteFile(p, []byte(failingCIScript), 0o644); err != nil {
			t.Fatal(err)
		}
		ci = "sh " + p
	}
	dir := repo(t)
	cfgDir := t.TempDir()
	write(t, cfgDir, "config.yaml", "models:\n  - id: only/model\n")
	s := &mcpServer{dir: dir, ci: ci, base: "HEAD", maxLines: defaultMaxLines, config: filepath.Join(cfgDir, "config.yaml")}
	raw, _ := json.Marshal(map[string]any{"name": "demo", "spec": "- **AC-1** dado x\n", "tickets": []map[string]any{
		{"title": "Foo", "body": "ajustar x.go", "acs": []string{"AC-1"}},
	}})
	if _, err := s.call("axyn_plan", raw); err != nil {
		t.Fatal(err)
	}
	return s
}

func gateText(t *testing.T, s *mcpServer, args string) string {
	t.Helper()
	text, err := s.call("axyn_gate", json.RawMessage(args))
	if err != nil {
		t.Fatal(err)
	}
	return text
}

// 021 AC-6, first half: a fake agent that only gets it right after receiving the exact
// diagnosis. One model, no other model asked for; the ticket records the strategy.
func TestRecoveryDiagnosisDeliversWithOneModel(t *testing.T) {
	s := oneModelServer(t, failingCI)
	first := gateText(t, s, "{}")
	if !strings.Contains(first, "reprovado [ci]") {
		t.Fatalf("first try should fail: %q", first)
	}
	if !strings.Contains(first, "passo 1 de 6 «diagnóstico exato»") ||
		!strings.Contains(first, "--- FAIL: TestFoo") || !strings.Contains(first, "x_test.go:3: expected 1, got 2") ||
		!strings.Contains(first, "x_test.go:3| ") {
		t.Fatalf("report should carry the exact diagnosis and the code: %q", first)
	}
	if strings.Contains(first, "noise line") || strings.Contains(first, "próximo modelo") {
		t.Fatalf("report should not carry the whole log nor ask for another model: %q", first)
	}

	write(t, s.dir, "fix.txt", "ok\n") // the fake agent acts on the diagnosis
	if got := gateText(t, s, "{}"); !strings.Contains(got, "gate: verde") {
		t.Fatalf("second try should pass: %q", got)
	}
	notes, err := s.call("axyn_ship", json.RawMessage(`{"message":"feat: foo"}`))
	if err != nil || !strings.Contains(notes, "estratégia diagnóstico exato") {
		t.Fatalf("ship: %v %q", err, notes)
	}
	pl, _, _ := s.loadPlan()
	if tk := pl.Tickets[0]; tk.Strategy != "diagnóstico exato" || tk.Model != "only/model" || len(tk.Attempts) != 3 { // the failure, the green gate and the one ship runs
		t.Fatalf("record: %+v", tk)
	}
}

// 021 AC-6, second half: an agent that never gets it right goes through the steps in
// order, within the caps, is asked the question and ends in a WIP commit with the reason.
func TestRecoveryStepsInOrderThenAskThenWIP(t *testing.T) {
	s := oneModelServer(t, failingCI)
	write(t, s.dir, "z.go", "package x\n")
	var seen []string
	var last string
	for i := 0; i < maxFailsPerModel; i++ {
		last = gateText(t, s, "{}")
		if len(last) > 2*maxStepChars {
			t.Fatalf("report over the step cap: %d chars", len(last))
		}
		for j, st := range recSteps {
			if strings.Contains(last, "«"+st.name+"»") && (len(seen) == 0 || seen[len(seen)-1] != st.name) {
				seen = append(seen, st.name)
				if j != len(seen)-1 {
					t.Fatalf("step %q out of order after %v", st.name, seen)
				}
			}
		}
		if strings.Contains(last, "escada esgotada") {
			break
		}
	}
	if len(seen) != len(recSteps) {
		t.Fatalf("steps seen %v", seen)
	}
	if !strings.Contains(last, "escada esgotada") || !strings.Contains(last, "commit WIP") || !strings.Contains(last, "[ci]") {
		t.Fatalf("should end in WIP with the reason: %q", last)
	}
	pl, _, _ := s.loadPlan()
	if n := len(pl.Tickets[0].Attempts); n != maxFailsPerModel {
		t.Fatalf("attempts %d, cap %d", n, maxFailsPerModel)
	}
	if log, _ := s.git("log", "-1", "--format=%s"); !strings.HasPrefix(log, "wip:") {
		t.Fatalf("last commit should be WIP: %q", log)
	}
}

// The question step shows options, and the answer becomes a decision in the spec.
func TestRecoveryAskRecordsDecisionInSpec(t *testing.T) {
	s := oneModelServer(t, failingCI)
	var text string
	for i := 0; i < maxFailsPerModel-1 && !strings.Contains(text, "pergunta ao usuário (as tentativas"); i++ {
		text = gateText(t, s, "{}")
	}
	if !strings.Contains(text, "A) ") || !strings.Contains(text, "axyn_decide") {
		t.Fatalf("question with options expected: %q", text)
	}
	out, err := s.call("axyn_decide", json.RawMessage(`{"question":"qual arquivo?","answer":"só x.go"}`))
	if err != nil || !strings.Contains(out, "D-1") {
		t.Fatalf("decide: %v %q", err, out)
	}
	pl, _, _ := s.loadPlan()
	b, _ := os.ReadFile(filepath.Join(s.dir, pl.Spec))
	if !strings.Contains(string(b), "(decisão do usuário) qual arquivo? → só x.go") {
		t.Fatalf("spec: %q", b)
	}
	if st, _ := s.git("status", "--porcelain"); strings.Contains(st, "spec.md") {
		t.Fatalf("decision should be committed on its own: %q", st)
	}
	if _, err := s.call("axyn_decide", json.RawMessage(`{"question":"","answer":"x"}`)); err == nil {
		t.Fatal("empty question must be refused")
	}
}

// A plan naming files outside the ticket is a sign of confusion: the engine rejects it
// and jumps to the question.
func TestRecoveryPlanOutsideTicketJumpsToQuestion(t *testing.T) {
	s := oneModelServer(t, "true")
	text := gateText(t, s, `{"plan":["x.go","secrets.env"]}`)
	if !strings.Contains(text, "reprovado [plan]") || !strings.Contains(text, "secrets.env") || !strings.Contains(text, "pergunta ao usuário (o plano cita arquivos fora do ticket)") {
		t.Fatalf("plan check: %q", text)
	}
	if got := gateText(t, s, `{"plan":["x.go"]}`); strings.Contains(got, "reprovado [plan]") {
		t.Fatalf("a plan within the ticket is accepted: %q", got)
	}
}

func TestStepFor(t *testing.T) {
	if i, _ := stepFor(0); i != -1 {
		t.Fatal("no failure, no step")
	}
	want := []string{"diagnóstico exato", "mais contexto", "várias tentativas", "várias tentativas", "ticket dividido", "ticket dividido", "ticket dividido", stepPlan, stepAsk}
	for n, w := range want {
		if i, _ := stepFor(n + 1); recSteps[i].name != w {
			t.Fatalf("after %d fails want %q got %q", n+1, w, recSteps[i].name)
		}
	}
	if i, _ := stepFor(len(want) + 1); i != len(recSteps) || maxFailsPerModel != len(want)+1 {
		t.Fatalf("past the last step: %d, cap %d", i, maxFailsPerModel)
	}
}
