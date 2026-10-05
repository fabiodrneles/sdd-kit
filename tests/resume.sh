#!/bin/sh
# Testes do sdd-resume.sh (spec 014, retomada em um comando) com um gh falso.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts/sdd-resume.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }
git_() { git -c user.email=t@t -c user.name=t "$@"; }

# gh falso (REST): épico, comentários, PRs, check-runs e sub-issues vêm de $G/*.json.
G="$tmp/gh"; mkdir -p "$G" "$tmp/bin"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
jq="" url=""
shift # api
while [ $# -gt 0 ]; do
  case "$1" in
    --jq) jq="$2"; shift 2 ;;
    --paginate | --silent) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$url" in
  *issues\?labels*) f=issues ;;
  */sub_issues*) f=sub ;;
  */comments*) f=comments ;;
  */pulls/*) echo '{"head":{"sha":"abc1234"}}' | jq -r "$jq"; exit 0 ;;
  */pulls\?*) f=pulls ;;
  */check-runs*) f=checks ;;
  */status*) echo '{"statuses":[]}' | jq -r "$jq"; exit 0 ;;
  *) echo "gh falso: $url" >&2; exit 1 ;;
esac
jq -r "$jq" "$G/$f.json"
SH
chmod +x "$tmp/bin/gh"
export PATH="$tmp/bin:$PATH" G

# origin com main e feat/12-x; clone de trabalho em main.
git init -q --bare "$tmp/origin.git"
git init -q -b main "$tmp/seed"
git_ -C "$tmp/seed" commit -q --allow-empty -m a
git_ -C "$tmp/seed" checkout -q -b feat/12-x
git_ -C "$tmp/seed" commit -q --allow-empty -m b
git_ -C "$tmp/seed" push -q "$tmp/origin.git" main feat/12-x
git clone -q "$tmp/origin.git" "$tmp/w"
cd "$tmp/w"
git checkout -q main

echo '[{"number":7}]' > "$G/issues.json"
# shellcheck disable=SC2016 # crases são Markdown
printf '%s\n' '[{"id":5,"body":"<!-- sdd-checkpoint -->\n## Checkpoint\n\n- **Branch:** `feat/12-x` em `abc1234`; alterações locais: 0; commits não enviados: 0\n- **Feito:** a\n- **Próximo:** b\n- **Bloqueia:** nada"}]' > "$G/comments.json"
echo '[{"number":12,"title":"feat: x"}]' > "$G/pulls.json"
echo '{"total_count":1,"check_runs":[{"id":1,"name":"make ci","status":"completed","conclusion":"success","app":{"slug":"github-actions"}}]}' > "$G/checks.json"
echo '[{"number":20,"title":"T1","state":"open"},{"number":21,"title":"T2","state":"closed"}]' > "$G/sub.json"

# 014: checkpoint, entra na branch, PRs com CI e sub-issues abertas.
out="$(sh "$s" --repo o/r)" || fail "sdd-resume falhou: $out"
printf '%s\n' "$out" | grep -q 'Checkpoint do épico #7' || fail "sem o checkpoint: $out"
printf '%s\n' "$out" | grep -q 'agora em feat/12-x' || fail "não entrou na branch: $out"
[ "$(git rev-parse --abbrev-ref HEAD)" = feat/12-x ] || fail "HEAD não mudou"
printf '%s\n' "$out" | grep -qx '  #12 feat: x: CI verde (1)' || fail "PR sem o CI: $out"
printf '%s\n' "$out" | grep -qx '  #20 T1' || fail "sem a sub-issue aberta: $out"
if printf '%s\n' "$out" | grep -q '#21'; then fail "listou sub-issue fechada: $out"; fi

# CI vermelho aparece no resumo do PR.
echo '{"total_count":1,"check_runs":[{"id":1,"name":"make ci","status":"completed","conclusion":"failure","app":{"slug":"x"}}]}' > "$G/checks.json"
out="$(sh "$s" --repo o/r)" || fail "sdd-resume com CI vermelho falhou"
printf '%s\n' "$out" | grep -q '#12 feat: x: CI vermelho: make ci' || fail "CI vermelho não apareceu: $out"

# Nunca descarta alterações locais: árvore suja não troca de branch.
git checkout -q main
echo x > sujo.txt
out="$(sh "$s" --repo o/r)" || fail "sdd-resume com árvore suja falhou"
printf '%s\n' "$out" | grep -q 'alterações locais; não troco' || fail "não avisou da árvore suja: $out"
[ "$(git rev-parse --abbrev-ref HEAD)" = main ] || fail "trocou de branch com a árvore suja"
rm sujo.txt

# Branch local com commits não enviados também não é sobrescrita.
git checkout -q -B feat/12-x origin/feat/12-x
git_ commit -q --allow-empty -m local
local_sha="$(git rev-parse HEAD)"
git checkout -q main
out="$(sh "$s" --repo o/r)" || fail "sdd-resume com commits locais falhou"
printf '%s\n' "$out" | grep -q 'commits não enviados; não troco' || fail "não avisou dos commits locais: $out"
[ "$(git rev-parse feat/12-x)" = "$local_sha" ] || fail "descartou commits locais"

# Branch inexistente no origin: avisa e fica.
sed -i.bak 's#feat/12-x#feat/99-y#' "$G/comments.json"
out="$(sh "$s" --repo o/r)" || fail "sdd-resume com branch ausente falhou"
printf '%s\n' "$out" | grep -q 'feat/99-y não existe no origin' || fail "não avisou da branch ausente: $out"

# Sem épico aberto e sem gh: avisa e sai com 0.
echo '[]' > "$G/issues.json"
out="$(sh "$s" --repo o/r)" || fail "sdd-resume sem épico falhou"
printf '%s\n' "$out" | grep -q 'nenhum épico aberto' || fail "sem aviso de épico: $out"
mkdir "$tmp/nogh"
for t in sh git jq sed grep dirname cat; do ln -s "$(command -v "$t")" "$tmp/nogh/$t"; done
out="$(PATH="$tmp/nogh" sh "$s")" || fail "sdd-resume sem gh falhou: $out"
printf '%s\n' "$out" | grep -q 'gh ausente' || fail "sem aviso de gh ausente: $out"
echo "tests/resume.sh ok"
