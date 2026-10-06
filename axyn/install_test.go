package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readCfg(t *testing.T, dir string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func section(t *testing.T, cfg map[string]any, key string) map[string]any {
	t.Helper()
	m, ok := cfg[key].(map[string]any)
	if !ok {
		t.Fatalf("seção %q ausente em %v", key, cfg)
	}
	return m
}

// 021 FR-2: without an opencode.json, install creates one with everything.
func TestInstallCreates(t *testing.T) {
	dir := t.TempDir()
	out, _, code := runCLI("install", "--dir", dir)
	if code != exitOK || !strings.Contains(out, "opencode.json") {
		t.Fatalf("code %d, out %q", code, out)
	}
	cfg := readCfg(t, dir)
	for _, key := range []string{"mcp", "command"} {
		if _, ok := section(t, cfg, key)["axyn"]; !ok {
			t.Errorf("%s.axyn ausente", key)
		}
	}
	agents := section(t, cfg, "agent")
	for _, name := range []string{"axyn-plan", "axyn-code"} {
		if _, ok := agents[name]; !ok {
			t.Errorf("agente %s ausente", name)
		}
	}
	tools := agents["axyn-code"].(map[string]any)["tools"].(map[string]any)
	if tools["axyn_*"] != false || tools["edit"] != true {
		t.Errorf("ferramentas do axyn-code: %v", tools)
	}
}

// 021 AC-3: the user's own configuration stays, with axyn added beside it.
func TestInstallKeepsUserConfig(t *testing.T) {
	dir := t.TempDir()
	mine := `{"model":"openrouter/x","mcp":{"fs":{"type":"local","command":["fs-mcp"]}},` +
		`"agent":{"review":{"mode":"primary"}},"command":{"test":{"template":"run tests"}}}`
	if err := os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, code := runCLI("install", "--dir", dir); code != exitOK {
		t.Fatalf("code %d", code)
	}
	cfg := readCfg(t, dir)
	if cfg["model"] != "openrouter/x" {
		t.Errorf("model perdido: %v", cfg["model"])
	}
	want := map[string][]string{"mcp": {"fs", "axyn"}, "agent": {"review", "axyn-plan", "axyn-code"}, "command": {"test", "axyn"}}
	for key, names := range want {
		sec := section(t, cfg, key)
		for _, name := range names {
			if _, ok := sec[name]; !ok {
				t.Errorf("%s.%s ausente", key, name)
			}
		}
	}
}

func TestInstallIdempotent(t *testing.T) {
	dir := t.TempDir()
	runCLI("install", "--dir", dir)
	first, _ := os.ReadFile(filepath.Join(dir, "opencode.json"))
	runCLI("install", "--dir", dir)
	second, _ := os.ReadFile(filepath.Join(dir, "opencode.json"))
	if string(first) != string(second) {
		t.Fatal("segunda instalação mudou o arquivo")
	}
}

// A file install cannot parse is never overwritten.
func TestInstallRefusesInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")
	orig := "{\n // meu comentário\n \"model\": \"x\"\n}\n"
	if err := os.WriteFile(path, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	_, errOut, code := runCLI("install", "--dir", dir)
	got, _ := os.ReadFile(path)
	if code != exitFail || string(got) != orig || !strings.Contains(errOut, "nada foi alterado") {
		t.Fatalf("code %d, err %q, file %q", code, errOut, got)
	}
}

func TestInstallRefusesNonObjectSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")
	orig := `{"mcp":[]}`
	if err := os.WriteFile(path, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, code := runCLI("install", "--dir", dir)
	got, _ := os.ReadFile(path)
	if code != exitFail || string(got) != orig {
		t.Fatalf("code %d, file %q", code, got)
	}
}
