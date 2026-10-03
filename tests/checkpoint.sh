#!/bin/sh
# Testes da spec 014 (sdd-checkpoint.sh) com um gh falso.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts/sdd-checkpoint.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

# gh falso: issues, comentários e PRs vêm de $G/*.json; POST e PATCH gravam o
# corpo em comments.json (com um rodapé, como o servidor real) e contam as escritas em $G/writes.
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
  "GET "*issues\?labels*) f="$G/issues.json" ;;
  "GET "*/comments*) f="$G/comments.json" ;;
  "GET "*/pulls*) f="$G/pulls.json" ;;
  "PATCH "*)
    id="${url##*/}"; echo w >> "$G/writes"
    jq --arg id "$id" --rawfile b "$body" 'map(if (.id|tostring) == $id then .body = $b + "\n---\n_rodapé do servidor_" else . end)' "$G/comments.json" > "$G/c" && mv "$G/c" "$G/comments.json"; exit 0 ;;
  "POST "*)
    echo w >> "$G/writes"
    jq --rawfile b "$body" '. + [{id: 99, body: ($b + "\n---\n_rodapé do servidor_")}]' "$G/comments.json" > "$G/c" && mv "$G/c" "$G/comments.json"; exit 0 ;;
  *) echo "gh falso: $method $url" >&2; exit 1 ;;
esac
jq -r "$jq" "$f"
SH
chmod +x "$tmp/bin/gh"
export PATH="$tmp/bin:$PATH" G
echo '[{"number":7}]' > "$G/issues.json"
echo '[{"id":1,"body":"outro comentário"}]' > "$G/comments.json"
echo '[{"number":12,"title":"feat: x","head":{"ref":"feat/12-x"}}]' > "$G/pulls.json"
: > "$G/writes"

r="$tmp/repo"; mkdir "$r"; cd "$r"
git init -q -b feat/12-x
git -c user.email=t@t -c user.name=t commit -q --allow-empty -m a
body() { jq -r '.[] | select(.id == 99) | .body' "$G/comments.json"; }

# 014 AC-1: save cria o comentário e depois o edita.
sh "$s" --repo o/r save "T1 pronto" "abrir o PR do T2" > /dev/null
b="$(body)"
printf '%s\n' "$b" | head -n 1 | grep -qx '<!-- sdd-checkpoint -->' || fail "comentário sem a marca: $b"
printf '%s\n' "$b" | grep -q 'feat/12-x' || fail "checkpoint sem a branch: $b"
printf '%s\n' "$b" | grep -qx -- '- \*\*Próximo:\*\* abrir o PR do T2' || fail "checkpoint sem o próximo: $b"
printf '%s\n' "$b" | grep -q '#12 feat: x' || fail "checkpoint sem os PRs abertos: $b"
sh "$s" --repo o/r save "T2 pronto" "merge do T2" > /dev/null
[ "$(jq '[.[] | select(.body | startswith("<!-- sdd-checkpoint -->"))] | length' "$G/comments.json")" -eq 1 ] || fail "save criou um segundo checkpoint"
body | grep -q 'merge do T2' || fail "save não editou o checkpoint"

# 014 AC-2: show imprime; sem épico, sai com 0 e avisa.
sh "$s" --repo o/r show | grep -q 'merge do T2' || fail "show não imprimiu o checkpoint"
echo '[]' > "$G/issues.json"
out="$(sh "$s" --repo o/r show)" || fail "show sem épico falhou"
printf '%s\n' "$out" | grep -q 'nenhum épico aberto' || fail "show sem épico não avisou: $out"
sh "$s" --repo o/r save a b > /dev/null || fail "save sem épico falhou"
echo '[{"number":7}]' > "$G/issues.json"

# 014 AC-3: auto não escreve sem mudança; com commit novo, atualiza e mantém o resto.
n="$(wc -l < "$G/writes")"
sh "$s" --repo o/r auto > /dev/null
[ "$(wc -l < "$G/writes")" -eq "$n" ] || fail "auto escreveu sem mudança"
git -c user.email=t@t -c user.name=t commit -q --allow-empty -m b
sh "$s" --repo o/r auto > /dev/null
body | grep -q "$(git rev-parse --short HEAD)" || fail "auto não registrou o commit novo"
body | grep -qx -- '- \*\*Feito:\*\* T2 pronto' || fail "auto perdeu o feito"

# 014 AC-4: hooks de início de sessão e hook Stop do template.
for h in "$root"/template/*/.claude/hooks/session-start.sh; do
  grep -q 'sdd-checkpoint.sh show' "$h" || fail "$h não mostra o checkpoint"
done
jq -e '.hooks.Stop[0].hooks[0].command | test("sdd-checkpoint.sh auto")' "$root/template/common/.claude/settings.json" > /dev/null ||
  fail "settings.json do template sem o hook Stop"
echo "tests/checkpoint.sh ok"
