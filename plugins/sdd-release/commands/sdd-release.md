---
description: Calcula a próxima versão pelos commits (go-release-manager) e propõe o fechamento da fase com ela
argument-hint: "[canal de pré-release, ex.: rc]"
---

Calcule a **próxima versão** deste repositório e proponha o fechamento da fase com ela. Não crie tags: a tag é do dono.

1. Confira se o `go-release-manager` está instalado (`go-release-manager --version`). Se não estiver, explique ao dono as duas formas de instalar e **pare**:
   - `go install github.com/fabiodrneles/go-release-manager@latest`
   - binário da página <https://github.com/fabiodrneles/go-release-manager/releases>
2. Na `main` atualizada, rode:

   ```text
   go-release-manager create --dry-run --output json
   ```

   Com um canal de pré-release em $ARGUMENTS, acrescente `--pre-release $ARGUMENTS`.
3. Mostre ao dono, em poucas linhas:
   - a versão anterior (`previous`), a próxima (`next`) e o incremento (`increment`);
   - os commits que determinaram o incremento (`git log <previous>..HEAD --oneline`, só os `feat`, `fix` e incompatíveis).
4. Compare com a versão da fase em `specs/ROADMAP.md` (a fase aberta, `→ \`vX.Y.Z\``):
   - **iguais:** siga;
   - **diferentes, ou `next` vazio:** explique ao dono por quê. Por exemplo, num projeto de testes os commits `test:` não geram release pela regra padrão. Proponha uma das saídas e **pare** até ele escolher:
     - seguir o ROADMAP com `--release-as vX.Y.Z` (exige go-release-manager `v1.1.0` ou mais novo);
     - seguir os commits e atualizar o ROADMAP;
     - um `.go-releaserc.yml` que faça os tipos do projeto gerarem release (ex.: `test: minor`).
5. Proponha o fechamento com `/sdd-close <versão>` (skill `sdd-delivery`). Depois do merge do PR de fechamento, a tag é criada:
   - pelo dono, ou pelo agente, se o dono delegou;
   - pelo terminal (`go-release-manager create [--release-as vX.Y.Z] [--ref <commit>]`) ou pelo workflow `release-tag.yml` deste plugin (em `templates/`), que aceita `release-as` e `ref`.
