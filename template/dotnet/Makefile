# Verificação local igual à do CI: rode `make ci` antes de todo push.
# Convenção: os projetos de teste usam o coverlet.collector (já vem nos
# modelos xunit, nunit e mstest do `dotnet new`).
DOTNET ?= dotnet
# Cobertura mínima de linhas exigida pelo make ci (spec 010 do sdd-kit).
COVERAGE_MIN ?= 80
COVERAGE_DIR := $(SDD_OUT)/coverage

# Artefatos do make ci (cobertura) ficam dentro de .git/, que o git nunca versiona:
# nada solto na árvore depois do make ci, sem depender do .gitignore (#180).
SDD_OUT := $(shell git rev-parse --git-path sdd-out 2>/dev/null || echo .sdd-out)

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

.PHONY: linkcheck
linkcheck: ## Links quebrados nos .md (precisa do lychee; o CI sempre verifica)
	@command -v lychee >/dev/null || { echo "lychee não instalado: https://lychee.cli.rs" >&2; exit 1; }
	lychee --config lychee.toml --no-progress './**/*.md'

.PHONY: sdd-check
sdd-check: ## Rastreabilidade specs × testes × ROADMAP (falha se houver aviso)
	sh scripts/sdd-check.sh --strict
