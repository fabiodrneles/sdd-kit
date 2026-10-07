#!/bin/sh
# Testes do scripts/release-notes.sh: a seção da versão e o bloco de atualização.
set -eu
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }
printf '# Changelog\n\n## [2.0.0] - 2026-01-02\n\n### Adicionado\n\n- coisa nova\n\n## [1.0.0] - 2026-01-01\n\n- antiga\n' > "$tmp/CHANGELOG.md"
out="$(sh "$root/scripts/release-notes.sh" v2.0.0 "$tmp/CHANGELOG.md")"
echo "$out" | grep -q "coisa nova" || fail "sem a seção da versão"
echo "$out" | grep -q "antiga" && fail "trouxe a seção de outra versão"
echo "$out" | grep -q "## Como atualizar o axyn" || fail "sem o bloco de atualização"
echo "$out" | grep -q "install-axyn.ps1 | iex" || fail "sem o comando do Windows"
if sh "$root/scripts/release-notes.sh" v9.9.9 "$tmp/CHANGELOG.md" 2>/dev/null; then fail "versão ausente deveria falhar"; fi
sh "$root/scripts/release-notes.sh" v1.16.0 "$root/CHANGELOG.md" > /dev/null || fail "o CHANGELOG do kit não tem a v1.16.0"
echo "release-notes: ok"
