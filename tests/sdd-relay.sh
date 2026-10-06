#!/bin/sh
# Testes do sdd-relay.sh (spec 018) com um gh falso e um agente falso.
# shellcheck disable=SC2016 # os scripts falsos expandem as variáveis ao rodar
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

# gh falso: estado em $G (subs.json: sub-issues do épico 5; pulls.json: PRs abertos;
# pr-N.json e issue-N.json para o sdd-wait.sh).
G="$tmp/gh"; mkdir -p "$G" "$tmp/bin"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
jq=""; url=""
shift # api
while [ $# -gt 0 ]; do
  case "$1" in --jq) jq="$2"; shift 2 ;; --paginate | --silent) shift ;; *) url="$1"; shift ;; esac
done
case "$url" in
  *issues\?labels*) f="$G/epics.json" ;;
  */issues/5/sub_issues*) f="$G/view.json"
    for i in "$G"/issue-*.json; do [ -f "$i" ] && [ ! -f "$G/stale-subs" ] || continue; n="${i##*/issue-}"; n="${n%.json}"
      jq --argjson n "$n" --slurpfile st "$i" 'map(if .number == $n then .state = $st[0].state else . end)' "$G/subs.json" > "$G/s" && mv "$G/s" "$G/subs.json"; done
    cp "$G/subs.json" "$f" ;;
  */pulls\?state=open*) f="$G/view.json"
    jq -c '.[]' "$G/pulls.json" | while read -r p; do n="$(printf '%s' "$p" | jq .number)"
      if [ -f "$G/pr-$n.json" ] && [ "$(jq -r .state "$G/pr-$n.json")" != open ] && [ -f "$G/seen-$n" ]; then continue; fi
      touch "$G/seen-$n"; printf '%s\n' "$p"; done | jq -s . > "$f" ;;
  */pulls/[0-9]*) f="$G/pr-${url##*/}.json"
    # O merge fecha a issue do "Closes #N" ($G/closes-PR), como o GitHub.
    if [ -f "$G/closes-${url##*/}" ] && [ "$(jq -r '.merged_at // ""' "$f")" != "" ]; then
      echo '{"state":"closed"}' > "$G/issue-$(cat "$G/closes-${url##*/}").json"; fi ;;
  */issues/[0-9]*) n="${url##*/}"; f="$G/issue-$n.json"
    [ -f "$f" ] || f="$G/open.json"
    jq --argjson n "$n" --slurpfile st "$f" '.[] | select(.number == $n) | . + $st[0]' "$G/subs.json" > "$G/view.json"; f="$G/view.json" ;;
  *) echo "gh falso: $url" >&2; exit 1 ;;
esac
jq -r "$jq" "$f"
SH
chmod +x "$tmp/bin/gh"
# Agente falso: guarda a entrada, "abre" o PR 10N do ticket N e o dono mescla
# (o PR sai da lista de abertos depois da consulta do relé, a issue fecha).
cat > "$tmp/bin/agent" <<'SH'
#!/bin/sh
cat > "$G/in"
n="$(sed -n '1s/^# Ticket #\([0-9]*\):.*/\1/p' "$G/in")"
cp "$G/in" "$G/in-$n"; echo "$n" >> "$G/calls"
[ ! -f "$G/no-pr" ] || exit 0
jq --argjson n "$n" '. + [{number: (100 + $n), body: "Closes #\($n) · Épico #5"}]' "$G/pulls.json" > "$G/p" && mv "$G/p" "$G/pulls.json"
echo '{"state":"closed","merged_at":"2026-01-01T00:00:00Z"}' > "$G/pr-$((100 + n)).json"
echo '{"state":"closed","merged_at":null}' > "$G/issue-$n.json"
SH
chmod +x "$tmp/bin/agent"
export PATH="$tmp/bin:$PATH" G SDD_AGENT_CMD=agent

