#!/bin/sh
# Gera, a partir do template, os workflows do motor orientado a eventos que o
# próprio sdd-kit roda (spec 015 FR-8). Uso: sh scripts/self-sync.sh [--check]
#   (sem argumento) escreve .github/workflows/<workflow>.yml
#   --check         não escreve; falha listando os arquivos que divergem
# Os workflows gerados chamam os scripts em template/common/scripts/ (nada é
# duplicado). Não edite os gerados: mude o template e rode `make self-sync`.
#
# Adaptações do kit, todas aqui:
#   ENGINE        workflows do template/common/.github/workflows que o kit roda
#   SCRIPTS_FROM  prefixo dos scripts nos projetos adotantes
#   SCRIPTS_TO    onde os scripts estão no kit
#   PROJECT/OWNER/REPO  valores dos marcadores {{...}} do template
# O workflow de CI do kit se chama "CI" (ci.yml), como o sdd-ci-summary espera;
# release-tag.yml e o .sdd-release do kit são próprios e não são gerados.
set -eu

ENGINE="sdd-ci-summary sdd-update-prs sdd-on-merge sdd-on-phase-done sdd-on-release-merge sdd-auto-merge"
SCRIPTS_FROM="sh scripts/"
SCRIPTS_TO="sh template/common/scripts/"
PROJECT="sdd-kit"
OWNER="fabiodrneles"
REPO="sdd-kit"

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
check=0
[ "${1:-}" != "--check" ] || check=1

# Escapa o texto para o uso num s### do sed.
esc() { printf '%s' "$1" | sed 's/[\\&#.[*^$]/\\&/g'; }

from="$(esc "$SCRIPTS_FROM")"
to="$(esc "$SCRIPTS_TO")"
bad=""
for w in $ENGINE; do
  src="template/common/.github/workflows/$w.yml"
  dst=".github/workflows/$w.yml"
  [ -f "$src" ] || { echo "self-sync: $src não existe" >&2; exit 1; }
  out="$(mktemp)"
  {
    echo "# Gerado por make self-sync a partir de $src; não edite."
    sed -e "s#$from#$to#g" \
      -e "s/{{PROJECT}}/$PROJECT/g" -e "s/{{OWNER}}/$OWNER/g" -e "s/{{REPO}}/$REPO/g" "$src"
  } > "$out"
  if [ "$check" = 1 ]; then
    cmp -s "$out" "$dst" || bad="$bad $dst"
    rm -f "$out"
  else
    mkdir -p .github/workflows
    mv "$out" "$dst"
    chmod 644 "$dst"
    echo "self-sync: $dst"
  fi
done
if [ -n "$bad" ]; then
  echo "self-sync: divergem do template:$bad" >&2
  echo "self-sync: rode 'make self-sync' e commite o resultado" >&2
  exit 1
fi
