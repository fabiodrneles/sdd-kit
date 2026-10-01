---
description: Pega o próximo ticket do épico aberto e o leva até um PR com CI verde
argument-hint: "[número do ticket]"
---

Use a skill `sdd-delivery` para executar o **próximo ticket**. Ticket específico, se informado: $ARGUMENTS

1. Leia o comentário "Estado da fase" mais recente do épico aberto e escolha o próximo ticket na ordem do épico.
2. Crie a branch `<tipo>/<nº>-<descrição>`, escreva os testes a partir dos critérios de aceite (citando `NNN AC-n`) e depois o código.
3. Rode a verificação local completa (`make ci`), faça a checagem de mutação dos testes novos e releia o diff.
4. Abra o PR (`Closes #N · Épico #M · Spec NNN`), acompanhe o CI até ficar verde e atualize o "Estado da fase".
