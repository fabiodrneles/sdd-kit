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

# Fake gh for ticket and phase: GETs come from $G/*.json; POST and PATCH write
# the report comment to comments.json and count writes in $G/writes.
G="$tmp/gh"; mkdir -p "$G" "$tmp/bin"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
method=GET url="" jq="" body=""
shift # api
while [ $# -gt 0 ]; do
  case "$1" in
    -X) method="$2"; shift 2 ;;
    --jq) jq="$2"; shift 2 ;;
    -F) body="${2#body=@}"; shift 2 ;;
    --paginate | --silent) shift ;;
    *) url="$1"; shift ;;
  esac
done
[ -z "$body" ] || [ "$method" != GET ] || method=POST
case "$method $url" in
  "GET "*/pulls/*/commits*) n="${url%/commits*}"; f="$G/commits-${n##*/}.json" ;;
  "GET "*/pulls\?*) f="$G/pulls.json" ;;
  "GET "*actions/runs*) b="${url#*branch=}"; f="$G/runs-${b%%&*}.json" ;;
  "GET "*/sub_issues*) f="$G/subs.json" ;;
  "GET "*/comments*) f="$G/comments.json" ;;
  "PATCH "*)
    id="${url##*/}"; echo w >> "$G/writes"
    jq --arg id "$id" --rawfile b "$body" 'map(if (.id|tostring) == $id then .body = $b else . end)' "$G/comments.json" > "$G/c" && mv "$G/c" "$G/comments.json"; exit 0 ;;
  "POST "*)
    echo w >> "$G/writes"
    jq --rawfile b "$body" '. + [{id: 99, body: $b}]' "$G/comments.json" > "$G/c" && mv "$G/c" "$G/comments.json"; exit 0 ;;
  *) echo "gh falso: $method $url" >&2; exit 1 ;;
esac
[ -f "$f" ] || { echo "gh falso: sem $f" >&2; exit 1; }
jq -r "$jq" "$f"
SH
chmod +x "$tmp/bin/gh"
cat > "$G/pulls.json" <<'J'
[{"number":20,"body":"Closes #5 · Épico #9 · Spec 017","head":{"ref":"feat/5-a"}},
 {"number":21,"body":"Closes #50 · Épico #9","head":{"ref":"feat/50-x"}},
 {"number":22,"body":"Closes #6 · Épico #9","head":{"ref":"feat/6-b"}}]
J
echo '[{"commit":{"author":{"date":"2026-03-01T10:00:00Z"},"committer":{"date":"2026-03-01T10:05:00Z"}}},
 {"commit":{"author":{"date":"2026-03-01T10:20:00Z"},"committer":{"date":"2026-03-01T11:00:00Z"}}}]' > "$G/commits-20.json"
echo '[{"commit":{"author":{"date":"2026-03-02T09:00:00Z"},"committer":{"date":"2026-03-02T09:30:00Z"}}}]' > "$G/commits-22.json"
mkdir -p "$G/runs-feat"
echo '{"workflow_runs":[{"head_sha":"b","created_at":"2026-03-01T11:01:00Z","status":"completed","conclusion":"success"},
 {"head_sha":"a","created_at":"2026-03-01T10:06:00Z","status":"completed","conclusion":"failure"},
 {"head_sha":"a","created_at":"2026-03-01T10:06:00Z","status":"completed","conclusion":"success"}]}' > "$G/runs-feat/5-a.json"
echo '{"workflow_runs":[{"head_sha":"c","created_at":"2026-03-02T09:31:00Z","status":"completed","conclusion":"success"}]}' > "$G/runs-feat/6-b.json"
echo '[{"number":5,"title":"T1: a | b"},{"number":6,"title":"T2: c"}]' > "$G/subs.json"
echo '[{"id":1,"body":"outro comentário"}]' > "$G/comments.json"
: > "$G/writes"
export PATH="$tmp/bin:$PATH" G

