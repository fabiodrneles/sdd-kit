# Verificação local igual à do CI: rode `make ci` antes de todo push.
# Convenção: os projetos de teste usam o coverlet.collector (já vem nos
# modelos xunit, nunit e mstest do `dotnet new`).
DOTNET ?= dotnet
# Cobertura mínima de linhas exigida pelo make ci (spec 010 do sdd-kit).
COVERAGE_MIN ?= 80
COVERAGE_DIR := .coverage

.DEFAULT_GOAL := ci

.PHONY: ci
ci: ## Formatação, build sem avisos e testes com cobertura mínima
	$(DOTNET) format --verify-no-changes
	$(DOTNET) build -warnaserror
	rm -rf $(COVERAGE_DIR)
	$(DOTNET) test --no-build --collect:"XPlat Code Coverage" --results-directory $(COVERAGE_DIR)
	sh scripts/coverage.sh $(COVERAGE_DIR) $(COVERAGE_MIN)

.PHONY: deps
deps: ## Restaura os pacotes NuGet
	$(DOTNET) restore

.PHONY: docs
docs: ## markdownlint (o CI também verifica links)
	npx --yes markdownlint-cli2@0.23.3

.PHONY: sdd-check
sdd-check: ## Rastreabilidade specs × testes × ROADMAP (falha se houver aviso)
	sh scripts/sdd-check.sh --strict
