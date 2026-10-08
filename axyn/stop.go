package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"
)

// axyn stop (#363) ends what the axyn is doing in this folder, even in the background: the
// worker and the models it called (the whole process tree). Nothing is lost: a run keeps
// the half-done code for axyn run --resume, and a bench keeps every result already measured.
func runStopCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn stop", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "raiz do repositório")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	abs, _ := filepath.Abs(*dir)
	var st *runState
	var err error
	if fs.Arg(0) != "" {
		st, err = loadRun(abs, fs.Arg(0))
	} else if st = busy(abs); st == nil {
		err = fmt.Errorf("nada rodando")
	}
	if err != nil || !alive(st) {
		_, _ = fmt.Fprintln(stdout, "nada rodando nesta pasta; para ver a última execução: axyn status")
		return exitOK
	}
	if st.PID != 0 {
		if err := killTree(st.PID); err != nil {
			_, _ = fmt.Fprintf(stderr, "axyn stop: não consegui parar o processo %d: %v\n", st.PID, err)
			return exitFail
		}
		for i := 0; i < 50 && processAlive(st.PID); i++ {
			time.Sleep(100 * time.Millisecond)
		}
	}
	msg := "parada por você (axyn stop); o código da tentativa em andamento fica guardado. Para continuar de onde parou: axyn run --resume"
	if st.Opts.Bench != nil {
		msg = "avaliação parada por você (axyn stop); o que já foi avaliado ficou salvo. Para ver: axyn bench --show; para avaliar o resto: axyn bench"
	}
	st.Status, st.Message = runStopped, msg
	if err := saveRun(abs, st); err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn stop: %v\n", err)
		return exitFail
	}
	_, _ = fmt.Fprintf(stdout, "parei a execução %s (%s).\n%s\n", st.ID, requestLabel(st), msg)
	return exitOK
}
