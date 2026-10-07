// Command axyn is the sdd-kit engine inside opencode: the deterministic part of the
// process (state, gates, delivery) runs here, and the LLM only writes code (spec 021).
package main

import (
	"fmt"
	"io"
	"os"
)

// version is set at release time through -ldflags "-X main.version=...".
var version = "dev"

const (
	exitOK    = 0
	exitFail  = 2
	exitUsage = 3
)

const usage = `axyn: o sdd-kit no opencode, com o modelo que você tiver

Uso:
  axyn version        mostra a versão
  axyn gate [--base REF] [--max-lines N] [--ci CMD] [--protect CAMINHO]
                      portões do diff: CI, teste afrouxado, caminhos protegidos e tamanho
  axyn doctor [--dir DIR]
                      confere o que o repositório precisa (GitHub, Actions, merge automático,
                      ferramentas), uma linha por item, com o comando de cada coisa que falta
  axyn setup [--protect-main] [--dir DIR]
                      configura pelo gh o que dá: identidade do git, repositório no GitHub,
                      Actions com escrita, merge automático e, opcional, a proteção da main
  axyn init [--stack S] [--dir DIR]
                      prepara o projeto com o template da stack (Makefile com o make ci, CI do
                      GitHub e lint), sem alterar arquivo existente; o axyn run faz isso sozinho
  axyn install [--dir DIR]
                      configura o opencode do projeto: servidor MCP, agentes e o comando /axyn
  axyn mcp            servidor MCP (stdio) para o opencode: axyn_run, axyn_status e as ferramentas do laço
  axyn run [--wait] [--resume] [--model M] "PEDIDO"
                      o motor conduz o laço: planeja, escolhe o ticket, chama o opencode run para o
                      código, portões, recuperação e entrega; em segundo plano, a menos de --wait
  axyn status [ID]    andamento de uma execução: ticket, portões, modelo e tentativas
  axyn help         mostra esta ajuda
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stdout, usage)
		return exitOK
	}
	switch args[0] {
	case "version", "--version":
		_, _ = fmt.Fprintf(stdout, "axyn %s\n", version)
		return exitOK
	case "gate":
		return runGate(args[1:], stdout, stderr)
	case "doctor":
		return runSetupCmd(args[1:], stdout, stderr, false)
	case "setup":
		return runSetupCmd(args[1:], stdout, stderr, true)
	case "init":
		return runInitCmd(args[1:], stdout, stderr)
	case "install":
		return runInstall(args[1:], stdout, stderr)
	case "run":
		return runRunCmd(args[1:], stdout, stderr)
	case "status":
		return runStatusCmd(args[1:], stdout, stderr)
	case "mcp":
		return runMCP(args[1:], os.Stdin, stdout, stderr)
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(stdout, usage)
		return exitOK
	default:
		_, _ = fmt.Fprintf(stderr, "axyn: comando desconhecido: %s\n\n%s", args[0], usage)
		return exitUsage
	}
}
