# 004 — Script de adoção

- **Prioridade:** P0
- **Status:** Done — entregue na `v0.1.0`
- **Código afetado:** `scripts/adopt.sh`, `scripts/adopt.ps1`, `tests/`
- **Resolve:** A3, #6

## Contexto

D1 (a): um único script, para repositórios novos e antigos, que copia o template sem sobrescrever o que existe (constituição, princípios 2, 3 e 5).

## Requisitos funcionais

- **FR-1** `scripts/adopt.sh` (POSIX sh) e `scripts/adopt.ps1` (PowerShell 7) MUST aceitar: diretório de destino, `--lang go|node|java|python|rust|dotnet`, `--project`, `--owner`, `--repo`, `--dry-run` e `--force`.
- **FR-2** O script MUST copiar `template/common/` e `template/<lang>/` para o destino, substituindo os marcadores da spec 003 FR-3.
- **FR-3** Um arquivo que já existe no destino MUST NOT ser sobrescrito, a menos que `--force` seja passado.
- **FR-4** Ao terminar, o script MUST imprimir um relatório com os arquivos criados e os ignorados (já existiam), e sair com código 0.
- **FR-5** Linguagem inválida ou argumento obrigatório ausente MUST sair com código 2 e uma mensagem de uso, sem escrever nada.
- **FR-6** Sem `--owner`/`--repo`, o script SHOULD deduzi-los do `git remote get-url origin` do destino.
- **FR-7** `--dry-run` MUST listar o que seria feito sem escrever nada.
- **FR-8** O script MUST funcionar também via `curl … | sh -s -- …` (baixando o template da tag correspondente), documentado no README.

## Critérios de aceite

- **AC-1** Dado um diretório vazio, quando o script roda com `--lang go`, então todos os arquivos de `common` + `go` existem e nenhum contém `{{`.
- **AC-2** Dado um destino com `README.md` e `CLAUDE.md` próprios, quando o script roda, então esses arquivos ficam byte a byte iguais e aparecem como ignorados no relatório.
- **AC-3** Dado um destino já adotado, quando o script roda de novo, então nenhum arquivo muda (idempotência).
- **AC-4** Dado `--lang cobol`, quando o script roda, então sai com código 2 e o destino fica intacto.
- **AC-5** Dado `--dry-run`, quando o script roda, então o destino fica intacto.
- **AC-6** Os testes AC-1..AC-5 rodam no CI em Linux, macOS (sh) e Windows (PowerShell), e as duas implementações produzem a mesma árvore.

## Fora de escopo

- Mesclar conteúdo dentro de um arquivo existente (ex.: juntar dois `CLAUDE.md`): o relatório aponta, a pessoa decide.
- Atualizar repositórios já adotados: spec 005.

## Decisões

- D1 — script único, sem repositório-template no GitHub.
- Confirmado pelo dono (2026-10-01): o `adopt.ps1` usa as mesmas opções do `adopt.sh` (`--lang`, …), e não o estilo `-Lang` do PowerShell, para que documentação e testes sejam os mesmos e erro de uso saia com código 2.
