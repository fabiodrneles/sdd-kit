# Specs — sdd-kit

Este diretório organiza o desenvolvimento do sdd-kit em **Spec Driven Development (SDD)**: nenhuma mudança entra sem uma spec que a descreva e critérios de aceite que a verifiquem. O processo é o mesmo que o kit distribui, descrito na skill `sdd-delivery`.

## Fluxo

```text
spec.md (O QUÊ / POR QUÊ)  →  revisão  →  testes a partir dos critérios de aceite  →  implementação  →  status: Done
```

1. **Especificar** — requisitos (`FR-*`), não funcionais (`NFR-*`) e critérios de aceite (`AC-*`) no formato Dado/Quando/Então.
2. **Resolver decisões** — itens em aberto são respondidos pelo dono antes de implementar.
3. **Testar primeiro** — cada `AC-*` vira ao menos uma verificação automatizada no CI.
4. **Implementar** — o PR referencia os IDs (ex.: `004 FR-2, AC-1`).
5. **Fechar** — no PR de fechamento da fase (e não em cada PR de ticket), atualizar o status abaixo, o [ROADMAP](ROADMAP.md) e o `CHANGELOG.md`.

Convenções: `MUST`/`SHOULD`/`MAY` seguem a RFC 2119. Prioridades: **P0** (bloqueia uso real), **P1** (confiabilidade), **P2** (polimento).

## Documentos

| Documento | Conteúdo |
|---|---|
| [ANALYSIS.md](ANALYSIS.md) | Ponto de partida e decisões da Fase 0 |
| [constitution.md](constitution.md) | Princípios inegociáveis do projeto |
| [ROADMAP.md](ROADMAP.md) | Tarefas por fase, ligadas às specs |

## Specs

| ID | Spec | Prioridade | Status |
|---|---|---|---|
| 001 | [Base do repositório](001-repo-base/spec.md) | P0 | Done |
| 002 | [Skill e plugin](002-skill-plugin/spec.md) | P0 | Done |
| 003 | [Template de repositório](003-template/spec.md) | P0 | Done |
| 004 | [Script de adoção](004-adoption-script/spec.md) | P0 | Done |
| 005 | [Sincronização dos repositórios](005-sync/spec.md) | P1 | Done |
| 006 | [Recursos inspirados na comunidade SDD](006-community-features/spec.md) | P1 | Done |
| 007 | [Versão 1.0: profissional e pronto para a comunidade](007-v1/spec.md) | P1 | Done |
| 008 | [Plugin sdd-release](008-sdd-release/spec.md) | P1 | Done |
| 009 | [Scripts para os passos mecânicos](009-mechanical-scripts/spec.md) | P1 | Done |
| 010 | [Esteira de qualidade criada pela adoção](010-quality-pipeline/spec.md) | P1 | Done |

Status possíveis: `Draft` → `Approved` → `In Progress` → `Done`.
