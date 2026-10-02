# 003 — Template de repositório

- **Prioridade:** P0
- **Status:** Done — entregue na `v0.1.0`
- **Código afetado:** `template/`
- **Resolve:** A2, #5

## Contexto

Os arquivos de processo do cv-craft citam Go, `make ci` e golden files. O template é a versão genérica deles, com uma parte comum e uma parte por linguagem (constituição, princípio 4).

## Requisitos funcionais

- **FR-1** `template/common/` MUST conter: `CLAUDE.md`, `AGENTS.md` (spec 006 FR-5), `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/` (tarefa, bug, config), `.github/pull_request_template.md`, `.github/CODEOWNERS`, `.claude/settings.json` (hook de sessão e o plugin `sdd-delivery@sdd-kit` habilitado), `specs/` (`README.md`, `constitution.md`, `ROADMAP.md`, `ANALYSIS.md`), `.markdownlint-cli2.yaml`, `lychee.toml` e o workflow de documentação.
- **FR-2** `template/<lang>/` MUST existir para `go`, `node`, `java`, `python` (D2), `rust` e `dotnet` (D10), cada um com: workflow `ci.yml` que roda `make ci`, `.github/dependabot.yml`, `.claude/hooks/session-start.sh` e um `Makefile` com `make ci` (o mesmo que o CI roda) e `make docs` (markdownlint).
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
- Java usa Maven por padrão (`mvn -B verify`, ou `./mvnw` se existir); Gradle fica fora da v0.1.
- Python segue a convenção de declarar `ruff` e `pytest` no extra `dev` do `pyproject.toml`; Node exige o script `test` e roda `lint` e `build` se existirem.
- Revisado na implementação: `.claude/settings.json` é igual em todas as linguagens e foi para `common/`; o `dependabot.yml` depende da linguagem e foi para `<lang>/`. O CI de cada linguagem roda `make ci`, para que a verificação local e a do CI sejam a mesma.
