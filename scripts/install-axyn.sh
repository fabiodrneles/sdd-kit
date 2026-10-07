#!/bin/sh
# Instala o axyn com um comando (spec 021 FR-1):
#   curl -fsSL https://raw.githubusercontent.com/fabiodrneles/sdd-kit/main/scripts/install-axyn.sh | sh
# Sem o Go (ou com AXYN_BINARY=1) baixa o binário da release e confere o sha256;
# com o Go, usa `go install`. Num repositório git, roda `axyn install` no fim.
# Variáveis: AXYN_VERSION (padrão latest), AXYN_BIN (destino do binário, padrão
# ~/.local/bin), GOBIN (destino do go install), AXYN_SRC (diretório local, para
# testes), AXYN_RELEASE_URL (base do download, para testes).
set -eu

die() {
	echo "install-axyn: $*" >&2
	exit 1
}

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

if command -v go >/dev/null 2>&1 && [ "${AXYN_BINARY:-}" != 1 ]; then
	if [ -n "${AXYN_SRC:-}" ]; then
		(cd "$AXYN_SRC" && go install .)
	else
		go install "github.com/fabiodrneles/sdd-kit/axyn@${AXYN_VERSION:-latest}"
	fi
	bin="${GOBIN:-$(go env GOBIN)}"
	[ -n "$bin" ] || bin="$(go env GOPATH)/bin"
else
	command -v curl >/dev/null 2>&1 || die "sem o Go (1.22 ou mais novo, https://go.dev/dl/) e sem o curl para baixar o binário"
	case "$(uname -s)" in
	Linux) os=linux ;;
	Darwin) os=darwin ;;
	*) die "sistema $(uname -s) sem binário; no Windows use scripts/install-axyn.ps1" ;;
	esac
	case "$(uname -m)" in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) die "arquitetura $(uname -m) sem binário; instale o Go e rode de novo" ;;
	esac
	name="axyn_${os}_${arch}"
	if [ -n "${AXYN_RELEASE_URL:-}" ]; then
		base="$AXYN_RELEASE_URL"
	elif [ -n "${AXYN_VERSION:-}" ]; then
		base="https://github.com/fabiodrneles/sdd-kit/releases/download/$AXYN_VERSION"
	else
		base="https://github.com/fabiodrneles/sdd-kit/releases/latest/download"
	fi
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	curl -fsSL "$base/$name" -o "$tmp/$name" || die "não consegui baixar $base/$name"
	curl -fsSL "$base/axyn_checksums.txt" -o "$tmp/sums" || die "não consegui baixar $base/axyn_checksums.txt"
	want="$(awk -v n="$name" '$2 == n || $2 == "*" n {print $1; exit}' "$tmp/sums")"
	[ -n "$want" ] || die "$name não está no axyn_checksums.txt; nada foi instalado"
	got="$(sha256_of "$tmp/$name")"
	[ "$got" = "$want" ] || die "sha256 de $name não confere (esperado $want, veio $got); nada foi instalado"
	bin="${AXYN_BIN:-$HOME/.local/bin}"
	mkdir -p "$bin"
	cp "$tmp/$name" "$bin/axyn.new"
	chmod 755 "$bin/axyn.new"
	mv "$bin/axyn.new" "$bin/axyn"
fi

echo "axyn instalado em $bin/axyn"
# Fora do PATH: a linha vai para o arquivo que o shell do usuário lê ao abrir, para o
# iniciante não precisar colar nada (AXYN_NO_MODIFY_PATH=1 pula).
case ":$PATH:" in
*":$bin:"*) ;;
*)
	line="export PATH=\"$bin:\$PATH\""
	case "$(basename "${SHELL:-sh}")" in
	zsh) rc="$HOME/.zshrc" ;;
	bash) if [ "$(uname -s)" = Darwin ]; then rc="$HOME/.bash_profile"; else rc="$HOME/.bashrc"; fi ;;
	*) rc="$HOME/.profile" ;;
	esac
	if [ -n "${AXYN_NO_MODIFY_PATH:-}" ]; then
		echo "adicione ao PATH: $line"
	else
		grep -qsF "$line" "$rc" || printf '\n# axyn\n%s\n' "$line" >> "$rc"
		echo "axyn adicionado ao PATH em $rc; neste terminal, rode: $line (ou abra um terminal novo)"
	fi
	;;
esac

if command -v git >/dev/null 2>&1 && git rev-parse --git-dir >/dev/null 2>&1; then
	"$bin/axyn" install
	# O que falta no repositório (GitHub, Actions, ferramentas), com o comando de cada item.
	"$bin/axyn" doctor || echo "para configurar o que dá automaticamente: axyn setup"
else
	echo "esta pasta não é um repositório git: entre na pasta do projeto (cd caminho/do/projeto) e rode lá: axyn install, depois axyn doctor"
fi
