# 018 — Relé de sessões curtas

- **Prioridade:** P1
- **Status:** Done — entregue na `v1.11.0` (pedida pelo dono em 2026-10-06: "uma ideia que realmente nos dê uma economia muito grande de tokens e contexto")
- **Código afetado:** `template/common/scripts/sdd-relay.sh` e `sdd-context.sh` (novos), `template/common/CLAUDE.md`, `CLAUDE.md`, `plugins/sdd-delivery/`
- **Depende de:** 014 (checkpoint), 015 (motor), 016 (vigia do merge), 017 (medição)

## Contexto: onde está o gasto

A medição da spec 017 mostra que o custo não é gerar código: é **reler a conversa**. Numa sessão longa, cada chamada relê em média 375 mil tokens para gerar 533; o contexto começa em cerca de 72 mil e só cresce. O custo total é a soma do contexto de cada chamada, então cresce com o **quadrado** do tamanho da sessão: dobrar a sessão quadruplica o gasto. Despertar uma sessão longa só para ler um aviso custa o mesmo que uma chamada de trabalho.

Os scripts e o motor já tiraram da LLM o trabalho mecânico. O que falta é tirar dela a **memória**: a conversa longa é o estado do trabalho guardado no lugar mais caro que existe.

## A ideia: o estado vive no GitHub, a LLM só passa pelo ticket

Princípio: **nenhuma sessão de LLM vive mais do que um ticket.** Quem conduz a fila é um relé em shell, sem LLM. Para cada ticket, ele abre uma sessão nova e pequena, entrega um pacote de contexto montado por script, e a sessão termina quando o PR é aberto. O próximo ticket começa do zero, com o checkpoint, e não com a conversa anterior.

1. **Relé sem LLM** (`sdd-relay.sh`). Lê o "Próximo" do checkpoint, monta o pacote do ticket e abre o agente em modo não interativo com ele: `claude -p` por padrão, configurável para outro agente. Espera o PR com o vigia (spec 016), e no merge passa ao ticket seguinte. Para quando o Próximo for "perguntar ao dono", quando o CI ficar vermelho duas vezes ou quando o orçamento acabar.
2. **Pacote de contexto determinístico** (`sdd-context.sh '#N'`). Monta só o que o ticket precisa: o texto da issue, as linhas dos FR e AC citados (não a spec inteira), as decisões e armadilhas citadas, os arquivos que o ticket deve tocar (achados por `git grep` dos identificadores e pelo histórico), assinaturas em vez de corpos e os comandos de entrega. Tem um teto de tamanho; o que não cabe vira caminho de arquivo, não conteúdo. A sessão começa sabendo o que fazer, sem explorar o repositório.
3. **Teto de contexto por sessão.** Cada sessão tem um limite (por exemplo, 150 mil tokens). Ao chegar perto, ela grava um commit WIP e o checkpoint e termina; o relé abre outra a partir do checkpoint. O custo fica linear: nenhuma chamada relê mais do que o teto.
4. **Zero despertar de sessão longa.** Avisos de PR, CI e merge são tratados pelo relé e pelo motor, nunca por uma sessão de LLM esperando. A conversa com o dono fica curta: pede, aprova e mergeia.
5. **Modelo por papel.** A sessão do ticket usa o modelo configurado para implementar; o modelo mais caro entra só na escalada (CI vermelho duas vezes, ticket marcado como difícil, revisão).

Estimativa, a ser confirmada pela spec 017 e nunca anunciada antes disso: um ticket que hoje custa algo na casa de 25 chamadas × 375 mil tokens (cerca de 9 milhões) passaria a 25 chamadas × no máximo 60 a 100 mil (2 a 2,5 milhões), de **4 a 6 vezes menos**. Com o pacote encolhendo o contexto inicial e o teto cortando o crescimento, o alvo é **10 vezes menos por ticket**.

## Requisitos funcionais

