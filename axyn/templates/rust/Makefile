# Verificação local igual à do CI: rode `make ci` antes de todo push.
# Convenção: cobertura com o cargo-llvm-cov (`make deps` instala).
CARGO ?= cargo
# Cobertura mínima de linhas exigida pelo make ci (spec 010 do sdd-kit).
COVERAGE_MIN ?= 80

.DEFAULT_GOAL := ci

.PHONY: ci
ci: ## Formatação, clippy sem avisos e testes com cobertura mínima
	$(CARGO) fmt --all -- --check
	$(CARGO) clippy --all-targets -- -D warnings
	$(CARGO) llvm-cov --all-targets --fail-under-lines $(COVERAGE_MIN)

.PHONY: deps
deps: ## Instala clippy, rustfmt e o cargo-llvm-cov, se faltarem
	rustup component add clippy rustfmt llvm-tools-preview
	$(CARGO) llvm-cov --version >/dev/null 2>&1 || $(CARGO) install cargo-llvm-cov --locked

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
