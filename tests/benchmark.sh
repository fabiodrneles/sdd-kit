#!/bin/sh
# Testes do scripts/benchmark.sh (spec 020 FR-6) com um projeto falso, em --dry-run.
# shellcheck disable=SC2016 # as crases são do Markdown esperado
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
b="$root/scripts/benchmark.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

# Projeto falso com os arquivos do kit e o guarda de cobertura no Makefile.
p="$tmp/proj"; mkdir -p "$p/specs" "$p/scripts" "$p/.claude" "$p/.github/workflows"
cd "$p"
git init -q -b main
echo '# CLAUDE' > CLAUDE.md; echo '# AGENTS' > AGENTS.md; echo s > specs/x.md; echo '{}' > .claude/settings.json
echo x > scripts/sdd-pr.sh; echo y > scripts/build.sh; echo w > .github/workflows/sdd-on-merge.yml; echo c > .github/workflows/ci.yml
printf 'test:\n\tsh scripts/sdd-cover-guard.sh out/c.out -- go test ./...\n' > Makefile
git add -A; git -c user.email=t@t -c user.name=t commit -qm base
cat > "$tmp/tasks.md" <<'MD'
# Tarefas

## Primeira

Faça A.

Arquivos: a.go, a_test.go

## Segunda

Faça B.
MD

# 020 AC-6: as duas cópias, o kit fora do lado sem sdd-kit e um pedido por tarefa.
out="$(sh "$b" --repo "$p" --tasks "$tmp/tasks.md" --out "$tmp/out" --dry-run)" || fail "dry-run falhou: $out"
printf '%s\n' "$out" | grep -qF '2 tarefas' || fail "sem a contagem: $out"
printf '%s\n' "$out" | grep -qF 'removidos CLAUDE.md, AGENTS.md, specs/' || fail "sem os removidos: $out"
w="$tmp/out/without"
for f in CLAUDE.md AGENTS.md specs .claude scripts/sdd-pr.sh .github/workflows/sdd-on-merge.yml; do
  [ ! -e "$w/$f" ] || fail "o lado sem sdd-kit ainda tem $f"
done
if [ ! -f "$w/scripts/build.sh" ] || [ ! -f "$w/.github/workflows/ci.yml" ]; then fail "tirou arquivos do projeto"; fi
grep -q 'sdd-cover-guard' "$w/Makefile" && fail "Makefile sem sdd-kit ainda chama o guarda"
grep -q 'go test ./...' "$w/Makefile" || fail "Makefile sem o go test: $(cat "$w/Makefile")"
[ -f "$tmp/out/with/CLAUDE.md" ] || fail "o lado com sdd-kit perdeu o CLAUDE.md"
[ -z "$(git -C "$w" status --porcelain)" ] || fail "lado sem sdd-kit com mudança fora de commit"
grep -qF '## Tarefa 1: Primeira' "$tmp/out/prompt-without.md" || fail "pedido sem sdd-kit sem a tarefa 1"
grep -qF '## Tarefa 2: Segunda' "$tmp/out/prompt-without.md" || fail "pedido sem sdd-kit sem a tarefa 2"
grep -qF -- '- `a_test.go`' "$tmp/out/prompt-with-1.md" || fail "pacote da tarefa 1 sem os arquivos: $(cat "$tmp/out/prompt-with-1.md")"
grep -qF 'Segunda' "$tmp/out/prompt-with-1.md" && fail "pacote da tarefa 1 com a tarefa 2"
grep -qF 'Nunca afrouxe' "$tmp/out/prompt-with-2.md" || fail "pacote da tarefa 2 sem a regra dos testes"

# Uso inválido: 3.
rc=0; sh "$b" --bogus > /dev/null 2>&1 || rc=$?
[ "$rc" -eq 3 ] || fail "opção inválida saiu com $rc"
echo "benchmark: ok"