- **FR-1** `sdd-context.sh '#N'` MUST montar, sem LLM, o pacote do ticket: issue, linhas dos FR/AC citados, decisões e armadilhas citadas, arquivos prováveis com assinaturas e os comandos de entrega; com teto de tamanho configurável (`SDD_CONTEXT_MAX`, em bytes), além do qual entram só caminhos.
- **FR-2** `sdd-relay.sh` MUST, sem LLM, ler o Próximo do checkpoint, montar o pacote, abrir uma sessão nova do agente com ele (`SDD_AGENT_CMD`, padrão `claude -p`), esperar o PR e o merge (vigia, spec 016) e seguir para o próximo ticket do épico.
- **FR-3** O relé MUST parar e avisar o dono (um comentário no épico) quando o Próximo for "perguntar ao dono", quando o CI do PR falhar duas vezes seguidas, ou quando o orçamento da fase (`SDD_BUDGET_TOKENS`, medido pela spec 017) acabar.
- **FR-4** A sessão do ticket MUST respeitar um teto de contexto (`SDD_SESSION_MAX_TOKENS`): ao chegar perto, grava WIP e checkpoint e termina, e o relé abre outra a partir do checkpoint.
- **FR-5** O `CLAUDE.md` (kit e template) MUST dizer que uma sessão trabalha um ticket e termina; e que conversas longas com o dono não acumulam trabalho de tickets.
- **FR-6** O relé MUST registrar no épico, por ticket, o custo medido (spec 017), para comparar com e sem relé.

## Requisitos não funcionais

- **NFR-1** O relé e o pacote MUST rodar em sh POSIX, só com `git`, `jq` e `gh`, e MUST NOT guardar estado fora do GitHub e do repositório: matar o relé no meio e rodar de novo retoma do checkpoint.
- **NFR-2** Nenhuma alegação de economia MUST ser publicada (README, release, artigo) sem a medição da spec 017 de antes e depois, na mesma fase de trabalho.

## Critérios de aceite

- **AC-1** Dado um ticket que cita `003 FR-1` e `AC-1`, quando `sdd-context.sh` roda, então o pacote tem essas linhas da spec e não o resto dela, e respeita o teto de tamanho.
- **AC-2** Dado um épico com dois tickets e um agente falso que abre um PR por ticket, quando `sdd-relay.sh` roda e os PRs são mergeados, então o agente é chamado duas vezes, cada uma com o pacote do seu ticket, e o relé termina com o Próximo da fase.
- **AC-3** Dado um CI vermelho duas vezes no mesmo PR, quando o relé o vê, então para e comenta no épico, sem chamar o agente de novo.
- **AC-4** Dado o relé interrompido no meio de um ticket, quando roda de novo, então retoma do checkpoint, sem repetir o ticket já mergeado.
- **AC-5** Dada uma fase feita com o relé, quando o `sdd-report.sh phase` roda, então o custo por ticket aparece ao lado da medição sem relé, para comparação.

## Mudanças

### v1.11.0

- ADDED FR-1 — `sdd-context.sh '#N'`: o pacote do ticket sem LLM (issue, só as linhas dos FR/NFR/AC citados, decisões, arquivos prováveis com assinaturas, armadilhas desses arquivos e comandos de entrega), com teto em `SDD_CONTEXT_MAX` (T53, #226).
- ADDED FR-2, NFR-1 — `sdd-relay.sh`: um agente novo por ticket (`SDD_AGENT_CMD`, padrão `claude -p`) com o pacote do `sdd-context.sh` na entrada, espera do merge pelo `sdd-wait.sh` e próximo ticket do épico; sem estado próprio, retoma do GitHub; para se o agente terminar sem PR, se o PR fechar sem merge ou se um ticket voltar aberto depois do merge (T54, #227).
- ADDED FR-3, FR-4 — paradas do relé com comentário no épico (Próximo "perguntar ao dono", CI vermelho duas vezes seguidas, a primeira com uma sessão de correção, e `SDD_BUDGET_TOKENS` gasto desde a abertura do épico); teto por sessão com `SDD_SESSION_MAX_TOKENS` e o hook PostToolUse `sdd-session-guard.sh`, e sessão nova a partir do checkpoint quando a anterior parou com WIP (T55, #228).
- ADDED FR-5, FR-6, AC-5 — `CLAUDE.md` (kit e template) e skill: uma sessão, um ticket; o relé grava na issue o custo exato do ticket (só os arquivos de sessão que surgiram nas chamadas dele), e o `sdd-report.sh phase` mostra a medição de cada ticket (relé ou janela) e o custo médio com e sem relé (T56, #229).
