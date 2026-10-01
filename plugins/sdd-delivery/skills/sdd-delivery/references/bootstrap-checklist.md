# Checklist de reaplicação num repositório novo

Siga em ordem. Itens marcados **(dono)** são configurações ou decisões do dono: o agente sugere,
não executa. Os `⏸` são pontos de parada.

## 0. Antes de começar

- [ ] Confirmar com o dono: branch principal, idioma dos documentos, SOs e versões suportados,
      se haverá release automatizada.
- [ ] Verificar o que já existe (`specs/`, `CONTRIBUTING.md`, `.github/`, CI, labels, épicos)
      para **estender**, não duplicar.

## 1. Descoberta

- [ ] Ler todo o código; compilar; rodar lint e testes existentes.
- [ ] Executar cada comando documentado e os casos de borda (entrada inválida, arquivo
      existente, stdin fechado / sem TTY, `--help`, `--version`).
- [ ] Verificar cada promessa do README (instalação inclusive).
- [ ] `specs/ANALYSIS.md` com achados por severidade + evidência, positivos, README,
      melhorias por fase e decisões `D1..Dn` com recomendação.
- [ ] Mensagem curta ao dono com as observações principais e as decisões.
- [ ] ⏸ **Aguardar as respostas.** Nenhum commit de código antes.

## 2. Artefatos SDD

- [ ] `specs/constitution.md` com 5–10 princípios verificáveis.
- [ ] `specs/README.md` (fluxo SDD, convenções, índice de specs com status).
- [ ] Uma spec por área: `specs/NNN-nome/spec.md` (FR/NFR, AC Dado/Quando/Então, Fora de
      escopo, Decisões com as respostas do dono).
- [ ] `specs/ROADMAP.md` com fases → versões e tarefas ligadas a `FR`/`AC`.

## 3. GitHub

- [ ] Labels: `épico`, `fase-0..N`, `tipo:feature|docs|ci|teste|chore` ou `bug` (label padrão do GitHub), `P1..P3`
      (criadas pela primeira issue que as usar, ou explicitamente).
- [ ] Um épico por fase, com a ordem sugerida de revisão.
- [ ] Tickets: criar a issue → anexar como sub-issue do épico.

## 4. Qualidade

- [ ] CI com os gates de [quality-gates.md](quality-gates.md): lint, testes em todos os SOs,
      race, cobertura, smoke do artefato real, cross-build, vulnerabilidades, ensaio de release,
      docs (markdownlint, links, comandos).
- [ ] Alvo local equivalente (`make ci` ou similar) com os mesmos comandos e limites.
- [ ] Golden files para saídas geradas e alvo para regravá-los.

## 5. Guia prático

- [ ] Templates de issue (tarefa, bug) com Contexto / O que fazer / Critérios de aceite /
      Spec(s); bug pede versão, SO, comando, reprodução mínima, esperado × obtido e lembra de
      não publicar dados pessoais.
- [ ] Template de PR começando com `Closes # · Épico # · Spec`.
- [ ] `CONTRIBUTING.md` resumindo o processo na prática (fluxo, branches, commits, PRs,
      revisão, fechamento, ambiente e comandos).
- [ ] `CODEOWNERS`.
- [ ] (Opcional) spec de processo própria do repo, adaptando [process.md](process.md) e
      fixando os valores locais (labels, comandos, fases).

## 6. Release

- [ ] `CHANGELOG.md` em Keep a Changelog com `[Unreleased]`.
- [ ] Workflow de release por tag `v*` que chama o CI completo antes de publicar com checksums.

## 7. Pedir ao dono

- [ ] **(dono)** Proteção da `main`: CI obrigatório + revisão de Code Owners.
- [ ] **(dono)** Política de merge: merge commit para PRs de fase com empilhados; squash permitido nos demais.
- [ ] **(dono)** Permissões de Actions e segredos necessários para a release.
