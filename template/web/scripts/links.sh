#!/bin/sh
# Links locais das páginas HTML (spec 021 FR-4 do sdd-kit): todo href/src relativo
# tem de apontar para um arquivo que existe. URLs externas, âncoras, mailto e tel
# ficam de fora (o lychee verifica as externas nos .md).
# Uso: sh scripts/links.sh página.html ...
set -eu

[ "$#" -gt 0 ] || { echo "uso: links.sh página.html ..." >&2; exit 2; }
fail=0
for page in "$@"; do
  dir="$(dirname "$page")"
  grep -oE '(href|src)="[^"]*"' "$page" | sed -e 's/^[a-z]*="//' -e 's/"$//' > "${TMPDIR:-/tmp}/links.$$" || true
  while IFS= read -r link; do
    case "$link" in
      '' | '#'* | http:* | https:* | //* | mailto:* | tel:* | data:* | javascript:*) continue ;;
    esac
    target="${link%%[#?]*}"
    case "$target" in
      /*) path=".$target" ;;
      *) path="$dir/$target" ;;
    esac
    if [ ! -e "$path" ]; then
      echo "$page: link quebrado: $link" >&2
      fail=1
    fi
  done < "${TMPDIR:-/tmp}/links.$$"
  rm -f "${TMPDIR:-/tmp}/links.$$"
done
exit "$fail"
