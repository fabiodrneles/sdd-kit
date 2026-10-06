# 016 — Merge percebido sem tokens

- **Prioridade:** P1
- **Status:** Approved — pedida pelo dono em 2026-10-06
- **Código afetado:** `template/common/scripts/sdd-wait.sh`, `template/common/scripts/sdd-pr.sh`, `template/common/CLAUDE.md`, `CLAUDE.md`, `plugins/sdd-delivery/`
- **Resolve:** a cada merge o dono precisa escrever "já mergeei" para a sessão continuar, gastando o tempo dele e o contexto da LLM

## Contexto

A spec 015 tirou da LLM o trabalho mecânico depois de um merge: o motor (GitHub Actions) atualiza o checkpoint, abre o PR de fechamento e dispara a release. Mas a sessão do agente não sabe que o merge aconteceu: ela não assina os eventos de PR, porque cada evento (CI, revisão, conflito) acorda a sessão e recarrega a conversa inteira. Resultado, visto na sessão de 2026-10-06: o dono mergeia e precisa escrever na conversa para o trabalho continuar.

Princípio: **quem espera é um shell, não a LLM.** Um vigia em segundo plano consulta a API do GitHub sem LLM e só termina quando um PR é mergeado; o fim do vigia acorda a sessão uma única vez, já com o motivo. Esperar não custa tokens; o único despertar é o que tem trabalho a fazer.

Limite conhecido: se o contêiner da sessão for reciclado, o vigia morre junto. Nesse caso nada se perde: o motor já gravou o checkpoint, e a próxima sessão o segue (spec 014).

## Requisitos funcionais

- **FR-1** `sdd-wait.sh merged-any` MUST esperar, sem LLM, até que qualquer um dos PRs abertos do repositório (na hora em que começa) seja mergeado ou fechado, e MUST imprimir uma única linha com o PR e o que houve (`#N mergeado: <título>` ou `#N fechado sem merge`).
- **FR-2** Os códigos de saída MUST seguir os do `sdd-wait.sh`: 0 mergeado, 1 fechado sem merge, 2 tempo esgotado, 3 uso; sem PR aberto, MUST sair com 3 e dizer que não há o que vigiar.
- **FR-3** O `sdd-pr.sh --no-wait` e o `sdd-release.sh` MUST imprimir, no fim, o comando exato do vigia, para o agente iniciá-lo em segundo plano.
- **FR-4** O `CLAUDE.md` (kit e template) MUST mandar o agente, depois de abrir ou atualizar um PR e encerrar a resposta, manter **um** vigia `sdd-wait.sh merged-any` em segundo plano, nunca assinar os eventos de PR e nunca fazer polling na conversa.
- **FR-5** Quando o vigia acorda a sessão, o agente MUST rodar o `sdd-resume.sh` e fazer o "Próximo" (spec 014), sem esperar mensagem do dono; se o Próximo depende do dono (outro merge, uma decisão), MUST religar o vigia e encerrar com uma linha. Tempo esgotado MUST religar o vigia em silêncio, sem mensagem ao dono.

## Requisitos não funcionais

- **NFR-1** Esperar MUST NOT chamar a LLM: só `gh api` (REST) com intervalo de no mínimo 60 segundos, para não gastar a cota da API.
- **NFR-2** Um merge MUST acordar a sessão no máximo uma vez, e eventos de CI e de revisão MUST NOT acordá-la.

## Critérios de aceite

- **AC-1** Dados dois PRs abertos, quando um deles é mergeado, então `sdd-wait.sh merged-any` sai com 0 e imprime `#N mergeado` com o número dele.
- **AC-2** Dado um PR aberto fechado sem merge, quando o vigia o vê, então sai com 1 e imprime `#N fechado sem merge`; esgotado o tempo, sai com 2.
- **AC-3** Sem PR aberto, quando `sdd-wait.sh merged-any` roda, então sai com 3 sem esperar.
- **AC-4** Dado um PR aberto pelo `sdd-pr.sh --no-wait`, quando ele termina, então a última linha é o comando do vigia.
- **AC-5** Dados o `CLAUDE.md` do kit e o do template, quando o `make ci` roda, então os dois têm a regra do vigia (um vigia em segundo plano, sem assinar eventos de PR).

## Mudanças

### Não lançado

- ADDED FR-1, FR-2, NFR-1 — `sdd-wait.sh merged-any`: uma consulta à lista de PRs abertos por rodada (60 s por padrão); conclui só com o PR de fato fechado, mesmo se a lista falhar (T47, #204).
- ADDED FR-3 — o `sdd-pr.sh` termina com o comando do vigia (`vigia: sh scripts/sdd-wait.sh merged-any`); o `sdd-release.sh` o herda, porque termina chamando o `sdd-pr.sh` (T48, #205).
- ADDED FR-4, FR-5, NFR-2 — `CLAUDE.md` (kit e template) e skill: um vigia `merged-any` em segundo plano antes de encerrar; no despertar, `sdd-resume.sh` e o Próximo; tempo esgotado religa em silêncio (T49, #206).
