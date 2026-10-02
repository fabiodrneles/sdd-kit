# 012 — Arquivos do projeto fora da sincronização

- **Prioridade:** P1
- **Status:** Done — entregue na `v1.2.0`
- **Código afetado:** `template/`, `scripts/adopt.sh`, `scripts/adopt.ps1`, `template/common/scripts/sdd-sync.sh`, `tests/plugin.sh`
- **Resolve:** risco de sobrescrever specs e CHANGELOG do projeto na sincronização; incidente da `v1.0.0` (a tag saiu com o `plugin.json` na versão anterior)

## Contexto

A adoção grava no `.sdd-kit.json` o hash de todo arquivo do template, inclusive `specs/*.md` e `CHANGELOG.md`, que o projeto preenche logo depois. Se uma versão do kit mudar esses modelos, o `sdd-sync` substitui o ROADMAP ou o CHANGELOG do projeto pelo modelo vazio e só lista o caso como conflito; o `--force` da adoção também os sobrescreve. Esses arquivos são do projeto: o kit só deve criá-los quando faltam.

No kit, a release `v1.0.0` falhou porque o PR de fechamento abriu `[1.0.0]` no CHANGELOG sem subir o `plugin.json`, e nada no CI comparou os dois.

## Requisitos funcionais

- **FR-1** Os modelos que o projeto preenche (`specs/README.md`, `specs/ROADMAP.md`, `specs/ANALYSIS.md`, `specs/constitution.md` e `CHANGELOG.md`) MUST ficar em `template/seed/`: a adoção os cria quando faltam, nunca os sobrescreve (nem com `--force`) e não os registra no `.sdd-kit.json`.
- **FR-2** O `sdd-sync` MUST tratar como gerenciados só os arquivos que a adoção da versão alvo registra: um modelo do projeto que falta é criado, um que existe nunca muda, e os dois saem do estado.
- **FR-3** O CI do kit MUST falhar quando a versão mais recente do `CHANGELOG.md` não é a do `plugin.json` do `sdd-delivery`.

## Critérios de aceite

- **AC-1** Dado um repositório adotado, quando a adoção roda de novo com `--force`, então `specs/ROADMAP.md` e `CHANGELOG.md` alterados pelo projeto ficam intactos e não aparecem no `.sdd-kit.json`; num diretório vazio, eles são criados.
- **AC-2** Dado um repositório adotado numa versão que registrava os modelos no estado, quando o kit muda o modelo do ROADMAP e o `sdd-sync` roda, então o ROADMAP do projeto fica intacto, não é listado como conflito e sai do estado.
- **AC-3** Dado o CHANGELOG com `## [X.Y.Z]` mais recente diferente da versão do `plugin.json`, quando o `make ci` roda, então ele falha apontando as duas versões.

## Mudanças

### v1.2.0

- ADDED FR-1 — `specs/*.md` e `CHANGELOG.md` vão para `template/seed/`: criados quando faltam, nunca sobrescritos (nem com `--force`) e fora do `.sdd-kit.json` (T33, #100).
- ADDED FR-2 — o `sdd-sync` gerencia só o que a adoção da versão alvo registra; modelos do projeto são criados se faltam e saem do estado (T33, #100).
- ADDED FR-3 — o `make ci` do kit falha quando a versão mais recente do `CHANGELOG.md` não é a do `plugin.json` (T34, #101).
