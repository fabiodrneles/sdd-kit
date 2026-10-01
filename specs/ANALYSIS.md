# Análise e ponto de partida do sdd-kit

## 1. Resumo executivo

O processo SDD nasceu no [cv-craft](https://github.com/fabiodrneles/cv-craft) (spec 009 e skill `sdd-delivery`) e funcionou: o cv-craft saiu de protótipo para `v1.x` com specs, CI em três sistemas e release automatizada. Hoje o processo existe só dentro do cv-craft, copiado à mão. O sdd-kit extrai esse processo para um repositório próprio, instalável em qualquer projeto. Origem: [cv-craft#42](https://github.com/fabiodrneles/cv-craft/issues/42).

## 2. O que foi verificado

| Verificação | Resultado |
|---|---|
| `fabiodrneles/sdd-kit` na `main` | Só `README.md` (commit inicial) |
| Skill em `cv-craft/.claude/skills/sdd-delivery/` | `SKILL.md` (175 linhas) + `references/` (process, templates, quality-gates, bootstrap-checklist) |
| Arquivos de processo no cv-craft | `CLAUDE.md`, `.claude/settings.json`, `.claude/hooks/session-start.sh`, `.github/` (ISSUE_TEMPLATE, PR template, CODEOWNERS, dependabot, workflows), `CONTRIBUTING.md`, `.markdownlint-cli2.yaml`, `lychee.toml` |

## 3. Observações

### Altas

- **A1** A skill tem uma única cópia, dentro de um projeto de aplicação; outros repositórios precisam copiá-la à mão e ela diverge. → spec 002.
- **A2** Os arquivos de processo citam o cv-craft (Go, `make ci`, golden files) e não servem como molde sem edição. → spec 003.
- **A3** Não há forma de adotar o processo num repositório existente sem risco de sobrescrever arquivos. → spec 004.

### Médias

- **M1** Melhorias na skill não chegam aos repositórios que já a usam. → spec 005 (Fase 2).

## 4. Pontos positivos (manter)

- A skill já é genérica (não depende de linguagem) e foi validada em uso real.
- Os gates de CI do cv-craft (lint, markdownlint, links, comandos dos READMEs) pegaram regressões reais.

## 5. Decisões da Fase 0 (respondidas pelo dono em 2026-10-01)

| ID | Pergunta | Resposta |
|---|---|---|
| D1 | Forma de adoção | **(a)** Um script só (sh e PowerShell), para repositórios novos e antigos, que copia `template/` sem sobrescrever o que existe. |
| D2 | Linguagens com CI e hook prontos na v0.1 | **Go, mais as linguagens do trabalho do dono:** Node/TS (React), Java e Python (lista confirmada pelo dono). |
| D3 | Atualização de quem já usa o kit | **(a)** PR automático semanal de sincronização, na Fase 2. |
| D4 | Idioma | **(a)** Português, com um `README.en.md`. |

## 6. Decisões da Fase 2 (respondidas pelo dono em 2026-10-01)

Pedido do dono em 2026-10-01: tornar o kit uma referência aproveitando o que a comunidade de SDD já criou. Propostas na [spec 006](006-community-features/spec.md).

| ID | Pergunta | Opções | Recomendação e resposta |
|---|---|---|---|
| D5 | Critérios em EARS (Kiro)? | (a) EARS opcional ao lado de Dado/Quando/Então; (b) EARS obrigatório; (c) manter só Dado/Quando/Então | **(a)** — **respondida: (a)**: EARS é ótimo para requisitos de sistema, Dado/Quando/Então para comportamento visto pelo usuário |
| D6 | Checagem de rastreabilidade AC → teste no CI? | (a) aviso na Fase 2, erro a partir da Fase 3; (b) erro desde já; (c) não | **(a)** — **respondida: (a)**: dá tempo de os repositórios antigos citarem os IDs nos testes |
| D7 | Comandos de barra no plugin (spec-kit, OpenSpec)? | (a) sim, os seis da spec 006 FR-3; (b) só `/sdd-status` e `/sdd-next`; (c) não | **(a)** — **respondida: (a)**: deixa o fluxo descobrível sem decorar frases |
| D8 | Deltas de mudança (OpenSpec)? | (a) seção "Mudanças" com `ADDED`/`MODIFIED`/`REMOVED` dentro de cada spec, gerando o CHANGELOG; (b) pasta `specs/changes/` como no OpenSpec; (c) não | **(a)** — **respondida: (a)**: mesmo ganho de rastreio, sem um segundo lugar para procurar |
| D9 | `AGENTS.md` para outros agentes? | (a) sim, no template, apontando para o `CLAUDE.md`; (b) não | **(a)** — **respondida: (a)**: custo baixo e amplia o público do kit |
