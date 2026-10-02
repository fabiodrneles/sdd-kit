#!/bin/sh
# Testes da spec 007: arquivos de comunidade (007 AC-2) e estudo de caso no
# job de links (007 AC-3).
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
fail() { echo "FALHOU: $*" >&2; exit 1; }
cd "$root"

# 007 AC-2: itens do Community Standards do GitHub.
for f in README.md LICENSE CODE_OF_CONDUCT.md CONTRIBUTING.md SECURITY.md \
  .github/pull_request_template.md .github/ISSUE_TEMPLATE/config.yml; do
  [ -s "$f" ] || fail "arquivo de comunidade ausente ou vazio: $f"
done
n=0
for f in .github/ISSUE_TEMPLATE/*.yml; do
  [ "$(basename "$f")" = config.yml ] || n=$((n + 1))
done
[ "$n" -gt 0 ] || fail "sem template de issue"
grep -q 'description:' .github/ISSUE_TEMPLATE/language.yml || fail "template de issue \"nova linguagem\" sem description"

# 007 AC-3: o estudo de caso existe nos dois idiomas e o job de links do CI
# cobre todos os .md.
for f in docs/case-study.md docs/case-study.en.md; do
  [ -s "$f" ] || fail "estudo de caso ausente: $f"
done
grep -q "lychee-action" .github/workflows/ci.yml || fail "CI sem o job de links"
grep -qF "'./**/*.md'" .github/workflows/ci.yml || fail "job de links não cobre todos os .md"
echo "tests/community.sh ok"