# Sessions: one call before the #5 window, two inside it (one repeated), one
# after it, and one inside the #6 window.
mkdir -p "$tmp/t"
cat > "$tmp/t/s.jsonl" <<'J'
{"timestamp":"2026-03-01T09:59:59.000Z","message":{"id":"a0","usage":{"input_tokens":1,"cache_read_input_tokens":9000,"output_tokens":9}}}
{"timestamp":"2026-03-01T10:10:00.000Z","message":{"id":"a1","usage":{"input_tokens":1,"cache_read_input_tokens":100,"cache_creation_input_tokens":10,"output_tokens":5}}}
{"timestamp":"2026-03-01T10:10:00.000Z","message":{"id":"a1","usage":{"input_tokens":1,"cache_read_input_tokens":100,"cache_creation_input_tokens":10,"output_tokens":5}}}
{"timestamp":"2026-03-01T10:59:00.000Z","message":{"id":"a2","usage":{"input_tokens":1,"cache_read_input_tokens":200,"output_tokens":7}}}
{"timestamp":"2026-03-01T11:00:01.000Z","message":{"id":"a3","usage":{"input_tokens":1,"cache_read_input_tokens":7000,"output_tokens":9}}}
{"timestamp":"2026-03-02T09:10:00.000Z","message":{"id":"b1","usage":{"input_tokens":1,"cache_read_input_tokens":50,"output_tokens":3}}}
J
export SDD_SESSIONS_DIR="$tmp/t"

# 017 AC-2: only the calls inside the window of the PR that closes the ticket count
# (not #50's PR); time and first-round CI come with them.
out="$(sh "$r" --repo o/r ticket '#5')" || fail "ticket falhou: $out"
for l in 'ticket: #5 (PR #20)' 'janela: 2026-03-01T10:00:00Z → 2026-03-01T11:00:00Z (60 min)' \
  'CI verde na primeira rodada: não' 'chamadas: 2' 'relidos do cache: 300' 'gravados no cache: 10' 'gerados: 12'; do
  printf '%s\n' "$out" | grep -qx "$l" || fail "ticket sem '$l': $out"
done
out="$(sh "$r" --repo o/r ticket 6)" || fail "ticket 6 falhou: $out"
printf '%s\n' "$out" | grep -qx 'CI verde na primeira rodada: sim' || fail "ticket 6: $out"
rc=0; sh "$r" --repo o/r ticket '#7' > /dev/null 2>&1 || rc=$?
[ "$rc" -eq 1 ] || fail "ticket sem PR saiu com $rc"
# NFR-2: no session file, still time and CI, and no invented number.
out="$(SDD_SESSIONS_DIR="$tmp/vazio" sh "$r" --repo o/r ticket '#5')" || fail "ticket sem sessão falhou"
printf '%s\n' "$out" | grep -q 'sem dado de tokens' || fail "ticket sem sessão: $out"
printf '%s\n' "$out" | grep -q '(60 min)' || fail "ticket sem sessão sem o tempo: $out"

# 017 AC-3: two runs leave one report comment, one row per ticket and the total.
sh "$r" --repo o/r phase '#9' > /dev/null || fail "phase falhou"
sh "$r" --repo o/r phase '#9' > /dev/null || fail "phase (2ª) falhou"
[ "$(wc -l < "$G/writes")" -eq 2 ] || fail "phase escreveu $(wc -l < "$G/writes") vezes"
[ "$(jq '[.[] | select(.body | startswith("<!-- sdd-report -->"))] | length' "$G/comments.json")" -eq 1 ] \
  || fail "mais de um comentário de relatório: $(cat "$G/comments.json")"
b="$(jq -r '.[] | select(.id == 99) | .body' "$G/comments.json")"
printf '%s\n' "$b" | grep -qF '| #5 T1: a \| b | #20 | 60 | não | 2 | 300 | 10 | 12 |' || fail "linha do #5: $b"
printf '%s\n' "$b" | grep -qF '| #6 T2: c | #22 | 30 | sim | 1 | 50 | 0 | 3 |' || fail "linha do #6: $b"
printf '%s\n' "$b" | grep -qF '| **Total** | 2 PRs | 90 | 1 de 2 | 3 | 350 | 10 | 15 |' || fail "total: $b"

# Uso inválido: 3.
rc=0; sh "$r" bogus > /dev/null 2>&1 || rc=$?
[ "$rc" -eq 3 ] || fail "uso inválido saiu com $rc"
echo "sdd-report: ok"
