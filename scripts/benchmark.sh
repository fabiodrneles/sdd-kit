#!/bin/sh
# benchmark: quanto custa fazer as mesmas tarefas com e sem o sdd-kit (spec 020 FR-6).
#
# Uso: scripts/benchmark.sh [--repo URL] [--tasks ARQ] [--out PASTA] [--dry-run]
#   Clona o projeto (padrão: o sdd-kit-demo) em duas cópias descartáveis, sem push:
#   - sem sdd-kit: tira os arquivos do kit (CLAUDE.md, AGENTS.md, specs, .claude,
#     scripts sdd-*, workflows sdd-*) e dá todas as tarefas a uma sessão comum do
#     claude, como um usuário faria;
#   - com sdd-kit: um agente enxuto por tarefa (só as ferramentas da entrega, sem
#     skills nem MCP), com o pacote da tarefa, como o sdd-relay.sh faz.
#   Mede cada lado pelos arquivos de sessão do Claude Code (sdd-report.sh, sem LLM):
#   tokens, chamadas e contexto por chamada; mais o tempo, o make ci no fim, os
#   commits e as funções de teste novas. Grava PASTA/result.md e PASTA/result.json.
#   --tasks    as tarefas: cada "## " abre uma; "Arquivos:" lista os prováveis
#              (padrão: docs/benchmark/tasks.md)
#   --dry-run  mostra as cópias e os pedidos, sem chamar o agente
#   SDD_BENCH_AGENT: o comando do agente (padrão: claude -p); os dois lados usam o
#   mesmo, com as mesmas permissões.
# Códigos: 0 ok; 1 falha; 3 uso. Requer git, jq, make e o claude.
# shellcheck disable=SC2016 # as crases são do Markdown dos pedidos
set -eu

here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/.." && pwd)"
repo_url="https://github.com/fabiodrneles/sdd-kit-demo.git" tasks="$root/docs/benchmark/tasks.md"
out="" dry=0
while [ $# -gt 0 ]; do
  case "$1" in
    --repo) repo_url="${2:?}"; shift 2 ;;
    --tasks) tasks="${2:?}"; shift 2 ;;
    --out) out="${2:?}"; shift 2 ;;
    --dry-run) dry=1; shift ;;
    -h | --help) sed -n '2,19p' "$0"; exit 0 ;;
    *) echo "benchmark: opção desconhecida: $1" >&2; exit 3 ;;
  esac
done
[ -f "$tasks" ] || { echo "benchmark: $tasks não existe" >&2; exit 3; }
out="${out:-$(mktemp -d)}"
mkdir -p "$out"
report="$root/template/common/scripts/sdd-report.sh"
agent="${SDD_BENCH_AGENT:-claude -p}"
perm="--permission-mode acceptEdits --allowedTools 'Bash(git:*)' 'Bash(make:*)' 'Bash(go:*)' 'Bash(sh:*)' 'Bash(gofmt:*)' 'Bash(ls:*)' 'Bash(grep:*)' 'Bash(sed:*)' 'Bash(cat:*)' 'Bash(head:*)' 'Bash(tail:*)' 'Bash(find:*)' 'Bash(wc:*)'"
lean="--tools Bash Read Edit Write Grep Glob --disable-slash-commands --strict-mcp-config"
say() { echo "benchmark: $*"; }

# As tarefas, uma por arquivo: $out/task-N.md (título na 1ª linha) e task-N.files.
awk -v d="$out" '/^## / { n++; f = d "/task-" n ".md"; print substr($0, 4) > f; next }
  n && /^Arquivos: / { sub(/^Arquivos: /, ""); print > (d "/task-" n ".files"); next }
  n { print >> f }' "$tasks"
count="$(find "$out" -name 'task-*.md' | wc -l | tr -d ' ')"
[ "$count" -gt 0 ] || { echo "benchmark: nenhuma tarefa em $tasks" >&2; exit 3; }

# As duas cópias.
for side in without with; do
  rm -rf "${out:?}/$side"
  git clone -q --depth 1 "$repo_url" "$out/$side"
  git -C "$out/$side" config user.email bench@local
  git -C "$out/$side" config user.name benchmark
done
(
  cd "$out/without"
  rm -rf CLAUDE.md AGENTS.md specs .claude scripts/sdd-*.sh .github/workflows/sdd-*.yml
  # O make ci de um projeto comum: go test direto, sem o guarda de cobertura do kit.
  sed -i.b 's#sh scripts/sdd-cover-guard.sh [^ ]* -- ##' Makefile && rm -f Makefile.b
  git add -A && git commit -qm "chore: the project without sdd-kit"
)
# A base de cada lado: os commits e os testes novos contam a partir dela.
git -C "$out/without" rev-parse HEAD > "$out/base-without"
git -C "$out/with" rev-parse HEAD > "$out/base-with"
removed="CLAUDE.md, AGENTS.md, specs/, .claude/, scripts/sdd-*.sh, .github/workflows/sdd-*.yml"

