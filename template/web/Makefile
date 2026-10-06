# Verificação local igual à do CI: rode `make ci` antes de todo push.
# Projeto web estático (spec 021 FR-4 do sdd-kit): HTML validado e links locais verificados.
# Precisa de node (só para o npx do html-validate).
HTML_VALIDATE := npx --yes html-validate@11.16.2
# Páginas verificadas: todo .html fora de node_modules, .git e dist.
HTML_FILES = $(shell find . -name '*.html' -not -path './node_modules/*' -not -path './.git/*' -not -path './dist/*' | sort)

.DEFAULT_GOAL := ci

.PHONY: ci
ci: ## Tudo o que o CI verifica
	@[ -n "$(HTML_FILES)" ] || { echo "nenhum .html no projeto" >&2; exit 1; }
	$(HTML_VALIDATE) $(HTML_FILES)
	sh scripts/links.sh $(HTML_FILES)

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
