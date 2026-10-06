#!/bin/sh
# Testes do sdd-on-phase-done.sh (spec 015 FR-4, issue #150) com gh e go-release-manager falsos
# e um origin bare: o sdd-release.sh e o sdd-pr.sh são os de verdade.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }
export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@t GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@t TMPDIR="$tmp"

# gh falso: épico aberto #7; sub-issues em $G/subs.json ("number state" pelo --jq);
# PRs abertos em $G/open.json; POST em pulls grava $G/created e conta em $G/writes.
G="$tmp/gh"; mkdir -p "$G" "$tmp/bin"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
shift # api
method=GET url="" jq="" title=""
while [ $# -gt 0 ]; do
  case "$1" in
    -X) method="$2"; shift 2 ;;
    --jq) jq="$2"; shift 2 ;;
    -f) case "$2" in title=*) title="${2#title=}" ;; esac; method=POST; shift 2 ;;
    -F) method=POST; shift 2 ;;
    --paginate | --silent) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$method $url" in
  "POST repos/o/r/pulls") echo w >> "$G/writes"; echo "title=$title" >> "$G/created"
    echo '{"number":50,"html_url":"https://github.com/o/r/pull/50"}' | jq -r "$jq"; exit 0 ;;
  "POST "*/dispatches) echo w >> "$G/writes"; echo "$url" >> "$G/dispatched"; exit 0 ;;
  "POST "*) echo w >> "$G/writes"; exit 0 ;;
  "GET "*pulls\?state=open\&head=*) exit 0 ;;
  "GET "*pulls\?state=open*) jq -r "$jq" "$G/open.json"; exit 0 ;;
  "GET "*pulls\?state=closed*) exit 0 ;;
  "GET "*issues\?labels*) echo '[{"number":7}]' | jq -r "$jq"; exit 0 ;;
  "GET "*issues/7/sub_issues*) jq -r "$jq" "$G/subs.json"; exit 0 ;;
  "GET "*/comments*) exit 0 ;;
  *) echo "gh falso: $method $url" >&2; exit 1 ;;
esac
SH
chmod +x "$tmp/bin/gh"
# go-release-manager falso: imprime $GRM_NEXT.
# shellcheck disable=SC2016 # o script falso expande $1 e $GRM_NEXT ao rodar
printf '#!/bin/sh\n[ "$1" = next ] && [ -n "${GRM_NEXT:-}" ] && echo "$GRM_NEXT"\nexit 0\n' > "$tmp/bin/go-release-manager"
chmod +x "$tmp/bin/go-release-manager"
export PATH="$tmp/bin:$PATH" G GRM_NEXT=v1.1.0
echo '[]' > "$G/open.json"
: > "$G/writes"; : > "$G/created"

git init -q --bare -b main "$tmp/origin.git"
r="$tmp/repo"; mkdir -p "$r/scripts"
cp "$s/sdd-on-phase-done.sh" "$s/sdd-release.sh" "$s/sdd-pr.sh" "$s/sdd-ci.sh" "$s/sdd-checkpoint.sh" "$r/scripts/"
# #182: no motor o make ci local é pulado (o runner não tem as ferramentas): um ci
# que falha não pode impedir o PR de fechamento.
printf 'ci:\n\t@echo sem-ferramentas; exit 127\n' > "$r/Makefile"
printf '# Changelog\n\n## [Unreleased]\n\n### Adicionado\n\n- item\n\n## [1.0.0] - 2026-01-01\n\nPrimeira.\n' > "$r/CHANGELOG.md"
cd "$r"
git init -q -b main
git remote add origin "$tmp/origin.git"
git add -A; git commit -q -m "chore: base"
git tag -a v1.0.0 -m v1.0.0
git push -q origin main v1.0.0
run="sh scripts/sdd-on-phase-done.sh --repo o/r"
subs() { printf '[{"number":1,"state":"%s"},{"number":2,"state":"%s"},{"number":3,"state":"%s"}]\n' "$1" "$2" "$3" > "$G/subs.json"; }
created() { grep -c . "$G/created" || true; }

# Uso inválido.
rc=0; out="$($run 2>&1)" || rc=$?
[ "$rc" -eq 3 ] || fail "sem a issue saiu com $rc: $out"

# 015 AC-4: ticket fechado com outros abertos: nada.
subs closed open closed
out="$($run 2 2>&1)" || fail "ticket com outros abertos falhou: $out"
printf '%s\n' "$out" | grep -q 'ainda tem tickets abertos' || fail "sem o aviso: $out"
[ ! -s "$G/writes" ] || fail "escreveu com tickets abertos"
git rev-parse -q --verify refs/heads/chore/release-v1.1.0 > /dev/null && fail "criou a branch com tickets abertos"

# Issue que não é sub-issue do épico: nada.
subs closed closed closed
out="$($run 99 2>&1)" || fail "issue de fora falhou: $out"
printf '%s\n' "$out" | grep -q 'não é sub-issue' || fail "sem o aviso: $out"
[ ! -s "$G/writes" ] || fail "escreveu para issue de fora do épico"

# 015 AC-7: SDD_ENGINE=off, nem lê nem escreve (o gh falso falharia sem o subs.json).
rm "$G/subs.json"
out="$(SDD_ENGINE=off $run 3 2>&1)" || fail "SDD_ENGINE=off falhou: $out"
[ ! -s "$G/writes" ] || fail "SDD_ENGINE=off escreveu"
git rev-parse -q --verify refs/heads/chore/release-v1.1.0 > /dev/null && fail "SDD_ENGINE=off criou a branch"

# 015 AC-4: PR de fechamento já aberto: nada (idempotente).
subs closed closed closed
echo '[{"number":60,"head":{"ref":"chore/release-v1.1.0"}}]' > "$G/open.json"
out="$($run 3 2>&1)" || fail "com PR existente falhou: $out"
printf '%s\n' "$out" | grep -q 'já há PR de fechamento' || fail "sem o aviso: $out"
[ ! -s "$G/writes" ] || fail "escreveu com PR de fechamento aberto"
echo '[]' > "$G/open.json"

# 015 AC-4: último ticket fechado: o PR chore/release-v1.1.0 é aberto, uma vez.
out="$($run 3 2>&1)" || fail "último ticket falhou: $out"
git ls-remote --exit-code origin chore/release-v1.1.0 > /dev/null || fail "a branch de release não foi enviada: $out"
grep -qx 'title=chore: release v1.1.0' "$G/created" || fail "título do PR errado: $(cat "$G/created")"
[ "$(created)" -eq 1 ] || fail "abriu $(created) PRs"
grep -q '^## \[1.1.0\]' CHANGELOG.md || fail "CHANGELOG sem a versão"
# #182: make ci local pulado e, sem SDD_ENGINE_TOKEN, o CI disparado na branch do PR.
printf '%s\n' "$out" | grep -q 'make ci local pulado' || fail "não pulou o make ci local: $out"
grep -q 'actions/workflows/ci.yml/dispatches' "$G/dispatched" || fail "não disparou o CI do PR de fechamento"

# Segunda execução (agora o PR existe): nada novo.
echo '[{"number":50,"head":{"ref":"chore/release-v1.1.0"}}]' > "$G/open.json"
out="$($run 3 2>&1)" || fail "segunda execução falhou: $out"
[ "$(created)" -eq 1 ] || fail "a segunda execução abriu outro PR"
echo "sdd-on-phase-done: ok"