# Os pedidos: um só, com todas as tarefas, sem o kit; um por tarefa, com o pacote, com o kit.
{
  echo "Implemente as tarefas abaixo neste repositório, cada uma com testes e um commit, e deixe o \`make ci\` passando."
  i=1; while [ "$i" -le "$count" ]; do printf '\n## Tarefa %s: ' "$i"; cat "$out/task-$i.md"; i=$((i + 1)); done
} > "$out/prompt-without.md"
i=1
while [ "$i" -le "$count" ]; do
  {
    printf '# Tarefa %s: ' "$i"; cat "$out/task-$i.md"
    if [ -f "$out/task-$i.files" ]; then
      printf '\n## Arquivos prováveis\n\n'
      tr ',' '\n' < "$out/task-$i.files" | sed 's/^ *//; /^$/d; s/^/- `/; s/$/`/'
    fi
    printf '\n## Entrega\n\n- Rode `make ci` e faça um commit com esta tarefa; não faça push.\n'
    printf -- '- Nunca afrouxe, pule ou apague um teste para o CI passar.\n- Trabalhe só esta tarefa e termine.\n'
  } > "$out/prompt-with-$i.md"
  i=$((i + 1))
done

if [ "$dry" -eq 1 ]; then
  say "[dry-run] cópias em $out/without e $out/with ($count tarefas)"
  say "[dry-run] sem sdd-kit: removidos $removed; um pedido: $(wc -c < "$out/prompt-without.md") bytes"
  i=1; while [ "$i" -le "$count" ]; do say "[dry-run] com sdd-kit, tarefa $i: $(wc -c < "$out/prompt-with-$i.md") bytes"; i=$((i + 1)); done
  exit 0
fi

sessions() { find "${SDD_SESSIONS_DIR:-$HOME/.claude/projects}" -name '*.jsonl' -type f 2> /dev/null | sort; }
# Roda o agente na pasta $1 com o pedido $2 e as opções $3; anota as sessões novas em $4.
run() {
  sessions > "$out/before"
  (cd "$1" && env -u CLAUDE_CODE_SESSION_ID -u CLAUDE_CODE_CHILD_SESSION -u CLAUDE_CODE_REMOTE_SESSION_ID \
    sh -c "$agent $3 $perm" < "$2" > /dev/null 2>&1) || say "aviso: o agente saiu com erro em $1"
  sessions | comm -13 "$out/before" - >> "$4"
}
measure() {
  # shellcheck disable=SC2046 # um --session por arquivo, caminhos sem espaços
  if [ -s "$1" ]; then sh "$report" tokens $(sed 's/^/--session /' "$1") --json; else echo null; fi
}
quality() {
  ci=0; (cd "$1" && make ci > "$out/ci-$2.log" 2>&1) || ci=1
  base="$(cat "$out/base-$2")"
  commits="$(git -C "$1" rev-list --count "$base..HEAD")"
  tests="$(git -C "$1" diff "$base" HEAD | grep -c '^+func Test' || true)"
  jq -nc --argjson ci "$ci" --argjson c "$commits" --argjson t "$tests" '{ci_green: ($ci == 0), commits: $c, new_tests: $t}'
}

: > "$out/s-without"; : > "$out/s-with"
say "sem sdd-kit: uma sessão com as $count tarefas"
t0="$(date +%s)"; run "$out/without" "$out/prompt-without.md" "" "$out/s-without"; t1="$(date +%s)"
i=1
while [ "$i" -le "$count" ]; do
  say "com sdd-kit: tarefa $i num agente enxuto novo"
  run "$out/with" "$out/prompt-with-$i.md" "$lean" "$out/s-with"
  i=$((i + 1))
done
t2="$(date +%s)"
jq -n --argjson a "$(measure "$out/s-without")" --argjson b "$(measure "$out/s-with")" \
  --argjson qa "$(quality "$out/without" without)" --argjson qb "$(quality "$out/with" with)" \
  --argjson ta "$((t1 - t0))" --argjson tb "$((t2 - t1))" --argjson n "$count" \
  '{tasks: $n, without: ($qa + {seconds: $ta, usage: $a}), with: ($qb + {seconds: $tb, usage: $b})}' > "$out/result.json"
jq -r 'def tot: if . == null then null else .read + .created + .input + .output end;
  def k($v): if $v == null then "—" else ($v | tostring) end;
  (.without.usage | tot) as $a | (.with.usage | tot) as $b
  | "| | Sem sdd-kit | Com sdd-kit |", "|---|---|---|",
    "| Tokens no total | \(k($a)) | \(k($b)) |",
    "| Chamadas | \(k(.without.usage.calls)) | \(k(.with.usage.calls)) |",
    "| Contexto médio por chamada | \(k(.without.usage.avg)) | \(k(.with.usage.avg)) |",
    "| Tempo (s) | \(.without.seconds) | \(.with.seconds) |",
    "| make ci verde | \(.without.ci_green) | \(.with.ci_green) |",
    "| Commits | \(.without.commits) | \(.with.commits) |",
    "| Funções de teste novas | \(.without.new_tests) | \(.with.new_tests) |",
    "", (if $a and $b and $b > 0 then "Com sdd-kit: \(($a / $b * 10 | floor) / 10) vezes menos tokens nas \(.tasks) tarefas." else empty end)' \
  "$out/result.json" | tee "$out/result.md"
say "resultado em $out/result.md e $out/result.json"
