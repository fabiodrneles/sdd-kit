# AGENTS.md

Instruções para agentes de código (Codex, Copilot, Cursor, Claude Code e outros) que trabalham no sdd-kit. O guia completo está no [CLAUDE.md](CLAUDE.md).

- O kit é desenvolvido com o próprio processo: specs em [`specs/`](specs/README.md), um ticket por tarefa, uma branch e um PR por ticket.
- Verificação antes de todo push: `make ci` (markdownlint, shellcheck, actionlint e testes dos scripts).
- Scripts rodam em sh POSIX e têm um equivalente em PowerShell com o mesmo comportamento.
- `template/` usa os marcadores `{{PROJECT}}`, `{{OWNER}}` e `{{REPO}}`.
- Merge, tag e release são do dono.
