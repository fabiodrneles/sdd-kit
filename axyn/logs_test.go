package main

import (
	"fmt"
	"os"
	"os/exec"
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

// #360: the diff an agent prints after an edit folds into one line; the prose stays whole.
func TestLogPrettyFoldsDiffs(t *testing.T) {
	p := &logPretty{}
	var got []string
	prose := "I'll help you debug the CI failure. Since this involves a test failure, I need to use the systematic-debugging skill first to properly investigate it."
	for _, l := range []string{
		`← Edit slug.go`,
		`Index: C:\Users\G\AppData\Local\Temp\axyn-bench-code-001-1\slug.go`,
		"===================================================================",
		`--- C:\x\slug.go`, `+++ C:\x\slug.go`, "@@ -1,9 +1,49 @@", " package slug", "+import (", "+\t\"strings\"", "-\treturn \"\"", "",
		"Now let me verify the fixes work:", prose,
	} {
		got = append(got, p.lines(l)...)
	}
	all := strings.Join(got, "\n")
	if !strings.Contains(all, "slug.go: +2 −1 linhas") || strings.Contains(all, "import") || strings.Contains(all, "@@") {
		t.Fatalf("diff not folded:\n%s", all)
	}
	if !strings.Contains(all, "Now let me verify") || !strings.Contains(all, prose) {
		t.Fatalf("prose must stay whole:\n%s", all)
	}
}

// #360: the classic Windows console has no braille nor check marks; the axyn uses the
// glyphs every Windows font has.
func TestClassicConsoleGlyphs(t *testing.T) {
	defer func(v bool) { classicConsole = v }(classicConsole)
	classicConsole = true
	if got := prettyLine("--- nota 100, passou true, trapaça false: ok", false); !strings.Contains(got, "√ nota 100") {
		t.Errorf("score: %q", got)
	}
	if f := spinnerFrame(0); f != "▌" {
		t.Errorf("spinner: %q", f)
	}
}

// #360: each kind of line has its color, so the log is not a wall of white.
func TestPrettyLineColors(t *testing.T) {
	for line, color := range map[string]string{
		"→ Read slug.go":                cCyan,
		"$ go test ./...":               cYellow,
		"ok  \tbenchcode\t3.1s":         "\x1b[92m",
		"FAIL\tbenchfix [build failed]": "\x1b[91m",
		"**Summary of fixes**":          cWhite,
	} {
		if got := prettyLine(line, true); !strings.HasPrefix(got, color) {
			t.Errorf("%q: %q", line, got)
		}
	}
	if got := prettyLine(`go: unlinkat C:\x\b.test.exe: O arquivo já está sendo usado por outro processo.`, false); !strings.Contains(got, "não conta") {
		t.Errorf("Windows cleanup warning: %q", got)
	}
}

// #362: the bench says how many models and steps make the attempts.
func TestBenchCountIsClear(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("sem go")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("AXYN_OPENCODE", benchAgent(t))
	var o, e strings.Builder
	if code := runBenchCmd([]string{"plano", "testes", "--models", "bom,trapaceiro", "--timeout", "2m"}, &o, &e); code != exitOK {
		t.Fatalf("bench: %d %s", code, e.String())
	}
	if !strings.Contains(o.String(), "avaliando 2 modelo(s) em 2 etapa(s) (plano, testes) = 4 tentativas") {
		t.Errorf("contagem:\n%s", o.String())
	}
	st := &runState{Phase: "avaliando modelos", Done: 1, Of: 4, Detail: "modelo 1 de 2: bom, plano", Started: time.Now(), PhaseSince: time.Now()}
	if l := liveLine(st, 0, time.Now()); !strings.Contains(l, "1/4 tentativas") || !strings.Contains(l, "modelo 1 de 2") {
		t.Errorf("linha ao vivo: %q", l)
	}
}

// #362: models that failed the same steps for the same reason share one line.
func TestReportTermGroupsFailures(t *testing.T) {
	now := time.Now()
	p := &benchProfile{Date: now, Models: []string{"openrouter/a", "openrouter/b", "openrouter/c", "openrouter/d"}}
	for i, m := range p.Models {
		p.Results = append(p.Results, benchResult{Model: m, Task: "code-001", Role: roleCode, Date: now,
			Runs: []benchRun{{Notes: []string{fmt.Sprintf("o CI não passou: FAIL benchcode %d.%03ds", i, i)}}}})
	}
	p.Routing, p.Contained = route(p.Results, 50)
	got := reportTerm(p, "", false)
	if !strings.Contains(got, "4 modelos: openrouter/a, openrouter/b, openrouter/c +1") || strings.Count(got, "o CI não passou") != 1 {
		t.Errorf("falhas iguais numa linha:\n%s", got)
	}
}
