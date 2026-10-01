---
description: Abre o PR de fechamento da fase (status das specs, ROADMAP e CHANGELOG) para o dono criar a tag
argument-hint: "<versão, ex.: v0.2.0>"
---

Use a skill `sdd-delivery` para **fechar a fase** da versão $ARGUMENTS.

1. Confirme que todos os PRs dos tickets do épico foram mergeados; se faltar algum, liste-os e **pare**.
2. Abra o PR de fechamento: status das specs (`specs/README.md`, cabeçalhos e "Estado atual"), checkboxes do ROADMAP e `CHANGELOG.md` (`[Unreleased]` → `[X.Y.Z] - AAAA-MM-DD`). Rode `sh scripts/sdd-check.sh --strict` se a fase exigir rastreabilidade completa.
3. **Pare**: merge e tag são do dono.
