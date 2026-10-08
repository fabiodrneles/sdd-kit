//go:build !windows

package main

import (
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// #363: axyn stop ends a run in the background, with what it started, and says how to go on.
func TestStopEndsTheBackgroundRun(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("sh", "-c", "sleep 60 & sleep 60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = cmd.Wait() }()
	now := time.Now()
	st := &runState{ID: "1", Request: "avaliação dos modelos", Status: runRunning, PID: cmd.Process.Pid, Started: now, Opts: runOpts{Bench: &benchSpec{}}}
	if err := saveRun(dir, st); err != nil {
		t.Fatal(err)
	}
	var o, e strings.Builder
	if code := runStopCmd([]string{"--dir", dir}, &o, &e); code != exitOK {
		t.Fatalf("stop: %d %s", code, e.String())
	}
	time.Sleep(200 * time.Millisecond)
	if processAlive(cmd.Process.Pid) {
		t.Error("o processo continua vivo")
	}
	got, _ := loadRun(dir, "")
	if got.Status != runStopped || !strings.Contains(got.Message, "axyn bench") || !strings.Contains(o.String(), "parei a execução 1") {
		t.Errorf("estado: %+v\n%s", got, o.String())
	}
	o.Reset()
	if runStopCmd([]string{"--dir", dir}, &o, &e); !strings.Contains(o.String(), "nada rodando") {
		t.Errorf("de novo: %s", o.String())
	}
}

// #363: a model bench after the work does not hide it: status, history, decide and
// --resume are about the project's last run.
func TestBenchDoesNotHideTheWork(t *testing.T) {
	dir := t.TempDir()
	work := &runState{ID: "20261007-100000", Request: "notas apagar", Status: runStopped, Message: "o ticket 2 precisa de uma resposta sua", Started: time.Now()}
	bench := &runState{ID: "20261007-230000", Request: "avaliação dos modelos", Status: runDone, Started: time.Now(), Opts: runOpts{Bench: &benchSpec{}}}
	for _, st := range []*runState{work, bench} {
		if err := saveRun(dir, st); err != nil {
			t.Fatal(err)
		}
	}
	if st, _ := loadWork(dir, ""); st.ID != work.ID {
		t.Errorf("loadWork: %s", st.ID)
	}
	if st, _ := shownRun(dir, ""); st.ID != work.ID {
		t.Errorf("status depois da avaliação mostra o trabalho: %s", st.ID)
	}
	bench.Status = runRunning
	bench.PID = 0
	bench.Updated = time.Now()
	_ = saveRun(dir, bench)
	if st, _ := shownRun(dir, ""); st.ID != bench.ID {
		t.Errorf("com a avaliação rodando, o status mostra a avaliação: %s", st.ID)
	}
}
