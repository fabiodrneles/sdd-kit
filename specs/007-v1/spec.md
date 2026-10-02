# 007 — Versão 1.0: profissional e pronto para a comunidade

- **Prioridade:** P1
- **Status:** Draft — aguarda as decisões D10–D12 do dono
- **Código afetado:** `Makefile`, `template/`, `plugins/sdd-delivery/`, `docs/`, arquivos de comunidade
- **Resolve:** #39, #40

## Contexto

A `v0.1.0` entregou o kit funcionando (Fases 1 e 2). A `v1.0.0` fecha o que um projeto de referência precisa:

- rastreabilidade obrigatória;
- arquivos de comunidade;
- uma prova pública do processo de ponta a ponta;
- mais linguagens;
- uma skill revisada a partir do uso real.

## Requisitos funcionais

- **FR-1** O CI do kit e o workflow `docs.yml` do template MUST rodar `sdd-check --strict` (D6: erro a partir da Fase 3).
- **FR-2** O repositório MUST ter `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` (Contributor Covenant 2.1), `SECURITY.md` e um issue template "Nova linguagem".
- **FR-3** O repositório MUST ter um estudo de caso em `docs/` com números verificáveis (cada número com link para a issue, o PR ou a release de origem).
- **FR-4 (D10)** O template SHOULD ganhar as linguagens escolhidas pelo dono, cada uma com `make ci`, CI, hook, dependabot e o e2e da spec 003 AC-4.
- **FR-5 (D11)** A skill MUST ser revisada depois do uso em pelo menos dois repositórios além do cv-craft (T10). Cada problema encontrado vira um ticket, e a revisão é registrada no CHANGELOG.
- **FR-6 (D12)** A skill SHOULD estar disponível para quem não lê português, na forma escolhida pelo dono.

## Critérios de aceite

- **AC-1** Dada uma spec `Done` com um AC sem teste, quando o CI do kit roda, então falha.
- **AC-2** Dado o repositório, quando se abre Insights → Community Standards no GitHub, então todos os itens estão completos.
- **AC-3** Dado o estudo de caso, quando o job de links roda, então todos os links resolvem.
- **AC-4** Dada cada linguagem nova, quando o job "Template" roda, então a adoção num projeto mínimo e o `make ci` gerado passam.

## Fora de escopo

- Site de documentação próprio: o README e `docs/` no GitHub bastam na `v1.0.0`.

## Decisões em aberto

Ver [ANALYSIS.md §7](../ANALYSIS.md#7-decisões-da-fase-3-em-aberto).
