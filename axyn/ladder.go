package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The model ladder (spec 021 FR-6) and the per-ticket record (FR-7).

// model is one rung of the ladder. KeyEnv names the environment variable that holds the
// key: the key itself never goes in the file (NFR-2).
type model struct {
	ID     string
	KeyEnv string
}

// attempt is one gate run on a ticket diff.
type attempt struct {
	Model  string   `json:"model"`
	Green  bool     `json:"green"`
	Reason []string `json:"reason,omitempty"`
	Step   string   `json:"step,omitempty"` // the FR-8 recovery step this attempt ran under
	Plan   []string `json:"plan,omitempty"`
}

// defaultConfigPath is ~/.config/axyn/config.yaml, or $XDG_CONFIG_HOME/axyn/config.yaml.
func defaultConfigPath() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "axyn", "config.yaml")
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(h, ".config", "axyn", "config.yaml")
}

// parseModels reads the `models:` list of the config, a small YAML subset:
//
//	models:
//	  - id: openrouter/qwen/qwen3-coder:free
//	    key_env: OPENROUTER_API_KEY
//	  - anthropic/claude-sonnet-5-5
//
// A literal key (api_key, key) is refused, so a secret cannot be committed by accident.
func parseModels(text string) ([]model, error) {
	var out []model
	in := false
	for n, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, " \t\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			in = trimmed == "models:"
			continue
		}
		if !in {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			trimmed = strings.TrimSpace(trimmed[2:])
			if !strings.Contains(trimmed, ": ") && !strings.HasSuffix(trimmed, ":") {
				out = append(out, model{ID: unquote(trimmed)})
				continue
			}
			out = append(out, model{})
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("config: linha %d fora de um item de models", n+1)
		}
		k, v, ok := strings.Cut(trimmed, ":")
		if !ok {
			return nil, fmt.Errorf("config: linha %d inválida", n+1)
		}
		v = unquote(strings.TrimSpace(v))
		m := &out[len(out)-1]
		switch strings.TrimSpace(k) {
		case "id":
			m.ID = v
		case "key_env":
			m.KeyEnv = v
		case "api_key", "key":
			return nil, fmt.Errorf("config: linha %d traz a chave no arquivo; use key_env com o nome da variável de ambiente", n+1)
		}
	}
	for i, m := range out {
		if m.ID == "" {
			return nil, fmt.Errorf("config: modelo %d sem id", i+1)
		}
	}
	return out, nil
}

func unquote(s string) string { return strings.Trim(s, `"'`) }

// loadModels returns the ladder; no config file means no ladder (nil, nil).
func loadModels(path string) ([]model, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseModels(string(b))
}

// ladderState says which rung a ticket is on, from its recorded attempts;
// len(models) means the ladder is exhausted.
func ladderState(models []model, attempts []attempt) int {
	idx := 0
	for idx < len(models) && countFails(attempts, models[idx].ID) >= maxFailsPerModel {
		idx++
	}
	return idx
}

func countFails(attempts []attempt, id string) int {
	n := 0
	for _, a := range attempts {
		if a.Model == id && !a.Green {
			n++
		}
	}
	return n
}

func keyHint(m model) string {
	if m.KeyEnv != "" && os.Getenv(m.KeyEnv) == "" {
		return " (falta a variável " + m.KeyEnv + ")"
	}
	return ""
}

// recordAttempt adds a gate run to the open ticket and returns the advice for the report:
// the recovery step of FR-8 or the ladder move of FR-6. recovered says the advice already
// carries the diagnosis, so the report need not repeat the CI log. With no ladder, no plan
// or no open ticket it does nothing.
func (s *mcpServer) recordAttempt(green bool, findings []finding, plan []string, ciLog string) (advice string, recovered bool) {
	models, err := loadModels(s.config)
	if err == nil && len(models) == 0 && s.fallback {
		models = []model{{ID: placeholder}}
	}
	if err != nil || len(models) == 0 {
		return "", false
	}
	pl, t := s.openTicket()
	if t == nil {
		return "", false
	}
	_, planPath, _ := s.loadPlan()
	cur := models[min(ladderState(models, t.Attempts), len(models)-1)]
	before := countFails(t.Attempts, cur.ID)
	a := attempt{Model: cur.ID, Green: green, Plan: plan}
	if i, _ := stepFor(before); i >= 0 && i < len(recSteps) {
		a.Step = recSteps[i].name
	}
	for _, f := range findings {
		a.Reason = append(a.Reason, "["+f.gate+"] "+f.reason)
	}
	t.Attempts = append(t.Attempts, a)
	if green {
		t.Model = cur.ID
		switch {
		case a.Step != "":
			t.Strategy = a.Step
		case len(t.Attempts) > 1:
			t.Strategy = "outro modelo"
		}
	}
	defer func() { _ = s.savePlan(pl, planPath) }()
	if green {
		return "", false
	}
	idx := ladderState(models, t.Attempts)
	switch {
	case idx >= len(models):
		return s.exhausted(t, models), false
	case models[idx].ID != cur.ID:
		return fmt.Sprintf("escada: %s reprovou %d vez(es), com a recuperação toda; use o próximo modelo: %s%s",
			cur.ID, maxFailsPerModel, models[idx].ID, keyHint(models[idx])), false
	}
	fails := countFails(t.Attempts, cur.ID)
	line := fmt.Sprintf("escada: modelo atual %s (reprovações: %d de %d)", cur.ID, fails, maxFailsPerModel)
	if rec := s.recoveryAdvice(t, fails, ciLog); rec != "" {
		return line + "\n" + rec, true
	}
	return line, false
}

// exhausted keeps the work in a WIP commit on the ticket branch and says what was tried.
func (s *mcpServer) exhausted(t *ticket, models []model) string {
	var b strings.Builder
	fmt.Fprintf(&b, "escada esgotada: %d modelo(s), %d tentativa(s) no ticket %d (%s)\n", len(models), len(t.Attempts), t.ID, t.Title)
	for i, a := range t.Attempts {
		step := ""
		if a.Step != "" {
			step = " [" + a.Step + "]"
		}
		fmt.Fprintf(&b, "  %d. %s%s: %s\n", i+1, a.Model, step, strings.Join(a.Reason, "; "))
	}
	b.WriteString(s.saveWIP(t, "escada de modelos esgotada"))
	return strings.TrimRight(b.String(), "\n")
}

// saveWIP keeps the work of a ticket in a WIP commit on its branch and says so.
func (s *mcpServer) saveWIP(t *ticket, why string) string {
	branch, _ := s.git("rev-parse", "--abbrev-ref", "HEAD")
	if branch == "main" || branch == "master" || branch == "HEAD" {
		branch = s.uniqueBranch(fmt.Sprintf("feat/%d-wip", t.ID))
		if out, err := s.git("checkout", "-b", branch); err != nil {
			return "WIP não gravado: " + out
		}
	}
	_, _ = s.git("add", "-A")
	if out, err := s.git("commit", "-m", fmt.Sprintf("wip: ticket %d sem resolver (%s)", t.ID, why)); err != nil {
		return "WIP não gravado: " + out
	}
	return fmt.Sprintf("trabalho guardado num commit WIP em %s; falta o portão passar, nada foi quebrado.", branch)
}
