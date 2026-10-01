# Changelog

Todas as mudanças relevantes deste projeto. Formato [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/), versões [SemVer](https://semver.org/lang/pt-BR/).

## [Unreleased]

## [0.1.0] - 2026-10-01

Primeira versão pública: Fases 1 e 2 do [ROADMAP](specs/ROADMAP.md).

### Adicionado

- Skill `sdd-delivery` como plugin do Claude Code (`/plugin marketplace add fabiodrneles/sdd-kit`) e zip para o claude.ai, publicado com SHA-256 na release (spec 002).
- Comandos `/sdd-analyze`, `/sdd-specs`, `/sdd-epic`, `/sdd-next`, `/sdd-status` e `/sdd-close` (spec 006 FR-3).
- `template/` com `CLAUDE.md`, `AGENTS.md`, `CONTRIBUTING.md`, templates de issue e PR, esqueleto de `specs/`, hook de sessão, plugin habilitado e CI pronto para Go, Node/TS, Java e Python (spec 003).
- Script de adoção em sh e PowerShell, para repositórios novos e existentes, sem sobrescrever nada, idempotente e com `--dry-run` (spec 004).
- `.sdd-kit.json` e sincronização semanal por PR, com conflitos listados para revisão (spec 005).
- `sdd-check`: rastreabilidade AC → teste, status das specs e IDs do ROADMAP (spec 006 FR-2).
- Critérios de aceite em EARS e seção "Mudanças" por spec, que alimenta o CHANGELOG (spec 006 FR-1, FR-4).
- CI do kit em Linux, macOS e Windows, com e2e dos templates nas quatro linguagens.

[Unreleased]: https://github.com/fabiodrneles/sdd-kit/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/fabiodrneles/sdd-kit/releases/tag/v0.1.0
