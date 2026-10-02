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
# description inline ou em bloco (>- ou |), com 1 a 1024 caracteres (limite do
# Claude; mesma regra do TestRepoSkills do cv-craft).
desc="$(printf '%s\n' "$fm" | awk '
  /^description:/ {
    v = $0; sub(/^description: */, "", v)
    if (v ~ /^[>|][-+]?$/) { block = 1; next }
    print v; exit
  }
  block && /^[ \t]/ { sub(/^[ \t]+/, ""); printf "%s%s", sep, $0; sep = " "; next }
  block { exit }')"
n="$(printf '%s' "$desc" | LC_ALL=C.UTF-8 wc -m | tr -d ' ')"
if [ "$n" -lt 1 ] || [ "$n" -gt 1024 ]; then
  err "SKILL.md: description com $n caracteres (deve ter de 1 a 1024)"
fi

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

# Todos os plugins do marketplace (spec 008 FR-5): name igual ao diretório,
# version X.Y.Z e comandos com description.
for src in $(jq -r '.plugins[].source' "$market" 2>/dev/null); do
  dir="${src#./}"
  manifest="$dir/.claude-plugin/plugin.json"
  [ -f "$manifest" ] || continue
  jq -e --arg n "$(basename "$dir")" '(.name == $n) and (.version | test("^[0-9]+\\.[0-9]+\\.[0-9]+$"))' \
    "$manifest" >/dev/null 2>&1 || err "$manifest: name deve ser $(basename "$dir") e version X.Y.Z"
  for cmd in "$dir"/commands/*.md; do
    [ -f "$cmd" ] || continue
    head -n1 "$cmd" | grep -qx -- '---' || err "$cmd: sem frontmatter"
    awk 'NR > 1 && $0 == "---" { exit } NR > 1' "$cmd" | grep -Eq '^description: *[^ ]' ||
      err "$cmd: frontmatter sem description"
  done
done

# sdd-release (spec 008 AC-1): listado, com o comando /sdd-release.
jq -e '.plugins | any(.name == "sdd-release")' "$market" >/dev/null 2>&1 || err "$market: falta o plugin sdd-release"
[ -f plugins/sdd-release/commands/sdd-release.md ] || err "falta o comando /sdd-release"

# Idioma (spec 007 FR-6, AC-5): a skill e os comandos estão em inglês e mandam
# escrever no idioma do dono, achado no CLAUDE.md/AGENTS.md ou perguntado uma vez.
lang="$(awk '/^## Language$/ { on = 1; next } /^## / { on = 0 } on' "$skill/SKILL.md")"
[ -n "$lang" ] || err "SKILL.md: falta a seção ## Language"
for rule in "owner's language" 'CLAUDE.md' 'AGENTS.md' 'ask the owner once' 'Commits and code'; do
  printf '%s\n' "$lang" | grep -qF "$rule" || err "SKILL.md: a seção Language não cita: $rule"
done
# Os termos em português que os scripts do template leem estão no glossário.
for term in 'épico' 'fase-' 'tipo:' 'Estado da fase' '## Fase' '## Decis' '## Mudanças' '## Estado atual'; do
  printf '%s\n' "$lang" | grep -qF -- "$term" || err "SKILL.md: o glossário não tem $term"
  [ ! -d template/common/scripts ] || grep -qF -- "$term" template/common/scripts/*.sh ||
    err "glossário da skill: $term não aparece mais nos scripts do template"
done
# Texto corrido em inglês: palavras comuns do português fora de código, links e
# do glossário reprovam.
for md in "$skill"/SKILL.md "$skill"/references/*.md plugins/*/commands/*.md; do
  # shellcheck disable=SC2016 # crases literais (código Markdown)
  pt="$(awk '/^```/ { code = !code; next } /^\| English \| Português \|/ { gl = 1 } gl && !/^\|/ { gl = 0 } !code && !gl' "$md" |
    sed 's/`[^`]*`//g; s|https*://[^ )>]*||g' | grep -inwE 'não|para|uma|com|que|dono|são|também|pelo' | head -n 3)"
  [ -z "$pt" ] || err "$md: texto em português (a skill é em inglês): $pt"
done

# Economia de uso (spec 007 AC-6): a skill e o CLAUDE.md do template mandam
# mandar saída longa para arquivo e acompanhar o CI pelo sdd-ci.sh.
saving="$(awk '/^## Resuming and saving usage$/ { on = 1; next } /^## / { on = 0 } on' "$skill/SKILL.md")"
for rule in 'echo "exit $?"; tail -n' 'sdd-ci.sh' 'unsubscribe' 'assert old in s' 'fields'; do
  printf '%s\n' "$saving" | grep -qF -- "$rule" || err "SKILL.md: a economia de uso não cita: $rule"
done
if [ -f template/common/CLAUDE.md ]; then
  for rule in 'echo "exit $?"; tail -n' 'scripts/sdd-ci.sh' 'cancele'; do
    grep -qF -- "$rule" template/common/CLAUDE.md || err "template/common/CLAUDE.md: a economia de uso não cita: $rule"
  done
fi

[ "$fail" -eq 0 ] && echo "plugin ok"
exit "$fail"
