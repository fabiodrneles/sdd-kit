# Verificação local igual à do CI: rode `make ci` antes de todo push.
GOLANGCI_LINT_VERSION := v2.14.0
# Cobertura mínima de linhas exigida pelo make ci (spec 010 do sdd-kit).
COVERAGE_MIN ?= 80

.DEFAULT_GOAL := ci

.PHONY: ci
ci: lint test build ## Tudo o que o CI verifica

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run

.PHONY: test
test: ## Testes com race detector e cobertura mínima (COVERAGE_MIN)
	go test -race -coverprofile=coverage.out ./...
	@total="$$(go tool cover -func=coverage.out | awk '/^total:/ { sub("%", "", $$3); print $$3 }')"; \
		awk -v t="$$total" -v m="$(COVERAGE_MIN)" 'BEGIN { printf "cobertura: %.1f%% (mínimo %s%%)\n", t, m; exit !(t + 0 >= m + 0) }' || \
		{ echo "cobertura abaixo do mínimo; suba os testes ou ajuste COVERAGE_MIN no Makefile" >&2; exit 1; }

.PHONY: build
build: ## Compila tudo
	go build ./...

.PHONY: docs
docs: ## markdownlint (o CI também verifica links)
	npx --yes markdownlint-cli2@0.23.3

.PHONY: sdd-check
sdd-check: ## Rastreabilidade specs × testes × ROADMAP (falha se houver aviso)
	sh scripts/sdd-check.sh --strict
