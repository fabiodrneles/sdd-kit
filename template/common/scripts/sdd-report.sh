#!/bin/sh
# sdd-report: quanto custam as sessões do agente em tokens, sem LLM (spec 017).
#
# Uso: sdd-report.sh tokens [--session ARQ | --since AAAA-MM-DD[THH:MM]]
#   Lê os arquivos de sessão do Claude Code (*.jsonl em $SDD_SESSIONS_DIR, padrão
#   ~/.claude/projects), conta cada chamada uma vez (por message.id) e imprime:
#   chamadas, tokens relidos e gravados no cache, de entrada e gerados, e o
#   contexto médio e máximo por chamada (o que cada chamada relê).
# Sem arquivo de sessão, diz "sem dado de tokens" e sai com 0 (NFR-2).
# Códigos: 0 ok; 3 uso. Requer jq.
set -eu

usage() { sed -n '2,10p' "$0"; exit 3; }
[ "${1:-}" = tokens ] || usage
shift
session="" since=""
while [ $# -gt 0 ]; do
  case "$1" in
    --session) session="${2:?}"; shift 2 ;;
    --since) since="${2:?}"; shift 2 ;;
    -h | --help) usage ;;
    *) echo "sdd-report: opção desconhecida: $1" >&2; exit 3 ;;
  esac
done

if [ -n "$session" ]; then
  [ -f "$session" ] || { echo "sdd-report: $session não existe" >&2; exit 3; }
  files="$session"
else
  dir="${SDD_SESSIONS_DIR:-$HOME/.claude/projects}"
  files="$(find "$dir" -name '*.jsonl' -type f 2> /dev/null || true)"
fi
[ -n "$files" ] || { echo "sdd-report: sem dado de tokens (nenhum arquivo de sessão)"; exit 0; }

# Uma linha por chamada (id, hora, uso), depois os totais; --since compara o texto
# ISO da hora, que ordena como a data.
# shellcheck disable=SC2086 # $files é uma lista de caminhos sem espaços
out="$(jq -c --arg since "$since" 'select(.message.usage and .message.id and ((.timestamp // "") >= $since))
    | {id: .message.id, u: .message.usage}' $files 2> /dev/null | jq -rs '
  unique_by(.id) | map(.u) as $u
  | if ($u | length) == 0 then "vazio" else
    ($u | map((.cache_read_input_tokens // 0) + (.cache_creation_input_tokens // 0) + (.input_tokens // 0))) as $ctx
    | [ "chamadas: \($u | length)",
        "relidos do cache: \($u | map(.cache_read_input_tokens // 0) | add)",
        "gravados no cache: \($u | map(.cache_creation_input_tokens // 0) | add)",
        "entrada: \($u | map(.input_tokens // 0) | add)",
        "gerados: \($u | map(.output_tokens // 0) | add)",
        "contexto médio por chamada: \(($ctx | add) / ($ctx | length) | floor)",
        "contexto máximo: \($ctx | max)" ] | .[] end')"
if [ "$out" = vazio ]; then
  echo "sdd-report: sem dado de tokens (nenhuma chamada no período)"
else
  printf '%s\n' "$out"
fi
