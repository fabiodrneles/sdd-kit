#!/bin/sh
# Testes do sdd-release.sh (issue #142) com um gh falso, um origin bare e um repo temporário.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }
export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@t GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@t TMPDIR="$tmp"

# gh falso: PRs fechados em $G/closed.json (filtrados pelo --jq do script); POST em
# pulls grava $G/created; POST de dispatches grava os argumentos em $G/dispatch;
# todo POST conta em $G/writes.
G="$tmp/gh"; mkdir -p "$G" "$tmp/bin"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
shift # api
method=GET url="" jq="" title="" args=""
while [ $# -gt 0 ]; do
  case "$1" in
    -X) method="$2"; shift 2 ;;
    --jq) jq="$2"; shift 2 ;;
    -f) args="$args $2"; case "$2" in title=*) title="${2#title=}" ;; esac; method=POST; shift 2 ;;
    -F) method=POST; shift 2 ;;
    --paginate | --silent) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$method $url" in
  "POST repos/o/r/pulls") echo w >> "$G/writes"; echo "title=$title" > "$G/created"
    echo '{"number":50,"html_url":"https://github.com/o/r/pull/50"}' | jq -r "$jq"; exit 0 ;;
  "POST repos/o/r/actions/workflows/release-tag.yml/dispatches") echo w >> "$G/writes"; echo "$args" > "$G/dispatch"; exit 0 ;;
  "POST "*) echo w >> "$G/writes"; exit 0 ;;
  "GET "*pulls\?state=closed*) jq -r "$jq" "$G/closed.json"; exit 0 ;;
  "GET "*pulls\?state=open*) exit 0 ;;
  "GET "*issues\?labels*) echo '[{"number":7}]' | jq -r "$jq"; exit 0 ;;
  "GET "*/comments*) exit 0 ;;
  *) echo "gh falso: $method $url" >&2; exit 1 ;;
esac
SH
chmod +x "$tmp/bin/gh"
# go-release-manager falso: imprime $GRM_NEXT (vazio = nada a lançar).
# shellcheck disable=SC2016 # o script falso expande $1 e $GRM_NEXT ao rodar
printf '#!/bin/sh\n[ "$1" = next ] && [ -n "${GRM_NEXT:-}" ] && echo "$GRM_NEXT"\nexit 0\n' > "$tmp/bin/go-release-manager"
chmod +x "$tmp/bin/go-release-manager"
export PATH="$tmp/bin:$PATH" G
: > "$G/writes"
cat > "$G/closed.json" <<'JSON'
[
 {"number":10,"title":"feat: antigo","merged_at":"2000-01-01T00:00:00Z"},
 {"number":11,"title":"feat(x): add x","merged_at":"2999-01-01T00:00:00Z"},
 {"number":12,"title":"fix: bug","merged_at":"2999-01-02T00:00:00Z"},
 {"number":13,"title":"refactor: ref","merged_at":"2999-01-03T00:00:00Z"},
 {"number":14,"title":"chore: skip","merged_at":"2999-01-04T00:00:00Z"},
 {"number":15,"title":"docs: skip","merged_at":"2999-01-05T00:00:00Z"},
 {"number":16,"title":"Sem prefixo","merged_at":"2999-01-06T00:00:00Z"},
 {"number":17,"title":"feat: fechado sem merge","merged_at":null}
]
JSON

git init -q --bare -b main "$tmp/origin.git"
r="$tmp/repo"; mkdir -p "$r/scripts" "$r/.github"
cp "$s/sdd-release.sh" "$s/sdd-pr.sh" "$s/sdd-ci.sh" "$s/sdd-checkpoint.sh" "$r/scripts/"
printf 'ci:\n\t@echo CI-OK\n' > "$r/Makefile"
printf '1.0.0\n' > "$r/VERSION"
printf 'baixe sdd/v1.0.0/x e v1.0.0 fica; 1.0.0 fica\n' > "$r/README.md"
cat > "$r/.sdd-release" <<'EOF'
# comentário
VERSION
README.md sdd/v@V@/x
EOF
cat > "$r/CHANGELOG.md" <<'EOF'
# Changelog

## [Unreleased]

### Adicionado

- item manual

## [1.0.0] - 2026-01-01

Primeira.
EOF
cd "$r"
git init -q -b main
git remote add origin "$tmp/origin.git"
git add -A; git commit -q -m "chore: base"
git tag -a v1.0.0 -m v1.0.0
git push -q origin main v1.0.0
rel="sh scripts/sdd-release.sh --repo o/r"

# Argumentos inválidos e árvore suja.
rc=0; out="$($rel 2>&1)" || rc=$?
[ "$rc" -eq 3 ] || fail "sem versão saiu com $rc: $out"
rc=0; out="$($rel 1.1 2>&1)" || rc=$?
[ "$rc" -eq 3 ] || fail "versão inválida saiu com $rc: $out"
echo sujo > novo.txt
rc=0; out="$($rel 1.1.0 2>&1)" || rc=$?
if ! { [ "$rc" -eq 3 ] && printf '%s\n' "$out" | grep -q 'novo.txt'; }; then fail "não recusou a árvore suja ($rc): $out"; fi
git rev-parse -q --verify refs/heads/chore/release-v1.1.0 > /dev/null && fail "criou a branch com a árvore suja"
rm novo.txt

