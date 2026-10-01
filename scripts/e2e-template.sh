#!/bin/sh
# Spec 003 AC-4: adota o template num projeto mínimo da linguagem e roda o
# `make ci` gerado (o mesmo comando que o ci.yml gerado roda). Precisa da
# linguagem instalada; o CI roda um job por linguagem.
# Uso: scripts/e2e-template.sh go|node|java|python [diretório-de-trabalho]
set -eu

lang="${1:?uso: e2e.sh go|node|java|python}"
root="$(cd "$(dirname "$0")/.." && pwd)"
work="${2:-$(mktemp -d)}"
d="$work/$lang"
rm -rf "$d"
cp -R "$root/tests/fixtures/$lang" "$d"
sh "$root/scripts/adopt.sh" --lang "$lang" --project demo --owner acme --repo demo "$d" > /dev/null
cd "$d"
if grep -q '^deps:' Makefile; then make deps; fi
make ci
echo "e2e-template.sh $lang ok"
