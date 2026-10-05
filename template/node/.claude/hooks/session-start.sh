#!/bin/bash
# Prepara o ambiente do Claude Code na web para rodar `make ci` sem instalar
# nada no meio do trabalho. Só roda em sessões remotas.
set -euo pipefail

if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

cd "${CLAUDE_PROJECT_DIR:-$(pwd)}"

# Spec 014: o ponto de retomada, a branch, os PRs e as issues do épico aberto entram no contexto da sessão.
sh scripts/sdd-resume.sh 2>/dev/null || true

if [ -f package-lock.json ]; then
  npm ci --no-audit --no-fund
fi
