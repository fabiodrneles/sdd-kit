package main

import (
	"fmt"
	"io"
	"os"
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
// On a terminal, a live line (spinner, elapsed time, progress bar and estimate) is redrawn
// every fraction of a second while something is running (#348); in a file or a pipe only
// the change lines are written.
func watchRun(dir, id string, interval time.Duration, out io.Writer) int {
	tty := isTerminal(out)
	if tty {
		enableVT()
	}
	last := ""
	shownID := ""
	var st *runState
	next := time.Time{}
	frame := 0
	clear := func() {
		if tty {
			_, _ = fmt.Fprint(out, "\r\x1b[K")
		}
	}
	for {
		if now := time.Now(); !now.Before(next) {
			next = now.Add(interval)
			var err error
			if st, err = loadRun(dir, id); err != nil {
				clear()
				_, _ = fmt.Fprintf(out, "axyn: %v\n", err)
				return exitFail
			}
			if shownID != st.ID {
				clear()
				_, _ = fmt.Fprintf(out, "acompanhando a execução %s: %s (Ctrl + C para sair; o axyn continua trabalhando)\n", st.ID, oneLine(st.Request))
				shownID = st.ID
			}
			if line := watchLine(st); line != last {
				clear()
				_, _ = fmt.Fprintf(out, "%s  %s\n", time.Now().Format("15:04:05"), line)
				last = line
			}
			if !alive(st) {
				clear()
				_, _ = fmt.Fprintf(out, "\a\n%s\n", renderStatus(st))
				switch {
				case st.Status == runDone:
					notify("axyn: pronto", "Os tickets foram entregues: "+fmt.Sprint(len(st.Delivered))+" PR(s) para revisar.")
					return exitOK
				case strings.Contains(st.Message, "precisa de uma resposta sua") || strings.Contains(st.Message, "e precisa de você") || strings.Contains(st.Message, "axyn coverage auto"):
					notify("axyn: precisa de você", "O axyn tem uma pergunta: abra o opencode ou rode axyn status.")
				default:
					notify("axyn: parou", firstLine(renderStatus(st)))
				}
				return exitFail
			}
		}
		if !tty {
			time.Sleep(interval)
			continue
		}
		_, _ = fmt.Fprint(out, "\r\x1b[K"+liveLine(st, frame, time.Now()))
		frame++
		time.Sleep(120 * time.Millisecond)
	}
}

var spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// The axyn palette: yellow and white (#348); gray only for what is secondary.
const (
	cReset  = "\x1b[0m"
	cYellow = "\x1b[1;93m" // bright yellow, bold: the spinner and the bar
	cWhite  = "\x1b[1;97m" // bright white, bold: the phase and the numbers
	cGray   = "\x1b[90m"
)

func clock(d time.Duration) string {
	d = d.Round(time.Second)
	h, m, s := int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// liveLine is the animated line, in the style the community knows: the "dots" spinner of
// cli-spinners (npm, pnpm, Vercel) and the progress bar of Python's Rich (pip), in the
// axyn colors: yellow fading to white on the filled part, dim gray on the rest.
func liveLine(st *runState, frame int, now time.Time) string {
	since := st.PhaseSince
	if since.IsZero() {
		since = st.Started
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s%s%s %s%s%s", cYellow, spinner[frame%len(spinner)], cReset, cWhite, st.Phase, cReset)
	if st.Of > 0 {
		fmt.Fprintf(&b, "  %s  %s%3d%%%s  %d/%d", richBar(st.Done, st.Of, 30), cWhite, 100*st.Done/st.Of, cReset, st.Done, st.Of)
	}
	fmt.Fprintf(&b, "  %s%s%s", cYellow, clock(now.Sub(since)), cReset)
	if st.Of > 0 && st.Done > 0 && st.Done < st.Of {
		left := time.Duration(float64(now.Sub(since)) / float64(st.Done) * float64(st.Of-st.Done))
		fmt.Fprintf(&b, "  %sfaltam ~%s%s", cWhite, clock(left), cReset)
	}
	detail := st.Detail
	if detail == "" && st.Ticket > 0 {
		detail = fmt.Sprintf("ticket %d de %d, tentativa %d", st.Ticket, st.Total, st.Attempts+1)
	}
	if detail != "" {
		fmt.Fprintf(&b, "  %s· %s%s", cGray, detail, cReset)
	}
	return b.String()
}

// gradient goes from the axyn yellow to white (256-color codes).
var gradient = []int{220, 221, 222, 223, 229, 230, 231}

// richBar is Rich's bar: ━ filled with a ╸ tip, ━ dim for the rest.
func richBar(done, of, width int) string {
	fill := done * width / of
	var b strings.Builder
	for i := 0; i < fill; i++ {
		fmt.Fprintf(&b, "\x1b[38;5;%dm━", gradient[i*len(gradient)/width])
	}
	rest := width - fill
	if rest > 0 && fill > 0 {
		fmt.Fprintf(&b, "\x1b[38;5;%dm╸", gradient[len(gradient)-1])
		rest--
	}
	fmt.Fprintf(&b, "%s%s%s", cGray, strings.Repeat("━", rest), cReset)
	return b.String()
}

// isTerminal says whether out is a console (the live line would garble a file).
func isTerminal(out io.Writer) bool {
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