# --dry-run: mostra o bloco do CHANGELOG e não escreve nada.
head="$(git rev-parse HEAD)"
out="$($rel --dry-run 1.1.0 2>&1)" || fail "dry-run falhou: $out"
printf '%s\n' "$out" | grep -q '## \[1.1.0\]' || fail "dry-run sem o bloco: $out"
printf '%s\n' "$out" | grep -q 'add x (#11)' || fail "dry-run sem o rascunho: $out"
printf '%s\n' "$out" | grep -q 'subiria a versão em VERSION' || fail "dry-run sem os bumps: $out"

# Sem X.Y.Z, a versão vem do go-release-manager; com X.Y.Z diferente, avisa.
out="$(GRM_NEXT=v1.1.0 $rel --dry-run 2>&1)" || fail "dry-run sem versão falhou: $out"
printf '%s\n' "$out" | grep -q 'v1.1.0 calculada pelo go-release-manager' || fail "não usou o go-release-manager: $out"
printf '%s\n' "$out" | grep -q '## \[1.1.0\]' || fail "versão do go-release-manager não entrou no CHANGELOG: $out"
out="$(GRM_NEXT=v1.1.0 $rel --dry-run 2.0.0 2>&1)" || fail "dry-run com release-as falhou: $out"
printf '%s\n' "$out" | grep -q 'calcula v1.1.0; usando v2.0.0' || fail "sem aviso de divergência: $out"
out="$($rel --dry-run 2>&1)" && fail "sem versão nem go-release-manager deveria falhar: $out"
printf '%s\n' "$out" | grep -q 'go install github.com/fabiodrneles/go-release-manager' || fail "não explicou como instalar: $out"
if ! { [ -z "$(git status --porcelain)" ] && [ "$(git rev-parse HEAD)" = "$head" ] && [ "$(git rev-parse --abbrev-ref HEAD)" = main ]; }; then fail "dry-run mexeu no repositório"; fi
git rev-parse -q --verify refs/heads/chore/release-v1.1.0 > /dev/null && fail "dry-run criou a branch"
[ ! -s "$G/writes" ] || fail "dry-run escreveu na API"

# Preparar: CHANGELOG, bumps, commit, branch enviada e PR.
out="$($rel 1.1.0 2>&1)" || fail "preparar falhou: $out"
[ "$(git rev-parse --abbrev-ref HEAD)" = chore/release-v1.1.0 ] || fail "branch errada"
[ "$(git log -1 --format=%s)" = "chore: release v1.1.0" ] || fail "commit errado: $(git log -1 --format=%s)"
git ls-remote --exit-code origin chore/release-v1.1.0 > /dev/null || fail "a branch não foi enviada"
grep -qx 'title=chore: release v1.1.0' "$G/created" || fail "título do PR errado: $(cat "$G/created")"
printf '%s\n' "$out" | grep -q 'PR #50 aberto' || fail "não abriu o PR: $out"
c="$(cat CHANGELOG.md)"
today="$(date +%Y-%m-%d)"
# Unreleased vazio e a versão logo abaixo, com a data de hoje.
[ "$(sed -n '3,5p' CHANGELOG.md | tr '\n' '|')" = "## [Unreleased]||## [1.1.0] - $today|" ] || fail "cabeçalho errado: $(sed -n '1,8p' CHANGELOG.md)"
# O item manual ficou na versão, antes do rascunho da mesma seção.
if ! printf '%s\n' "$c" | awk '/^- item manual/ { m = NR } /^- add x \(#11\)/ { a = NR } END { exit !(m && a && m < a) }'; then fail "item manual e rascunho fora de ordem: $c"; fi
printf '%s\n' "$c" | grep -qx -- '- bug (#12)' || fail "sem o fix: $c"
printf '%s\n' "$c" | grep -qx -- '- ref (#13)' || fail "sem o refactor: $c"
printf '%s\n' "$c" | grep -qx -- '- Sem prefixo (#16)' || fail "sem o título sem prefixo: $c"
printf '%s\n' "$c" | grep -q '#10\|#14\|#15\|#17' && fail "entrou PR que deveria ficar de fora: $c"
# Cada item sob o título certo.
sec() { printf '%s\n' "$c" | awk -v h="$1" '/^### /{on = ($0 == h)} /^## /{on=0} on'; }
sec '### Adicionado' | grep -q 'add x' || fail "feat fora de Adicionado"
sec '### Corrigido' | grep -q 'bug (#12)' || fail "fix fora de Corrigido"
sec '### Alterado' | grep -q 'ref (#13)' || fail "refactor fora de Alterado"
sec '### Alterado' | grep -q 'Sem prefixo' || fail "sem prefixo fora de Alterado"
printf '%s\n' "$c" | grep -q '^## \[1.0.0\] - 2026-01-01' || fail "perdeu a versão anterior"
# Bumps: VERSION inteira; README só o trecho com v@V@.
[ "$(cat VERSION)" = "1.1.0" ] || fail "VERSION não subiu: $(cat VERSION)"
[ "$(cat README.md)" = "baixe sdd/v1.1.0/x e v1.0.0 fica; 1.0.0 fica" ] || fail "README errado: $(cat README.md)"

