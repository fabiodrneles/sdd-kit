#!/bin/sh
# Testes do sdd-wait.sh (issue #147) com um gh falso cujas respostas mudam a cada chamada.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

# gh falso: cada consulta a pulls/N ou issues/N incrementa $G/count; a resposta
# vem de $G/seq (uma linha "estado merged_at" por chamada; a última se repete).
G="$tmp/gh"; mkdir -p "$G" "$tmp/bin"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
shift # api
url="" jq=""
while [ $# -gt 0 ]; do
  case "$1" in --jq) jq="$2"; shift 2 ;; *) url="$1"; shift ;; esac
done
case "$url" in
  repos/o/r/pulls\?state=open*)
    # Lista de PRs abertos: uma linha JSON de $G/lists por chamada (a última se repete).
    c=$(($(cat "$G/lcount" 2>/dev/null || echo 0) + 1)); echo "$c" > "$G/lcount"
    l="$(sed -n "${c}p" "$G/lists")"; [ -n "$l" ] || l="$(tail -n 1 "$G/lists")"
    echo "$l" | jq -r "$jq" ;;
  repos/o/r/pulls/* | repos/o/r/issues/*)
    c=$(($(cat "$G/count" 2>/dev/null || echo 0) + 1)); echo "$c" > "$G/count"
    l="$(sed -n "${c}p" "$G/seq")"; [ -n "$l" ] || l="$(tail -n 1 "$G/seq")"
    st="${l%% *}" m="${l#* }"
    if [ "$m" = - ]; then m=null; else m="\"$m\""; fi
    echo "{\"state\":\"$st\",\"merged_at\":$m,\"title\":\"feat: x\",\"head\":{\"sha\":\"abc1234\"}}" | jq -r "$jq" ;;
  *check-runs*) echo "{\"total_count\":1,\"check_runs\":[{\"id\":1,\"name\":\"ci\",\"status\":\"completed\",\"conclusion\":\"$(cat "$G/concl")\",\"app\":{\"slug\":\"x\"}}]}" | jq -r "$jq" ;;
  *status*) echo '{"statuses":[]}' | jq -r "$jq" ;;
  *) echo "gh falso: $url" >&2; exit 1 ;;
esac
SH
chmod +x "$tmp/bin/gh"
export PATH="$tmp/bin:$PATH" G
cp "$s/sdd-wait.sh" "$s/sdd-ci.sh" "$tmp/"
w="sh $tmp/sdd-wait.sh --repo o/r --interval 0"

# run SEQ CMD...: grava a sequência de respostas; saída em $tmp/out, código em $rc.
run() {
  rm -f "$G/count"
  printf '%s\n' "$1" > "$G/seq"
  shift
  rc=0; "$@" > "$tmp/out" 2>&1 || rc=$?
}

# 015 AC-1: PR aberto que passa a mergeado sai com 0 (na 3ª consulta).
# shellcheck disable=SC2086 # $w é um comando com argumentos
run "open -
open -
closed 2026-01-01T00:00:00Z" $w --timeout 5 pr-merged '#42'
[ "$rc" -eq 0 ] || fail "merged: código $rc"
grep -q 'sdd-wait: #42 mergeado' "$tmp/out" || fail "merged: saída: $(cat "$tmp/out")"
[ "$(cat "$G/count")" -eq 3 ] || fail "merged: esperava 3 consultas, houve $(cat "$G/count")"

# 015 AC-1: fechado sem merge sai com 1.
# shellcheck disable=SC2086
run "open -
closed -" $w --timeout 5 pr-merged '#42'
[ "$rc" -eq 1 ] || fail "closed: código $rc"
grep -q 'fechado sem merge' "$tmp/out" || fail "closed: saída: $(cat "$tmp/out")"

# 015 AC-1: tempo esgotado sai com 2, com uma única linha de saída.
# shellcheck disable=SC2086
run "open -" $w --timeout 1 pr-merged '#42'
[ "$rc" -eq 2 ] || fail "timeout: código $rc"
grep -q 'tempo esgotado' "$tmp/out" || fail "timeout: saída: $(cat "$tmp/out")"
[ "$(wc -l < "$tmp/out" | tr -d ' ')" -eq 1 ] || fail "timeout: mais de uma linha"

# issue-closed: 0 ao fechar, 2 no tempo.
# shellcheck disable=SC2086
run "open -
closed -" $w --timeout 5 issue-closed '#7'
[ "$rc" -eq 0 ] || fail "issue: código $rc"
grep -q '#7 fechada' "$tmp/out" || fail "issue: saída: $(cat "$tmp/out")"
# shellcheck disable=SC2086
run "open -" $w --timeout 1 issue-closed '#7'
[ "$rc" -eq 2 ] || fail "issue timeout: código $rc"

# ci: repassa o código do sdd-ci.sh (verde 0, vermelho 1).
printf 'open -\n' > "$G/seq"
echo success > "$G/concl"
rc=0; $w ci '#42' > "$tmp/out" 2>&1 || rc=$?
[ "$rc" -eq 0 ] || fail "ci verde: código $rc: $(cat "$tmp/out")"
grep -q 'check(s) verdes' "$tmp/out" || fail "ci verde: saída"
echo failure > "$G/concl"
rc=0; $w ci '#42' > "$tmp/out" 2>&1 || rc=$?
[ "$rc" -eq 1 ] || fail "ci vermelho: código $rc"

# 016: o vigia merged-any. runany LISTAS SEQ ARGS...: listas de PRs abertos por rodada
# e respostas de pulls/N; saída em $tmp/out, código em $rc.
runany() {
  rm -f "$G/count" "$G/lcount"
  printf '%s\n' "$1" > "$G/lists"; printf '%s\n' "$2" > "$G/seq"
  shift 2
  rc=0; "$@" > "$tmp/out" 2>&1 || rc=$?
}
# 016 AC-1: dois PRs abertos; o #1 sai da lista mergeado: código 0 e uma linha.
# shellcheck disable=SC2086
runany '[{"number":1},{"number":2}]
[{"number":1},{"number":2}]
[{"number":2}]' "closed 2026-01-01T00:00:00Z" $w --timeout 5 merged-any
[ "$rc" -eq 0 ] || fail "merged-any: código $rc: $(cat "$tmp/out")"
grep -qx 'sdd-wait: #1 mergeado: feat: x' "$tmp/out" || fail "merged-any: saída: $(cat "$tmp/out")"
[ "$(cat "$G/count")" -eq 1 ] || fail "merged-any: consultou PRs que seguem abertos ($(cat "$G/count"))"
# 016 AC-2: fechado sem merge sai com 1; sem merge no tempo, 2.
# shellcheck disable=SC2086
runany '[{"number":1}]
[]' "closed -" $w --timeout 5 merged-any
[ "$rc" -eq 1 ] || fail "merged-any fechado: código $rc"
grep -q '#1 fechado sem merge' "$tmp/out" || fail "merged-any fechado: saída: $(cat "$tmp/out")"
# shellcheck disable=SC2086
runany '[{"number":1}]' "open -" $w --timeout 1 merged-any
[ "$rc" -eq 2 ] || fail "merged-any tempo: código $rc"
# Lista vazia por falha momentânea, com o PR ainda aberto: não conclui nada.
# shellcheck disable=SC2086
runany '[{"number":1}]
[]' "open -" $w --timeout 1 merged-any
[ "$rc" -eq 2 ] || fail "merged-any com lista vazia momentânea: código $rc: $(cat "$tmp/out")"
# 016 AC-3: sem PR aberto, sai com 3 sem esperar.
# shellcheck disable=SC2086
runany '[]' "open -" $w --timeout 30 merged-any
[ "$rc" -eq 3 ] || fail "merged-any sem PR: código $rc"
grep -q 'nada a vigiar' "$tmp/out" || fail "merged-any sem PR: saída: $(cat "$tmp/out")"

# uso: 3.
for a in "" "bogus #1" "pr-merged 42" "--nope pr-merged #1" "merged-any #1"; do
  rc=0
  # shellcheck disable=SC2086
  $w $a > "$tmp/out" 2>&1 || rc=$?
  [ "$rc" -eq 3 ] || fail "uso '$a': código $rc"
done

echo "sdd-wait: ok"
