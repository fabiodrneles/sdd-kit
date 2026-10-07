package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #356: the real lines of the owner's bench log, as a person reads them.
func TestPrettyLine(t *testing.T) {
	cases := map[string]string{
		`=== opencode/ling-3.1-flash-free em plan-001 (C:\Users\Gamer\AppData\Local\Temp\axyn-bench-plan-001-3600530417)`: "▶ opencode/ling-3.1-flash-free · plano",
		`== opencode axyn-plan-open`:                                                                     "agente axyn-plan-open",
		`Error: OpenCode's free tier can only be used from within OpenCode`:                              "⚠ Error: OpenCode's free tier",
		`--- nota 100, passou true, trapaça false: cobertura 100%, 8 de 8 bugs injetados pegos`:          "✔ nota 100 · cobertura 100%, 8 de 8 bugs",
		`--- nota 0, passou false, trapaça false: o plano não foi gravado com a ferramenta axyn_plan`:    "✘ nota 0 · o plano não foi gravado",
		`--- nota 0, passou false, trapaça true: trapaça: alterou o teste slug_test.go em vez do código`: "✘ trapaça (veto)",
		"⚙ axyn_axyn_plan {\"name\":\"notas-apagar-id\",\"spec\":\"" + strings.Repeat("x", 3000) + "\"}": "axyn_axyn_plan {…} (3.0 KB)",
	}
	for in, want := range cases {
		if got := prettyLine(in, false); !strings.Contains(got, want) {
			t.Errorf("prettyLine(%.60q) = %q, quer %q", in, got, want)
		}
	}
	if got := prettyLine("--- nota 0, passou false, trapaça false: indisponível durante a avaliação: fora do ar (x)", false); !strings.Contains(got, "⏸ indisponível") {
		t.Errorf("indisponível não é reprovado: %q", got)
	}
	if prettyLine("== opencode axyn-code\r", false) == "" || prettyLine("   ", false) != "" {
		t.Error("linhas vazias somem; a do agente fica")
	}
}

func TestLogsOnce(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	_ = os.MkdirAll(benchDir(), 0o755)
	_ = writeText(filepath.Join(benchDir(), "bench-1.log"), "=== m em code-001 (x)\n--- nota 90, passou true, trapaça false: \n")
	var o, e strings.Builder
	if code := runLogsCmd([]string{"--dir", dir, "--once"}, &o, &e); code != exitOK || !strings.Contains(o.String(), "▶ m · código") || !strings.Contains(o.String(), "✔ nota 90") {
		t.Errorf("logs: %d %s %s", code, o.String(), e.String())
	}
}
