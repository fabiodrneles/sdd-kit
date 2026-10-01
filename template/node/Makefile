# Verificação local igual à do CI: rode `make ci` antes de todo push.
# Os scripts "lint" e "build" do package.json são opcionais; "test" é obrigatório.
.DEFAULT_GOAL := ci

.PHONY: ci
ci: ## Tudo o que o CI verifica
	npm run lint --if-present
	CI=true npm test
	npm run build --if-present

.PHONY: deps
deps: ## Instala as dependências exatamente como no lockfile
	npm ci

.PHONY: docs
docs: ## markdownlint (o CI também verifica links)
	npx --yes markdownlint-cli2@0.23.3
