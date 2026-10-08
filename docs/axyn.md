# Como o axyn funciona (documentação técnica)

Este documento explica o que o axyn faz por dentro, passo a passo, para que você consiga auditar qualquer execução sozinho: saber o que ele fez, por que parou e o que corrigir, sem depender de outra IA. Os valores citados aqui são os do código (`axyn/`); quando um valor mudar, este documento muda no mesmo PR.

## Em uma frase

O axyn recebe um pedido, escreve uma spec com critérios de aceite, divide o trabalho em tickets e, para cada ticket, chama um modelo pelo opencode (sem abrir a interface), confere o código com portões determinísticos e só faz commit e PR quando tudo passa. Quem decide se o código está pronto são o `make ci` e as regras do axyn, nunca o modelo.

## Peças

| Peça | O que é | Onde fica |
|---|---|---|
| `axyn` | Um binário Go, sem dependências | `C:\Users\<você>\AppData\Local\axyn\axyn.exe` (Windows) ou `~/.local/bin/axyn` |
| opencode | Executa o modelo. O axyn usa o modo sem interface, `opencode run --agent <agente>` | instalado à parte |
| Agentes do opencode | `axyn` (conversa no `/axyn`), `axyn-plan` (escreve a spec e os tickets) e `axyn-code` (escreve o código de um ticket) | `opencode.json` do projeto, gravado pelo `axyn install` |
| Servidor MCP | `axyn mcp`: as ferramentas que os agentes chamam (`axyn_plan`, `axyn_next`, `axyn_gate`, `axyn_ship`, `axyn_decide`, `axyn_run`, `axyn_status`, `axyn_history`) | iniciado pelo opencode |
| Escada de modelos | A lista de modelos que o axyn usa, em ordem | `~/.config/axyn/config.yaml`, gravado pelo `axyn model` |

## O ciclo de uma execução

```text
axyn run "pedido"  (ou /axyn no opencode)
  └─ processo em segundo plano (sobrevive ao terminal e ao opencode)
       1. preparação  → Makefile com make ci, CI do GitHub e lint, se o projeto não tiver
       2. plano       → agente axyn-plan grava spec (FR/AC) e tickets
       3. para cada ticket:
            a. branch a partir da base
            b. agente axyn-code escreve o código (com o modelo da escada)
            c. portões: protegidos, testes, tamanho, make ci
            d. verde → commit, push, PR    |   reprovado → escada de recuperação
       4. fim: volta à branch em que você estava
```

### 1. Preparação

Se o projeto não tem um `make ci`, o axyn detecta a stack (Go, Node, Python, Java, .NET, Rust ou web) e copia o template dela: o `Makefile` com o alvo `ci`, o workflow do GitHub e a configuração do lint. Ele nunca sobrescreve um arquivo que já existe.

Antes do primeiro ticket, o axyn confere se as ferramentas do `make ci` estão instaladas (por exemplo, `golangci-lint` no Go e o `sh` do Git no Windows). Se faltar alguma, ele para **sem gastar tentativa** e mostra o comando completo de instalação.

### Cobertura de testes, sem editar nada à mão

O `Makefile` guarda a meta de cobertura (`COVERAGE_MIN`, 80% por padrão): a porcentagem do código que precisa rodar nos testes. O axyn cuida disso sozinho:

1. **No início de toda execução, ele mede a cobertura com os testes que você já tem** (roda o `make ci` na branch base; se o lint falhar antes dos testes, mede com `make test` e avisa os erros que a base já tinha). A medição nunca fica marcada como "feita": só o mínimo alcançado é guardado.
2. **Se ela está abaixo da meta**, ele diz quanto falta e onde falta (em Go, os arquivos menos cobertos, com quantas instruções estão sem teste) e pergunta **uma vez** quem escreve os testes que faltam:
   - `axyn coverage auto`: o axyn escreve, num ticket antes dos outros ("Testes para a cobertura chegar a 80%"), sem mudar o código de produção;
   - `axyn coverage manual`: você escreve, quando quiser; o axyn segue já com o seu pedido.
   Para não ser perguntado: `AXYN_COVERAGE=auto` (ou `manual`). Para ver o estado a qualquer momento: `axyn coverage`.
