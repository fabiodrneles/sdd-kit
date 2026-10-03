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

11. **`sdd ship`:** commit, push, PR com o corpo gerado do ticket, da spec e do diff, espera do CI, merge quando verde e delegado, e checkpoint. O maior ganho: hoje são 6 a 8 chamadas por PR.
12. **`sdd release vX`:** fechamento completo (ROADMAP, CHANGELOG gerado das seções "Mudanças", versão em todos os arquivos, artefatos, release e épico fechado).
13. **`sdd sync` com regiões do projeto:** marcas no template para o merge de três vias preservar o que é do projeto (ex.: descrição e armadilhas no `CLAUDE.md`), sem resolver conflito à mão.
14. **`sdd ci why`:** só o erro e o contexto mínimo de um job vermelho, em vez do log inteiro.
15. **`sdd context`:** início de sessão num resumo (checkpoint, PRs e CI, issues da fase, próximo passo), com as consultas em paralelo.
16. **`sdd mutate`:** checagem de mutação automática de um teste novo (quebra, roda, restaura, relata).
17. **`sdd plan`:** spec, fase no ROADMAP, índice e épico a partir de um YAML curto, com os números das issues anotados de volta.

### Contra as dores comuns das ferramentas de SDD

18. **Spec viva:** um PR que muda comportamento sem linha em "Mudanças" da spec reprova no CI; specs que não envelhecem.
19. **Modo leve:** caminho rápido para bug e mudança pequena (ticket com teste, sem spec nova), porque cerimônia demais para tarefa pequena afasta quem adota.
20. **Contexto por ticket:** o motor monta só o pedaço de spec, código e decisões de que o ticket precisa, em vez de o agente ler specs inteiras.
21. **Evidência por PR:** um resumo automático no PR mostra cada critério de aceite e o teste que o prova, para o dono revisar em segundos.
22. **Times:** mais de um dono e revisor, atribuição de tickets e regras de quem aprova o quê.
23. **Gate de segurança:** segredos, dependências vulneráveis e licenças no `make ci` do template.
