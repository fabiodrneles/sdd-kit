# CLAUDE.md

Guia rápido para agentes (Claude Code) trabalharem neste repositório. O processo é o da skill `sdd-delivery`, que o próprio kit distribui (constituição, princípio 7).

## Retomar o trabalho (sessão nova ou contexto perdido)

1. Leia o **comentário "Estado da fase"** mais recente no épico aberto (label `épico`).
2. Liste os **PRs abertos** com o CI de cada um e as **issues abertas** da fase.
3. Continue do próximo passo registrado. Não refaça análise que já está em specs, issues ou PRs.

## O projeto

| Caminho | O que tem |
|---|---|
| `specs/` | Constituição, specs `NNN-nome/spec.md`, `ROADMAP.md`, `ANALYSIS.md` (decisões D1–D4) |
| `plugins/sdd-delivery/` | A skill, empacotada como plugin do Claude Code (spec 002) |
| `.claude-plugin/marketplace.json` | O repositório como marketplace de plugins |
| `template/common/`, `template/<lang>/` | O que a adoção copia para outros repositórios (spec 003) |
| `scripts/` | Adoção (`adopt.sh`, `adopt.ps1`), empacotamento da skill, verificações (spec 004) |
| `tests/` | Testes dos scripts, rodados por `make ci` |

## Comandos

```text
make ci     # markdownlint + shellcheck + testes (rode antes de todo push)
make links  # verificação de links com lychee (se instalado; o CI sempre roda)
```

Numa sessão na web, o hook `.claude/hooks/session-start.sh` instala o shellcheck na versão do CI.

## Convenções

- **Idioma:** specs, issues, PRs e documentação em português; `README.en.md` em inglês; commits e código em inglês.
- **Commits:** Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `ci:`, `chore:`; `!` para mudança incompatível).
- **Branch:** uma por ticket, `<tipo>/<nº-da-issue>-<descrição>`.
- **PR:** começa com `Closes #N · Épico #M · Spec NNN` e segue o template.
- **Arquivos de status** (status das specs, checkboxes do ROADMAP, CHANGELOG) só mudam no PR de fechamento da fase.
- **Merge, tag e release** são do dono.

## Armadilhas

- Arquivos em `template/` usam marcadores `{{PROJECT}}`, `{{OWNER}}`, `{{REPO}}`; o lint deles roda sobre uma cópia com os marcadores substituídos.
- Scripts precisam rodar em sh POSIX (Linux e macOS) e o equivalente em PowerShell no Windows; o shellcheck roda com `-s sh` nos scripts de adoção.

## Economia de uso

- Leia trechos (`sed -n`, `grep -n`) em vez de arquivos inteiros, e não releia o que já leu.
- No CI, peça só o resumo dos checks; numa falha, leia o fim do log do job.
- Valide tudo com `make ci` antes do push.
