#!/bin/sh
# Falha se a tag (vX.Y.Z) não corresponde à versão do plugin.json (spec 002 FR-6).
# Uso: scripts/check-version.sh v0.1.0
set -eu

tag="${1:?uso: check-version.sh vX.Y.Z}"
root="$(cd "$(dirname "$0")/.." && pwd)"
version="$(jq -r .version "$root/plugins/sdd-delivery/.claude-plugin/plugin.json")"
if [ "$tag" != "v$version" ]; then
  echo "erro: tag $tag diferente da versão do plugin.json (v$version)" >&2
  exit 1
fi
echo "versão ok: $tag"
