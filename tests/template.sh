#!/bin/sh
# Testes da spec 003: estrutura, marcadores e workflows do template (AC-1, AC-2).
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root/template"
fail=0
err() { echo "FALHOU: $*" >&2; fail=1; }

for lang in go node java python; do
  for f in Makefile .github/workflows/ci.yml .github/dependabot.yml .claude/hooks/session-start.sh; do
    [ -f "$lang/$f" ] || err "template/$lang/$f não existe"
  done
  grep -q '^ci:' "$lang/Makefile" || err "template/$lang/Makefile sem alvo ci"
  grep -q 'make ci' "$lang/.github/workflows/ci.yml" || err "template/$lang: o CI não roda make ci"
done

# Outros agentes leem AGENTS.md (spec 006 FR-5).
[ -f common/AGENTS.md ] || err "template/common/AGENTS.md não existe"
grep -q 'make ci' common/AGENTS.md || err "template/common/AGENTS.md não cita make ci"

# O template habilita a skill pelo plugin do kit (spec 003 FR-1).
jq -e '.extraKnownMarketplaces["sdd-kit"].source.repo == "fabiodrneles/sdd-kit"
  and .enabledPlugins["sdd-delivery@sdd-kit"] == true' common/.claude/settings.json >/dev/null 2>&1 ||
  err "template/common/.claude/settings.json não habilita sdd-delivery@sdd-kit"

# AC-2: nada específico do projeto de origem.
if grep -rniI 'cv-craft' . >/dev/null; then
  err "o template cita cv-craft: $(grep -rlniI 'cv-craft' . | tr '\n' ' ')"
fi

# Só os marcadores conhecidos (spec 003 FR-3).
unknown="$(grep -rhoI '{{[A-Z_]*}}' . | sort -u | grep -vxE '\{\{(PROJECT|OWNER|REPO)\}\}' || true)"
[ -z "$unknown" ] || err "marcadores desconhecidos: $unknown"

# AC-1: workflows válidos.
command -v actionlint >/dev/null || { echo "actionlint não instalado (veja .claude/hooks/session-start.sh)" >&2; exit 1; }
# shellcheck disable=SC2046 # lista de arquivos sem espaços
actionlint $(find . -path '*/.github/workflows/*.yml') || fail=1

[ "$fail" -eq 0 ] && echo "tests/template.sh ok"
exit "$fail"
