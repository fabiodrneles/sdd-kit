---
description: Publica o comentário "Estado da fase" no épico aberto (PRs, CI, decisões, próximo passo)
---

Use a skill `sdd-delivery` para atualizar o **estado da fase**.

1. Rode `sh scripts/sdd-phase-status.sh` (se existir): ele monta o comentário com a tabela ticket → PR → CI, as decisões pendentes e o próximo passo. Revise, acrescente os conflitos previstos e publique com `--post` (ou `--next "texto"` para ajustar o próximo passo). Sem o script, monte o mesmo comentário à mão.
2. O comentário se chama "Estado da fase — AAAA-MM-DD".
3. Responda no chat com um resumo de três linhas: feito, falta, bloqueia.
