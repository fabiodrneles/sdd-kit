# 013 — Primeira impressão

- **Prioridade:** P1
- **Status:** Done — entregue na `v1.3.0`
- **Código afetado:** `README.md`, `README.en.md`, `docs/`, `.github/ISSUE_TEMPLATE/config.yml`, `.github/workflows/ci.yml`, `tests/`
- **Resolve:** atrito de quem chega pela divulgação (LinkedIn, 2026-10-02): o comando de adoção sem explicação, nenhuma demonstração visual, nenhum caminho para perguntas e contribuições

## Contexto

Com a `v1.2.0` divulgada, a maior parte dos visitantes decide em segundos se testa o kit. O README mostra o comando de adoção sem dizer o que ele copia nem o que fazer depois, não há demonstração do resultado, as dúvidas não têm outro lugar além de issues e não há tarefas marcadas para quem quer contribuir. Princípio: **quem chega entende o que o kit faz em 30 segundos e testa em 5 minutos, com um caminho claro para perguntar e contribuir.**

## Requisitos funcionais

- **FR-1** O README (pt e en) MUST trazer um "Comece em 5 minutos": pré-requisitos, o que o comando de adoção copia, a garantia de não sobrescrever, `--dry-run` e o primeiro pedido ao agente; os comandos de adoção do guia MUST rodar no CI do kit como estão escritos, contra o script local (a tag do README só existe depois da release).
- **FR-2** O README MUST abrir com uma demonstração animada gerada a partir da saída real da adoção e do `make ci`, com o script que a gera versionado.
- **FR-3** O formulário de nova issue MUST levar perguntas para as Discussions, e o repositório MUST ter issues marcadas `good first issue` com o passo a passo para começar.

## Critérios de aceite

- **AC-1** Dado o README (pt e en), quando o CI do kit roda, então cada comando de adoção do guia, inclusive a variante `--dry-run`, roda com o script local num diretório vazio e passa; um argumento inválido reprova o CI.
- **AC-2** Dado o script da demonstração, quando ele roda, então gera a animação a partir da saída real dos comandos, e o README a referencia por um caminho que existe.
- **AC-3** Dado o `config.yml` dos modelos de issue, quando o CI roda, então ele tem um link de contato para as Discussions.

## Mudanças

### v1.3.0

- ADDED FR-1 — "Começar num repositório novo" explica pré-requisitos, o que a adoção copia, `--dry-run` e `--skeleton`; os comandos do guia rodam no CI (`tests/readme.sh`) (T35, #108).
- ADDED FR-2 — demonstração animada no topo do README, gerada por `docs/demo/demo.sh` a partir da saída real da adoção e do `make ci` (T36, #109).
- ADDED FR-3 — o formulário de nova issue leva perguntas e relatos de uso para as Discussions; issues `good first issue` para quem quer começar a contribuir (T37, #110).
