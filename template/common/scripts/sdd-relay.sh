#!/bin/sh
# sdd-relay: um agente novo por ticket, sem LLM no meio (spec 018 FR-2). Cada
# ticket roda numa sessão curta que começa só com o pacote do sdd-context.sh, em vez
# de uma sessão longa que relê a conversa inteira a cada chamada.
#
# Uso: sdd-relay.sh [--repo DONO/REPO] [--max N] [--dry-run]
#   Repete: acha o épico aberto e a primeira sub-issue aberta; se ela já tem PR
#   aberto ("Closes #N"), espera o merge; senão monta o pacote, chama o agente com
#   ele na entrada padrão ($SDD_AGENT_CMD, padrão "claude -p"), e espera o merge do
#   PR que o agente abriu (sdd-wait.sh pr-merged). Sem sub-issue aberta, imprime o
#   Próximo da fase e termina.
#   --max      no máximo N agentes nesta rodada (padrão: sem limite)
#   --dry-run  mostra o ticket e o pacote, sem chamar o agente
#   SDD_RELAY_WAIT: segundos de cada espera do merge (padrão 3600; repete até o merge).
#   Paradas (FR-3, comentário no épico): Próximo do checkpoint "perguntar ao dono";
#   CI do PR vermelho duas vezes seguidas (a primeira vira uma sessão de correção);
#   SDD_BUDGET_TOKENS gasto desde a abertura do épico (sdd-report.sh tokens).
#   Teto por sessão (FR-4): SDD_SESSION_MAX_TOKENS (padrão 150000) chega ao agente,
#   e o hook sdd-session-guard.sh o faz gravar WIP e terminar; sem PR mas com a
#   branch do ticket, o relé abre outra sessão (até SDD_RELAY_SESSIONS, padrão 3).
# Sem estado próprio (NFR-1): tudo vem do GitHub; matar e rodar de novo retoma.
# Códigos: 0 fase sem ticket aberto (ou --max atingido); 1 parou (agente sem PR,
#          PR fechado sem merge); 3 uso. Requer gh e jq.
set -eu

usage() { sed -n '2,23p' "$0"; exit 3; }
repo="" max="" dry=0
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo="${2:?}"; shift 2 ;;
    --max) max="${2:?}"; shift 2 ;;
    --dry-run) dry=1; shift ;;
    -h | --help) usage ;;
    *) echo "sdd-relay: opção desconhecida: $1" >&2; exit 3 ;;
  esac
done
case "$max" in *[!0-9]*) usage ;; esac
here="$(cd "$(dirname "$0")" && pwd)"
if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2> /dev/null || true)"
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi
[ -n "$repo" ] || { echo "sdd-relay: repositório desconhecido (use --repo)" >&2; exit 3; }
agent="${SDD_AGENT_CMD:-claude -p}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
say() { echo "sdd-relay: $*"; }

# PR aberto que fecha a issue $1 (o corpo começa por "Closes #N"), ou vazio.
pr_for() {
  gh api "repos/$repo/pulls?state=open&per_page=100" \
    --jq ".[] | select((.body // \"\") | test(\"^(Closes|Fixes|Resolves) #$1([^0-9]|\$)\"; \"i\")) | .number" | head -n 1
}

# Espera o merge do PR $1; 0 mergeado, 1 fechado sem merge (repete no tempo esgotado).
wait_merge() {
  while :; do
    rc=0; sh "$here/sdd-wait.sh" --repo "$repo" --timeout "${SDD_RELAY_WAIT:-3600}" pr-merged "#$1" || rc=$?
    [ "$rc" -eq 2 ] || return "$rc"
  done
}

# Para, avisa o dono num comentário do épico (FR-3) e sai com $1.
stop() {
  say "$2; parei"
  if [ -n "${epic:-}" ] && [ "$dry" -eq 0 ]; then
    printf '<!-- sdd-relay -->\n**Relé parado:** %s.\n' "$2" > "$tmp/c"
    gh api "repos/$repo/issues/$epic/comments" -F body=@"$tmp/c" --silent 2> /dev/null \
      || say "aviso: não consegui comentar no épico #$epic"
  fi
  exit "$1"
}

# O pedido ao agente: o pacote do ticket $1 e a tarefa $2.
prompt() {
  sh "$here/sdd-context.sh" --repo "$repo" "#$1" > "$tmp/pack"
  { cat "$tmp/pack"; printf '\n## Sua tarefa nesta sessão\n\n%s\n' "$2"; } > "$tmp/prompt"
}

# Uma sessão nova do agente, com o teto de contexto (FR-4, sdd-session-guard.sh).
run_agent() {
  runs=$((runs + 1))
  SDD_SESSION_MAX_TOKENS="${SDD_SESSION_MAX_TOKENS:-150000}" sh -c "$agent" < "$tmp/prompt" \
    || say "aviso: o agente saiu com erro no ticket #$n"
}

# Tokens gastos nas sessões desde $1 (spec 017), para o orçamento da fase.
spent() {
  sh "$here/sdd-report.sh" tokens --since "$1" 2> /dev/null \
    | awk -F ': ' '/^(relidos do cache|gravados no cache|entrada|gerados): / { s += $2 } END { print s + 0 }'
}

