#!/bin/sh
# Testes do sdd-update-prs.sh (issue #152, spec 015) com um gh falso.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }
export TMPDIR="$tmp" SDD_UPDATE_WAIT=0

# gh falso: $G/pulls.tsv tem "número sha repo-da-cabeça"; o estado de cada PR vem de
# $G/state.N e o atraso de $G/behind.SHA. PUT update-branch grava $G/updated,
# POST de comentário grava $G/comments (um corpo por linha "N") e todo
# método que escreve conta em $G/writes. Comentários já postados voltam em GET.
G="$tmp/gh"; mkdir -p "$G" "$tmp/bin"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
shift # api
method=GET url="" jq="" body=""
while [ $# -gt 0 ]; do
  case "$1" in
    -X) method="$2"; shift 2 ;;
    --jq) jq="$2"; shift 2 ;;
    -f) case "$2" in body=*) body="${2#body=}" ;; esac; shift 2 ;;
    --paginate | --silent) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$method $url" in
  "GET repos/o/r/pulls?state=open"*)
    awk '{ printf "{\"number\":%s,\"head\":{\"sha\":\"%s\",\"repo\":%s}}\n", $1, $2, ($3 == "-" ? "null" : "{\"full_name\":\"" $3 "\"}") }' "$G/pulls.tsv" | jq -s '.' | jq -r "$jq" ;;
  "GET repos/o/r/pulls/"*)
    n="${url##*/}"; echo "$n" >> "$G/gets"
    if [ -f "$G/lazy.$n" ]; then rm "$G/lazy.$n"; echo '{"mergeable_state":null}' | jq -r "$jq"; exit 0; fi
    jq -n --arg s "$(cat "$G/state.$n")" '{mergeable_state: (if $s == "" then null else $s end)}' | jq -r "$jq" ;;
  "GET repos/o/r/compare/main..."*)
    sha="${url##*...}"; echo "{\"behind_by\":$(cat "$G/behind.$sha")}" | jq -r "$jq" ;;
  "GET repos/o/r/issues/"*"/comments"*)
    n="${url#repos/o/r/issues/}"; n="${n%%/*}"
    [ -f "$G/comment.$n" ] && cat "$G/comment.$n"; exit 0 ;;
  "POST repos/o/r/issues/"*"/comments")
    echo w >> "$G/writes"
    n="${url#repos/o/r/issues/}"; n="${n%%/*}"
    printf '%s\n' "$body" >> "$G/comment.$n"; echo "$n" >> "$G/comments"; exit 0 ;;
  "PUT repos/o/r/pulls/"*"/update-branch")
    echo w >> "$G/writes"
    n="${url#repos/o/r/pulls/}"; n="${n%%/*}"; echo "$n" >> "$G/updated"; exit 0 ;;
  *) echo "gh falso: $method $url" >&2; exit 1 ;;
esac
SH
chmod +x "$tmp/bin/gh"
export PATH="$tmp/bin:$PATH" G

# 1 limpo e atrás; 2 em dia; 3 com conflito; 4 de fork; 5 limpo, mas o estado só
# aparece depois de uma releitura (mergeabilidade calculada sob demanda).
cat > "$G/pulls.tsv" <<'EOF'
1 sha1 o/r
2 sha2 o/r
3 sha3 o/r
4 sha4 outro/r
5 sha5 o/r
EOF
echo behind > "$G/state.1"; echo 2 > "$G/behind.sha1"
echo clean > "$G/state.2"; echo 0 > "$G/behind.sha2"
echo dirty > "$G/state.3"; echo 3 > "$G/behind.sha3"
echo behind > "$G/state.4"; echo 1 > "$G/behind.sha4"
echo behind > "$G/state.5"; echo 1 > "$G/behind.sha5"; : > "$G/lazy.5"
: > "$G/writes"; : > "$G/updated"; : > "$G/comments"; : > "$G/gets"
run() { sh "$s/sdd-update-prs.sh" --repo o/r; }

# 015 AC-6: o PR limpo e atrás é atualizado; o que está em dia, não; o com conflito
# recebe um único comentário; o de fork é ignorado (NFR-1).
out="$(run)" || fail "rodada 1 saiu com erro: $out"
printf '%s\n' "$out" | grep -qx 'atualizado #1' || fail "faltou 'atualizado #1': $out"
printf '%s\n' "$out" | grep -qx 'em dia #2' || fail "faltou 'em dia #2': $out"
printf '%s\n' "$out" | grep -qx 'conflito #3 (avisado)' || fail "faltou 'conflito #3 (avisado)': $out"
printf '%s\n' "$out" | grep -qx 'ignorado #4 (branch de fork)' || fail "PR de fork deveria ser ignorado: $out"
printf '%s\n' "$out" | grep -qx 'atualizado #5' || fail "faltou 'atualizado #5': $out"
[ "$(sort "$G/updated" | tr '\n' ' ')" = "1 5 " ] || fail "update-branch nos PRs errados: $(cat "$G/updated")"
[ "$(cat "$G/comments")" = "3" ] || fail "esperava um comentário só no #3: $(cat "$G/comments")"
grep -q 'sdd-update-prs:conflict' "$G/comment.3" || fail "comentário sem o marcador"

# 015 AC-6 (NFR-2): rodar de novo não duplica o comentário.
out="$(run)" || fail "rodada 2 saiu com erro: $out"
printf '%s\n' "$out" | grep -qx 'conflito #3 (avisado)' || fail "rodada 2 sem a linha do conflito: $out"
[ "$(wc -l < "$G/comments" | tr -d ' ')" -eq 1 ] || fail "comentário duplicado: $(cat "$G/comments")"

# 015 AC-7: com SDD_ENGINE=off nada é escrito (nem lido).
: > "$G/writes"; : > "$G/gets"; rm -f "$G/comment.3"; : > "$G/comments"
out="$(SDD_ENGINE=off run)" || fail "SDD_ENGINE=off saiu com erro: $out"
[ ! -s "$G/writes" ] || fail "SDD_ENGINE=off escreveu na API"
[ ! -s "$G/gets" ] || fail "SDD_ENGINE=off leu PRs"

# Falha no update-branch (ex.: cabeça mudou) vira linha 'falhou' e código 1.
cat > "$tmp/bin/gh.fail" <<'SH'
#!/bin/sh
case "$*" in *update-branch*) exit 1 ;; esac
exec "$(dirname "$0")/gh.real" "$@"
SH
mv "$tmp/bin/gh" "$tmp/bin/gh.real"; mv "$tmp/bin/gh.fail" "$tmp/bin/gh"; chmod +x "$tmp/bin/gh"
rc=0; out="$(run)" || rc=$?
[ "$rc" -eq 1 ] || fail "falha no update-branch deveria sair 1, saiu $rc"
printf '%s\n' "$out" | grep -q 'falhou #1' || fail "faltou 'falhou #1': $out"

echo "sdd-update-prs: ok"
