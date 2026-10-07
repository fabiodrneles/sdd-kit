package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const notesBody = "### Adicionado\n\n- `axyn x`: faz x (#330)\n- `axyn y`: faz y (#331)\n- z\n- w\n\n## Como atualizar o axyn\n"

// fakeRelease serves tag as the latest release and counts the requests.
func fakeRelease(t *testing.T, tag string, cur string) *int32 {
	t.Helper()
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&n, 1)
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `","html_url":"https://example.com/r","body":` + quote(notesBody) + `}`))
	}))
	t.Cleanup(srv.Close)
	oldURL, oldV := latestURL, version
	t.Cleanup(func() { latestURL, version = oldURL, oldV })
	latestURL, version = srv.URL, cur
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("AXYN_NO_UPDATE_CHECK", "")
	return &n
}

func quote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return `"` + r.Replace(s) + `"`
}

// 021 #328 AC-1, AC-4: a newer release shows version, news and the command; the same day uses the cache.
func TestUpdateNoticeNewer(t *testing.T) {
	n := fakeRelease(t, "v1.17.0", "1.16.0")
	msg := updateNotice()
	for _, want := range []string{"Saiu o axyn v1.17.0 (você tem a v1.16.0)", "`axyn x`: faz x", "`axyn y`", "- z", "na raiz do projeto", "install-axyn", "https://example.com/r"} {
		if !strings.Contains(msg, want) {
			t.Errorf("o aviso não traz %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "(#330)") || strings.Contains(msg, "- w") {
		t.Errorf("só 3 novidades, sem o número do PR:\n%s", msg)
	}
	_ = updateNotice()
	if atomic.LoadInt32(n) != 1 {
		t.Errorf("a segunda chamada no dia deveria usar o cache: %d requisições", *n)
	}
	if !strings.Contains(withUpdate("status"), "status\n\nSaiu o axyn") {
		t.Errorf("withUpdate deveria pôr o aviso depois da mensagem")
	}
}

// 021 #328 AC-2: same or older version, or a dev build: no notice.
func TestUpdateNoticeNotNewer(t *testing.T) {
	for _, c := range [][2]string{{"v1.16.0", "1.16.0"}, {"v1.15.2", "1.16.0"}, {"v1.17.0", "dev"}} {
		fakeRelease(t, c[0], c[1])
		if msg := updateNotice(); msg != "" {
			t.Errorf("%s com %s: sem aviso, veio %q", c[0], c[1], msg)
		}
	}
}

// 021 #328 AC-3: a failing network gives no notice, quickly.
func TestUpdateNoticeNetworkDown(t *testing.T) {
	fakeRelease(t, "v9.0.0", "1.16.0")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(3 * time.Second)
	}))
	defer srv.Close()
	latestURL = srv.URL
	start := time.Now()
	if msg := updateNotice(); msg != "" {
		t.Errorf("sem rede, nenhum aviso: %q", msg)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("o aviso não pode atrasar o comando: %v", d)
	}
	latestURL = "http://127.0.0.1:1"
	if msg := updateNotice(); msg != "" {
		t.Errorf("conexão recusada, nenhum aviso: %q", msg)
	}
}

// 021 #328 AC-5: AXYN_NO_UPDATE_CHECK=1 makes no request.
func TestUpdateNoticeOptOut(t *testing.T) {
	n := fakeRelease(t, "v1.17.0", "1.16.0")
	t.Setenv("AXYN_NO_UPDATE_CHECK", "1")
	if msg := updateNotice(); msg != "" || atomic.LoadInt32(n) != 0 {
		t.Errorf("desligado: sem aviso e sem requisição (%q, %d)", msg, *n)
	}
	_ = os.Unsetenv("AXYN_NO_UPDATE_CHECK")
}

// 021 #328 AC-1: doctor and status print the notice.
func TestUpdateNoticeInDoctorAndStatus(t *testing.T) {
	fakeRelease(t, "v1.17.0", "1.16.0")
	dir := repo(t)
	var out, errb strings.Builder
	runSetupCmd([]string{"--dir", dir}, &out, &errb, false)
	if !strings.Contains(out.String(), "Saiu o axyn v1.17.0") {
		t.Errorf("o doctor deveria avisar:\n%s", out.String())
	}
	if err := saveRun(dir, &runState{ID: "20260101-000000", Request: "x", Status: runDone, Started: time.Now()}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	runStatusCmd([]string{"--dir", dir}, &out, &errb)
	if !strings.Contains(out.String(), "Saiu o axyn v1.17.0") {
		t.Errorf("o status deveria avisar:\n%s", out.String())
	}
	if text, _ := callTool(t, dir, "axyn_status", map[string]any{}); !strings.Contains(text, "Saiu o axyn v1.17.0") {
		t.Errorf("o /axyn (status) deveria avisar:\n%s", text)
	}
}
