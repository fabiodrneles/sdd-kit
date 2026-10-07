package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownReason(t *testing.T) {
	cases := map[string]string{
		"Error: 429 Too Many Requests":                    "limite",
		"AI_APICallError: insufficient_quota":             "limite",
		"provider returned 503 Service Unavailable":       "fora do ar",
		"TypeError: fetch failed (ECONNREFUSED)":          "fora do ar",
		"tudo certo, adicionei a flag --timeout ao check": "",
	}
	for out, want := range cases {
		got := downReason(out, false)
		if (want == "") != (got == "") || !strings.Contains(got, want) {
			t.Errorf("downReason(%q) = %q, quer %q", out, got, want)
		}
	}
	if downReason("Error: 429 Too Many Requests", true) != "" {
		t.Error("um modelo que mudou arquivos não está fora do ar")
	}
}

// #344: a model that hits its limit leaves the queue without spending attempts, and the
// same ticket goes to the next model.
func TestDownModelIsReplaced(t *testing.T) {
	dir := repo(t)
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	write(t, filepath.Dir(cfg), "config.yaml", "models:\n  - id: limitado\n  - id: reserva\n")
	t.Setenv("AXYN_OPENCODE", fakeAgent(t, `if [ "$5" = limitado ]; then echo "Error: 429 Too Many Requests"; else printf 'ok %s\n' "$n" > "f$n.txt"; fi`))
	var out, errb strings.Builder
	code := runRunCmd([]string{"--wait", "--dir", dir, "--ci", "true", "--config", cfg, "crie uma coisa"}, &out, &errb)
	if code != exitOK {
		t.Fatalf("deveria entregar com o modelo reserva: %d\n%s%s", code, out.String(), errb.String())
	}
	if !strings.Contains(out.String()+errb.String(), "modelo limitado limite de uso ou de tokens atingido") {
		t.Errorf("o log deveria dizer a troca:\n%s", out.String())
	}
	pl, _, _ := (&mcpServer{dir: dir}).loadPlan()
	for _, tk := range pl.Tickets {
		for _, a := range tk.Attempts {
			if a.Model == "limitado" {
				t.Errorf("o modelo fora do ar não pode gastar tentativa: %+v", a)
			}
		}
	}
	_ = os.Remove(cfg)
}

// OpenCode's free tier refuses the restricted planner: the engine plans again with the
// open one and undoes what it touched, keeping only the spec.
func TestPlannerFallsBackOnFreeTier(t *testing.T) {
	dir := repo(t)
	bin := filepath.Join(t.TempDir(), "axyn")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	fd := t.TempDir()
	write(t, fd, "plan.json", planRequest+"\n")
	write(t, fd, "fake.sh", `case "$3" in
axyn-plan) echo "Error: OpenCode's free tier can only be used from within OpenCode" ;;
axyn-plan-open) echo lixo > lixo.txt; '`+bin+`' mcp < '`+filepath.Join(fd, "plan.json")+`' > /dev/null ;;
axyn-code) n=$(printf '%s' "$*" | sed -n 's/.*Ticket \([0-9]*\):.*/\1/p' | head -n 1); printf 'ok %s\n' "$n" > "f$n.txt"; printf 'package x\n' > "f${n}_test.go" ;;
esac
`)
	t.Setenv("AXYN_OPENCODE", "sh "+filepath.Join(fd, "fake.sh"))
	code, out := runLoopIn(t, dir)
	if code != exitOK {
		t.Fatalf("deveria planejar com o agente aberto e entregar: %d\n%s", code, out)
	}
	if !strings.Contains(out, "refazendo com o axyn-plan-open") || !strings.Contains(out, "as mudanças foram desfeitas") {
		t.Errorf("o log deveria dizer a troca e a limpeza:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "lixo.txt")); err == nil {
		t.Error("o arquivo do planejador deveria ter sido desfeito")
	}
}
