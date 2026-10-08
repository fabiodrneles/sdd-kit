//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
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
	st := &runState{ID: "bench-1", Request: "avaliação dos modelos", Status: runRunning, PID: cmd.Process.Pid, Started: now, Opts: runOpts{Bench: &benchSpec{}}}
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
	got, _ := loadRun(dir, "bench-1")
	if got.Status != runStopped || !strings.Contains(got.Message, "axyn bench") || !strings.Contains(o.String(), "parei a execução bench-1") {
		t.Errorf("estado: %+v\n%s", got, o.String())
	}
	o.Reset()
	if runStopCmd([]string{"--dir", dir}, &o, &e); !strings.Contains(o.String(), "nada rodando") {
		t.Errorf("de novo: %s", o.String())
	}
}

// #363: the work and the benches keep separate lists: a bench after the work does not
// hide it from status, history, decide and --resume.
func TestBenchHasItsOwnList(t *testing.T) {
	dir := t.TempDir()
	work := &runState{ID: "20261007-100000", Request: "notas apagar", Status: runStopped, Message: "o ticket 2 precisa de uma resposta sua", Started: time.Now()}
	legacy := &runState{ID: "20261007-200000", Request: "avaliação dos modelos", Status: runDone, Started: time.Now(), Opts: runOpts{Bench: &benchSpec{}}}
	bench := &runState{ID: "bench-20261007-230000", Request: "avaliação dos modelos", Status: runDone, Started: time.Now(), Opts: runOpts{Bench: &benchSpec{}}}
	for _, st := range []*runState{work, legacy, bench} {
		if err := saveRun(dir, st); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(benchRunsDir(dir), bench.ID+".json")); err != nil {
		t.Errorf("a avaliação grava na lista dela: %v", err)
	}
	if st, _ := loadRun(dir, ""); st.ID != work.ID {
		t.Errorf("o trabalho do projeto continua sendo o último: %s", st.ID)
	}
	if b, _ := loadBenchRun(dir); b.ID != bench.ID {
		t.Errorf("última avaliação: %s", b.ID)
	}
	if busy(dir) != nil {
		t.Error("nada rodando")
	}
	bench.Status, bench.PID = runRunning, 0
	_ = saveRun(dir, bench)
	if b := busy(dir); b == nil || b.ID != bench.ID {
		t.Errorf("a avaliação rodando ocupa a pasta: %+v", b)
	}
}
