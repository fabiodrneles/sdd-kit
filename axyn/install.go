package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const opencodeSchema = "https://opencode.ai/config.json"

const planPrompt = `Você é o axyn-plan. Transforme o pedido do usuário em uma spec e em tickets pequenos.
Chame a ferramenta axyn_plan com o nome, a spec e os tickets; ela valida e grava. Se ela recusar, corrija o que ela apontou e chame de novo.
Não escreva código do projeto.`

const codePrompt = `Você é o axyn-code. Escreva o código de um único ticket, só o que o ticket pede.
Nunca afrouxe, pule ou apague um teste para o CI passar; se o teste estiver certo, conserte o código.
Não mexa em caminhos protegidos, nem em workflows, nem na configuração do axyn. Termine quando o ticket estiver pronto.`

const commandTemplate = `Chame a ferramenta axyn_run com o pedido do usuário, exatamente como ele escreveu: $ARGUMENTS

O axyn conduz o resto sozinho (plano, código, portões, recuperação e entrega). Não decida nada do processo e não escreva código: mostre o andamento com axyn_status até ele terminar ou parar.
Se o axyn_status trouxer uma pergunta, faça-a ao usuário, grave a resposta com axyn_decide e chame axyn_run de novo com resume.`

// axynConfig is what install adds to opencode.json: the MCP server, the two agents
// and the /axyn command (spec 021 FR-2).
func axynConfig() map[string]map[string]any {
	return map[string]map[string]any{
		"mcp": {
			"axyn": map[string]any{
				"type":    "local",
				"command": []string{"axyn", "mcp"},
				"enabled": true,
			},
		},
		// mode "all": o motor chama os agentes com `opencode run --agent`, que recusa um
		// "subagent" e cai no agente padrão, sem o prompt nem as ferramentas do axyn.
		"agent": {
			"axyn-plan": map[string]any{
				"description": "Escreve a spec e os tickets de um pedido, pelo axyn",
				"mode":        "all",
				"prompt":      planPrompt,
				"tools": map[string]any{
					"axyn_*": true, "read": true, "glob": true, "grep": true, "list": true,
					"write": false, "edit": false, "bash": false,
				},
			},
			"axyn-code": map[string]any{
				"description": "Escreve o código de um ticket, só com as ferramentas de código",
				"mode":        "all",
				"prompt":      codePrompt,
				"tools": map[string]any{
					"axyn_*": false, "read": true, "glob": true, "grep": true, "list": true,
					"write": true, "edit": true, "bash": true,
				},
				// Only the engine commits and pushes, after the gates (axyn_ship); the
				// coding agent may run the build and tests, never ship by itself.
				"permission": map[string]any{
					"bash": map[string]any{
						"git commit *": "deny", "git push *": "deny", "git reset *": "deny",
						"git checkout *": "deny", "git rebase *": "deny", "gh *": "deny",
					},
				},
			},
		},
		"command": {
			"axyn": map[string]any{
				"description": "Constrói o pedido com o processo do axyn",
				"template":    commandTemplate,
			},
		},
	}
}

// mergeSection adds our entries to cfg[key], keeping every other entry the user has.
func mergeSection(cfg map[string]any, key string, add map[string]any) error {
	cur := map[string]any{}
	if v, ok := cfg[key]; ok {
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%q do opencode.json não é um objeto; nada foi alterado", key)
		}
		cur = m
	}
	for k, v := range add {
		cur[k] = v
	}
	cfg[key] = cur
	return nil
}

// installOpencode merges the axyn setup into dir/opencode.json. A file it cannot
// parse is left untouched and reported, never overwritten.
func installOpencode(dir string) (string, error) {
	path := filepath.Join(dir, "opencode.json")
	cfg := map[string]any{}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if len(bytes.TrimSpace(data)) > 0 {
			if err := json.Unmarshal(data, &cfg); err != nil {
				return path, fmt.Errorf("%s não é JSON válido (comentários?): %v; nada foi alterado", path, err)
			}
		}
	case !os.IsNotExist(err):
		return path, err
	}
	if _, ok := cfg["$schema"]; !ok {
		cfg["$schema"] = opencodeSchema
	}
	for key, add := range axynConfig() {
		if err := mergeSection(cfg, key, add); err != nil {
			return path, err
		}
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return path, err
	}
	tmp := path + ".axyn-tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0o644); err != nil {
		return path, err
	}
	return path, os.Rename(tmp, path)
}

func runInstall(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axyn install", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "diretório do projeto")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	path, err := installOpencode(*dir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "axyn: %v\n", err)
		return exitFail
	}
	if note := excludeConfig(*dir); note != "" {
		_, _ = fmt.Fprintln(stdout, note)
	}
	_, _ = fmt.Fprintf(stdout, "axyn: %s configurado (servidor MCP axyn, agentes axyn-plan e axyn-code, comando /axyn)\n", path)
	return exitOK
}

// excludeConfig keeps a new opencode.json out of `git status` (through .git/info/exclude,
// local to this clone) so the engine's clean-tree check does not stop the first /axyn.
// A file the project already tracks is left alone: its change is the user's to commit.
func excludeConfig(dir string) string {
	s := &mcpServer{dir: dir}
	if _, err := s.git("ls-files", "--error-unmatch", "opencode.json"); err == nil {
		return "axyn: opencode.json já é versionado; faça commit da mudança antes do /axyn"
	}
	path, err := s.git("rev-parse", "--git-path", "info/exclude")
	if err != nil {
		return ""
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	if data, _ := os.ReadFile(path); bytes.Contains(data, []byte("\n/opencode.json\n")) {
		return ""
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ""
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	_, _ = f.WriteString("\n/opencode.json\n")
	return ""
}
