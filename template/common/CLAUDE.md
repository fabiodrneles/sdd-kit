# CLAUDE.md

Guia rápido para agentes (Claude Code) trabalharem no {{PROJECT}} sem redescobrir o projeto a cada sessão. O processo é o da skill [`sdd-delivery`](https://github.com/fabiodrneles/sdd-kit).

## Retomar o trabalho (sessão nova ou contexto perdido)

1. Leia o **comentário "Estado da fase"** mais recente no épico aberto (issues com o label `épico`): PRs, estado do CI, decisões e próximo passo.
2. Liste os **PRs abertos** e o CI de cada um, e as **issues abertas** da fase.
3. Continue do próximo passo registrado. Não refaça análise que já está em specs, issues ou PRs.

O estado do trabalho vive no GitHub, e não na conversa. Abra o ticket e o PR assim que a tarefa começar e terminar, e atualize o comentário de estado do épico a cada marco.

## O projeto

<!-- Preencha: o que o projeto faz e a tabela "caminho → o que tem". -->

| Caminho | O que tem |
|---|---|
| `specs/` | Constituição, specs `NNN-nome/spec.md`, `ROADMAP.md`, `ANALYSIS.md` |

## Comandos

```text
make ci     # a mesma verificação do CI (rode antes de todo push)
make docs   # markdownlint (os links são verificados no CI)
make linkcheck  # links quebrados nos .md (precisa do lychee)
make sdd-check  # cada AC de spec In Progress/Done citado num teste ("NNN AC-n")
```

Numa sessão na web, o hook `.claude/hooks/session-start.sh` instala as dependências e as ferramentas do CI.

## Convenções

- **Idioma:** specs, issues, PRs e documentação em português; commits e código (identificadores) em inglês.
- **Commits:** Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `ci:`, `chore:`; `!` para mudança incompatível).
- **Branch:** uma por ticket, `<tipo>/<nº-da-issue>-<descrição>`, a partir da `main`.
- **PR:** começa com `Closes #N · Épico #M · Spec NNN` e segue o template.
- **Arquivos de status** (status das specs, checkboxes do ROADMAP, CHANGELOG) só mudam no PR de fechamento da fase.
- **Merge, tag e release** são do dono, salvo delegação explícita para uma rodada.
- **Comandos do README:** um bloco ```` ```bash ```` com `<!-- doc-commands -->` na linha anterior roda no CI (`sh scripts/doc-commands.sh`); marque os exemplos que devem continuar funcionando.

## Armadilhas já conhecidas

<!-- Registre aqui o que já custou tempo: arquivos gerados, testes frágeis, diferenças entre sistemas. -->

## Economia de uso

Cada regra abaixo reduziu o gasto de sessões reais; aplique desde a primeira mensagem.

- Leia trechos (`sed -n 'a,bp'`, `grep -n`) em vez de arquivos inteiros, e não releia o que já leu, nem depois de editar.
- Junte leituras e checagens independentes num comando só.
- Saída longa vai para um arquivo; mostre só o código de saída e o fim: `make ci > /tmp/ci.log 2>&1; echo "exit $?"; tail -n 3 /tmp/ci.log`.
- Valide tudo com `make ci`, uma vez, antes do push.
- CI dos PRs: `sh scripts/sdd-ci.sh '#PR'` (uma linha por check e só o fim do log das falhas). Não assine os eventos do PR; se a sessão assinar sozinha, cancele.
- Edição mecânica por script que falha se o trecho não existir (ex.: `assert old in s` antes do `replace` em Python), sem reler o arquivo.
- Checagem de mutação sem reler: copie o arquivo, quebre, rode o teste, restaure com `cp`.
- Confira ferramentas e rede antes de começar (o proxy pode bloquear downloads); tente o gerenciador de pacotes do sistema.
- Nas ferramentas do GitHub, peça só os campos necessários (`fields`, `minimal_output`, `perPage`).
- Branch de PR já mergeado: recomece da `main` num comando (`git fetch origin main && git checkout -B <branch> origin/main`).
- Subagentes só para buscas amplas de leitura, num modelo pequeno.
- "Estado da fase" só nos marcos e em poucas linhas; no chat, três linhas (feito, falta, bloqueia); detalhes nos PRs e nas issues.
