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

# 007 AC-5: a skill manda escrever no idioma do dono (CLAUDE.md/AGENTS.md ou
# perguntar uma vez) e está em inglês; o glossário acompanha os scripts.
mkdir -p "$tmp/template/common"
cp -R "$root/template/common/scripts" "$tmp/template/common/"
sed 's/ask the owner once/ask the owner/' "$root/plugins/sdd-delivery/skills/sdd-delivery/SKILL.md" > "$skill_md"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou a skill sem perguntar o idioma uma vez" >&2
  exit 1
fi
sed '/AGENTS.md/d' "$root/plugins/sdd-delivery/skills/sdd-delivery/SKILL.md" > "$skill_md"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou a skill sem o AGENTS.md na regra de idioma" >&2
  exit 1
fi
cp "$root/plugins/sdd-delivery/skills/sdd-delivery/SKILL.md" "$skill_md"
ref="$tmp/plugins/sdd-delivery/skills/sdd-delivery/references/process.md"
echo 'Escreva as specs no idioma do dono.' >> "$ref"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou texto em português na skill" >&2
  exit 1
fi
cp "$root/plugins/sdd-delivery/skills/sdd-delivery/references/process.md" "$ref"
sed 's/## Mudanças/## Changes/' "$root/template/common/scripts/sdd-check.sh" > "$tmp/template/common/scripts/sdd-check.sh"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou o glossário fora de sincronia com os scripts" >&2
  exit 1
fi
cp "$root/template/common/scripts/sdd-check.sh" "$tmp/template/common/scripts/"
sh "$tmp/scripts/check-plugin.sh" >/dev/null

# 006 AC-3 (estrutura): comando sem description e comando ausente são recusados.
cp "$root/plugins/sdd-delivery/skills/sdd-delivery/SKILL.md" "$tmp/plugins/sdd-delivery/skills/sdd-delivery/SKILL.md"
sed '/^description:/d' "$root/plugins/sdd-delivery/commands/sdd-status.md" > "$tmp/plugins/sdd-delivery/commands/sdd-status.md"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou comando sem description" >&2
  exit 1
fi
cp "$root/plugins/sdd-delivery/commands/sdd-status.md" "$tmp/plugins/sdd-delivery/commands/"
rm "$tmp/plugins/sdd-delivery/commands/sdd-next.md"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou a falta do /sdd-next" >&2
  exit 1
fi

# 008 AC-2: /sdd-release sem description e manifest com name errado são recusados.
cp "$root/plugins/sdd-delivery/commands/sdd-next.md" "$tmp/plugins/sdd-delivery/commands/"
sed '/^description:/d' "$root/plugins/sdd-release/commands/sdd-release.md" > "$tmp/plugins/sdd-release/commands/sdd-release.md"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou /sdd-release sem description" >&2
  exit 1
fi
cp "$root/plugins/sdd-release/commands/sdd-release.md" "$tmp/plugins/sdd-release/commands/"
jq '.name = "outro"' "$root/plugins/sdd-release/.claude-plugin/plugin.json" > "$tmp/plugins/sdd-release/.claude-plugin/plugin.json"
if sh "$tmp/scripts/check-plugin.sh" 2>/dev/null; then
  echo "FALHOU: check-plugin aceitou plugin.json com name diferente do diretório" >&2
  exit 1
fi
cp "$root/plugins/sdd-release/.claude-plugin/plugin.json" "$tmp/plugins/sdd-release/.claude-plugin/"
sh "$tmp/scripts/check-plugin.sh" >/dev/null

# 008 AC-3: o modelo de workflow passa no actionlint e só cria a tag em workflow_dispatch.
wf="$root/plugins/sdd-release/templates/release-tag.yml"
if command -v actionlint >/dev/null; then actionlint "$wf"; fi
grep -q 'uses: fabiodrneles/go-release-manager@' "$wf" || { echo "FALHOU: release-tag.yml não usa a Action" >&2; exit 1; }
grep -q 'create: true' "$wf" || { echo "FALHOU: release-tag.yml sem create: true" >&2; exit 1; }
# 008 FR-6: o modelo aceita release-as e ref; o comando compara com o ROADMAP.
# shellcheck disable=SC2016 # expressões do GitHub Actions, literais
if ! grep -q 'release-as: ${{ inputs.release-as }}' "$wf" || ! grep -q 'ref: ${{ inputs.ref }}' "$wf"; then
  echo "FALHOU: release-tag.yml sem release-as/ref" >&2
  exit 1
fi
grep -q 'ROADMAP' "$root/plugins/sdd-release/commands/sdd-release.md" ||
  { echo "FALHOU: /sdd-release não compara com o ROADMAP" >&2; exit 1; }
if [ "$(sed -n '/^on:/,/^[a-z]/p' "$wf" | grep -cE '^  [a-z_]+:')" -ne 1 ] || ! grep -q '^  workflow_dispatch:' "$wf"; then
  echo "FALHOU: release-tag.yml deve rodar só em workflow_dispatch" >&2
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
