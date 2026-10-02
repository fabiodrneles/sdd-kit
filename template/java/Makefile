# Verificação local igual à do CI: rode `make ci` antes de todo push.
# Funciona com Maven (pom.xml) e com Gradle (build.gradle.kts ou build.gradle),
# usando o wrapper (./mvnw, ./gradlew) quando existe.
MVN := $(if $(wildcard ./mvnw),./mvnw,mvn)
GRADLE := $(if $(wildcard ./gradlew),./gradlew,gradle)
BUILD := $(if $(wildcard build.gradle.kts build.gradle),gradle,maven)
# Cobertura mínima de linhas exigida pelo make ci (spec 010 do sdd-kit).
COVERAGE_MIN ?= 80
JACOCO := org.jacoco:jacoco-maven-plugin:0.8.15

.DEFAULT_GOAL := ci

.PHONY: ci
ci: build-$(BUILD) coverage ## Compila, testa, roda as verificações do build e exige a cobertura mínima

.PHONY: build-maven
build-maven:
	$(MVN) -B -ntp $(JACOCO):prepare-agent verify $(JACOCO):report

.PHONY: build-gradle
build-gradle:
	$(GRADLE) --no-daemon --init-script scripts/jacoco.init.gradle check

# Soma as linhas de todos os relatórios do JaCoCo (um por módulo).
.PHONY: coverage
coverage:
	@total="$$(find . -path '*/jacoco*' -name '*.csv' -not -path './node_modules/*' -exec cat {} + | \
		awk -F, '$$1 != "GROUP" { m += $$8; c += $$9 } END { if (m + c == 0) print 0; else print 100 * c / (m + c) }')"; \
		awk -v t="$$total" -v m="$(COVERAGE_MIN)" 'BEGIN { printf "cobertura: %.1f%% (mínimo %s%%)\n", t, m; exit !(t + 0 >= m + 0) }' || \
		{ echo "cobertura abaixo do mínimo; suba os testes ou ajuste COVERAGE_MIN no Makefile" >&2; exit 1; }

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
