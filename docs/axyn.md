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
   - `axyn decide "instrução"`: a sua instrução vira uma decisão na spec, o modelo passa a recebê-la e o ticket ganha novas tentativas;
   - `axyn model` e depois `axyn run --resume`: outro modelo, com outras 10 tentativas;
   - corrigir na branch WIP você mesmo, ou com outra IA, e rodar `axyn run --resume`: os portões rodam primeiro no seu código;
   - quando o problema é cobertura, baixar o `COVERAGE_MIN` no `Makefile` da branch base.

O código da última tentativa nunca se perde. Ele fica num commit WIP, na branch `feat/<n>-wip`. Com remoto e `gh`, essa branch vira um **PR em rascunho** com o título `[não passou nos portões] ...`. Ele traz o ticket, os erros e um texto pronto para colar em outra IA. Sem remoto, o mesmo texto vai para `.axyn/ajuda-ticket-<n>.md`.

### 7. Entrega

Com os portões verdes, a ferramenta `axyn_ship` faz o commit (`feat: <título do ticket>`) na branch do ticket e, se houver um remoto `origin`, o push. Com o `gh` instalado, ela abre o PR contra a base. O merge é seu, a não ser que o repositório tenha o merge automático ligado (`axyn setup`). Cada ticket tem a sua própria branch e o seu próprio PR.

## Interrupções e retomada

- Cada execução grava o seu estado em `.axyn/runs/<ID>.json`, com o PID do processo, e o log completo em `.axyn/runs/<ID>.log`.
- O axyn considera uma execução **interrompida** quando o processo morreu (o computador desligou ou travou) ou quando ela está sem sinal há mais de 30 minutos.
- `axyn run --resume` continua do ticket aberto. Se ele achar código de uma tentativa interrompida, guarda esse código num commit WIP antes de seguir. Nada se perde.
- Fechar o terminal ou o opencode não para o axyn: ele roda num processo próprio.

## Como auditar uma execução

Rode todos estes comandos na raiz do projeto:

| Pergunta | Comando ou arquivo |
|---|---|
| O que está acontecendo agora? | `axyn status`, ou `axyn status --watch` para ver uma linha a cada mudança |
| O que aconteceu, do começo ao fim? | `axyn history`, que grava `.axyn/axyn_history-<ID>.md` com o pedido, o plano, cada tentativa, os portões, as perguntas, os commits, o ambiente e o log, sem chaves nem tokens |
| Qual foi a saída exata do `make ci`? | `.axyn/runs/<ID>.log` |
| O que o axyn decidiu fazer com cada ticket? | `.git/axyn/plan.json`: tickets, tentativas (modelo, degrau, motivo), branch WIP |
| O que você decidiu? | A seção `## Decisões` da spec (`specs/NNN-nome/spec.md`) |
| O código da tentativa reprovada | `git log feat/<n>-wip` e `git diff main...feat/<n>-wip` |
| Qual a cobertura, o mínimo, a meta e onde falta teste? | `axyn coverage` |
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
