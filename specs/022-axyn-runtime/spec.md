# 022 — axyn runtime: os modelos sem depender de um caminho só

- **Prioridade:** P1
- **Status:** Draft — escrita em 2026-10-07; implementação a partir da semana de 2026-10-12, depois da aprovação do dono
- **Código afetado:** `axyn/` (novos pacotes de provedores, agente próprio, agendador de cotas; o motor, os portões, a escada e a avaliação ficam como estão), `docs/axyn.md`, `.github/workflows/` (teste diário)
- **Resolve:** o axyn chega aos modelos só pelo `opencode run`, e quem decide o que esse caminho aceita é o opencode. Em 2026-10-07, o plano gratuito do opencode passou a recusar o agente de plano do axyn ("OpenCode's free tier can only be used from within OpenCode", issues [#50081](https://github.com/anomalyco/opencode/issues/50081) e [#49592](https://github.com/anomalyco/opencode/issues/49592) do opencode), e a avaliação dos modelos deu zero no plano para todos os onze modelos da máquina do dono

## Contexto

A spec 021 fez do axyn um orquestrador: o motor faz tudo o que é determinístico (plano validado, portões, escada de recuperação, cobertura, entrega, retomada, avaliação dos modelos), e o opencode leva o pedido ao modelo. Essa divisão funcionou no que é do axyn, mas deixou um ponto único de falha no que não é: se o opencode muda uma regra, o axyn para.

Os modelos gratuitos do opencode (OpenCode Zen) só podem ser usados de dentro do opencode, e é isso que a mensagem de erro bloqueia. Burlar essa checagem não é uma opção, nem técnica nem ética. O caminho é outro: vários provedores que oferecem uso gratuito por API de forma oficial, um agente mínimo próprio para falar com eles, e o opencode como mais um caminho, não o único.

Com isso o axyn deixa de ser só um orquestrador e passa a ter o seu próprio runtime de agentes: o laço que dá ao modelo as ferramentas necessárias, dentro dos portões. Não é um clone do opencode (sem interface de chat, sem ecossistema de plugins); é o mínimo que o processo do axyn precisa, com o processo como produto.

## Decisões do dono (2026-10-07)

- **D19** O axyn passa de orquestrador a runtime: tem conectores próprios para provedores de modelos e um agente mínimo próprio; o opencode continua suportado como um dos caminhos (backend), e o usuário que já usa o opencode não perde nada.
- **D20** Nada de contornar regra de provedor: o axyn só chama um modelo pelo caminho que o provedor permite (os gratuitos do OpenCode Zen, só pelo opencode; os outros, pela API oficial de cada um, com a chave do usuário).
- **D21** Gratuito primeiro, como na 021: os provedores com uso gratuito oficial entram antes dos pagos, e a avaliação dos modelos (`axyn bench`) mede cada modelo em cada caminho.
- **D22** Implementação em fases, cada uma entregando valor sozinha: primeiro dois provedores e o agente mínimo; depois o agendador de cotas e os outros provedores; por fim o teste diário e a versão do opencode fixada.

## Requisitos funcionais

- **FR-1** Backends: cada modelo da escada e da avaliação MUST ter um caminho: `opencode` (como hoje, `opencode run --agent`) ou `api` (o agente próprio do FR-3, falando com o provedor). A configuração (`~/.config/axyn/config.yaml`) MUST aceitar o caminho por modelo, com `opencode` como padrão para não quebrar quem já usa.
- **FR-2** Conectores de provedor por API compatível com a da OpenAI (chat com chamada de ferramentas), com a chave sempre por variável de ambiente (`key_env`, nunca no arquivo):
  - **fase 1:** GitHub Models (conta do GitHub, chave do `gh auth token`) e OpenRouter (modelos `:free`, chave grátis);
  - **fase 2:** Google Gemini (AI Studio), Groq, Cerebras, Mistral e Ollama local (sem limite de uso, para máquinas que aguentam).
- **FR-3** Agente mínimo próprio (`api`): o laço modelo → ferramenta → resultado, com só as ferramentas do processo:
  - código: ler arquivo, listar, buscar, editar trecho, escrever arquivo e rodar o CI do projeto;
  - plano: resposta em JSON com a spec e os tickets, validada pelo mesmo `axyn_plan` de hoje, sem servidor MCP.

  As ferramentas MUST respeitar a guia do modo contido (escopo, testes existentes, tamanho) antes de gravar, e os portões continuam decidindo o que entra.
- **FR-4** Agendador de cotas (fase 2): por provedor, o axyn MUST respeitar os limites por minuto e por dia (lidos do `429`, do `Retry-After` e dos cabeçalhos de limite), distribuir o trabalho entre os provedores e, ao esgotar um, seguir pelo próximo capaz da etapa (o que a 021 já faz para modelo fora do ar, agora com previsão em vez de só reação).
- **FR-5** `axyn doctor` MUST mostrar cada caminho configurado e se ele responde (um pedido mínimo, com tempo de resposta), e `axyn model` MUST listar os modelos de todos os caminhos, marcando os gratuitos.
- **FR-6** `axyn bench` MUST avaliar cada modelo em cada caminho disponível e rotear cada etapa para o melhor par modelo e caminho; um caminho que recusa o pedido (como a recusa do plano gratuito) conta como indisponível, nunca como nota baixa.
- **FR-7** Proteção contra mudanças de terceiros (fase 3): o axyn MUST conhecer as versões do opencode já testadas e avisar no `doctor` quando a instalada for outra; um workflow diário MUST chamar cada caminho com um pedido mínimo (com chaves gratuitas guardadas como secrets) e abrir uma issue quando algum deixar de funcionar.

## Requisitos não funcionais

- **NFR-1** Nenhuma dependência paga e nenhum serviço próprio: o runtime roda na máquina do usuário, com as chaves dele.
- **NFR-2** Os portões, a guia, a escada, a cobertura e a avaliação valem igual para todos os caminhos: nenhum diff entra sem passar por eles, qualquer que seja o provedor.
- **NFR-3** Leve: o runtime MUST rodar bem numa máquina como a do dono (i3 de 4ª geração, 4 GB de RAM), sem processo extra além do axyn.
- **NFR-4** Sem dependência de bibliotecas de terceiros para os conectores: HTTP e JSON da biblioteca padrão do Go, como o resto do axyn.

## Critérios de aceite

- **AC-1** Dado um provedor falso compatível com a OpenAI e um ticket, quando o axyn roda o agente próprio, então o modelo lê e edita arquivos só pelas ferramentas do FR-3, o CI roda, e o ticket passa pelos mesmos portões e é entregue.
- **AC-2** Dado um provedor falso que responde `429` com `Retry-After`, quando o agendador recebe o próximo pedido, então espera o tempo pedido ou segue para outro provedor capaz, e a tentativa não conta.
- **AC-3** Dado um plano em JSON do agente próprio com um ticket que cita um AC inexistente, quando o axyn valida, então recusa com o motivo, exatamente como o `axyn_plan` de hoje.
- **AC-4** Dado o opencode recusando o pedido (a recusa do plano gratuito) e um caminho `api` configurado, quando o axyn planeja, então segue pelo caminho `api` sem parar e registra a troca.
- **AC-5** Dado um modelo em dois caminhos, quando o `axyn bench` roda, então o relatório mostra a nota de cada caminho, e o roteamento escolhe o melhor par para cada etapa.
- **AC-6** Dado um modelo do agente próprio tentando editar um teste existente no modo contido, quando a ferramenta de edição é chamada, então a edição é recusada antes de gravar, com o motivo devolvido ao modelo.

## Fora do escopo

Interface de chat própria, plugins, compartilhamento de sessões e tudo o que o opencode oferece além do laço de agente. O objetivo não é substituir o opencode para quem gosta dele, e sim não depender dele.

## Por que isto é importante para o projeto

1. **Confiabilidade.** Hoje um ajuste de regra de terceiros para o axyn inteiro, como aconteceu em 2026-10-07. Com vários caminhos, uma mudança vira uma troca automática de caminho, não uma parada.
2. **A promessa do produto.** O axyn existe para quem só tem modelos gratuitos. Essa promessa não pode depender da política de um único fornecedor de modelos gratuitos.
3. **O processo continua sendo o diferencial.** Portões, guia, escada, cobertura e avaliação são do axyn e valem para qualquer caminho. O runtime só garante que sempre haverá um caminho até um modelo.
4. **Mais modelos gratuitos.** GitHub Models, OpenRouter, Gemini, Groq, Cerebras, Mistral e Ollama somam dezenas de modelos gratuitos, e a avaliação já sabe escolher entre eles.
5. **Controle sobre o que o modelo recebe.** Com o agente próprio, o axyn decide exatamente quais ferramentas e quanto contexto o modelo vê, o que reduz custo de tokens e o espaço para o modelo sair do roteiro.

## Mudanças

### Não lançado

- Spec escrita (Draft), a partir da recusa do plano gratuito do opencode na avaliação dos modelos de 2026-10-07.
