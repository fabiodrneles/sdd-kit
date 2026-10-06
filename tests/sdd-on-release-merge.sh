#!/bin/sh
# Testes do sdd-on-release-merge.sh (issue #151, spec 015) com gh e sdd-release falsos.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }
export TMPDIR="$tmp"

# gh falso: $G/pr.N tem "merged ref repo"; $G/tags lista as tags existentes;
# $G/runs tem o JSON das execuções do release-tag.yml. Todo acesso é registrado em
# $G/calls.
G="$tmp/gh"; mkdir -p "$G" "$tmp/bin"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
shift # api
jq="" url=""
while [ $# -gt 0 ]; do
  case "$1" in
    --jq) jq="$2"; shift 2 ;;
    *) url="$1"; shift ;;
  esac
done
echo "$url" >> "$G/calls"
case "$url" in
  repos/o/r/pulls/*)
    n="${url##*/}"; read -r merged ref repo < "$G/pr.$n"
    jq -n --arg m "$merged" --arg r "$ref" --arg p "$repo" \
      '{merged: ($m == "true"), head: {ref: $r, repo: {full_name: $p}}, merged_at: "2026-01-02T00:00:00Z"}' | jq -r "$jq" ;;
  repos/o/r/git/ref/tags/*)
    grep -qx "${url##*/}" "$G/tags" ;;
  repos/o/r/actions/workflows/release-tag.yml/runs*)
    jq -r "$jq" < "$G/runs" ;;
  *) echo "gh falso: $url" >&2; exit 1 ;;
esac
SH
chmod +x "$tmp/bin/gh"
# sdd-release.sh falso: registra os argumentos de cada chamada.
cat > "$tmp/release.sh" <<'SH'
#!/bin/sh
echo "$* pre=${SDD_RELEASE_PRE:-}" >> "$G/dispatch"
SH
export PATH="$tmp/bin:$PATH" G SDD_RELEASE_SH="$tmp/release.sh"

echo "true chore/release-v1.2.3 o/r" > "$G/pr.10"
echo "true feat/9-algo o/r" > "$G/pr.11"
echo "false chore/release-v1.2.3 o/r" > "$G/pr.12"
echo "true chore/release-v1.2.3 fork/r" > "$G/pr.13"
: > "$G/tags"; echo '{"workflow_runs":[]}' > "$G/runs"; : > "$G/dispatch"; : > "$G/calls"
run() { sh "$s/sdd-on-release-merge.sh" --repo o/r "$@"; }
count() { wc -l < "$G/dispatch" | tr -d ' '; }

# 015 AC-5: o PR de fechamento mesclado dispara a release da versão, uma vez.
out="$(run 10)" || fail "PR de fechamento saiu com erro: $out"
[ "$(cat "$G/dispatch")" = "--repo o/r --tag 1.2.3 pre=ci" ] || fail "disparo errado (#184: com SDD_RELEASE_PRE=ci): $(cat "$G/dispatch")"

# 015 AC-5 (NFR-2): com a tag criada, rodar de novo não dispara outra vez.
echo "v1.2.3" > "$G/tags"
out="$(run 10)" || fail "rerun saiu com erro: $out"
[ "$(count)" -eq 1 ] || fail "segundo disparo com a tag existente"

# NFR-2: execução do Release tag em andamento (ainda sem tag) também impede.
: > "$G/tags"
echo '{"workflow_runs":[{"id":7,"status":"in_progress","conclusion":null,"created_at":"2026-01-02T00:05:00Z"}]}' > "$G/runs"
run 10 > /dev/null || fail "execução em andamento saiu com erro"
[ "$(count)" -eq 1 ] || fail "disparou com execução em andamento"
# Execução anterior ao merge não conta.
echo '{"workflow_runs":[{"id":6,"status":"completed","conclusion":"success","created_at":"2025-12-01T00:00:00Z"}]}' > "$G/runs"
run 10 > /dev/null || fail "execução antiga saiu com erro"
[ "$(count)" -eq 2 ] || fail "execução anterior ao merge não deveria impedir"

# Outros PRs (comum, não mesclado, de fork) não fazem nada (NFR-1).
echo '{"workflow_runs":[]}' > "$G/runs"
for n in 11 12 13; do run "$n" > /dev/null || fail "PR $n saiu com erro"; done
[ "$(count)" -eq 2 ] || fail "PR que não é de fechamento disparou a release"

# 015 AC-7: com SDD_ENGINE=off nada é lido nem disparado.
: > "$G/calls"
SDD_ENGINE=off run 10 > /dev/null || fail "SDD_ENGINE=off saiu com erro"
[ ! -s "$G/calls" ] || fail "SDD_ENGINE=off leu a API"
[ "$(count)" -eq 2 ] || fail "SDD_ENGINE=off disparou a release"

# Falha do sdd-release.sh vira código 1.
printf '#!/bin/sh\nexit 1\n' > "$tmp/release.sh"
rc=0; run 10 > /dev/null 2>&1 || rc=$?
[ "$rc" -eq 1 ] || fail "falha do sdd-release.sh deveria sair 1, saiu $rc"

echo "sdd-on-release-merge: ok"