# --tag antes do merge: recusa (a main não tem a entrada).
git checkout -q main
rc=0; out="$($rel --tag 1.1.0 2>&1)" || rc=$?
if ! { [ "$rc" -eq 3 ] && printf '%s\n' "$out" | grep -q 'CHANGELOG'; }; then fail "--tag sem o merge ($rc): $out"; fi

# Merge do PR de fechamento (pelo dono) e --tag com o workflow Release tag.
git merge -q --ff-only chore/release-v1.1.0
mkdir -p .github/workflows
printf 'name: Release tag\non: workflow_dispatch\n' > .github/workflows/release-tag.yml
printf 'echo "$@" >> "%s/check.log"\n' "$tmp" > scripts/sdd-release-check.sh
git add -A; git commit -q -m "ci: workflow"; git push -q origin main
: > "$G/writes"
out="$($rel --tag --dry-run 1.1.0 2>&1)" || fail "--tag --dry-run falhou: $out"
[ ! -s "$G/writes" ] || fail "--tag --dry-run escreveu na API"
[ ! -e "$tmp/check.log" ] || fail "--tag --dry-run rodou o pre"
out="$($rel --tag 1.1.0 2>&1)" || fail "--tag falhou: $out"
d="$(cat "$G/dispatch")"
case "$d" in *"ref=main"*"inputs[release-as]=v1.1.0"*"inputs[ref]=$(git rev-parse origin/main)"*) ;; *) fail "dispatch errado: $d" ;; esac
git ls-remote --exit-code origin refs/tags/v1.1.0 > /dev/null 2>&1 && fail "criou a tag direto com o workflow"
grep -qx 'pre v1.1.0' "$tmp/check.log" || fail "não rodou o pre: $(cat "$tmp/check.log")"
printf '%s\n' "$out" | grep -q 'sdd-release-check.sh post v1.1.0' || fail "sem o post: $out"

# --tag sem workflow: tag anotada enviada para a origin/main.
git rm -q .github/workflows/release-tag.yml; git commit -q -m "ci: sem workflow"; git push -q origin main
: > "$G/writes"
out="$($rel --tag 1.1.0 2>&1)" || fail "--tag sem workflow falhou: $out"
[ ! -s "$G/writes" ] || fail "tag direta usou a API"
git ls-remote --exit-code origin refs/tags/v1.1.0 > /dev/null || fail "a tag não foi enviada"
[ "$(git cat-file -t v1.1.0)" = tag ] || fail "a tag não é anotada"
[ "$(git rev-parse 'v1.1.0^{commit}')" = "$(git rev-parse origin/main)" ] || fail "tag fora da origin/main"
rc=0; out="$($rel --tag 1.1.0 2>&1)" || rc=$?
[ "$rc" -eq 3 ] || fail "tag repetida saiu com $rc: $out"

# Tentativa anterior: com a branch chore/release-v1.2.0 já na origin (ou só local), reaproveita
# e traz a origin/main em vez de falhar.
git checkout -q main
out="$($rel 1.2.0 2>&1)" || fail "primeira tentativa 1.2.0 falhou: $out"
git checkout -q main
git branch -q -D chore/release-v1.2.0
echo novo > extra.txt; git add -A; git commit -q -m "docs: extra"; git push -q origin main
out="$($rel 1.2.0 2>&1)" || fail "branch existente na origin não foi reaproveitada: $out"
printf '%s\n' "$out" | grep -q 'já existe; reaproveitando' || fail "sem aviso de reaproveitamento: $out"
[ "$(git rev-parse --abbrev-ref HEAD)" = chore/release-v1.2.0 ] || fail "branch errada após reaproveitar"
git merge-base --is-ancestor origin/main HEAD || fail "não trouxe a origin/main para a branch reaproveitada"
[ "$(grep -c '^## \[1.2.0\]' CHANGELOG.md)" -eq 1 ] || fail "CHANGELOG duplicou a versão: $(sed -n '1,12p' CHANGELOG.md)"
[ "$(cat VERSION)" = "1.2.0" ] || fail "VERSION errada ao reaproveitar"
git checkout -q main
git push -q origin --delete chore/release-v1.2.0
out="$($rel 1.2.0 2>&1)" || fail "branch só local não foi reaproveitada: $out"
printf '%s\n' "$out" | grep -q 'já existe; reaproveitando' || fail "sem aviso (branch local): $out"
echo "sdd-release: ok"
