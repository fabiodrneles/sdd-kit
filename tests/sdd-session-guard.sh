#!/bin/sh
# Testes do sdd-session-guard.sh (spec 018 FR-4): o teto de contexto da sessão.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
g="$root/template/common/scripts/sdd-session-guard.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

# A última chamada tem 9000 de contexto (8000 relidos + 900 gravados + 100 de entrada).
cat > "$tmp/t.jsonl" <<'J'
{"type":"assistant","message":{"id":"a","usage":{"input_tokens":1,"cache_read_input_tokens":500}}}
{"type":"user","message":{"content":"x"}}
{"type":"assistant","message":{"id":"b","usage":{"input_tokens":100,"cache_read_input_tokens":8000,"cache_creation_input_tokens":900,"output_tokens":7}}}
J
in="$(jq -nc --arg t "$tmp/t.jsonl" '{transcript_path: $t}')"

# Acima do teto: bloqueia com a ordem de WIP, checkpoint e fim.
out="$(printf '%s' "$in" | SDD_SESSION_MAX_TOKENS=9000 sh "$g")" || fail "guarda saiu com erro"
[ "$(printf '%s' "$out" | jq -r .decision)" = block ] || fail "não bloqueou: $out"
printf '%s' "$out" | jq -r .reason | grep -qF '9000 de 9000 tokens' || fail "sem o tamanho: $out"
printf '%s' "$out" | jq -r .reason | grep -qF 'commit WIP' || fail "sem a ordem de WIP: $out"
# Abaixo do teto, sem teto, sem transcrição ou com entrada inválida: nada, e sai com 0.
out="$(printf '%s' "$in" | SDD_SESSION_MAX_TOKENS=9001 sh "$g")" || fail "abaixo do teto saiu com erro"
[ -z "$out" ] || fail "abaixo do teto falou: $out"
out="$(printf '%s' "$in" | sh "$g")" || fail "sem teto saiu com erro"
[ -z "$out" ] || fail "sem teto falou: $out"
out="$(printf '{"transcript_path":"%s/nada"}' "$tmp" | SDD_SESSION_MAX_TOKENS=1 sh "$g")" || fail "sem transcrição saiu com erro"
[ -z "$out" ] || fail "sem transcrição falou: $out"
out="$(printf 'lixo' | SDD_SESSION_MAX_TOKENS=1 sh "$g")" || fail "entrada inválida saiu com erro"
[ -z "$out" ] || fail "entrada inválida falou: $out"

# O hook está ligado no template e no kit.
for f in "$root/template/common/.claude/settings.json" "$root/.claude/settings.json"; do
  jq -e '.hooks.PostToolUse[0].hooks[0].command | test("sdd-session-guard.sh")' "$f" > /dev/null \
    || fail "hook PostToolUse sem o guarda em $f"
done
echo "sdd-session-guard: ok"
