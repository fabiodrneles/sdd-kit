# Tarefas do benchmark com e sem sdd-kit

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
