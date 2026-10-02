#!/bin/sh
# Spec 003 AC-4 e 010 AC-1: adota o template num projeto mínimo da linguagem e roda o
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
# Spec 010 AC-1: um arquivo sem nenhum teste derruba a cobertura abaixo do mínimo,
# e o make ci falha mostrando a cobertura (arquivos não carregados também contam).
case "$lang" in
  go) printf 'package demo\n\n// Untested has no test.\nfunc Untested(n int) int {\n\tfor i := 0; i < n; i++ {\n\t\tn += i\n\t}\n\tif n > 10 {\n\t\treturn n * 2\n\t}\n\treturn n\n}\n' > untested.go ;;
  node) printf 'export function untested(n) {\n  let t = n;\n  for (let i = 0; i < n; i++) t += i;\n  if (t > 10) return t * 2;\n  return t;\n}\n' > untested.js ;;
  python) printf 'def untested(n: int) -> int:\n    t = n\n    for i in range(n):\n        t += i\n    if t > 10:\n        return t * 2\n    return t\n' > demo/untested.py ;;
  java) printf 'package demo;\n\npublic final class Untested {\n  private Untested() {}\n\n  public static int run(int n) {\n    int t = n;\n    for (int i = 0; i < n; i++) {\n      t += i;\n    }\n    if (t > 10) {\n      return t * 2;\n    }\n    return t;\n  }\n}\n' > src/main/java/demo/Untested.java ;;
esac
if make ci > coverage-gate.log 2>&1; then
  echo "FALHOU: make ci passou com um arquivo sem testes (a cobertura mínima não é exigida)" >&2
  tail -20 coverage-gate.log >&2
  exit 1
fi
grep -qiE 'cobertura|coverage' coverage-gate.log || { echo "FALHOU: make ci não mostrou a cobertura" >&2; tail -20 coverage-gate.log >&2; exit 1; }
echo "e2e-template.sh $lang ok"
