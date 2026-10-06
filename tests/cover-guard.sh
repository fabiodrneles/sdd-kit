#!/bin/sh
# Testes do template/go/scripts/sdd-cover-guard.sh (010 FR-1, #172): perfil coerente
# passa sem rodar de novo; perfil com blocos de versões diferentes (ou além do fim do
# arquivo) é refeito com um cache frio; incoerente também a frio falha.
set -eu
root="$(cd "$(dirname "$0")/.." && pwd)"
g="$root/template/go/scripts/sdd-cover-guard.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

mkdir -p "$tmp/bin" "$tmp/mod"
printf '#!/bin/sh\necho example.com/m\n' > "$tmp/bin/go"
chmod +x "$tmp/bin/go"
PATH="$tmp/bin:$PATH"
export PATH
printf 'package main\n\nfunc main() {}\n\nfunc a() {}\n\nfunc b() {}\n' > "$tmp/mod/main.go"
good='mode: set
example.com/m/main.go:3.13,3.15 0 1
example.com/m/main.go:5.10,5.12 0 1
example.com/m/main.go:3.13,3.15 0 0'
overlap="$good
example.com/m/main.go:2.13,4.2 1 0"
pasteof="$good
example.com/m/main.go:90.1,91.2 1 0"
printf '%s\n' "$good" > "$tmp/good"
printf '%s\n' "$overlap" > "$tmp/overlap"
printf '%s\n' "$pasteof" > "$tmp/pasteof"

# Comando falso: com GOCACHE=hot grava $HOT; com outro GOCACHE (frio) grava $COLD.
cat > "$tmp/cmd.sh" <<'SH'
#!/bin/sh
echo x >> "$CALLS"
if [ "${GOCACHE:-}" = hot ]; then cp "$HOT" coverage.out; else cp "$COLD" coverage.out; fi
exit "${RC:-0}"
SH
run() { (cd "$tmp/mod" && : > "$tmp/calls" && CALLS="$tmp/calls" GOCACHE=hot HOT="$tmp/$1" COLD="$tmp/$2" sh "$g" coverage.out -- sh "$tmp/cmd.sh" 2>&1); }
calls() { wc -l < "$tmp/calls" | tr -d ' '; }

out="$(run good good)" || fail "perfil coerente falhou: $out"
[ "$(calls)" = 1 ] || fail "perfil coerente rodou de novo"

out="$(run overlap good)" || fail "não recuperou com o cache frio: $out"
printf '%s\n' "$out" | grep -q 'versões diferentes de example.com/m/main.go' || fail "não avisou: $out"
printf '%s\n' "$out" | grep -q 'refeito com o cache frio' || fail "não disse que refez: $out"
[ "$(calls)" = 2 ] || fail "não rodou de novo a frio"

out="$(run pasteof good)" || fail "bloco além do fim do arquivo não recuperou: $out"
[ "$(calls)" = 2 ] || fail "bloco além do fim do arquivo não foi percebido"

if out="$(run overlap overlap)"; then fail "incoerente também a frio devia falhar: $out"; fi
printf '%s\n' "$out" | grep -q 'continua incoerente' || fail "sem a mensagem de falha: $out"

if out="$(cd "$tmp/mod" && : > "$tmp/calls" && CALLS="$tmp/calls" GOCACHE=hot HOT="$tmp/good" COLD="$tmp/good" RC=3 sh "$g" coverage.out -- sh "$tmp/cmd.sh" 2>&1)"; then fail "não repassou a falha do comando"; fi
[ "$(calls)" = 1 ] || fail "rodou de novo depois de o comando falhar"

out="$(cd "$tmp/mod" && : > "$tmp/calls" && CALLS="$tmp/calls" GOCACHE=hot HOT="$tmp/overlap" COLD="$tmp/good" SDD_COVER_GUARD=off sh "$g" coverage.out -- sh "$tmp/cmd.sh" 2>&1)" || fail "SDD_COVER_GUARD=off falhou: $out"
[ "$(calls)" = 1 ] || fail "SDD_COVER_GUARD=off ainda conferiu"

if sh "$g" coverage.out 2> /dev/null; then fail "uso inválido devia falhar"; fi
echo "cover-guard: ok"
