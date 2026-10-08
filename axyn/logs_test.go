package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// #356: axyn bench plano testes is the short form of --tasks plan,tests, in PT or EN.
func TestPickTasks(t *testing.T) {
	roles := func(ts []benchTask) string {
		var r []string
		for _, x := range ts {
			r = append(r, x.Role)
		}
		return strings.Join(r, ",")
	}
	if got := roles(pickTasks([]string{"plano", "Testes"})); got != roleplan+","+roleTests {
		t.Fatalf("plano testes: %s", got)
	}
	if got := roles(pickTasks([]string{"código"})); got != roleCode {
		t.Fatalf("código: %s", got)
	}
	if got := pickTasks([]string{""}); len(got) != len(benchTasks) {
		t.Fatalf("nothing named must mean all: %d", len(got))
	}
}

// #356: the log view of the panel shows the end of the log, then only what is added.
func TestLogFollow(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	_ = os.MkdirAll(runsDir(dir), 0o755)
	p := filepath.Join(runsDir(dir), "1.log")
	_ = os.WriteFile(p, []byte("=== m em plan-001 (x)\n--- nota 100, passou true, trapaça false: ok\n"), 0o644)
	f := newLogFollow(dir, 15)
	first := strings.Join(f.lines(), "\n")
	if !strings.Contains(first, "plano") || !strings.Contains(first, "nota 100") {
		t.Fatalf("backlog: %q", first)
	}
	if again := f.lines(); len(again) != 0 {
		t.Fatalf("nothing new must show nothing: %q", again)
	}
	fh, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = fh.WriteString("Error: boom\nmeia")
	_ = fh.Close()
	got := strings.Join(f.lines(), "\n")
	if !strings.Contains(got, "⚠ Error: boom") || strings.Contains(got, "meia") {
		t.Fatalf("new lines (and no partial line): %q", got)
	}
}

// #359: the console report has the same facts as the file, laid out for a glance.
func TestReportTerm(t *testing.T) {
	now := time.Now()
	p := &benchProfile{Date: now, Axyn: "v1", Opencode: "1", Models: []string{"opencode/forte", "opencode/fraco"},
		Results: []benchResult{
			{Model: "opencode/forte", Task: "plan-001", Role: roleplan, Date: now, Runs: []benchRun{{Score: 100, Pass: true}}},
			{Model: "opencode/fraco", Task: "plan-001", Role: roleplan, Date: now, Runs: []benchRun{{Score: 0, Notes: []string{"o plano não foi gravado"}}}},
			{Model: "opencode/forte", Task: "code-001", Role: roleCode, Runs: []benchRun{{Score: 80, Pass: true}}},
		}}
	p.Routing, p.Contained = route(p.Results, 50)
	got := reportTerm(p, "x.md", false)
	for _, want := range []string{"forte", "100", "80*", "* de uma rodada anterior", "plano     forte", "no modo guiado: fraco", "✘ fraco", "o plano não foi gravado", "x.md"} {
		if !strings.Contains(got, want) {
			t.Errorf("falta %q em:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Error("fora do terminal, sem cores")
	}
	if !strings.Contains(reportTerm(p, "", true), "\x1b[1;92m") {
		t.Error("no terminal, a nota forte sai em verde")
	}
}
