package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeOpencode answers `opencode models` with a fixed list, and the config lives in a temp dir.
func fakeOpencode(t *testing.T) string {
	t.Helper()
	bin := t.TempDir()
	write(t, bin, "opencode", "#!/bin/sh\nprintf '\\nopencode/big-pickle\\nopencode/nemotron-3-ultra-free\\nopencode/exo-free\\nopenrouter/qwen/qwen3-coder:free\\nanthropic/claude-sonnet-5-5\\n\\n'\n")
	if err := os.Chmod(filepath.Join(bin, "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	old := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/true", nil }
	t.Cleanup(func() { lookPath = old })
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	return filepath.Join(cfg, "axyn", "config.yaml")
}

func modelCmd(t *testing.T, input string, args ...string) (int, string) {
	t.Helper()
	old := stdin
	stdin = strings.NewReader(input)
	defer func() { stdin = old }()
	var out, errb bytes.Buffer
	code := runModelCmd(args, &out, &errb)
	return code, out.String() + errb.String()
}

func readLadder(t *testing.T, path string) []model {
	t.Helper()
	ms, err := loadModels(path)
	if err != nil {
		t.Fatal(err)
	}
	return ms
}

// 021 FR-6: the menu lists only the free models, and the number picks one into the config.
func TestModelMenuPicksAFreeModel(t *testing.T) {
	cfg := fakeOpencode(t)
	code, out := modelCmd(t, "1\n")
	if code != exitOK {
		t.Fatalf("code %d\n%s", code, out)
	}
	if strings.Contains(out, "big-pickle") || strings.Contains(out, "claude") || !strings.Contains(out, " 3) openrouter/qwen/qwen3-coder:free") {
		t.Errorf("a lista deveria ter só os gratuitos, numerados:\n%s", out)
	}
	if ms := readLadder(t, cfg); len(ms) != 1 || ms[0].ID != "opencode/nemotron-3-ultra-free" || ms[0].KeyEnv != "" {
		t.Errorf("gravou %+v", ms)
	}
	if !strings.Contains(out, "nenhum (o axyn usa o modelo padrão do opencode)") {
		t.Errorf("sem o modelo atual:\n%s", out)
	}

	// OpenRouter: asks the variable name (Enter keeps the default) and never stores a key.
	code, out = modelCmd(t, "3\n\n")
	if code != exitOK || !strings.Contains(out, "modelo atual: opencode/nemotron-3-ultra-free") {
		t.Fatalf("code %d\n%s", code, out)
	}
	ms := readLadder(t, cfg)
	if len(ms) != 2 || ms[0].ID != "openrouter/qwen/qwen3-coder:free" || ms[0].KeyEnv != "OPENROUTER_API_KEY" || ms[1].ID != "opencode/nemotron-3-ultra-free" {
		t.Errorf("o novo vai para o topo e o anterior fica na escada: %+v", ms)
	}
}

func TestModelMenuRefusesBadChoice(t *testing.T) {
	cfg := fakeOpencode(t)
	for _, in := range []string{"9\n", "abc\n"} {
		if code, out := modelCmd(t, in); code != exitUsage || !strings.Contains(out, "nada mudou") {
			t.Errorf("entrada %q: code %d\n%s", in, code, out)
		}
	}
	if code, _ := modelCmd(t, "\n"); code != exitOK {
		t.Errorf("Enter vazio cancela sem erro")
	}
	if _, err := os.Stat(cfg); err == nil {
		t.Errorf("gravou com uma escolha inválida ou cancelada")
	}
}

func TestModelByIDAndOnly(t *testing.T) {
	cfg := fakeOpencode(t)
	if code, out := modelCmd(t, "", "opencode/exo-free"); code != exitOK {
		t.Fatalf("code %d\n%s", code, out)
	}
	if code, out := modelCmd(t, "", "openrouter/qwen/qwen3-coder:free", "--key-env", "MINHA_CHAVE"); code != exitOK {
		t.Fatalf("code %d\n%s", code, out)
	}
	ms := readLadder(t, cfg)
	if len(ms) != 2 || ms[0].KeyEnv != "MINHA_CHAVE" || ms[1].ID != "opencode/exo-free" {
		t.Errorf("escada: %+v", ms)
	}
	if code, out := modelCmd(t, "", "opencode/exo-free", "--only"); code != exitOK {
		t.Fatalf("code %d\n%s", code, out)
	}
	if ms := readLadder(t, cfg); len(ms) != 1 || ms[0].ID != "opencode/exo-free" {
		t.Errorf("--only deveria deixar um: %+v", ms)
	}
	if code, out := modelCmd(t, "", "opencode/exo-free", "--key-env", "sk-or-v1-abc"); code != exitUsage || !strings.Contains(out, "nunca a chave") {
		t.Errorf("uma chave em --key-env deveria ser recusada: %d %s", code, out)
	}
	if b, _ := os.ReadFile(cfg); strings.Contains(string(b), "sk-or") {
		t.Errorf("a chave foi parar no arquivo")
	}
}

func TestDoctorShowsTheModel(t *testing.T) {
	fakeGH(t, true)
	dir := githubRepoDir(t)
	if _, out := setupCmd(t, false, "--dir", dir); !strings.Contains(out, "modelo do axyn: nenhum escolhido") || !strings.Contains(out, "axyn model") {
		t.Errorf("sem config, o doctor deveria sugerir axyn model:\n%s", out)
	}
	cfg := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "axyn", "config.yaml")
	write(t, filepath.Dir(cfg), "config.yaml", formatModels([]model{{ID: "openrouter/x:free", KeyEnv: "AXYN_TEST_SEM_CHAVE"}}))
	if code, out := setupCmd(t, false, "--dir", dir); code == exitOK || !strings.Contains(out, "AXYN_TEST_SEM_CHAVE não está definida") {
		t.Errorf("variável da chave ausente deveria faltar: %d\n%s", code, out)
	}
	t.Setenv("AXYN_TEST_SEM_CHAVE", "x")
	if code, out := setupCmd(t, false, "--dir", dir); code != exitOK || !strings.Contains(out, "modelo do axyn: openrouter/x:free; para trocar: axyn model") {
		t.Errorf("com a chave: %d\n%s", code, out)
	}
}

// 021 FR-6: the planner runs on the ladder's first model too, not on opencode's default.
func TestPlannerUsesTheLadderModel(t *testing.T) {
	dir := repo(t)
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `printf 'ticket %s\n' "$n" > "f$n.txt"`))
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	write(t, filepath.Dir(cfg), "config.yaml", formatModels([]model{{ID: "opencode/nemotron-3-ultra-free"}}))
	var out, errb bytes.Buffer
	if code := runRunCmd([]string{"--wait", "--dir", dir, "--ci", "true", "--config", cfg, "crie uma coisa"}, &out, &errb); code != exitOK {
		t.Fatalf("code %d\n%s%s", code, out.String(), errb.String())
	}
	calls, _ := os.ReadFile(os.Getenv("FAKE_LOG"))
	first := strings.SplitN(string(calls), "\n", 2)[0]
	if !strings.HasPrefix(first, "run --agent axyn-plan --model opencode/nemotron-3-ultra-free") {
		t.Errorf("o plano deveria rodar no modelo da escada: %s", first)
	}
}
