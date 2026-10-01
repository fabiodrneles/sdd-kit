# sdd-kit

[English](README.en.md)

Kit de **Spec Driven Development (SDD)** para projetos conduzidos com o Claude Code: a skill `sdd-delivery`, um template de repositório com CI por linguagem e um script de adoção. O processo é o que levou o [cv-craft](https://github.com/fabiodrneles/cv-craft) de protótipo a projeto profissional: **o dono decide, o agente executa**.

```text
descoberta → ANALYSIS.md → decisões do dono → constituição + specs + ROADMAP
  → épico por fase → tickets (sub-issues) → uma branch e um PR por ticket
  → CI verde → revisão do dono → merge → PR de fechamento → tag → release
```

> **Estado:** em construção. A Fase 1 (`v0.1.0`) está no [épico #1](https://github.com/fabiodrneles/sdd-kit/issues/1) e no [ROADMAP](specs/ROADMAP.md).

## O que vem no kit

| Parte | Para quê | Spec |
|---|---|---|
| Skill `sdd-delivery` | Ensina o agente a conduzir o processo: análise, specs, épicos, PRs, revisão, release | [002](specs/002-skill-plugin/spec.md) |
| `template/` | `CLAUDE.md`, hook de sessão, templates de issue e PR, `specs/` e CI para Go, Node/TS, Java e Python | [003](specs/003-template/spec.md) |
| Script de adoção | Copia o template para um repositório novo ou antigo sem sobrescrever nada | [004](specs/004-adoption-script/spec.md) |

## Instalar a skill

No Claude Code (a partir da `v0.1.0`):

```text
/plugin marketplace add fabiodrneles/sdd-kit
/plugin install sdd-delivery@sdd-kit
```

No claude.ai: baixe `sdd-delivery.zip` da [última release](https://github.com/fabiodrneles/sdd-kit/releases) e envie em *Configurações → Capacidades → Skills*.

## Contribuir

O kit é desenvolvido com o próprio processo. Veja a [constituição](specs/constitution.md), as [specs](specs/README.md) e o [CLAUDE.md](CLAUDE.md). Antes de todo push:

```text
make ci
```

## Licença

[MIT](LICENSE)
