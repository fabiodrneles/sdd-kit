#!/bin/sh
# Instalação do axyn por um comando (spec 021 FR-1): o script instala o binário e `axyn version` roda.
set -eu
cd "$(dirname "$0")/.."

bin="$(mktemp -d)"
trap 'rm -rf "$bin"' EXIT

out="$(GOBIN="$bin" AXYN_SRC="$PWD/axyn" sh scripts/install-axyn.sh)"
[ -x "$bin/axyn" ] || { echo "FAIL: $bin/axyn não foi criado"; exit 1; }
"$bin/axyn" version | grep -q . || { echo "FAIL: axyn version sem saída"; exit 1; }
echo "$out" | grep -q "axyn instalado em $bin/axyn" || { echo "FAIL: saída sem o destino: $out"; exit 1; }

# sem Go no PATH, o script para com o motivo
if PATH=/nonexistent /bin/sh scripts/install-axyn.sh 2>"$bin/err"; then
	echo "FAIL: deveria falhar sem o Go"
	exit 1
fi
grep -q "Go" "$bin/err" || { echo "FAIL: sem o motivo"; exit 1; }
echo "ok: install-axyn"
