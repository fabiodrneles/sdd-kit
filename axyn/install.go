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

const commandTemplate = `Pedido do usuário: $ARGUMENTS

Siga o processo do axyn:
1. Use o agente axyn-plan para gravar a spec e os tickets com axyn_plan.
2. Para cada ticket: chame axyn_next, use o agente axyn-code para escrever o código, rode axyn_gate e, só com o portão verde, axyn_ship.
3. Se o portão não ficar verde, pare e mostre ao usuário o motivo e o que falta.`

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
		"agent": {
			"axyn-plan": map[string]any{
				"description": "Escreve a spec e os tickets de um pedido, pelo axyn",
				"mode":        "subagent",
				"prompt":      planPrompt,
				"tools": map[string]any{
					"axyn_*": true, "read": true, "glob": true, "grep": true, "list": true,
					"write": false, "edit": false, "bash": false,
				},
			},
			"axyn-code": map[string]any{
				"description": "Escreve o código de um ticket, só com as ferramentas de código",
				"mode":        "subagent",
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
	_, _ = fmt.Fprintf(stdout, "axyn: %s configurado (servidor MCP axyn, agentes axyn-plan e axyn-code, comando /axyn)\n", path)
	return exitOK
}
