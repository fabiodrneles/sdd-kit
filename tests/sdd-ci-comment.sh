#!/bin/sh
# Testes do sdd-ci-comment.sh (spec 015 FR-3) com um gh falso.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }
export TMPDIR="$tmp"

# gh falso: $G/conc é a conclusão do check ("failure" ou "success"), $G/head o SHA
# do PR, $G/comments o corpo do comentário existente (se houver). POST/PATCH gravam
# o corpo em $G/comments e contam em $G/writes e $G/ops.
G="$tmp/gh"; mkdir -p "$G" "$tmp/bin"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
shift # api
method=GET url="" jq="" body=""
while [ $# -gt 0 ]; do
  case "$1" in
    -X) method="$2"; shift 2 ;;
    --jq) jq="$2"; shift 2 ;;
    -F) body="${2#body=@}"; shift 2 ;;
    --paginate | --silent) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$method $url" in
  "GET repos/o/r/pulls/42") echo "{\"head\":{\"sha\":\"$(cat "$G/head")\"}}" | jq -r "$jq"; exit 0 ;;
  "GET repos/o/r/issues/42/comments"*)
    [ -f "$G/comments" ] || exit 0
    jq -n --rawfile b "$G/comments" '[{id:77,body:"outro"},{id:99,body:$b}]' | jq -r "$jq"; exit 0 ;;
  "POST repos/o/r/issues/42/comments") echo w >> "$G/writes"; echo post >> "$G/ops"; cp "$body" "$G/comments"; exit 0 ;;
  "PATCH repos/o/r/issues/comments/99") echo w >> "$G/writes"; echo patch >> "$G/ops"; cp "$body" "$G/comments"; exit 0 ;;
  "GET repos/o/r/commits/abc1234" | "GET repos/o/r/commits/def5678") echo "{\"sha\":\"${url#repos/o/r/commits/}\"}" | jq -r "$jq"; exit 0 ;;
  "GET "*check-runs*)
    echo "{\"total_count\":1,\"check_runs\":[{\"id\":1,\"name\":\"make ci\",\"status\":\"completed\",\"conclusion\":\"$(cat "$G/conc")\",\"app\":{\"slug\":\"github-actions\"}}]}" | jq -r "$jq"; exit 0 ;;
  "GET "*/status*) echo '{"statuses":[]}' | jq -r "$jq"; exit 0 ;;
  "GET repos/o/r/actions/jobs/1") echo '{"steps":[{"name":"Rodar make ci","conclusion":"failure"}]}' | jq -r "$jq"; exit 0 ;;
  "GET repos/o/r/actions/jobs/1/logs") seq 1 40 | sed 's/^/2026-01-01T00:00:00.0000000Z linha /'; exit 0 ;;
  *) echo "gh falso: $method $url" >&2; exit 1 ;;
esac
SH
chmod +x "$tmp/bin/gh"
export PATH="$tmp/bin:$PATH" G
: > "$G/writes"; : > "$G/ops"
cc="sh $s/sdd-ci-comment.sh --repo o/r --pr 42"
marker='<!-- sdd-ci-summary -->'

# Verde sem comentário anterior: não escreve nada.
echo abc1234 > "$G/head"; echo success > "$G/conc"
$cc --sha abc1234 > /dev/null || fail "verde sem comentário falhou"
[ ! -s "$G/writes" ] || fail "escreveu num PR verde sem comentário"

# 015 AC-3: primeira falha cria o comentário, com o check e só o fim do log.
echo failure > "$G/conc"
$cc --sha abc1234 > /dev/null || fail "primeira falha deu erro"
[ "$(cat "$G/ops")" = post ] || fail "a primeira falha não criou o comentário: $(cat "$G/ops")"
grep -qxF "$marker" "$G/comments" || fail "comentário sem o marcador"
grep -q '^FALHA make ci' "$G/comments" || fail "comentário sem a linha do check: $(cat "$G/comments")"
grep -q 'CI vermelho em abc1234' "$G/comments" || fail "comentário sem o título"
grep -q 'linha 40' "$G/comments" || fail "comentário sem o fim do log"
if grep -q 'linha 5$' "$G/comments"; then fail "comentário com mais que o fim do log"; fi

# 015 AC-3: segunda falha (novo commit) edita o mesmo comentário, sem criar outro.
echo def5678 > "$G/head"
$cc --sha def5678 > /dev/null || fail "segunda falha deu erro"
[ "$(cat "$G/ops")" = "post
patch" ] || fail "a segunda falha não editou o comentário: $(cat "$G/ops")"
grep -q 'def5678' "$G/comments" || fail "o comentário não ficou com o commit novo"

# NFR-2: repetir a mesma rodada não duplica (edita de novo).
$cc --sha def5678 > /dev/null || fail "repetição deu erro"
[ "$(grep -c post "$G/ops")" -eq 1 ] || fail "duplicou o comentário: $(cat "$G/ops")"

# Rodada antiga (o SHA não é mais a ponta do PR): não escreve.
n="$(wc -l < "$G/writes")"
$cc --sha abc1234 > /dev/null || fail "rodada antiga deu erro"
[ "$(wc -l < "$G/writes")" -eq "$n" ] || fail "uma rodada antiga sobrescreveu o resumo"

# 015 AC-3: CI verde depois edita o comentário existente para dizer isso.
echo success > "$G/conc"
$cc --sha def5678 > /dev/null || fail "verde com comentário falhou"
grep -q 'CI verde em def5678' "$G/comments" || fail "o comentário não ficou verde: $(cat "$G/comments")"
grep -qxF "$marker" "$G/comments" || fail "o comentário verde perdeu o marcador"
[ "$(grep -c post "$G/ops")" -eq 1 ] || fail "o verde criou outro comentário"

# 015 AC-7: SDD_ENGINE=off não lê nem escreve nada.
echo failure > "$G/conc"; rm "$G/comments"; : > "$G/writes"
SDD_ENGINE=off $cc --sha def5678 > /dev/null || fail "SDD_ENGINE=off deu erro"
[ ! -s "$G/writes" ] || fail "SDD_ENGINE=off escreveu"
[ ! -e "$G/comments" ] || fail "SDD_ENGINE=off criou comentário"

# Uso: sem --sha, erro 3.
rc=0; $cc > /dev/null 2>&1 || rc=$?
[ "$rc" -eq 3 ] || fail "sem --sha deveria sair com 3 ($rc)"
echo "sdd-ci-comment: ok"
