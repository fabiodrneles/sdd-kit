# 021 — axyn: o sdd-kit no terminal, com o modelo que o usuário tiver

- **Prioridade:** P1
- **Status:** Approved — aprovada pelo dono em 2026-10-06
- **Código afetado:** `axyn/` (novo, Go, neste repositório até funcionar; depois `fabiodrneles/axyn`), `template/web/` (novo), `template/common/scripts/` (o motor atual, como referência de comportamento)
- **Resolve:** quem não pode pagar o Claude Code não consegue usar o sdd-kit; os modelos gratuitos são úteis para escrever código, mas erram mais e não seguem processo

## Contexto

O sdd-kit hoje depende do Claude Code. O opencode, de código aberto, já fala com dezenas de provedores, inclusive gratuitos, mas não tem processo: nada impede o modelo de afrouxar um teste ou quebrar o build. No T63 (#249), até o Claude afrouxou testes sozinho para o CI passar; um modelo gratuito fará isso mais vezes.

A divisão do axyn: **o motor faz tudo o que é determinístico** (estado, specs, tickets, worktree, CI, PR, checkpoint, merge, portões de qualidade), **a LLM só escreve o código do ticket**, num worktree isolado, a partir do pacote dele, e o motor decide se o diff entra.

## Decisões do dono (2026-10-06)

- **D13** O axyn roda **dentro do opencode**: a instalação configura no projeto o servidor MCP `axyn`, os agentes (planejamento e código) e o comando `/axyn`; o opencode é a interface, e o axyn é o cérebro. Uma interface própria fica para depois, se precisar.
- **D14** O usuário traz as próprias chaves (BYOK), como no opencode; o axyn usa os modelos que ele configurar.
- **D15** A escada de modelos vai dos gratuitos aos pagos que o usuário tiver; os gratuitos são testados primeiro.
- **D16** Nome: **axyn** ("archon" já é um projeto de IA para programação com mais de 20 mil estrelas); código em `axyn/` neste repositório até funcionar, depois `github.com/fabiodrneles/axyn`; página no GitHub Pages, sem domínio pago.
- **D17** O teste de aceitação é o do dono: no opencode, com um modelo gratuito, instalar o axyn com um comando, pedir uma landing page num repositório clonado e receber tudo construído, com PRs e CI verde.
- Tudo gratuito e de código aberto (MIT): distribuição pelo GitHub Releases, sem serviço pago.

## Requisitos funcionais

- **FR-1** `axyn` MUST ser um binário único em Go para Linux, macOS e Windows, sem `jq` nem `sh`, instalado por um comando (`curl ... | sh`, ou `go install`), com `axyn version`.
- **FR-2** `axyn install` MUST configurar o opencode do projeto: o servidor MCP `axyn` no `opencode.json`, o agente `axyn-plan` (escreve spec e tickets), o agente `axyn-code` (só as ferramentas de código; escreve o código de um ticket) e o comando `/axyn PEDIDO`; sem apagar a configuração que o usuário já tem.
- **FR-3** O servidor MCP (`axyn mcp`) MUST oferecer ao opencode as ferramentas determinísticas: `axyn_plan` (valida e grava a spec e os tickets), `axyn_next` (o próximo ticket e o pacote dele), `axyn_gate` (os portões do FR-5 sobre o diff atual) e `axyn_ship` (commit, push e PR, só com o portão verde para aquele diff).
- **FR-4** Num repositório vazio ou sem CI, o axyn MUST preparar o projeto antes do primeiro ticket, com o template da stack (o template `web` para uma landing page: HTML validado e links verificados), para os portões terem o que verificar.
- **FR-5** Todo diff MUST passar pelos portões antes do commit: o CI do projeto (`make ci`); nenhum teste apagado, pulado ou com asserção a menos; nenhum arquivo protegido alterado (workflows, specs aprovadas, configuração do axyn, versão); tamanho do diff dentro do limite do ticket; cada AC do ticket citado num teste.
- **FR-6** Escada de modelos: a configuração (`~/.config/axyn/config.yaml`) MUST listar os modelos em ordem, com as chaves só em variáveis de ambiente, nunca no repositório (OpenRouter, Gemini, Groq, Ollama, Anthropic, OpenAI e o que o opencode aceitar); diff reprovado volta ao mesmo modelo uma vez, com o motivo; reprovado de novo, o `axyn_gate` indica o próximo modelo; esgotada a escada, o ticket para com o motivo, sem quebrar nada.
- **FR-7** Cada ticket entregue MUST registrar o modelo, as tentativas e o custo; `axyn eval` MUST dar a nota de cada modelo configurado por tipo de tarefa, com a bateria do `benchmark.sh`.

## Requisitos não funcionais

- **NFR-1** Nenhuma dependência paga: com Ollama, o axyn MUST funcionar sem internet além do git.
- **NFR-2** Nenhum diff entra sem passar pelos portões do FR-4, qualquer que seja o modelo.
- **NFR-3** Os testes atuais dos scripts do sdd-kit são a especificação de comportamento do motor em Go.

## Critérios de aceite

- **AC-1** Dado um agente falso que apaga um teste, quando o `axyn_gate` roda, então reprova com o motivo, e o `axyn_ship` recusa o commit.
- **AC-2** Dados dois modelos na escada e o primeiro reprovado duas vezes, quando o `axyn_gate` roda, então indica o segundo modelo, e o ticket registra as tentativas.
- **AC-3** Dado um projeto com configuração própria do opencode, quando `axyn install` roda, então ela continua lá, com o servidor MCP, os agentes e o comando `/axyn` acrescentados.
- **AC-4** Dado um repositório vazio, quando o axyn prepara o projeto web, então o `make ci` dele roda e reprova um HTML inválido.
- **AC-5** (D17) Dado o opencode com um modelo gratuito e o axyn instalado num repositório clonado, quando o usuário pede `/axyn crie uma landing page`, então há uma spec, tickets e um PR por ticket com CI verde, ou o axyn para com o motivo, sem quebrar o projeto; no CI, o mesmo fluxo roda com um agente falso.

## Fora do escopo do axyn mínimo

Interface própria de terminal, tickets em paralelo, lote adaptativo, GitLab. Entram em fases seguintes, se o mínimo provar o valor.

## Mudanças

### Não lançado
