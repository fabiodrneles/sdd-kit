# 021 — axyn: o sdd-kit no terminal, com o modelo que o usuário tiver

- **Prioridade:** P1
- **Status:** Draft — proposta ao dono em 2026-10-06
- **Código afetado:** novo repositório `fabiodrneles/axyn` (Go), `template/common/scripts/` (o motor atual, como referência de comportamento)
- **Resolve:** quem não pode pagar o Claude Code não consegue usar o sdd-kit; os modelos gratuitos são úteis para escrever código, mas erram mais e não seguem processo

## Contexto

O sdd-kit hoje depende do Claude Code. O opencode, de código aberto, já fala com dezenas de provedores, inclusive gratuitos, mas não tem processo: nada impede o modelo de afrouxar um teste ou quebrar o build. No T63 (#249), até o Claude afrouxou testes sozinho para o CI passar; um modelo gratuito fará isso mais vezes.

A divisão do axyn: **o motor faz tudo o que é determinístico** (estado, specs, tickets, worktree, CI, PR, checkpoint, merge, portões de qualidade), **a LLM só escreve o código do ticket**, num worktree isolado, a partir do pacote dele, e o motor decide se o diff entra.

## Decisões do dono (2026-10-06)

- **D13** O axyn começa usando o opencode como agente; uma interface própria fica para depois, se precisar.
- **D14** O usuário traz as próprias chaves (BYOK), como no opencode; o axyn usa os modelos que ele configurar.
- **D15** A escada de modelos vai dos gratuitos aos pagos que o usuário tiver; os gratuitos são testados primeiro.
- **D16** Nome: **axyn** ("archon" já é um projeto de IA para programação com mais de 20 mil estrelas); repositório em `github.com/fabiodrneles/axyn`, página no GitHub Pages, sem domínio pago.
- Tudo gratuito e de código aberto (MIT): distribuição pelo GitHub Releases, sem serviço pago.

## Requisitos funcionais

- **FR-1** `axyn` MUST ser um binário único em Go para Linux, macOS e Windows, sem `jq` nem `sh`; `axyn` sem argumentos mostra o próximo passo do projeto (o "Próximo" do checkpoint) e `axyn run '#N'` trabalha um ticket.
- **FR-2** A configuração (`~/.config/axyn/config.yaml`) MUST listar os modelos em ordem de preferência, com as chaves lidas de variáveis de ambiente, nunca gravadas no repositório; MUST aceitar ao menos OpenRouter, Gemini, Groq, Ollama (local), Anthropic e OpenAI.
- **FR-3** O axyn MUST chamar o opencode sem interação, com o modelo escolhido, num worktree do ticket, com o pacote do ticket (o mesmo do `sdd-context.sh`) e só as ferramentas da entrega.
- **FR-4** Todo diff MUST passar pelos portões antes de virar commit: `make ci` (ou o comando do projeto); nenhum teste apagado, pulado ou com `assert` a menos; nenhum arquivo protegido alterado (workflows, specs, scripts do motor, versão); tamanho do diff dentro do limite do ticket; cada AC do ticket citado num teste.
- **FR-5** Diff reprovado MUST voltar ao mesmo modelo uma vez, com o motivo; reprovado de novo, MUST subir para o próximo modelo da escada; esgotada a escada, o ticket para e o axyn diz o que faltou, sem quebrar nada.
- **FR-6** Com os portões verdes, o motor MUST fazer o commit, o push e o PR (como o `sdd-pr.sh`) e gravar no ticket o modelo, as tentativas e o custo.
- **FR-7** `axyn eval` MUST rodar a bateria de tarefas (como o `benchmark.sh`) com cada modelo configurado e gravar uma tabela de notas por tipo de tarefa, que o FR-5 usa para começar pelo modelo mais barato que passa naquele tipo.

## Requisitos não funcionais

- **NFR-1** Nenhuma dependência paga: com Ollama, o axyn MUST funcionar sem internet além do git.
- **NFR-2** Nenhum diff entra sem passar pelos portões do FR-4, qualquer que seja o modelo.
- **NFR-3** Os testes atuais dos scripts do sdd-kit são a especificação de comportamento do motor em Go.

## Critérios de aceite

- **AC-1** Dado um agente falso que apaga um teste, quando o axyn roda o ticket, então o diff é reprovado com o motivo e nada é commitado.
- **AC-2** Dados dois modelos na escada e o primeiro reprovado duas vezes, quando o axyn roda, então o segundo é chamado com o motivo, e o ticket registra as três tentativas.
- **AC-3** Dado um ticket com os portões verdes, quando o axyn roda, então há um commit, um push e um PR, e o ticket tem o modelo e o custo.
- **AC-4** Dado um modelo local (Ollama) configurado e nenhuma chave de API, quando `axyn run` roda um ticket do sdd-kit-demo, então o ticket vai de ponta a ponta ou para com o motivo, sem chamar serviço pago.
- **AC-5** Dados dois modelos e a bateria, quando `axyn eval` roda, então há uma tabela com a nota de cada modelo por tipo de tarefa.

## Fora do escopo do axyn mínimo

Interface própria de terminal, tickets em paralelo, lote adaptativo, GitLab. Entram em fases seguintes, se o mínimo provar o valor.

## Mudanças

### Não lançado
