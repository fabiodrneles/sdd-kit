#!/bin/sh
# Instala o axyn com um comando (spec 021 FR-1):
#   curl -fsSL https://raw.githubusercontent.com/fabiodrneles/sdd-kit/main/scripts/install-axyn.sh | sh
# Variáveis: AXYN_VERSION (padrão latest), GOBIN (destino), AXYN_SRC (diretório local, para testes).
set -eu

command -v go >/dev/null 2>&1 || {
	echo "install-axyn: o Go (1.22 ou mais novo) não está instalado: https://go.dev/dl/" >&2
	exit 1
}

if [ -n "${AXYN_SRC:-}" ]; then
	(cd "$AXYN_SRC" && go install .)
else
	go install "github.com/fabiodrneles/sdd-kit/axyn@${AXYN_VERSION:-latest}"
fi

bin="${GOBIN:-$(go env GOBIN)}"
[ -n "$bin" ] || bin="$(go env GOPATH)/bin"
echo "axyn instalado em $bin/axyn"
case ":$PATH:" in
*":$bin:"*) ;;
*) echo "adicione ao PATH: export PATH=\"$bin:\$PATH\"" ;;
esac
echo "no repositório do projeto: axyn install"
