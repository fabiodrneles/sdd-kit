# 014 — Checkpoints de retomada

- **Prioridade:** P1
- **Status:** Done — entregue na `v1.4.0`
- **Código afetado:** `template/common/scripts/`, `template/common/.claude/settings.json`, `template/<lang>/.claude/hooks/`, `template/common/CLAUDE.md`
- **Resolve:** a sessão pode acabar sem aviso (limite de uso, contexto); hoje o dono precisa pedir um handoff e colar um prompt para continuar

## Contexto

O "Estado da fase" do épico só é escrito nos marcos, e quando o agente lembra. Se a sessão acaba no meio, a próxima não sabe de onde continuar sem o dono explicar. Princípio: **o dono fecha a sessão quando quiser e abre outra; ela continua sozinha do último checkpoint.**

## Requisitos funcionais

- **FR-1** `scripts/sdd-checkpoint.sh save "feito" "próximo" ["bloqueia"]` MUST manter um único comentário de checkpoint no épico aberto (marcado com `<!-- sdd-checkpoint -->`), com branch, commit, alterações locais, commits não enviados, PRs abertos, feito, próximo e bloqueio.
- **FR-2** `show` MUST imprimir o checkpoint, e o hook de início de sessão de cada linguagem MUST rodá-lo, para ele entrar no contexto da sessão nova. Sem épico, sem checkpoint ou sem `gh`, o script MUST sair com 0 e uma mensagem, sem travar a sessão.
- **FR-3** `auto` MUST atualizar só o estado do git, mantendo o feito e o próximo do último `save`, e MUST não escrever nada quando o checkpoint não mudou; o template MUST rodá-lo num hook `Stop` (a cada resposta).
- **FR-4** O `CLAUDE.md` do template MUST mandar retomar do checkpoint sem esperar instrução e gravar um checkpoint depois de cada passo.

## Critérios de aceite

- **AC-1** Dado um épico aberto sem checkpoint, quando `save` roda, então um comentário com a marca, a branch, o feito e o próximo é criado; rodando de novo, o mesmo comentário é editado.
- **AC-2** Dado um checkpoint, quando `show` roda, então ele é impresso; sem épico aberto, `show` e `save` saem com 0 e avisam.
- **AC-3** Dado um checkpoint, quando `auto` roda sem nada mudado, então nada é escrito; com um commit novo, o comentário passa a ter o commit novo e mantém o feito e o próximo.
- **AC-4** Dados o template e as linguagens, quando o CI roda, então cada hook de início de sessão chama `sdd-resume.sh` (que mostra o checkpoint com `sdd-checkpoint.sh show`) e o `settings.json` tem o hook `Stop` com `sdd-checkpoint.sh auto`.

## Mudanças

### Não lançado

- MODIFIED FR-2 — sem épico aberto, o `sdd-resume.sh` aponta a próxima fase do ROADMAP com tarefa aberta (`sdd-next-phase.sh`), e o motor abre o épico dessa fase depois da release do fechamento: um "continue" numa sessão nova achava "nada a fazer" depois da v0.2.0 do demo (#191).

### v1.5.0

- MODIFIED FR-2 — o hook de início de sessão chama `sdd-resume.sh`: checkpoint, branch do checkpoint, PRs abertos com o CI e sub-issues do épico (#136).

### v1.4.0

- ADDED FR-1 — `sdd-checkpoint.sh save` mantém um comentário de checkpoint no épico aberto (T38, #123).
- ADDED FR-2 — `show` no hook de início de sessão de cada linguagem (T38, #123).
- ADDED FR-3 — `auto` no hook `Stop` do template, só com o estado do git (T38, #123).
- ADDED FR-4 — o `CLAUDE.md` do template retoma do checkpoint e grava um a cada passo (T38, #123).
