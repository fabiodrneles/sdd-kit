package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// rpc sends requests to the server in dir and returns the tool text and isError of each tools/call.
func callTool(t *testing.T, dir, tool string, args map[string]any) (string, bool) {
	t.Helper()
	s := &mcpServer{dir: dir}
	raw, _ := json.Marshal(args)
	text, err := s.call(tool, raw)
	if err != nil {
		return err.Error(), true
	}
	return text, false
}

func TestMCPProtocol(t *testing.T) {
	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"bogus"}`,
	}, "\n") + "\n"
	var out, errb bytes.Buffer
	if code := runMCP(strings.NewReader(in), &out, &errb); code != exitOK {
		t.Fatalf("code %d", code)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 replies (no reply to the notification), got %d: %q", len(lines), out.String())
	}
	for _, name := range []string{"axyn_plan", "axyn_next", "axyn_gate", "axyn_ship"} {
		if !strings.Contains(lines[1], `"`+name+`"`) {
			t.Errorf("tools/list sem %s: %s", name, lines[1])
		}
	}
	if !strings.Contains(lines[2], "-32601") {
		t.Errorf("método desconhecido deveria dar -32601: %s", lines[2])
	}
}

// 021 AC-1: a fake agent deletes a test; axyn_gate fails with the reason and axyn_ship refuses the commit.
func TestMCPGateFailsAndShipRefuses(t *testing.T) {
	dir := repo(t)
	write(t, dir, "x_test.go", strings.Replace(twoTests, "func TestB", "func helperB", 1))

	text, isErr := callTool(t, dir, "axyn_gate", map[string]any{"ci": ""})
	if isErr || !strings.Contains(text, "reprovado [tests]") || !strings.Contains(text, "teste(s) a menos") {
		t.Fatalf("gate: isErr %v, %q", isErr, text)
	}

	before, _ := (&mcpServer{dir: dir}).git("rev-parse", "HEAD")
	text, isErr = callTool(t, dir, "axyn_ship", map[string]any{"message": "feat: x", "ci": ""})
	if !isErr || !strings.Contains(text, "ship recusado") || !strings.Contains(text, "teste(s) a menos") {
		t.Fatalf("ship: isErr %v, %q", isErr, text)
	}
	after, _ := (&mcpServer{dir: dir}).git("rev-parse", "HEAD")
	if before != after {
		t.Fatalf("ship recusado não pode commitar: %s -> %s", before, after)
	}
}

func TestMCPShipGreen(t *testing.T) {
	dir := repo(t)
	write(t, dir, "y.go", "package x\n")
	text, isErr := callTool(t, dir, "axyn_ship", map[string]any{"message": "feat: add y", "ci": ""})
	if isErr || !strings.Contains(text, "commit em feat/add-y") {
		t.Fatalf("isErr %v, %q", isErr, text)
	}
	if out, _ := (&mcpServer{dir: dir}).git("status", "--porcelain"); out != "" {
		t.Fatalf("árvore suja depois do ship: %q", out)
	}
}

func TestMCPPlanAndNext(t *testing.T) {
	dir := repo(t)
	spec := "# Spec\n\n- **FR-1** faz x\n- **AC-1** dado x, então y\n- **AC-2** dado z\n"
	bad := map[string]any{"name": "demo", "spec": spec, "tickets": []map[string]any{{"title": "t", "acs": []string{"AC-9"}}}}
	if text, isErr := callTool(t, dir, "axyn_plan", bad); !isErr || !strings.Contains(text, "AC-9") {
		t.Fatalf("plano com AC inexistente deveria ser recusado: %v %q", isErr, text)
	}
	if _, isErr := callTool(t, dir, "axyn_next", nil); !isErr {
		t.Fatal("axyn_next sem plano deveria dar erro")
	}

	good := map[string]any{"name": "demo", "spec": spec, "tickets": []map[string]any{
		{"title": "primeiro", "body": "faça x", "acs": []string{"AC-1"}},
		{"title": "segundo", "acs": []string{"AC-2"}},
	}}
	if text, isErr := callTool(t, dir, "axyn_plan", good); isErr || !strings.Contains(text, "specs/001-demo/spec.md") {
		t.Fatalf("plan: %v %q", isErr, text)
	}
	text, isErr := callTool(t, dir, "axyn_next", nil)
	if isErr || !strings.Contains(text, "Ticket 1: primeiro") || !strings.Contains(text, "AC-1** dado x") || strings.Contains(text, "AC-2**") {
		t.Fatalf("next: %v %q", isErr, text)
	}

	write(t, dir, "z.go", "package x\n")
	if text, isErr := callTool(t, dir, "axyn_ship", map[string]any{"message": "feat: z", "ci": ""}); isErr {
		t.Fatalf("ship: %q", text)
	}
	text, _ = callTool(t, dir, "axyn_next", nil)
	if !strings.Contains(text, "Ticket 2: segundo") {
		t.Fatalf("next depois do ship: %q", text)
	}
}
