# Roadmap e tarefas

Cada tarefa referencia a spec e os critérios de aceite que ela fecha. Ordem sugerida = ordem da lista.

## Fase 0 — Decisões (antes de codar)

- [x] Responder D1–D4 em [ANALYSIS.md §5](ANALYSIS.md#5-decisões-da-fase-0-respondidas-pelo-dono-em-2026-10-01) e mover specs para `Approved`.

## Fase 1 — Kit utilizável → `v0.1.0` · épico #1

- [x] **T1** Base de specs: constituição, specs, ROADMAP — #2
- [x] **T2** Base do repositório: LICENSE, READMEs, CLAUDE.md, hook, CI de docs e shellcheck — 001 FR-1..5, AC-1..3 — #3
- [x] **T3** Skill empacotada como plugin do Claude Code e zip do claude.ai — 002 FR-1..4, FR-6, AC-1..4 — #4
- [x] **T4** `template/` comum e por linguagem (Go, Node/TS, Java, Python) — 003 FR-1..5, AC-1..3 — #5
- [x] **T5** Script de adoção sh + PowerShell com testes — 004 FR-1..8, AC-1..6; 003 AC-4 — #6
- [x] **T6** cv-craft consome a skill do kit — 002 FR-5 — #7

## Fase 2 — Atualização contínua e recursos da comunidade · épico #21 *(antecipada para a `v0.1.0`)*

- [x] **T7** Arquivo de estado `.sdd-kit.json` escrito pela adoção — 005 FR-2 — #27
- [x] **T8** Workflow semanal de sincronização por PR — 005 FR-1, FR-3, FR-4, AC-1..3 — #28
- [x] **T11** Critérios em EARS na skill e no template — 006 FR-1 — #23
- [x] **T12** `sdd-check`: rastreabilidade AC → teste, status e ROADMAP — 006 FR-2, AC-1/2 — #24
- [x] **T13** Comandos de barra no plugin — 006 FR-3, AC-3 — #25
- [x] **T14** Seção "Mudanças" nas specs gerando o CHANGELOG — 006 FR-4 — #26
- [x] **T15** `AGENTS.md` no template — 006 FR-5, AC-4 — #22

Também entraram na `v0.1.0`: checagem da `description` da skill (#13), README com o fluxo completo (#15, #36), template habilitando o plugin (#18) e a spec 006 (#16).

## Fase 3 — Profissional → `v1.0.0` · épico #39

- [x] **T9** Linguagens adicionais (um ticket por linguagem) — 003, 007 FR-4, AC-4 *(depende de D10)*
- [x] **T10** Revisão da skill com o uso em pelo menos dois repositórios além do cv-craft — 007 FR-5 *(depende de D11)*
- [x] **T16** `sdd-check --strict` no CI do kit e do template — 007 FR-1, AC-1 — #41
- [x] **T17** Arquivos de comunidade — 007 FR-2, AC-2 — #42
- [x] **T18** Estudo de caso do cv-craft — 007 FR-3, AC-3 — #43
- [x] **T19** Skill para quem não lê português — 007 FR-6 *(depende de D12)*
- [x] **T20** Plugin `sdd-release`: próxima versão pelo go-release-manager — 008 FR-1 a FR-5, AC-1 a AC-3 — #53
- [x] **T21** Scripts para os passos mecânicos (`sdd-ci`, `sdd-mark`, `sdd-phase-status`, `sdd-epic`, `sdd-release-check`) — 009 FR-1 a FR-8, AC-1 a AC-6 — #55

## Fase 4 — Esteira de qualidade → `v0.2.0` · épico #59 *(antes da v1.0.0: a Fase 3 continua aberta)*

- [x] **T22** Cobertura mínima no `make ci` de cada linguagem — 010 FR-1, AC-1 — #60
- [x] **T23** `--skeleton` na adoção: CI verde num repositório vazio — 010 FR-2, AC-2 — #61
- [x] **T24** Gradle no template Java — 010 FR-3, AC-3 — #62
- [x] **T25** `release-tag.yml` em todos os templates — 010 FR-4, AC-4 — #63
- [x] **T26** Validação no qa-portfolio (Node, sem CI) — 010 FR-5, AC-5 — #64

## Fase 5 — Adoção ajustada pelo uso real (P2) → `v1.1.0` · épico #85

- [x] **T27** Avisos da adoção Node: lockfile fora de sincronia e projeto sem testes — 011 FR-1, AC-1 — #86
- [x] **T28** Hook de sessão Go com a toolchain completa do `go.mod` — 011 FR-2, AC-2 — #87
- [x] **T29** `doc-commands.sh`, `LICENSE` e `make linkcheck` no template comum — 011 FR-3, AC-3 — #88
- [x] **T30** Fases do ROADMAP numeradas a partir da última tag — 011 FR-4, AC-4 — #89
- [x] **T31** `CHANGELOG.md` com `[Unreleased]` criado pela adoção — 011 FR-5, AC-5 — #90
- [x] **T32** Checagem de acessibilidade com axe no template Node — 011 FR-6, AC-6 — #91

## Fase 6 — Arquivos do projeto fora da sincronização (P1) → `v1.2.0` · épico #99

- [x] **T33** Modelos do projeto em `template/seed/`: criados uma vez, fora do estado e da sincronização — 012 FR-1, FR-2, AC-1, AC-2 — #100
- [x] **T34** CI do kit compara a versão do CHANGELOG com a do `plugin.json` — 012 FR-3, AC-3 — #101

## Fase 7 — Primeira impressão (P1) → `v1.3.0` · épico #107

- [x] **T35** "Comece em 5 minutos" no README, com o comando rodando no CI — 013 FR-1, AC-1 — #108
- [x] **T36** Demonstração animada no topo do README, gerada da saída real — 013 FR-2, AC-2 — #109
- [x] **T37** Perguntas nas Discussions e issues `good first issue` — 013 FR-3, AC-3 — #110

## Fase 8 — Checkpoints de retomada (P1) → `v1.4.0` · épico #122

- [x] **T38** `sdd-checkpoint.sh` e os hooks de início de sessão e `Stop` — 014 FR-1 a FR-4, AC-1 a AC-4 — #123

## Fase 9 — Motor orientado a eventos (P1) → `v1.6.0` · épico #146

- [x] **T39** `sdd-wait.sh`: espera sem LLM por merge, CI ou issue — 015 FR-1, AC-1 — #147
- [x] **T40** Workflow de PR mergeado: checkpoint do épico atualizado — 015 FR-2, NFR-1, NFR-2, AC-2, AC-7 — #148
- [x] **T41** Workflow de CI vermelho: comentário único de resumo no PR — 015 FR-3, NFR-1, NFR-2, AC-3 — #149
- [x] **T42** Workflow de fase concluída: PR de fechamento com `sdd-release.sh` — 015 FR-4, NFR-1, NFR-2, AC-4 — #150
- [x] **T43** Workflow de fechamento mergeado: release disparada — 015 FR-5, NFR-1, NFR-2, AC-5 — #151
- [x] **T44** Workflow de `main` alterada: PRs atualizados ou avisados do conflito — 015 FR-6, NFR-1, NFR-2, AC-6 — #152
- [x] **T45** `CLAUDE.md` e skill: o agente não espera eventos, o motor faz — 015 FR-7 — #153
- [x] **T46** O sdd-kit roda o próprio motor (`make self-sync` e checagem de divergência) — 015 FR-8, AC-8 — #167

## Fase 10 — Merge percebido sem tokens (P1) → `v1.9.0`

- [x] **T47** `sdd-wait.sh merged-any`: vigia sem LLM do merge de qualquer PR aberto — 016 FR-1, FR-2, NFR-1, AC-1, AC-2, AC-3
- [x] **T48** `sdd-pr.sh` e `sdd-release.sh` imprimem o comando do vigia — 016 FR-3, AC-4
- [x] **T49** `CLAUDE.md` (kit e template) e skill: um vigia em segundo plano, sem assinar eventos de PR — 016 FR-4, FR-5, NFR-2, AC-5

## Fase 11 — Relatório de entrega e de custo (P1)

- [x] **T50** `sdd-report.sh tokens`: totais de uma sessão sem LLM — 017 FR-1, NFR-1, NFR-2, AC-1, AC-5
- [x] **T51** `sdd-report.sh ticket` e `phase`: custo por ticket e comentário único no épico — 017 FR-2, FR-3, AC-2, AC-3
- [x] **T52** Despertares sem mensagem do dono separados; total no PR de fechamento — 017 FR-4, FR-5, AC-4

## Fase 12 — Relé de sessões curtas (P1)

- [x] **T53** `sdd-context.sh`: pacote de contexto do ticket com teto de tamanho — 018 FR-1, AC-1
- [x] **T54** `sdd-relay.sh`: um agente novo por ticket, vigia e próximo ticket — 018 FR-2, NFR-1, AC-2, AC-4
- [x] **T55** Paradas do relé (dono, CI vermelho duas vezes, orçamento) e teto de contexto por sessão — 018 FR-3, FR-4, AC-3
- [x] **T56** `CLAUDE.md` e skill: uma sessão, um ticket; custo por ticket no épico, com e sem relé — 018 FR-5, FR-6, NFR-2, AC-5

## Fase 13 — Relé na prática (P1)

- [x] **T57** Agente padrão do relé sem prompts de permissão e com sessão própria; README de como rodar o relé — 019 FR-1, FR-2, AC-1
- [x] **T58** `adopt.sh` e `adopt.ps1`: exemplos de uso no `--help` (#115), feito pelo relé
- [x] **T59** `doc-commands.sh`: rodar também os blocos `sh` marcados (#114), feito pelo relé
- [x] **T60** Despertar ocioso: só o turno sem commit nem push — 019 FR-3, AC-2
- [x] **T61** Comparação com a Fase 12 e a economia no README — 019 FR-4, AC-3

## Fase 14 — Agente enxuto do relé (P1)

- [x] **T62** Agente padrão do relé só com as ferramentas da entrega, sem skills e sem MCP; `SDD_AGENT_TOOLS` — 020 FR-1, AC-1
- [x] **T63** `sdd-report.sh`: contexto inicial de cada sessão, feito pelo relé — 020 FR-2, AC-2
- [x] **T64** `sdd-context.sh`: testes que citam os AC do ticket no pacote, feito pelo relé — 020 FR-3, AC-3
- [x] **T66** Modelo por ticket: label `modelo:NOME` ou `SDD_AGENT_MODEL`, e o modelo no custo gravado — 020 FR-5, AC-5
- [x] **T67** Benchmark com e sem sdd-kit no sdd-kit-demo (`scripts/benchmark.sh`) — 020 FR-6, AC-6
- [x] **T68** Pacote com as assinaturas do código (Go, Python, JS/TS, Rust) e `sdd-context.sh --task` — 020 FR-7, AC-7
- [x] **T65** Comparação com as Fases 13 e 12 e o gráfico do README atualizado — 020 FR-4, AC-4

## Fase 15 — axyn mínimo (P1)

O motor em Go como cérebro do opencode, com o modelo que o usuário tiver (spec 021, aprovada em 2026-10-06).

- [x] **T70** `axyn/`: esqueleto em Go, `axyn version`, `make ci` e CI do kit cobrindo o Go — 021 FR-1
- [x] **T71** `axyn gate`: CI do projeto, teste afrouxado, caminhos protegidos e limite de diff — 021 FR-5, AC-1
- [x] **T72** `axyn mcp`: `axyn_plan`, `axyn_next`, `axyn_gate` e `axyn_ship` — 021 FR-3, AC-1
- [x] **T73** `axyn install`: servidor MCP, agentes e o comando `/axyn` no opencode, sem apagar o que existe — 021 FR-2, AC-3
- [x] **T74** Template `web` e preparo de repositório vazio — 021 FR-4, AC-4
- [x] **T75** Escada de modelos e registro do modelo, das tentativas e do custo — 021 FR-6, FR-7, AC-2
- [x] **T77** Recuperação sem trocar de modelo: diagnóstico exato, mais contexto, várias tentativas, ticket dividido, plano antes do código e pergunta ao usuário — 021 FR-8, AC-6
- [x] **T78** O motor conduz o laço: /axyn chama só axyn_run, e o axyn usa o opencode run para o código — 021 FR-9, AC-7
- [ ] **T76** Instalação por um comando e o teste de aceitação do dono (landing page com modelo gratuito; e2e com agente falso no CI) — 021 FR-1, AC-5
  - Falta o teste de aceitação do dono (AC-5, primeira parte): `/axyn crie uma landing page` no opencode com modelo gratuito. O e2e com agente falso já roda no CI.
- [x] **T79** axyn sem Go: binários na release, instalação num comando e guia do usuário — 021 FR-1, AC-5

## Próximas fases (visão, sem versão)

Ideias aprovadas pelo dono em 2026-10-03, em ordem de impacto. Cada uma vira spec e fase quando chegar a vez.

1. **Independente do agente, com prova:** o mesmo fluxo validado de ponta a ponta com Codex, Cursor e Gemini CLI, com e2e no CI.
2. **Métricas de entrega (`sdd-report`):** por fase, tempo por ticket, PRs verdes de primeira, ACs cobertos e custo em tokens por ticket.
3. **Motor em Go (`sdd`):** um binário único no lugar dos scripts sh/PowerShell duplicados, sem depender de jq; base do orquestrador abaixo.
4. **Tickets em paralelo:** o motor lê o grafo de dependências do épico e roda um agente por ticket independente, cada um no seu worktree e branch, com checkpoint próprio.
5. **Revisor contra a spec:** um agente confere cada PR contra os critérios de aceite antes do dono.
6. **Desvio entre spec e código:** aviso quando o código muda e a spec não acompanha.
7. **Modo autônomo com rédea:** rotinas agendadas puxam o próximo ticket; o dono aprova por label ou comentário.
8. **Orçamento por fase:** limite de tokens ou de tempo; ao chegar perto, checkpoint e pausa.
9. **`sdd doctor`:** diagnóstico da adoção (hooks, versões, estado, labels, permissões do workflow) com a correção sugerida.
10. **Alcance:** GitLab e Bitbucket, mais linguagens pela comunidade, specs a partir de código legado.

### Motor em Go: comandos que trocam dezenas de passos da LLM por uma linha

Levantados no uso real (sessão de 2026-10-02/03): cada um substitui uma sequência mecânica que hoje o agente faz chamada a chamada. O motor também pode rodar como servidor MCP, com respostas curtas e estruturadas.

- **11 · `sdd ship`:** commit, push, PR com o corpo gerado do ticket, da spec e do diff, espera do CI, merge quando verde e delegado, e checkpoint. O maior ganho: hoje são 6 a 8 chamadas por PR.
- **12 · `sdd release vX`:** fechamento completo (ROADMAP, CHANGELOG gerado das seções "Mudanças", versão em todos os arquivos, artefatos, release e épico fechado).
- **13 · `sdd sync` com regiões do projeto:** marcas no template para o merge de três vias preservar o que é do projeto (ex.: descrição e armadilhas no `CLAUDE.md`), sem resolver conflito à mão.
- **14 · `sdd ci why`:** só o erro e o contexto mínimo de um job vermelho, em vez do log inteiro.
- **15 · `sdd context`:** início de sessão num resumo (checkpoint, PRs e CI, issues da fase, próximo passo), com as consultas em paralelo.
- **16 · `sdd mutate`:** checagem de mutação automática de um teste novo (quebra, roda, restaura, relata).
- **17 · `sdd plan`:** spec, fase no ROADMAP, índice e épico a partir de um YAML curto, com os números das issues anotados de volta.

### Arquitetura do motor `sdd` (decisão de visão)

Sem painel web, o motor é um **monólito modular local-first**: um binário Go por plataforma, sem servidor nem banco de dados.

- **Monólito modular, distribuído como CLI:** um binário com pacotes independentes (`ship`, `release`, `sync`, `orch`, `mcp`...), cada um com testes próprios; um bug num pacote se corrige e se testa só nele.
- **Portas e adaptadores (hexagonal):** o núcleo (specs, ACs, fases, tickets) fala com interfaces; adaptadores ligam GitHub, GitLab, Gitea e os agentes (Claude Code, Codex, Cursor, Gemini CLI). Base dos itens 1 e 31.
- **Supervisor com um processo por ticket:** no orquestrador (item 4), cada agente roda como processo do sistema no seu worktree; goroutines com `recover`; falha de um ticket é registrada no épico e no checkpoint e não para os outros (`sdd run --ticket N` retoma só ele). Isolamento de falhas como requisito.
- **Comandos isolados e idempotentes:** cada comando é um processo que começa e termina; cada passo é retomável a partir do estado no GitHub e no checkpoint, sem estado em memória.
- **Estado local-first no Git e no GitHub:** o repositório (specs, ROADMAP) e a forja (épicos, issues, PRs, checkpoint) são a fonte da verdade.
- **Três modos do mesmo binário:** CLI (`sdd ship`), servidor MCP local por stdio (`sdd mcp`) e orquestrador (`sdd run --epic N`).
- **Microserviços só se houver painel web ou modo para times (pós-v2.0),** como serviço opcional; o núcleo segue local.
- **Também vão para o motor:** contexto por ticket (20), spec viva e desvio (18, 6), evidência por PR (21), guardas do PR (26–29), importador (39), benchmark (40), orçamento por fase (8), `sdd report` (2) e `sdd doctor` (9).

Resumo para o README: *"The sdd engine is a local-first modular monolith in Go, using ports and adapters for forges and agents, and a supervisor that runs one isolated process per ticket; all state lives in Git and GitHub."*

### Contra as dores comuns das ferramentas de SDD

- **18 · Spec viva:** um PR que muda comportamento sem linha em "Mudanças" da spec reprova no CI; specs que não envelhecem.
- **19 · Modo leve:** caminho rápido para bug e mudança pequena (ticket com teste, sem spec nova), porque cerimônia demais para tarefa pequena afasta quem adota.
- **20 · Contexto por ticket:** o motor monta só o pedaço de spec, código e decisões de que o ticket precisa, em vez de o agente ler specs inteiras.
- **21 · Evidência por PR:** um resumo automático no PR mostra cada critério de aceite e o teste que o prova, para o dono revisar em segundos.
- **22 · Times:** mais de um dono e revisor, atribuição de tickets e regras de quem aprova o quê.
- **23 · Gate de segurança:** segredos, dependências vulneráveis e licenças no `make ci` do template.
- **24 · Projeto legado (brownfield):** specs geradas por engenharia reversa do código e dos testes existentes, porque quase toda ferramenta de SDD assume projeto do zero.
- **25 · Spec ambígua:** um verificador aponta termos vagos, AC sem critério mensurável e conflitos entre specs, e gera perguntas ao dono antes de codar.
- **26 · Dependência alucinada:** o CI reprova pacote inexistente, recém-publicado ou com nome parecido com um conhecido (slopsquatting) antes da instalação.
- **27 · Teste de fachada:** detecta teste sem asserção, que só repete o mock ou que passa com o código quebrado, ligado ao `sdd mutate`.
- **28 · Escopo do ticket:** alerta quando o PR mexe em arquivos ou áreas fora do que o ticket e a spec citam (o agente que "aproveita" para refatorar).
- **29 · Decisão contrariada:** o diff é conferido contra as decisões D1..Dn e a constituição; contrariar uma exige registrar a nova decisão.
- **30 · Monorepo:** várias linguagens e pacotes num repositório, com CI, cobertura e specs por pacote.
- **31 · Além do GitHub:** GitLab, Bitbucket e Gitea (issues, PRs e CI), porque o kit hoje depende do GitHub.
- **32 · Desfazer:** reverter um ticket ou uma fase inteira de forma limpa (código, status, CHANGELOG e issues).
- **33 · Visão para humanos:** um resumo de arquitetura e estado sempre atual, gerado das specs e do código, para quem entra no projeto sem ler tudo.

### Comunidade e alcance de pessoas

Funcionalidade não basta: o peso de um projeto aberto vem de quem o usa.

- **34 · Inglês e francês de primeira classe:** `README.en.md` e `README.fr.md`, guia, exemplos e documentação em inglês e francês no mesmo nível do português, com seletor de idioma no topo de cada README e verificação no CI (seções e links equivalentes nos três idiomas). O francês abre o mercado de Quebec e do Canadá francófono.
- **35 · Casos de adoção:** histórias documentadas de projetos reais (antes e depois, números), começando pelo `sdd-kit-demo`.
- **36 · Contribuição fácil:** `CONTRIBUTING`, issues `good first issue`, guia para adicionar uma linguagem e resposta rápida a quem chega.
- **37 · Decisões explicadas:** artigos curtos em inglês sobre o porquê de cada escolha (checkpoint no épico, sh POSIX, sync de três vias).
- **38 · Divulgação contínua:** cada release com nota e demonstração curtas nas comunidades de Claude Code e de agentes.

### Posicionamento: "os outros escrevem a spec; o sdd-kit entrega e prova"

Concorrentes (Spec Kit, Kiro, BMAD, OpenSpec, Agent OS) param na spec; as plataformas absorvem o genérico. O diferencial é a garantia por código (CI), o estado no GitHub e o ciclo inteiro até a release.

- **39 · Importar specs de concorrentes:** ler specs do Spec Kit, Kiro e OpenSpec e entregá-las com o nosso fluxo; quem usa outra ferramenta adota sem trocar nada.
- **40 · Benchmark público:** a mesma tarefa com cada ferramenta, medindo PRs verdes de primeira, ACs com teste, retrabalho e tokens; resultados e roteiro abertos e reprodutíveis.
- **41 · Nicho primeiro:** devs solo e times pequenos com Claude Code; guias, exemplos e divulgação focados nesse público antes de expandir.

### Rumo ao Archon (visão do dono, 2026-10-05)

O sdd-kit amadurece e vira o **Archon**: um harness que entrega aplicações no GitHub como um time de TI, gastando o mínimo da LLM cara. Princípio: **cada papel com a peça mais barata que o faz bem**, e a LLM grande só onde nada mais resolve. Ordem, do mais barato e garantido ao mais caro e incerto:

- **42 · Contexto determinístico primeiro (custo zero):** mapa do repositório com tree-sitter (assinaturas e tipos, sem corpos), busca lexical (BM25/ctags) e o grafo de quem chama quem montam o contexto do ticket; amplia o item 20. Medir tokens antes e depois.
- **43 · Modelo por papel, configurável:** roteador, planejador, executor, verificador e redator como papéis do motor (portas do item 3); cada um aponta para um modelo local, um free tier (Groq, OpenRouter, Gemini) ou a LLM principal, com queda automática para o local quando a cota acaba. Painel no terminal (`sdd models`) para escolher; sem escolha, valem os padrões.
- **44 · Classificadores pequenos onde a decisão é fechada:** spec ambígua (25), ticket pronto ou não, tipo da mudança (bug, feature, refatoração, para o modo leve do 19); modelos de classificação (família BERT) treinados com os dados abaixo, rodando em CPU.
- **45 · Dados do próprio uso (flywheel):** cada spec, ticket, PR e revisão aprovada vira exemplo rotulado; geração sintética só como complemento, validada por regra antes de entrar no conjunto.
- **46 · Laboratório de modelos (`archon-lab`, repositório separado):** gerar dados, ajustar (LoRA/QLoRA no Colab ou Kaggle), medir contra um conjunto fixo e exportar (GGUF, Hugging Face Hub), num comando; nenhum modelo entra no motor sem vencer a linha de base no benchmark.
- **47 · Piso de hardware medido:** o motor e os classificadores rodam numa máquina modesta (i3 de 4ª geração, 4 GB); um SLM generativo local de 1B a 1,5B quantizado é opcional e lento nessa máquina, e o free tier é o caminho padrão para os papéis generativos. Números de latência e memória publicados, não estimados.
- **48 · Prova antes do discurso:** toda alegação de economia (tokens, custo, acerto) sai do benchmark do item 40, com a mesma tarefa feita com e sem o roteamento; o artigo sobre a arquitetura usa só números medidos.
- **49 · Laya como adaptador de decisão (candidato, entra só se vencer):** o [Laya](https://github.com/NandhaKishorM/laya) (Apache-2.0, decisões tipadas sem gerar texto, confiança calibrada com abstenção, servidor HTTP e MCP, ONNX INT8) é o candidato para os itens 43 e 44, ligado ao motor como adaptador opcional. Usos: rotear o ticket entre modelo pequeno e LLM cara, classificar o tipo da mudança, julgar spec ambígua e ticket pronto (abstenção vai ao dono) e barrar prompt injection em comentários de issues e PRs. **Critério de entrada:** no benchmark do item 40, com limiar calibrado nos nossos dados, ele precisa reduzir tokens da LLM cara sem baixar a taxa de PRs verdes de primeira, acertar as decisões acima da linha de base e caber no piso do item 47 (ONNX INT8, um checkpoint). Se não passar, fica de fora.

### Rumo à v2.0

A `v2.0.0` marca a troca de base, não a soma de recursos. Critérios:

- o motor `sdd` em Go substitui os scripts sh/PowerShell (itens 3 e 11–17), com migração automática e um único binário por plataforma;
- o fluxo validado em mais de um agente (item 1) e fora do GitHub (item 31);
- tickets em paralelo (item 4) e revisor contra a spec (item 5) estáveis;
- formato de spec, estado e configuração versionado e documentado como contrato público (o que permite a mudança incompatível do major);
- adoção externa real: projetos de terceiros usando, com casos documentados (item 35).

Depois da v2.0: ecossistema (plugins de linguagem e de gate pela comunidade), painel web das fases e métricas (item 2), e o modo autônomo (item 7) como padrão para times.
