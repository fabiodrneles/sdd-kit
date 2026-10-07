#!/bin/sh
# Compila o axyn para as cinco plataformas da release (spec 021 FR-1):
#   scripts/build-axyn.sh DIST
# Grava DIST/axyn_<os>_<arch> (.exe no Windows) e DIST/axyn_checksums.txt (sha256).
# AXYN_VERSION, se definida, entra no binário (`axyn version`).
set -eu

[ $# -eq 1 ] || { echo "uso: $0 DIST" >&2; exit 2; }
root="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$1"
dist="$(cd "$1" && pwd)"

ldflags="-s -w"
[ -z "${AXYN_VERSION:-}" ] || ldflags="$ldflags -X main.version=$AXYN_VERSION"

for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
	os="${target%/*}"
	arch="${target#*/}"
	out="$dist/axyn_${os}_${arch}"
	[ "$os" != windows ] || out="$out.exe"
	(cd "$root/axyn" && CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "$ldflags" -o "$out" .)
done

cd "$dist"
if command -v sha256sum >/dev/null 2>&1; then
	sha256sum axyn_linux_* axyn_darwin_* axyn_windows_* > axyn_checksums.txt
else
	shasum -a 256 axyn_linux_* axyn_darwin_* axyn_windows_* > axyn_checksums.txt
fi
echo "axyn: 5 binários e axyn_checksums.txt em $dist"
