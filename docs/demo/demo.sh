#!/bin/sh
# Gera a demonstração do README (spec 013 FR-2) a partir da saída real: adota o
# kit num repositório Go vazio, roda o make ci gerado e grava a transcrição em
# docs/demo/transcript.txt; depois desenha docs/demo/demo.gif com o render.cjs.
#
# Uso: sh docs/demo/demo.sh [--transcript-only]
# Precisa de: go e golangci-lint (para o make ci) e, para o GIF, node com o
# pacote playwright (NODE_PATH) e ffmpeg.
set -eu

root="$(cd "$(dirname "$0")/../.." && pwd)"
out="$root/docs/demo"
version="v$(jq -r .version "$root/plugins/sdd-delivery/.claude-plugin/plugin.json")"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
p="$tmp/meu-projeto"
mkdir "$p"
cd "$p"
git init -q

{
  echo "\$ curl -fsSL https://raw.githubusercontent.com/fabiodrneles/sdd-kit/$version/scripts/adopt.sh | sh -s -- --lang go --skeleton ."
  sh "$root/scripts/adopt.sh" --lang go --project meu-projeto --owner voce --repo meu-projeto --skeleton . > "$tmp/adopt.log"
  grep -E '^criado: (CLAUDE.md|AGENTS.md|specs/ROADMAP.md|\.github/workflows/ci.yml|Makefile|hello_test.go)$' "$tmp/adopt.log"
  echo "…"
  grep '^sdd-kit go em ' "$tmp/adopt.log" | sed "s#$p#.#"
  echo "\$ make ci"
  make ci 2>&1 | grep -v '^$'
  echo "\$ claude"
  echo "> use a skill sdd-delivery: crie as specs e o ROADMAP para a minha ideia"
} > "$out/transcript.txt"
echo "demo: transcrição em docs/demo/transcript.txt ($(wc -l < "$out/transcript.txt") linhas)"

[ "${1:-}" != --transcript-only ] || exit 0
node "$out/render.cjs" "$out/transcript.txt" "$out/demo.gif"
echo "demo: docs/demo/demo.gif ($(wc -c < "$out/demo.gif") bytes)"
