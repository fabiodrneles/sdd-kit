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
	passRe    = regexp.MustCompile(`^(ok\s|PASS$|--- PASS|=== RUN)`)
	failRe    = regexp.MustCompile(`^(FAIL|--- FAIL|panic:|# \S+ \[)|\.go:\d+:\d+: `)
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
		return "\n" + c(cYellow, glyph("▶", "►")+" "+m[1]) + c(cWhite, " · "+task)
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
			return c(cYellow, "  "+glyph("⏸", "■")+" "+shorten(m[4], 160)+" (não conta como nota)")
		case m[3] == "true":
			return c("\x1b[1;91m", "  "+glyph("✘", "×")+" trapaça (veto)"+why)
		case m[2] == "true":
			return c("\x1b[1;92m", "  "+glyph("✔", "√")+" nota "+m[1]+why)
		}
		return c("\x1b[1;91m", "  "+glyph("✘", "×")+" nota "+m[1]+why)
	case strings.HasPrefix(t, "Error:") || strings.HasPrefix(t, "error:"):
		return c("\x1b[91m", "  "+glyph("⚠", "!")+" "+shorten(t, 300))
	case strings.HasPrefix(t, "> "):
		return c(cGray, "  "+t)
	case strings.HasPrefix(t, "→ ") || strings.HasPrefix(t, "← ") || strings.HasPrefix(t, "✱ ") || strings.HasPrefix(t, "⚙ ") || strings.HasPrefix(t, "◇ "):
		if i := strings.IndexAny(t, "{["); len([]rune(t)) > 160 && i > 0 && i < 60 {
			return c(cCyan, "  "+t[:i]+"{…}") + c(cGray, fmt.Sprintf(" (%s)", sizeText(len(t))))
		}
		return c(cCyan, "  "+t) // an action of the agent: read, edit, search, tool
	case strings.HasPrefix(t, "$ "):
		return c(cYellow, "  "+t) // a command the agent ran
	case passRe.MatchString(t):
		return c("\x1b[92m", "  "+t)
	case failRe.MatchString(t):
		return c("\x1b[91m", "  "+t)
	case cleanupErr.MatchString(t):
		return c(cGray, "  "+shorten(t, 120)+" (aviso do Windows; não conta)")
	case strings.HasPrefix(t, "**") || strings.HasPrefix(t, "#"):
		return c(cWhite, "  "+strings.Trim(t, "*# "))
	case strings.HasPrefix(t, "|") || strings.HasPrefix(t, "Mode ") || strings.HasPrefix(t, "-a---") || strings.HasPrefix(t, "----"):
		return c(cGray, "  "+t) // tables and listings
	}
	// Long tool calls (JSON) become one line with their size; the models' prose stays
	// whole (#360), unless it is huge.
	if i := strings.IndexAny(t, "{["); len([]rune(t)) > 160 && i > 0 && i < 60 {
		return "  " + t[:i] + "{…}" + c(cGray, fmt.Sprintf(" (%s)", sizeText(len(t))))
	}
	if len([]rune(t)) > 1000 {
		return "  " + shorten(t, 900) + c(cGray, fmt.Sprintf(" (%s)", sizeText(len(t))))
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
	pretty := &logPretty{color: color}
	show := func(l string) {
		if *raw {
			_, _ = fmt.Fprintln(stdout, strings.TrimPrefix(l, utf8BOM))
			return
		}
		for _, p := range pretty.lines(l) {
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

// logPretty formats a log line by line, and folds the diffs the agents print after each
// edit into one line per file (#360): "✎ slug.go: +47 −1 linhas". axyn logs --raw keeps them.
type logPretty struct {
	color    bool
	inDiff   bool
	file     string
	add, del int
}

func (p *logPretty) lines(raw string) []string {
	l := strings.TrimPrefix(ansiRe.ReplaceAllString(strings.TrimRight(raw, "\r"), ""), utf8BOM)
	var out []string
	if p.inDiff {
		switch {
		case strings.HasPrefix(l, "===") || strings.HasPrefix(l, "--- ") || strings.HasPrefix(l, "+++ ") || strings.HasPrefix(l, "@@") || strings.HasPrefix(l, "\\"):
			return nil
		case strings.HasPrefix(l, "+"):
			p.add++
			return nil
		case strings.HasPrefix(l, "-"):
			p.del++
			return nil
		case l == "" || strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t"):
			return nil
		}
		out = append(out, p.summary())
		p.inDiff = false
	}
	if strings.HasPrefix(l, "Index: ") {
		p.inDiff, p.file, p.add, p.del = true, filepath.Base(strings.ReplaceAll(strings.TrimSpace(l[len("Index: "):]), "\\", "/")), 0, 0
		return out
	}
	if s := prettyLine(l, p.color); s != "" {
		out = append(out, s)
	}
	return out
}

func (p *logPretty) summary() string {
	s := fmt.Sprintf("  %s %s: +%d −%d linhas", glyph("✎", "»"), p.file, p.add, p.del)
	if p.color {
		return cGray + s + cReset
	}
	return s
}
