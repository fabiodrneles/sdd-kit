#!/bin/sh
# Testes da sincronização (spec 005 AC-1, AC-2 e AC-3) com um kit local.
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

# 005 AC-2 de novo: repetir com a mesma versão não muda nada.
before="$(snapshot .)"
sh scripts/sdd-sync.sh --kit "$kit" --version v0.0.2 > /dev/null
[ "$before" = "$(snapshot .)" ] || fail "segunda sincronização mudou arquivos"
echo "tests/sync.sh ok"
