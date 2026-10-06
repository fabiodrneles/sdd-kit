# Medições de custo

Resultados brutos do `sdd-report.sh` e do `scripts/benchmark.sh`, sem LLM na medição. Ficam fora do README do projeto até o kit ser mais barato que um `claude` comum no benchmark com e sem sdd-kit.

| Medição | Resultado |
|---|---|
| Relé contra uma sessão longa de trabalho (Fase 12, com PRs, CI e conversa) | 0,40 milhão contra 6,50 milhões de tokens por ticket |
| Agente enxuto contra o agente padrão, no mesmo ticket e com o mesmo resultado | 166 mil contra 528 mil tokens |
| Com e sem sdd-kit, 3 tarefas pequenas ([`result-3.json`](result-3.json), [`tasks.md`](tasks.md)) | 375 mil contra 533 mil tokens |
| Com e sem sdd-kit, 10 tarefas pequenas ([`result-10.json`](result-10.json), [`tasks-10.md`](tasks-10.md)) | 1,42 milhão contra 1,16 milhão de tokens |

Com 10 tarefas entregues de uma vez, cada agente novo gasta cerca de 8 chamadas reconhecendo o código, e a sessão comum paga isso uma vez só. Para refazer: `sh scripts/benchmark.sh --tasks docs/benchmark/tasks-10.md`.
