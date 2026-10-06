# 010 — Esteira de qualidade criada pela adoção

- **Prioridade:** P1
- **Status:** Done — entregue na `v0.2.0`
- **Código afetado:** `template/<lang>/`, `scripts/adopt.sh`, `scripts/adopt.ps1`, `plugins/sdd-release/`
- **Resolve:** achados 1 e 2 do #51; Gradle (caso exemplo-automacao-page-object)

## Contexto

A adoção já cria o `ci.yml` por linguagem, o `docs.yml`, o `sdd-sync.yml` e o `Makefile`. Faltam três coisas para quem cria um repositório sem saber montar uma esteira: o CI não exige cobertura mínima (só o Go mostra o número, sem falhar), um repositório vazio recebe um CI que nasce vermelho, e só o Go tem release. No caso Java, o template também não servia para Gradle. Princípio: **depois da adoção, o primeiro push já roda lint, testes com cobertura mínima e build, e uma tag publica a release.**

## Requisitos funcionais

- **FR-1** O `make ci` de cada linguagem (Go, Node, Java, Python) MUST falhar quando a cobertura de linhas ficar abaixo de `COVERAGE_MIN` (padrão 80, configurável no `Makefile`) e MUST imprimir a cobertura medida.
- **FR-2** A adoção MUST aceitar `--skeleton`, que cria o mínimo para o CI nascer verde num repositório vazio (um módulo, um teste e a configuração de lint e cobertura), sem sobrescrever arquivos existentes.
- **FR-3** O template Java MUST suportar Gradle (Kotlin DSL) além de Maven; a adoção MUST escolher pelo arquivo de build presente (`build.gradle.kts`/`build.gradle` ou `pom.xml`).
- **FR-4** Cada template MUST trazer um workflow `release-tag.yml` disparável (inputs `release-as` e `ref`) que cria a tag e publica a release com notas geradas; a próxima versão vem do go-release-manager quando `release-as` fica vazio.
- **FR-5** A esteira MUST ser validada num repositório real sem CI (qa-portfolio, Node), registrando os achados no #51.

## Critérios de aceite

- **AC-1** Dado o projeto mínimo de cada linguagem em `tests/fixtures/` com cobertura abaixo de `COVERAGE_MIN`, quando `make ci` roda, então falha e mostra a cobertura; com cobertura suficiente, passa.
- **AC-2** Dado um diretório vazio, quando a adoção roda com `--skeleton` para cada linguagem, então `make ci` passa sem edição manual; rodar de novo não sobrescreve nada.
- **AC-3** Dado um projeto Gradle, quando a adoção roda, então o `Makefile` e o `ci.yml` usam o Gradle e `make ci` passa.
- **AC-4** Dados os templates, quando o CI do kit roda, então cada `release-tag.yml` passa no actionlint e aceita `release-as` e `ref`.
- **AC-5** Dado o qa-portfolio adotado com a esteira, quando um PR é aberto, então o CI roda lint, testes com cobertura mínima e build.

## Mudanças

### v1.7.0

- MODIFIED FR-1 — o `go test` do template Go usa `-count=1`: com o cache quente e um `package main` coberto por `-coverpkg=./...` (Go 1.25 ou mais novo), o go reaproveitava metadados de cobertura de uma versão antiga do arquivo e a cobertura saía menor que a real (#172).
- ADDED FR-1 — válvula do cache: `scripts/sdd-cover-guard.sh` confere o perfil antes do gate (blocos sobrepostos ou além do fim do arquivo = versões misturadas) e, se vier misturado, refaz o `go test` num `GOCACHE` frio e descartável, avisando numa linha; `SDD_COVER_GUARD=off` desliga (#172).

### v1.3.1

- MODIFIED FR-1 — a cobertura do template Go usa `-coverpkg=./...`, e conta pacotes testados só por outros pacotes (achado no sdd-kit-demo, v1.3.1).
