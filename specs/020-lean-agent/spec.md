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
- **FR-4** O fechamento da Fase 14 MUST comparar o custo por ticket com o da Fase 13 (agente padrão) e o da Fase 12 (sem relé), e atualizar o gráfico do README com o número medido.

## Critérios de aceite

- **AC-1** Dado o agente padrão, quando o relé o chama, então o comando tem `--tools`, `--disable-slash-commands` e `--strict-mcp-config`, e `SDD_AGENT_TOOLS` troca a lista.
- **AC-2** Dada uma sessão com várias chamadas, quando `sdd-report.sh tokens` roda, então a saída tem o contexto da primeira chamada.
- **AC-3** Dado um ticket que cita `018 AC-2` e um teste que cita "018 AC-2", quando `sdd-context.sh` roda, então o pacote lista esse teste.
- **AC-5** Dado um ticket com o label `modelo:haiku`, quando o relé chama o agente, então o comando tem `--model haiku` e o custo gravado diz o modelo; sem label, usa `SDD_AGENT_MODEL`.
- **AC-4** Dada a Fase 14 feita pelo relé, quando o `sdd-report.sh phase --compare` roda contra a Fase 13, então mostra a diferença por ticket e por chamada.

## Mudanças

### Não lançado

- ADDED FR-1 — agente padrão do relé só com as ferramentas da entrega (`SDD_AGENT_TOOLS`, padrão `Bash Read Edit Write Grep Glob`), sem skills e sem servidores MCP: contexto inicial de 37 mil para 11 mil tokens; o relé roda de uma cópia dos scripts, para o agente trocar de branch sem quebrá-lo (T62, #248).
- ADDED FR-5 — modelo por ticket: o label `modelo:NOME` da issue, senão `SDD_AGENT_MODEL`; o agente padrão recebe `--model NOME`, e o custo gravado na issue diz o modelo (T66, #251).
- FIXED FR-1 — o agente não herda as variáveis do relé (`SDD_SCRIPTS_DIR`, `SDD_RELAY_SELF`), que quebravam o `make ci` dele; o pedido manda começar da `origin/main` e proíbe afrouxar teste (achados do T63, #249).
