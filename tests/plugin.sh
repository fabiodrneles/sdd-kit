#!/bin/sh
# Testes da spec 002: manifests (002 AC-1), skill (002 AC-2) e zip (002 AC-3).
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

sh "$root/scripts/check-plugin.sh"

# A checagem precisa falhar quando o manifest está errado (mutação).
cp -R "$root/.claude-plugin" "$root/plugins" "$root/scripts" "$tmp/"
jq 'del(.plugins[0].source)' "$root/.claude-plugin/marketplace.json" > "$tmp/.claude-plugin/marketplace.json"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou marketplace sem plugins[].source" >&2
  exit 1
fi
cp "$root/.claude-plugin/marketplace.json" "$tmp/.claude-plugin/"
echo '[quebrado](references/nao-existe.md)' >> "$tmp/plugins/sdd-delivery/skills/sdd-delivery/SKILL.md"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou link quebrado na skill" >&2
  exit 1
fi

# description vazia e longa demais (AC-2).
skill_md="$tmp/plugins/sdd-delivery/skills/sdd-delivery/SKILL.md"
cp "$root/plugins/sdd-delivery/skills/sdd-delivery/SKILL.md" "$skill_md"
awk 'NR == 1 { print; next } /^description:/ { print "description: >-"; skip = 1; next } skip && /^[ \t]/ { next } { skip = 0; print }' \
  "$root/plugins/sdd-delivery/skills/sdd-delivery/SKILL.md" > "$skill_md"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou description vazia" >&2
  exit 1
fi
long="$(awk 'BEGIN { for (i = 0; i < 1025; i++) printf "a" }')"
awk -v long="$long" 'NR == 1 { print; next } /^description:/ { print "description: " long; skip = 1; next } skip && /^[ \t]/ { next } { skip = 0; print }' \
  "$root/plugins/sdd-delivery/skills/sdd-delivery/SKILL.md" > "$skill_md"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou description com mais de 1024 caracteres" >&2
  exit 1
fi

zip="$(sh "$root/scripts/package-skill.sh" "$tmp/out/sdd-delivery.zip")"
list="$(unzip -Z1 "$zip")"
for f in SKILL.md references/process.md references/templates.md \
  references/quality-gates.md references/bootstrap-checklist.md; do
  printf '%s\n' "$list" | grep -qx "sdd-delivery/$f" || { echo "FALHOU: zip sem sdd-delivery/$f" >&2; exit 1; }
done
echo "tests/plugin.sh ok"

# 002 AC-4: a release recusa tag diferente da versão do plugin.json.
v="$(jq -r .version "$root/plugins/sdd-delivery/.claude-plugin/plugin.json")"
sh "$root/scripts/check-version.sh" "v$v" >/dev/null
if sh "$root/scripts/check-version.sh" "v$v-outra" 2>/dev/null; then
  echo "FALHOU: check-version aceitou tag diferente" >&2
  exit 1
fi
echo "tests/plugin.sh (versão) ok"
