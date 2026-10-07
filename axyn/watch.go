package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// axyn status --watch is the sentinel (#323): it prints a line only when the run changes
// and, when it ends, stops or asks something, beeps and shows a notification, so nobody
// has to run axyn status by hand.

// notify shows a desktop notification; replaced in tests.
var notify = func(title, msg string) {
	msg = strings.ReplaceAll(firstLine(msg), "'", "")
	var cmd *exec.Cmd
	switch hostOS {
	case "windows":
		script := "Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; " +
			"$n = New-Object System.Windows.Forms.NotifyIcon; $n.Icon = [System.Drawing.SystemIcons]::Information; " +
			"$n.Visible = $true; $n.ShowBalloonTip(15000, '" + title + "', '" + msg + "', 'Info'); Start-Sleep 15; $n.Dispose()"
		// No -WindowStyle Hidden: the child shares the user's console, and hiding it hid the
		// user's terminal. hideWindow gives the child no console at all.
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
		hideWindow(cmd)
	case "darwin":
		cmd = exec.Command("osascript", "-e", fmt.Sprintf("display notification %q with title %q", msg, title))
	default:
		if _, err := lookPath("notify-send"); err != nil {
			return
		}
		cmd = exec.Command("notify-send", title, msg)
	}
	_ = cmd.Start() // never blocks the watch
}

func watchLine(st *runState) string {
	parts := []string{}
	if st.Ticket > 0 {
		parts = append(parts, fmt.Sprintf("ticket %d de %d «%s»", st.Ticket, st.Total, st.Title))
	}
	parts = append(parts, "fase: "+st.Phase)
	if st.Attempts > 0 {
		parts = append(parts, fmt.Sprintf("tentativa %d", st.Attempts))
	}
	if st.Gate != "" {
		parts = append(parts, "portões: "+st.Gate)
	}
	return strings.Join(parts, " · ")
}

// watchRun follows a run until it is no longer running. 0 when everything was delivered.
func watchRun(dir, id string, interval time.Duration, out io.Writer) int {
	last := ""
	shownID := ""
	for {
		st, err := loadRun(dir, id)
		if err != nil {
			_, _ = fmt.Fprintf(out, "axyn: %v\n", err)
			return exitFail
		}
		if shownID != st.ID {
			_, _ = fmt.Fprintf(out, "acompanhando a execução %s: %s (Ctrl + C para sair; o axyn continua trabalhando)\n", st.ID, oneLine(st.Request))
			shownID = st.ID
		}
		if line := watchLine(st); line != last {
			_, _ = fmt.Fprintf(out, "%s  %s\n", time.Now().Format("15:04:05"), line)
			last = line
		}
		if !alive(st) {
			_, _ = fmt.Fprintf(out, "\a\n%s\n", renderStatus(st))
			switch {
			case st.Status == runDone:
				notify("axyn: pronto", "Os tickets foram entregues: "+fmt.Sprint(len(st.Delivered))+" PR(s) para revisar.")
				return exitOK
			case strings.Contains(st.Message, "precisa de uma resposta sua") || strings.Contains(st.Message, "e precisa de você"):
				notify("axyn: precisa de você", "O axyn tem uma pergunta: abra o opencode ou rode axyn status.")
			default:
				notify("axyn: parou", firstLine(renderStatus(st)))
			}
			return exitFail
		}
		time.Sleep(interval)
	}
}
