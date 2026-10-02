# 011 — Adoção ajustada pelo uso real

- **Prioridade:** P2
- **Status:** Approved — escopo do #80, pedido do dono em 2026-10-02
- **Código afetado:** `scripts/adopt.sh`, `scripts/adopt.ps1`, `template/common/`, `template/go/`, `template/node/`
- **Resolve:** achados 2, 5, 8, 12, 13 e 16 do #51 (#80)

## Contexto

O uso do kit em outros repositórios (#51) gerou achados de processo, que entraram na skill na `v1.0.0`, e achados que exigem código no template ou na adoção, guardados no #80. Esta spec cobre os do segundo grupo. Princípio: **a adoção avisa o que vai quebrar o CI antes do primeiro push e deixa prontos os arquivos que os scripts do processo exigem.**

## Requisitos funcionais

- **FR-1** A adoção Node MUST avisar quando o lockfile está fora de sincronia com o `package.json` (o `npm ci` do CI falharia) e quando o projeto não tem testes.
- **FR-2** O hook de sessão do template Go MUST instalar por completo a versão do Go do `go.mod`, com o `covdata`, em vez de depender da toolchain baixada pelo `GOTOOLCHAIN`.
- **FR-3** O template comum MUST trazer `doc-commands.sh` (os blocos `bash` do README rodam no CI), um `LICENSE` quando o repositório não tem um, e o alvo `make linkcheck`.
- **FR-4** A adoção (`adopt.sh` e `adopt.ps1`) MUST detectar a última tag `vX.Y.Z` do repositório, e o ROADMAP do template MUST numerar as fases a partir dela.
- **FR-5** A adoção MUST criar o `CHANGELOG.md` com a seção `[Unreleased]` quando ele não existe, sem mexer num existente.
- **FR-6** O template Node MUST trazer uma checagem de acessibilidade com axe, que o projeto web liga no `make ci`.

## Critérios de aceite

- **AC-1** Dado um projeto Node com lockfile fora de sincronia ou sem testes, quando a adoção roda, então ela avisa cada problema e não falha; com o projeto em ordem, não avisa.
- **AC-2** Dado o hook de sessão Go num `go.mod` com versão diferente da instalada, quando ele roda, então `go tool covdata` funciona e `make ci` mede a cobertura.
- **AC-3** Dado um repositório adotado, quando o `make ci` roda, então os blocos `bash` do README rodam e um link quebrado faz o `make linkcheck` falhar; o `LICENSE` existente nunca é sobrescrito.
- **AC-4** Dado um repositório com a tag `v1.4.0`, quando a adoção roda, então o ROADMAP criado começa na `v1.5.0`; sem tag, começa na `v0.1.0`.
- **AC-5** Dado um repositório sem `CHANGELOG.md`, quando a adoção roda, então ele é criado com `## [Unreleased]` e o `sdd-mark close` funciona sem edição manual.
- **AC-6** Dada uma página com uma violação de acessibilidade no fixture Node web, quando o `make ci` roda com a checagem ligada, então ele falha e mostra a violação.

## Mudanças

### Não lançado

- ADDED FR-1 — a adoção Node avisa lockfile ausente ou fora de sincronia e projeto sem script de teste (T27, #86).
