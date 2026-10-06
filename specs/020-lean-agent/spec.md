# 020 — Agente enxuto do relé

- **Prioridade:** P1
- **Status:** Approved — pedida pelo dono em 2026-10-06
- **Código afetado:** `template/common/scripts/sdd-relay.sh`, `template/common/scripts/sdd-report.sh`, `template/common/scripts/sdd-context.sh`, `README.md`, `README.en.md`, `docs/assets/`
- **Resolve:** cerca de 80% de cada chamada de um agente do relé é custo fixo (prompt, ferramentas, skills, servidores MCP), relido a cada chamada

## Contexto

Medição de 2026-10-06, com `claude -p` na raiz do repositório e só "Responda só: ok": o agente padrão do relé começa com 37.008 tokens de contexto; com só as ferramentas da entrega (`--tools Bash Read Edit Write Grep Glob`), sem skills (`--disable-slash-commands`) e sem servidores MCP (`--strict-mcp-config`), começa com 10.972, 70% menos. Os agentes da Fase 13 terminaram cada ticket com cerca de 46 mil tokens de contexto por chamada, então o custo fixo era a maior parte do ticket. O agente do relé fala com o GitHub pelo `gh` e trabalha a partir do pacote do ticket, sem precisar das skills nem do servidor MCP.

## Requisitos funcionais

- **FR-1** O agente padrão do relé MUST rodar só com as ferramentas da entrega, sem skills e sem servidores MCP; `SDD_AGENT_TOOLS` MUST trocar a lista de ferramentas, e `SDD_AGENT_CMD` continua trocando o agente inteiro.
- **FR-2** O `sdd-report.sh` MUST mostrar o contexto inicial de cada sessão (a primeira chamada), para medir o custo fixo separado do trabalho.
- **FR-3** O `sdd-context.sh` MUST incluir no pacote os arquivos de teste que citam os AC do ticket, para o agente não precisar procurá-los.
- **FR-5** O relé MUST escolher o modelo do agente por ticket: o label `modelo:NOME` da issue, senão `SDD_AGENT_MODEL`, senão o padrão do agente; o custo gravado na issue MUST dizer o modelo, para comparar preço além de tokens.
- **FR-6** `scripts/benchmark.sh` MUST medir, sem LLM na medição, as mesmas tarefas num projeto real com e sem sdd-kit: sem sdd-kit, uma cópia sem os arquivos do kit e uma sessão comum com todas as tarefas; com sdd-kit, um agente enxuto por tarefa, com o pacote dela. MUST relatar tokens, chamadas, tempo, `make ci` e testes novos de cada lado.
- **FR-7** O pacote MUST trazer as assinaturas dos arquivos prováveis em sh, Go, Python, JS/TS e Rust, com a linha de cada uma, e o `sdd-context.sh --task ARQ` MUST montar o mesmo pacote a partir de um arquivo de tarefa, sem GitHub; o `benchmark.sh` MUST usá-lo.
- **FR-4** O fechamento da Fase 14 MUST comparar o custo por ticket com o da Fase 13 (agente padrão) e o da Fase 12 (sem relé), e atualizar o gráfico do README com o número medido.

## Critérios de aceite

- **AC-1** Dado o agente padrão, quando o relé o chama, então o comando tem `--tools`, `--disable-slash-commands` e `--strict-mcp-config`, e `SDD_AGENT_TOOLS` troca a lista.
- **AC-2** Dada uma sessão com várias chamadas, quando `sdd-report.sh tokens` roda, então a saída tem o contexto da primeira chamada.
- **AC-3** Dado um ticket que cita `018 AC-2` e um teste que cita "018 AC-2", quando `sdd-context.sh` roda, então o pacote lista esse teste.
- **AC-5** Dado um ticket com o label `modelo:haiku`, quando o relé chama o agente, então o comando tem `--model haiku` e o custo gravado diz o modelo; sem label, usa `SDD_AGENT_MODEL`.
- **AC-6** Dado o `benchmark.sh --dry-run`, quando roda, então mostra as duas cópias, os arquivos do kit removidos do lado sem sdd-kit e um pedido por tarefa do lado com sdd-kit, sem chamar o agente.
- **AC-7** Dado um arquivo de tarefa que lista um `.go`, um `.py` e um `.ts`, quando `sdd-context.sh --task` roda, então o pacote tem as funções e os tipos de cada um, com a linha, e a entrega sem PR.
- **AC-4** Dada a Fase 14 feita pelo relé, quando o `sdd-report.sh phase --compare` roda contra a Fase 13, então mostra a diferença por ticket e por chamada.

## Mudanças

### Não lançado

- ADDED FR-1 — agente padrão do relé só com as ferramentas da entrega (`SDD_AGENT_TOOLS`, padrão `Bash Read Edit Write Grep Glob`), sem skills e sem servidores MCP: contexto inicial de 37 mil para 11 mil tokens; o relé roda de uma cópia dos scripts, para o agente trocar de branch sem quebrá-lo (T62, #248).
- ADDED FR-5 — modelo por ticket: o label `modelo:NOME` da issue, senão `SDD_AGENT_MODEL`; o agente padrão recebe `--model NOME`, e o custo gravado na issue diz o modelo (T66, #251).
- FIXED FR-1 — o agente não herda as variáveis do relé (`SDD_SCRIPTS_DIR`, `SDD_RELAY_SELF`), que quebravam o `make ci` dele; o pedido manda começar da `origin/main` e proíbe afrouxar teste (achados do T63, #249).
- ADDED FR-6 — `scripts/benchmark.sh`: as mesmas tarefas (`docs/benchmark/tasks.md`) num projeto real com e sem sdd-kit, medidas pelos arquivos de sessão (T67, #257).
- ADDED FR-7 — o pacote traz as assinaturas de sh, Go, Python, JS/TS e Rust com a linha; `sdd-context.sh --task ARQ` monta o pacote sem GitHub, e o `benchmark.sh` o usa (T68, #259).
