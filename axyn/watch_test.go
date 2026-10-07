package main

import (
	"bytes"
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
