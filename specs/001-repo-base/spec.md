# 001 — Base do repositório

- **Prioridade:** P0
- **Status:** Approved
- **Código afetado:** raiz (`README*.md`, `LICENSE`, `CLAUDE.md`, `Makefile`), `.claude/`, `.github/`
- **Resolve:** #3

## Contexto

O repositório só tem um README. Para que o próprio kit siga o processo que distribui (constituição, princípio 7), ele precisa de CI, de um alvo local equivalente ao CI e do que um agente precisa para retomar o trabalho numa sessão nova.

## Estado atual (verificado)

- `main` contém só `README.md` com duas linhas.

## Requisitos funcionais

- **FR-1** O repositório MUST ter `LICENSE` (MIT), `README.md` em português e `README.en.md` em inglês (D4), com instalação da skill, adoção do template e link para as specs.
- **FR-2** O repositório MUST ter `CLAUDE.md` (mapa, comandos, convenções, armadilhas) e um hook de início de sessão em `.claude/hooks/` que instala as ferramentas do CI numa sessão web.
- **FR-3** `make ci` MUST rodar localmente todas as verificações do CI que não dependem de rede: markdownlint, shellcheck e os testes dos scripts.
- **FR-4** O workflow `ci.yml` MUST rodar em todo PR e push na `main`: markdownlint, shellcheck, verificação de links (lychee) e os testes das specs 002–004.
- **FR-5** `.github/` MUST ter templates de issue (tarefa, bug) e de PR, `CODEOWNERS` e `dependabot.yml` para GitHub Actions.

## Requisitos não funcionais

- **NFR-1** O job de CI de documentação SHOULD terminar em menos de 2 minutos.
- **NFR-2** Versões de ferramentas (markdownlint-cli2, shellcheck) MUST ser fixadas e iguais no CI, no `Makefile` e no hook.

## Critérios de aceite

- **AC-1** Dado o repositório limpo, quando `make ci` roda, então termina com código 0.
- **AC-2** Dado um `.md` com erro de lint (ex.: dois títulos H1), quando o CI roda, então o job falha.
- **AC-3** Dado um `.sh` com aviso do shellcheck (ex.: variável sem aspas), quando o CI roda, então o job falha.

## Fora de escopo

- Release automatizada: entra com o zip da skill (spec 002).

## Decisões

- D4 — README em português e `README.en.md` em inglês.
