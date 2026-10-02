# Verificação local igual à do CI: rode `make ci` antes de todo push.
# Usa o Maven Wrapper (./mvnw) quando existe.
MVN := $(if $(wildcard ./mvnw),./mvnw,mvn)
# Cobertura mínima de linhas exigida pelo make ci (spec 010 do sdd-kit).
COVERAGE_MIN ?= 80
JACOCO := org.jacoco:jacoco-maven-plugin:0.8.15

.DEFAULT_GOAL := ci

.PHONY: ci
ci: ## Compila, testa, roda as verificações do pom.xml e exige a cobertura mínima
	$(MVN) -B -ntp $(JACOCO):prepare-agent verify $(JACOCO):report
	@total="$$(awk -F, 'NR > 1 { m += $$8; c += $$9 } END { if (m + c == 0) print 0; else print 100 * c / (m + c) }' target/site/jacoco/jacoco.csv)"; \
		awk -v t="$$total" -v m="$(COVERAGE_MIN)" 'BEGIN { printf "cobertura: %.1f%% (mínimo %s%%)\n", t, m; exit !(t + 0 >= m + 0) }' || \
		{ echo "cobertura abaixo do mínimo; suba os testes ou ajuste COVERAGE_MIN no Makefile" >&2; exit 1; }

.PHONY: docs
docs: ## markdownlint (o CI também verifica links)
	npx --yes markdownlint-cli2@0.23.3

.PHONY: sdd-check
sdd-check: ## Rastreabilidade specs × testes × ROADMAP (falha se houver aviso)
	sh scripts/sdd-check.sh --strict
