#!/bin/sh
# Testes da sincronização (spec 005 AC-1 a AC-5) com um kit local.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }
snapshot() { (cd "$1" && find . -type f -exec cksum {} + | LC_ALL=C sort); }

kit="$tmp/kit"
mkdir "$kit"
cp -R "$root/scripts" "$root/template" "$kit/"
repo="$tmp/repo"
mkdir "$repo"
printf 'meu\n' > "$repo/own.txt"
SDD_KIT_REF=v0.0.1 sh "$kit/scripts/adopt.sh" --lang go --project Demo --owner acme --repo demo "$repo" > /dev/null
cd "$repo"

# 005 AC-2: já na versão alvo, nada muda.
before="$(snapshot .)"
sh scripts/sdd-sync.sh --kit "$kit" --version v0.0.1 | grep -q 'nada a fazer' || fail "mesma versão deveria não fazer nada"
[ "$before" = "$(snapshot .)" ] || fail "mesma versão mudou arquivos"

# A própria sdd-sync.sh muda na versão nova (o sh lê o script aos poucos).
{ head -n 1 "$kit/template/common/scripts/sdd-sync.sh"; for i in $(seq 1 40); do echo "# linha nova $i, que desloca o resto do script"; done
  tail -n +2 "$kit/template/common/scripts/sdd-sync.sh"; } > "$tmp/sync.new" && cp "$tmp/sync.new" "$kit/template/common/scripts/sdd-sync.sh"
# Nova versão do kit: CONTRIBUTING e CLAUDE mudam, surgem NEW.txt e own.txt.
printf 'linha nova\n' >> "$kit/template/common/CONTRIBUTING.md"
printf 'kit novo\n' >> "$kit/template/common/CLAUDE.md"
printf 'novo\n' > "$kit/template/common/NEW.txt"
printf 'do kit\n' > "$kit/template/common/own.txt"
# Alterações locais: CLAUDE.md (o kit também mudou) e CODEOWNERS (o kit não mudou).
printf 'local\n' >> CLAUDE.md
printf 'local\n' >> .github/CODEOWNERS
codeowners="$(cat .github/CODEOWNERS)"

sh scripts/sdd-sync.sh --kit "$kit" --version v0.0.2 --summary "$tmp/s.md"

# 005 AC-1: arquivo sem alteração local é atualizado; arquivo novo é criado; estado na versão nova.
grep -q 'linha nova' CONTRIBUTING.md || fail "CONTRIBUTING.md não foi atualizado"
[ -f NEW.txt ] || fail "NEW.txt não foi criado"
[ "$(jq -r .version .sdd-kit.json)" = v0.0.2 ] || fail "estado não foi para v0.0.2"
jq -e '.files["NEW.txt"] and .project == "Demo"' .sdd-kit.json > /dev/null || fail "estado sem NEW.txt ou sem project"
# Arquivo do repositório (não gerenciado) fica intocado, e alteração local em arquivo
# que o kit não mudou também.
[ "$(cat own.txt)" = meu ] || fail "own.txt do repositório foi sobrescrito"
jq -e '.files | has("own.txt") | not' .sdd-kit.json > /dev/null || fail "own.txt virou gerenciado"
[ "$(cat .github/CODEOWNERS)" = "$codeowners" ] || fail "CODEOWNERS alterado localmente foi sobrescrito sem mudança no kit"

# 005 AC-3: alteração local em arquivo que o kit mudou aparece como conflito.
grep -q 'kit novo' CLAUDE.md || fail "CLAUDE.md não recebeu a versão nova"
grep -qxF -e "- \`CLAUDE.md\`" "$tmp/s.md" || fail "CLAUDE.md não listado como conflito"
if grep -qxF -e "- \`CONTRIBUTING.md\`" "$tmp/s.md"; then fail "CONTRIBUTING.md listado como conflito por engano"; fi

