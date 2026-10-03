#!/bin/bash
# Prepara o ambiente do Claude Code na web para rodar `make ci` sem instalar
# nada no meio do trabalho. Só roda em sessões remotas.
set -euo pipefail

if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

cd "${CLAUDE_PROJECT_DIR:-$(pwd)}"

# Spec 014: o ponto de retomada do épico aberto entra no contexto da sessão.
sh scripts/sdd-checkpoint.sh show 2>/dev/null || true

if [ -f pom.xml ]; then
  mvn="mvn"
  [ -x ./mvnw ] && mvn="./mvnw"
  "$mvn" -B -q -ntp dependency:go-offline
fi
