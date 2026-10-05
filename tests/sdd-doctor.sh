#!/bin/sh
# Testes do sdd-doctor.sh (issue #143) com ferramentas falsas num PATH temporário.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
doctor="$root/template/common/scripts/sdd-doctor.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }

# Projeto Go mínimo e ferramentas falsas: go (GOPATH e GOTOOLDIR sob $tmp),
# npx, e dois golangci-lint (o antigo em $tmp/old, o certo em GOPATH/bin).
p="$tmp/proj"; mkdir -p "$p" "$tmp/bin" "$tmp/old" "$tmp/gopath/bin" "$tmp/tool" "$tmp/home"
printf 'module x\n\ngo 1.26\n' > "$p/go.mod"
printf 'GOLANGCI_LINT_VERSION := v2.14.0\n.PHONY: lint\nlint:\n\tgolangci-lint run\n' > "$p/Makefile"
cat > "$tmp/bin/go" <<'SH'
#!/bin/sh
case "$1 $2" in
  "env GOPATH") echo "$FAKE/gopath" ;;
  "env GOTOOLDIR") echo "$FAKE/tool" ;;
  "build -o") : > "$3"; chmod +x "$3" ;;
  *) exit 1 ;;
esac
SH
printf '#!/bin/sh\nexit 0\n' > "$tmp/bin/npx"
printf '#!/bin/sh\necho "golangci-lint has version 2.14.0 built with go1.26 from x"\n' > "$tmp/gopath/bin/golangci-lint"
printf '#!/bin/sh\necho "golangci-lint has version 2.9.0 built with go1.25 from x"\n' > "$tmp/old/golangci-lint"
chmod +x "$tmp/bin/go" "$tmp/bin/npx" "$tmp/gopath/bin/golangci-lint" "$tmp/old/golangci-lint"
: > "$tmp/tool/covdata"; chmod +x "$tmp/tool/covdata"

# Roda o doctor no projeto; guarda a saída em $out e o código em $rc.
run() { # args do doctor; PATH_EXTRA e LOC no ambiente
  rc=0
  out="$(cd "$p" && env FAKE="$tmp" HOME="$tmp/home" LC_ALL="${LOC:-C.UTF-8}" \
    PATH="${PATH_EXTRA:+$PATH_EXTRA:}$tmp/gopath/bin:$tmp/bin:$PATH" sh "$doctor" "$@" 2>&1)" || rc=$?
}

# Tudo certo: sai com 0 e só imprime ok.
run --check
[ "$rc" -eq 0 ] || fail "ambiente certo saiu com $rc: $out"
printf '%s\n' "$out" | grep -qx 'ok covdata' || fail "sem 'ok covdata': $out"
printf '%s\n' "$out" | grep -qx 'ok golangci-lint 2.14.0' || fail "sem 'ok golangci-lint 2.14.0': $out"
printf '%s\n' "$out" | grep -q '^FALHA' && fail "ambiente certo com FALHA: $out"

# golangci-lint antigo na frente no PATH: aponta e imprime a linha de export.
PATH_EXTRA="$tmp/old" run --check
[ "$rc" -eq 1 ] || fail "golangci-lint ofuscado saiu com $rc (quero 1): $out"
printf '%s\n' "$out" | grep -qF "FALHA golangci-lint em $tmp/old/golangci-lint esconde o de $tmp/gopath/bin: export PATH=\"$tmp/gopath/bin:\$PATH\"" ||
  fail "não imprimiu a linha de PATH: $out"
# Com CLAUDE_ENV_FILE (e sem --check) a linha é gravada, uma vez só.
ef="$tmp/env"; : > "$ef"
rc=0; out="$(cd "$p" && env FAKE="$tmp" HOME="$tmp/home" LC_ALL=C.UTF-8 CLAUDE_ENV_FILE="$ef" PATH="$tmp/old:$tmp/gopath/bin:$tmp/bin:$PATH" sh "$doctor" 2>&1)" || rc=$?
printf '%s\n' "$out" | grep -q '^corrigido golangci-lint em' || fail "não corrigiu com CLAUDE_ENV_FILE: $out"
(cd "$p" && env FAKE="$tmp" HOME="$tmp/home" LC_ALL=C.UTF-8 CLAUDE_ENV_FILE="$ef" PATH="$tmp/old:$tmp/gopath/bin:$tmp/bin:$PATH" sh "$doctor" >/dev/null 2>&1) || true
[ "$(grep -c . "$ef")" -eq 1 ] || fail "CLAUDE_ENV_FILE com linhas repetidas: $(cat "$ef")"
grep -qxF "export PATH=\"$tmp/gopath/bin:\$PATH\"" "$ef" || fail "CLAUDE_ENV_FILE sem a linha de PATH: $(cat "$ef")"
# Se o Makefile já prefere o GOPATH/bin, o ofuscamento é inofensivo.
# shellcheck disable=SC2016 # o $(shell ...) é literal, do Makefile
sed 's|golangci-lint run|$(shell go env GOPATH)/bin/golangci-lint run|' "$p/Makefile" > "$tmp/mk" && cp "$tmp/mk" "$p/Makefile"
PATH_EXTRA="$tmp/old" run --check
[ "$rc" -eq 0 ] || fail "Makefile que prefere o GOPATH/bin saiu com $rc: $out"
printf 'GOLANGCI_LINT_VERSION := v2.14.0\nlint:\n\tgolangci-lint run\n' > "$p/Makefile"

# covdata ausente: --check relata e não instala; sem --check compila.
rm "$tmp/tool/covdata"
run --check
[ "$rc" -eq 1 ] || fail "covdata ausente saiu com $rc (quero 1): $out"
printf '%s\n' "$out" | grep -q '^FALHA covdata: ' || fail "não reportou o covdata: $out"
[ ! -e "$tmp/tool/covdata" ] || fail "--check instalou o covdata"
run
[ "$rc" -eq 0 ] || fail "doctor sem --check saiu com $rc: $out"
printf '%s\n' "$out" | grep -q '^corrigido covdata' || fail "não corrigiu o covdata: $out"
[ -x "$tmp/tool/covdata" ] || fail "o covdata não foi compilado"

# golangci-lint ausente em GOPATH/bin: --check relata a versão do Makefile.
mv "$tmp/gopath/bin/golangci-lint" "$tmp/gl"
run --check
[ "$rc" -eq 1 ] || fail "golangci-lint ausente saiu com $rc: $out"
printf '%s\n' "$out" | grep -q '^FALHA golangci-lint: falta a v2.14.0' || fail "não reportou o golangci-lint: $out"
mv "$tmp/gl" "$tmp/gopath/bin/golangci-lint"

# Locale que não é UTF-8: relata a linha de ambiente.
LOC=C run --check
[ "$rc" -eq 1 ] || fail "locale C saiu com $rc (quero 1): $out"
printf '%s\n' "$out" | grep -q '^FALHA locale (C não é UTF-8.*: export LC_ALL=' || fail "não reportou o locale: $out"

# Opção inválida.
rc=0; (cd "$p" && sh "$doctor" --nao-existe >/dev/null 2>&1) || rc=$?
[ "$rc" -eq 3 ] || fail "opção inválida saiu com $rc (quero 3)"

echo "tests/sdd-doctor.sh ok"
