# Verificação local igual à do CI: rode `make ci` antes de todo push.
# Os scripts "lint" e "build" do package.json são opcionais; "test" é obrigatório.
# A cobertura é medida pelo c8 (V8), que funciona com qualquer executor de testes
# em Node (node --test, Vitest, Mocha); quem já mede no próprio executor pode trocar.
# Cobertura mínima de linhas exigida pelo make ci (spec 010 do sdd-kit).
COVERAGE_MIN ?= 80
C8 := npx --yes c8@12.0.0
# Acessibilidade (spec 011 do sdd-kit): páginas HTML que o axe verifica depois do
# build, ex.: A11Y_PAGES := dist/index.html. Vazio desliga a checagem.
A11Y_PAGES ?=
A11Y := npx --yes -p axe-core@4.13.0 -p jsdom@26.1.0 node scripts/a11y

.DEFAULT_GOAL := ci

.PHONY: ci
ci: ## Tudo o que o CI verifica
	npm run lint --if-present
	CI=true $(C8) --all --check-coverage --lines $(COVERAGE_MIN) --reporter=text-summary npm test
	npm run build --if-present
	@[ -z "$(A11Y_PAGES)" ] || $(MAKE) --no-print-directory a11y

.PHONY: a11y
a11y: ## Acessibilidade com o axe nas páginas de A11Y_PAGES
	$(A11Y) $(A11Y_PAGES)

.PHONY: deps
deps: ## Instala as dependências exatamente como no lockfile
	npm ci

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
