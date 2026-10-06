#!/bin/sh
# Testes do self-sync (spec 015 AC-8): o kit roda o motor gerado do template e
# a divergência reprova.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
fail() { echo "FALHOU: $*" >&2; exit 1; }
tpl=template/common/.github/workflows

# 015 AC-8: a cópia versionada é a que o self-sync geraria.
out="$(sh scripts/self-sync.sh --check 2>&1)" || fail "015 AC-8: cópia do motor divergente: $out"

# Os gerados trazem o cabeçalho e chamam os scripts do template, sem duplicá-los.
for f in .github/workflows/sdd-*.yml; do
  head -n 1 "$f" | grep -q 'make self-sync' || fail "$f sem o cabeçalho de arquivo gerado"
  ! grep -q 'sh scripts/' "$f" || fail "$f chama scripts/ em vez de template/common/scripts/"
  grep -q 'SDD_ENGINE' "$f" || fail "$f perdeu o desligamento SDD_ENGINE"
done
grep -q 'workflows: \["CI"\]' .github/workflows/sdd-ci-summary.yml || fail "sdd-ci-summary não escuta o CI do kit"
grep -q '^name: CI$' .github/workflows/ci.yml || fail "o workflow de CI do kit não se chama CI"

# 015 AC-8: um workflow do template alterado sem self-sync reprova, nomeando o arquivo.
cp "$tpl/sdd-on-merge.yml" "$tpl/sdd-on-merge.yml.bak"
trap 'mv -f "$tpl/sdd-on-merge.yml.bak" "$tpl/sdd-on-merge.yml"' EXIT
printf '# mutação\n' >> "$tpl/sdd-on-merge.yml"
if out="$(sh scripts/self-sync.sh --check 2>&1)"; then fail "015 AC-8: divergência não detectada"; fi
printf '%s' "$out" | grep -q '.github/workflows/sdd-on-merge.yml' || fail "015 AC-8: não nomeou o arquivo: $out"
printf '%s' "$out" | grep -q 'make self-sync' || fail "015 AC-8: não mandou rodar make self-sync: $out"
mv -f "$tpl/sdd-on-merge.yml.bak" "$tpl/sdd-on-merge.yml"
trap - EXIT
sh scripts/self-sync.sh --check || fail "depois de restaurar, a checagem deveria passar"
echo "tests/self-sync.sh: ok"
