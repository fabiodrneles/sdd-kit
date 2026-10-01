# Verificação local igual à do CI: rode `make ci` antes de todo push.
# Convenção: o pyproject.toml declara ruff e pytest no extra "dev".
PYTHON ?= python3

.DEFAULT_GOAL := ci

.PHONY: ci
ci: ## Lint, formatação e testes
	$(PYTHON) -m ruff check .
	$(PYTHON) -m ruff format --check .
	$(PYTHON) -m pytest

.PHONY: deps
deps: ## Instala o projeto com as dependências de desenvolvimento
	$(PYTHON) -m pip install -e ".[dev]"

.PHONY: docs
docs: ## markdownlint (o CI também verifica links)
	npx --yes markdownlint-cli2@0.23.3

.PHONY: sdd-check
sdd-check: ## Rastreabilidade specs × testes × ROADMAP (--strict para falhar)
	sh scripts/sdd-check.sh