3. **A cobertura nunca cai.** O valor medido vira o mínimo dos portões. Cada ticket entregue sobe esse mínimo até a cobertura que ele alcançou, até a meta, e grava o número novo no `COVERAGE_MIN` do `Makefile`, no próprio commit do ticket, para o CI do PR exigir o mesmo. Uma linha de comentário no `Makefile` guarda a meta.
4. **Se a cobertura cair abaixo do mínimo já alcançado** (um teste apagado ou desligado entre uma execução e outra), o axyn para antes de qualquer ticket. Ele mostra quanto caiu, os commits que mexeram em testes desde a última medição e três saídas: restaurar os testes e rodar `axyn run --resume`; `axyn coverage auto`, para o axyn escrever testes até voltar ao mínimo; ou `axyn coverage accept`, para aceitar o mínimo novo, o que fica registrado como decisão na spec.
5. **Todo código novo precisa de teste.** Um ticket que muda código sem nenhum teste novo ou alterado é reprovado (`[tests]`). Em Go, pelo menos 80% das linhas que o ticket acrescenta precisam rodar nos testes (`[coverage]`); a reprovação cita o arquivo e as linhas sem teste.

### 2. Plano

O agente `axyn-plan` recebe o pedido e grava, pela ferramenta `axyn_plan`:

- a spec em `specs/NNN-nome/spec.md`, com requisitos (FR-n) e critérios de aceite (AC-n);
- a lista de tickets, cada um citando os ACs que cumpre.

