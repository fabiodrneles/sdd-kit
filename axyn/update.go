package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// The update notice (#328): doctor, status and the /axyn answers say when a newer axyn
// is out, what it brings and how to update. At most one request a day, a short timeout,
// and silence on any failure: the notice never gets in the way of the work.

var (
	latestURL     = "https://api.github.com/repos/fabiodrneles/sdd-kit/releases/latest"
	updateTimeout = 1500 * time.Millisecond
	updateEvery   = 24 * time.Hour
)

type releaseInfo struct {
	Checked time.Time `json:"checked"`
	Tag     string    `json:"tag_name"`
	Body    string    `json:"body"`
	URL     string    `json:"html_url"`
}

func updateCachePath() string {
	cfg := defaultConfigPath()
	if cfg == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(cfg), "update-check.json")
}

// semver reads "v1.2.3" or "1.2.3"; ok is false for anything else (dev builds).
func semver(v string) (out [3]int, ok bool) {
	parts := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".", 3)
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func newer(latest, current string) bool {
	l, ok1 := semver(latest)
	c, ok2 := semver(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// latestRelease returns the cached release when it is fresh, else asks GitHub.
func latestRelease() (releaseInfo, bool) {
	var r releaseInfo
	cache := updateCachePath()
	if cache != "" {
		if b, err := os.ReadFile(cache); err == nil && json.Unmarshal(b, &r) == nil && time.Since(r.Checked) < updateEvery {
			return r, r.Tag != ""
		}
	}
	client := &http.Client{Timeout: updateTimeout}
	req, err := http.NewRequest("GET", latestURL, nil)
	if err != nil {
		return r, false
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return r, false
	}
	defer func() { _ = resp.Body.Close() }()
	r = releaseInfo{}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&r) != nil {
		return r, false
	}
	r.Checked = time.Now()
	if cache != "" {
		if b, err := json.Marshal(r); err == nil {
			_ = os.MkdirAll(filepath.Dir(cache), 0o755)
			_ = os.WriteFile(cache, b, 0o644)
		}
	}
	return r, r.Tag != ""
}

var prRef = regexp.MustCompile(`\s*\(#[0-9][^)]*\)\s*$`)

// highlights are the first items of the release notes (the CHANGELOG section).
func highlights(body string, max int) []string {
	var out []string
	for _, l := range strings.Split(body, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "## ") && len(out) > 0 {
			break
		}
		if strings.HasPrefix(l, "- ") {
			out = append(out, prRef.ReplaceAllString(l, ""))
			if len(out) == max {
				break
			}
		}
	}
	return out
}

func updateCommand(_ string) string {
	return "axyn update"
}

// updateNotice is the text to show, or "" when there is nothing newer (or no way to know).
func updateNotice() string {
	if os.Getenv("AXYN_NO_UPDATE_CHECK") == "1" {
		return ""
	}
	if _, ok := semver(version); !ok {
		return ""
	}
	r, ok := latestRelease()
	if !ok || !newer(r.Tag, version) {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Saiu o axyn %s (você tem a v%s).", r.Tag, strings.TrimPrefix(version, "v"))
	if hs := highlights(r.Body, 3); len(hs) > 0 {
		b.WriteString(" Novidades:\n")
		for _, h := range hs {
			fmt.Fprintf(&b, "  %s\n", h)
		}
	} else {
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "Para atualizar, feche o opencode e rode no terminal, na raiz do projeto:\n  %s\n", updateCommand(runtime.GOOS))
	if r.URL != "" {
		fmt.Fprintf(&b, "Tudo o que mudou: %s\n", r.URL)
	}
	b.WriteString("(para não ver este aviso: AXYN_NO_UPDATE_CHECK=1)")
	return b.String()
}

// withUpdate appends the notice to a message, after a blank line.
func withUpdate(msg string) string {
	if n := updateNotice(); n != "" {
		return msg + "\n\n" + n
	}
	return msg
}
