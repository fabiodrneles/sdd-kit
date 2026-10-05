# 015 — Motor orientado a eventos

- **Prioridade:** P1
- **Status:** Draft
- **Código afetado:** `template/common/scripts/`, `template/common/.github/workflows/`, `template/common/CLAUDE.md`, `plugins/sdd-delivery/`
- **Resolve:** cada evento de PR (CI, merge, conflito) acorda a sessão do agente, e cada despertar recarrega a conversa inteira; esperar CI e merges custa tokens e contexto de LLM sem precisar de julgamento

## Contexto

Na sessão de 2026-10-05, só ler notificações de PR e cancelar assinaturas custou seis rodadas de contexto completo, e esperar merges exigiu laços escritos à mão. Quase tudo o que acontece entre um PR aberto e a release é mecânico: atualizar o checkpoint, resumir um CI vermelho, abrir o PR de fechamento, disparar a tag, trazer a `main` para os PRs abertos. Princípio: **o GitHub Actions reage aos eventos sem LLM; o agente só entra quando há julgamento a fazer (código, causa raiz, conflito real, revisão).**

## Requisitos funcionais

- **FR-1** `scripts/sdd-wait.sh` MUST esperar, sem LLM, até uma condição valer: PR mergeado ou fechado, CI de um SHA concluído, issue fechada; com intervalo e tempo máximo configuráveis, uma linha de saída no fim e código de saída que diga se a condição valeu, falhou ou expirou.
- **FR-2** Quando um PR de ticket é mergeado, um workflow MUST atualizar o checkpoint do épico aberto (feito: o PR; próximo: o próximo ticket aberto do épico) sem LLM.
- **FR-3** Quando o CI de um PR falha, um workflow MUST comentar no PR um único resumo (uma linha por check e só o fim do log das falhas, como o `sdd-ci.sh`), editando o mesmo comentário nas rodadas seguintes.
- **FR-4** Quando o último ticket aberto do épico é fechado, um workflow MUST abrir o PR de fechamento da versão da fase com o `sdd-release.sh`.
- **FR-5** Quando o PR de fechamento (`chore/release-vX.Y.Z`) é mergeado, um workflow MUST disparar a release da versão (o `sdd-release.sh --tag`).
- **FR-6** Quando a `main` muda, um workflow MUST atualizar a branch dos PRs abertos que estão atrás dela e não têm conflito (REST `update-branch`), e MUST comentar uma vez nos que têm conflito real.
- **FR-7** O `CLAUDE.md` do template MUST mandar o agente não esperar nem acompanhar eventos de PR: abrir o PR, gravar o checkpoint e encerrar a resposta; o motor faz o resto, e o agente volta só quando há julgamento a fazer.

## Requisitos não funcionais

- **NFR-1** Os workflows MUST usar só o `GITHUB_TOKEN` com as permissões mínimas de cada um, sem segredos de LLM, e MUST rodar só para PRs do próprio repositório (sem `pull_request_target` sobre código de fork).
- **NFR-2** Cada workflow MUST ser idempotente (rodar duas vezes não duplica comentário, PR nem tag) e MUST poder ser desligado por uma variável do repositório (`SDD_ENGINE=off`).

## Critérios de aceite

- **AC-1** Dado um PR aberto, quando `sdd-wait.sh pr-merged '#N'` roda e o PR é mergeado, então sai com 0; fechado sem merge, com 1; esgotado o tempo, com 2.
- **AC-2** Dado um épico com checkpoint, quando um PR de ticket é mergeado, então o checkpoint passa a citar o PR como feito e o próximo ticket aberto como próximo.
- **AC-3** Dado um PR com um check vermelho, quando o CI termina, então o PR tem um comentário de resumo com o check e o fim do log; numa segunda falha, o mesmo comentário é editado.
- **AC-4** Dado um épico com um último ticket aberto, quando ele é fechado, então um PR `chore/release-vX.Y.Z` é aberto com o CHANGELOG e as versões da fase.
- **AC-5** Dado o PR de fechamento, quando é mergeado, então a release `vX.Y.Z` é disparada uma única vez.
- **AC-6** Dados dois PRs abertos atrás da `main`, um limpo e um com conflito, quando a `main` muda, então o limpo é atualizado e o com conflito recebe um único comentário.
- **AC-7** Dado `SDD_ENGINE=off`, quando qualquer um dos eventos acima acontece, então nenhum workflow escreve nada.
