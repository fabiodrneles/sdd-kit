# Changelog

Todas as mudanças relevantes deste projeto. Formato [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/), versões [SemVer](https://semver.org/lang/pt-BR/).

## [Unreleased]

## [1.23.8] - 2026-10-08

### Corrigido

- ticket só de testes: se o código guardado de uma tentativa anterior mudou produção, o ticket recomeça da base, só com testes, sem testes quebrados que empurrem o modelo a implementar a spec de novo (#379)
- trava do ticket só de testes: depois de cada tentativa, o axyn desfaz sozinho qualquer mudança em código de produção antes dos portões; os testes precisam passar com o código como ele está (#379)
- comandos do modelo no opencode ganham 15 minutos em vez de 2 (`OPENCODE_EXPERIMENTAL_BASH_DEFAULT_TIMEOUT_MS`), e o pedido diz para usar um tempo maior no `make test`: o `make ci` lento no Windows era cortado no meio (#379)

## [1.23.7] - 2026-10-08

### Corrigido

- no Windows, o pedido chega inteiro ao modelo: o `opencode.cmd` corta o texto nas aspas e em `& | < >`, e o modelo ficava sem o ticket ("qual ticket devo trabalhar?"); esses caracteres são trocados por equivalentes que o cmd.exe não corta (#377)
- `main.go` sai da cobertura do código novo: `main()` abre e fecha o programa e nenhum teste a roda (#377)

## [1.23.6] - 2026-10-08

### Corrigido

- no ticket só de testes, o axyn volta os arquivos de produção à versão da base antes de cada tentativa e ao retomar o código guardado; os testes ficam, e o modelo é avisado para apagar os testes que dependiam dessas mudanças (#375)

## [1.23.5] - 2026-10-08

### Corrigido

- o ticket só de testes (o da cobertura) que muda código de produção é reprovado pelo novo portão `[escopo]`; só os arquivos com erros que o `make ci` já tinha na base podem mudar, e o pedido ao modelo diz para não implementar nada da spec ali (#373)

## [1.23.4] - 2026-10-08

### Corrigido

- no Windows, um `make ci` que falha só porque o antivírus prendeu o arquivo de teste (`unlinkat ... sendo usado por outro processo`) é rodado de novo, até 2 vezes; uma falha de verdade não é repetida (#370)
- a mensagem de cobertura diz que mede a branch base e que os testes de um ticket só contam depois que o PR dele é mesclado (#370)
- README: o aviso `unlinkat` e como excluir as pastas `go-build*` do Defender (#370)

## [1.23.3] - 2026-10-08

### Corrigido

- linhas novas de teste não contam no limite de tamanho: um ticket só de testes para a cobertura deixa de ser reprovado por `[size]` (#368)
- no Windows, o opencode é apontado para o Git Bash (`SHELL` e `OPENCODE_GIT_BASH_PATH`), e cada pedido ao modelo diz como escrever comandos para o terminal dele (sem `&&`, `||`, `grep`; argumentos com `=` entre aspas) (#369)

## [1.23.2] - 2026-10-08

### Corrigido

- "git diff falhou: exit status 128" nos portões: uma pasta com repositório próprio e arquivos com acento ficam fora do diff, e o erro passa a trazer a mensagem do git (#366)
- no Windows, o axyn põe o `sh` e o `bash` do Git no PATH: o `make ci` e os comandos dos modelos funcionam como no Linux (#366)
- perfis de cobertura que o modelo deixa soltos (`coverage.out` e parecidos) são apagados antes dos portões e não entram no PR (#366)
- `axyn run` fora da pasta de um projeto avisa na hora (#366)
- o ticket de cobertura diz quando parar: medir com `make test`, parar ao passar da meta e não cobrir `main()` nem `os.Exit` (#367)

## [1.23.1] - 2026-10-08

### Corrigido

- atualizar o axyn ou o opencode não refaz mais a avaliação inteira: só um modelo novo é avaliado, e o resto fica como estava (#364)
- status e perguntas coloridos no terminal: estado, rótulos, comandos a digitar e arquivos sem teste (#364)
- na janela clássica do Windows, o título mostra o spinner e o tempo a cada segundo, e a linha do progresso se atualiza a cada 30 s, sem atrapalhar a rolagem (#364)
- a dica da cobertura mostra um comando que o PowerShell aceita; a retomada não aparece mais como "avaliação dos modelos" (#364)

## [1.23.0] - 2026-10-08

### Adicionado

- `axyn stop` para a execução ou a avaliação em andamento, mesmo em segundo plano, junto com os modelos que o axyn chamou; nada se perde, e `axyn run --resume` continua (#362)

### Corrigido

- a avaliação dos modelos tem uma lista própria (`.axyn/bench-runs/`): depois de um `axyn bench`, o `axyn run --resume`, o `axyn decide`, o `axyn status` e o `axyn history` voltam a mostrar o trabalho do projeto, e não a avaliação (#362)
- a avaliação mostra a conta (`27 modelo(s) em 2 etapa(s) = 54 tentativas`) e em que modelo está; os nomes das etapas podem vir antes ou depois das opções (#361)
- `Missing Authentication header` (chave de API que não chega ao provedor) conta como indisponível, e não como nota zero (#361)
- relatório no terminal mais enxuto: falhas iguais numa linha só; no `.md`, modelo indisponível aparece sem nota (#361)

## [1.22.1] - 2026-10-08

### Corrigido

- na janela clássica do Windows PowerShell, símbolos que a fonte tem (`√ × ►`) no lugar das caixinhas, e o progresso sem animação, para a tela poder ser rolada (#359)
- `p` e Enter pausam a escrita do painel para ler com calma, e a avaliação continua (#359)
- log colorido por tipo de linha: ações, comandos, testes que passaram, falhas, tabelas (#359)
- o diff de cada edição vira uma linha por arquivo, e o texto do modelo deixa de ser cortado (#359)
- modelo sem chave de API válida ou sem crédito fica como indisponível, e não como nota zero; depois da primeira queda, as outras tarefas dele na rodada são puladas na hora (#360)

## [1.22.0] - 2026-10-08

### Adicionado

- `axyn bench` roda em segundo plano com um painel na mesma janela: Enter alterna entre o progresso e o log ao vivo, Ctrl + C fecha só o painel e `axyn bench --watch` volta (#357)
- comando curto: `axyn bench plano testes` refaz só essas etapas, com o nome em português ou em inglês (#357)
- `axyn logs` acompanha o log de forma legível, com cores (#357)
- relatório da avaliação colorido no terminal: notas por cor, quem faz cada etapa e só o que deu errado (#358)
- README explica a avaliação, as teclas do painel, como personalizar e os problemas comuns (#357)

### Corrigido

- resultados de uma rodada anterior aparecem marcados com `*`, e modelos sem chave saem do relatório e da escolha (#358)
- modelo sem rota no servidor conta como indisponível, e não como nota zero (#357)
- acentos certos no PowerShell: logs e relatórios gravados com a marca de UTF-8 (#357)

## [1.21.1] - 2026-10-07

### Corrigido

- avaliação justa no Windows e no plano gratuito do opencode, paralelismo conforme a máquina (#352)

## [1.21.0] - 2026-10-07

### Adicionado

- indicador de progresso nas cores do axyn, axyn update e o diff de cada tentativa da avaliação (#349)

## [1.20.0] - 2026-10-07

### Adicionado

- motor de avaliação dos modelos (axyn bench), modo guiado e troca automática de modelo (#346)

## [1.19.0] - 2026-10-07

### Adicionado

- axyn release fecha uma versão do projeto com o go-release-manager (#341)

### Corrigido

- cobertura medida a cada execução, queda detectada, axyn retry (#343)

## [1.18.1] - 2026-10-07

### Corrigido

- medir a cobertura quando o lint da base falha e voltar sempre à base (#338)

## [1.18.0] - 2026-10-07

### Adicionado

- cobertura de testes automática, sem editar o Makefile (#335)

## [1.17.0] - 2026-10-07

### Adicionado

- avisar quando sair uma versão nova, com as novidades e como atualizar (#329)

### Corrigido

- explicar o ticket parado, cobertura a partir de 0 em projeto sem testes, terminal visível no Windows (#331)

## [1.16.0] - 2026-10-07

### Adicionado

- `axyn model`: escolhe o modelo numa lista numerada dos gratuitos do opencode, sem editar o `config.yaml`; o `axyn doctor` mostra o modelo em uso (#319)
- `axyn status --watch`: acompanha a execução e avisa (bipe e notificação) quando ela termina, para ou faz uma pergunta (#325)
- Ticket que não passa nos portões vira PR em rascunho, com o código, os erros e um texto pronto para pedir ajuda a outra IA; uma correção feita na branch WIP passa pelos portões no `axyn run --resume` (#325)
- README: "Problemas comuns e como resolver", e o guia diz onde rodar cada comando e a ordem (opencode fechado até o passo 8) (#317, #325)

### Corrigido

- Os instaladores põem o axyn no PATH sozinhos e, no Windows, não mostram mais o erro do git fora de um repositório (#317)
- O agente do plano usa o modelo escolhido, e as ferramentas de um `make ci` que já existe (como o `golangci-lint`) são conferidas antes do primeiro ticket, com o comando de instalação (#319)
- No Windows, o `make ci` usa o `sh` do Git sozinho (#325)
- Execução interrompida (travamento, terminal fechado) é reconhecida na hora e retomada com o código salvo num WIP; o portão não deixa mais o índice do git quebrando o `git stash` (#325)
- O `/axyn` roda num agente só com as ferramentas de conversa, mostra "axyn, por favor: (seu pedido)" e não sai mais do roteiro; uma execução em andamento mostra o andamento em vez de erro (#325)

## [1.15.0] - 2026-10-07

### Adicionado

- prepare projects without CI from embedded templates (#307)
- print the full install command for missing tools (#310)
- axyn doctor and axyn setup configure the repository (#311)
- axyn history writes one file with everything a run did (#312)
- step-by-step guide for beginners in the README, from the terminal to a merged PR (#314)

### Corrigido

- never reopen a delivered phase in sdd-on-release-merge.sh (#302)
- make /axyn work in the real opencode (#303)
- start every ticket from the base branch, with unique names (#309)

## [1.14.0] - 2026-10-07

### Adicionado

- axyn skeleton in Go with axyn version, covered by make ci (#273)
- axyn gate with CI, loosened-test, protected-path and diff-size gates (#274)
- mcp server with axyn_plan, axyn_next, axyn_gate and axyn_ship (#275)
- sdd-relay.sh --auto-merge merges the ticket PR once its CI is green (#277)
- install adds the mcp server, agents and /axyn command to opencode.json without erasing the user's config (#278)
- web template with HTML validation and local link check (spec 021 FR-4, AC-4) (#280)
- model ladder with per-ticket attempt, model and cost record (#281)
- recovery without changing model, from exact diagnosis to a question to the user (spec 021 FR-8, AC-6) (#283)
- auto merge in one command (sdd-auto-merge.sh on|off|status) for every PR (#284)
- engine drives the loop, /axyn only calls axyn_run (spec 021 FR-9, AC-7) (#286)
- one-command install script and AC-5 e2e with a fake agent (#287)
- release binaries, Go-free install and user guide (T79) (#290)

### Corrigido

- isolate the sdd-auto-merge test from the CI runner environment (#285)
- forbid git stash in the relay agent prompt (#289)

## [1.13.0] - 2026-10-06

### Adicionado

- **Agente enxuto do relé (spec 020):** só as ferramentas da entrega, sem skills e sem servidores MCP. O contexto inicial cai de 37 mil para 11 mil tokens, e no mesmo ticket o agente custa 3,2 vezes menos (#253). O relé roda de uma cópia dos scripts, para o agente trocar de branch sem quebrá-lo.
- Modelo por ticket: o label `modelo:NOME` da issue, ou `SDD_AGENT_MODEL`. O custo gravado na issue diz o modelo. O agente não herda mais as variáveis do relé e nunca pode afrouxar um teste para o CI passar (#254).
- `sdd-report.sh tokens` mostra o contexto da primeira chamada de cada sessão (#255). O pacote do ticket lista os testes que citam os AC dele (#256) e as assinaturas do código em sh, Go, Python, JS/TS e Rust, com a linha. O `sdd-context.sh --task` monta o pacote sem GitHub (#260).
- `scripts/benchmark.sh`: as mesmas tarefas num projeto real, com e sem sdd-kit, medidas sem LLM. Os resultados ficam em `docs/benchmark/` (#258).
- Spec 021 aprovada: o axyn, o sdd-kit dentro do opencode com o modelo que o usuário tiver, inclusive gratuito, entra como Fase 15 do ROADMAP.

### Mudado

- README sem números de custo enquanto o kit não for mais barato que um `claude` comum no benchmark com e sem sdd-kit (#261).

## [1.12.0] - 2026-10-06

### Adicionado

- **Economia medida (spec 019):** a primeira fase feita pelo relé custou em média 403 mil tokens por ticket, contra 6,5 milhões por ticket numa sessão longa (Fase 12). São 16 vezes menos por ticket e 5,4 vezes menos contexto relido por chamada. O gráfico e a tabela por ticket estão no topo dos READMEs, e `sdd-report.sh phase '#ÉPICO' --compare '#OUTRO'` refaz a conta (#245).
- O relé roda de verdade com o `claude -p`: o agente padrão não pede permissão (só edita e roda os comandos da entrega) e começa uma sessão própria, sem herdar o id da sessão que o chamou. Os READMEs ganharam a seção do relé (#241).
- `adopt.sh` e `adopt.ps1`: exemplos de uso no `--help` (#242). O `doc-commands.sh` roda também os blocos `sh` marcados (#243). Os dois foram feitos pelo relé.

### Mudado

- `sdd-report.sh`: só conta como despertar ocioso o turno sem `git commit`, `git push` nem `sdd-pr.sh`; o trabalho que seguiu sozinho depois de um aviso não entra (#244).

## [1.11.0] - 2026-10-06

### Adicionado

- **Relé de sessões curtas (spec 018):** `sdd-relay.sh` trabalha o épico aberto ticket a ticket, sem LLM no meio: abre um agente novo por ticket (`SDD_AGENT_CMD`, padrão `claude -p`), espera o merge pelo `sdd-wait.sh` e segue para o próximo. Não guarda estado: interrompido, retoma do GitHub sem refazer ticket mesclado. Uma trava para o relé se um ticket mesclado voltar aberto (#231).
- `sdd-context.sh '#N'` monta o pacote de uma sessão nova: a issue, só as linhas dos FR/NFR/AC citados, as decisões citadas, os arquivos prováveis com as assinaturas, as armadilhas desses arquivos e os comandos de entrega, com teto em `SDD_CONTEXT_MAX` (#230).
- O relé para e avisa o dono no épico quando o Próximo é "perguntar ao dono", quando o CI do PR fica vermelho duas vezes seguidas (a primeira falha ganha uma sessão de correção) e quando o orçamento `SDD_BUDGET_TOKENS` acaba. O hook `sdd-session-guard.sh` faz a sessão gravar WIP e terminar no teto `SDD_SESSION_MAX_TOKENS`, e o relé abre outra a partir do checkpoint (#232).
- Uma sessão, um ticket: `CLAUDE.md` (kit e template) e skill. O relé grava na issue o custo exato do ticket (só as sessões que abriu para ele), e o `sdd-report.sh phase` mostra a medição de cada ticket e o custo médio com e sem relé (#233).

### Mudado

- `sdd-report.sh ticket`: sem relé, a janela começa no último merge antes do primeiro commit, não mais no primeiro commit, que subestimava o ticket (017 FR-2, #233).

## [1.10.0] - 2026-10-06

### Adicionado

- **Custo medido, sem LLM (spec 017):** `sdd-report.sh tokens [--since DATA]` soma as chamadas das sessões do Claude Code (cada uma uma vez, por `message.id`): tokens relidos, gravados e gerados, e o contexto médio e máximo por chamada (#219).
- `sdd-report.sh ticket '#N'` mede o ticket na janela do PR que o fecha, com o tempo e se o CI passou de primeira; `sdd-report.sh phase '#ÉPICO'` junta os tickets numa tabela, num único comentário do épico editado a cada rodada (#220).
- Os despertares sem mensagem do dono (avisos, fim de tarefas em segundo plano) aparecem numa linha e numa coluna próprias, com a contagem e os tokens; o PR de fechamento cita a tabela da fase (#221, #224).

### Corrigido

- O motor fecha o épico da fase entregue antes de abrir a próxima fase (#218).

## [1.9.0] - 2026-10-06

### Adicionado

- **Vigia do merge sem tokens (spec 016):** `sdd-wait.sh merged-any` espera, sem LLM, qualquer PR aberto ser mergeado e acorda a sessão uma única vez. São uma consulta por rodada (60 s por padrão), e o vigia só conclui com o PR de fato fechado, mesmo se a lista falhar (#207). O dono não precisa mais avisar o agente de cada merge.
- O `sdd-pr.sh` termina com o comando do vigia, e o `sdd-release.sh` o herda (#208).
- `CLAUDE.md` (kit e template) e skill: o agente deixa um vigia em segundo plano e, no despertar, faz o Próximo do checkpoint (#211).
- Specs 017 (relatório de entrega e de custo) e 018 (relé de sessões curtas) aprovadas, como Fases 11 e 12 do ROADMAP (#210).

## [1.8.0] - 2026-10-06

### Adicionado

- A versão é sempre a do go-release-manager, calculada pelos commits: as fases do ROADMAP não fixam versão, e o fechamento (`sdd-mark.sh close`) acha a fase pela tarefa aberta e grava `→ vX.Y.Z` no cabeçalho dela como registro. A adoção não numera mais as fases a partir da última tag, e uma versão forçada (`release-as`) só vale a pedido explícito do dono (#199).

## [1.7.3] - 2026-10-06

### Corrigido

- Retomada com uma regra só: a saída do `sdd-resume.sh` sempre termina numa linha "Próximo:" (do checkpoint, da fase já aprovada no ROADMAP ou "perguntar ao dono"), e a sessão faz esse Próximo e nada além. Uma sessão nova não escolhe mais sozinha uma fase não aprovada (#196).
- A conversa segue o idioma em que o dono escreve; uma mensagem curta como "continue" não define o idioma (skill e `CLAUDE.md` do template, #193).

## [1.7.2] - 2026-10-06

### Corrigido

- `sdd-ci.sh` não informa mais como falha um check que ainda está rodando (#190).
- Sessão nova depois do fechamento de uma fase: o `sdd-resume.sh` aponta a próxima fase do ROADMAP, e o motor abre o épico dela no merge do PR de fechamento; "continue" não termina mais em "nada a fazer" (#192).

## [1.7.1] - 2026-10-06

Correções achadas no primeiro ciclo completo do motor num projeto adotado (sdd-kit-demo, da fase 2 à release v0.2.0).

### Corrigido

- O CI do push na `main` não é mais cancelado pelo CI da release: o grupo de concorrência separa por evento, e só PRs cancelam a rodada anterior. Vale no kit e nos seis templates (#179).
- O `make ci` não deixa mais artefato de cobertura solto na árvore (`coverage.out`, `coverage/`, `.coverage`): os Makefiles gravam esses artefatos em `.git/sdd-out`, que o git não versiona. O template .NET deixou de montar o caminho antes de a variável existir, o que gerava `rm -rf /coverage` (#181).
- O motor abre o PR de fechamento mesmo num runner sem as ferramentas do projeto: quem verifica é o CI do PR, que o motor dispara na branch (spec 015 FR-4, #183).
- Antes da tag, o motor espera o CI da `main` verde no commit da release, em vez de rodar `make ci` num runner sem as ferramentas (spec 015 FR-5, #185).
- O `sdd-release.sh` lista os PRs com paginação explícita: o `--paginate` do gh seguia links que alguns proxies recusam, a partir de 100 PRs fechados (#187).

## [1.7.0] - 2026-10-06

Correções achadas ao usar a v1.6.0 de verdade, no próprio kit e no sdd-kit-demo.

### Adicionado

- **Válvula do cache do Go (`scripts/sdd-cover-guard.sh`):** o `make test` do template Go confere o perfil de cobertura antes do gate. Blocos do mesmo arquivo que se sobrepõem, ou que passam da última linha, indicam um cache quente com versões misturadas. Nesse caso a válvula refaz o `go test` num `GOCACHE` frio e descartável e avisa numa linha. `SDD_COVER_GUARD=off` desliga (spec 010 FR-1, #173).

### Corrigido

- A cobertura do template Go saía menor que a real com o cache quente e um `package main` no `-coverpkg` (Go 1.25+): o `go test` passa a rodar com `-count=1`. É a causa do "76,7%" visto no demo (spec 010 FR-1, #173).
- O `sdd-sync` não perde mais a sincronização quando a versão nova traz workflows, que o `GITHUB_TOKEN` não pode gravar. Com o segredo `SDD_SYNC_TOKEN`, aplica tudo; sem ele, aplica o resto e lista os workflows pendentes, que voltam a aparecer até serem aplicados (`sdd-sync.sh --only-workflows`). Sem permissão para criar o PR, o job deixa a branch e o link, e termina com um aviso em vez de falhar (spec 005, #171).
- O motor dispara o CI depois de trazer a `main` para um PR: o push do `GITHUB_TOKEN` não dispara o `pull_request`, e o PR ficava sem nenhum check (spec 015 FR-6, #175).
- O `sdd-pr.sh` aceita as branches de sincronização do kit (`chore/sync-*`), como já aceitava as de fechamento (#176).

## [1.6.0] - 2026-10-06

Fase 9 do [ROADMAP](specs/ROADMAP.md): motor orientado a eventos (spec 015). Workflows do GitHub Actions reagem aos eventos de PR e de CI sem LLM; a sessão do agente não espera nem acompanha eventos e só volta para o que pede julgamento.

### Adicionado

- **`scripts/sdd-wait.sh`:** espera um PR mergeado, um CI concluído ou uma issue fechada, sem LLM, com intervalo e tempo máximo configuráveis (spec 015 FR-1, #158).
- **Checkpoint no merge (`sdd-on-merge.yml`):** o merge de um PR de ticket atualiza o checkpoint do épico com o PR feito e o próximo ticket aberto (FR-2, #162).
- **Resumo do CI vermelho (`sdd-ci-summary.yml`):** um único comentário no PR, com uma linha por check e só o fim do log das falhas, editado a cada rodada (FR-3, #160).
- **Fase concluída (`sdd-on-phase-done.yml`):** quando o último ticket do épico fecha, o PR de fechamento é aberto com o `sdd-release.sh`. O script também reaproveita uma branch de release que já existe (FR-4, #164).
- **Release no merge do fechamento (`sdd-on-release-merge.yml`):** o merge do PR `chore/release-vX.Y.Z` dispara o *Release tag* uma única vez (FR-5, #163).
- **`main` levada aos PRs (`sdd-update-prs.yml`):** quando a `main` muda, os PRs abertos que estão atrás dela são atualizados, e os que têm conflito recebem um único aviso (FR-6, #161).
- O `CLAUDE.md` do template manda o agente abrir o PR, gravar o checkpoint e encerrar, sem esperar eventos; o README documenta o motor (FR-7, #165).
- **O próprio sdd-kit roda o motor:** `make self-sync` gera os workflows do kit a partir do template, sem cópias editadas à mão, e o `make ci` falha se elas divergirem. A partir desta versão, o fechamento e a release do kit também saem pelo motor (FR-8, #168).

### Corrigido

- O `sdd-pr.sh` dá ao PR o título do primeiro commit da branch, e não do último; e, numa branch com o número do épico aberto, escreve `Refs` em vez de `Closes`, para o merge não fechar o épico (#159).

### Configuração

- Para o motor abrir PRs (fechamento da fase) e para o `sdd-sync` abrir o PR de atualização, ative em *Settings → Actions → General → Workflow permissions* a opção "Allow GitHub Actions to create and approve pull requests". Opcional: o segredo `SDD_ENGINE_TOKEN` faz o CI rodar nos pushes do motor; a variável `SDD_ENGINE=off` desliga tudo.

## [1.5.0] - 2026-10-05

Scripts que tiram do agente os passos mecânicos de retomar, entregar, preparar o ambiente e fechar a versão, para que a LLM gaste tokens e contexto só com código e decisões.

### Adicionado

- **`scripts/sdd-resume.sh`:** a retomada da sessão num comando. Mostra o checkpoint do épico aberto e entra na branch dele (só com a árvore limpa e sem commits não enviados). Depois lista os PRs abertos com o resumo do CI e as sub-issues abertas do épico. Os hooks de início de sessão passam a chamá-lo (spec 014 FR-2, #139).
- **`scripts/sdd-pr.sh`:** a entrega do ticket num comando. Faz merge da `main` e roda o `make ci`, mostrando só o fim do log em falha. Depois faz push e abre o PR com o template preenchido, ou reaproveita o que já existe. Por fim espera o CI e grava o checkpoint (#138).
- **`scripts/sdd-doctor.sh`:** deixa o ambiente local igual ao do CI quando o hook de sessão não rodou. Cobre as ferramentas nas versões fixadas, o `covdata` do Go, um `golangci-lint` antigo escondendo o certo no PATH e o locale UTF-8. No template Go, o `make lint` passa a preferir o `golangci-lint` do `GOPATH/bin` (spec 010, #144).
- **`scripts/sdd-release.sh`:** o fechamento da versão num comando. A versão vem do [go-release-manager](https://github.com/fabiodrneles/go-release-manager) pelos Conventional Commits, e um X.Y.Z passado à mão só a força, com aviso. O script monta o rascunho do CHANGELOG pelos PRs mesclados, sobe a versão nos arquivos de `.sdd-release` e abre o PR. Depois do merge, `--tag` dispara o *Release tag* (spec 008, #145, #155).

### Corrigido

- O `sdd-pr.sh` espera o CI pelo SHA enviado: logo depois do push, `#PR` ainda podia ler o head anterior e dar um falso verde (#141).

## [1.4.0] - 2026-10-03

Fase 8 do [ROADMAP](specs/ROADMAP.md): checkpoints de retomada (spec 014).

### Adicionado

- **Checkpoints de retomada:** `scripts/sdd-checkpoint.sh save` mantém um comentário de checkpoint no épico aberto (branch, commit, PRs, feito, próximo e bloqueio). O hook de início de sessão o mostra, o hook `Stop` atualiza o estado do git a cada resposta, e o `CLAUDE.md` manda retomar dele e gravar um a cada passo. A sessão pode acabar sem aviso, e a próxima continua sozinha (spec 014, #123).

## [1.3.1] - 2026-10-03

Correções achadas ao sincronizar o [sdd-kit-demo](https://github.com/fabiodrneles/sdd-kit-demo) da `v0.1.0` para a `v1.3.0`.

### Corrigido

- O `sdd-sync.sh` roda uma cópia de si mesmo: a sincronização o atualizava enquanto ele rodava, e o `sh` terminava com `Syntax error` antes de abrir o PR. Quem adotou até a `v1.3.0` roda a cópia uma vez à mão (veja "Atualização automática" no README) (spec 005 FR-1).
- A cobertura do template Go usa `-coverpkg=./...`: um pacote testado só pelos testes de outro contava 0% (no demo, 49% em vez de 88%) (spec 010 FR-1).

## [1.3.0] - 2026-10-02

Fase 7 do [ROADMAP](specs/ROADMAP.md): primeira impressão para quem chega pela divulgação (spec 013).

### Adicionado

- **Demonstração animada no topo do README**, gerada da saída real da adoção e do `make ci` por `docs/demo/demo.sh` (spec 013 FR-2, #109).
- **Perguntas e relatos de uso nas Discussions**, pelo formulário de nova issue; seção "Primeira contribuição" no CONTRIBUTING e issues `good first issue` (spec 013 FR-3, #110).

### Mudado

- "Começar num repositório novo" (pt e en) explica pré-requisitos, o que a adoção copia, a garantia de não sobrescrever, `--dry-run` e `--skeleton`; os comandos do guia rodam no CI do kit (spec 013 FR-1, #108).

## [1.2.0] - 2026-10-02

Fase 6 do [ROADMAP](specs/ROADMAP.md): arquivos do projeto fora da sincronização (spec 012).

### Mudado

- **`specs/` e `CHANGELOG.md` são do projeto:** passam a `template/seed/`, e a adoção os cria quando faltam, nunca os sobrescreve (nem com `--force`) e não os registra no `.sdd-kit.json`. O `sdd-sync` gerencia só o que a adoção da versão alvo registra, então uma mudança do kit nesses modelos não substitui mais o ROADMAP ou o CHANGELOG preenchido. Quem adotou antes tem esses arquivos retirados do estado na segunda sincronização (spec 012 FR-1, FR-2, #100).

### Adicionado

- O CI do kit falha quando a versão mais recente do CHANGELOG não é a do `plugin.json`, o que teria barrado no PR o incidente da `v1.0.0` (spec 012 FR-3, #101).

## [1.1.0] - 2026-10-02

Fase 5 do [ROADMAP](specs/ROADMAP.md): a adoção ajustada pelo uso real (achados do #51, spec 011).

### Adicionado

- **Avisos da adoção Node:** `package-lock.json` ausente ou fora de sincronia com o `package.json` (o `npm ci` do CI falharia) e projeto sem script de teste, sem falhar a adoção (spec 011 FR-1, #86).
- **`scripts/doc-commands.sh`** no template comum: o CI de docs roda os blocos ```` ```bash ```` do README marcados com `<!-- doc-commands -->` (spec 011 FR-3, #88).
- **`make linkcheck`** em toda linguagem, com o lychee (spec 011 FR-3, #88).
- Aviso de `LICENSE` ausente na adoção; a licença continua decisão do dono (spec 011 FR-3, #88).
- **`CHANGELOG.md`** com `[Unreleased]` criado pela adoção, que o `sdd-mark close` exige (spec 011 FR-5, #90).
- **Acessibilidade no template Node:** com `A11Y_PAGES`, o `make ci` roda o axe-core (num DOM do jsdom, sem contraste de cor) e falha em qualquer violação (spec 011 FR-6, #91).

### Mudado

- A adoção numera as fases do ROADMAP criado a partir da última tag `vX.Y.Z` do repositório (ex.: `v1.4.0` → Fase 1 na `v1.5.0`) (spec 011 FR-4, #89).

### Corrigido

- O hook de sessão Go compila o `covdata` que falta na toolchain baixada pelo `GOTOOLCHAIN`; sem ele, `go test -coverprofile ./...` falhava num pacote sem testes (spec 011 FR-2, #87).

## [1.0.0] - 2026-10-02

Fim da Fase 3 do [ROADMAP](specs/ROADMAP.md) (profissional e pronto para a comunidade). O `sdd-check --strict`, os arquivos de comunidade, o estudo de caso, o plugin `sdd-release` e os scripts dos passos mecânicos já saíram na 0.2.0.

### Adicionado

- **Template Rust**: `make ci` com `cargo fmt`, clippy e cobertura do `cargo llvm-cov`, CI, hook, dependabot, esqueleto e e2e (spec 007 FR-4, #48).
- **Template C#/.NET**: `make ci` com `dotnet format`, build sem avisos e cobertura do coverlet, CI, hook, dependabot, esqueleto e e2e (spec 007 FR-4, #49).
- **Regras de economia de uso** na skill, no `CLAUDE.md` do template e no do kit (spec 007 AC-6, #78).

### Mudado

- A skill e os comandos dos plugins estão em inglês; specs, issues e PRs continuam escritos no idioma do dono (spec 007 AC-5, D12, #50).
- Skill revisada com o uso em outros repositórios: numeração pela última tag, CI verde primeiro em repositório existente, PRs do Dependabot, lições de toolchain e lockfile (spec 007 FR-5, #51).

### Corrigido

- O esqueleto e a fixture .NET saem sem BOM UTF-8 (#49).
- A adoção via `curl` e o README baixam o template da versão atual, e não mais da `v0.1.0`, que não tinha Rust, .NET nem `--skeleton`.

## [0.2.0] - 2026-10-02

Fase 4 do [ROADMAP](specs/ROADMAP.md) (esteira de qualidade) e as entregas da Fase 3 já mergeadas; a v1.0.0 fica para o fim da Fase 3.

### Adicionado

- **Cobertura mínima no `make ci`** de todos os templates: falha abaixo de `COVERAGE_MIN` (80 por padrão), com Go `coverprofile`, c8 no Node, JaCoCo no Java e pytest-cov no Python (spec 010 FR-1, #68).
- **Workflow Release tag** em todos os templates: cria a tag (go-release-manager ou `release-as`, no commit `ref`) e publica a release com notas geradas (spec 010 FR-4, #70).
- **Gradle** no template Java, além de Maven, com JaCoCo por init script (spec 010 FR-3, #71).
- **`adopt --skeleton`** (sh e PowerShell): projeto mínimo com um teste num repositório vazio, para o CI nascer verde (spec 010 FR-2, #72).
- **Plugin `sdd-release`**: próxima versão pelo go-release-manager, comparada com a versão do ROADMAP (spec 008, #54, #58).
- **Scripts dos passos mecânicos**: `sdd-ci`, `sdd-mark`, `sdd-phase-status`, `sdd-epic` e `sdd-release-check` (spec 009, #56).
- `sdd-check --strict` no CI do kit e do template, arquivos de comunidade e estudo de caso do cv-craft (spec 007, #45, #46, #47).

### Mudado

- **Incompatível:** adotantes abaixo de 80% de cobertura passam a ter o `make ci` vermelho depois da sincronização; baixe `COVERAGE_MIN` no `Makefile` para manter o comportamento anterior (#68).
- O check de links publica cada link quebrado como anotação e aceita 503 como recusa temporária, como o 429 (#69).

### Corrigido

- `sdd-ci.sh` espera e reporta também os status de commit (ex.: Vercel) (spec 009 AC-7, #67).

## [0.1.0] - 2026-10-01

Primeira versão pública: Fases 1 e 2 do [ROADMAP](specs/ROADMAP.md).

### Adicionado

- Skill `sdd-delivery` como plugin do Claude Code (`/plugin marketplace add fabiodrneles/sdd-kit`) e zip para o claude.ai, publicado com SHA-256 na release (spec 002).
- Comandos `/sdd-analyze`, `/sdd-specs`, `/sdd-epic`, `/sdd-next`, `/sdd-status` e `/sdd-close` (spec 006 FR-3).
- `template/` com `CLAUDE.md`, `AGENTS.md`, `CONTRIBUTING.md`, templates de issue e PR, esqueleto de `specs/`, hook de sessão, plugin habilitado e CI pronto para Go, Node/TS, Java e Python (spec 003).
- Script de adoção em sh e PowerShell, para repositórios novos e existentes, sem sobrescrever nada, idempotente e com `--dry-run` (spec 004).
- `.sdd-kit.json` e sincronização semanal por PR, com conflitos listados para revisão (spec 005).
- `sdd-check`: rastreabilidade AC → teste, status das specs e IDs do ROADMAP (spec 006 FR-2).
- Critérios de aceite em EARS e seção "Mudanças" por spec, que alimenta o CHANGELOG (spec 006 FR-1, FR-4).
- CI do kit em Linux, macOS e Windows, com e2e dos templates nas quatro linguagens.

[Unreleased]: https://github.com/fabiodrneles/sdd-kit/compare/v1.4.0...HEAD
[1.4.0]: https://github.com/fabiodrneles/sdd-kit/compare/v1.3.1...v1.4.0
[1.3.1]: https://github.com/fabiodrneles/sdd-kit/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/fabiodrneles/sdd-kit/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/fabiodrneles/sdd-kit/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/fabiodrneles/sdd-kit/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/fabiodrneles/sdd-kit/compare/v0.2.0...v1.0.0
[0.2.0]: https://github.com/fabiodrneles/sdd-kit/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/fabiodrneles/sdd-kit/releases/tag/v0.1.0
