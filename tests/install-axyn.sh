#!/bin/sh
# Instalação do axyn por um comando (spec 021 FR-1, AC-5): com o Go (go install) e sem o Go
# (binário da release, com sha256), sem rede.
set -eu
cd "$(dirname "$0")/.."
root="$PWD"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FAIL: $*" >&2; exit 1; }

bin="$tmp/gobin"
out="$(GOBIN="$bin" AXYN_SRC="$root/axyn" sh scripts/install-axyn.sh)"
[ -x "$bin/axyn" ] || fail "$bin/axyn não foi criado"
"$bin/axyn" version | grep -q . || fail "axyn version sem saída"
echo "$out" | grep -q "axyn instalado em $bin/axyn" || fail "saída sem o destino: $out"

# sem Go (nem curl) no PATH, o script para com o motivo
if PATH=/nonexistent /bin/sh scripts/install-axyn.sh 2>"$tmp/err"; then
	fail "deveria falhar sem o Go"
fi
grep -q "Go" "$tmp/err" || fail "sem o motivo"

# build-axyn.sh: os cinco binários e o arquivo de checksums
dist="$tmp/dist"
AXYN_VERSION=v9.9.9 sh scripts/build-axyn.sh "$dist" > /dev/null
for f in axyn_linux_amd64 axyn_linux_arm64 axyn_darwin_amd64 axyn_darwin_arm64 axyn_windows_amd64.exe; do
	[ -s "$dist/$f" ] || fail "build-axyn.sh não gerou $f"
	grep -q " \*\{0,1\}$f\$" "$dist/axyn_checksums.txt" || fail "$f fora do axyn_checksums.txt"
done
[ "$(wc -l < "$dist/axyn_checksums.txt")" -eq 5 ] || fail "axyn_checksums.txt deveria ter 5 linhas"

# sem o Go no PATH: um PATH só com as ferramentas do script
nogo="$tmp/nogo"
mkdir "$nogo"
for t in sh uname curl sha256sum shasum awk mkdir mktemp cp chmod mv rm git dirname basename cat sed grep head tr; do
	p="$(command -v "$t" 2>/dev/null || true)"
	case "$p" in /*) ln -s "$p" "$nogo/$t" ;; esac
done
PATH="$nogo" command -v go > /dev/null 2>&1 && fail "o PATH de teste ainda tem o Go"

repo="$tmp/repo"
mkdir "$repo"
git -C "$repo" init -q
home="$tmp/home"
mkdir "$home"
out="$(cd "$repo" && PATH="$nogo" HOME="$home" AXYN_BIN="$home/bin" AXYN_RELEASE_URL="file://$dist" /bin/sh "$root/scripts/install-axyn.sh")" ||
	fail "instalação sem Go falhou: $out"
[ -x "$home/bin/axyn" ] || fail "binário não foi instalado em $home/bin"
"$home/bin/axyn" version | grep -q "v9.9.9" || fail "axyn version não traz a versão do build"
echo "$out" | grep -q "axyn instalado em $home/bin/axyn" || fail "saída sem o destino: $out"
[ -f "$repo/opencode.json" ] || fail "axyn install não rodou no repositório git"

# fora do PATH: a linha vai para o arquivo do shell do usuário, uma vez só
for i in 1 2; do
	(cd "$repo" && PATH="$nogo" HOME="$home" SHELL=/bin/bash AXYN_BIN="$home/bin" AXYN_RELEASE_URL="file://$dist" /bin/sh "$root/scripts/install-axyn.sh") > "$tmp/rc.out" ||
		fail "instalação $i com SHELL=bash falhou: $(cat "$tmp/rc.out")"
done
[ "$(grep -c "export PATH=\"$home/bin:" "$home/.bashrc")" = 1 ] || fail "o .bashrc deveria ter a linha do PATH uma vez: $(cat "$home/.bashrc" 2>/dev/null)"
grep -q "adicionado ao PATH em $home/.bashrc" "$tmp/rc.out" || fail "sem o aviso do PATH: $(cat "$tmp/rc.out")"
(cd "$repo" && PATH="$nogo" HOME="$home" SHELL=/bin/zsh AXYN_NO_MODIFY_PATH=1 AXYN_BIN="$home/bin" AXYN_RELEASE_URL="file://$dist" /bin/sh "$root/scripts/install-axyn.sh") > "$tmp/rc.out"
[ ! -e "$home/.zshrc" ] || fail "AXYN_NO_MODIFY_PATH=1 deveria não mexer no .zshrc"

# fora de um repositório git, só orienta
plain="$tmp/plain"
mkdir "$plain"
out="$(cd "$plain" && PATH="$nogo" HOME="$home" AXYN_BIN="$home/bin2" AXYN_RELEASE_URL="file://$dist" /bin/sh "$root/scripts/install-axyn.sh")"
[ ! -e "$plain/opencode.json" ] || fail "criou opencode.json fora de um repositório git"
echo "$out" | grep -q "não é um repositório git.*axyn install" || fail "sem a orientação de entrar na pasta do projeto: $out"

# checksum errado: para com o motivo e não instala nada
bad="$tmp/bad"
cp -r "$dist" "$bad"
sed 's/^[0-9a-f]\{8\}/00000000/' "$dist/axyn_checksums.txt" > "$bad/axyn_checksums.txt"
if (cd "$repo" && PATH="$nogo" HOME="$home" AXYN_BIN="$home/bin3" AXYN_RELEASE_URL="file://$bad" /bin/sh "$root/scripts/install-axyn.sh") 2>"$tmp/err"; then
	fail "checksum errado deveria falhar"
fi
grep -q "sha256" "$tmp/err" || fail "sem o motivo do checksum: $(cat "$tmp/err")"
[ ! -e "$home/bin3/axyn" ] || fail "instalou com checksum errado"

echo "ok: install-axyn"
