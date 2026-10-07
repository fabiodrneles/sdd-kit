#!/bin/sh
# Copia os templates do kit para axyn/templates, que o axyn embute no binário (spec 021
# FR-4): o preparo do projeto funciona sem rede e na versão do próprio axyn.
# Uso: sh scripts/axyn-templates.sh [--check]
#   (sem argumento) regenera axyn/templates
#   --check         não escreve; falha se a cópia divergir de template/
# Não edite axyn/templates: mude template/ e rode `make axyn-templates`.
set -eu

PARTS="common seed web node python go java dotnet rust"

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
check=0
[ "${1:-}" != "--check" ] || check=1

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/templates"
for p in $PARTS; do
  [ -d "template/$p" ] || { echo "axyn-templates: template/$p não existe" >&2; exit 1; }
  cp -R "template/$p" "$tmp/templates/$p"
done

if [ "$check" -eq 1 ]; then
  if ! diff -r "$tmp/templates" axyn/templates > "$tmp/diff" 2>&1; then
    echo "axyn-templates: axyn/templates diverge de template/; rode make axyn-templates" >&2
    head -n 20 "$tmp/diff" >&2
    exit 1
  fi
  echo "axyn-templates: ok"
  exit 0
fi
rm -rf axyn/templates
cp -R "$tmp/templates" axyn/templates
echo "axyn-templates: axyn/templates regenerado ($PARTS)"
