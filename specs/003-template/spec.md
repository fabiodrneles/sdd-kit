# 003 — Template de repositório

- **Prioridade:** P0
- **Status:** Approved
- **Código afetado:** `template/`
- **Resolve:** A2, #5

## Contexto

Os arquivos de processo do cv-craft citam Go, `make ci` e golden files. O template é a versão genérica deles, com uma parte comum e uma parte por linguagem (constituição, princípio 4).

## Requisitos funcionais

- **FR-1** `template/common/` MUST conter: `CLAUDE.md`, `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/` (tarefa, bug, config), `.github/pull_request_template.md`, `.github/CODEOWNERS`, `.github/dependabot.yml`, `specs/` (`README.md`, `constitution.md`, `ROADMAP.md`, `ANALYSIS.md`), `.markdownlint-cli2.yaml`, `lychee.toml` e o workflow de documentação.
- **FR-2** `template/<lang>/` MUST existir para `go`, `node`, `java` e `python` (D2), cada um com: workflow `ci.yml` (lint + testes + build), `.claude/hooks/session-start.sh` e `.claude/settings.json`, e um ponto de entrada local equivalente ao CI documentado no `CLAUDE.md` (`make ci`).
- **FR-3** Os arquivos MUST usar os marcadores `{{PROJECT}}`, `{{OWNER}}` e `{{REPO}}`, substituídos pelo script de adoção (spec 004).
- **FR-4** Nenhum arquivo do template MAY citar "cv-craft".
- **FR-5** O `dependabot.yml` de cada linguagem MUST cobrir GitHub Actions e o gerenciador de pacotes da linguagem (gomod, npm, maven/gradle, pip).

## Requisitos não funcionais

- **NFR-1** Os workflows MUST fixar actions por versão maior (`@vN`) e declarar `permissions: contents: read`.

## Critérios de aceite

- **AC-1** Dado o template, quando o CI roda, então todos os workflows de `template/` passam no actionlint.
- **AC-2** Dado o template, quando o CI roda, então `grep -ri cv-craft template/` não encontra nada.
- **AC-3** Dado o template, quando o CI roda, então todo `.sh` passa no shellcheck e todo `.md` (com os marcadores substituídos) passa no markdownlint.
- **AC-4** Dado um projeto mínimo de cada linguagem adotado com o template (spec 004), quando o `ci.yml` gerado roda no CI do kit, então ele passa.

## Fora de escopo

- Outras linguagens (Rust, C#, …): tickets futuros, um por linguagem.
- Workflow de release por linguagem: cada projeto escolhe sua ferramenta.

## Decisões

- D2 — Go, Node/TS (React), Java e Python na v0.1.
- Java usa Maven por padrão (`mvn -B verify`); Gradle fica fora da v0.1.
