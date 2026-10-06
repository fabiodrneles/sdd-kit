#!/bin/sh
# Testes do sdd-pr.sh (issue #137) com um gh falso, um origin bare e um repo temporário.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }
export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@t GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@t TMPDIR="$tmp"

# gh falso: lista de PRs abertos em $G/pulls.json; POST em pulls grava $G/created
# (título, head, base e corpo) e conta em $G/writes; o resto responde o mínimo.
G="$tmp/gh"; mkdir -p "$G" "$tmp/bin"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
shift # api
method=GET url="" jq="" title="" head="" base="" body=""
while [ $# -gt 0 ]; do
  case "$1" in
    -X) method="$2"; shift 2 ;;
    --jq) jq="$2"; shift 2 ;;
    -f) case "$2" in title=*) title="${2#title=}" ;; head=*) head="${2#head=}" ;; base=*) base="${2#base=}" ;; esac; method=POST; shift 2 ;;
    -F) body="${2#body=@}"; method=POST; shift 2 ;;
    --paginate | --silent) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$method $url" in
  "POST repos/o/r/pulls")
    echo w >> "$G/writes"
    { echo "title=$title"; echo "head=$head"; echo "base=$base"; echo "--"; cat "$body"; } > "$G/created"
    echo '{"number":42,"html_url":"https://github.com/o/r/pull/42"}' | jq -r "$jq"; exit 0 ;;
  "POST "*) echo w >> "$G/writes"; exit 0 ;;
  "GET "*pulls\?state=open*) echo "$G/pulls.json" > /dev/null; jq -r "$jq" "$G/pulls.json" | sed '/^null$/d'; exit 0 ;;
  "GET "*issues\?labels*) echo '[{"number":7}]' | jq -r "$jq"; exit 0 ;;
  "GET "*/comments*) exit 0 ;;
  "GET "*pulls/42) echo '{"head":{"sha":"abc1234"}}' | jq -r "$jq"; exit 0 ;;
  "GET "*check-runs*) echo "$url" >> "$G/ci-urls"; echo '{"total_count":1,"check_runs":[{"id":1,"name":"ci","status":"completed","conclusion":"success","app":{"slug":"github-actions"}}]}' | jq -r "$jq"; exit 0 ;;
  "GET "*/status*) echo '{"statuses":[]}' | jq -r "$jq"; exit 0 ;;
  *) echo "gh falso: $method $url" >&2; exit 1 ;;
esac
SH
chmod +x "$tmp/bin/gh"
export PATH="$tmp/bin:$PATH" G
echo '[]' > "$G/pulls.json"
: > "$G/writes"

git init -q --bare -b main "$tmp/origin.git"
r="$tmp/repo"; mkdir -p "$r/scripts" "$r/.github"
cp "$s/sdd-pr.sh" "$s/sdd-ci.sh" "$s/sdd-checkpoint.sh" "$r/scripts/"
cat > "$r/.github/pull_request_template.md" <<'MD'
<!-- Título no formato Conventional Commits -->

Closes # · Épico # · Spec <!-- ex.: Closes #12 -->

## O que muda

<!-- O problema
e a solução. -->

## Checklist

- [ ] `make ci` passa
MD
printf 'ci:\n\t@echo CI-OK\n' > "$r/Makefile"
cd "$r"
git init -q -b main
git remote add origin "$tmp/origin.git"
git add -A; git commit -q -m "chore: base"
git push -q origin main
pr="sh scripts/sdd-pr.sh --repo o/r"
rc=0

# Recusa: main, branch fora do padrão, árvore suja.
out="$($pr 2>&1)" || rc=$?
if ! { [ "$rc" -eq 3 ] && printf '%s\n' "$out" | grep -q 'main'; }; then fail "não recusou a main ($rc): $out"; fi
git checkout -q -b semnumero
rc=0; out="$($pr 2>&1)" || rc=$?
if ! { [ "$rc" -eq 3 ] && printf '%s\n' "$out" | grep -q 'não segue'; }; then fail "não recusou a branch sem número ($rc): $out"; fi
git checkout -q -b feat/9-x
echo sujo > novo.txt
rc=0; out="$($pr 2>&1)" || rc=$?
if ! { [ "$rc" -eq 3 ] && printf '%s\n' "$out" | grep -q 'novo.txt'; }; then fail "não recusou a árvore suja ($rc): $out"; fi
rm novo.txt

# Dry-run: mostra o PR, não envia nem escreve.
echo a > a.txt; git add a.txt; git commit -q -m "feat: add a"
out="$($pr --dry-run --spec 014 2>&1)" || fail "dry-run falhou: $out"
printf '%s\n' "$out" | grep -q "criaria o PR 'feat: add a'" || fail "dry-run sem o título: $out"
printf '%s\n' "$out" | grep -q 'Closes #9 · Épico #7 · Spec 014' || fail "dry-run sem a primeira linha: $out"
git ls-remote --exit-code origin feat/9-x > /dev/null 2>&1 && fail "dry-run enviou a branch"
[ ! -s "$G/writes" ] || fail "dry-run escreveu na API"

