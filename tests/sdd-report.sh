#!/bin/sh
# Testes do sdd-report.sh (spec 017) com arquivos de sessão de teste.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
r="$root/template/common/scripts/sdd-report.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

# Sessão de teste: a chamada m1 aparece duas vezes (o Claude Code grava uma linha
# por bloco da resposta), m2 uma vez, e uma linha sem uso.
mkdir -p "$tmp/p/x"
cat > "$tmp/p/x/s.jsonl" <<'J'
{"type":"user","timestamp":"2026-01-01T00:00:00Z","message":{"content":"oi"}}
{"type":"assistant","timestamp":"2026-01-01T00:00:01Z","message":{"id":"m1","usage":{"input_tokens":2,"cache_read_input_tokens":1000,"cache_creation_input_tokens":100,"output_tokens":50}}}
{"type":"assistant","timestamp":"2026-01-01T00:00:01Z","message":{"id":"m1","usage":{"input_tokens":2,"cache_read_input_tokens":1000,"cache_creation_input_tokens":100,"output_tokens":50}}}
{"type":"assistant","timestamp":"2026-01-02T00:00:00Z","message":{"id":"m2","usage":{"input_tokens":4,"cache_read_input_tokens":3000,"cache_creation_input_tokens":0,"output_tokens":10}}}
J

# 017 AC-1: cada chamada conta uma vez; os totais batem com o arquivo.
out="$(sh "$r" tokens --session "$tmp/p/x/s.jsonl")" || fail "tokens falhou: $out"
for l in 'chamadas: 2' 'relidos do cache: 4000' 'gravados no cache: 100' 'entrada: 6' 'gerados: 60' \
  'contexto médio por chamada: 2053' 'contexto máximo: 3004'; do
  printf '%s\n' "$out" | grep -qx "$l" || fail "sem '$l': $out"
done
# Pelo diretório de sessões, e com --since cortando a primeira chamada.
out="$(SDD_SESSIONS_DIR="$tmp/p" sh "$r" tokens --since 2026-01-02)" || fail "--since falhou"
printf '%s\n' "$out" | grep -qx 'chamadas: 1' || fail "--since não cortou: $out"
printf '%s\n' "$out" | grep -qx 'relidos do cache: 3000' || fail "--since: $out"

# 017 AC-5: sem arquivo de sessão (ou sem chamada no período), diz e sai com 0.
mkdir "$tmp/vazio"
out="$(SDD_SESSIONS_DIR="$tmp/vazio" sh "$r" tokens)" || fail "sem sessão saiu com erro"
printf '%s\n' "$out" | grep -q 'sem dado de tokens' || fail "sem sessão: $out"
out="$(SDD_SESSIONS_DIR="$tmp/p" sh "$r" tokens --since 2030-01-01)" || fail "período vazio saiu com erro"
printf '%s\n' "$out" | grep -q 'sem dado de tokens' || fail "período vazio: $out"

# Uso inválido: 3.
rc=0; sh "$r" bogus > /dev/null 2>&1 || rc=$?
[ "$rc" -eq 3 ] || fail "uso inválido saiu com $rc"
echo "sdd-report: ok"