deliver="implemente, rode \`make ci\`, abra o PR com \`sdd-pr.sh --no-wait\` e termine. Não espere o merge nem comece outro ticket: o relé (sdd-relay.sh) cuida disso."
runs=0 merged=" "
while :; do
  epic="$(gh api "repos/$repo/issues?labels=%C3%A9pico&state=open&per_page=1" --jq '.[0].number // empty')"
  [ -n "$epic" ] || { say "nenhum épico aberto; Próximo: perguntar ao dono a próxima fase"; exit 0; }
  next="$(sh "$here/sdd-checkpoint.sh" --repo "$repo" show 2> /dev/null | sed -n 's/^- \*\*Próximo:\*\* //p' | head -n 1)"
  case "$next" in *"perguntar ao dono"*) stop 0 "o Próximo do checkpoint é \"$next\"" ;; esac
  n="$(gh api "repos/$repo/issues/$epic/sub_issues?per_page=100" --jq '[.[] | select(.state == "open")][0].number // empty')"
  if [ -z "$n" ]; then
    say "épico #$epic sem ticket aberto; Próximo: PR de fechamento da fase (o motor abre)"
    exit 0
  fi
  # Trava: um ticket já mergeado nesta rodada que volta aberto (lista atrasada, PR
  # sem "Closes") para o relé, em vez de chamar o agente de novo ou esperar sem fim.
  case "$merged" in *" $n "*) say "o ticket #$n voltou aberto depois do merge; parei"; exit 1 ;; esac
  budget="${SDD_BUDGET_TOKENS:-}"
  if [ -n "$budget" ]; then
    since="$(gh api "repos/$repo/issues/$epic" --jq '.created_at // empty')"
    [ -n "$since" ] || stop 1 "não consegui ler a abertura do épico #$epic para o orçamento"
    used="$(spent "$since")"
    [ "$used" -lt "$budget" ] || stop 0 "o orçamento da fase acabou ($used de $budget tokens, SDD_BUDGET_TOKENS)"
  fi
  pr="$(pr_for "$n")"
  if [ -z "$pr" ]; then
    if [ -n "$max" ] && [ "$runs" -ge "$max" ]; then say "--max $max atingido; Próximo: #$n"; exit 0; fi
    task="Trabalhe só o ticket #$n: $deliver" sessions=0
    while :; do
      prompt "$n" "$task"
      if [ "$dry" -eq 1 ]; then
        say "[dry-run] chamaria '$agent' para o ticket #$n com $(wc -c < "$tmp/prompt") bytes:"
        cat "$tmp/prompt"
        exit 0
      fi
      say "ticket #$n (épico #$epic): agente novo"
      run_agent; sessions=$((sessions + 1))
      pr="$(pr_for "$n")"
      [ -z "$pr" ] || break
      # Sem PR, mas com a branch do ticket: a sessão parou no teto de contexto (WIP).
      b="$(gh api "repos/$repo/branches?per_page=100" --jq ".[] | select(.name | test(\"^[a-z]+/$n-\")) | .name" | head -n 1)"
      if [ -z "$b" ] || [ "$sessions" -ge "${SDD_RELAY_SESSIONS:-3}" ]; then stop 1 "o agente terminou sem PR para #$n"; fi
      say "ticket #$n sem PR, com a branch $b: sessão nova a partir do checkpoint"
      task="Continue o ticket #$n na branch \`$b\`, a partir do checkpoint do épico (a sessão anterior parou no teto de contexto): $deliver"
    done
  fi
  # CI do PR: uma falha vira uma sessão de correção; a segunda seguida para (FR-3).
  reds=0
  while :; do
    rc=0; sh "$here/sdd-ci.sh" --repo "$repo" --timeout "${SDD_RELAY_CI_WAIT:-1800}" "#$pr" > "$tmp/ci" 2>&1 || rc=$?
    case "$rc" in
      0) break ;;
      2) say "CI do PR #$pr ainda pendente; esperando" ;;
      1)
        reds=$((reds + 1))
        [ "$reds" -lt 2 ] || stop 1 "o CI do PR #$pr falhou duas vezes seguidas"
        say "CI do PR #$pr vermelho: sessão de correção"
        prompt "$n" "O CI do PR #$pr (ticket #$n) falhou. Ache a causa raiz, corrija na mesma branch, rode \`make ci\`, faça push e termine. Saída do sdd-ci.sh:

$(tail -n 40 "$tmp/ci")"
        run_agent ;;
      *) stop 1 "o sdd-ci.sh falhou ao ler o CI do PR #$pr" ;;
    esac
  done
  say "PR #$pr do ticket #$n: esperando o merge"
  rc=0; wait_merge "$pr" || rc=$?
  [ "$rc" -eq 0 ] || stop 1 "o PR #$pr foi fechado sem merge"
  # O GitHub fecha a issue do "Closes #N" logo depois do merge; sem isso, o mesmo
  # ticket voltaria na próxima volta.
  sh "$here/sdd-wait.sh" --repo "$repo" --interval 5 --timeout "${SDD_RELAY_CLOSE_WAIT:-120}" issue-closed "#$n" > /dev/null 2>&1 \
    || { say "PR #$pr mergeado, mas a issue #$n continua aberta; parei"; exit 1; }
  merged="$merged$n "
done
