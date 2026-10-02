#!/bin/sh
# Testes da spec 013: os comandos de adoção do README (pt e en) rodam como
# estão escritos, contra o script local em vez do da tag publicada (013 AC-1).
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

n=0
for readme in README.md README.en.md; do
  grep -oE 'curl -fsSL https://raw\.githubusercontent\.com/fabiodrneles/sdd-kit/v[0-9]+\.[0-9]+\.[0-9]+/scripts/adopt\.sh \| sh -s -- .*$' \
    "$root/$readme" | sed 's/^.*| sh -s -- //' > "$tmp/args"
  [ "$(wc -l < "$tmp/args")" -ge 2 ] || fail "$readme: esperava o comando de adoção e a variante --dry-run"
  grep -q -- '--dry-run' "$tmp/args" || fail "$readme: falta o exemplo com --dry-run"
  while IFS= read -r args; do
    n=$((n + 1))
    d="$tmp/p$n"; mkdir "$d"
    # shellcheck disable=SC2086 # os argumentos vêm separados, como no README
    (cd "$d" && sh "$root/scripts/adopt.sh" $args --project demo --owner acme --repo demo > out 2>&1) ||
      fail "$readme: 'adopt.sh $args' falhou: $(cat "$d/out")"
    grep -q 'CLAUDE.md' "$d/out" || fail "$readme: 'adopt.sh $args' não listou o CLAUDE.md: $(cat "$d/out")"
    case "$args" in
      *--dry-run*) [ ! -e "$d/CLAUDE.md" ] || fail "$readme: --dry-run gravou arquivos" ;;
      *) [ -f "$d/CLAUDE.md" ] || fail "$readme: 'adopt.sh $args' não criou o CLAUDE.md" ;;
    esac
  done < "$tmp/args"
done

# 013 AC-2: a demonstração existe, os dois READMEs a mostram e ela vem da saída
# real (a transcrição tem a adoção e o make ci verde, e o demo.sh a gera).
demo="$root/docs/demo"
[ -s "$demo/demo.gif" ] || fail "docs/demo/demo.gif não existe"
[ "$(wc -c < "$demo/demo.gif")" -lt 2000000 ] || fail "docs/demo/demo.gif passa de 2 MB"
for readme in README.md README.en.md; do
  grep -qF '](docs/demo/demo.gif)' "$root/$readme" || fail "$readme não mostra a demonstração"
done
grep -q '^criado: CLAUDE.md$' "$demo/transcript.txt" || fail "a transcrição não tem a adoção"
grep -q '^cobertura: ' "$demo/transcript.txt" || fail "a transcrição não tem o make ci"
grep -q 'transcript.txt' "$demo/demo.sh" || fail "o demo.sh não gera a transcrição"
echo "tests/readme.sh ok ($n comandos, demonstração)"
