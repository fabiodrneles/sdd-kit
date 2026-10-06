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
jq="" url="" body=""
shift # api
while [ $# -gt 0 ]; do
  case "$1" in --jq) jq="$2"; shift 2 ;; -F) body="${2#body=@}"; shift 2 ;; --paginate | --silent) shift ;; *) url="$1"; shift ;; esac
done
# POST de comentário no épico: guarda o corpo em $G/posted.
[ -z "$body" ] || { cat "$body" >> "$G/posted"; exit 0; }
case "$url" in
  */issues/5/comments*) f="$G/comments.json" ;;
  */branches*) f="$G/branches.json" ;;
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
    jq --argjson n "$n" --slurpfile st "$f" '(map(select(.number == $n)) | .[0] // {}) + $st[0]' "$G/subs.json" > "$G/view.json"; f="$G/view.json" ;;
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
echo "${CLAUDE_CODE_SESSION_ID:-sem}" > "$G/env-session"
echo "${SDD_AGENT_MODEL:-}" > "$G/model-$n"
# O agente troca de branch na pasta do relé: aqui, apaga o script de que o relé veio.
[ ! -f "$G/clobber" ] || : > scripts/sdd-relay.sh
# A sessão do agente: um arquivo novo, com 1000×N tokens relidos.
printf '{"timestamp":"%s","message":{"id":"s%s-%s","usage":{"input_tokens":1,"cache_read_input_tokens":%s,"output_tokens":5}}}\n' \
  "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$n" "$(grep -c . "$G/calls")" "$((1000 * n))" > "$SDD_SESSIONS_DIR/agent-$n-$(grep -c . "$G/calls").jsonl"
[ ! -f "$G/no-pr" ] || exit 0
jq --argjson n "$n" '. + [{number: (100 + $n), body: "Closes #\($n) · Épico #5"}]' "$G/pulls.json" > "$G/p" && mv "$G/p" "$G/pulls.json"
echo '{"state":"closed","merged_at":"2026-01-01T00:00:00Z"}' > "$G/pr-$((100 + n)).json"
echo '{"state":"closed","merged_at":null}' > "$G/issue-$n.json"
SH
chmod +x "$tmp/bin/agent"
mkdir -p "$tmp/sessions"
export PATH="$tmp/bin:$PATH" G SDD_AGENT_CMD=agent SDD_SESSIONS_DIR="$tmp/sessions"

