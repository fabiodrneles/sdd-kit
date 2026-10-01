# Roadmap e tarefas

Cada tarefa referencia a spec e os critérios de aceite que ela fecha. Ordem sugerida = ordem da lista.

## Fase 0 — Decisões (antes de codar)

- [x] Responder D1–D4 em [ANALYSIS.md §5](ANALYSIS.md#5-decisões-da-fase-0-respondidas-pelo-dono-em-2026-10-01) e mover specs para `Approved`.

## Fase 1 — Kit utilizável → `v0.1.0` · épico #1

- [x] **T1** Base de specs: constituição, specs, ROADMAP — #2
- [x] **T2** Base do repositório: LICENSE, READMEs, CLAUDE.md, hook, CI de docs e shellcheck — 001 FR-1..5, AC-1..3 — #3
- [x] **T3** Skill empacotada como plugin do Claude Code e zip do claude.ai — 002 FR-1..4, FR-6, AC-1..4 — #4
- [x] **T4** `template/` comum e por linguagem (Go, Node/TS, Java, Python) — 003 FR-1..5, AC-1..3 — #5
- [x] **T5** Script de adoção sh + PowerShell com testes — 004 FR-1..8, AC-1..6; 003 AC-4 — #6
- [x] **T6** cv-craft consome a skill do kit — 002 FR-5 — #7

## Fase 2 — Atualização contínua e recursos da comunidade · épico #21 *(antecipada para a `v0.1.0`)*

- [x] **T7** Arquivo de estado `.sdd-kit.json` escrito pela adoção — 005 FR-2 — #27
- [x] **T8** Workflow semanal de sincronização por PR — 005 FR-1, FR-3, FR-4, AC-1..3 — #28
- [x] **T11** Critérios em EARS na skill e no template — 006 FR-1 — #23
- [x] **T12** `sdd-check`: rastreabilidade AC → teste, status e ROADMAP — 006 FR-2, AC-1/2 — #24
- [x] **T13** Comandos de barra no plugin — 006 FR-3, AC-3 — #25
- [x] **T14** Seção "Mudanças" nas specs gerando o CHANGELOG — 006 FR-4 — #26
- [x] **T15** `AGENTS.md` no template — 006 FR-5, AC-4 — #22

Também entraram na `v0.1.0`: checagem da `description` da skill (#13), README com o fluxo completo (#15, #36), template habilitando o plugin (#18) e a spec 006 (#16).

## Fase 3 — Profissional → `v1.0.0`

- [ ] **T9** Linguagens adicionais sob demanda (um ticket por linguagem) — 003
- [ ] **T10** Revisão da skill com o uso em pelo menos dois repositórios além do cv-craft
