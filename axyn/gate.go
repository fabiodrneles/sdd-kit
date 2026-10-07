package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

const (
	exitGateFailed = 1

	defaultMaxLines = 400
	defaultCICmd    = "make ci"
)

// defaultProtected are the paths a ticket diff may not touch (spec 021 FR-5). The
// CI definition and the linters' configs are here too: a model that edits the
// Makefile could make "make ci" pass without running the tests.
var defaultProtected = []string{
	".github/workflows",
	"Makefile",
	".golangci.yml",
	".golangci.yaml",
	".eslintrc*",
	"eslint.config.*",
	".markdownlint*",
	"lychee.toml",
	"specs",
	".axyn",
	"axyn.yaml",
	".sdd-release",
	"plugin.json",
}

// fileDiff is what changed in one file, counted from the unified diff.
type fileDiff struct {
	path    string
	deleted bool
	added   []string
	removed []string
}

// finding is one reason a diff failed a gate.
type finding struct {
	gate, reason string
}

var (
	testFileRe = regexp.MustCompile(`(_test\.go|(^|/)test_[^/]*\.py|_test\.py|\.(test|spec)\.[jt]sx?|Tests?\.java|(^|/)tests?/[^/]+\.(sh|bats|py|js|ts))$`)
	testDeclRe = regexp.MustCompile(`^\s*(func Test\w+\(|def test_\w+\(|(it|test|describe)\(|@Test\b)`)
	assertRe   = regexp.MustCompile(`\b(assert\w*|expect|require\.\w+|t\.(Error|Errorf|Fatal|Fatalf|Fail|FailNow))\b`)
	skipRe     = regexp.MustCompile(`(t\.Skip\w*\(|pytest\.mark\.(skip|xfail)|pytest\.skip\(|\b(it|test|describe)\.skip\b|\bx(it|describe)\(|@Disabled|@Ignore)`)
	diffFileRe = regexp.MustCompile(`^diff --git a/(.*) b/(.*)$`)
)

// parseDiff reads `git diff` output into per-file added and removed lines.
func parseDiff(r io.Reader) []fileDiff {
	var files []fileDiff
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		line := sc.Text()
		if m := diffFileRe.FindStringSubmatch(line); m != nil {
			files = append(files, fileDiff{path: m[2]})
			continue
		}
		if len(files) == 0 {
			continue
		}
		f := &files[len(files)-1]
		switch {
		case strings.HasPrefix(line, "deleted file mode"):
			f.deleted = true
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		case strings.HasPrefix(line, "+"):
			f.added = append(f.added, line[1:])
		case strings.HasPrefix(line, "-"):
			f.removed = append(f.removed, line[1:])
		}
	}
	return files
}

func countMatches(re *regexp.Regexp, lines []string) int {
	n := 0
	for _, l := range lines {
		if re.MatchString(l) {
			n++
		}
	}
	return n
}

// checkLoosenedTests reports deleted test files, fewer tests, fewer assertions
// and new skips in test files.
func checkLoosenedTests(files []fileDiff) []finding {
	var out []finding
	for _, f := range files {
		if !testFileRe.MatchString(f.path) {
			continue
		}
		if f.deleted {
			out = append(out, finding{"tests", "arquivo de teste apagado: " + f.path})
			continue
		}
		if r, a := countMatches(testDeclRe, f.removed), countMatches(testDeclRe, f.added); r > a {
			out = append(out, finding{"tests", fmt.Sprintf("%s: %d teste(s) a menos", f.path, r-a)})
		}
		if r, a := countMatches(assertRe, f.removed), countMatches(assertRe, f.added); r > a {
			out = append(out, finding{"tests", fmt.Sprintf("%s: %d asserção(ões) a menos", f.path, r-a)})
		}
		if s := countMatches(skipRe, f.added) - countMatches(skipRe, f.removed); s > 0 {
			out = append(out, finding{"tests", fmt.Sprintf("%s: %d teste(s) pulado(s)", f.path, s)})
		}
	}
	return out
}

