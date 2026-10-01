# Verificações locais equivalentes ao CI (spec 001 FR-3). As versões abaixo são
# lidas também pelo CI e pelo hook de sessão: mude só aqui.
SHELLCHECK_VERSION := v0.10.0
MARKDOWNLINT_VERSION := 0.23.3

.DEFAULT_GOAL := help

.PHONY: help
help: ## Lista os alvos
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-8s %s\n", $$1, $$2}'

.PHONY: ci
ci: md sh test ## Tudo o que o CI verifica sem rede (rode antes de todo push)

.PHONY: md
md: ## markdownlint em todos os .md
	npx --yes markdownlint-cli2@$(MARKDOWNLINT_VERSION)

.PHONY: sh
sh: ## shellcheck em todos os scripts versionados
	@command -v shellcheck >/dev/null || { echo "shellcheck não instalado (veja .claude/hooks/session-start.sh)"; exit 1; }
	git ls-files -z '*.sh' | xargs -0 -r shellcheck

.PHONY: test
test: ## Testes dos scripts (tests/*.sh)
	@set -e; for t in tests/*.sh; do [ -e "$$t" ] || continue; echo "== $$t"; sh "$$t"; done

.PHONY: links
links: ## Verificação de links (requer lychee; o CI sempre roda)
	@if command -v lychee >/dev/null; then lychee --config lychee.toml --no-progress './**/*.md'; \
	else echo "lychee não instalado: links não verificados localmente (o CI verifica)"; fi
