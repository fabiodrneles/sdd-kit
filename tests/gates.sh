#!/bin/sh
# Os gates de lint do CI pegam erros reais (spec 001 AC-2 e 001 AC-3): um .md
# com dois H1 e um .sh com variável sem aspas fazem `make md` e `make sh` falharem.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cp "$root/Makefile" "$root/.markdownlint-cli2.yaml" "$tmp/"
cd "$tmp"
git init -q

# Controle: arquivos corretos passam (senão o teste passaria à toa).
printf '# a\n' > bom.md
printf '#!/bin/sh\necho "ok"\n' > bom.sh
make -s md >/dev/null 2>&1 || { echo "FALHOU: make md recusou um .md correto" >&2; exit 1; }
make -s sh >/dev/null 2>&1 || { echo "FALHOU: make sh recusou um .sh correto" >&2; exit 1; }

printf '# a\n\n# b\n' > ruim.md
if make -s md >/dev/null 2>&1; then
  echo "FALHOU: markdownlint aceitou dois H1 (001 AC-2)" >&2
  exit 1
fi
rm ruim.md

# shellcheck disable=SC2016 # o $1 sem aspas é o erro proposital
printf '#!/bin/sh\necho $1\n' > ruim.sh
if make -s sh >/dev/null 2>&1; then
  echo "FALHOU: shellcheck aceitou variável sem aspas (001 AC-3)" >&2
  exit 1
fi
echo "tests/gates.sh ok"
