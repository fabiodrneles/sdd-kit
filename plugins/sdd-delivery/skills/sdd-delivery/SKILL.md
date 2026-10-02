---
name: sdd-delivery
description: >-
  Processo de entrega Spec Driven Development (SDD) ponta a ponta para qualquer repositório, com o dono decidindo e o agente executando: auditoria do repo com evidências (specs/ANALYSIS.md e decisões D1..Dn), constituição, uma spec por área (FR/NFR/AC), ROADMAP por fases e versões, épico por fase com tickets como sub-issues, uma branch e um PR por ticket, CI com gates fortes, CI vermelho tratado pela causa raiz, fase de revisão, PR de fechamento, CHANGELOG e tag. Use sempre que o usuário pedir para analisar, verificar, terminar, organizar ou profissionalizar um repositório, mesmo sem citar SDD. Gatilhos: "análise e verificação do repo", "organize as specs", "crie as specs", "gere os tickets", "crie os épicos", "um PR por issue", "uma branch por ticket", "fase de revisão", "feche a fase", "prepare a release", "trabalhe como no cv-craft", "spec driven development", "SDD", "phase plan", "ticket per branch", "one PR per issue", "audit this repo", "make this repo production ready".
---

# SDD Delivery — análise → specs → tickets → PRs → revisão → release

Processo usado para levar um repositório de protótipo a projeto profissional, com
**o dono decidindo** e **o agente executando**. É genérico: não depende de linguagem
nem de ferramenta de build, só de GitHub (issues, sub-issues, Actions) e de um alvo
local equivalente ao CI.

Regras normativas completas: [references/process.md](references/process.md).
Modelos de documentos e textos: [references/templates.md](references/templates.md).
Gates de CI/testes e lições aprendidas: [references/quality-gates.md](references/quality-gates.md).
Checklist para começar num repositório novo: [references/bootstrap-checklist.md](references/bootstrap-checklist.md).

## Visão geral

```text
descoberta → ANALYSIS.md → ⏸ decisões do dono → constituição + specs + ROADMAP
  → épico por fase → tickets (sub-issues) → branch por ticket
  → testes + código → validação local → push → PR → CI verde
  → ⏸ revisão do dono → merge (dono) → PR de fechamento → tag (dono) → release
```

`⏸` = **ponto de parada**: o agente para e espera o dono. Não avance sozinho.

## Papéis (inegociável)

- **Dono** decide: decisões em aberto, escopo, prioridades, aprovação, **merge, tags,
  releases, configurações do repositório** e qualquer ação irreversível ou visível para fora.
- **Agente** executa: análise, specs, épicos, tickets, código, testes, PRs, manter o CI verde.
- O agente **nunca**: faz merge (salvo quando o dono delega explicitamente uma rodada de
  merges: vale só para ela, segue a ordem do épico e exige CI verde em cada head); dá force-push em branch alheia; reescreve histórico
  publicado; trabalha fora das branches do seu ticket; cria tag ou release.

## Fase de descoberta (repositório novo para o agente)

Objetivo: entender o estado real antes de mudar qualquer linha de código.

1. Leia **todo** o código. Compile. Rode **todos** os comandos documentados e os casos de
   borda relevantes (entrada inválida, arquivo existente, stdin fechado, sem TTY, `--help`).
2. Confira cada promessa do README executando-a (instalação, exemplos, flags).
3. Registre tudo em `specs/ANALYSIS.md` (modelo em templates.md):
   resumo executivo; tabela "o que foi verificado" (comando → resultado); achados
   **crítico / alto / médio / baixo**, cada um com evidência (`arquivo:linha` ou saída de
   comando) e a spec que o resolve; pontos positivos a preservar; avaliação do README;
   melhorias priorizadas por fase; **decisões em aberto D1..Dn**, cada uma com opções e
   uma recomendação.
4. Separe sempre o que foi **verificado** (executado) do que é **inferido** (leitura de código).
5. Escreva `specs/constitution.md` (5–10 princípios verificáveis), uma spec por área em
   `specs/NNN-nome/spec.md`, `specs/README.md` (fluxo, convenções, índice com status) e
   `specs/ROADMAP.md` (fases → versões, tarefas `T1..Tn` citando os IDs que fecham).
   Fases padrão: **Fase 0** decisões → **Fase 1** "funcionar de verdade" `v0.1.0` →
   **Fase 2** "confiável" `v0.2.0` → **Fase 3** "profissional" `v1.0.0`.
