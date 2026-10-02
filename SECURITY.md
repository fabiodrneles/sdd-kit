# Política de segurança

## Versões com suporte

Correções de segurança saem na última versão publicada ([releases](https://github.com/fabiodrneles/sdd-kit/releases)).

## Como reportar uma vulnerabilidade

**Não abra uma issue pública.** Use o reporte privado do GitHub: aba **Security → Report a vulnerability** deste repositório. Se a opção não estiver disponível, contate o mantenedor, [@fabiodrneles](https://github.com/fabiodrneles), pelos contatos do perfil.

Inclua: o que é afetado (script de adoção, sincronização, workflow do template, skill), como reproduzir e o impacto. Você recebe uma resposta em até 7 dias.

## Escopo que mais importa

- `scripts/adopt.sh`, `scripts/adopt.ps1` e `template/common/scripts/sdd-sync.sh`: rodam no repositório e na máquina de quem adota o kit.
- Workflows do template: rodam com permissões de escrita no repositório adotado (o de sincronização abre PRs).
