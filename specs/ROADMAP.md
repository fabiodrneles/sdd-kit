# Roadmap e tarefas

Cada tarefa referencia a spec e os critérios de aceite que ela fecha. Ordem sugerida = ordem da lista.

## Fase 0 — Decisões (antes de codar)

- [x] Responder D1–D4 em [ANALYSIS.md §5](ANALYSIS.md#5-decisões-da-fase-0-respondidas-pelo-dono-em-2026-10-01) e mover specs para `Approved`.

## Fase 1 — Kit utilizável → `v0.1.0` · épico #1

- [ ] **T1** Base de specs: constituição, specs, ROADMAP — #2
- [ ] **T2** Base do repositório: LICENSE, READMEs, CLAUDE.md, hook, CI de docs e shellcheck — 001 FR-1..5, AC-1..3 — #3
- [ ] **T3** Skill empacotada como plugin do Claude Code e zip do claude.ai — 002 FR-1..4, FR-6, AC-1..4 — #4
- [ ] **T4** `template/` comum e por linguagem (Go, Node/TS, Java, Python) — 003 FR-1..5, AC-1..3 — #5
- [ ] **T5** Script de adoção sh + PowerShell com testes — 004 FR-1..8, AC-1..6; 003 AC-4 — #6
- [ ] **T6** cv-craft consome a skill do kit — 002 FR-5 — #7

## Fase 2 — Atualização contínua → `v0.2.0`

- [ ] **T7** Arquivo de estado `.sdd-kit.json` escrito pela adoção — 005 FR-2
- [ ] **T8** Workflow semanal de sincronização por PR — 005 FR-1, FR-3, FR-4, AC-1..3

## Fase 3 — Profissional → `v1.0.0`

- [ ] **T9** Linguagens adicionais sob demanda (um ticket por linguagem) — 003
- [ ] **T10** Revisão da skill com o uso em pelo menos dois repositórios além do cv-craft