func isProtected(p string, protected []string) bool {
	for _, pat := range protected {
		pat = strings.TrimSuffix(pat, "/")
		if p == pat || strings.HasPrefix(p, pat+"/") {
			return true
		}
		if ok, _ := path.Match(pat, p); ok {
			return true
		}
		if !strings.Contains(pat, "/") {
			if ok, _ := path.Match(pat, path.Base(p)); ok {
				return true
			}
		}
	}
	return false
}

func checkProtected(files []fileDiff, protected []string) []finding {
	var out []finding
	for _, f := range files {
		if isProtected(f.path, protected) {
			out = append(out, finding{"protected", "arquivo protegido alterado: " + f.path})
		}
	}
	return out
}

func checkSize(files []fileDiff, max int) []finding {
	n := 0
	for _, f := range files {
		n += len(f.added) + len(f.removed)
	}
	if n > max {
		return []finding{{"size", fmt.Sprintf("diff de %d linhas passa do limite de %d", n, max)}}
	}
	return nil
}

// gitDiff returns the working tree diff against base, untracked files included.
func gitDiff(dir, base string) (string, error) {
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		b, err := cmd.Output()
		return string(b), err
	}
	untracked, err := git("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return "", err
	}
	// intent-to-add makes untracked files show up in `git diff` as additions; it is undone
	// right after, or it stays in the index and breaks `git stash` ("not uptodate").
	var added []string
	for _, p := range strings.Split(strings.TrimSpace(untracked), "\n") {
		if p == "" {
			continue
		}
		if _, err := git("add", "-N", "--", p); err != nil {
			return "", err
		}
		added = append(added, p)
	}
	defer func() {
		if len(added) > 0 {
			_, _ = git(append([]string{"reset", "-q", "--"}, added...)...)
		}
	}()
	return git("diff", "--no-color", "--no-ext-diff", base)
}

type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

// evalGate runs every gate on the working tree diff against base; the CI output goes to ciOut.
func evalGate(dir, base string, maxLines int, ci string, extraProtect []string, ciOut io.Writer) (int, []finding, error) {
	diff, err := gitDiff(dir, base)
	if err != nil {
		return 0, nil, err
	}
	files := parseDiff(strings.NewReader(diff))

	protected := append(append([]string{}, defaultProtected...), extraProtect...)
	var findings []finding
	findings = append(findings, checkProtected(files, protected)...)
	findings = append(findings, checkLoosenedTests(files)...)
	findings = append(findings, checkSize(files, maxLines)...)

	if strings.TrimSpace(ci) != "" {
		cmd, err := shellCommand(ci)
		if err != nil {
			return 0, nil, err // a missing tool is not a failed attempt: the run stops with how to fix it
		}
		cmd.Dir = dir
		cmd.Stdout, cmd.Stderr = ciOut, ciOut
		if err := cmd.Run(); err != nil {
			findings = append(findings, finding{"ci", fmt.Sprintf("`%s` falhou: %v", ci, err)})
		}
	}
	return len(files), findings, nil
}

// runGate implements `axyn gate`: the deterministic gates of spec 021 FR-5.
func runGate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	base := fs.String("base", "HEAD", "referência do git contra a qual o diff é medido")
	maxLines := fs.Int("max-lines", defaultMaxLines, "limite de linhas alteradas (adicionadas + removidas)")
	ci := fs.String("ci", defaultCICmd, "comando do CI do projeto; vazio pula")
	dir := fs.String("dir", ".", "raiz do repositório")
	var protect stringList
	fs.Var(&protect, "protect", "caminho protegido adicional (repetível)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	nfiles, findings, err := evalGate(*dir, *base, *maxLines, *ci, protect, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn gate: git diff falhou: %v\n", err)
		return exitUsage
	}

	if len(findings) == 0 {
		_, _ = fmt.Fprintf(stdout, "gate: verde (%d arquivo(s) no diff)\n", nfiles)
		return exitOK
	}
	for _, f := range findings {
		_, _ = fmt.Fprintf(stdout, "gate: reprovado [%s] %s\n", f.gate, f.reason)
	}
	return exitGateFailed
}
