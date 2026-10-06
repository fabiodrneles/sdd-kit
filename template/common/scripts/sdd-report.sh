#!/bin/sh
# sdd-report: quanto custam as sessões do agente em tokens, sem LLM (spec 017).
#
# Uso: sdd-report.sh [--repo DONO/REPO] tokens [--session ARQ] [--since DATA] [--until DATA]
#      sdd-report.sh [--repo DONO/REPO] ticket '#N'
#      sdd-report.sh [--repo DONO/REPO] phase '#ÉPICO'
#   tokens: lê os arquivos de sessão do Claude Code (*.jsonl em $SDD_SESSIONS_DIR,
#   padrão ~/.claude/projects), conta cada chamada uma vez (por message.id) e imprime:
#   chamadas, tokens relidos e gravados no cache, de entrada e gerados, e o
#   contexto médio e máximo por chamada (o que cada chamada relê). DATA é ISO (UTC).
#   ticket: os mesmos totais na janela do PR que fecha a issue (do primeiro commit
#   ao último push), o tempo e se o CI ficou verde na primeira rodada (FR-2).
#   phase: uma linha por sub-issue do épico e o total, num único comentário do
#   épico (<!-- sdd-report -->), editado nas rodadas seguintes (FR-3).
# Sem arquivo de sessão, diz "sem dado de tokens" e sai com 0 (NFR-2).
# Códigos: 0 ok; 1 sem PR para o ticket ou falha do gh; 3 uso. Requer jq (e gh para ticket e phase).
set -eu

usage() { sed -n '2,17p' "$0"; exit 3; }
repo=""
[ "${1:-}" != --repo ] || { repo="${2:?}"; shift 2; }
cmd="${1:-}"
case "$cmd" in tokens | ticket | phase) shift ;; *) usage ;; esac

