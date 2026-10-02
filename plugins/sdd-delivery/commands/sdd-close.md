---
description: Abre o PR de fechamento da fase (status das specs, ROADMAP e CHANGELOG) para o dono criar a tag
argument-hint: "<versão, ex.: v0.2.0>"
---

Use a skill `sdd-delivery` para **fechar a fase** da versão $ARGUMENTS.

1. Confirme que todos os PRs dos tickets do épico foram mergeados; se faltar algum, liste-os e **pare**.
2. Abra o PR de fechamento. Use `sh scripts/sdd-mark.sh close $ARGUMENTS` (se existir): ele marca o ROADMAP, move as specs para `Done` ou `In Progress` e abre `[X.Y.Z] - AAAA-MM-DD` no CHANGELOG, sem risco de esvaziar arquivos. Revise os avisos e o texto do CHANGELOG. Rode `sh scripts/sdd-check.sh --strict` se a fase exigir rastreabilidade completa.
3. Em projetos Go com GoReleaser, rode `sh scripts/sdd-release-check.sh pre $ARGUMENTS` antes de pedir a tag: ele simula o release num clone descartável.
4. **Pare**: merge e tag são do dono. Depois da tag, `sh scripts/sdd-release-check.sh post $ARGUMENTS` confere a release publicada.
