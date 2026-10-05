#!/bin/bash
# Prepara o ambiente do Claude Code na web para rodar `make ci` sem instalar
# nada no meio do trabalho. Só roda em sessões remotas.
set -euo pipefail

if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

cd "${CLAUDE_PROJECT_DIR:-$(pwd)}"

# Spec 014: o ponto de retomada do épico aberto entra no contexto da sessão.
sh template/common/scripts/sdd-resume.sh 2>/dev/null || true

# Versão do shellcheck igual à do CI e do Makefile.
want="$(sed -n 's/^SHELLCHECK_VERSION *:\?= *//p' Makefile)"
bin="$HOME/.local/bin"
mkdir -p "$bin"
if ! "$bin/shellcheck" --version 2>/dev/null | grep -q "version: ${want#v}$"; then
  curl -fsSL "https://github.com/koalaman/shellcheck/releases/download/${want}/shellcheck-${want}.linux.x86_64.tar.xz" |
    tar -xJ -C /tmp
  cp "/tmp/shellcheck-${want}/shellcheck" "$bin/"
fi
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
  echo "export PATH=\"$bin:\$PATH\"" >> "$CLAUDE_ENV_FILE"
fi

# actionlint na mesma versão do CI.
want="$(sed -n 's/^ACTIONLINT_VERSION *:\?= *//p' Makefile)"
if ! "$bin/actionlint" --version 2>/dev/null | grep -qx "${want#v}"; then
  curl -fsSL "https://github.com/rhysd/actionlint/releases/download/${want}/actionlint_${want#v}_linux_amd64.tar.gz" |
    tar -xz -C "$bin" actionlint
fi

# Cache do markdownlint-cli2 usado pelo `make ci`.
npx --yes "markdownlint-cli2@$(sed -n 's/^MARKDOWNLINT_VERSION *:\?= *//p' Makefile)" --help >/dev/null 2>&1 || true
