# 017 — Relatório de entrega e de custo (`sdd-report`)

- **Prioridade:** P1
- **Status:** Done — entregue na `v1.10.0`
- **Código afetado:** `template/common/scripts/sdd-report.sh` (novo), `template/common/CLAUDE.md`, `CLAUDE.md`
- **Resolve:** não sabemos quanto cada ticket custa em tokens nem onde o gasto está; sem número, não dá para provar economia

## Contexto

O Claude Code grava, para cada chamada à LLM, o uso de tokens no arquivo da sessão (`~/.claude/projects/**/<sessão>.jsonl`, campo `message.usage`). Medição de 2026-10-06, numa sessão longa (714 chamadas, sem duplicatas por `message.id`):

| | Tokens |
|---|---|
| Contexto relido do cache | 266.000.000 |
| Contexto novo gravado no cache | 1.750.000 |
| Texto gerado | 381.000 |
| Contexto médio por chamada | 375.000 (as 10 primeiras: 72.000) |
| Saída média por chamada | 533 |

Cada chamada relê a conversa inteira; a sessão cresce, e o custo total cresce com o quadrado do número de chamadas. Quase todo o gasto é **reler**, não **gerar**. Na mesma sessão houve cerca de 140 despertares sem mensagem do dono (avisos de assinatura de PR, fim de tarefas em segundo plano), e cada um relê a conversa inteira. Este relatório transforma isso em número por ticket e por fase, sem LLM, para que toda alegação de economia seja medida (ROADMAP, item 48).

## Requisitos funcionais

- **FR-1** `sdd-report.sh tokens [--session ARQ | --since DATA]` MUST somar, sem LLM, as chamadas dos arquivos de sessão do Claude Code (sem duplicatas por `message.id`): chamadas, tokens relidos, gravados, de entrada e gerados, contexto médio e máximo por chamada.
- **FR-2** `sdd-report.sh ticket '#N'` MUST atribuir ao ticket as chamadas feitas entre o primeiro commit da branch do PR que fecha a issue e o último push dele, e imprimir os mesmos totais do FR-1, mais o tempo e se o CI do PR ficou verde na primeira rodada.
- **FR-3** `sdd-report.sh phase '#ÉPICO'` MUST juntar os tickets do épico numa tabela (uma linha por ticket e o total) e MUST gravá-la num único comentário do épico, editado nas rodadas seguintes, para o número sobreviver ao fim do contêiner.
- **FR-4** O relatório MUST separar os **despertares sem mensagem do dono** (avisos, fim de tarefas em segundo plano) e o custo deles, porque é gasto que não produz trabalho.
- **FR-5** O fechamento da fase (`sdd-release.sh`) SHOULD rodar o `sdd-report.sh phase` e citar o total no PR de fechamento.

## Requisitos não funcionais

- **NFR-1** O relatório MUST NOT chamar a LLM: só leitura local dos arquivos de sessão (`jq`) e REST do GitHub (`gh api`).
- **NFR-2** Sem arquivo de sessão (outro agente, outro contêiner), o relatório MUST dizer que não há dado de tokens e mostrar o resto (tempo, CI), nunca um número inventado.

## Critérios de aceite

- **AC-1** Dado um arquivo de sessão de teste com chamadas repetidas por `message.id`, quando `sdd-report.sh tokens --session` roda, então os totais contam cada chamada uma vez e batem com os valores do arquivo.
- **AC-2** Dado um ticket com PR e commits em horários conhecidos, quando `sdd-report.sh ticket` roda, então só as chamadas dentro da janela entram no total.
- **AC-3** Dado um épico com dois tickets, quando `sdd-report.sh phase` roda duas vezes, então o épico tem um único comentário de relatório, com uma linha por ticket e o total.
- **AC-4** Dados despertares sem mensagem do dono no arquivo de sessão, quando o relatório roda, então eles aparecem numa linha própria, com a contagem e os tokens.
- **AC-5** Sem arquivo de sessão, quando o relatório roda, então diz "sem dado de tokens" e sai com 0.

## Mudanças

### v1.10.0

- ADDED FR-1, NFR-1, NFR-2 — `sdd-report.sh tokens [--session ARQ | --since DATA]`: totais das sessões do Claude Code sem LLM, cada chamada contada uma vez por `message.id` (T50, #214).
- ADDED FR-2, FR-3 — `sdd-report.sh ticket '#N'` (totais na janela do PR que fecha a issue, tempo e CI verde na primeira rodada) e `sdd-report.sh phase '#ÉPICO'` (uma linha por ticket e o total, num único comentário do épico) (T51, #215).
- ADDED FR-4, FR-5 — despertares sem mensagem do dono (turnos sem `origin.kind` "human") numa linha própria, com contagem e tokens, e numa coluna do `phase`; o `sdd-release.sh` cita a tabela da fase no PR de fechamento (T52, #216).
