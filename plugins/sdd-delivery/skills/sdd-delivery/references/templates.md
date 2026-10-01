# Modelos

Esqueletos para copiar e preencher. Substitua `<…>`. Escreva no idioma do dono.

## Sumário

- [specs/ANALYSIS.md](#specsanalysismd)
- [specs/constitution.md](#specsconstitutionmd)
- [specs/README.md](#specsreadmemd)
- [specs/NNN-nome/spec.md](#specsnnn-nomespecmd)
- [Formatos de critério de aceite](#formatos-de-critério-de-aceite)
- [specs/ROADMAP.md](#specsroadmapmd)
- [Épico](#épico)
- [Ticket](#ticket)
- [Pull request de ticket](#pull-request-de-ticket)
- [Comentário de conflitos (fase de revisão)](#comentário-de-conflitos-fase-de-revisão)
- [PR de fechamento de fase](#pr-de-fechamento-de-fase)
- [CHANGELOG.md](#changelogmd)
- [Mensagem ao dono ao fim da descoberta](#mensagem-ao-dono-ao-fim-da-descoberta)

## specs/ANALYSIS.md

````markdown
# Análise e verificação do <projeto>

> Data: <AAAA-MM-DD> · Base: commit `<sha>` (branch `<main>`)
> Método: leitura completa do código + build, lint e execução real de todos os comandos e casos de borda.

## 1. Resumo executivo

<2–3 frases: está pronto para uso real? Por quê?>

- <problema mais grave, em linguagem de usuário>
- <…>

<Uma frase sobre a base: reescrever ou corrigir/consolidar/testar?>

## 2. O que foi verificado

| Verificação | Resultado |
|---|---|
| `<comando de build>` | ✅ OK |
| `<comando documentado>` | ⚠️ <funciona com ressalva> |
| `<caso de borda: stdin fechado / arquivo existente / entrada inválida>` | ❌ <o que aconteceu> |
| Testes automatizados | <existem? passam?> |

## 3. Observações por severidade

### Críticas (bloqueiam o "funciona de verdade")

| # | Observação | Evidência | Spec |
|---|---|---|---|
| C1 | <o quê> | `<arquivo:linha>` ou saída de comando | <NNN> |

### Altas

| # | Observação | Evidência | Spec |
|---|---|---|---|
| A1 | | | |

### Médias

| # | Observação | Spec |
|---|---|---|
| M1 | | |

### Baixas / higiene

- <…>

## 4. Pontos positivos (manter)

- <…>

## 5. Avaliação do README

<Cada promessa verificada: funciona? falta o quê? promete o que não entrega?>

## 6. Melhorias recomendadas (priorizadas)

**Fase 1 — Funcionar de verdade (P0)**
1. <…> (C1)

**Fase 2 — Confiável (P1)**
1. <…>

**Fase 3 — Profissional (P2)**
1. <…>

## 7. Decisões em aberto

| # | Pergunta | Opções | Recomendação |
|---|---|---|---|
| D1 | <…> | <a / b / c> | <b, porque …> |
````

Depois das respostas, troque o título da seção 7 para "Decisões", troque a coluna
"Recomendação" por "Decisão" e registre a data da aprovação.

## specs/constitution.md

```markdown
# Constituição do <projeto>

Princípios que toda spec e todo PR devem respeitar. Mudá-los exige uma spec própria.

1. **<Nome curto>.** <Regra verificável. Ex.: "Fonte única da verdade: todos os formatos derivam do mesmo modelo e não omitem campos em silêncio.">
2. **Falhar alto, falhar cedo.** <Ex.: erros reportados todos de uma vez, com caminho do campo e exit code ≠ 0.>
3. **Determinismo.** <Mesma entrada + mesma versão ⇒ mesma saída.>
4. **Scriptável.** <Funciona sem TTY; flags para tudo; exit codes documentados.>
5. **Uma implementação por comportamento.** <Sem lógica duplicada entre interfaces.>
6. **Testado.** Todo critério de aceite tem teste automatizado; CI bloqueia merge vermelho.
7. **Dependências mínimas.** <Só as justificadas em spec.>
```

5–10 princípios, cada um verificável por teste ou revisão.

## specs/README.md

````markdown
# Specs — <projeto>

Este diretório organiza o desenvolvimento em **Spec Driven Development (SDD)**: nenhuma mudança
de comportamento entra no código sem uma spec que a descreva e critérios de aceite que a verifiquem.

## Fluxo

```text
spec.md (O QUÊ / POR QUÊ) → revisão → testes a partir dos AC → implementação → status: Done
```

1. **Especificar** — `FR-*`, `NFR-*` e `AC-*` (Dado/Quando/Então ou EARS).
2. **Resolver decisões** — antes de implementar.
3. **Testar primeiro** — cada `AC-*` vira ao menos um teste.
4. **Implementar** — o PR referencia os IDs (`003 FR-1, AC-2`).
5. **Fechar** — status atualizado no PR de fechamento da fase.

Convenções: MUST/SHOULD/MAY (RFC 2119). Prioridades: **P0** (bloqueia uso real), **P1** (confiabilidade), **P2** (polimento).

## Documentos

| Documento | Conteúdo |
|---|---|
| [ANALYSIS.md](ANALYSIS.md) | Relatório da verificação |
| [constitution.md](constitution.md) | Princípios inegociáveis |
| [ROADMAP.md](ROADMAP.md) | Tarefas por fase |

## Specs

| ID | Spec | Prioridade | Status |
|---|---|---|---|
| 001 | [<Área>](001-<area>/spec.md) | P0 | Draft |

Status possíveis: `Draft` → `Approved` → `In Progress` → `Done`.
````

## specs/NNN-nome/spec.md

```markdown
# NNN — <Área>

- **Prioridade:** P0 | P1 | P2
- **Status:** Draft
- **Código afetado:** `<caminhos>`
- **Resolve:** <C1, A3, M2 — IDs do ANALYSIS.md, ou #issue>

## Contexto

<Por que esta área importa e o que está errado hoje.>

## Estado atual (verificado)

- <fato observado executando, com o comando ou arquivo>

## Requisitos funcionais

- **FR-1** <O sistema> MUST <comportamento verificável>.
- **FR-2** <…> SHOULD <…>.

## Requisitos não funcionais

- **NFR-1** <desempenho, tamanho, portabilidade, acessibilidade — com número>.

## Critérios de aceite

- **AC-1** Dado <contexto>, quando <ação>, então <resultado observável>.
- **AC-2** Golden test: <saída de exemplo> é igual a `<testdata/arquivo>`.
- **AC-3** QUANDO <gatilho>, O SISTEMA DEVE <resposta observável>.   ← EARS

## Fora de escopo

- <o que não será feito, e por quê>

## Decisões

- D<n> — <decisão do dono e o porquê>.
- <Revisado na implementação: o que mudou em relação ao requisito original e por quê.>
```

## Formatos de critério de aceite

Os dois formatos são aceitos; escolha pelo tipo de requisito e mantenha um só por spec quando possível.

| Formato | Use para | Exemplo |
|---|---|---|
| Dado/Quando/Então | Comportamento visto pelo usuário, cenários com contexto | Dado um YAML sem `name`, quando `build` roda, então sai com código 2 e cita `name` |
| EARS | Requisitos de sistema: eventos, estados, erros, opções | QUANDO o arquivo de saída já existe, O SISTEMA DEVE perguntar antes de sobrescrever |

Padrões EARS (Easy Approach to Requirements Syntax), em português:

| Padrão | Forma |
|---|---|
| Ubíquo | `O SISTEMA DEVE <resposta>` |
| Evento | `QUANDO <gatilho>, O SISTEMA DEVE <resposta>` |
| Estado | `ENQUANTO <estado>, O SISTEMA DEVE <resposta>` |
| Indesejado | `SE <condição indesejada>, ENTÃO O SISTEMA DEVE <resposta>` |
| Opcional | `ONDE <recurso presente>, O SISTEMA DEVE <resposta>` |

Em ambos, o resultado precisa ser **observável** por um teste automatizado.

## specs/ROADMAP.md

```markdown
# Roadmap e tarefas

Cada tarefa referencia a spec e os critérios de aceite que ela fecha. Ordem sugerida = ordem da lista.

## Fase 0 — Decisões (antes de codar)

- [ ] Responder D1–Dn em [ANALYSIS.md §7](ANALYSIS.md#7-decisões-em-aberto) e mover specs para `Approved`.

## Fase 1 — Funcionar de verdade (P0) → `v0.1.0`

- [ ] **T1** <tarefa> — 007 FR-1..5
- [ ] **T2** <tarefa> — 003 FR-1, AC-1/2

## Fase 2 — Confiável (P1) → `v0.2.0`

- [ ] **Tn** <tarefa> — <IDs>

## Fase 3 — Profissional (P2) → `v1.0.0`

- [ ] **Tn** <tarefa> — <IDs>
```

Tarefa antecipada para uma fase anterior: marque `*(antecipada)*`. Tarefa abandonada: risque (`~~…~~`) e explique.

## Épico

Título: `Fase N — <nome> (vX.Y.Z)` · Labels: `épico`, `fase-N`

```markdown
## Objetivo

<O que esta fase entrega, em uma ou duas frases.> Versão: `vX.Y.Z`.

## Tarefas (ordem sugerida de revisão e merge)

As tarefas são as sub-issues deste épico. Ordem sugerida:

1. #<n> <título> — <por que primeiro: base de outros, desbloqueia CI…>
2. #<n> <título>

## Critério de pronto

- Todos os PRs mergeados com CI verde.
- PR de fechamento mergeado (specs, ROADMAP, CHANGELOG).
- Tag `vX.Y.Z` publicada pelo dono.

Specs: <NNN, NNN> · Roadmap: <T11–T16>

---
_Generated by [Claude Code](https://claude.ai/code)_
```

## Ticket

Título em Conventional Commits (`feat: add --format all`) · Labels: `fase-N`, `tipo:<…>`, `P<1-3>`

```markdown
## Contexto

<O problema e para quem. Achado de origem: C2 em specs/ANALYSIS.md.>

## O que fazer

- <item>
- <item>

## Critérios de aceite

- [ ] Dado <…>, quando <…>, então <…> (teste automatizado).
- [ ] <verificação no CI / documentação atualizada>.

## Decisão para a revisão (opcional)

<Escolha embutida no ticket que o dono precisa ver, com a recomendação.>

**Spec(s):** <NNN FR-x, AC-y> · **Épico:** #<M>

---
_Generated by [Claude Code](https://claude.ai/code)_
```

Ordem das chamadas: criar a issue (com labels) → obter o `id` → anexar como sub-issue do épico.

## Pull request de ticket

Branch: `<tipo>/<nº>-<descrição>` · Título em Conventional Commits.

```markdown
Closes #<N> · Épico #<M> · Spec <NNN>

> PR empilhado sobre #<NN>.   ← só se for empilhado

## O que muda

<Problema e solução do ponto de vista de quem usa. Mudanças fora do escopo que eram
pré-condição para o CI ficar verde, explicadas aqui.>

## Specs e critérios de aceite

<003 FR-8, AC-6. Spec atualizada neste PR? Por quê?>

## Como foi testado

- <testes novos/alterados; cada um passou pela checagem de mutação: o que foi quebrado>
- `<make ci>` verde localmente em <SO>.
- <o que NÃO foi verificado localmente (ex.: Windows só no CI)>

## Notas para a revisão

- <decisões tomadas, alternativas, conflitos esperados com outros PRs da fase>

## Checklist

- [ ] Verificação local completa passa
- [ ] Testes cobrem os critérios de aceite
- [ ] Golden files regravados **e revisados** (se as saídas mudaram)
- [ ] Spec atualizada (se o comportamento mudou)
- [ ] README/docs atualizados (se a interface mudou)
- [ ] Mudança incompatível sinalizada com `!` e explicada
- [ ] Não altera status de specs, checkboxes do ROADMAP nem entradas do CHANGELOG

🤖 Generated with [Claude Code](https://claude.com/claude-code)
<link da sessão>
```

Se o repositório tiver `pull_request_template.md`, use-o e mantenha a primeira linha normativa.

## Comentário de conflitos (fase de revisão)

```markdown
### Conflitos com outros PRs da fase

Simulado com `git merge-tree --write-tree origin/<esta> origin/<outra>` em <data> (commits `<sha>`/`<sha>`):

| Com | Arquivos | Resolução |
|---|---|---|
| #<n> | `Makefile`, `.github/workflows/ci.yml` | manter os dois alvos; ordem: … |
| #<n> | — | sem conflito |

Ordem de merge sugerida: #a → #b → #c.

---
_Generated by [Claude Code](https://claude.ai/code)_
```

## PR de fechamento de fase

Branch: `chore/<épico>-close-phase-N` · Título: `chore: close phase N (vX.Y.Z)`

```markdown
Closes #<épico> · Épico #<épico> · Spec —

## O que muda

Fecha a Fase N → `vX.Y.Z`.

## Checklist de fechamento

- [ ] Todos os PRs da fase mergeados (lista: #a, #b, #c)
- [ ] `specs/README.md`: status atualizado de cada spec tocada
- [ ] Cabeçalho `Status:` de cada spec igual ao do índice
- [ ] Seção "Estado atual" das specs que a tiverem, refletindo o que foi entregue
- [ ] `specs/ROADMAP.md`: checkboxes das tarefas concluídas; antecipadas/riscadas explicadas
- [ ] `CHANGELOG.md`: `[Unreleased]` → `[X.Y.Z] - AAAA-MM-DD`; nova `[Unreleased]` vazia; links de comparação
- [ ] `specs/ANALYSIS.md`: achados resolvidos marcados (se o projeto mantiver esse controle)
- [ ] Verificação local completa verde; CI verde
- [ ] Próximo passo do dono: após o merge, `git tag vX.Y.Z && git push origin vX.Y.Z`

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

## CHANGELOG.md

```markdown
# Changelog

Formato: [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/) · Versionamento: [SemVer](https://semver.org/lang/pt-BR/).

## [Unreleased]

## [0.1.0] - AAAA-MM-DD

### Adicionado

- <…>

### Alterado

### Corrigido

### Removido

[Unreleased]: https://github.com/<o>/<r>/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/<o>/<r>/releases/tag/v0.1.0
```

## Mensagem ao dono ao fim da descoberta

```markdown
Análise concluída e registrada em `specs/ANALYSIS.md` (commit `<sha>`).

**Principais observações**
- 🔴 C1 <…>
- 🔴 C2 <…>
- 🟠 A1 <…>

**Decisões que preciso de você** (minha recomendação entre parênteses)
- D1 <pergunta> (<recomendação>)
- D2 <…>

Verificado: <build, comandos X/Y, casos de borda>. Não verificado: <Windows, release>.
Nada de código foi alterado; aguardo as decisões para começar a Fase 1.
```
