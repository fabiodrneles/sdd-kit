#!/bin/sh
# Soma as linhas cobertas de todos os relatórios Cobertura (coverlet) em DIR e
# falha se a cobertura de linhas ficar abaixo de MIN (spec 010 do sdd-kit).
# Uso: sh scripts/coverage.sh DIR MIN
set -eu

dir="${1:?uso: coverage.sh DIR MIN}" min="${2:?uso: coverage.sh DIR MIN}"
files="$(find "$dir" -name 'coverage.cobertura.xml')"
[ -n "$files" ] || { echo "coverage: nenhum coverage.cobertura.xml em $dir (o projeto de teste tem coverlet.collector?)" >&2; exit 1; }
# shellcheck disable=SC2086 # um caminho por relatório
grep -h '<coverage ' $files |
  awk -v min="$min" '
    function attr(name) { return match($0, name "=\"[0-9]+\"") ? substr($0, RSTART + length(name) + 2, RLENGTH - length(name) - 3) : 0 }
    { c += attr("lines-covered"); v += attr("lines-valid") }
    END {
      pct = v ? 100 * c / v : 100
      printf "coverage: %.2f%% das linhas (%d de %d); mínimo %s%%\n", pct, c, v, min
      if (pct < min) exit 1
    }'
