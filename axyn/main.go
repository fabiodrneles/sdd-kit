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
  axyn update         atualiza o axyn para a versão mais nova (rode na raiz do projeto, com o opencode fechado)
  axyn gate [--base REF] [--max-lines N] [--ci CMD] [--protect CAMINHO]
                      portões do diff: CI, teste afrouxado, caminhos protegidos e tamanho
  axyn model [ID] [--all] [--only] [--key-env VAR]
                      escolhe o modelo do axyn: sem ID, mostra os modelos gratuitos do opencode
                      numa lista numerada; com ID, troca direto (grava o config.yaml)
  axyn bench [--models A,B] [--all] [--tasks plan,code,tests,fix] [--runs N] [--parallel N]
            [--min-score N] [--ask] [--show] [--apply] [--off] [--set ETAPA=MODELO]
                      avalia os modelos gratuitos da máquina em tarefas fixas de cada etapa (plano,
                      código, testes, conserto), com notas só de verificações automáticas e veto a
                      quem enfraquece teste, e manda cada etapa para o modelo que melhor a resolve;
                      roda sozinho na primeira execução e quando a avaliação vence
  axyn release [--yes] [--tag X.Y.Z]
                      fecha uma versão do projeto: mostra a versão (go-release-manager) e o que entra,
                      e com o seu sim abre o PR de fechamento com o CHANGELOG
  axyn coverage [auto|manual] [--no-resume]
                      sem argumento, mostra a cobertura medida, o mínimo, a meta e onde falta teste;
                      auto: o axyn escreve os testes que faltam; manual: você escreve
  axyn retry [--no-resume]
                      mais 10 tentativas no ticket parado, com o mesmo modelo, que agora recebe o que
                      já falhou para não repetir
  axyn decide RESPOSTA [--no-resume]
                      responde a pergunta de uma execução parada (A, B ou a instrução entre aspas):
                      grava a decisão na spec e retoma a execução em segundo plano
  axyn history [ID] [--out ARQ]
                      grava num arquivo tudo o que uma execução fez (pedido, plano, tentativas,
                      portões, perguntas, commits, ambiente e log), sem chaves nem tokens
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
  axyn status [ID] [--watch]
                      andamento de uma execução: ticket, portões, modelo e tentativas; com --watch,
                      fica acompanhando e avisa (bipe e notificação) quando termina, para ou pergunta
  axyn help         mostra esta ajuda
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cleanOldBinary()
	if len(args) == 0 {
		_, _ = fmt.Fprint(stdout, usage)
		return exitOK
	}
	switch args[0] {
	case "update":
		return runUpdateCmd(args[1:], stdout, stderr)
	case "version", "--version":
		_, _ = fmt.Fprintf(stdout, "axyn %s\n", version)
		return exitOK
	case "gate":
		return runGate(args[1:], stdout, stderr)
	case "model":
		return runModelCmd(args[1:], stdout, stderr)
	case "bench":
		return runBenchCmd(args[1:], stdout, stderr)
	case "release":
		return runReleaseCmd(args[1:], stdout, stderr)
	case "retry":
		return runRetryCmd(args[1:], stdout, stderr)
	case "coverage":
		return runCoverageCmd(args[1:], stdout, stderr)
	case "decide":
		return runDecideCmd(args[1:], stdout, stderr)
	case "history":
		return runHistoryCmd(args[1:], stdout, stderr)
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
