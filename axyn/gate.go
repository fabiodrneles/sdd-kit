package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	addedAt []int // line number of each added line in the new file
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
	hunkRe     = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)`)
)

// parseDiff reads `git diff` output into per-file added and removed lines.
func parseDiff(r io.Reader) []fileDiff {
	var files []fileDiff
	newLine := 0
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
		case strings.HasPrefix(line, "@@"):
			if m := hunkRe.FindStringSubmatch(line); m != nil {
				newLine, _ = strconv.Atoi(m[1])
			}
		case strings.HasPrefix(line, "+"):
			f.added = append(f.added, line[1:])
			f.addedAt = append(f.addedAt, newLine)
			newLine++
		case strings.HasPrefix(line, "-"):
			f.removed = append(f.removed, line[1:])
		case strings.HasPrefix(line, " "):
			newLine++
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

// checkSize limits the production code a ticket changes. Test files do not count (#368):
// a ticket that only adds tests to reach the coverage goal wrote 1180 lines of tests on
// the owner's project and was rejected, though tests are what the axyn asks for. Removed
// test lines still count, and the loosened-tests gate watches what changes in them.
func checkSize(files []fileDiff, max int) []finding {
	n, tests := 0, 0
	for _, f := range files {
		if isTestPath(f.path) {
			tests += len(f.added)
			n += len(f.removed)
			continue
		}
		n += len(f.added) + len(f.removed)
	}
	if n > max {
		msg := fmt.Sprintf("diff de %d linhas passa do limite de %d", n, max)
		if tests > 0 {
			msg += fmt.Sprintf(" (fora %d linhas novas de teste, que não contam)", tests)
		}
		return []finding{{"size", msg}}
	}
	return nil
}

// gitDiff returns the working tree diff against base, untracked files included.
func gitDiff(dir, base string) (string, error) {
	// git's own message goes into the error (#365): "exit status 128" alone said nothing.
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-c", "core.quotepath=off"}, args...)...)
		cmd.Dir = dir
		var stderr strings.Builder
		cmd.Stderr = &stderr
		b, err := cmd.Output()
		if err != nil {
			if msg := strings.TrimSpace(stderr.String()); msg != "" {
				err = fmt.Errorf("git %s: %s", args[0], msg)
			}
		}
		return string(b), err
	}
	// -z: names with accents or spaces come raw, not quoted ("caf\303\251"), so git add finds them.
	untracked, err := git("ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return "", err
	}
	// intent-to-add makes untracked files show up in `git diff` as additions; it is undone
	// right after, or it stays in the index and breaks `git stash` ("not uptodate"). A file
	// git refuses to add (a reserved Windows name, a nested repository) is left out of the
	// diff instead of stopping the whole gate.
	var added []string
	for _, p := range strings.Split(untracked, "\x00") {
		// "sub/": a nested repository (git lists it as a folder); adding it makes a gitlink
		// that git diff cannot hash ("does not have a commit checked out"), exit 128.
		if p == "" || strings.HasSuffix(p, "/") {
			continue
		}
		if _, err := git("add", "-N", "--", p); err != nil {
			continue
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
	cleanCoverageProfiles(dir)
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
	findings = append(findings, checkCodeHasTests(files)...)

	if strings.TrimSpace(ci) != "" {
		cmd, err := shellCommand(ci)
		if err != nil {
			return 0, nil, err // a missing tool is not a failed attempt: the run stops with how to fix it
		}
		cmd.Dir = dir
		var out strings.Builder
		w := io.MultiWriter(ciOut, &out)
		cmd.Stdout, cmd.Stderr = w, w
		floor, ok := coverageFloor(dir)
		if ok {
			if cmd.Env == nil {
				cmd.Env = os.Environ()
			}
			cmd.Env = append(cmd.Env, fmt.Sprintf("COVERAGE_MIN=%d", floor))
		}
		lastCoverage = -1
		started := time.Now()
		err = cmd.Run()
		if c, found := parseCoverage(out.String()); found {
			lastCoverage = c
		}
		if err == nil {
			findings = append(findings, checkPatchCoverage(dir, files, started)...)
		}
		if err != nil {
			msg := fmt.Sprintf("`%s` falhou: %v", ci, err)
			if lines := ciErrors(out.String(), 6); len(lines) > 0 {
				msg += "; erros: " + strings.Join(lines, " | ")
			}
			findings = append(findings, finding{"ci", msg})
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

// cleanCoverageProfiles removes the Go coverage profiles a model left loose in the tree
// (coverage.out, full_coverage, ...): untracked files that start with "mode: set|count|
// atomic" (#366). They are scratch output, not part of the ticket: in the diff they would
// count against the size gate and land in the PR.
func cleanCoverageProfiles(dir string) {
	cmd := exec.Command("git", "ls-files", "-z", "--others", "--exclude-standard")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return
	}
	for _, p := range strings.Split(string(out), "\x00") {
		if p == "" || strings.HasSuffix(p, ".go") {
			continue
		}
		full := filepath.Join(dir, filepath.FromSlash(p))
		f, err := os.Open(full)
		if err != nil {
			continue
		}
		head := make([]byte, 16)
		n, _ := f.Read(head)
		_ = f.Close()
		if h := string(head[:n]); strings.HasPrefix(h, "mode: set") || strings.HasPrefix(h, "mode: count") || strings.HasPrefix(h, "mode: atomic") {
			_ = os.Remove(full)
		}
	}
}
