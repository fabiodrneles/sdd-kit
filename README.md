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

## Adotar num repositório

Com o kit clonado, rode o script na raiz do repositório de destino (novo ou existente). Nada que já existe é sobrescrito; o relatório lista o que foi criado e o que foi ignorado.

```text
sh /caminho/do/sdd-kit/scripts/adopt.sh --lang go .
```

Sem clonar (baixa o template da versão do script):

```text
curl -fsSL https://raw.githubusercontent.com/fabiodrneles/sdd-kit/v0.1.0/scripts/adopt.sh | sh -s -- --lang node .
```

No Windows, com as mesmas opções:

```text
pwsh -File C:\caminho\do\sdd-kit\scripts\adopt.ps1 --lang java .
```

| Opção | Efeito |
|---|---|
| `--lang` | `go`, `node`, `java` ou `python` (obrigatório) |
| `--project` | Nome do projeto (padrão: nome do diretório) |
| `--owner`, `--repo` | Dono e repositório no GitHub (padrão: deduzidos do `origin`) |
| `--dry-run` | Mostra o que seria feito, sem escrever |
| `--force` | Sobrescreve arquivos existentes |

Depois da adoção: preencha `CLAUDE.md` e `specs/`, e peça ao Claude Code para rodar a fase de descoberta da skill `sdd-delivery`.

## Contribuir

O kit é desenvolvido com o próprio processo. Veja a [constituição](specs/constitution.md), as [specs](specs/README.md) e o [CLAUDE.md](CLAUDE.md). Antes de todo push:

```text
make ci
```

## Licença

[MIT](LICENSE)
