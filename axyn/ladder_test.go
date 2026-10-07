package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const ladderCfg = `# axyn
models:
  - id: openrouter/free-model
    key_env: AXYN_TEST_KEY_ONE
  - id: anthropic/strong-model
`

func ladderServer(t *testing.T, ci string) *mcpServer {
	t.Helper()
	dir := repo(t)
	cfgDir := t.TempDir()
	write(t, cfgDir, "config.yaml", ladderCfg)
	cfg := cfgDir + "/config.yaml"
	s := &mcpServer{dir: dir, ci: ci, base: "HEAD", maxLines: defaultMaxLines, config: cfg}
	spec := "- **AC-1** dado x\n"
	raw, _ := json.Marshal(map[string]any{"name": "demo", "spec": spec, "tickets": []map[string]any{
		{"title": "primeiro", "acs": []string{"AC-1"}},
	}})
	if _, err := s.call("axyn_plan", raw); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestParseModels(t *testing.T) {
	ms, err := parseModels(ladderCfg)
	if err != nil || len(ms) != 2 || ms[0].ID != "openrouter/free-model" || ms[0].KeyEnv != "AXYN_TEST_KEY_ONE" || ms[1].ID != "anthropic/strong-model" {
		t.Fatalf("models: %+v %v", ms, err)
	}
	if _, err := parseModels("models:\n  - id: x\n    api_key: sk-123\n"); err == nil || !strings.Contains(err.Error(), "key_env") {
		t.Fatalf("a literal key must be refused: %v", err)
	}
	if _, err := parseModels("models:\n  - key_env: A\n"); err == nil {
		t.Fatal("a model without id must be refused")
	}
	if ms, err := loadModels("/nonexistent/config.yaml"); err != nil || ms != nil {
		t.Fatalf("no config means no ladder: %v %v", ms, err)
	}
}

// 021 AC-2: two models in the ladder and the first rejected twice; axyn_gate points to the
// second, and the ticket records the attempts and the model that delivered (FR-7).
func TestLadderMovesToSecondModelAndRecords(t *testing.T) {
	s := ladderServer(t, "false")
	write(t, s.dir, "z.go", "package x\n")
	write(t, s.dir, "z_new_test.go", "package x\n") // new code comes with a test (#334)

	first, _ := s.call("axyn_gate", json.RawMessage("{}"))
	if !strings.Contains(first, "modelo atual openrouter/free-model") || strings.Contains(first, "próximo modelo") {
		t.Fatalf("first failure stays on the first model: %q", first)
	}
	var second string
	for i := 1; i < maxFailsPerModel; i++ {
		second, _ = s.call("axyn_gate", json.RawMessage("{}"))
	}
	if !strings.Contains(second, "use o próximo modelo: anthropic/strong-model") {
		t.Fatalf("second failure should point to the second model: %q", second)
	}

	s.ci = "true"
	t.Setenv("AXYN_COST_USD", "0.25")
	notes, err := s.call("axyn_ship", json.RawMessage(`{"message":"feat: z"}`))
	if err != nil || !strings.Contains(notes, fmt.Sprintf("registro: modelo anthropic/strong-model, %d tentativa(s)", maxFailsPerModel+1)) || !strings.Contains(notes, "0.2500") {
		t.Fatalf("ship: %v %q", err, notes)
	}
	pl, _, err := s.loadPlan()
	if err != nil {
		t.Fatal(err)
	}
	tk := pl.Tickets[0]
	if !tk.Done || tk.Model != "anthropic/strong-model" || len(tk.Attempts) != maxFailsPerModel+1 || tk.CostUSD != 0.25 || tk.Attempts[0].Model != "openrouter/free-model" || tk.Attempts[0].Green {
		t.Fatalf("ticket record: %+v", tk)
	}
}

// 021 FR-6: with the ladder exhausted the work is kept in a WIP commit and the report
// says what was tried.
func TestLadderExhaustedKeepsWIP(t *testing.T) {
	s := ladderServer(t, "false")
	write(t, s.dir, "z.go", "package x\n")
	write(t, s.dir, "z_new_test.go", "package x\n") // new code comes with a test (#334)
	var text string
	for i := 0; i < 2*maxFailsPerModel; i++ {
		text, _ = s.call("axyn_gate", json.RawMessage("{}"))
	}
	if !strings.Contains(text, "escada esgotada") || !strings.Contains(text, "openrouter/free-model") || !strings.Contains(text, "commit WIP") {
		t.Fatalf("exhausted report: %q", text)
	}
	if log, _ := s.git("log", "-1", "--format=%s"); !strings.HasPrefix(log, "wip:") {
		t.Fatalf("last commit should be WIP: %q", log)
	}
	if out, _ := s.git("status", "--porcelain"); out != "" {
		t.Fatalf("work should be committed: %q", out)
	}
}
