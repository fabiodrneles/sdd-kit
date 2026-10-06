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
	exitUsage = 3
)

const usage = `axyn: o sdd-kit no opencode, com o modelo que você tiver

Uso:
  axyn version        mostra a versão
  axyn gate [--base REF] [--max-lines N] [--ci CMD] [--protect CAMINHO]
                      portões do diff: CI, teste afrouxado, caminhos protegidos e tamanho
  axyn mcp            servidor MCP (stdio) para o opencode: axyn_plan, axyn_next, axyn_gate, axyn_ship
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
	case "mcp":
		return runMCP(os.Stdin, stdout, stderr)
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(stdout, usage)
		return exitOK
	default:
		_, _ = fmt.Fprintf(stderr, "axyn: comando desconhecido: %s\n\n%s", args[0], usage)
		return exitUsage
	}
}
