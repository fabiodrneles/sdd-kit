# 008 — Plugin sdd-release

- **Prioridade:** P1
- **Status:** Approved — decisão do dono em 2026-10-02 (o go-release-manager vira um plugin do kit)
- **Código afetado:** `plugins/sdd-release/`, `.claude-plugin/marketplace.json`, `scripts/check-plugin.sh`
- **Resolve:** #53

## Contexto

O fechamento de fase (`/sdd-close`) recebe a versão como argumento, escolhida a olho. O [go-release-manager](https://github.com/fabiodrneles/go-release-manager) calcula a próxima versão SemVer a partir dos Conventional Commits, que o processo já exige. Antes de entrar no kit, ele passou pelo próprio processo SDD (Fases 1–3, `v1.0.0`), com os bugs de cálculo corrigidos e testados.

## Requisitos funcionais

- **FR-1** O marketplace MUST oferecer o plugin `sdd-release`, instalável separado do `sdd-delivery`.
- **FR-2** O comando `/sdd-release` MUST calcular a próxima versão com `go-release-manager create --dry-run --output json`, mostrar a versão anterior, o incremento e o motivo, e propor o `/sdd-close` com essa versão. Ele MUST NOT criar tags: a tag continua sendo do dono.
- **FR-3** Sem o go-release-manager instalado, o comando MUST explicar como instalar (`go install` ou a página de Releases).
- **FR-4** O plugin MUST trazer um modelo de workflow de release, disparado à mão pelo dono (`workflow_dispatch`), que usa a GitHub Action do go-release-manager para criar a tag.
- **FR-5** `scripts/check-plugin.sh` MUST validar todos os plugins do marketplace: manifest com `name` igual ao diretório, `version` SemVer e comandos com `description`.

## Critérios de aceite

- **AC-1** Dado o marketplace, quando o CI roda `check-plugin.sh`, então o `sdd-release` está listado, com manifest válido e o comando `/sdd-release`.
- **AC-2** Dado o comando `/sdd-release` sem `description`, ou o manifest com `name` diferente do diretório, quando `check-plugin.sh` roda, então falha.
- **AC-3** Dado o modelo de workflow, quando o CI roda o actionlint, então passa, e ele usa `fabiodrneles/go-release-manager` com `create: true` só em `workflow_dispatch`.

## Fora de escopo

- Gerar o texto do CHANGELOG a partir dos commits: o `/sdd-close` continua escrevendo o CHANGELOG a partir das specs e dos PRs.
