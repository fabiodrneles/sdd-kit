#!/bin/sh
# Testes dos scripts mecânicos (spec 009).
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
s="$root/template/common/scripts"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

d="$tmp/repo"
mkdir -p "$d/specs/001-x" "$d/scripts"
cp "$s/sdd-check.sh" "$d/scripts/"
cat > "$d/specs/README.md" <<'MD'
| ID | Spec | Prioridade | Status |
|---|---|---|---|
| 001 | [X](001-x/spec.md) | P0 | Draft |
MD
cat > "$d/specs/001-x/spec.md" <<'MD'
# 001 — X

- **Status:** Draft — aguarda as decisões D1 e D2

- **AC-1** Dado a, quando b, então c.
MD
cat > "$d/specs/ANALYSIS.md" <<'MD'
## 7. Decisões em aberto

| ID | Pergunta | Opções | Recomendação |
|---|---|---|---|
| D1 | x? | (a) sim; (b) não | **(a)** |
| D2 | y? | (a) sim; (b) não | **(b)** |
MD
cat > "$d/specs/ROADMAP.md" <<'MD'
# Roadmap

## Fase 0 — Decisões

- [ ] Responder as decisões.

## Fase 1 — Funcionar (P0) → `v0.1.0`

- [ ] **T1** Fazer x — 001 AC-1
MD
printf '# Changelog\n\n## [Unreleased]\n\n### Adicionado\n\n- x.\n' > "$d/CHANGELOG.md"
printf '# 001 AC-1\n' > "$d/x_test.sh"
cd "$d"

# 009 AC-1: decisão parcial mantém Draft; a última aprova as specs e marca a Fase 0.
sh "$s/sdd-mark.sh" decide --date 2026-01-02 D1=a > /dev/null
grep -q '| D1 .*respondida: (a)' specs/ANALYSIS.md || fail "D1 não marcada"
grep -q 'Status:\*\* Draft' specs/001-x/spec.md || fail "spec saiu de Draft com decisão em aberto"
sh "$s/sdd-mark.sh" decide --date 2026-01-02 D2=b > /dev/null
grep -q 'Status:\*\* Approved' specs/001-x/spec.md || fail "spec não foi para Approved"
grep -q '| 001 .*| Approved |' specs/README.md || fail "índice não foi para Approved"
grep -q '^- \[x\] Responder' specs/ROADMAP.md || fail "Fase 0 não marcada"

# 009 AC-2: decisão inexistente falha sem mudar nada.
before="$(cat specs/ANALYSIS.md specs/README.md specs/ROADMAP.md | cksum)"
if sh "$s/sdd-mark.sh" decide D9=a 2>/dev/null; then fail "aceitou D9"; fi
[ "$(cat specs/ANALYSIS.md specs/README.md specs/ROADMAP.md | cksum)" = "$before" ] || fail "D9 alterou arquivos"

# 009 AC-4: épico em dry-run, sem GitHub.
out="$(GH_TOKEN=invalido sh "$s/sdd-epic.sh" --repo exemplo/repo --dry-run 1 2>&1)" || fail "sdd-epic --dry-run falhou: $out"
printf '%s\n' "$out" | grep -q 'T1' || fail "sdd-epic --dry-run sem T1: $out"

# 009 AC-3: fechamento da fase; idempotente.
sh "$s/sdd-mark.sh" close --date 2026-01-03 v0.1.0 > /dev/null
grep -q '^- \[x\] \*\*T1\*\*' specs/ROADMAP.md || fail "T1 não marcada"
# shellcheck disable=SC2016 # crases literais do Markdown
grep -q 'Status:\*\* Done — entregue na `v0.1.0`' specs/001-x/spec.md || fail "001 não foi para Done"
grep -q '^## \[0.1.0\] - 2026-01-03' CHANGELOG.md || fail "CHANGELOG sem [0.1.0]"
snap="$(cat specs/*.md specs/001-x/spec.md CHANGELOG.md | cksum)"
sh "$s/sdd-mark.sh" close --date 2026-01-03 v0.1.0 > /dev/null
[ "$(cat specs/*.md specs/001-x/spec.md CHANGELOG.md | cksum)" = "$snap" ] || fail "close não é idempotente"
for f in specs/README.md specs/ROADMAP.md specs/ANALYSIS.md CHANGELOG.md; do [ -s "$f" ] || fail "$f ficou vazio"; done

# 009 AC-5: uso inválido sai com o código de uso, sem GitHub.
rc=0; sh "$s/sdd-ci.sh" --nao-existe >/dev/null 2>&1 || rc=$?
[ "$rc" -eq 3 ] || fail "sdd-ci com opção inválida saiu com $rc (quero 3)"
rc=0; sh "$s/sdd-phase-status.sh" --nao-existe >/dev/null 2>&1 || rc=$?
[ "$rc" -ne 0 ] || fail "sdd-phase-status aceitou opção inválida"
rc=0; sh "$root/template/go/scripts/sdd-release-check.sh" >/dev/null 2>&1 || rc=$?
[ "$rc" -eq 2 ] || fail "sdd-release-check sem argumentos saiu com $rc (quero 2)"


# 009 AC-7: status de commit (ex.: Vercel) contam como checks, com um gh falso.
mkdir -p "$tmp/bin" "$tmp/gh"
cat > "$tmp/bin/gh" <<'SH'
#!/bin/sh
# gh falso: "gh api URL --jq EXPR" aplica EXPR ao JSON de $FAKE_GH/<recurso>.json.
url="$2"; expr="$4"
case "$url" in
  */check-runs*) f=check-runs ;;
  */status*) f=status ;;
  */commits/*) echo "${url##*/}"; exit 0 ;;
  *) echo "gh falso: $url" >&2; exit 1 ;;
esac
jq -r "$expr" "$FAKE_GH/$f.json"
SH
chmod +x "$tmp/bin/gh"
echo '{"total_count":1,"check_runs":[{"id":1,"name":"make ci","status":"completed","conclusion":"success","app":{"slug":"github-actions"}}]}' > "$tmp/gh/check-runs.json"
echo '{"statuses":[{"context":"Vercel","state":"failure","target_url":"https://vercel.example/dpl"}]}' > "$tmp/gh/status.json"
rc=0; out="$(PATH="$tmp/bin:$PATH" FAKE_GH="$tmp/gh" sh "$s/sdd-ci.sh" --repo o/r --no-wait 0123456)" || rc=$?
[ "$rc" -eq 1 ] || fail "sdd-ci com status vermelho saiu com $rc (quero 1): $out"
printf '%s\n' "$out" | grep -qx 'FALHA Vercel (status)' || fail "sdd-ci não listou o status: $out"
printf '%s\n' "$out" | grep -q 'https://vercel.example/dpl' || fail "sdd-ci não mostrou o link do status: $out"
echo '{"statuses":[{"context":"Vercel","state":"success","target_url":""}]}' > "$tmp/gh/status.json"
out="$(PATH="$tmp/bin:$PATH" FAKE_GH="$tmp/gh" sh "$s/sdd-ci.sh" --repo o/r --no-wait 0123456)" || fail "sdd-ci com tudo verde falhou: $out"
printf '%s\n' "$out" | grep -qx 'ok Vercel (status)' || fail "sdd-ci não listou o status verde: $out"
echo "tests/scripts.sh ok"
