package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
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
	keys := enterKeys(tty) // Enter toggles between the progress and the live log (#356)
	var follow *logFollow
	paused := false // p + Enter: nothing is written, so the window can be scrolled (#360)
	lastLive := ""
	var lastTitle, lastDraw time.Time
	if classicConsole && tty {
		defer setTitle("Windows PowerShell")
	}
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
				_, _ = fmt.Fprintf(out, "acompanhando a execução %s: %s (Ctrl + C para sair; o axyn continua trabalhando)\n", st.ID, requestLabel(st))
				shownID = st.ID
			}
			if paused && alive(st) {
				// nothing is written while paused; the log is read later, from where it stopped
			} else if follow != nil {
				for _, l := range follow.lines() {
					clear()
					_, _ = fmt.Fprintln(out, l)
				}
			} else if line := watchLine(st); line != last {
				clear()
				_, _ = fmt.Fprintf(out, "%s  %s\n", time.Now().Format("15:04:05"), line)
				last = line
			}
			if !alive(st) {
				clear()
				if p, err := loadProfile(); err == nil && st.Status == runDone && st.Opts.Bench != nil && tty {
					_, _ = fmt.Fprint(out, "\a"+reportTerm(p, benchReportPath(p), true))
				} else {
					_, _ = fmt.Fprintf(out, "\a\n%s\n", colorStatus(renderStatus(st)))
				}
				switch {
				case st.Status == runDone && st.Opts.Bench != nil:
					notify("axyn: avaliação concluída", "O axyn já usa o melhor modelo de cada etapa; o resumo está no terminal.")
					return exitOK
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
		select {
		case k := <-keys:
			clear()
			switch {
			case k == "p" && !paused:
				paused = true
				_, _ = fmt.Fprintln(out, cGray+"── pausado: role a tela à vontade; a avaliação continua (Enter volta) ──"+cReset)
			case paused:
				paused = false
				_, _ = fmt.Fprintln(out, cGray+"── continuando ──"+cReset)
			case follow == nil:
				follow = newLogFollow(dir, 15)
				_, _ = fmt.Fprintln(out, cGray+"── log ao vivo (Enter volta ao progresso; p + Enter pausa para rolar a tela) ──"+cReset)
			default:
				follow = nil
				last = ""
				_, _ = fmt.Fprintln(out, cGray+"── progresso (Enter mostra o log ao vivo; p + Enter pausa para rolar a tela) ──"+cReset)
			}
			next = time.Time{}
		default:
		}
		if paused {
			time.Sleep(120 * time.Millisecond)
			continue
		}
		if classicConsole {
			// The classic console jumps back to the end on every write, so a line redrawn
			// all the time would forbid scrolling (#360). The window title, which writes
			// nothing, shows the spinner and the clock every second (#364); the line is
			// redrawn in place every 30 s, and a new line starts at each change.
			now := time.Now()
			line := liveLine(st, frame, now)
			if now.Sub(lastTitle) >= time.Second {
				setTitle("axyn " + ansiRe.ReplaceAllString(line, ""))
				lastTitle = now
				frame++
			}
			if k := fmt.Sprint(st.Phase, st.Done, st.Of, st.Detail, st.Ticket, st.Attempts); k != lastLive || now.Sub(lastDraw) >= 30*time.Second {
				if k != lastLive && lastLive != "" {
					_, _ = fmt.Fprint(out, "\n")
				}
				_, _ = fmt.Fprint(out, "\r\x1b[K"+line)
				lastLive, lastDraw = k, now
			}
			time.Sleep(120 * time.Millisecond)
			continue
		}
		_, _ = fmt.Fprint(out, "\r\x1b[K"+liveLine(st, frame, time.Now()))
		frame++
		time.Sleep(120 * time.Millisecond)
	}
}

var spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// The classic Windows console (conhost, the "Windows PowerShell" window) draws with a font
// that has no braille or check marks: they show as empty boxes (#360). There, the axyn uses
// the glyphs every Windows font has; Windows Terminal and the other systems keep the rich ones.
var classicConsole = hostOS == "windows" && os.Getenv("WT_SESSION") == "" && os.Getenv("TERM_PROGRAM") == ""

var spinnerClassic = []string{"▌", "▀", "▐", "▄"}

// glyph is fancy, or plain on the classic Windows console.
func glyph(fancy, plain string) string {
	if classicConsole {
		return plain
	}
	return fancy
}

