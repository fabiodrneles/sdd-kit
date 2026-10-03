# 005 — Sincronização dos repositórios

- **Prioridade:** P1
- **Status:** Done — entregue na `v0.1.0`
- **Código afetado:** `template/common/.github/workflows/sdd-sync.yml`, `scripts/`
- **Resolve:** M1

## Contexto

D3 (a): melhorias na skill e nos modelos precisam chegar aos repositórios que já adotaram o kit, sem trabalho manual e sem sobrescrever personalizações (constituição, princípio 2).

## Requisitos funcionais

- **FR-1** O template MUST incluir um workflow semanal (e `workflow_dispatch`) que compara os arquivos gerenciados pelo kit com a última release e abre (ou atualiza) um único PR com as diferenças.
- **FR-2** A lista de arquivos gerenciados e a versão adotada MUST ficar num arquivo de estado (`.sdd-kit.json`) escrito pela adoção.
- **FR-3** Um arquivo gerenciado que o repositório alterou MUST NOT ser sobrescrito em silêncio: o PR traz a nova versão e a descrição lista o conflito.
- **FR-4** Sem diferenças, o workflow MUST NOT abrir PR.

## Critérios de aceite

- **AC-1** Dado um repositório na versão anterior sem alterações locais, quando o workflow roda, então abre um PR com a atualização.
- **AC-2** Dado um repositório já na última versão, quando o workflow roda, então nenhum PR é aberto.
- **AC-3** Dado um arquivo gerenciado alterado localmente, quando o workflow roda, então o PR lista o arquivo como conflito.

## Fora de escopo

- Sincronizar arquivos que o repositório criou por conta própria.

## Decisões

- D3 — PR automático semanal, entregue na Fase 2 (`v0.2.0`).
- Revisado na implementação (#28):
  - `template/common/scripts/sdd-sync.sh` gera o template da versão alvo com os valores do `.sdd-kit.json` (que passa a guardar `project`, `owner` e `repo`) e compara arquivo a arquivo;
  - o workflow `sdd-sync.yml` abre ou atualiza o PR da branch `sdd-kit/sync`;
  - arquivo que o kit não mudou fica intocado, mesmo se o repositório o alterou;
  - arquivo alterado localmente que o kit também mudou recebe a versão nova no PR e aparece em "Conflitos" na descrição, para o dono restaurar o que quiser no próprio PR;
  - o estado guarda o hash do conteúdo do kit, não do arquivo local, para que uma alteração local continue detectável;
  - o repositório precisa permitir que o GitHub Actions crie PRs (Settings → Actions → General).

## Mudanças

### v1.3.1

- MODIFIED FR-1 — o `sdd-sync.sh` roda uma cópia de si mesmo, porque a sincronização o atualiza enquanto ele roda (achado na sincronização do sdd-kit-demo, v1.3.1).
