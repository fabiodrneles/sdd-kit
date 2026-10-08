package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// axyn logs (#356): follow what is happening, in a readable form. It picks the most recent
// log (a run of this project or a model bench), formats it and keeps following, like
// tail -f: a colored header per attempt, short action lines, errors in red, scores
// highlighted, and the huge JSON of tool calls cut to one line with its size.

var (
	ansiRe    = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	attemptRe = regexp.MustCompile(`^=== (\S+) em (\S+)`)
	scoreRe   = regexp.MustCompile(`^--- nota (\d+), passou (\w+), trapaça (\w+)(?::\s*(.*))?$`)
	agentRe   = regexp.MustCompile(`^== \S+ (axyn-[\w-]+)`)
)

var taskNames = map[string]string{"plan-001": "plano", "code-001": "código", "tests-001": "testes", "fix-001": "conserto"}

// latestLog is the newest of this project's run logs and the bench logs.
func latestLog(dir string) (string, error) {
	var all []string
	for _, pat := range []string{filepath.Join(runsDir(dir), "*.log"), filepath.Join(benchDir(), "bench-*.log")} {
		m, _ := filepath.Glob(pat)
		all = append(all, m...)
	}
	if len(all) == 0 {
		return "", fmt.Errorf("nenhum log ainda: rode axyn run ou axyn bench")
	}
	sort.Slice(all, func(i, j int) bool {
		a, _ := os.Stat(all[i])
		b, _ := os.Stat(all[j])
		return a.ModTime().After(b.ModTime())
	})
	return all[0], nil
}

// prettyLine turns one raw log line into what a person reads ("" hides it).
func prettyLine(l string, color bool) string {
	l = strings.TrimPrefix(ansiRe.ReplaceAllString(strings.TrimRight(l, "\r"), ""), utf8BOM)
	c := func(code, s string) string {
		if !color {
			return s
		}
		return code + s + cReset
	}
	t := strings.TrimSpace(l)
	switch {
	case t == "":
		return ""
	case attemptRe.MatchString(t):
		m := attemptRe.FindStringSubmatch(t)
		task := taskNames[m[2]]
		if task == "" {
			task = m[2]
		}
		return "\n" + c(cYellow, "▶ "+m[1]) + c(cWhite, " · "+task)
	case agentRe.MatchString(t):
		return c(cGray, "  agente "+agentRe.FindStringSubmatch(t)[1])
	case strings.HasPrefix(t, "== "):
		return ""
	case scoreRe.MatchString(t):
		m := scoreRe.FindStringSubmatch(t)
		why := ""
		if m[4] != "" {
			why = " · " + shorten(m[4], 160)
		}
		switch {
		case strings.HasPrefix(m[4], "indisponível"):
			return c(cYellow, "  ⏸ "+shorten(m[4], 160)+" (não conta como nota)")
		case m[3] == "true":
			return c("\x1b[1;91m", "  ✘ trapaça (veto)"+why)
		case m[2] == "true":
			return c("\x1b[1;92m", "  ✔ nota "+m[1]+why)
		}
		return c("\x1b[1;91m", "  ✘ nota "+m[1]+why)
	case strings.HasPrefix(t, "Error:") || strings.HasPrefix(t, "error:"):
		return c("\x1b[91m", "  ⚠ "+shorten(t, 200))
	case strings.HasPrefix(t, "> "):
		return c(cGray, "  "+t)
	}
	if len([]rune(t)) > 160 {
		head := t
		if i := strings.IndexAny(t, "{["); i > 0 && i < 60 {
			head = t[:i] + "{…}"
		} else {
			head = shorten(t, 140)
		}
		return "  " + head + c(cGray, fmt.Sprintf(" (%s)", sizeText(len(t))))
	}
	return "  " + t
}

func sizeText(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%d caracteres", n)
	}
	return fmt.Sprintf("%.1f KB", float64(n)/1024)
}

func runLogsCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn logs", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "raiz do repositório (para os logs das execuções)")
	raw := fs.Bool("raw", false, "o texto completo, sem formatação")
	once := fs.Bool("once", false, "mostra e sai, sem continuar acompanhando")
	lines := fs.Int("n", 40, "quantas linhas do fim mostrar antes de acompanhar")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	path := fs.Arg(0)
	if path == "" {
		p, err := latestLog(*dir)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "axyn logs: %v\n", err)
			return exitFail
		}
		path = p
	}
	f, err := os.Open(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn logs: %v\n", err)
		return exitFail
	}
	defer func() { _ = f.Close() }()
	color := isTerminal(stdout)
	if color {
		enableVT()
	}
	_, _ = fmt.Fprintf(stdout, "log: %s%s\n", path, map[bool]string{true: "", false: " (Ctrl + C para sair; o trabalho continua)"}[*once])
	show := func(l string) {
		if *raw {
			_, _ = fmt.Fprintln(stdout, strings.TrimPrefix(l, utf8BOM))
			return
		}
		if p := prettyLine(l, color); p != "" {
			_, _ = fmt.Fprintln(stdout, p)
		}
	}
	r := bufio.NewReader(f)
	var tail []string
	for {
		l, err := r.ReadString('\n')
		if l != "" && err == nil {
			tail = append(tail, strings.TrimSuffix(l, "\n"))
			if len(tail) > *lines {
				tail = tail[1:]
			}
		}
		if err != nil {
			for _, t := range tail {
				show(t)
			}
			if l != "" {
				show(l)
			}
			break
		}
	}
	if *once {
		return exitOK
	}
	partial := ""
	for {
		l, err := r.ReadString('\n')
		if err != nil {
			partial += l
			time.Sleep(500 * time.Millisecond)
			continue
		}
		show(strings.TrimSuffix(partial+l, "\n"))
		partial = ""
	}
}