**Plano gratuito do opencode.** O plano gratuito (OpenCode Zen) recusa pedidos de agentes que não têm as ferramentas de edição e de terminal ("free tier can only be used from within OpenCode"; issues [#50081](https://github.com/anomalyco/opencode/issues/50081) e [#49592](https://github.com/anomalyco/opencode/issues/49592) do opencode). Quando isso acontece, o axyn refaz o plano com o `axyn-plan-open`, o mesmo planejador com essas ferramentas ligadas, e desfaz qualquer arquivo que ele tenha mexido. Só a spec, gravada pelo `axyn_plan` num commit próprio, fica. A recusa não conta como tentativa.

O plano fica em `.git/axyn/plan.json`: dentro do `.git`, para nunca sujar a árvore do projeto. O axyn tenta gravar o plano até 2 vezes. Se o agente não gravar, a execução para com essa mensagem.

### 3. Cada ticket

1. O axyn volta à branch base (a branch em que você estava ao pedir) e cria a branch do ticket.
2. Chama o agente `axyn-code` com o ticket, as linhas da spec com os ACs dele e as **decisões que você já tomou** (`axyn decide`). Se a tentativa anterior foi reprovada, o prompt também leva o motivo exato.
3. Cada chamada ao modelo tem um teto de 20 minutos. Um modelo que trava conta como tentativa reprovada.
4. Roda os portões no diff da branch contra a base.

### 4. Os portões

Os portões são determinísticos: o mesmo código dá sempre o mesmo resultado. Cada reprovação tem um código entre colchetes, que aparece no status, na mensagem de parada e no histórico:

| Código | O que reprova | Como corrigir |
|---|---|---|
| `[protected]` | O diff mexe num caminho protegido: `.github/workflows`, `Makefile`, configuração do lint (`.golangci.yml`, `eslint`, `.markdownlint`), `lychee.toml`, `specs`, `.axyn`, `axyn.yaml`, `.sdd-release`, `plugin.json` | O modelo não pode mudar as regras do jogo. Mude esses arquivos você mesmo, na branch base, com commit |
| `[tests]` | Um arquivo de teste foi apagado ou ficou com menos testes | Os testes só podem aumentar |
| `[size]` | O diff passa de 400 linhas (adicionadas mais removidas) | Ticket grande demais: a escada divide o ticket |
| `[tests]` | O ticket muda código sem nenhum teste novo ou alterado | Escreva um teste para cada critério de aceite |
| `[coverage]` | Em Go, menos de 80% das linhas novas rodam nos testes | A reprovação cita arquivo e linhas sem teste |
| `[ci]` | O `make ci` falhou | O motivo traz até 6 linhas de erro da saída do `make ci` |
| `[guia]` | Um modelo no modo guiado mexeu fora dos arquivos do ticket ou alterou um teste que já existia | Faça só o que o ticket pede, nos arquivos citados; acrescente testes novos em vez de alterar os antigos |
| `[plan]` | O agente alterou o plano do axyn | O plano é restaurado e a tentativa conta como reprovada |

Para rodar os mesmos portões à mão, na raiz do projeto: `axyn gate --base main`.

### 5. Escada de recuperação

Uma tentativa reprovada não é o fim. Para cada modelo, o axyn sobe uma escada de estratégias, e cada degrau ganha um número de tentativas:

| Degrau | Tentativas | O que o axyn faz |
|---|---|---|
| primeira tentativa | 1 | Só o ticket |
| diagnóstico exato | 1 | Devolve ao modelo o erro exato, com o trecho do código em volta |
| mais contexto | 1 | Mostra os arquivos e os testes ligados ao erro |
| várias tentativas | 2 | Pede versões diferentes do código; vale a primeira que passa |
| ticket dividido | 3 | Primeiro só o teste, depois o código mínimo, depois o ajuste final |
| plano antes do código | 1 | O modelo lista os arquivos que vai mexer; o axyn só aceita os do ticket |
| pergunta ao usuário | 1 | O axyn para e explica a situação para você |

São 10 tentativas por modelo. Esgotado um modelo, o axyn passa ao próximo da escada (`axyn model` define a lista). Esgotados todos, ele para.

### 6. Quando o axyn para

A mensagem de parada é a mesma no `axyn status`, no `/axyn` e no histórico. Ela traz:

1. em que ticket parou (por exemplo, 1 de 3) e quantas vezes o modelo escreveu o código;
2. o que foi tentado em cada tentativa: o degrau, o modelo e o motivo da reprovação;
3. o erro da última tentativa, com as linhas reais do `make ci`;
4. o que esse erro quer dizer, em palavras simples;
5. as saídas, com o comando de cada uma:
   - `axyn retry`: mais 10 tentativas com o mesmo modelo, que agora recebe o resumo do que já falhou para fazer diferente;
   - `axyn decide "instrução"`: a sua instrução vira uma decisão na spec, o modelo passa a recebê-la e o ticket ganha novas tentativas;
   - `axyn model` e depois `axyn run --resume`: outro modelo, com outras 10 tentativas;
   - corrigir na branch WIP você mesmo, ou com outra IA, e rodar `axyn run --resume`: os portões rodam primeiro no seu código;
   - quando o problema é cobertura, baixar o `COVERAGE_MIN` no `Makefile` da branch base.

O código da última tentativa nunca se perde. Ele fica num commit WIP, na branch `feat/<n>-wip`. Com remoto e `gh`, essa branch vira um **PR em rascunho** com o título `[não passou nos portões] ...`. Ele traz o ticket, os erros e um texto pronto para colar em outra IA. Sem remoto, o mesmo texto vai para `.axyn/ajuda-ticket-<n>.md`.

### 7. Entrega

Com os portões verdes, a ferramenta `axyn_ship` faz o commit (`feat: <título do ticket>`) na branch do ticket e, se houver um remoto `origin`, o push. Com o `gh` instalado, ela abre o PR contra a base. O merge é seu, a não ser que o repositório tenha o merge automático ligado (`axyn setup`). Cada ticket tem a sua própria branch e o seu próprio PR.

## Avaliação dos modelos (axyn bench)

O axyn escolhe sozinho o melhor modelo para cada etapa. Ele não usa uma tabela genérica da internet: testa os modelos que **você** tem no opencode, nas tarefas do **seu** processo.

**Automático por padrão.** Na primeira execução, ou quando a avaliação vence (mais de 30 dias, outra versão do opencode ou do axyn, ou um modelo novo), o `axyn run` avalia todos os modelos gratuitos da máquina antes de planejar e aplica o resultado. O status mostra a fase "avaliando modelos". Para pular: `AXYN_BENCH=off`.

**O conjunto fixo de tarefas.** Cada tarefa tem um identificador estável e é sempre a mesma, para dar para comparar entre rodadas:

| Tarefa | Etapa | O que o modelo faz | Como a nota é dada (0 a 100) |
|---|---|---|---|
| `plan-001` | plano | escreve a spec e os tickets de um comando novo, pela ferramenta `axyn_plan` | plano gravado (30), pelo menos 3 ACs (15), pelo menos 2 ACs de erro (20), AC de borda (10), sem AC repetido (10), tickets citando ACs que existem, até 6 (15) |
| `code-001` | código | implementa uma função com acentos e casos de borda para um teste pronto passar | CI verde (70), rapidez (até 20), diff pequeno (10) |
| `tests-001` | testes | escreve os testes de um pacote de preços | testes passam 3 vezes seguidas (sem instabilidade) (20), cobertura (até 30), **bugs injetados que os testes pegam** (até 40, 8 bugs), sem `sleep`, sem `skip` e com comparação de valor (10) |
| `fix-001` | conserto | conserta o código para o `go vet` e os testes passarem | CI verde (70), rapidez (até 20), diff pequeno (10) |

**Regras que vêm dos materiais de QA:**

- Nota só de verificações automáticas: nenhum modelo julga outro.
- **Veto:** alterar ou enfraquecer um teste, ou mexer no código numa tarefa de testes, zera o modelo naquela etapa ("build verde não é qualidade"), e a média não salva.
- Para receber uma etapa, o modelo precisa de nota mínima (`--min-score`, 50 por padrão): só passar não basta.
- Testes escritos pelo modelo precisam pegar bugs injetados ("testar o testador").
- Cada rodada fica gravada, e a seção "Pioraram desde a última rodada" mostra quedas de 15 pontos ou mais.
- O commit de cada ticket registra o modelo autor (`Axyn-Model:`), para medir depois qual código sobrevive.

**Nenhum modelo é descartado.** Um modelo abaixo da nota mínima numa etapa, ou pego enfraquecendo um teste, continua na fila dessa etapa, depois dos outros, rodando no **modo guiado**. A guia é aplicada pelo próprio axyn, faça o modelo o que fizer:

- **escopo travado:** quando o ticket cita arquivos, só eles e os testes podem mudar;
- **testes existentes travados:** o modelo só pode acrescentar testes novos;
- **diff menor:** no máximo 150 linhas;
- **instruções estritas:** o prompt começa com as regras do modo guiado.

O que sair da guia é reprovado com o código `[guia]`, e os portões de sempre continuam valendo. Assim, mesmo uma máquina só com modelos fracos consegue usar o axyn.

**Modelo fora do ar ou sem limite.** Modelos gratuitos atingem limite de tokens ou de pedidos, e provedores saem do ar. Quando a saída do opencode traz um erro de provedor (429, cota, 503, conexão recusada, modelo sobrecarregado) **e** o modelo não mudou nenhum arquivo, o axyn tira o modelo da fila naquela execução. A tentativa não conta, e o mesmo trabalho passa para o próximo modelo capaz daquela etapa; o status mostra a troca. Se todos caírem, o axyn para explicando e você retoma depois com `axyn run --resume`. Na avaliação, um modelo fora do ar, sem chave de API válida (401, chave inválida) ou sem crédito (402) fica como "indisponível", sem nota ruim, e é avaliado de novo na próxima rodada. Depois da primeira queda, as outras tarefas dele naquela rodada são puladas na hora, sem gastar tempo. Entram na avaliação só os modelos cuja chave a máquina tem (variável de ambiente ou `opencode auth login`); com a chave do OpenRouter configurada, os modelos gratuitos dele entram também.

**Como o resultado é usado.** O planejador usa o melhor modelo de plano. Um ticket de código usa os melhores de código e de conserto, e um ticket de testes, os melhores de testes. A escada do `axyn model` continua depois deles. Um forçado (`--model`) vence tudo.

**Personalizar** (todos na raiz do projeto ou em qualquer pasta):

| Quero | Comando |
|---|---|
| Ver a última avaliação | `axyn bench --show` |
| Refazer só algumas etapas, sem perder as outras | `axyn bench plano testes` (as etapas: `plano`, `codigo`, `testes`, `conserto`, ou em inglês `plan`, `code`, `tests`, `fix`; dá para juntar com `--models A,B`) |
| Voltar ao painel de uma avaliação em andamento | `axyn bench --watch` |
| Rodar nesta janela, sem segundo plano | `axyn bench --here` |
| Avaliar de novo agora | `axyn bench` |
| Só alguns modelos, ou também os pagos | `axyn bench --models A,B` ou `axyn bench --all` |
| Mais confiança (cada tarefa várias vezes) | `axyn bench --runs 3` |
| Escolher quantos modelos avaliar ao mesmo tempo (o padrão se ajusta à máquina: 1 com menos de 8 GB de RAM ou até 4 núcleos, 3 com 16 GB e 8 núcleos, senão 2) | `axyn bench --parallel 2` |
| Ser perguntado antes de aplicar | `axyn bench --ask` |
| Fixar à mão o modelo de uma etapa | `axyn bench --set plano=MODELO` (desfazer: `--set plano=`) |
| Desligar e voltar à escada do `axyn model` | `axyn bench --off` (religar: `--apply`) |

Arquivos: `~/.config/axyn/bench/profile.json` (o resultado em uso), `bench-DATA.md` (o relatório de cada rodada, com a nota e o motivo de cada tentativa) e `bench-DATA.log` (a saída completa dos modelos) e a pasta `bench-DATA/`, com o código que o modelo escreveu em cada tentativa (um `.diff` por tentativa). Com os três, dá para conferir se cada nota foi justa e ajustar o processo de avaliação. **Resultados de rodadas anteriores.** Quando uma rodada refaz só algumas etapas (`axyn bench plano testes`), as outras etapas mantêm o resultado da rodada anterior. Na tabela, esse resultado vem marcado com `*`, e nos detalhes aparece a data da rodada. Um modelo que ficou sem chave sai do relatório e da escolha junto com os resultados antigos.

**Uma janela só.** O `axyn bench` roda em segundo plano, como o `axyn run`, e a janela vira um painel. **Enter** alterna entre o progresso (barra, contagem, tempo e estimativa) e o log ao vivo, formatado. **Ctrl + C** fecha só o painel, e a avaliação continua; `axyn bench --watch` volta para ele, e `axyn status` mostra em que pé está. No fim, o painel mostra o resumo e uma notificação avisa. O mesmo Enter vale no `axyn status --watch` de uma execução. Para ver só o log, sem o painel: `axyn logs`. Para ler com calma o que já passou, digite `p` e aperte Enter: o painel para de escrever, a janela pode ser rolada à vontade e a avaliação continua; Enter volta, do ponto em que parou. Na janela clássica do Windows PowerShell, que volta sozinha para o fim a cada escrita, o progresso não é animado: sai uma linha a cada mudança, e a tela pode ser rolada. As linhas do log têm cor por tipo: ações do agente em ciano, comandos em amarelo, testes que passaram em verde, falhas em vermelho, tabelas e listagens em cinza. No log formatado, o diff que o modelo imprime depois de cada edição vira uma linha por arquivo (`✎ slug.go: +47 −1 linhas`); o texto do modelo aparece inteiro. Na janela clássica do Windows PowerShell, cuja fonte não tem alguns símbolos, o axyn usa os que toda fonte do Windows tem (`√`, `×`, `►`); no Windows Terminal ficam os símbolos completos. O comando mostra o log mais recente, de uma execução deste projeto ou de uma avaliação, de forma legível: um cabeçalho por tentativa, uma linha por ação, erros em vermelho, notas em verde, indisponíveis em amarelo e JSON longo resumido com o tamanho. Para ver o texto completo, sem formatação, use `axyn logs --raw`.

Os logs, relatórios e históricos do axyn são gravados em UTF-8 com a marca (BOM) no início, para o PowerShell 5.1 e os editores mostrarem os acentos certos. As tarefas são em Go e precisam do Go instalado.

## Fechar uma versão (release)

O projeto que o axyn prepara recebe o `scripts/sdd-release.sh` e o workflow de release do sdd-kit. O `axyn release` usa os dois. Rode-o no terminal, na raiz do projeto, ou peça no `/axyn` ("feche a versão"):

1. **Prévia, sem mudar nada:** o axyn mostra a próxima versão e o que entra nela. A versão é calculada pelo [go-release-manager](https://github.com/fabiodrneles/go-release-manager) a partir dos commits na `main`: `feat` sobe a versão do meio, `fix` a do fim e `!` a primeira. O que entra é o rascunho do CHANGELOG, montado pelos PRs mesclados.
2. **Com o seu sim** (`s` no terminal, ou "sim" no `/axyn`): abre o PR de fechamento (`chore/release-vX.Y.Z`) com o CHANGELOG e as versões dos arquivos.
3. **Depois do merge desse PR:** o workflow do projeto cria a tag e publica a página da release, com as novidades e o "Como atualizar". Sem o workflow, rode `axyn release --tag X.Y.Z`.

`axyn release --yes` pula a pergunta. A release precisa do `gh` e do Go (o go-release-manager roda com `go run`). Se faltar algum, o axyn mostra o comando de instalação. Se o projeto não tiver o script, `axyn init` copia o que falta do template sem mexer no que já existe.

## Interrupções e retomada

- Cada execução grava o seu estado em `.axyn/runs/<ID>.json`, com o PID do processo, e o log completo em `.axyn/runs/<ID>.log`.
- O axyn considera uma execução **interrompida** quando o processo morreu (o computador desligou ou travou) ou quando ela está sem sinal há mais de 30 minutos.
- `axyn run --resume` continua do ticket aberto. Se ele achar código de uma tentativa interrompida, guarda esse código num commit WIP antes de seguir. Nada se perde.
- Fechar o terminal ou o opencode não para o axyn: ele roda num processo próprio.

## Como auditar uma execução

Rode todos estes comandos na raiz do projeto:

| Pergunta | Comando ou arquivo |
|---|---|
| O que o modelo está fazendo agora, ao vivo? | `axyn logs` (ou `axyn logs --raw` para o texto completo) |
| O que está acontecendo agora? | `axyn status`, ou `axyn status --watch`: uma linha a cada mudança e, num terminal, um indicador animado (amarelo e branco) com o tempo da fase e, na avaliação dos modelos, a barra de progresso, a contagem e quanto falta |
| O que aconteceu, do começo ao fim? | `axyn history`, que grava `.axyn/axyn_history-<ID>.md` com o pedido, o plano, cada tentativa, os portões, as perguntas, os commits, o ambiente e o log, sem chaves nem tokens |
| Qual foi a saída exata do `make ci`? | `.axyn/runs/<ID>.log` |
| O que o axyn decidiu fazer com cada ticket? | `.git/axyn/plan.json`: tickets, tentativas (modelo, degrau, motivo), branch WIP |
| O que você decidiu? | A seção `## Decisões` da spec (`specs/NNN-nome/spec.md`) |
| O código da tentativa reprovada | `git log feat/<n>-wip` e `git diff main...feat/<n>-wip` |
| Qual a cobertura, o mínimo, a meta e onde falta teste? | `axyn coverage` |
| Que modelo o axyn usa em cada etapa, e por quê? | `axyn bench --show` e o relatório `~/.config/axyn/bench/bench-DATA.md` |
| Qual seria a próxima versão e o que entra nela? | `axyn release`, respondendo `N` na pergunta |
| O ambiente está certo? | `axyn doctor`: uma linha por item, com o comando de cada coisa que falta |
| Os portões dão o mesmo resultado à mão? | `axyn gate --base main` |

## Arquivos que o axyn cria

| Arquivo | Versionado? | Para quê |
|---|---|---|
| `.axyn/` | não (tem um `.gitignore` com `*`) | Estado das execuções, logs, histórico, ajuda dos tickets parados |
| `.git/axyn/plan.json` | não (fica dentro do `.git`) | Plano: tickets e tentativas |
| `specs/NNN-nome/spec.md` | sim, com commit próprio | Spec e decisões |
| `opencode.json` | sim | Servidor MCP, agentes e o comando `/axyn` |
| `Makefile`, `.github/workflows/...`, lint | sim, no commit `chore: prepare project` | Só quando o projeto não tinha `make ci` |
| `~/.config/axyn/config.yaml` | fora do projeto | A escada de modelos |
| `~/.config/axyn/update-check.json` | fora do projeto | Cache diário do aviso de versão nova |

## Variáveis de ambiente

| Variável | Efeito |
|---|---|
| `AXYN_COVERAGE=auto` ou `manual` | Responde de antemão quem escreve os testes que faltam |
| `AXYN_NO_UPDATE_CHECK=1` | Desliga o aviso de versão nova |
| `AXYN_NO_MODIFY_PATH=1` | O instalador não mexe no PATH |
| `AXYN_OPENCODE` | Caminho de outro executável no lugar do `opencode` (usado nos testes) |
| A `key_env` de cada modelo | A chave de API do modelo, quando ele não é gratuito (por exemplo, `OPENROUTER_API_KEY`) |

## Garantias

- Nenhum commit sem os portões verdes, exceto os commits WIP, que ficam só nas branches `-wip`.
- O modelo não altera testes existentes para baixo, não altera specs, workflows nem a configuração do lint e não altera o plano.
- Uma ferramenta que falta nunca gasta tentativa.
- O histórico remove chaves e tokens (`sk-…`, `ghp_…`, `github_pat_…`, `Bearer …`, `*_KEY=`, `*_TOKEN=` e credenciais em URLs).