// The axyn palette: yellow and white (#348); gray only for what is secondary.
const (
	cReset  = "\x1b[0m"
	cYellow = "\x1b[1;93m" // bright yellow, bold: the spinner and the bar
	cWhite  = "\x1b[1;97m" // bright white, bold: the phase and the numbers
	cGray   = "\x1b[90m"
	cCyan   = "\x1b[96m" // the actions of an agent in the log
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
	fmt.Fprintf(&b, "%s%s%s %s%s%s", cYellow, spinnerFrame(frame), cReset, cWhite, st.Phase, cReset)
	if st.Of > 0 {
		unit := ""
		if st.Phase == "avaliando modelos" {
			unit = " tentativas"
		}
		fmt.Fprintf(&b, "  %s  %s%3d%%%s  %d/%d%s", richBar(st.Done, st.Of, 30), cWhite, 100*st.Done/st.Of, cReset, st.Done, st.Of, unit)
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

// enterKeys sends what was typed every time Enter is pressed on the console (nil off a terminal).
func enterKeys(tty bool) <-chan string {
	if !tty || !isTerminal(os.Stdin) {
		return nil
	}
	ch := make(chan string, 1)
	go func() {
		r := bufio.NewReader(os.Stdin)
		for {
			l, err := r.ReadString('\n')
			if err != nil {
				return
			}
			select {
			case ch <- strings.ToLower(strings.TrimSpace(l)):
			default:
			}
		}
	}()
	return ch
}

// logFollow reads, formatted, what was added to the newest log since the last call.
type logFollow struct {
	dir, path string
	off       int64
	partial   string
	backlog   []string // the end of the log, shown when the log view opens
	pretty    *logPretty
}

func newLogFollow(dir string, back int) *logFollow {
	f := &logFollow{dir: dir, pretty: &logPretty{color: true}}
	p, err := latestLog(dir)
	if err != nil {
		f.backlog = []string{"  " + err.Error()}
		return f
	}
	f.path = p
	b, err := os.ReadFile(p)
	if err != nil {
		return f
	}
	f.off = int64(len(b))
	var shown []string
	for _, l := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		shown = append(shown, f.pretty.lines(l)...)
	}
	if len(shown) > back {
		shown = shown[len(shown)-back:]
	}
	f.backlog = shown
	return f
}

func (f *logFollow) lines() []string {
	out := f.backlog
	f.backlog = nil
	if p, err := latestLog(f.dir); err == nil && p != f.path {
		f.path, f.off, f.partial, f.pretty = p, 0, "", &logPretty{color: true} // a newer log started (the next step)
	}
	if f.path == "" {
		return out
	}
	fh, err := os.Open(f.path)
	if err != nil {
		return out
	}
	defer func() { _ = fh.Close() }()
	if _, err := fh.Seek(f.off, io.SeekStart); err != nil {
		return out
	}
	b, _ := io.ReadAll(fh)
	f.off += int64(len(b))
	text := f.partial + string(b)
	i := strings.LastIndex(text, "\n")
	if i < 0 {
		f.partial = text
		return out
	}
	f.partial = text[i+1:]
	for _, l := range strings.Split(text[:i], "\n") {
		out = append(out, f.pretty.lines(l)...)
	}
	return out
}

func spinnerFrame(frame int) string {
	if classicConsole {
		return spinnerClassic[frame%len(spinnerClassic)]
	}
	return spinner[frame%len(spinner)]
}

// colorStatus colors the status and the questions for the console (#364): the state in
// green, yellow or red, labels in gray, the commands to type in yellow, files without
// tests in red, headings in white. Off a console the text stays plain.
func colorStatus(text string) string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(l, "execução "):
			code := cYellow
			switch {
			case strings.Contains(l, ": "+runDone):
				code = "\x1b[1;92m"
			case strings.Contains(l, ": "+runStopped) || strings.Contains(l, "interrompida"):
				code = "\x1b[1;91m"
			}
			if i := strings.Index(l, ": "); i > 0 {
				l = cWhite + l[:i] + cReset + ": " + code + l[i+2:] + cReset
			}
		case labelRe.MatchString(l):
			m := labelRe.FindStringSubmatch(l)
			l = cGray + m[1] + cReset + m[2]
		case strings.HasPrefix(t, "axyn ") || strings.HasPrefix(t, "$env:") || strings.HasPrefix(t, "[Environment]") || strings.HasPrefix(t, "export "):
			l = strings.Replace(l, t, cYellow+t+cReset, 1)
		case zeroCovRe.MatchString(t):
			l = "\x1b[91m" + l + cReset
		case strings.HasSuffix(t, ":") && !strings.HasPrefix(t, "-"):
			l = cWhite + l + cReset
		case strings.Contains(t, "Error") || strings.Contains(t, "reprovad") || strings.Contains(t, "falhou"):
			l = "\x1b[91m" + l + cReset
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

var (
	labelRe   = regexp.MustCompile(`^((?:pedido|ticket|fase|modelo|tentativas no ticket|portões|entregue): )(.*)$`)
	zeroCovRe = regexp.MustCompile(`^\S+\.go: 0%`)
)
