# 019 — Relé na prática

- **Prioridade:** P1
- **Status:** Approved — pedida pelo dono em 2026-10-06
- **Código afetado:** `template/common/scripts/sdd-relay.sh`, `template/common/scripts/sdd-report.sh`, `README.md`, `README.en.md`
- **Resolve:** o relé (spec 018) só rodou com um agente falso; com o `claude -p` de verdade, o agente não tem permissão para editar nem rodar comandos, e numa sessão do Claude Code ele herda o id da sessão que o chamou, misturando o custo

## Contexto

Medição de 2026-10-06: um `claude -p` novo começa com cerca de 32 mil tokens de contexto, contra cerca de 230 mil por chamada na sessão longa da Fase 12 (6,5 milhões de tokens por ticket, sem relé). Rodado de dentro de uma sessão, o agente herda `CLAUDE_CODE_SESSION_ID` e grava no arquivo de sessão da sessão que o chamou; sem essas variáveis, ganha uma sessão e um arquivo próprios.

## Requisitos funcionais

- **FR-1** O agente padrão do relé MUST rodar sem prompts de permissão (`claude -p --permission-mode acceptEdits` com as ferramentas de que o ticket precisa) e MUST começar uma sessão própria, sem as variáveis de sessão herdadas de quem o chamou.
- **FR-2** O README MUST explicar como rodar o relé (onde, com que variáveis e como parar).
- **FR-3** O relatório MUST contar como despertar ocioso só o turno sem mensagem do dono que termina sem commit nem push; o resto é trabalho que seguiu sozinho.
- **FR-4** O fechamento da Fase 13 MUST comparar o custo por ticket com relé ao da Fase 12 (sem relé), e só então o README MAY citar a economia, com o número medido (018 NFR-2).

## Critérios de aceite

- **AC-1** Dado o agente padrão, quando o relé o chama, então o comando tem o modo de permissão e as ferramentas, e roda sem `CLAUDE_CODE_SESSION_ID` no ambiente.
- **AC-2** Dado um turno de despertar com um `git push` e outro sem, quando o relatório roda, então só o segundo conta como despertar ocioso.
- **AC-3** Dada a fase feita pelo relé, quando o `sdd-report.sh phase` roda, então a linha de comparação mostra o custo médio com relé e o da Fase 12.

## Mudanças

### Não lançado

- ADDED FR-1, FR-2 — agente padrão do relé sem prompts de permissão (`--permission-mode acceptEdits` e só os comandos da entrega) e com sessão própria (sem `CLAUDE_CODE_SESSION_ID` herdado); README com a seção do relé (T57, #236).
