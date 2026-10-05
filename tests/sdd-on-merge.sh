#!/bin/sh
# Testes do sdd-on-merge.sh (spec 015 FR-2) com um gh falso.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

# gh falso: PR, épico, sub-issues, comentários e PRs abertos vêm de $G/*.json; POST e
# PATCH gravam o corpo em comments.json e contam as escritas em $G/writes.
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
  "GET "*/sub_issues*) f="$G/sub.json" ;;
  "GET "*/comments*) f="$G/comments.json" ;;
  "GET "*pulls\?state=open*) f="$G/pulls.json" ;;
  "GET "*/pulls/*) f="$G/pr.json" ;;
  "PATCH "*)
    id="${url##*/}"; echo w >> "$G/writes"
    jq --arg id "$id" --rawfile b "$body" 'map(if (.id|tostring) == $id then .body = $b else . end)' "$G/comments.json" > "$G/c" && mv "$G/c" "$G/comments.json"; exit 0 ;;
  "POST "*)
    echo w >> "$G/writes"
    jq --rawfile b "$body" '. + [{id: 99, body: $b}]' "$G/comments.json" > "$G/c" && mv "$G/c" "$G/comments.json"; exit 0 ;;
  *) echo "gh falso: $method $url" >&2; exit 1 ;;
esac
if [ -n "$jq" ]; then jq -r "$jq" "$f"; else cat "$f"; fi
SH
chmod +x "$tmp/bin/gh"
export PATH="$tmp/bin:$PATH" G

mkdir "$tmp/scripts"
cp "$s/sdd-on-merge.sh" "$s/sdd-checkpoint.sh" "$tmp/scripts/"
cd "$tmp"
git init -q -b main repo && cd repo
git -c user.email=t@t -c user.name=t commit -q --allow-empty -m a

echo '[{"number":7}]' > "$G/issues.json"
echo '[{"id":1,"body":"outro comentário"}]' > "$G/comments.json"
echo '[]' > "$G/pulls.json"
# Fora de ordem de propósito; o #20 é o ticket que o PR fecha e ainda consta aberto.
echo '[{"number":22,"title":"T3","state":"open"},{"number":20,"title":"T1","state":"open"},{"number":21,"title":"T2","state":"open"},{"number":19,"title":"T0","state":"closed"}]' > "$G/sub.json"
echo '{"merged":true,"title":"feat: faz o T1","head":{"ref":"feat/20-t1"},"body":"Closes #20 · Épico #7 · Spec 015"}' > "$G/pr.json"
: > "$G/writes"
run() { sh "$tmp/scripts/sdd-on-merge.sh" --repo o/r "$@"; }
body() { jq -r '.[] | select(.id == 99) | .body' "$G/comments.json"; }

# 015 AC-2: o checkpoint cita o PR como feito e o próximo ticket aberto como próximo.
run --pr 30 > /dev/null
b="$(body)"
printf '%s\n' "$b" | head -n 1 | grep -qx '<!-- sdd-checkpoint -->' || fail "comentário sem a marca: $b"
printf '%s\n' "$b" | grep -qx -- '- \*\*Feito:\*\* #30 feat: faz o T1 (mergeado)' || fail "checkpoint sem o PR como feito: $b"
printf '%s\n' "$b" | grep -qx -- '- \*\*Próximo:\*\* #21 T2' || fail "próximo não é o #21: $b"
# shellcheck disable=SC2016 # crases são Markdown
printf '%s\n' "$b" | grep -q -- '- \*\*Branch:\*\* `main` em `' || fail "checkpoint sem a linha da branch: $b"
printf '%s\n' "$b" | grep -q 'alterações locais' && fail "modo CI não deveria trazer o estado local: $b"

# 015 AC-2: sem ticket aberto sobrando, o próximo é o fechamento da fase.
echo '[{"number":20,"title":"T1","state":"open"}]' > "$G/sub.json"
run --pr 30 > /dev/null
body | grep -q 'Próximo:.* fechamento da fase' || fail "sem tickets abertos não apontou o fechamento: $(body)"
echo '[{"number":22,"title":"T3","state":"open"},{"number":21,"title":"T2","state":"open"}]' > "$G/sub.json"

# 015 NFR-2: rodar de novo não duplica nem reescreve o comentário.
run --pr 30 > /dev/null
n="$(wc -l < "$G/writes")"
run --pr 30 > /dev/null
[ "$(wc -l < "$G/writes")" -eq "$n" ] || fail "a segunda rodada escreveu de novo"
[ "$(jq '[.[] | select(.body | startswith("<!-- sdd-checkpoint -->"))] | length' "$G/comments.json")" -eq 1 ] || fail "criou um segundo checkpoint"

# PR de release e PR não mergeado: nada é escrito.
echo '{"merged":true,"title":"chore: release v1.2.3","head":{"ref":"chore/release-v1.2.3"},"body":""}' > "$G/pr.json"
run --pr 31 > /dev/null
echo '{"merged":false,"title":"feat: x","head":{"ref":"feat/1-x"},"body":""}' > "$G/pr.json"
run --pr 32 > /dev/null
[ "$(wc -l < "$G/writes")" -eq "$n" ] || fail "release ou PR não mergeado escreveu"

# 015 AC-7: SDD_ENGINE=off não escreve nada (nem lê o GitHub).
echo '{"merged":true,"title":"feat: outro","head":{"ref":"feat/2-y"},"body":""}' > "$G/pr.json"
SDD_ENGINE=off run --pr 33 | grep -q 'SDD_ENGINE=off' || fail "SDD_ENGINE=off não avisou"
[ "$(wc -l < "$G/writes")" -eq "$n" ] || fail "SDD_ENGINE=off escreveu"
body | grep -q '#33' && fail "SDD_ENGINE=off alterou o checkpoint"

# Workflow do template: só PRs mergeados do próprio repositório, sem checkout do PR, desligável.
w="$root/template/common/.github/workflows/sdd-on-merge.yml"
for t in 'types: [closed]' 'github.event.pull_request.merged' 'head.repo.full_name == github.repository' "vars.SDD_ENGINE != 'off'" 'sdd-on-merge.sh'; do
  grep -qF "$t" "$w" || fail "$w sem '$t'"
done
grep -q 'pull_request_target' "$w" && fail "$w usa pull_request_target"
echo "tests/sdd-on-merge.sh ok"
