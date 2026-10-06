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
# Sem estado próprio (NFR-1): tudo vem do GitHub; matar e rodar de novo retoma.
# Códigos: 0 fase sem ticket aberto (ou --max atingido); 1 parou (agente sem PR,
#          PR fechado sem merge); 3 uso. Requer gh e jq.
set -eu

usage() { sed -n '2,19p' "$0"; exit 3; }
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

runs=0 merged=" "
while :; do
  epic="$(gh api "repos/$repo/issues?labels=%C3%A9pico&state=open&per_page=1" --jq '.[0].number // empty')"
  [ -n "$epic" ] || { say "nenhum épico aberto; Próximo: perguntar ao dono a próxima fase"; exit 0; }
  n="$(gh api "repos/$repo/issues/$epic/sub_issues?per_page=100" --jq '[.[] | select(.state == "open")][0].number // empty')"
  if [ -z "$n" ]; then
    say "épico #$epic sem ticket aberto; Próximo: PR de fechamento da fase (o motor abre)"
    exit 0
  fi
  # Trava: um ticket já mergeado nesta rodada que volta aberto (lista atrasada, PR
  # sem "Closes") para o relé, em vez de chamar o agente de novo ou esperar sem fim.
  case "$merged" in *" $n "*) say "o ticket #$n voltou aberto depois do merge; parei"; exit 1 ;; esac
  pr="$(pr_for "$n")"
  if [ -z "$pr" ]; then
    if [ -n "$max" ] && [ "$runs" -ge "$max" ]; then say "--max $max atingido; Próximo: #$n"; exit 0; fi
    sh "$here/sdd-context.sh" --repo "$repo" "#$n" > "$tmp/pack"
    {
      cat "$tmp/pack"
      printf '\n## Sua tarefa nesta sessão\n\n'
      echo "Trabalhe só o ticket #$n: implemente, rode \`make ci\`, abra o PR com \`sdd-pr.sh --no-wait\` e termine."
      echo "Não espere o merge nem comece outro ticket: o relé (sdd-relay.sh) cuida disso."
    } > "$tmp/prompt"
    if [ "$dry" -eq 1 ]; then
      say "[dry-run] chamaria '$agent' para o ticket #$n com $(wc -c < "$tmp/prompt") bytes:"
      cat "$tmp/prompt"
      exit 0
    fi
    say "ticket #$n (épico #$epic): agente novo"
    runs=$((runs + 1))
    sh -c "$agent" < "$tmp/prompt" || say "aviso: o agente saiu com erro no ticket #$n"
    pr="$(pr_for "$n")"
    [ -n "$pr" ] || { say "o agente terminou sem PR para #$n; parei"; exit 1; }
  fi
  say "PR #$pr do ticket #$n: esperando o merge"
  rc=0; wait_merge "$pr" || rc=$?
  [ "$rc" -eq 0 ] || { say "PR #$pr fechado sem merge; parei"; exit 1; }
  # O GitHub fecha a issue do "Closes #N" logo depois do merge; sem isso, o mesmo
  # ticket voltaria na próxima volta.
  sh "$here/sdd-wait.sh" --repo "$repo" --interval 5 --timeout "${SDD_RELAY_CLOSE_WAIT:-120}" issue-closed "#$n" > /dev/null 2>&1 \
    || { say "PR #$pr mergeado, mas a issue #$n continua aberta; parei"; exit 1; }
  merged="$merged$n "
done
