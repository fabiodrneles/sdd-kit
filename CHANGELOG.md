# Changelog

Todas as mudanças relevantes deste projeto. Formato [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/), versões [SemVer](https://semver.org/lang/pt-BR/).

## [Unreleased]

## [1.5.0] - 2026-10-05

Scripts que tiram do agente os passos mecânicos de retomar, entregar, preparar o ambiente e fechar a versão, para que a LLM gaste tokens e contexto só com código e decisões.

### Adicionado

- **`scripts/sdd-resume.sh`:** a retomada da sessão num comando. Mostra o checkpoint do épico aberto e entra na branch dele (só com a árvore limpa e sem commits não enviados). Depois lista os PRs abertos com o resumo do CI e as sub-issues abertas do épico. Os hooks de início de sessão passam a chamá-lo (spec 014 FR-2, #139).
- **`scripts/sdd-pr.sh`:** a entrega do ticket num comando. Faz merge da `main` e roda o `make ci`, mostrando só o fim do log em falha. Depois faz push e abre o PR com o template preenchido, ou reaproveita o que já existe. Por fim espera o CI e grava o checkpoint (#138).
- **`scripts/sdd-doctor.sh`:** deixa o ambiente local igual ao do CI quando o hook de sessão não rodou. Cobre as ferramentas nas versões fixadas, o `covdata` do Go, um `golangci-lint` antigo escondendo o certo no PATH e o locale UTF-8. No template Go, o `make lint` passa a preferir o `golangci-lint` do `GOPATH/bin` (spec 010, #144).
- **`scripts/sdd-release.sh`:** o fechamento da versão num comando. A versão vem do [go-release-manager](https://github.com/fabiodrneles/go-release-manager) pelos Conventional Commits, e um X.Y.Z passado à mão só a força, com aviso. O script monta o rascunho do CHANGELOG pelos PRs mesclados, sobe a versão nos arquivos de `.sdd-release` e abre o PR. Depois do merge, `--tag` dispara o *Release tag* (spec 008, #145, #155).

### Corrigido

- O `sdd-pr.sh` espera o CI pelo SHA enviado: logo depois do push, `#PR` ainda podia ler o head anterior e dar um falso verde (#141).

## [1.4.0] - 2026-10-03

Fase 8 do [ROADMAP](specs/ROADMAP.md): checkpoints de retomada (spec 014).

### Adicionado

- **Checkpoints de retomada:** `scripts/sdd-checkpoint.sh save` mantém um comentário de checkpoint no épico aberto (branch, commit, PRs, feito, próximo e bloqueio). O hook de início de sessão o mostra, o hook `Stop` atualiza o estado do git a cada resposta, e o `CLAUDE.md` manda retomar dele e gravar um a cada passo. A sessão pode acabar sem aviso, e a próxima continua sozinha (spec 014, #123).

## [1.3.1] - 2026-10-03

Correções achadas ao sincronizar o [sdd-kit-demo](https://github.com/fabiodrneles/sdd-kit-demo) da `v0.1.0` para a `v1.3.0`.

### Corrigido

- O `sdd-sync.sh` roda uma cópia de si mesmo: a sincronização o atualizava enquanto ele rodava, e o `sh` terminava com `Syntax error` antes de abrir o PR. Quem adotou até a `v1.3.0` roda a cópia uma vez à mão (veja "Atualização automática" no README) (spec 005 FR-1).
- A cobertura do template Go usa `-coverpkg=./...`: um pacote testado só pelos testes de outro contava 0% (no demo, 49% em vez de 88%) (spec 010 FR-1).

## [1.3.0] - 2026-10-02

Fase 7 do [ROADMAP](specs/ROADMAP.md): primeira impressão para quem chega pela divulgação (spec 013).

### Adicionado

- **Demonstração animada no topo do README**, gerada da saída real da adoção e do `make ci` por `docs/demo/demo.sh` (spec 013 FR-2, #109).
- **Perguntas e relatos de uso nas Discussions**, pelo formulário de nova issue; seção "Primeira contribuição" no CONTRIBUTING e issues `good first issue` (spec 013 FR-3, #110).

### Mudado

- "Começar num repositório novo" (pt e en) explica pré-requisitos, o que a adoção copia, a garantia de não sobrescrever, `--dry-run` e `--skeleton`; os comandos do guia rodam no CI do kit (spec 013 FR-1, #108).

## [1.2.0] - 2026-10-02

Fase 6 do [ROADMAP](specs/ROADMAP.md): arquivos do projeto fora da sincronização (spec 012).

### Mudado

- **`specs/` e `CHANGELOG.md` são do projeto:** passam a `template/seed/`, e a adoção os cria quando faltam, nunca os sobrescreve (nem com `--force`) e não os registra no `.sdd-kit.json`. O `sdd-sync` gerencia só o que a adoção da versão alvo registra, então uma mudança do kit nesses modelos não substitui mais o ROADMAP ou o CHANGELOG preenchido. Quem adotou antes tem esses arquivos retirados do estado na segunda sincronização (spec 012 FR-1, FR-2, #100).

### Adicionado

- O CI do kit falha quando a versão mais recente do CHANGELOG não é a do `plugin.json`, o que teria barrado no PR o incidente da `v1.0.0` (spec 012 FR-3, #101).

## [1.1.0] - 2026-10-02

Fase 5 do [ROADMAP](specs/ROADMAP.md): a adoção ajustada pelo uso real (achados do #51, spec 011).

### Adicionado

- **Avisos da adoção Node:** `package-lock.json` ausente ou fora de sincronia com o `package.json` (o `npm ci` do CI falharia) e projeto sem script de teste, sem falhar a adoção (spec 011 FR-1, #86).
- **`scripts/doc-commands.sh`** no template comum: o CI de docs roda os blocos ```` ```bash ```` do README marcados com `<!-- doc-commands -->` (spec 011 FR-3, #88).
- **`make linkcheck`** em toda linguagem, com o lychee (spec 011 FR-3, #88).
- Aviso de `LICENSE` ausente na adoção; a licença continua decisão do dono (spec 011 FR-3, #88).
- **`CHANGELOG.md`** com `[Unreleased]` criado pela adoção, que o `sdd-mark close` exige (spec 011 FR-5, #90).
- **Acessibilidade no template Node:** com `A11Y_PAGES`, o `make ci` roda o axe-core (num DOM do jsdom, sem contraste de cor) e falha em qualquer violação (spec 011 FR-6, #91).

### Mudado

- A adoção numera as fases do ROADMAP criado a partir da última tag `vX.Y.Z` do repositório (ex.: `v1.4.0` → Fase 1 na `v1.5.0`) (spec 011 FR-4, #89).

### Corrigido

- O hook de sessão Go compila o `covdata` que falta na toolchain baixada pelo `GOTOOLCHAIN`; sem ele, `go test -coverprofile ./...` falhava num pacote sem testes (spec 011 FR-2, #87).

## [1.0.0] - 2026-10-02

Fim da Fase 3 do [ROADMAP](specs/ROADMAP.md) (profissional e pronto para a comunidade). O `sdd-check --strict`, os arquivos de comunidade, o estudo de caso, o plugin `sdd-release` e os scripts dos passos mecânicos já saíram na 0.2.0.

### Adicionado

- **Template Rust**: `make ci` com `cargo fmt`, clippy e cobertura do `cargo llvm-cov`, CI, hook, dependabot, esqueleto e e2e (spec 007 FR-4, #48).
- **Template C#/.NET**: `make ci` com `dotnet format`, build sem avisos e cobertura do coverlet, CI, hook, dependabot, esqueleto e e2e (spec 007 FR-4, #49).
- **Regras de economia de uso** na skill, no `CLAUDE.md` do template e no do kit (spec 007 AC-6, #78).

### Mudado

- A skill e os comandos dos plugins estão em inglês; specs, issues e PRs continuam escritos no idioma do dono (spec 007 AC-5, D12, #50).
- Skill revisada com o uso em outros repositórios: numeração pela última tag, CI verde primeiro em repositório existente, PRs do Dependabot, lições de toolchain e lockfile (spec 007 FR-5, #51).

### Corrigido

- O esqueleto e a fixture .NET saem sem BOM UTF-8 (#49).
- A adoção via `curl` e o README baixam o template da versão atual, e não mais da `v0.1.0`, que não tinha Rust, .NET nem `--skeleton`.

## [0.2.0] - 2026-10-02

Fase 4 do [ROADMAP](specs/ROADMAP.md) (esteira de qualidade) e as entregas da Fase 3 já mergeadas; a v1.0.0 fica para o fim da Fase 3.

### Adicionado

- **Cobertura mínima no `make ci`** de todos os templates: falha abaixo de `COVERAGE_MIN` (80 por padrão), com Go `coverprofile`, c8 no Node, JaCoCo no Java e pytest-cov no Python (spec 010 FR-1, #68).
- **Workflow Release tag** em todos os templates: cria a tag (go-release-manager ou `release-as`, no commit `ref`) e publica a release com notas geradas (spec 010 FR-4, #70).
- **Gradle** no template Java, além de Maven, com JaCoCo por init script (spec 010 FR-3, #71).
- **`adopt --skeleton`** (sh e PowerShell): projeto mínimo com um teste num repositório vazio, para o CI nascer verde (spec 010 FR-2, #72).
- **Plugin `sdd-release`**: próxima versão pelo go-release-manager, comparada com a versão do ROADMAP (spec 008, #54, #58).
- **Scripts dos passos mecânicos**: `sdd-ci`, `sdd-mark`, `sdd-phase-status`, `sdd-epic` e `sdd-release-check` (spec 009, #56).
- `sdd-check --strict` no CI do kit e do template, arquivos de comunidade e estudo de caso do cv-craft (spec 007, #45, #46, #47).

### Mudado

- **Incompatível:** adotantes abaixo de 80% de cobertura passam a ter o `make ci` vermelho depois da sincronização; baixe `COVERAGE_MIN` no `Makefile` para manter o comportamento anterior (#68).
- O check de links publica cada link quebrado como anotação e aceita 503 como recusa temporária, como o 429 (#69).

### Corrigido

- `sdd-ci.sh` espera e reporta também os status de commit (ex.: Vercel) (spec 009 AC-7, #67).

## [0.1.0] - 2026-10-01

Primeira versão pública: Fases 1 e 2 do [ROADMAP](specs/ROADMAP.md).

### Adicionado

- Skill `sdd-delivery` como plugin do Claude Code (`/plugin marketplace add fabiodrneles/sdd-kit`) e zip para o claude.ai, publicado com SHA-256 na release (spec 002).
- Comandos `/sdd-analyze`, `/sdd-specs`, `/sdd-epic`, `/sdd-next`, `/sdd-status` e `/sdd-close` (spec 006 FR-3).
- `template/` com `CLAUDE.md`, `AGENTS.md`, `CONTRIBUTING.md`, templates de issue e PR, esqueleto de `specs/`, hook de sessão, plugin habilitado e CI pronto para Go, Node/TS, Java e Python (spec 003).
- Script de adoção em sh e PowerShell, para repositórios novos e existentes, sem sobrescrever nada, idempotente e com `--dry-run` (spec 004).
- `.sdd-kit.json` e sincronização semanal por PR, com conflitos listados para revisão (spec 005).
- `sdd-check`: rastreabilidade AC → teste, status das specs e IDs do ROADMAP (spec 006 FR-2).
- Critérios de aceite em EARS e seção "Mudanças" por spec, que alimenta o CHANGELOG (spec 006 FR-1, FR-4).
- CI do kit em Linux, macOS e Windows, com e2e dos templates nas quatro linguagens.

[Unreleased]: https://github.com/fabiodrneles/sdd-kit/compare/v1.4.0...HEAD
[1.4.0]: https://github.com/fabiodrneles/sdd-kit/compare/v1.3.1...v1.4.0
[1.3.1]: https://github.com/fabiodrneles/sdd-kit/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/fabiodrneles/sdd-kit/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/fabiodrneles/sdd-kit/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/fabiodrneles/sdd-kit/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/fabiodrneles/sdd-kit/compare/v0.2.0...v1.0.0
[0.2.0]: https://github.com/fabiodrneles/sdd-kit/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/fabiodrneles/sdd-kit/releases/tag/v0.1.0
