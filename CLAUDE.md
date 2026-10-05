# CLAUDE.md

Guia rápido para agentes (Claude Code) trabalharem neste repositório. O processo é o da skill `sdd-delivery`, que o próprio kit distribui (constituição, princípio 7).

## Retomar o trabalho (sessão nova ou contexto perdido)

1. Rode `sh template/common/scripts/sdd-resume.sh` (o hook de início de sessão já o roda): mostra o **checkpoint** do épico aberto, entra na branch dele (árvore limpa), lista os **PRs abertos** com o CI de cada um e as **issues abertas** do épico. Continue do "Próximo" do checkpoint, sem esperar instrução. Sem checkpoint, leia o comentário "Estado da fase" mais recente do épico.
2. Continue do próximo passo registrado. Não refaça análise que já está em specs, issues ou PRs.

## O projeto

| Caminho | O que tem |
|---|---|
| `specs/` | Constituição, specs `NNN-nome/spec.md`, `ROADMAP.md`, `ANALYSIS.md` (decisões D1–D4) |
| `plugins/sdd-delivery/` | A skill, empacotada como plugin do Claude Code (spec 002) |
| `.claude-plugin/marketplace.json` | O repositório como marketplace de plugins |
| `template/common/`, `template/<lang>/` | O que a adoção copia para outros repositórios e a sincronização mantém (spec 003) |
| `template/seed/` | Modelos que o projeto preenche (`specs/`, `CHANGELOG.md`): criados uma vez, fora da sincronização (spec 012) |
| `scripts/` | Adoção (`adopt.sh`, `adopt.ps1`), empacotamento e checagens da skill, `e2e-template.sh` (spec 003 AC-4) |
| `tests/` | Testes dos scripts (`make ci`), `adopt.ps1` (Windows no CI) e `fixtures/` (projetos mínimos por linguagem) |

## Comandos

```text
make ci     # markdownlint + shellcheck + testes (template e actionlint incluídos) (rode antes de todo push)
make links  # verificação de links com lychee (se instalado; o CI sempre roda)
```

Numa sessão na web, o hook `.claude/hooks/session-start.sh` instala o shellcheck e o actionlint nas versões do CI.

## Convenções

- **Idioma:** specs, issues, PRs e documentação em português; `README.en.md`, a skill e os comandos de `plugins/` em inglês (a skill escreve no idioma do dono, D12); commits e código em inglês.
- **Commits:** Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `ci:`, `chore:`; `!` para mudança incompatível).
- **Branch:** uma por ticket, `<tipo>/<nº-da-issue>-<descrição>`.
- **PR:** começa com `Closes #N · Épico #M · Spec NNN` e segue o template.
- **Arquivos de status** (status das specs, checkboxes do ROADMAP, CHANGELOG) só mudam no PR de fechamento da fase.
- **Merge** é do dono. **Tag e release**: pelo workflow *Release tag* (Actions), que o agente pode disparar quando o dono pedir o fechamento da versão.

## Armadilhas

- O PR de fechamento sobe a versão no `plugin.json` do `sdd-delivery`, no `KIT_REF` do `adopt.sh`/`adopt.ps1` e no `curl` dos READMEs; o `tests/plugin.sh` exige que batam, e o *Release tag* falha se a tag não bater com o `plugin.json`.
- Arquivos em `template/` usam marcadores `{{PROJECT}}`, `{{OWNER}}`, `{{REPO}}`; o lint deles roda sobre uma cópia com os marcadores substituídos.
- Arquivos gerados por ferramentas (ex.: `dotnet new`) podem vir com BOM UTF-8 ou CRLF; o `adopt.ps1` remove o BOM e o teste de adoção compara as árvores. Normalize antes do commit: sem `pwsh` local, esse teste só roda no CI.
- Ferramenta antiga antes no PATH (ex.: `/usr/local/bin/golangci-lint` de outra versão do Go) ofusca a versão certa do `GOPATH/bin`; o `sdd-doctor.sh` aponta e imprime o `export PATH=...`. O shellcheck quebra em locale que não é UTF-8: use `LC_ALL=C.UTF-8`.
- Scripts precisam rodar em sh POSIX (Linux e macOS) e o equivalente em PowerShell no Windows; o shellcheck roda com `-s sh` nos scripts de adoção.

## Economia de uso

Cada regra abaixo reduziu o gasto de sessões reais; aplique desde a primeira mensagem.

- **Checkpoint contínuo:** a sessão pode acabar a qualquer momento, sem aviso. Depois de cada passo (commit, PR aberto, CI verde, merge), faça push e rode `sh template/common/scripts/sdd-checkpoint.sh save "feito" "próximo passo" ["o que bloqueia"]`: ele atualiza um único comentário de checkpoint no épico aberto, com branch, commit e PRs. Nunca deixe mais de um passo só na máquina; trabalho a meio vai num commit WIP na branch do ticket. O hook Stop mantém o estado do git atualizado a cada resposta.
- Leia trechos (`sed -n 'a,bp'`, `grep -n`) em vez de arquivos inteiros, e não releia o que já leu, nem depois de editar.
- Junte leituras e checagens independentes num comando só.
- Saída longa vai para um arquivo; mostre só o código de saída e o fim: `make ci > /tmp/ci.log 2>&1; echo "exit $?"; tail -n 3 /tmp/ci.log`.
- Valide tudo com `make ci`, uma vez, antes do push.
- CI dos PRs: `sh template/common/scripts/sdd-ci.sh '#PR'` (uma linha por check e só o fim do log das falhas). Não assine os eventos do PR; se a sessão assinar sozinha, cancele.
- Entrega do ticket num comando: `sh template/common/scripts/sdd-pr.sh [--spec NNN] [--dry-run]` (merge da `main`, `make ci`, push, PR ou o já aberto, CI do PR e checkpoint; não faz merge).
- Ambiente antes de `make ci`: `sh template/common/scripts/sdd-doctor.sh [--check]` confere e conserta ferramentas nas versões do CI, locale UTF-8 e PATH (uma linha por item); rode quando o hook de sessão não rodou (ex.: repositório anexado no meio da sessão) ou o `make ci` falhar por ferramenta.
- Fechamento da versão num comando: `sh template/common/scripts/sdd-release.sh [--dry-run]` (versão pelo go-release-manager; X.Y.Z só força; branch `chore/release-vX.Y.Z`, rascunho do CHANGELOG pelos PRs mesclados, versões listadas em `.sdd-release`, commit e PR); com o PR mesclado, `sh template/common/scripts/sdd-release.sh --tag X.Y.Z` (dispara o *Release tag* ou cria a tag; não faz merge).
- Edição mecânica por script que falha se o trecho não existir (ex.: `assert old in s` antes do `replace` em Python), sem reler o arquivo.
- Checagem de mutação sem reler: copie o arquivo, quebre, rode o teste, restaure com `cp`.
- Confira ferramentas e rede antes de começar (o proxy pode bloquear downloads); tente o gerenciador de pacotes do sistema.
- Nas ferramentas do GitHub, peça só os campos necessários (`fields`, `minimal_output`, `perPage`).
- Branch de PR já mergeado: recomece da `main` num comando (`git fetch origin main && git checkout -B <branch> origin/main`).
- Subagentes só para buscas amplas de leitura, num modelo pequeno.
- "Estado da fase" só nos marcos e em poucas linhas; no chat, três linhas (feito, falta, bloqueia); detalhes nos PRs e nas issues.
