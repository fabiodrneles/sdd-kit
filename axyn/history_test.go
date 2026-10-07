package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	in := "chave sk-abc123def456 e GITHUB_TOKEN=ghp_abcdef123456 e OPENROUTER_API_KEY: xyz789\n" +
		"Authorization: Bearer abcdefgh12345678\nremoto https://ana:segredo@github.com/o/r.git\n" +
		"github_pat_11ABCDEFG0123456789 AIzaSyA1234567890abcdefghijk"
	out := redact(in)
	for _, leak := range []string{"sk-abc123def456", "ghp_abcdef123456", "xyz789", "abcdefgh12345678", "segredo", "github_pat_11ABCDEFG", "AIzaSyA1234567890"} {
		if strings.Contains(out, leak) {
			t.Errorf("vazou %q:\n%s", leak, out)
		}
	}
	if !strings.Contains(out, "GITHUB_TOKEN=[removido]") || !strings.Contains(out, "github.com/o/r.git") {
		t.Errorf("a redação apagou demais:\n%s", out)
	}
}

// 021 FR-7: a run with failed attempts and a question: the history has each failure with
// its reason, the question and the answer, the commits and the log, and no secret.
func TestHistoryHasEverythingAndNoSecret(t *testing.T) {
	dir := repo(t)
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `rm -f x_test.go; echo "usando OPENAI_API_KEY=sk-vazou123456 GITHUB_TOKEN=ghp_vazou123456"`))
	if code, out := runLoopIn(t, dir); code == exitOK || !strings.Contains(out, askMarker) {
		t.Fatalf("a execução deveria parar com a pergunta: code %d\n%s", code, out)
	}
	if _, err := (&mcpServer{dir: dir}).toolDecide("o ticket «Um» ficou ambíguo?", "A) só os arquivos do ticket"); err != nil {
		t.Fatal(err)
	}

	var out, errb bytes.Buffer
	if code := runHistoryCmd([]string{"--dir", dir}, &out, &errb); code != exitOK {
		t.Fatalf("code %d: %s", code, errb.String())
	}
	path := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(out.String(), ";", 2)[0], "histórico gravado em "))
	if filepath.Dir(path) != filepath.Join(dir, ".axyn") {
		t.Errorf("o padrão é .axyn/ (fora do git): %s", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h := string(b)
	for _, want := range []string{
		"**Pedido:** crie uma coisa", "### Ticket 1: Um (aberto)", "Tentativa 1 (", "reprovado", "teste",
		"Tentativa 2 (", "pergunta ao usuário", "(decisão do usuário) o ticket «Um» ficou ambíguo? → A) só os arquivos do ticket",
		"docs: decision D-1 in the spec", "## Ambiente", "### axyn doctor", "## Log completo", "== sh axyn-code",
	} {
		if !strings.Contains(h, want) {
			t.Errorf("o histórico não traz %q", want)
		}
	}
	for _, leak := range []string{"sk-vazou123456", "ghp_vazou123456"} {
		if strings.Contains(h, leak) {
			t.Errorf("o histórico vazou %q", leak)
		}
	}
	if st, _ := (&mcpServer{dir: dir}).git("status", "--porcelain"); st != "" {
		t.Errorf("o histórico sujou a árvore: %s", st)
	}

	// --out goes to any folder, and the MCP tool does the same.
	other := filepath.Join(t.TempDir(), "meu-historico.md")
	text, err := (&mcpServer{dir: dir}).call("axyn_history", []byte(`{"out":"`+other+`"}`))
	if err != nil || !strings.Contains(text, other) {
		t.Fatalf("axyn_history: %v %s", err, text)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("--out não gravou: %v", err)
	}
}

func TestHistoryWithoutRun(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runHistoryCmd([]string{"--dir", t.TempDir()}, &out, &errb); code != exitFail || !strings.Contains(errb.String(), "nenhuma execução") {
		t.Errorf("sem execução: code %d %s", code, errb.String())
	}
}
