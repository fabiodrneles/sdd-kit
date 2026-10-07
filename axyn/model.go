package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// axyn model picks the model of the ladder (spec 021 FR-6) by a command, so nobody has
// to remember the config.yaml format: a numbered menu of the models opencode knows, or
// the id straight away.

// stdin is replaced in tests.
var stdin io.Reader = os.Stdin

// opencodeModels lists the ids `opencode models` prints; only the free ones unless all.
func opencodeModels(all bool) ([]string, error) {
	if _, err := lookPath("opencode"); err != nil {
		return nil, fmt.Errorf("opencode não instalado; instale: curl -fsSL https://opencode.ai/install | bash (no Windows: npm install -g opencode-ai)")
	}
	b, err := exec.Command("opencode", "models").Output()
	if err != nil {
		return nil, fmt.Errorf("opencode models falhou: %v", err)
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(l)
		if !strings.Contains(l, "/") || strings.ContainsAny(l, " \t") {
			continue
		}
		if all || strings.Contains(strings.ToLower(l), "free") {
			out = append(out, l)
		}
	}
	return out, nil
}

func formatModels(ms []model) string {
	var b strings.Builder
	b.WriteString("# Modelos do axyn, em ordem: se um não passa nos portões, o axyn tenta o próximo.\n")
	b.WriteString("# Troque com: axyn model. key_env é o NOME da variável com a chave, nunca a chave.\n")
	b.WriteString("models:\n")
	for _, m := range ms {
		fmt.Fprintf(&b, "  - id: %s\n", m.ID)
		if m.KeyEnv != "" {
			fmt.Fprintf(&b, "    key_env: %s\n", m.KeyEnv)
		}
	}
	return b.String()
}

// defaultKeyEnv is the variable a provider's key usually lives in ("" when none needed).
func defaultKeyEnv(id string) string {
	switch strings.SplitN(id, "/", 2)[0] {
	case "openrouter":
		return "OPENROUTER_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	case "google":
		return "GEMINI_API_KEY"
	}
	return ""
}

func setEnvHint(name string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`setx %s "sua-chave" (e abra um PowerShell novo)`, name)
	}
	return fmt.Sprintf(`echo 'export %s="sua-chave"' >> ~/.bashrc && source ~/.bashrc (no macOS, ~/.zshrc)`, name)
}

func runModelCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn model", flag.ContinueOnError)
	fs.SetOutput(stderr)
	all := fs.Bool("all", false, "lista todos os modelos do opencode, não só os gratuitos")
	only := fs.Bool("only", false, "deixa só este modelo (sem escada)")
	keyEnv := fs.String("key-env", "", "nome da variável de ambiente com a chave (padrão: a do provedor)")
	config := fs.String("config", defaultConfigPath(), "arquivo de configuração do axyn")
	// flags may come after the id: axyn model ID --only
	var pos []string
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			return exitUsage
		}
		if fs.NArg() == 0 {
			break
		}
		pos = append(pos, fs.Arg(0))
		rest = fs.Args()[1:]
	}
	current, err := loadModels(*config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn model: %v\n", err)
		return exitFail
	}
	in := bufio.NewReader(stdin)
	id := ""
	if len(pos) > 0 {
		id = pos[0]
	} else {
		if len(current) > 0 {
			_, _ = fmt.Fprintf(stdout, "modelo atual: %s\n", current[0].ID)
		} else {
			_, _ = fmt.Fprintln(stdout, "modelo atual: nenhum (o axyn usa o modelo padrão do opencode)")
		}
		list, err := opencodeModels(*all)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "axyn model: %v\n", err)
			return exitFail
		}
		if len(list) == 0 {
			_, _ = fmt.Fprintln(stderr, "axyn model: o opencode não listou nenhum modelo (tente axyn model --all)")
			return exitFail
		}
		kind := "gratuitos"
		if *all {
			kind = "disponíveis"
		}
		_, _ = fmt.Fprintf(stdout, "\nmodelos %s no seu opencode:\n", kind)
		for i, m := range list {
			_, _ = fmt.Fprintf(stdout, "  %2d) %s\n", i+1, m)
		}
		_, _ = fmt.Fprint(stdout, "\ndigite o número do modelo e aperte Enter (Enter vazio cancela): ")
		line, _ := in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			_, _ = fmt.Fprintln(stdout, "nada mudou")
			return exitOK
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(list) {
			_, _ = fmt.Fprintf(stderr, "axyn model: %q não é um número da lista (1 a %d); nada mudou\n", line, len(list))
			return exitUsage
		}
		id = list[n-1]
	}
	if strings.ContainsAny(id, " \t") || !strings.Contains(id, "/") {
		_, _ = fmt.Fprintf(stderr, "axyn model: %q não parece um id de modelo (provedor/modelo, como opencode/nemotron-3-ultra-free)\n", id)
		return exitUsage
	}

	m := model{ID: id, KeyEnv: *keyEnv}
	if m.KeyEnv == "" {
		m.KeyEnv = defaultKeyEnv(id)
		if m.KeyEnv != "" && len(pos) == 0 {
			_, _ = fmt.Fprintf(stdout, "nome da variável com a chave [%s]: ", m.KeyEnv)
			if line, _ := in.ReadString('\n'); strings.TrimSpace(line) != "" {
				m.KeyEnv = strings.TrimSpace(line)
			}
		}
	}
	if strings.HasPrefix(m.KeyEnv, "sk-") || strings.ContainsAny(m.KeyEnv, " -") {
		_, _ = fmt.Fprintln(stderr, "axyn model: --key-env é o NOME da variável (ex.: OPENROUTER_API_KEY), nunca a chave; nada foi gravado")
		return exitUsage
	}
	ladder := []model{m}
	if !*only {
		for _, c := range current {
			if c.ID != id {
				ladder = append(ladder, c)
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(*config), 0o755); err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn model: %v\n", err)
		return exitFail
	}
	if err := os.WriteFile(*config, []byte(formatModels(ladder)), 0o644); err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn model: %v\n", err)
		return exitFail
	}
	_, _ = fmt.Fprintf(stdout, "pronto: o axyn agora usa %s (gravado em %s)\n", id, *config)
	if len(ladder) > 1 {
		_, _ = fmt.Fprintf(stdout, "se ele não passar nos portões, tenta: %s\n", strings.Join(ids(ladder[1:]), ", "))
	}
	if m.KeyEnv != "" && os.Getenv(m.KeyEnv) == "" {
		_, _ = fmt.Fprintf(stdout, "atenção: a variável %s não está definida neste terminal; para guardar a chave: %s\n", m.KeyEnv, setEnvHint(m.KeyEnv))
	}
	_, _ = fmt.Fprintln(stdout, "no opencode, escolha o mesmo modelo com /models para a conversa")
	return exitOK
}

func ids(ms []model) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.ID
	}
	return out
}
