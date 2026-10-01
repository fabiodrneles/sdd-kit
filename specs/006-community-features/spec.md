# 006 — Recursos inspirados na comunidade SDD

- **Prioridade:** P1
- **Status:** Approved — D5–D9 respondidas (a)
- **Código afetado:** `plugins/sdd-delivery/`, `template/`, `scripts/`
- **Resolve:** #16

## Contexto

O dono quer que o sdd-kit seja uma referência em desenvolvimento orientado a especificação. A comunidade já resolveu bem partes do problema; esta spec traz o que se encaixa no kit sem mudar o que ele tem de diferente (a entrega no GitHub até a release, com o dono decidindo).

Ferramentas estudadas:

| Ferramenta | Ideia aproveitável | Onde entra |
|---|---|---|
| [spec-kit](https://github.com/github/spec-kit) (GitHub) | Comandos de barra por etapa; análise de consistência entre spec, plano e tarefas; checklists | D6, D7 |
| [OpenSpec](https://github.com/Fission-AI/OpenSpec) (Fission-AI) | Mudanças descritas como deltas `ADDED`/`MODIFIED`/`REMOVED`; foco em código existente; suporte a muitos agentes | D8, D9 |
| [Kiro](https://github.com/kirodotdev/Kiro) (AWS) | Requisitos em notação EARS; tarefas ligadas aos requisitos | D5 |
| [BMAD Method](https://github.com/bmad-code-org/BMAD-METHOD) | Papéis especializados (PM, arquiteto, QA) | Fora de escopo: o kit mantém dois papéis (dono e agente) |

## Requisitos funcionais

- **FR-1 (D5)** A skill e o template SHOULD aceitar critérios de aceite em **EARS** (`QUANDO <gatilho>, O SISTEMA DEVE <resposta>`; variantes `ENQUANTO`, `SE … ENTÃO`, `ONDE`) além de Dado/Quando/Então, com exemplos dos dois.
- **FR-2 (D6)** O kit MUST oferecer `scripts/sdd-check.sh` (e o template, um alvo `make sdd-check`) que verifica a **rastreabilidade**:
  - todo `AC-*` de spec `In Progress` ou `Done` é citado em pelo menos um teste, no formato `NNN AC-n` (no nome ou num comentário);
  - o status de cada spec é igual no cabeçalho e em `specs/README.md`;
  - toda tarefa do ROADMAP cita IDs (`NNN FR-x`, `AC-y`) que existem;
  - saída em tabela (spec → AC → testes) para colar no PR.
- **FR-3 (D7)** O plugin SHOULD trazer **comandos de barra** que disparam as etapas da skill: `/sdd-analyze` (descoberta), `/sdd-specs` (constituição, specs e ROADMAP), `/sdd-epic` (épico e tickets da fase), `/sdd-next` (próximo ticket), `/sdd-status` (comentário "Estado da fase") e `/sdd-close` (PR de fechamento).
- **FR-4 (D8)** Toda spec SHOULD ter uma seção **"Mudanças"** por versão, com itens `ADDED`, `MODIFIED` e `REMOVED` apontando para os FR/AC; o PR de fechamento gera o CHANGELOG a partir dela.
- **FR-5 (D9)** O template SHOULD gerar um `AGENTS.md` (lido por Codex, Copilot, Cursor e outros) que aponta para o `CLAUDE.md` e resume o processo, para que o repositório funcione com outros agentes.

## Critérios de aceite

- **AC-1** Dado um AC de spec `In Progress` ou `Done` sem teste que o cite, quando `sdd-check` roda, então a saída lista o AC e o código de saída é diferente de zero (na Fase 2, aviso com código 0; a partir da Fase 3, erro — D6).
- **AC-2** Dado o repositório do próprio kit, quando `sdd-check` roda no CI, então passa.
- **AC-3** Dado o plugin instalado, quando o usuário digita `/sdd-status`, então o agente lê o épico aberto e publica o comentário "Estado da fase".
- **AC-4** Dado um repositório adotado, quando um agente que lê `AGENTS.md` abre o repositório, então encontra o processo e os comandos de verificação.

## Fora de escopo

- Simular um time com vários papéis (BMAD): o kit mantém dono e agente.
- Gerar código direto da spec (spec como código-fonte).

## Decisões

Respondidas pelo dono em 2026-10-01, todas na opção (a) ([ANALYSIS.md §6](../ANALYSIS.md#6-decisões-da-fase-2-respondidas-pelo-dono-em-2026-10-01)):

- D5 — EARS opcional ao lado de Dado/Quando/Então.
- D6 — `sdd-check` como aviso na Fase 2 e erro a partir da Fase 3.
- D7 — os seis comandos de barra.
- D8 — seção "Mudanças" em cada spec, gerando o CHANGELOG.
- D9 — `AGENTS.md` no template.
- Revisado na implementação (#24): a exigência de teste vale para specs `In Progress` e `Done`, e não `Approved`. Uma spec aprovada ainda não foi implementada, e cobrar testes dela geraria avisos falsos. O script fica em `template/common/scripts/sdd-check.sh` (fonte única); o kit roda a mesma cópia.
- Revisado na implementação (#25): os comandos se chamam `/sdd-analyze`, `/sdd-status` etc. (e não `/sdd:status`). O Claude Code coloca comandos de plugin no namespace do plugin (`/sdd-delivery:status`), e `/status` sozinho colide com um comando nativo; com o prefixo `sdd-`, a forma curta funciona sem conflito.
