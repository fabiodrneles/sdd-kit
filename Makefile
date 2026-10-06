# Verificações locais equivalentes ao CI (spec 001 FR-3). As versões abaixo são
# lidas também pelo CI e pelo hook de sessão: mude só aqui.
SHELLCHECK_VERSION := v0.10.0
MARKDOWNLINT_VERSION := 0.23.3
ACTIONLINT_VERSION := v1.7.12

.DEFAULT_GOAL := help

.PHONY: help
help: ## Lista os alvos
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-8s %s\n", $$1, $$2}'

.PHONY: ci
ci: md sh test go sdd-check ## Tudo o que o CI verifica sem rede (rode antes de todo push; 001 AC-1, 003 AC-3)

.PHONY: md
md: ## markdownlint em todos os .md
	npx --yes markdownlint-cli2@$(MARKDOWNLINT_VERSION)

.PHONY: sh
sh: ## shellcheck em todos os scripts (versionados ou novos)
	@command -v shellcheck >/dev/null || { echo "shellcheck não instalado (veja .claude/hooks/session-start.sh)"; exit 1; }
	git ls-files -z -co --exclude-standard '*.sh' | xargs -0 -r shellcheck

.PHONY: test
test: ## Testes dos scripts (tests/*.sh)
	@set -e; for t in tests/*.sh; do [ -e "$$t" ] || continue; echo "== $$t"; sh "$$t"; done

.PHONY: go
go: ## axyn (Go): gofmt, go vet e testes com o race detector (spec 021)
	@cd axyn && fmt="$$(gofmt -l .)" && { [ -z "$$fmt" ] || { echo "gofmt: $$fmt"; exit 1; }; }
	cd axyn && go vet ./... && go test -race -count=1 ./...

.PHONY: sdd-check
sdd-check: ## Rastreabilidade specs × testes × ROADMAP do próprio kit (estrita: 007 AC-1)
	sh template/common/scripts/sdd-check.sh --strict

.PHONY: self-sync
self-sync: ## Regenera os workflows do motor do próprio kit a partir do template (015 FR-8)
	sh scripts/self-sync.sh

.PHONY: links
links: ## Verificação de links (requer lychee; o CI sempre roda)
	@if command -v lychee >/dev/null; then lychee --config lychee.toml --no-progress './**/*.md'; \
	else echo "lychee não instalado: links não verificados localmente (o CI verifica)"; fi
