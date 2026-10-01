#!/bin/sh
# Gera dist/sdd-delivery.zip para upload no claude.ai (spec 002 FR-3).
# Uso: scripts/package-skill.sh [arquivo-de-saída]
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
out="${1:-$root/dist/sdd-delivery.zip}"
case "$out" in /*) ;; *) out="$(pwd)/$out" ;; esac

mkdir -p "$(dirname "$out")"
rm -f "$out"
cd "$root/plugins/sdd-delivery/skills"
zip -qrX "$out" sdd-delivery
echo "$out"
