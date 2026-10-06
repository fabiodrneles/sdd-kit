# Tarefas do benchmark com e sem sdd-kit (10 tarefas)

As mesmas três tarefas nos dois lados, num projeto Go real ([sdd-kit-demo](https://github.com/fabiodrneles/sdd-kit-demo), o `linkcheck`). Cada título de nível 2 abre uma tarefa; a linha `Arquivos:` é o que o pacote do lado com sdd-kit lista como arquivos prováveis.

## --quiet

Adicione a opção `--quiet` ao `linkcheck`: sem nenhum problema, não imprime nada; com problemas, imprime só uma linha com a contagem (`N problemas`). O código de saída não muda. Cubra os dois casos com testes.

Arquivos: main.go, main_test.go

## --version

Adicione a opção `--version`: imprime `linkcheck <versão>` (a variável `version` do `main.go`) e sai com 0, sem verificar nenhum arquivo. Cubra com um teste.

Arquivos: main.go, main_test.go

## --format github

Adicione `github` aos valores de `--format`: cada problema vira uma anotação do GitHub Actions, `::error file=ARQUIVO,line=LINHA::DESTINO: MOTIVO`. Os formatos `text` e `json` não mudam, e um formato inválido continua saindo com 2. Cubra com testes.

Arquivos: main.go, format_test.go

## --warn-only

Adicione a opção `--warn-only`: imprime os problemas como hoje, mas sai com 0 mesmo quando há problemas. Erros de uso continuam saindo com 2. Cubra com testes.

Arquivos: main.go, main_test.go

## --max N

Adicione a opção `--max N`: imprime no máximo N problemas e, se houver mais, uma última linha `... e mais M`. O código de saída não muda. N precisa ser maior que zero (senão, uso inválido, saída 2). Cubra com testes.

Arquivos: main.go, main_test.go

## --exclude

Adicione a opção `--exclude PASTA`, repetível: ao percorrer as pastas, não entra nas pastas com esse nome (ex.: `vendor`, `node_modules`). Cubra com testes.

Arquivos: main.go, main_test.go

## --ignore na linha de comando

Adicione a opção `--ignore PADRÃO`, repetível, com o mesmo efeito da lista `ignore` do `.linkcheck.yml`; as duas listas se somam. Cubra com testes.

Arquivos: main.go, config_test.go, internal/config/config.go

## Arquivos .markdown

Ao percorrer as pastas, verifique também os arquivos com extensão `.markdown`, além de `.md`. Cubra com um teste.

Arquivos: main.go, main_test.go

## --summary

Adicione a opção `--summary`: no fim, imprime na saída de erro uma linha `N arquivos verificados, M problemas`. Não muda a saída normal nem o código de saída. Cubra com testes.

Arquivos: main.go, main_test.go

## mailto e tel

Links `mailto:` e `tel:` não são arquivos: o `linkcheck` não deve acusá-los como arquivo inexistente, com ou sem `--external`. Cubra com testes.

Arquivos: internal/check/check.go, internal/markdown/markdown.go, main_test.go
