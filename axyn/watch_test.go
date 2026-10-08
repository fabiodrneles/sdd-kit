package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// #323: the watch prints one line per change, nothing in between, and notifies at the end.
func TestWatchPrintsChangesAndNotifies(t *testing.T) {
	dir := repo(t)
	var got []string
	old := notify
	notify = func(title, msg string) { got = append(got, title) }
	t.Cleanup(func() { notify = old })

	st := &runState{ID: "20260101-000001", Request: "algo", Status: runRunning, Phase: "planejando", Started: time.Now()}
	if err := saveRun(dir, st); err != nil {
		t.Fatal(err)
	}
	go func() {
		for _, step := range []func(){
			func() { st.Phase = "código"; st.Ticket, st.Total, st.Title = 1, 1, "Um" },
			func() {}, // nothing changes: no line
			func() { st.Phase = "portões"; st.Attempts = 1 },
			func() { st.Status, st.Phase = runDone, "fim" },
		} {
			time.Sleep(30 * time.Millisecond)
			step()
			_ = saveRun(dir, st)
		}
	}()
	var out bytes.Buffer
	if code := watchRun(dir, "", 10*time.Millisecond, &out); code != exitOK {
		t.Fatalf("code %d\n%s", code, out.String())
	}
	text := out.String()
	for _, want := range []string{"fase: planejando", "ticket 1 de 1 «Um» · fase: código", "fase: portões · tentativa 1", "concluído"} {
		if !strings.Contains(text, want) {
			t.Errorf("faltou %q:\n%s", want, text)
		}
	}
	if n := strings.Count(text, "fase: código"); n != 1 {
		t.Errorf("a mesma linha não pode repetir (%d vezes):\n%s", n, text)
	}
	if len(got) != 1 || got[0] != "axyn: pronto" {
		t.Errorf("notificação no fim: %v", got)
	}
}

func TestWatchStoppedWithQuestion(t *testing.T) {
	dir := repo(t)
	var got []string
	old := notify
	notify = func(title, msg string) { got = append(got, title) }
	t.Cleanup(func() { notify = old })
	st := &runState{ID: "20260101-000002", Status: runStopped, Phase: "portões", Started: time.Now(),
		Message: "O axyn precisa de uma resposta sua para seguir:\npergunta ao usuário (x): qual?"}
	if err := saveRun(dir, st); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := watchRun(dir, "", time.Millisecond, &out); code == exitOK || !strings.Contains(out.String(), "qual?") {
		t.Errorf("parado com pergunta: %d\n%s", code, out.String())
	}
	if len(got) != 1 || got[0] != "axyn: precisa de você" {
		t.Errorf("notificação da pergunta: %v", got)
	}
}

// #348: the live line has the spinner, the elapsed time, the bar, the count, the estimate
// and the detail, in the axyn colors.
func TestLiveLine(t *testing.T) {
	now := time.Now()
	st := &runState{Phase: "avaliando modelos", PhaseSince: now.Add(-10 * time.Minute), Done: 4, Of: 16, Detail: "modelo-x, testes"}
	got := liveLine(st, 0, now)
	for _, want := range []string{"⠋", "avaliando modelos", "10:00", "━", "╸", " 25%", "4/16", "faltam ~30:00", "modelo-x, testes", cYellow, cWhite, "38;5;220"} {
		if !strings.Contains(got, want) {
			t.Errorf("a linha não traz %q: %q", want, got)
		}
	}
	if got := liveLine(&runState{Phase: "código", Started: now.Add(-time.Minute), Ticket: 1, Total: 3, Attempts: 2}, 3, now); !strings.Contains(got, "ticket 1 de 3, tentativa 3") || strings.Contains(got, "faltam") {
		t.Errorf("sem progresso conhecido, só o tempo e o ticket: %q", got)
	}
	if clock(65*time.Minute+5*time.Second) != "1:05:05" {
		t.Error("relógio com horas")
	}
}

func TestLiveLineDemo(t *testing.T) {
	if os.Getenv("AXYN_DEMO") == "" {
		t.Skip()
	}
	now := time.Now()
	st := &runState{Phase: "avaliando modelos", PhaseSince: now.Add(-12*time.Minute - 5*time.Second), Done: 5, Of: 16, Detail: "nemotron, testes"}
	for f := 0; f < 3; f++ {
		fmt.Println(liveLine(st, f, now))
	}
}

// #356: the files people open by hand start with the UTF-8 mark, once.
func TestUTF8Mark(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "r.md")
	if err := writeText(p, utf8BOM+"ação"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != utf8BOM+"ação" {
		t.Errorf("uma marca só: %q", b)
	}
	l := filepath.Join(dir, "x.log")
	for i := 0; i < 2; i++ {
		f, err := openLog(l)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.WriteString("linha\n")
		_ = f.Close()
	}
	if b, _ := os.ReadFile(l); string(b) != utf8BOM+"linha\nlinha\n" {
		t.Errorf("o log ganha a marca só quando é novo: %q", b)
	}
}
