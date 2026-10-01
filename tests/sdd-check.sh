#!/bin/sh
# Testes do sdd-check (spec 006 AC-1 e 006 AC-2).
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
check="$root/template/common/scripts/sdd-check.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

d="$tmp/repo"
mkdir -p "$d/specs/001-x" "$d/tests"
cat > "$d/specs/README.md" <<'MD'
| ID | Spec | Prioridade | Status |
|---|---|---|---|
| 001 | [X](001-x/spec.md) | P0 | Done |
MD
cat > "$d/specs/001-x/spec.md" <<'MD'
# 001 — X

- **Status:** Done

- **FR-1** O sistema MUST x.
- **AC-1** Dado a, quando b, então c.
- **AC-2 (D1)** Dado d, quando e, então f.
MD
printf -- '- [x] **T1** x — 001 FR-1, 001 AC-1\n' > "$d/specs/ROADMAP.md"
printf '# 001 AC-1\n' > "$d/tests/x_test.sh"

# 006 AC-1: AC sem teste aparece; sem --strict sai com 0, com --strict sai com 1.
out="$(sh "$check" "$d")" || fail "sem --strict deveria sair com 0"
printf '%s\n' "$out" | grep -q '001 AC-2 sem teste' || fail "AC-2 sem teste não foi apontado"
printf '%s\n' "$out" | grep -q '| 001 | Done | AC-1 | 1 |' || fail "AC-1 citado não aparece na tabela"
if sh "$check" --strict "$d" >/dev/null; then fail "--strict deveria falhar com AC sem teste"; fi

# Com todos os AC citados, --strict passa.
printf '# 001/AC-2\n' >> "$d/tests/x_test.sh"
sh "$check" --strict "$d" >/dev/null || fail "--strict falhou com tudo citado"

# Status divergente e ID inexistente no ROADMAP são apontados.
sed 's/| Done |/| Approved |/' "$d/specs/README.md" > "$tmp/r" && mv "$tmp/r" "$d/specs/README.md"
printf -- '- [ ] **T2** y — 001 FR-9\n' >> "$d/specs/ROADMAP.md"
out="$(sh "$check" "$d")"
printf '%s\n' "$out" | grep -q "status 'Done' no cabeçalho e 'Approved'" || fail "status divergente não apontado"
printf '%s\n' "$out" | grep -q 'ROADMAP cita 001 FR-9' || fail "ID inexistente no ROADMAP não apontado"

# 006 FR-4: linha ADDED/MODIFIED de "Mudanças" com ID inexistente é apontada; REMOVED não.
printf '\n## Mudanças\n\n### Não lançado\n\n- ADDED FR-1 — x\n- MODIFIED AC-7 — y\n- REMOVED FR-8 — z\n' >> "$d/specs/001-x/spec.md"
out="$(sh "$check" "$d")"
printf '%s\n' "$out" | grep -q '"Mudanças" cita AC-7' || fail "MODIFIED com ID inexistente não apontado"
if printf '%s\n' "$out" | grep -q 'cita FR-1,\|cita FR-8'; then fail "ID válido ou REMOVED apontado por engano"; fi

# 006 AC-2: o próprio kit passa.
sh "$check" "$root" >/dev/null || fail "sdd-check falhou no próprio kit"
echo "tests/sdd-check.sh ok"