# Totais das chamadas entre $1 e $2 (texto ISO; vazio = sem limite) em JSON, ou
# "null" sem chamada. Compara o texto ISO da hora, que ordena como a data.
usage_json() {
  if [ -n "$session" ]; then
    files="$session"
  else
    files="$(find "${SDD_SESSIONS_DIR:-$HOME/.claude/projects}" -name '*.jsonl' -type f 2> /dev/null || true)"
  fi
  [ -n "$files" ] || { echo null; return; }
  # Cada linha vira um início de turno (mensagem que não é resultado de ferramenta:
  # do dono ou um despertar, FR-4) ou uma chamada; a chamada herda o tipo do último
  # turno do mesmo arquivo. Um despertar é um turno sem origin.kind "human" (ou, sem
  # origin, um texto que começa por aviso do sistema).
  # shellcheck disable=SC2086 # $files é uma lista de caminhos sem espaços
  jq -c --arg since "$1" --arg until "$2" '
    def inwin: ((.timestamp // "") >= $since) and ($until == "" or (.timestamp // "") <= $until);
    def text: .message.content | if type == "string" then . elif type == "array"
      then (map(select(.type == "text") | .text) | first // "") else "" end;
    if .message.usage and .message.id then {k: "c", f: input_filename, id: .message.id, u: .message.usage, w: inwin}
    elif .type == "user" and (.message.content | type == "string" or (type == "array" and all(.[]; .type != "tool_result")))
      then {k: "t", f: input_filename, w: inwin, wake: (if .origin.kind then .origin.kind != "human"
        else (text | test("^\\s*(<task-notification>|<wake |\\[SYSTEM NOTIFICATION)")) end)}
    else empty end' $files 2> /dev/null | jq -cs '
    reduce .[] as $e ({wake: {}, calls: [], turns: 0};
      if $e.k == "t" then .wake[$e.f] = $e.wake | .turns += (if $e.wake and $e.w then 1 else 0 end)
      elif $e.w then .calls += [$e + {wake: (.wake[$e.f] // false)}] else . end)
    | .turns as $turns | (.calls | unique_by(.id)) as $c | ($c | map(.u)) as $u
    | def tot: map((.cache_read_input_tokens // 0) + (.cache_creation_input_tokens // 0) + (.input_tokens // 0) + (.output_tokens // 0)) | add // 0;
    if ($u | length) == 0 then null else
      ($u | map((.cache_read_input_tokens // 0) + (.cache_creation_input_tokens // 0) + (.input_tokens // 0))) as $ctx
      | {calls: ($u | length), read: ($u | map(.cache_read_input_tokens // 0) | add),
         created: ($u | map(.cache_creation_input_tokens // 0) | add),
         input: ($u | map(.input_tokens // 0) | add), output: ($u | map(.output_tokens // 0) | add),
         avg: (($ctx | add) / ($ctx | length) | floor), max: ($ctx | max),
         wakes: $turns, wake_calls: ($c | map(select(.wake)) | length), wake_tokens: ($c | map(select(.wake) | .u) | tot)} end'
}

print_usage() {
  if [ "$1" = null ]; then
    echo "sdd-report: sem dado de tokens${2:-}"
  else
    printf '%s\n' "$1" | jq -r '"chamadas: \(.calls)", "relidos do cache: \(.read)", "gravados no cache: \(.created)",
      "entrada: \(.input)", "gerados: \(.output)", "contexto médio por chamada: \(.avg)", "contexto máximo: \(.max)",
      "despertares sem mensagem do dono: \(.wakes) (\(.wake_calls) chamadas, \(.wake_tokens) tokens)"'
  fi
}

session="" since="" until=""
if [ "$cmd" = tokens ]; then
  while [ $# -gt 0 ]; do
    case "$1" in
      --session) session="${2:?}"; shift 2 ;;
      --since) since="${2:?}"; shift 2 ;;
      --until) until="${2:?}"; shift 2 ;;
      -h | --help) usage ;;
      *) echo "sdd-report: opção desconhecida: $1" >&2; exit 3 ;;
    esac
  done
  [ -z "$session" ] || [ -f "$session" ] || { echo "sdd-report: $session não existe" >&2; exit 3; }
  print_usage "$(usage_json "$since" "$until")" " (nenhuma chamada no período ou nenhum arquivo de sessão)"
  exit 0
fi

[ $# -eq 1 ] || usage
n="${1#\#}"
case "$n" in '' | *[!0-9]*) usage ;; esac
command -v gh > /dev/null 2>&1 || { echo "sdd-report: gh ausente" >&2; exit 1; }
if [ -z "$repo" ]; then
  url="$(git remote get-url origin 2> /dev/null || true)"
  repo="$(printf '%s\n' "$url" | sed -E 's#\.git$##; s#^.*[:/]([^/]+/[^/]+)$#\1#')"
fi
[ -n "$repo" ] || { echo "sdd-report: repositório desconhecido (use --repo)" >&2; exit 3; }

# Os itens de uma lista do GitHub, um por linha ($2: jq de cada item). Paginação
# explícita (page=N): o --paginate do gh segue links repositories/{id}, que alguns
# proxies recusam.
pages() {
  p=1
  while :; do
    out="$(gh api "$1&per_page=100&page=$p" --jq "length, (.[] | $2)")" || return 1
    printf '%s\n' "$out" | sed 1d
    { [ "$(printf '%s\n' "$out" | head -n 1)" -ge 100 ] && [ "$p" -lt 50 ]; } || break
    p=$((p + 1))
  done
}

# Um ticket em JSON: PR, janela, tempo em minutos, CI na primeira rodada e uso; o
# PR é o mais recente cujo corpo começa por "Closes #N" (a convenção do kit).
ticket_json() {
  pr="$(pages "repos/$repo/pulls?state=all" \
    "select((.body // \"\") | test(\"^(Closes|Fixes|Resolves) #$1([^0-9]|\$)\"; \"i\")) | \"\(.number) \(.head.ref)\"")" || return 1
  pr="$(printf '%s\n' "$pr" | sort -rn | head -n 1)"
  [ -n "$pr" ] || { echo null; return; }
  num="${pr%% *}" ref="${pr#* }"
  win="$(pages "repos/$repo/pulls/$num/commits?" '"\(.commit.author.date) \(.commit.committer.date)"' \
    | jq -Rsc 'split("\n") | map(select(. != "") | split(" ")) | {start: (map(.[0]) | min), end: (map(.[1]) | max)}')" || return 1
  # Primeira rodada: as execuções do head da execução mais antiga do PR.
  # shellcheck disable=SC2016 # $s é variável do jq
  ci="$(gh api "repos/$repo/actions/runs?event=pull_request&branch=$ref&per_page=100" \
    --jq '.workflow_runs | if length == 0 then "sem dado" else (min_by(.created_at).head_sha) as $s
      | map(select(.head_sha == $s)) | if all(.conclusion == "success" or .conclusion == "skipped") then "sim"
        elif any(.status != "completed") then "rodando" else "não" end end' 2> /dev/null || echo "sem dado")"
  start="$(printf '%s\n' "$win" | jq -r .start)" end="$(printf '%s\n' "$win" | jq -r .end)"
  u="$(usage_json "$start" "$end")"
  jq -nc --argjson n "$1" --argjson pr "$num" --argjson w "$win" --arg ci "$ci" --argjson u "$u" \
    '{ticket: $n, pr: $pr, start: $w.start, end: $w.end, ci: $ci, usage: $u,
      minutes: ((($w.end | fromdateiso8601) - ($w.start | fromdateiso8601)) / 60 | floor)}'
}

if [ "$cmd" = ticket ]; then
  t="$(ticket_json "$n")" || { echo "sdd-report: falha ao ler o GitHub" >&2; exit 1; }
  [ "$t" != null ] || { echo "sdd-report: nenhum PR fecha #$n" >&2; exit 1; }
  printf '%s\n' "$t" | jq -r '"ticket: #\(.ticket) (PR #\(.pr))", "janela: \(.start) → \(.end) (\(.minutes) min)",
    "CI verde na primeira rodada: \(.ci)"'
  print_usage "$(printf '%s\n' "$t" | jq -c .usage)" " na janela do ticket"
  exit 0
fi

# phase: uma linha por sub-issue, o total e um único comentário no épico.
subs="$(pages "repos/$repo/issues/$n/sub_issues?" '"\(.number)\t\(.title)"')" \
  || { echo "sdd-report: falha ao ler as sub-issues de #$n" >&2; exit 1; }
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
: > "$tmp/rows"
t="$(printf '\t')"
printf '%s\n' "$subs" | while IFS="$t" read -r num title; do
  [ -n "$num" ] || continue
  j="$(ticket_json "$num")" || j=null
  jq -nc --argjson n "$num" --arg title "$title" --argjson j "$j" '{ticket: $n, title: $title} + ($j // {})' >> "$tmp/rows"
done
marker='<!-- sdd-report -->'
jq -rs --arg m "$marker" --arg e "$n" '
  def k: if . == null then "—" else tostring end;
  def tok(f): if .usage == null then "—" else (.usage | f | tostring) end;
  (map(select(.usage != null) | .usage)) as $u
  | $m, "## Relatório da fase (épico #\($e))", "",
    "| Ticket | PR | Tempo (min) | CI verde na 1ª rodada | Chamadas | Relidos | Gravados | Gerados | Despertares (tokens) |",
    "|---|---|---|---|---|---|---|---|---|",
    (.[] | "| #\(.ticket) \(.title | gsub("\\|"; "\\|")) | \(if .pr then "#\(.pr)" else "sem PR" end) | \(.minutes | k) | \(.ci // "—") | \(tok(.calls)) | \(tok(.read)) | \(tok(.created)) | \(tok(.output)) | \(if .usage == null then "—" else "\(.usage.wakes) (\(.usage.wake_tokens))" end) |"),
    "| **Total** | \(map(select(.pr)) | length) PRs | \(map(.minutes // 0) | add // 0) | \(map(select(.ci == "sim")) | length) de \(map(select(.pr)) | length) | \($u | map(.calls) | add // 0) | \($u | map(.read) | add // 0) | \($u | map(.created) | add // 0) | \($u | map(.output) | add // 0) | \($u | map(.wakes) | add // 0) (\($u | map(.wake_tokens) | add // 0)) |",
    "",
    (if ($u | length) < (map(select(.pr)) | length) then "Tickets sem dado de tokens (outro contêiner ou outro agente) aparecem com —." else empty end),
    "Gerado por `sdd-report.sh phase` (spec 017), sem LLM; uma rodada nova edita este comentário."' "$tmp/rows" > "$tmp/body"
cat "$tmp/body"
id="$(pages "repos/$repo/issues/$n/comments?" "select(.body | startswith(\"$marker\")) | .id" | head -n 1)" || id=""
if [ -n "$id" ]; then
  gh api -X PATCH "repos/$repo/issues/comments/$id" -F body=@"$tmp/body" --silent
else
  gh api "repos/$repo/issues/$n/comments" -F body=@"$tmp/body" --silent
fi
echo "sdd-report: relatório gravado no épico #$n"
