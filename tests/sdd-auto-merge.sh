#!/bin/sh
# Testes do sdd-auto-merge.sh (a chave SDD_AUTO_MERGE) com um gh e um sdd-ci.sh falsos.
# shellcheck disable=SC2016,SC2086 # o sdd-ci.sh falso expande $G ao rodar; $am é o comando, separado de propósito
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

G="$tmp/gh"; mkdir -p "$G" "$tmp/bin" "$tmp/s"
cp "$root/template/common/scripts/sdd-auto-merge.sh" "$tmp/s/"
# sdd-ci.sh falso: o código vem de $G/ci (0 verde, 1 vermelho, 2 pendente).
printf '#!/bin/sh\nexit "$(cat "$G/ci")"\n' > "$tmp/s/sdd-ci.sh"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
method=GET url="" jq="" value="" input=""
shift # api
while [ $# -gt 0 ]; do
  case "$1" in
    -X) method="$2"; shift 2 ;; --jq) jq="$2"; shift 2 ;;
    -f) case "$2" in value=*) value="$2" ;; inputs*) input="$(printf '%s' "$2" | sed 's/^inputs\[\(.*\)\]=/\1=/')" ;; esac; shift 2 ;;
    -F) method=POST; shift 2 ;; --silent) shift ;; *) url="$1"; shift ;;
  esac
done
case "$method $url" in
  "PUT "*/merge) echo "$url" >> "$G/merged"; exit 0 ;;
  "POST "*/dispatches) w="${url%/dispatches}"; echo "${w##*/} $input" >> "$G/dispatched"; exit 0 ;;
  "POST "*/actions/variables | "PATCH "*/actions/variables/*) echo "$method $url $value" >> "$G/vars"; exit 0 ;;
  "GET "*/actions/variables/*) [ -f "$G/var-exists" ] || exit 1; [ -z "$jq" ] && exit 0; jq -nr --arg v "$(cat "$G/var-value" 2> /dev/null || echo off)" '{value: $v}' | jq -r "$jq"; exit 0 ;;
  "POST "*/comments) echo c >> "$G/commented"; exit 0 ;;
  "GET "*/comments*) f="$G/comments.json" ;;
  "GET "*pulls\?state=open*) f="$G/pulls.json" ;;
  "GET "*/pulls/7) f="$G/pr.json" ;;
  *) echo "gh falso: $method $url" >&2; exit 1 ;;
esac
jq -r "$jq" "$f"
SH
chmod +x "$tmp/bin/gh"
export PATH="$tmp/bin:$PATH" G
reset() {
  rm -f "$G/merged" "$G/dispatched" "$G/vars"
  echo '[{"number":7,"head":{"sha":"abc"}}]' > "$G/pulls.json"
  echo '{"mergeable":true,"body":"Closes #5 · Épico #2"}' > "$G/pr.json"
  echo 0 > "$G/ci"
}
am="sh $tmp/s/sdd-auto-merge.sh --repo o/r"
m="$am merge --sha abc --branch feat/7-x"

# Um comando liga, desliga e mostra a chave (a variável SDD_AUTO_MERGE do repositório).
reset
out="$($am on)" || fail "on falhou: $out"
grep -qx 'POST repos/o/r/actions/variables value=on' "$G/vars" || fail "on não criou a variável: $(cat "$G/vars")"
printf '%s\n' "$out" | grep -qF 'ligado em o/r' || fail "on: $out"
touch "$G/var-exists"
$am off > /dev/null || fail "off falhou"
grep -qx 'PATCH repos/o/r/actions/variables/SDD_AUTO_MERGE value=off' "$G/vars" || fail "off não atualizou: $(cat "$G/vars")"
echo on > "$G/var-value"
$am status | grep -qF 'on em o/r' || fail "status não mostrou on"
rm -f "$G/var-exists"
$am status | grep -qF 'off em o/r' || fail "status sem a variável não mostrou off"
rc=0; $am bogus > /dev/null 2>&1 || rc=$?
[ "$rc" -eq 3 ] || fail "subcomando inválido saiu com $rc"

# Desligada (padrão) ou off: nunca mescla.
reset
out="$(env -u SDD_AUTO_MERGE $m)" || fail "desligada saiu com erro"
printf '%s\n' "$out" | grep -qF 'desligado' || fail "desligada: $out"
SDD_AUTO_MERGE=off $m > /dev/null || fail "off saiu com erro"
[ ! -f "$G/merged" ] || fail "mesclou com a chave desligada"

# Ligada, CI verde e PR mesclável: mescla; fora do Actions, não dispara nada.
reset
out="$(SDD_AUTO_MERGE=on $m)" || fail "ligada falhou: $out"
grep -qF 'pulls/7/merge' "$G/merged" || fail "não mesclou: $out"
[ ! -f "$G/dispatched" ] || fail "disparou fora do Actions"

# Não mescla: checks pendentes ou vermelhos; cabeça nova; conflito.
for c in 1 2; do
  reset; echo "$c" > "$G/ci"
  SDD_AUTO_MERGE=on $m > /dev/null || fail "ci $c saiu com erro"
  [ ! -f "$G/merged" ] || fail "mesclou com o sdd-ci.sh saindo $c"
done
reset; echo '[{"number":7,"head":{"sha":"nova"}}]' > "$G/pulls.json"
SDD_AUTO_MERGE=on $m > /dev/null || fail "cabeça nova saiu com erro"
[ ! -f "$G/merged" ] || fail "mesclou um SHA que não é a cabeça do PR"
reset; echo '{"mergeable":false}' > "$G/pr.json"
SDD_AUTO_MERGE=on $m > /dev/null || fail "conflito saiu com erro"
[ ! -f "$G/merged" ] || fail "mesclou com conflito"

# No Actions, sem token pessoal: depois do merge, dispara o checkpoint e a fase da
# issue fechada; num PR de fechamento, a release; com o token, não dispara nada.
reset
GITHUB_ACTIONS=true SDD_AUTO_MERGE=on $m > /dev/null || fail "Actions falhou"
grep -qx 'sdd-on-merge.yml pr=7' "$G/dispatched" || fail "não disparou o checkpoint: $(cat "$G/dispatched" 2> /dev/null)"
grep -qx 'sdd-on-phase-done.yml issue=5' "$G/dispatched" || fail "não disparou a fase da issue 5"
reset
GITHUB_ACTIONS=true SDD_AUTO_MERGE=on $am merge --sha abc --branch chore/release-v1.2.0 > /dev/null || fail "release falhou"
grep -qx 'sdd-on-release-merge.yml pr=7' "$G/dispatched" || fail "não disparou a release: $(cat "$G/dispatched" 2> /dev/null)"
reset
GITHUB_ACTIONS=true SDD_ENGINE_TOKEN_SET=1 SDD_AUTO_MERGE=on $m > /dev/null || fail "com token falhou"
grep -qF 'pulls/7/merge' "$G/merged" || fail "com token não mesclou"
[ ! -f "$G/dispatched" ] || fail "com token, disparou de novo o que o GitHub já dispara"
echo "sdd-auto-merge: ok"
