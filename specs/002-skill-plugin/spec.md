# 002 — Skill e plugin

- **Prioridade:** P0
- **Status:** Approved
- **Código afetado:** `plugins/sdd-delivery/`, `.claude-plugin/marketplace.json`, `scripts/package-skill.sh`, `.github/workflows/release.yml`
- **Resolve:** A1, #4, #7

## Contexto

A skill `sdd-delivery` vive copiada em `cv-craft/.claude/skills/`. Ela precisa de uma fonte única (constituição, princípio 1), instalável no Claude Code (plugin) e no claude.ai (upload de zip).

## Estado atual (verificado)

- A skill está só no cv-craft: `SKILL.md` + `references/{process,templates,quality-gates,bootstrap-checklist}.md`.

## Requisitos funcionais

- **FR-1** A skill MUST viver em `plugins/sdd-delivery/skills/sdd-delivery/` (`SKILL.md` + `references/`), com frontmatter `name` e `description`.
- **FR-2** O repositório MUST ser um marketplace do Claude Code: `.claude-plugin/marketplace.json` na raiz lista o plugin `sdd-delivery`, e `plugins/sdd-delivery/.claude-plugin/plugin.json` declara nome, versão e descrição. Instalação: `/plugin marketplace add fabiodrneles/sdd-kit` e `/plugin install sdd-delivery@sdd-kit`.
- **FR-3** `scripts/package-skill.sh` MUST gerar `dist/sdd-delivery.zip` com a pasta `sdd-delivery/` na raiz do zip, pronta para upload no claude.ai.
- **FR-4** O workflow de release MUST, numa tag `v*`, rodar o CI e anexar `sdd-delivery.zip` e seu checksum SHA-256 à release.
- **FR-5** O cv-craft MUST passar a consumir a skill do kit (declarando o marketplace em `.claude/settings.json`) e remover a cópia local, num PR no cv-craft.
- **FR-6** A versão em `plugin.json` MUST ser igual à tag da release (verificado no workflow de release).

## Critérios de aceite

- **AC-1** Dado o repositório, quando o CI roda, então os JSON dos manifests são válidos e têm os campos obrigatórios (`name`, `owner`, `plugins[].name`, `plugins[].source` no marketplace; `name`, `version` no plugin).
- **AC-2** Dado o `SKILL.md`, quando o CI roda, então o frontmatter tem `name: sdd-delivery` e `description` não vazia, e todo link relativo da skill aponta para um arquivo existente.
- **AC-3** Dado `scripts/package-skill.sh`, quando roda, então o zip contém `sdd-delivery/SKILL.md` e os quatro arquivos de `references/`.
- **AC-4** Dada uma tag cuja versão difere de `plugin.json`, quando o workflow de release roda, então ele falha antes de publicar.

## Fora de escopo

- Publicar no marketplace oficial da Anthropic.

## Decisões

- A skill é idêntica à versão do cv-craft no momento da extração; mudanças no conteúdo dela são tickets próprios.
