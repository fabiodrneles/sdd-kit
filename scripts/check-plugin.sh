#!/bin/sh
# Verifica os manifests do marketplace e do plugin e a estrutura da skill
# (spec 002 AC-1, AC-2). Requer jq.
set -eu

cd "$(dirname "$0")/.."
fail=0
err() { echo "erro: $*" >&2; fail=1; }

market=.claude-plugin/marketplace.json
plugin=plugins/sdd-delivery/.claude-plugin/plugin.json
skill=plugins/sdd-delivery/skills/sdd-delivery

for f in "$market" "$plugin"; do
  jq empty "$f" 2>/dev/null || err "$f não é JSON válido"
done

jq -e '(.name | type == "string" and length > 0)
  and (.owner.name | type == "string" and length > 0)
  and (.plugins | type == "array" and length > 0)
  and all(.plugins[]; (.name | type == "string" and length > 0) and (.source | type == "string" and length > 0))' \
  "$market" >/dev/null 2>&1 || err "$market: faltam name, owner.name ou plugins[].name/source"

jq -r '.plugins[].source' "$market" 2>/dev/null | while read -r src; do
  [ -f "$src/.claude-plugin/plugin.json" ] || { echo "erro: $src/.claude-plugin/plugin.json não existe" >&2; exit 1; }
done || fail=1

jq -e '(.name == "sdd-delivery") and (.version | test("^[0-9]+\\.[0-9]+\\.[0-9]+$"))' \
  "$plugin" >/dev/null 2>&1 || err "$plugin: name deve ser sdd-delivery e version X.Y.Z"

# Frontmatter: primeira linha "---", name e description não vazios.
fm="$(awk 'NR == 1 && $0 != "---" { exit } NR > 1 && $0 == "---" { exit } NR > 1' "$skill/SKILL.md")"
printf '%s\n' "$fm" | grep -qx 'name: sdd-delivery' || err "SKILL.md: frontmatter sem name: sdd-delivery"
printf '%s\n' "$fm" | grep -Eq '^description: *[^ ]' || err "SKILL.md: frontmatter sem description"

# Links relativos fora de blocos de código apontam para arquivos existentes.
for md in "$skill"/SKILL.md "$skill"/references/*.md; do
  dir="$(dirname "$md")"
  awk '/^```/ { code = !code; next } !code' "$md" |
    grep -o '](\([^)#]*\))' | sed 's/^](//; s/)$//' | grep -v '^[a-z][a-z]*:' |
    while read -r link; do
      [ -z "$link" ] || [ -e "$dir/$link" ] || { echo "erro: $md: link quebrado: $link" >&2; exit 1; }
    done || fail=1
done

# Comandos de barra (spec 006 FR-3): frontmatter com description e uso da skill.
for cmd in plugins/sdd-delivery/commands/*.md; do
  [ -f "$cmd" ] || continue
  head -n1 "$cmd" | grep -qx -- '---' || err "$cmd: sem frontmatter"
  awk 'NR > 1 && $0 == "---" { exit } NR > 1' "$cmd" | grep -Eq '^description: *[^ ]' ||
    err "$cmd: frontmatter sem description"
  grep -q 'sdd-delivery' "$cmd" || err "$cmd: não usa a skill sdd-delivery"
done
for name in analyze specs epic next status close; do
  [ -f "plugins/sdd-delivery/commands/sdd-$name.md" ] || err "falta o comando /sdd-$name"
done

[ "$fail" -eq 0 ] && echo "plugin ok"
exit "$fail"