6. Apresente ao dono uma lista **curta** das observações principais (críticas primeiro) e as
   decisões D1..Dn com recomendação, apontando para os arquivos.

**⏸ PARE.** Nenhuma implementação antes de o dono responder D1..Dn. Registre as respostas
na seção "Decisões" das specs afetadas e em ANALYSIS.md; specs passam de `Draft` para `Approved`.

## Implementação por fase

1. **Épico** por fase (labels `épico`, `fase-N`), com a ordem sugerida de revisão.
2. **Ticket** por tarefa, como **sub-issue nativa** do épico. Crie a issue **primeiro**
   (isso cria labels inexistentes) e **depois** anexe como sub-issue — criar já com o pai
   falha se alguma label ainda não existir.
   - Corpo: **Contexto / O que fazer / Critérios de aceite / Spec(s) / Épico**
     (+ "Decisão para a revisão" opcional).
   - Labels: `fase-N`, `tipo:feature|docs|ci|teste|chore` ou `bug` (label padrão do GitHub), `P1|P2|P3`.
3. **Branch** por ticket: `<tipo>/<nº-da-issue>-<descrição-curta>`, a partir da `main` ou
   empilhada na branch da fase anterior ainda não mergeada.
4. **Testes a partir dos critérios de aceite**; depois o código. Cada `AC-*` vira ao menos
   um teste. Saídas geradas → golden files.
5. **Commits** em Conventional Commits, em inglês, no imperativo (`feat: add --watch mode`);
   incompatível usa `!` e explica no corpo. Inclua as linhas de atribuição exigidas pela ferramenta.
6. **Antes de cada push:** rode a verificação local completa (ex.: `make ci`) e só envie verde;
   faça a **checagem de mutação** de cada teste novo; releia o diff de forma adversarial
   (escopo, arquivos esquecidos, segredos, saídas geradas, README × specs × código).
7. **PR por ticket.** Título em Conventional Commits. Descrição começa com
   `Closes #N · Épico #M · Spec NNN` (`Spec —` se nenhuma), com **O que muda**,
   **Como foi testado** e **Notas para a revisão**. PR empilhado declara logo após a
   linha `Closes`: `> PR empilhado sobre #NN`. Termine com o rodapé de atribuição da ferramenta.
8. **Não toque nos arquivos de status compartilhados** num PR de ticket (status das specs em
   `specs/README.md` e cabeçalhos, checkboxes do ROADMAP, entradas do CHANGELOG) — isso é do
   PR de fechamento. Exceção: o ticket que cria o arquivo; e o conteúdo normativo da spec,
   que **muda junto com o código** quando a implementação diverge (com entrada em "Decisões").
9. Trabalho descoberto no meio vira **ticket novo** no épico (atual ou futuro), nunca carona.
10. Uma reestruturação coesa que não se divide sem estados quebrados MAY ser **um PR para a
    fase inteira**, listando as tarefas que fecha.
11. Inscreva-se na atividade de cada PR aberto; você é responsável por ele até ficar verde.

## CI vermelho

1. Leia o log exato do job (check runs / job logs). Reproduza localmente quando possível.
2. Ache a **causa raiz**, corrija, rode a verificação local, envie.
3. **Proibido:** chamar de "flake" sem evidência; pular, desativar ou enfraquecer testes ou
   gates (cobertura, lint); commit vazio para "re-rodar".
4. Se a correção sai do escopo mas é **pré-condição** para o PR ficar verde (ex.: linter
   incompatível com a nova versão da linguagem), ela entra no mesmo PR, explicada em "O que muda".

## Fase de revisão

Quando todos os PRs da fase estão abertos e **verdes**:

1. Verifique **conflitos par a par** entre as branches da fase com simulação real
   (`git merge-tree --write-tree origin/a origin/b`) — não adivinhe. Documente nos PRs
   afetados cada conflito e como resolvê-lo, e a ordem de merge sugerida.
2. Avise o dono: lista de PRs na ordem do épico, o que foi verificado e o que não foi.

**⏸ PARE.** O dono revisa na ordem do épico. O agente responde comentários; corrige pedidos
pequenos; para mudanças grandes ou de design, **propõe** no comentário e só implementa após
concordância. Merge é do dono: PRs de fase com outros empilhados → **merge commit**
(os empilhados são redirecionados para a `main` sem rebase); demais PRs MAY usar squash.
Depois que um PR base é mergeado, confira se o empilhado continua só com o seu ticket e verde.

## Fechamento de fase

