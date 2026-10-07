# Verificação local igual à do CI: rode `make ci` antes de todo push.
# Convenção: o pyproject.toml declara ruff e pytest no extra "dev".
PYTHON ?= python3
# Cobertura mínima de linhas exigida pelo make ci (spec 010 do sdd-kit).
COVERAGE_MIN ?= 80

# Artefatos do make ci (cobertura) ficam dentro de .git/, que o git nunca versiona:
# nada solto na árvore depois do make ci, sem depender do .gitignore (#180).
SDD_OUT := $(shell git rev-parse --git-path sdd-out 2>/dev/null || echo .sdd-out)

.DEFAULT_GOAL := ci

.PHONY: ci
ci: ## Lint, formatação e testes com cobertura mínima (precisa de pytest-cov no extra dev)
	$(PYTHON) -m ruff check .
	$(PYTHON) -m ruff format --check .
	@mkdir -p $(SDD_OUT)
	COVERAGE_FILE=$(SDD_OUT)/.coverage $(PYTHON) -m pytest --cov=. --cov-report=term --cov-fail-under=$(COVERAGE_MIN)

.PHONY: deps
deps: ## Instala o projeto com as dependências de desenvolvimento
	$(PYTHON) -m pip install -e ".[dev]"

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
