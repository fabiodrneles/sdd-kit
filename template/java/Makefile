# Verificação local igual à do CI: rode `make ci` antes de todo push.
# Usa o Maven Wrapper (./mvnw) quando existe.
MVN := $(if $(wildcard ./mvnw),./mvnw,mvn)

.DEFAULT_GOAL := ci

.PHONY: ci
ci: ## Compila, testa e roda as verificações do pom.xml
	$(MVN) -B -ntp verify

.PHONY: docs
docs: ## markdownlint (o CI também verifica links)
	npx --yes markdownlint-cli2@0.23.3