r="$tmp/repo"; mkdir -p "$r/scripts" "$r/specs/018-x"
cp "$s/sdd-relay.sh" "$s/sdd-context.sh" "$s/sdd-wait.sh" "$s/sdd-ci.sh" "$r/scripts/"
printf '# 018\n\n- **AC-1** Linha do AC-1.\n- **AC-2** Linha do AC-2.\n' > "$r/specs/018-x/spec.md"
cd "$r"; git init -q
reset() {
  rm -f "$G"/*.json "$G"/seen-* "$G"/closes-* "$G/calls" "$G/no-pr" "$G/stale-subs"; : > "$G/calls"
  echo '{"state":"open"}' > "$G/open.json"
  echo '[{"number":5}]' > "$G/epics.json"
  echo '[]' > "$G/pulls.json"
  jq -n '[{number: 1, state: "closed", title: "T0", body: ""},
    {number: 2, state: "open", title: "T1: um", body: "**Spec(s):** 018 AC-1 · **Épico:** #5"},
    {number: 3, state: "open", title: "T2: dois", body: "**Spec(s):** 018 AC-2 · **Épico:** #5"}]' > "$G/subs.json"
}
relay="sh scripts/sdd-relay.sh --repo o/r"

# 018 AC-2: dois tickets, um agente por ticket com o pacote dele, e o Próximo da fase.
reset
out="$($relay 2>&1)" || fail "relé falhou: $out"
[ "$(tr '\n' ' ' < "$G/calls")" = "2 3 " ] || fail "agente chamado para: $(cat "$G/calls")"
grep -qF -- '- **AC-1** Linha do AC-1.' "$G/in-2" || fail "pacote do #2 sem o AC-1: $(cat "$G/in-2")"
grep -qF 'AC-2' "$G/in-2" && fail "pacote do #2 com o AC-2 de outro ticket"
grep -qF -- '- **AC-2** Linha do AC-2.' "$G/in-3" || fail "pacote do #3 sem o AC-2"
grep -qF 'Trabalhe só o ticket #3' "$G/in-3" || fail "sem a instrução do ticket: $(cat "$G/in-3")"
printf '%s\n' "$out" | tail -n 1 | grep -qF 'Próximo: PR de fechamento da fase' || fail "sem o Próximo da fase: $out"

# 018 AC-4: retomada. O #2 já foi mergeado e o #3 tem PR aberto: o relé espera esse
# merge sem chamar o agente, e não refaz o #2.
reset
echo '{"state":"closed"}' > "$G/issue-2.json"
echo '[{"number":103,"body":"Closes #3 · Épico #5"}]' > "$G/pulls.json"
echo '{"state":"closed","merged_at":"2026-01-01T00:00:00Z"}' > "$G/pr-103.json"
echo 3 > "$G/closes-103"
out="$($relay 2>&1)" || fail "retomada falhou: $out"
[ ! -s "$G/calls" ] || fail "retomada chamou o agente: $(cat "$G/calls")"
printf '%s\n' "$out" | grep -qF 'PR #103 do ticket #3: esperando o merge' || fail "não esperou o PR aberto: $out"

# Paradas sem espera sem fim: a issue não fecha depois do merge; ou fecha, mas a
# lista de sub-issues atrasada ainda a mostra aberta (a trava do relé).
reset
echo '[{"number":103,"body":"Closes #2 · Épico #5"}]' > "$G/pulls.json"
echo '{"state":"closed","merged_at":"2026-01-01T00:00:00Z"}' > "$G/pr-103.json"
rc=0; out="$(SDD_RELAY_CLOSE_WAIT=1 timeout 60 sh scripts/sdd-relay.sh --repo o/r 2>&1)" || rc=$?
[ "$rc" -eq 1 ] || fail "issue que não fecha saiu com $rc: $out"
echo 2 > "$G/closes-103"; touch "$G/stale-subs"; rm -f "$G"/seen-*
rc=0; out="$(timeout 60 sh scripts/sdd-relay.sh --repo o/r 2>&1)" || rc=$?
[ "$rc" -eq 1 ] || fail "lista atrasada saiu com $rc: $out"
printf '%s\n' "$out" | grep -qF 'o ticket #2 voltou aberto depois do merge' || fail "sem a trava: $out"
[ ! -s "$G/calls" ] || fail "lista atrasada chamou o agente: $(cat "$G/calls")"

# --max e --dry-run.
reset
out="$($relay --max 1 2>&1)" || fail "--max falhou: $out"
[ "$(tr '\n' ' ' < "$G/calls")" = "2 " ] || fail "--max 1 chamou: $(cat "$G/calls")"
printf '%s\n' "$out" | grep -qF -- '--max 1 atingido; Próximo: #3' || fail "--max: $out"
reset
out="$($relay --dry-run 2>&1)" || fail "--dry-run falhou: $out"
[ ! -s "$G/calls" ] || fail "--dry-run chamou o agente"
printf '%s\n' "$out" | grep -qF '# Ticket #2: T1: um' || fail "--dry-run sem o pacote: $out"

# Agente que termina sem PR: para com 1, sem repetir o ticket.
reset; touch "$G/no-pr"
rc=0; out="$($relay 2>&1)" || rc=$?
[ "$rc" -eq 1 ] || fail "sem PR saiu com $rc: $out"
[ "$(tr '\n' ' ' < "$G/calls")" = "2 " ] || fail "sem PR chamou: $(cat "$G/calls")"

# Sem épico aberto: Próximo é perguntar ao dono.
reset; echo '[]' > "$G/epics.json"
out="$($relay 2>&1)" || fail "sem épico falhou"
printf '%s\n' "$out" | grep -qF 'perguntar ao dono' || fail "sem épico: $out"
echo "sdd-relay: ok"
