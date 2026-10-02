# Estudo de caso: o cv-craft com o processo SDD

[English](case-study.en.md)

O [cv-craft](https://github.com/fabiodrneles/cv-craft) é uma CLI em Go que gera currículos em PDF, Markdown e texto a partir de um YAML. Ele foi o primeiro projeto conduzido com o processo que virou o sdd-kit: **o dono decide, o agente executa**. Todos os números abaixo têm link para a fonte.

## Ponto de partida: um protótipo

O código foi escrito em outubro de 2025 e ficou parado como protótipo. A fase de descoberta do processo leu o código, rodou o que o README prometia e registrou tudo com evidência em [`specs/ANALYSIS.md`](https://github.com/fabiodrneles/cv-craft/blob/main/specs/ANALYSIS.md):

| Severidade | Achados | Exemplos |
|---|---|---|
| Crítica | 5 | acentos corrompidos no PDF (`ã`, `ç` viravam lixo); experiências perdendo responsabilidades e conquistas no Markdown e no texto; modo interativo em loop infinito com entrada fechada; `go install` do README que não funcionava |
| Alta | 8 | validações ausentes, conteúdo descartado em silêncio, ausência de testes |
| Média | 9 | consistência de saída, documentação, higiene |

Cada achado aponta `arquivo:linha` ou a saída do comando que o mostrou, e a spec que o resolve. Nenhuma linha de código mudou antes de o dono responder às **6 decisões em aberto (D1–D6)**.

## O que o processo produziu

| Item | cv-craft |
|---|---|
| Specs | 10 ([índice](https://github.com/fabiodrneles/cv-craft/blob/main/specs/README.md)), todas `Done` |
| Critérios de aceite | 71, todos com teste automatizado |
| Funções de teste e benchmark | 76, mais 9 golden files das saídas geradas |
| PRs mergeados | 24 ([lista](https://github.com/fabiodrneles/cv-craft/pulls?q=is%3Apr+is%3Amerged)) |
| CI | Linux, macOS e Windows, na versão mínima de Go e na estável, com lint, `-race`, cobertura mínima, smoke test e verificação da documentação |
| Releases | [`v0.2.0` e `v1.0.0`](https://github.com/fabiodrneles/cv-craft/releases), com binários e checksums gerados só depois do CI completo |

## Como o processo tratou os problemas

- **Nada de "funciona na minha máquina":** o achado C5 (instalação quebrada) só apareceu porque a descoberta exige **rodar** cada comando do README. Hoje os blocos de comando dos READMEs são executados no CI.
- **CI vermelho tratado pela causa raiz:** uma falha é corrigida no PR que a causou, nunca pulando um teste ou afrouxando um gate.
- **PRs pequenos e revisáveis:** um ticket, uma branch, um PR, na ordem do épico.
- **Retomada barata:** o estado de cada fase vive num comentário do épico, e o `CLAUDE.md` traz o mapa do código. Sessões novas continuaram de onde a anterior parou, sem reler o repositório.

## O mesmo processo, aplicado ao próprio kit

O sdd-kit foi construído do mesmo jeito ([épico da Fase 1](https://github.com/fabiodrneles/sdd-kit/issues/1), [épico da Fase 2](https://github.com/fabiodrneles/sdd-kit/issues/21)):

- 6 specs e 24 critérios de aceite, cada um citado por um teste. O `sdd-check --strict` confere isso no CI.
- 18 PRs mergeados até a [`v0.1.0`](https://github.com/fabiodrneles/sdd-kit/releases/tag/v0.1.0), com CI em Linux, macOS e Windows e um e2e que adota o template em Go, Node, Java e Python e roda o `make ci` gerado.
- Problemas reais pegos antes do merge, todos registrados nos PRs:
  - um comentário lido pelo shellcheck como diretiva;
  - a sincronização que sobrescreveria alterações locais em arquivos que o kit não mudou;
  - o `sdd-check` que não reconhecia IDs no formato `**FR-1 (D5)**`;
  - critérios de aceite que só tinham sido verificados à mão.

## O que levar daqui

1. **A descoberta com evidência compensa:** os achados críticos do cv-craft eram invisíveis para quem só lia o código.
2. **Decisões explícitas evitam retrabalho:** o agente para, o dono decide, e a decisão fica registrada na spec.
3. **Rastreabilidade AC → teste** transforma "parece pronto" em "está provado".

Para aplicar no seu repositório, veja [Começar num repositório novo](../README.md#começar-num-repositório-novo) e [Adotar num repositório existente](../README.md#adotar-num-repositório-existente).
