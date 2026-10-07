#!/bin/sh
# Notas da release (o que o release.yml publica): a seção da versão no CHANGELOG, em
# linguagem simples, e como atualizar o axyn. A lista de PRs do GitHub vem depois.
# Uso: sh scripts/release-notes.sh vX.Y.Z [CHANGELOG]
set -eu
tag="${1:?uso: release-notes.sh vX.Y.Z [CHANGELOG]}"
log="${2:-CHANGELOG.md}"
ver="${tag#v}"

section="$(awk -v v="$ver" '
	/^## \[/ { on = index($0, "## [" v "]") == 1; next }
	on { print }
' "$log")"
[ -n "$(printf '%s' "$section" | tr -d '[:space:]')" ] || { echo "release-notes: $ver não está em $log" >&2; exit 1; }

printf '%s\n' "$section" | sed -e '/./,$!d'
cat <<'NOTES'

## Como atualizar o axyn

No terminal (PowerShell, no Windows), **na raiz de cada projeto** em que você usa o axyn, com o opencode fechado:

```powershell
irm https://raw.githubusercontent.com/fabiodrneles/sdd-kit/main/scripts/install-axyn.ps1 | iex
```

```bash
curl -fsSL https://raw.githubusercontent.com/fabiodrneles/sdd-kit/main/scripts/install-axyn.sh | sh
```

O instalador troca o axyn pela versão nova, atualiza a configuração do opencode do projeto (`axyn install`) e roda o `axyn doctor`. Feche e abra o terminal, e confira com `axyn version`.
NOTES