# 012 AC-2: num repositório adotado por uma versão que registrava os modelos do
# projeto no estado, o kit muda o modelo do ROADMAP: o ROADMAP do projeto fica,
# não vira conflito e sai do estado.
h="$( (sha256sum specs/ROADMAP.md 2>/dev/null || shasum -a 256 specs/ROADMAP.md) | cut -d' ' -f1)"
jq --arg h "$h" '.files["specs/ROADMAP.md"] = $h' .sdd-kit.json > "$tmp/state" && cp "$tmp/state" .sdd-kit.json
echo '- [ ] **T9** tarefa do projeto' >> specs/ROADMAP.md
roadmap="$(cat specs/ROADMAP.md)"
printf 'modelo novo\n' >> "$kit/template/seed/specs/ROADMAP.md"
rm CHANGELOG.md
sh scripts/sdd-sync.sh --kit "$kit" --version v0.0.3 --summary "$tmp/s3.md"
[ "$(cat specs/ROADMAP.md)" = "$roadmap" ] || fail "a sincronização mudou o ROADMAP do projeto"
if grep -qF 'specs/ROADMAP.md' "$tmp/s3.md"; then fail "ROADMAP do projeto listado na sincronização: $(cat "$tmp/s3.md")"; fi
jq -e '.files | has("specs/ROADMAP.md") | not' .sdd-kit.json > /dev/null || fail "ROADMAP continuou no estado"
grep -q '^## \[Unreleased\]' CHANGELOG.md || fail "a sincronização não recriou o CHANGELOG.md que faltava"
jq -e '.files | has("CHANGELOG.md") | not' .sdd-kit.json > /dev/null || fail "CHANGELOG.md recriado entrou no estado"

# 005 AC-2 de novo: repetir com a mesma versão não muda nada.
before="$(snapshot .)"
sh scripts/sdd-sync.sh --kit "$kit" --version v0.0.3 > /dev/null
[ "$before" = "$(snapshot .)" ] || fail "segunda sincronização mudou arquivos"
# 005 AC-4: sem a permissão `workflows`, a sincronização atualiza o resto, deixa
# o workflow de fora, o lista e o `--only-workflows` o aplica depois.
printf '# kit novo\n' >> "$kit/template/common/.github/workflows/sdd-sync.yml"
printf 'name: novo\non: push\njobs: {}\n' > "$kit/template/common/.github/workflows/sdd-novo.yml"
printf 'mais\n' >> "$kit/template/common/CONTRIBUTING.md"
wf="$(cat .github/workflows/sdd-sync.yml)"
SDD_SYNC_SKIP_WORKFLOWS=1 sh scripts/sdd-sync.sh --kit "$kit" --version v0.0.4 --summary "$tmp/s4.md"
grep -q '^mais$' CONTRIBUTING.md || fail "AC-4: o resto não foi atualizado"
[ "$(cat .github/workflows/sdd-sync.yml)" = "$wf" ] || fail "AC-4: workflow alterado com SKIP_WORKFLOWS"
[ ! -e .github/workflows/sdd-novo.yml ] || fail "AC-4: workflow novo criado com SKIP_WORKFLOWS"
grep -q 'Workflows não aplicados' "$tmp/s4.md" || fail "AC-4: sem a seção de workflows"
grep -qxF -e "- \`.github/workflows/sdd-sync.yml\`" "$tmp/s4.md" || fail "AC-4: sdd-sync.yml não listado"
grep -qxF -e "- \`.github/workflows/sdd-novo.yml\`" "$tmp/s4.md" || fail "AC-4: sdd-novo.yml não listado"
grep -q -e '--only-workflows' "$tmp/s4.md" || fail "AC-4: sem a instrução --only-workflows"
[ "$(jq -r .version .sdd-kit.json)" = v0.0.4 ] || fail "AC-4: estado não foi para v0.0.4"
other="$(snapshot . | grep -v '\.github/workflows/\|\.sdd-kit\.json')"
sh scripts/sdd-sync.sh --kit "$kit" --version v0.0.4 --only-workflows --summary "$tmp/s5.md"
grep -q 'kit novo' .github/workflows/sdd-sync.yml || fail "AC-4: --only-workflows não atualizou o workflow"
[ -f .github/workflows/sdd-novo.yml ] || fail "AC-4: --only-workflows não criou o workflow novo"
[ "$other" = "$(snapshot . | grep -v '\.github/workflows/\|\.sdd-kit\.json')" ] || fail "AC-4: --only-workflows mexeu em outros arquivos"
[ "$(jq -r .version .sdd-kit.json)" = v0.0.4 ] || fail "AC-4: --only-workflows mexeu no estado"
grep -qxF -e "- \`.github/workflows/sdd-novo.yml\`" "$tmp/s5.md" || fail "AC-4: resumo do --only-workflows sem o workflow"

# 005 AC-5: o workflow deixa a branch enviada, grava o link e avisa, sem falhar.
yml="$root/template/common/.github/workflows/sdd-sync.yml"
grep -q 'SDD_SYNC_TOKEN' "$yml" || fail "AC-5: workflow sem SDD_SYNC_TOKEN"
grep -q '::warning::' "$yml" || fail "AC-5: workflow sem ::warning::"
grep -q 'compare/' "$yml" || fail "AC-5: workflow sem o link de comparação"

echo "tests/sync.sh ok"
