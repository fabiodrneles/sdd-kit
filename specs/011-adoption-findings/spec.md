# 011 — Adoção ajustada pelo uso real

- **Prioridade:** P2
- **Status:** Done — entregue na `v1.1.0`
- **Código afetado:** `scripts/adopt.sh`, `scripts/adopt.ps1`, `template/common/`, `template/go/`, `template/node/`
- **Resolve:** achados 2, 5, 8, 12, 13 e 16 do #51 (#80)

## Contexto

O uso do kit em outros repositórios (#51) gerou achados de processo, que entraram na skill na `v1.0.0`, e achados que exigem código no template ou na adoção, guardados no #80. Esta spec cobre os do segundo grupo. Princípio: **a adoção avisa o que vai quebrar o CI antes do primeiro push e deixa prontos os arquivos que os scripts do processo exigem.**

## Requisitos funcionais

- **FR-1** A adoção Node MUST avisar quando o lockfile está fora de sincronia com o `package.json` (o `npm ci` do CI falharia) e quando o projeto não tem testes.
- **FR-2** O hook de sessão do template Go MUST deixar a toolchain da versão do `go.mod` completa para a cobertura: a que o `GOTOOLCHAIN` baixa não traz o `covdata`, e o hook o compila nela.
- **FR-3** O template comum MUST trazer `doc-commands.sh` (os blocos `bash` do README marcados com `<!-- doc-commands -->` rodam no CI de docs) e o alvo `make linkcheck`; a adoção MUST avisar quando o repositório não tem `LICENSE`, sem criar um (a licença é decisão do dono).
- **FR-4** O ROADMAP criado pela adoção (`adopt.sh` e `adopt.ps1`) MUST NOT fixar versão nas fases: a versão vem do go-release-manager no fechamento, a partir da última tag.
- **FR-5** A adoção MUST criar o `CHANGELOG.md` com a seção `[Unreleased]` quando ele não existe, sem mexer num existente.
- **FR-6** O template Node MUST trazer uma checagem de acessibilidade com axe, que o projeto web liga no `make ci` com `A11Y_PAGES` (páginas HTML, verificadas num DOM do jsdom, sem medir contraste de cor).

## Critérios de aceite

- **AC-1** Dado um projeto Node com lockfile fora de sincronia ou sem testes, quando a adoção roda, então ela avisa cada problema e não falha; com o projeto em ordem, não avisa.
- **AC-2** Dado o hook de sessão Go num `go.mod` com versão diferente da instalada, quando ele roda, então `go tool covdata` funciona e `make ci` mede a cobertura.
- **AC-3** Dado um repositório adotado, quando o CI de docs roda, então os blocos marcados do README rodam (os outros, não) e um bloco que falha reprova; um link quebrado faz o `make linkcheck` falhar; sem `LICENSE`, a adoção avisa e não cria um, e um `LICENSE` existente fica intacto.
- **AC-4** Dado um repositório com a tag `v1.4.0`, ou sem tag, quando a adoção roda, então nenhuma fase do ROADMAP criado tem versão no cabeçalho.
- **AC-5** Dado um repositório sem `CHANGELOG.md`, quando a adoção roda, então ele é criado com `## [Unreleased]` e o `sdd-mark close` funciona sem edição manual.
- **AC-6** Dada uma página com uma violação de acessibilidade no fixture Node web, quando o `make ci` roda com a checagem ligada, então ele falha e mostra a violação.

## Mudanças

### v1.8.0

- MODIFIED FR-4, AC-4 — o ROADMAP criado não fixa versão nas fases; quem decide é o go-release-manager no fechamento (#198).

### v1.1.0

- ADDED FR-1 — a adoção Node avisa lockfile ausente ou fora de sincronia e projeto sem script de teste (T27, #86).
- ADDED FR-2 — o hook de sessão Go compila o `covdata` que falta na toolchain baixada pelo `GOTOOLCHAIN` (o go.dev, de onde viria a distribuição completa, costuma ser bloqueado na sessão na web) (T28, #87).
- ADDED FR-3 — `scripts/doc-commands.sh` no CI de docs, `make linkcheck` em toda linguagem e aviso de `LICENSE` ausente na adoção (T29, #88).
- ADDED FR-4 — a adoção numera as fases do ROADMAP criado a partir da última tag `vX.Y.Z` (T30, #89).
- ADDED FR-5 — a adoção cria o `CHANGELOG.md` com `[Unreleased]`, que o `sdd-mark close` exige (T31, #90).
- ADDED FR-6 — `make ci` do template Node roda o axe nas páginas de `A11Y_PAGES` (`scripts/a11y`, axe-core com jsdom) (T32, #91).
