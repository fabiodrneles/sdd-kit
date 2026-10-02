---
description: Cria o épico de uma fase do ROADMAP com os tickets como sub-issues
argument-hint: "<número da fase>"
---

Use a skill `sdd-delivery` para criar o **épico da Fase $ARGUMENTS** do `specs/ROADMAP.md`.

1. Rode `sh scripts/sdd-epic.sh --dry-run $ARGUMENTS` (se existir) e revise o plano. Depois rode sem `--dry-run`: ele cria o épico (labels `épico`, `fase-N`) e um ticket por tarefa como sub-issue. Para um corpo de ticket mais rico, use `--tickets DIR` com `DIR/Tn.md`.
2. Sem o script: crie o épico com objetivo, ordem sugerida de revisão e critério de pronto, e um ticket por tarefa (Contexto, O que fazer, Critérios de aceite, Spec(s), Épico; labels `fase-N`, `tipo:*`, `P1-3`). Só então anexe cada um como sub-issue.
3. Atualize o corpo do épico com os números e publique o primeiro comentário "Estado da fase".