1. Com todos os PRs mergeados, abra o **PR de fechamento**: status das specs
   (`specs/README.md` + cabeçalhos + seção "Estado atual"), checkboxes do ROADMAP, `CHANGELOG.md`
   (Keep a Changelog: `[Unreleased]` → `[X.Y.Z] - AAAA-MM-DD`). Checklist em templates.md.
2. **⏸ PARE.** O dono mergeia e cria a tag `vX.Y.Z` (SemVer). O workflow de release roda o CI
   completo antes de publicar binários e checksums.
3. O épico fecha quando a release está publicada.

## Comunicação

- Atualizações **curtas**: feito / falta / bloqueia.
- Achados vão para o **repositório** (specs, issues, PRs); o chat aponta para eles.
- Diga sempre o que foi **verificado** e o que **não** foi ("testado no Linux; Windows só no CI").
- Se o ambiente bloquear algo (proxy, rede, permissão), diga isso — nunca relate sucesso
  que não aconteceu.
- Mensagem nova do dono no meio de uma tarefa: trate-a e continue a tarefa em curso.
- Trabalho paralelo pedido pelo dono: dispare agentes, mas **verifique a saída deles**
  antes de usar ou publicar.
- Escreva no idioma do dono (specs, issues, PRs); commits e código em inglês.

## Retomada e economia de uso

O contexto pode acabar a qualquer momento: compactação, sessão nova ou limite de uso. Trabalhe de modo que o **repositório** baste para continuar:

- Crie o ticket ao começar a tarefa e abra o PR assim que ela passar na verificação local. Pedido novo do dono que não cabe na tarefa em curso vira ticket **na hora**.
- Mantenha no épico um comentário **"Estado da fase"**, atualizado a cada marco: PRs e CI, decisões, conflitos previstos, próximo passo.
- **Ao retomar:** leia o comentário de estado mais recente do épico, os PRs e as issues abertas e o `CLAUDE.md`, e continue do próximo passo.
- **No repositório:** um `CLAUDE.md` (mapa do código, comandos, convenções, armadilhas) e um hook de início de sessão que instale as ferramentas do CI na web.
- **Economia:**
  - leia trechos (`sed -n`, `grep -n`) e não releia;
  - no CI, só o resumo das conclusões, e o fim do log quando falhar;
  - valide num comando só;
  - subagentes só para buscas amplas;
  - chat curto, detalhes nos PRs.

## Scripts que poupam passos (spec 009)

Quando o repositório tem os scripts do template, chame-os em vez de fazer os passos à mão. A saída é curta: uma linha por resultado.

| Passo | Script |
|---|---|
| Esperar o CI e ler só as falhas | `sh scripts/sdd-ci.sh [SHA\|#PR\|branch]` |
| Registrar as decisões do dono | `sh scripts/sdd-mark.sh decide D1=a D2=b` |
| Arquivos de status do fechamento | `sh scripts/sdd-mark.sh close vX.Y.Z` |
| Comentário "Estado da fase" | `sh scripts/sdd-phase-status.sh [--post]` |
| Épico e sub-issues de uma fase | `sh scripts/sdd-epic.sh [--dry-run] N` |
| Release Go antes e depois da tag | `sh scripts/sdd-release-check.sh pre\|post vX.Y.Z` |

## Operações de GitHub usadas

Funciona com o GitHub MCP ou com `gh`:

| Operação | GitHub MCP | `gh` |
|---|---|---|
| Criar issue / épico | `issue_write` (create) | `gh issue create` |
| Anexar sub-issue | `sub_issue_write` (add) | `gh api repos/O/R/issues/EPIC/sub_issues -F sub_issue_id=<id>` |
| Criar PR | `create_pull_request` (procure antes o template do repo) | `gh pr create` |
| Acompanhar PR | `subscribe_pr_activity` | `gh pr checks --watch` |
| Ler CI | `pull_request_read` / `get_check_run` / `get_job_logs` | `gh run view --log-failed` |
| Responder revisão | `add_reply_to_pull_request_comment` | `gh pr comment` |

Note que o `sub_issue_id` é o **id** numérico da issue, não o número `#N`.

## Exemplo ilustrativo

No cv-craft (CLI em Go): a descoberta achou acentos corrompidos no PDF e conteúdo descartado;
D1..D6 foram respondidas; a Fase 1 virou um PR único (reestruturação coesa); a Fase 2 virou
tickets `chore/4-go-1.26`, `docs/5-contributing`, `ci/6-goreleaser`… em PRs empilhados,
cada um com `make ci` verde antes do push.