r="$tmp/repo"; mkdir -p "$r/scripts" "$r/specs/018-x"
cp "$s/sdd-relay.sh" "$s/sdd-context.sh" "$s/sdd-wait.sh" "$s/sdd-checkpoint.sh" "$s/sdd-report.sh" "$r/scripts/"
# sdd-ci.sh falso (o real tem testes próprios): $G/ci-PR lista os códigos, um por
# rodada (padrão 0, verde).
cat > "$r/scripts/sdd-ci.sh" <<'SH'
#!/bin/sh
for a; do pr="${a#\#}"; done
f="$G/ci-$pr"
[ -s "$f" ] || { echo "ok CI"; exit 0; }
rc="$(head -n 1 "$f")"; sed -i.b 1d "$f"; rm -f "$f.b"
[ "$rc" -eq 0 ] && echo "ok CI" || echo "FALHA CI: erro de teste"
exit "$rc"
SH
printf '# 018\n\n- **AC-1** Linha do AC-1.\n- **AC-2** Linha do AC-2.\n' > "$r/specs/018-x/spec.md"
cd "$r"; git init -q
reset() {
  rm -f "$tmp"/sessions/*.jsonl; rm -f "$G"/*.json "$G"/seen-* "$G"/closes-* "$G/calls" "$G/no-pr" "$G/stale-subs"; : > "$G/calls"
  echo '{"state":"open","created_at":"2026-01-01T00:00:00Z"}' > "$G/open.json"
  echo '[]' > "$G/comments.json"; echo '[]' > "$G/branches.json"; rm -f "$G/posted" "$G"/ci-*
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
# O pacote cita os scripts do repositório, não os da cópia de onde o relé roda.
grep -qF 'sh scripts/sdd-pr.sh --spec 018 --no-wait' "$G/in-3" || fail "pacote com o caminho da cópia: $(grep sdd-pr "$G/in-3")"
printf '%s\n' "$out" | tail -n 1 | grep -qF 'Próximo: PR de fechamento da fase' || fail "sem o Próximo da fase: $out"
# 018 FR-6: o custo exato de cada ticket (só a sessão dele) num comentário da issue.
c2="$(sed -n 's/^<!-- sdd-relay-cost \(.*\) -->$/\1/p' "$G/posted" | sed -n 1p)"
c3="$(sed -n 's/^<!-- sdd-relay-cost \(.*\) -->$/\1/p' "$G/posted" | sed -n 2p)"
[ "$(printf '%s' "$c2" | jq -c '[.sessions, .usage.calls, .usage.read]')" = '[1,1,2000]' ] || fail "custo do #2: $c2"
[ "$(printf '%s' "$c3" | jq -c '[.sessions, .usage.calls, .usage.read]')" = '[1,1,3000]' ] || fail "custo do #3: $c3"
grep -qF '**Custo do ticket pelo relé** (modelo padrão): 1 sessões, 1 chamadas, 2000 tokens relidos' "$G/posted" || fail "custo sem o resumo: $(cat "$G/posted")"

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

# 018 AC-3: CI vermelho duas vezes no mesmo PR: a primeira vira uma sessão de
# correção; a segunda para e comenta no épico, sem chamar o agente de novo.
reset
printf '1\n1\n' > "$G/ci-102"
rc=0; out="$($relay 2>&1)" || rc=$?
[ "$rc" -eq 1 ] || fail "CI vermelho saiu com $rc: $out"
[ "$(tr '\n' ' ' < "$G/calls")" = "2 2 " ] || fail "CI vermelho chamou: $(cat "$G/calls")"
grep -qF 'O CI do PR #102 (ticket #2) falhou' "$G/in" || fail "sessão de correção sem o pedido: $(cat "$G/in")"
grep -qF 'FALHA CI: erro de teste' "$G/in" || fail "sessão de correção sem a saída do CI"
grep -qF 'o CI do PR #102 falhou duas vezes seguidas' "$G/posted" || fail "sem comentário no épico: $(cat "$G/posted" 2> /dev/null)"
# Uma falha só: corrige e segue para o merge e o próximo ticket.
reset
printf '1\n0\n' > "$G/ci-102"
out="$($relay 2>&1)" || fail "CI vermelho uma vez falhou: $out"
[ "$(tr '\n' ' ' < "$G/calls")" = "2 2 3 " ] || fail "CI vermelho uma vez chamou: $(cat "$G/calls")"

# FR-3: o Próximo do checkpoint é perguntar ao dono: para e comenta, sem agente.
reset
jq -n '[{id: 1, body: "<!-- sdd-checkpoint -->\n## Checkpoint\n\n- **Próximo:** perguntar ao dono qual fase vem\n"}]' > "$G/comments.json"
out="$($relay 2>&1)" || fail "parada do dono saiu com erro: $out"
[ ! -s "$G/calls" ] || fail "parada do dono chamou o agente"
grep -qF 'perguntar ao dono qual fase vem' "$G/posted" || fail "parada do dono sem comentário: $out"

# FR-3: orçamento. As sessões desde a abertura do épico já gastaram 1100 tokens.
reset
mkdir -p "$tmp/sess"
printf '%s\n' '{"timestamp":"2026-01-02T00:00:00Z","message":{"id":"a","usage":{"input_tokens":100,"cache_read_input_tokens":1000}}}' > "$tmp/sess/s.jsonl"
out="$(SDD_SESSIONS_DIR="$tmp/sess" SDD_BUDGET_TOKENS=1000 $relay 2>&1)" || fail "orçamento saiu com erro: $out"
[ ! -s "$G/calls" ] || fail "orçamento esgotado chamou o agente"
grep -qF 'o orçamento da fase acabou (1100 de 1000 tokens' "$G/posted" || fail "orçamento sem comentário: $out"
out="$(SDD_SESSIONS_DIR="$tmp/sess" SDD_BUDGET_TOKENS=5000 $relay 2>&1)" || fail "orçamento com folga falhou: $out"
[ "$(tr '\n' ' ' < "$G/calls")" = "2 3 " ] || fail "orçamento com folga chamou: $(cat "$G/calls")"

# FR-4: a sessão parou no teto sem PR, mas com a branch do ticket: sessão nova a
# partir do checkpoint; na segunda, o agente abre o PR.
reset; touch "$G/no-pr"
echo '[{"name":"feat/2-um"}]' > "$G/branches.json"
cat > "$tmp/bin/agent2" <<'SH'
#!/bin/sh
[ ! -f "$G/calls" ] || [ "$(grep -c . "$G/calls")" -lt 1 ] || rm -f "$G/no-pr"
exec agent
SH
chmod +x "$tmp/bin/agent2"
out="$(SDD_AGENT_CMD=agent2 $relay 2>&1)" || fail "continuação falhou: $out"
[ "$(tr '\n' ' ' < "$G/calls")" = "2 2 3 " ] || fail "continuação chamou: $(cat "$G/calls")"
grep -qF 'Continue o ticket #2 na branch `feat/2-um`' "$G/in-2" || fail "continuação sem a branch: $(cat "$G/in-2")"
# Sem PR e sem fim: para depois de SDD_RELAY_SESSIONS sessões.
reset; touch "$G/no-pr"
echo '[{"name":"feat/2-um"}]' > "$G/branches.json"
rc=0; out="$(SDD_RELAY_SESSIONS=2 $relay 2>&1)" || rc=$?
[ "$rc" -eq 1 ] || fail "sessões sem PR saiu com $rc"
[ "$(tr '\n' ' ' < "$G/calls")" = "2 2 " ] || fail "sessões sem PR chamou: $(cat "$G/calls")"

# 019 AC-1: o agente roda sem o id da sessão que chamou o relé; o padrão não pede
# permissão e só libera os comandos da entrega.
reset
CLAUDE_CODE_SESSION_ID=pai $relay --max 1 > /dev/null 2>&1 || fail "relé com sessão pai falhou"
[ "$(cat "$G/env-session")" = sem ] || fail "o agente herdou CLAUDE_CODE_SESSION_ID: $(cat "$G/env-session")"
reset
out="$(env -u SDD_AGENT_CMD sh scripts/sdd-relay.sh --repo o/r --dry-run 2>&1)" || fail "dry-run padrão falhou"
printf '%s\n' "$out" | grep -qF "chamaria 'claude -p --permission-mode acceptEdits --tools" || fail "agente padrão sem o modo de permissão: $out"
printf '%s\n' "$out" | grep -qF "'Bash(git:*)'" || fail "agente padrão sem o git: $out"
# 020 AC-1: só as ferramentas da entrega, sem skills e sem MCP; SDD_AGENT_TOOLS troca.
printf '%s\n' "$out" | grep -qF -- "--tools Bash Read Edit Write Grep Glob --disable-slash-commands --strict-mcp-config" || fail "agente padrão não é enxuto: $out"
out="$(env -u SDD_AGENT_CMD SDD_AGENT_TOOLS="Bash Read" sh scripts/sdd-relay.sh --repo o/r --dry-run 2>&1)" || fail "dry-run com SDD_AGENT_TOOLS falhou"
printf '%s\n' "$out" | grep -qF -- "--tools Bash Read --disable-slash-commands" || fail "SDD_AGENT_TOOLS não trocou: $out"

# 020 FR-1: o relé roda de uma cópia; o agente apagar o script da pasta não o quebra.
reset; touch "$G/clobber"
cp scripts/sdd-relay.sh "$tmp/relay.bak"
out="$($relay 2>&1)" || fail "relé com o script trocado falhou: $out"
[ "$(tr '\n' ' ' < "$G/calls")" = "2 3 " ] || fail "relé com o script trocado chamou: $(cat "$G/calls")"
cp "$tmp/relay.bak" scripts/sdd-relay.sh; rm -f "$G/clobber"

# 020 AC-5: o label modelo:NOME escolhe o modelo do ticket; sem label, SDD_AGENT_MODEL.
reset
jq '(.[] | select(.number == 3)) += {labels: [{name: "modelo:haiku"}]}' "$G/subs.json" > "$G/x" && mv "$G/x" "$G/subs.json"
out="$(SDD_AGENT_MODEL=sonnet $relay 2>&1)" || fail "modelo por ticket falhou: $out"
[ "$(cat "$G/model-2")" = sonnet ] || fail "#2 sem o SDD_AGENT_MODEL: $(cat "$G/model-2")"
[ "$(cat "$G/model-3")" = haiku ] || fail "#3 sem o modelo do label: $(cat "$G/model-3")"
grep -qF '"model":"haiku"' "$G/posted" || fail "custo sem o modelo: $(cat "$G/posted")"
grep -qF '**Custo do ticket pelo relé** (modelo sonnet)' "$G/posted" || fail "resumo sem o modelo: $(cat "$G/posted")"
reset
jq '(.[] | select(.number == 2)) += {labels: [{name: "modelo:haiku"}]}' "$G/subs.json" > "$G/x" && mv "$G/x" "$G/subs.json"
out="$(env -u SDD_AGENT_CMD sh scripts/sdd-relay.sh --repo o/r --dry-run 2>&1)" || fail "dry-run com modelo falhou"
printf '%s\n' "$out" | grep -qF -- "--model haiku' para o ticket #2" || fail "agente padrão sem --model: $(printf '%s\n' "$out" | head -n 1)"

# Sem épico aberto: Próximo é perguntar ao dono.
reset; echo '[]' > "$G/epics.json"
out="$($relay 2>&1)" || fail "sem épico falhou"
printf '%s\n' "$out" | grep -qF 'perguntar ao dono' || fail "sem épico: $out"
echo "sdd-relay: ok"