# Conflito com a main: aborta o merge e lista o arquivo.
git checkout -q main
echo main > c.txt; git add c.txt; git commit -q -m "feat: c na main"; git push -q origin main
git checkout -q feat/9-x
echo branch > c.txt; git add c.txt; git commit -q -m "feat: c na branch"
rc=0; out="$($pr 2>&1)" || rc=$?
if ! { [ "$rc" -eq 1 ] && printf '%s\n' "$out" | grep -q '  c.txt'; }; then fail "conflito não listou o arquivo ($rc): $out"; fi
if ! { [ -z "$(git status --porcelain)" ] && [ ! -e .git/MERGE_HEAD ]; }; then fail "o merge não foi abortado"; fi
git reset -q --hard HEAD~1
good="$(git rev-parse HEAD)"

# CI local vermelho: só o fim do log e saída não zero; nada enviado.
printf 'ci:\n\t@seq 1 40; exit 1\n' > Makefile; git commit -q -am "test: ci vermelho"
rc=0; out="$($pr 2>&1)" || rc=$?
[ "$rc" -eq 1 ] || fail "make ci vermelho saiu com $rc"
if ! { printf '%s\n' "$out" | grep -qx '40' && ! printf '%s\n' "$out" | grep -qx '5'; }; then fail "não mostrou só o fim do log: $out"; fi
git ls-remote --exit-code origin feat/9-x > /dev/null 2>&1 && fail "enviou com o CI vermelho"
git reset -q --hard "$good"
echo t > t.txt; git add t.txt; git commit -q -m "test: cobre a"  # o título não pode virar o deste commit

# Caminho feliz: merge, CI, push, PR criado com título, corpo e checkpoint.
git checkout -q main; echo m2 > m2.txt; git add m2.txt; git commit -q -m "feat: m2"; git push -q origin main; git checkout -q feat/9-x
out="$($pr --spec 014 2>&1)" || fail "caminho feliz falhou: $out"
git ls-remote --exit-code origin feat/9-x > /dev/null || fail "a branch não foi enviada"
[ -f m2.txt ] || fail "não mesclou origin/main"
printf '%s\n' "$out" | grep -q 'PR #42 aberto' || fail "não reportou o PR: $out"
printf '%s\n' "$out" | grep -q 'check(s) verdes' || fail "não esperou o CI do PR: $out"
grep -q "commits/$(git rev-parse HEAD)/check-runs" "$G/ci-urls" || fail "não esperou o CI pelo SHA enviado: $(cat "$G/ci-urls")"
c="$(cat "$G/created")"
printf '%s\n' "$c" | grep -qx 'title=feat: add a' || fail "título errado: $c"
printf '%s\n' "$c" | grep -qx 'head=feat/9-x' || fail "head errado: $c"
printf '%s\n' "$c" | grep -qx 'Closes #9 · Épico #7 · Spec 014' || fail "corpo sem a primeira linha: $c"
printf '%s\n' "$c" | grep -qx '## O que muda' || fail "corpo sem as seções do template: $c"
printf '%s\n' "$c" | grep -q '<!--\|ex\.: Closes' && fail "comentário HTML no corpo: $c"
printf '%s\n' "$c" | grep -q 'e a solução' && fail "comentário de várias linhas no corpo: $c"
[ "$(wc -l < "$G/writes")" -ge 2 ] || fail "o checkpoint não foi gravado"

# Idempotente: com PR aberto, só reporta (sem segundo POST em pulls).
echo '[{"number":42,"html_url":"https://github.com/o/r/pull/42"}]' > "$G/pulls.json"
rm "$G/created"
out="$($pr --no-wait 2>&1)" || fail "segunda rodada falhou: $out"
printf '%s\n' "$out" | grep -q 'PR #42 já existe: https://github.com/o/r/pull/42' || fail "não reconheceu o PR existente: $out"
[ ! -e "$G/created" ] || fail "criou um segundo PR"

# Branch numerada como o épico aberto (#7): "Refs", nunca "Closes".
echo "[]" > "$G/pulls.json"
git checkout -q -b docs/7-x main
echo e > e.txt; git add e.txt; git commit -q -m "docs: e"
out="$($pr --dry-run 2>&1)" || fail "dry-run do épico falhou: $out"
printf '%s\n' "$out" | grep -q 'Refs #7 · Épico #7 · Spec —' || fail "branch do épico sem Refs: $out"
printf '%s\n' "$out" | grep -q 'Closes #7' && fail "branch do épico com Closes: $out"
# #170: a branch de sincronização do kit não tem ticket e não é recusada.
git checkout -q -b chore/sync-sdd-kit-v9.9.9 main
echo y > y.txt; git add y.txt; git commit -q -m "chore: sync sdd-kit files to v9.9.9"
out="$($pr --dry-run 2>&1)" || fail "branch de sincronização recusada: $out"
printf '%s\n' "$out" | grep -q '| Épico #7 · Spec —$' || fail "sincronização sem a primeira linha sem Closes: $out"
git checkout -q feat/9-x

# --body-file e --title substituem as seções do template.
echo '[]' > "$G/pulls.json"
printf '## O que muda\n\ntexto do agente\n' > "$tmp/body.md"
$pr --no-wait --title "fix: outro" --body-file "$tmp/body.md" > /dev/null 2>&1 || fail "--body-file falhou"
if ! { grep -qx 'title=fix: outro' "$G/created" && grep -qx 'texto do agente' "$G/created"; }; then fail "--title/--body-file ignorados: $(cat "$G/created")"; fi
echo "sdd-pr: ok"
