# Contribuindo com o sdd-kit

[English summary below](#english-summary)

Obrigado pelo interesse! O sdd-kit é desenvolvido com o próprio processo que ele distribui: **nenhuma mudança entra sem uma spec com critérios de aceite, e todo critério de aceite tem teste**.

## Como propor uma mudança

1. **Abra uma issue** (modelos: tarefa, bug ou nova linguagem). Diga o problema, para quem e como verificar.
2. **Spec:** se a mudança altera comportamento, ela muda ou cria uma spec em [`specs/`](specs/README.md). Critérios de aceite em Dado/Quando/Então ou EARS.
3. **Branch** a partir da `main`: `<tipo>/<nº-da-issue>-<descrição>` (ex.: `feat/52-rust-template`).
4. **Testes primeiro.** Cada `AC-n` novo é citado num teste como `NNN AC-n`; o `sdd-check` confere.
5. **Commits** em [Conventional Commits](https://www.conventionalcommits.org/pt-br/), em inglês, no imperativo (`feat: add rust template`).
6. **PR** começando com `Closes #N · Épico #M · Spec NNN` (o modelo já traz). Um PR por issue.
7. Antes de todo push:

   ```text
   make ci
   ```

   Roda markdownlint, shellcheck, actionlint, os testes dos scripts e o `sdd-check --strict`. O CI roda o mesmo em Linux, macOS e Windows, mais o e2e de cada linguagem do template.

Specs, issues e PRs em português; código, commits, `README.en.md` e a skill (`plugins/`) em inglês. A skill escreve no idioma do dono de cada repositório (constituição, princípio 8). Merge, tags e releases são do mantenedor.

## Adicionar uma linguagem ao template

1. Abra a issue **Nova linguagem**.
2. Crie `template/<lang>/` com `Makefile` (`make ci`, `make docs`, `make sdd-check`), `.github/workflows/ci.yml` que roda `make ci`, `.github/dependabot.yml` e `.claude/hooks/session-start.sh`.
3. Acrescente a linguagem em `LANGS` nos dois scripts de adoção (`scripts/adopt.sh` e `scripts/adopt.ps1`) e nos testes.
4. Crie um projeto mínimo em `tests/fixtures/<lang>/` e a linguagem na matriz `Template` do CI: o `make ci` gerado precisa passar.
5. Atualize a spec 003 e as tabelas do README.

## Reportar bugs e vulnerabilidades

Bugs: issue com o modelo **Bug**. Vulnerabilidades: veja [SECURITY.md](SECURITY.md). Participação: [Código de conduta](CODE_OF_CONDUCT.md).

## English summary

sdd-kit is built with the process it ships: open an issue, change or add a spec with acceptance criteria, cite each `AC-n` in a test as `NNN AC-n`, branch `<type>/<issue>-<desc>`, Conventional Commits in English, one PR per issue starting with `Closes #N`, and run `make ci` before every push. Specs, issues and PRs are written in Portuguese, but English issues are welcome. To add a template language, follow the checklist above (template folder, adoption scripts, fixture project and CI matrix).
