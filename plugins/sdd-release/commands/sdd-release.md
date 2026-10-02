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
4. Se `next` vier vazio, diga que nenhum commit desde a última versão gera release e **pare**.
5. Proponha o fechamento com `/sdd-close <next>` (skill `sdd-delivery`). Depois do merge do PR de fechamento, o dono cria a tag `<next>`: pelo terminal, ou pelo workflow `release-tag.yml` deste plugin (em `templates/`), se o repositório o adotou.
