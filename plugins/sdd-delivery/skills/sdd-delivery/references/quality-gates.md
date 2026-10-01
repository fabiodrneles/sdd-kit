# Gates de qualidade, testes e lições aprendidas

## Sumário

- [Gates de CI](#gates-de-ci)
- [Alvo local equivalente](#alvo-local-equivalente)
- [Testes a partir dos critérios de aceite](#testes-a-partir-dos-critérios-de-aceite)
- [Checagem de mutação](#checagem-de-mutação)
- [Testes de consistência documentação × código](#testes-de-consistência-documentação--código)
- [Releitura adversarial do diff](#releitura-adversarial-do-diff)
- [Release](#release)
- [Lições aprendidas (viraram checagens)](#lições-aprendidas-viraram-checagens)

## Gates de CI

Rode em todo PR e em todo push para `main`. Adapte as ferramentas à linguagem; mantenha os gates.

| Gate | O que garante | Exemplo (Go) |
|---|---|---|
| Dependências limpas | manifesto sem diff após normalizar | `go mod tidy && git diff --exit-code go.mod go.sum` |
| Formatação + lint | estilo e erros estáticos | `go vet`, `golangci-lint` |
| Testes em todos os SOs suportados | portabilidade real | matriz `ubuntu/macos/windows`, `fail-fast: false` |
| Versão mínima e última estável | não quebra quem usa a mínima declarada | matriz `go: [mínima, stable]` |
| Race detector | concorrência | `go test -race` (onde houver suporte) |
| Gate de cobertura | testes não regridem | script que falha abaixo de N% (ex.: 80% no código interno) |
| Smoke test do artefato real | o binário/pacote funciona como o usuário usa | script que compila e roda comandos checando exit codes e saídas |
| Cross-build | todos os alvos de release compilam | matriz `GOOS × GOARCH` |
| Vulnerabilidades | dependências sem CVEs conhecidas | `govulncheck`, `npm audit`, `pip-audit` |
| Ensaio de release | a release funciona sem publicar | `goreleaser release --snapshot` + script que confere os artefatos |
| Docs: markdownlint | Markdown válido (inclusive saídas geradas em Markdown) | `markdownlint-cli2` |
| Docs: links | nenhum link quebrado | `lychee` |
| Docs: comandos | todo comando de exemplo do README roda com exit 0 | script que extrai blocos `bash` e executa |

Boas práticas do workflow: `permissions: contents: read`; `concurrency` cancelando execuções
antigas do mesmo ref; ferramentas obrigatórias no CI mesmo quando opcionais localmente
(ex.: variável `REQUIRE_X=1` faz o teste falhar em vez de pular); fixar versões das ferramentas.

## Alvo local equivalente

Um alvo (`make ci`, `npm run ci`, `just ci`, `tox`) roda o subconjunto viável do CI **com os
mesmos comandos e limites** (mesmo script de cobertura, mesmo limiar, mesmo smoke). Ele é rodado
antes de **todo** push. Comente no topo do arquivo que ele espelha o workflow.

Smoke test típico (`scripts/smoke.sh`): compila o artefato real num diretório temporário,
define `check <descrição> <exit esperado> <comando…>` e `contains <arquivo> <texto>`, roda com
`</dev/null` (sem TTY), cobre caminho feliz, cada exit code documentado, sobrescrita,
entrada inválida e saída de cada formato; acumula falhas e termina com exit ≠ 0 se houver alguma.

## Testes a partir dos critérios de aceite

- Cada `AC-*` vira ao menos um teste automatizado; cite o ID no nome ou comentário do teste.
- **Golden files** para saídas geradas: comparação exata com `testdata/`; uma flag (`-update`)
  ou alvo (`make golden`) regrava após mudança **intencional**; o diff dos goldens é revisado no PR.
- Teste de paridade quando várias saídas derivam do mesmo modelo (nenhum campo omitido).
- Testes de CLI com exit codes e stdin fechado.
- Determinismo: datas e aleatoriedade injetáveis (ex.: `SOURCE_DATE_EPOCH`).

## Checagem de mutação

Para **cada** teste novo, antes do push:

1. Quebre temporariamente o código coberto (inverta uma condição, remova uma linha, troque o valor).
2. Rode só aquele teste: ele **tem** de falhar, pelo motivo esperado.
3. Desfaça a quebra (`git diff` deve mostrar só o que era para mostrar).
4. Registre em "Como foi testado" o que foi quebrado.

Isso já revelou um teste de quebra de página que não testava nada e um teste de alvo de
`Makefile` que passava com o alvo quebrado.

## Testes de consistência documentação × código

Quando a documentação e o código precisam concordar, escreva um teste que **force** isso:

- Todo campo do modelo/schema aparece na documentação do schema (reflexão sobre a struct × doc).
- Todo alvo `make <x>` citado em README/CONTRIBUTING existe no `Makefile`.
- Todo comando de exemplo do README é executado no CI (`scripts/doc-commands.sh`).
- Todo exemplo em `examples/` é validado pelo próprio programa.
- `CONTRIBUTING.md` não contradiz a spec de processo (revisão no PR que tocar algum dos dois).

## Releitura adversarial do diff

Antes do push, leia `git diff origin/<base>...HEAD` como revisor hostil:

- Escopo: só o ticket? Algo que deveria ser outro ticket?
- Esquecidos: arquivo novo sem `git add`? Spec, README, CHANGELOG (se for o PR que cria)?
- Arquivos de status compartilhados alterados num PR de ticket? (não pode)
- Segredos, caminhos locais, saídas geradas, binários.
- Afirmações na descrição do PR: cada uma foi verificada? (ex.: conflitos → simulados com
  `git merge-tree`, não presumidos.)
- README × specs × código dizem a mesma coisa?

## Release

- Disparada por tag `v*` criada **pelo dono**.
- O workflow de release **chama o CI completo** (`uses: ./.github/workflows/ci.yml`) e só
  depois publica binários + checksums (ex.: GoReleaser), com `contents: write` só nesse job.
- Script de verificação dos artefatos (quantidade, alvos, checksums, arquivos incluídos,
  versão embutida) roda também no ensaio de release do CI.
- Changelog agrupado pelos prefixos Conventional Commits.

## Lições aprendidas (viraram checagens)

| Situação | Checagem |
|---|---|
| Um glob na raiz do `.gitignore` (ex.: `cv-craft*`) escondeu arquivos de configuração de ferramentas | `git check-ignore -v <arquivo>` para cada arquivo novo de config; `git status` depois de criar. O `.gitignore` **não aceita comentário no fim da linha** — comentário só em linha própria. |
| Conflitos entre PRs foram descritos de cabeça | Simule: `git merge-tree --write-tree origin/a origin/b`; só escreva o que a simulação mostrou. |
| Proxy do ambiente bloqueou um host (download de ferramenta, link check) | Diga que não foi verificado e por quê; nunca relate sucesso. Deixe o CI verificar e confira o resultado. |
| Biblioteca de extração de texto de PDF truncava caracteres acima de U+00FF, escondendo justamente os bugs de Unicode | Valide a ferramenta de verificação em casos difíceis (acentos, não Latin-1) antes de confiar nela; prefira a ferramenta de referência (ex.: `pdftotext`). |
| Runner Windows tinha uma ferramenta do pacote mas não a outra (`pdftotext` sem `pdfinfo`) | Detecte cada dependência externa separadamente; no CI, exija-as explicitamente. |
| Subir a versão da linguagem quebrou o linter compilado com a versão anterior | Atualize a ferramenta no mesmo PR (pré-condição para o verde), explicando em "O que muda". |
| Base de PR empilhado mergeada e apagada | Confira que o PR foi redirecionado à `main`, que o diff tem só o ticket e que o CI está verde. |
