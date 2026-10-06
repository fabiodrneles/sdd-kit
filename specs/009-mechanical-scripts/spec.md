# 009 — Scripts para os passos mecânicos

- **Prioridade:** P1
- **Status:** Done — entregue na `v1.0.0`
- **Código afetado:** `template/common/scripts/`, `template/go/scripts/`, `plugins/sdd-delivery/`
- **Resolve:** #55; achados 4 e 10 do #51

## Contexto

Nos casos de uso (#51), o agente repetiu à mão passos mecânicos: esperar o CI e ler logs inteiros, editar os arquivos de status, montar o comentário "Estado da fase", criar o épico e as sub-issues, e conferir o release. Cada ciclo gastava dezenas de chamadas e milhares de tokens. Uma edição improvisada chegou a esvaziar arquivos (sdd-kit-demo `e69f70b`), e uma falha de release só apareceu depois da tag (go-release-manager#38). Princípio: **o script faz o trabalho mecânico e imprime uma linha por resultado; o agente lê a linha e decide.**

## Requisitos funcionais

- **FR-1** `sdd-ci.sh [REF]` MUST esperar os checks e os status de commit (ex.: Vercel) de um commit, PR (`#N`) ou branch, imprimir uma linha por check e, só para os que falharam, o passo e o fim do log (ou o link do status). Códigos: 0 verde, 1 falhou, 2 tempo esgotado, 3 uso.
- **FR-2** `sdd-mark.sh decide Dn=x ...` MUST marcar as decisões como respondidas em `specs/ANALYSIS.md`. Quando nenhuma ficar em aberto, MUST mover as specs `Draft` para `Approved` (cabeçalho e índice) e marcar a Fase 0 do ROADMAP.
- **FR-3** `sdd-mark.sh close vX.Y.Z` MUST marcar as tarefas da fase atual (a primeira com tarefa aberta, ou a que já aponta para `vX.Y.Z`), gravar `→ vX.Y.Z` no cabeçalho dela, mover as specs citadas para `Done` (ou `In Progress`, se tiverem tarefa aberta em outra fase) e abrir `## [X.Y.Z] - data` no CHANGELOG.
- **FR-4** `sdd-mark.sh` MUST preparar todas as edições antes de gravar e MUST NOT gravar nada se uma edição esvaziar um arquivo ou mudar o número de linhas além do esperado.
- **FR-5** `sdd-phase-status.sh` MUST montar o comentário "Estado da fase" (ticket → PR → CI, decisões pendentes, próximo passo); só com `--post` ele publica.
- **FR-6** `sdd-epic.sh FASE` MUST criar o épico da fase do ROADMAP e os tickets como sub-issues; `--dry-run` MUST só mostrar o plano.
- **FR-7** `sdd-release-check.sh pre vX.Y.Z` (template Go) MUST simular o release num clone descartável com a tag temporária; `post vX.Y.Z` MUST conferir os assets, os checksums e o `go install` da release publicada.
- **FR-8** A skill e os comandos `/sdd-*` MUST indicar os scripts no lugar dos passos manuais.

## Critérios de aceite

- **AC-1** Dado um repositório com D1 e D2 em aberto, quando `sdd-mark.sh decide D1=a` roda, então D1 fica respondida e as specs continuam `Draft`; com `D2=b`, as specs vão para `Approved` e a Fase 0 é marcada.
- **AC-2** Dada uma decisão inexistente (`D9=a`), quando `decide` roda, então sai com erro e nenhum arquivo muda.
- **AC-3** Dada a fase `→ v0.1.0` com tarefas citando a spec 001, quando `sdd-mark.sh close v0.1.0` roda, então as tarefas são marcadas, a 001 vai para `Done` e o CHANGELOG ganha `## [0.1.0] - data`; rodar de novo não muda nada.
- **AC-4** Dado o ROADMAP, quando `sdd-epic.sh --dry-run 1` roda, então imprime o épico e uma linha por tarefa, sem chamar o GitHub.
- **AC-5** Dado uso inválido, quando `sdd-ci.sh`, `sdd-phase-status.sh` ou `sdd-release-check.sh` rodam, então saem com o código de uso, sem chamar o GitHub.
- **AC-6** Dados os scripts, quando o CI roda, então passam no `shellcheck -s sh` e são distribuídos pela adoção (template).
- **AC-7** Dado um commit com os checks verdes e um status de commit vermelho, quando `sdd-ci.sh` roda, então lista `FALHA <contexto> (status)` com o link e sai com 1.

## Mudanças

- MODIFIED FR-3: a fase é a da tarefa aberta, e o fechamento grava a versão do go-release-manager no cabeçalho (#198).
- MODIFIED FR-1: status de commit também contam; ADDED AC-7 (#66).
