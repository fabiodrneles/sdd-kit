---
description: Fase de descoberta SDD — analisa e verifica o repositório e propõe as decisões em aberto
argument-hint: "[foco opcional]"
---

Use a skill `sdd-delivery` e execute a **fase de descoberta** neste repositório. Foco adicional, se houver: $ARGUMENTS

1. Leia o código e rode o que o README promete (instalação, comandos, testes), além de casos de borda.
2. Registre em `specs/ANALYSIS.md`: o que foi verificado (comando → resultado), achados por severidade com evidência (`arquivo:linha` ou saída) e decisões em aberto D1..Dn, cada uma com opções e recomendação.
3. Separe o que foi **verificado** do que é **inferido**.
4. Apresente um resumo curto com as decisões e **pare**: nada é implementado antes de o dono responder.
