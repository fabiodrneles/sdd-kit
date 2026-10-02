#!/bin/sh
# Testes da spec 004 (AC-1..AC-5) para scripts/adopt.sh. Com pwsh no PATH,
# também compara a árvore gerada por scripts/adopt.ps1 (AC-6).
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
adopt="$root/scripts/adopt.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail() { echo "FALHOU: $*" >&2; exit 1; }
snapshot() { (cd "$1" && find . -type f -exec cksum {} + | LC_ALL=C sort); }
opts="--project Demo --owner acme --repo demo"

for lang in go node java python rust dotnet; do
  # 004 AC-1: diretório vazio recebe common + linguagem, sem marcadores.
  d="$tmp/$lang-empty"; mkdir "$d"
  # shellcheck disable=SC2086 # opts são palavras separadas de propósito
  sh "$adopt" --lang "$lang" $opts "$d" > "$tmp/out"
  want="$( (cd "$root/template/common" && find . -type f; cd "$root/template/$lang" && find . -type f) | wc -l)"
  got="$(cd "$d" && find . -type f | wc -l)"
  want=$((want + 1)) # .sdd-kit.json
  [ "$want" -eq "$got" ] || fail "$lang: esperava $want arquivos, gerou $got"
  # 005 FR-2: estado com versão, linguagem e um hash por arquivo gerenciado.
  jq -e --arg l "$lang" '.kit == "sdd-kit" and .lang == $l and (.files | length) == '"$((want - 1))" "$d/.sdd-kit.json" >/dev/null ||
    fail "$lang: .sdd-kit.json inválido"
  h="$(jq -r '.files["CLAUDE.md"]' "$d/.sdd-kit.json")"
  [ "$h" = "$( (sha256sum "$d/CLAUDE.md" 2>/dev/null || shasum -a 256 "$d/CLAUDE.md") | cut -d' ' -f1)" ] ||
    fail "$lang: hash do CLAUDE.md no estado não confere"
  ! grep -rq '{{[A-Z_]*}}' "$d" || fail "$lang: sobrou marcador: $(grep -rl '{{[A-Z_]*}}' "$d")"
  grep -q 'Demo' "$d/CLAUDE.md" || fail "$lang: {{PROJECT}} não substituído"
  grep -q '@acme' "$d/.github/CODEOWNERS" || fail "$lang: {{OWNER}} não substituído"
  grep -q 'Demo' "$d/AGENTS.md" || fail "$lang: AGENTS.md sem {{PROJECT}} substituído (006 AC-4)"
  [ -x "$d/.claude/hooks/session-start.sh" ] || fail "$lang: hook perdeu permissão de execução"

  # 004 AC-3: segunda execução não muda nada.
  before="$(snapshot "$d")"
  # shellcheck disable=SC2086
  sh "$adopt" --lang "$lang" $opts "$d" > "$tmp/out"
  [ "$before" = "$(snapshot "$d")" ] || fail "$lang: segunda execução mudou arquivos"
  grep -q ': 0 criados' "$tmp/out" || fail "$lang: segunda execução criou arquivos"
done

# 004 AC-2: arquivos existentes ficam intactos e aparecem como ignorados.
d="$tmp/existing"; mkdir "$d"
printf 'meu readme\n' > "$d/README.md"
printf 'meu claude\n' > "$d/CLAUDE.md"
# shellcheck disable=SC2086
sh "$adopt" --lang go $opts "$d" > "$tmp/out"
[ "$(cat "$d/CLAUDE.md")" = "meu claude" ] || fail "CLAUDE.md existente foi sobrescrito"
[ "$(cat "$d/README.md")" = "meu readme" ] || fail "README.md existente foi alterado"
grep -qx 'ignorado (já existe): CLAUDE.md' "$tmp/out" || fail "CLAUDE.md não aparece como ignorado"
jq -e '.files | has("CLAUDE.md") | not' "$d/.sdd-kit.json" >/dev/null || fail "arquivo do repositório entrou no estado como gerenciado"
# --force sobrescreve.
# shellcheck disable=SC2086
sh "$adopt" --lang go $opts --force "$d" > "$tmp/out"
grep -q 'Demo' "$d/CLAUDE.md" || fail "--force não sobrescreveu CLAUDE.md"

# 004 AC-4: linguagem inválida e argumento ausente saem com 2 sem escrever nada.
d="$tmp/invalid"; mkdir "$d"
for args in "--lang cobol $opts" "$opts" "--lang"; do
  set +e
  # shellcheck disable=SC2086
  sh "$adopt" $args "$d" > /dev/null 2>&1
  code=$?
  set -e
  [ "$code" -eq 2 ] || fail "'$args' saiu com $code, esperava 2"
  [ -z "$(ls -A "$d")" ] || fail "'$args' escreveu no destino"
done

# 004 AC-5: --dry-run não escreve nada.
# shellcheck disable=SC2086
sh "$adopt" --lang python $opts --dry-run "$d" > "$tmp/out"
[ -z "$(ls -A "$d")" ] || fail "--dry-run escreveu no destino"
grep -q '^(dry-run) ' "$tmp/out" || fail "--dry-run sem resumo"

# FR-6: dono e repositório deduzidos do remote origin.
d="$tmp/remote"; mkdir "$d"
git -C "$d" init -q
git -C "$d" remote add origin git@github.com:octo/widget.git
sh "$adopt" --lang node "$d" > /dev/null
grep -q '@octo' "$d/.github/CODEOWNERS" || fail "owner não deduzido do remote ssh"
grep -q 'octo/widget' "$d/.github/ISSUE_TEMPLATE/config.yml" || fail "repo não deduzido do remote ssh"

# 010 FR-2, AC-2: --skeleton cria o projeto mínimo fora do estado gerenciado e
# nunca sobrescreve código do projeto, nem com --force.
sk="$tmp/skel"; mkdir "$sk"
sh "$root/scripts/adopt.sh" --lang go --project demo --owner acme --repo demo --skeleton "$sk" > /dev/null
[ -f "$sk/hello_test.go" ] || fail "--skeleton não criou o teste"
grep -q 'github.com/acme/demo' "$sk/go.mod" || fail "--skeleton não aplicou dono e repositório"
if grep -q 'hello.go' "$sk/.sdd-kit.json"; then fail "--skeleton registrou o esqueleto no estado"; fi
echo 'meu código' > "$sk/hello.go"
sh "$root/scripts/adopt.sh" --lang go --project demo --owner acme --repo demo --skeleton --force "$sk" > /dev/null
[ "$(cat "$sk/hello.go")" = 'meu código' ] || fail "--skeleton --force sobrescreveu código do projeto"

# 011 AC-4: com a tag v1.4.0, o ROADMAP criado começa na v1.5.0; sem tag, na
# v0.1.0. O estado guarda o hash do ROADMAP do kit (a sincronização não o toma
# por alterado pelo kit).
tg="$tmp/tagged"; mkdir "$tg"
git -C "$tg" init -q
git -C "$tg" -c user.email=t@t -c user.name=t commit -q --allow-empty -m init
git -C "$tg" tag v1.4.0; git -C "$tg" tag v1.10.0-rc1
cp -R "$tg" "$tmp/tagged-ps"
# shellcheck disable=SC2086
sh "$adopt" --lang go $opts "$tg" > /dev/null
[ "$(grep -c '^## Fase [123] .*.v1\.[567]\.0.$' "$tg/specs/ROADMAP.md")" -eq 3 ] ||
  fail "ROADMAP não começa na v1.5.0 depois da tag v1.4.0: $(grep '^## Fase' "$tg/specs/ROADMAP.md")"
grep -q '^## Fase 1 .*.v1\.5\.0.$' "$tg/specs/ROADMAP.md" || fail "Fase 1 não é a v1.5.0"
[ "$(jq -r '.files["specs/ROADMAP.md"]' "$tg/.sdd-kit.json")" = "$( (sha256sum "$root/template/common/specs/ROADMAP.md" 2>/dev/null || shasum -a 256 "$root/template/common/specs/ROADMAP.md") | cut -d' ' -f1)" ] ||
  fail "o estado guardou o hash do ROADMAP ajustado, não o do kit"
grep -q '^## Fase 1 .*.v0\.1\.0.$' "$tmp/go-empty/specs/ROADMAP.md" || fail "sem tag, a Fase 1 não é a v0.1.0"
if command -v pwsh >/dev/null; then
  pwsh -NoProfile -File "$root/scripts/adopt.ps1" --lang go --project Demo --owner acme --repo demo "$tmp/tagged-ps" > /dev/null
  diff -r -x .git "$tg" "$tmp/tagged-ps" >&2 || fail "adopt.ps1 numerou o ROADMAP diferente"
fi

# 011 AC-3: sem LICENSE, a adoção avisa e não cria uma (a licença é do dono);
# com LICENSE, não avisa nem mexe nela.
nolic="$tmp/unlicensed"; mkdir "$nolic"
# shellcheck disable=SC2086
sh "$adopt" --lang python $opts "$nolic" > "$tmp/out-lic"
grep -q '^aviso: sem LICENSE' "$tmp/out-lic" || fail "sem aviso de LICENSE ausente"
if [ -e "$nolic/LICENSE" ]; then fail "a adoção criou um LICENSE"; fi
lic="$tmp/licensed"; mkdir "$lic"; echo 'minha licença' > "$lic/LICENSE"
# shellcheck disable=SC2086
sh "$adopt" --lang go $opts "$lic" > "$tmp/out-lic"
! grep -q '^aviso: sem LICENSE' "$tmp/out-lic" || fail "avisou LICENSE ausente com LICENSE presente"
[ "$(cat "$lic/LICENSE")" = 'minha licença' ] || fail "a adoção mexeu no LICENSE"
if command -v pwsh >/dev/null; then
  pwsh -NoProfile -File "$root/scripts/adopt.ps1" --lang python --project Demo --owner acme --repo demo --dry-run "$nolic" > "$tmp/out-ps"
  grep -q '^aviso: sem LICENSE' "$tmp/out-ps" || fail "adopt.ps1 sem aviso de LICENSE ausente"
  pwsh -NoProfile -File "$root/scripts/adopt.ps1" --lang go --project Demo --owner acme --repo demo --dry-run "$lic" > "$tmp/out-ps"
  ! grep -q '^aviso: sem LICENSE' "$tmp/out-ps" || fail "adopt.ps1 avisou LICENSE ausente com LICENSE presente"
fi

# 011 AC-1: a adoção Node avisa lockfile ausente ou fora de sincronia e projeto
# sem testes, sem falhar; com o projeto em ordem, não avisa. Com pwsh, o
# adopt.ps1 avisa igual.
n="$tmp/node-warn"; mkdir "$n"
: > "$n/LICENSE"
warns() {
  # shellcheck disable=SC2086
  sh "$adopt" --lang node $opts --dry-run "$n" > "$tmp/out" || fail "adoção Node com avisos falhou"
  grep '^aviso:' "$tmp/out" > "$tmp/warns" || true
  if command -v pwsh >/dev/null; then
    # shellcheck disable=SC2086
    pwsh -NoProfile -File "$root/scripts/adopt.ps1" --lang node $opts --dry-run "$n" > "$tmp/out-ps"
    [ "$(cat "$tmp/warns")" = "$(grep '^aviso:' "$tmp/out-ps" || true)" ] ||
      fail "adopt.ps1 avisa diferente do adopt.sh: $(cat "$tmp/out-ps")"
  fi
}
printf '{"name":"x","scripts":{"test":"echo \\"Error: no test specified\\" && exit 1"}}\n' > "$n/package.json"
warns
grep -q '^aviso: package.json sem script "test"' "$tmp/warns" || fail "sem aviso de projeto sem testes"
grep -q '^aviso: sem package-lock.json' "$tmp/warns" || fail "sem aviso de lockfile ausente"
cp "$root/tests/fixtures/node/package.json" "$root/tests/fixtures/node/package-lock.json" "$n/"
warns
[ ! -s "$tmp/warns" ] || fail "projeto Node em ordem recebeu aviso: $(cat "$tmp/warns")"
jq '.devDependencies = {"c8": "^10.0.0"}' "$root/tests/fixtures/node/package.json" > "$n/package.json"
warns
grep -qx 'aviso: package-lock.json fora de sincronia .*' "$tmp/warns" || fail "sem aviso de lockfile fora de sincronia"
[ "$(wc -l < "$tmp/warns")" -eq 1 ] || fail "avisos demais: $(cat "$tmp/warns")"

# 004 AC-6: a versão PowerShell gera a mesma árvore.
if command -v pwsh >/dev/null; then
  for lang in go node java python rust dotnet; do
    a="$tmp/$lang-empty" b="$tmp/$lang-ps"; mkdir "$b"
    pwsh -NoProfile -File "$root/scripts/adopt.ps1" --lang "$lang" --project Demo --owner acme --repo demo "$b" > /dev/null
    [ "$(snapshot "$a")" = "$(snapshot "$b")" ] || { diff -r "$a" "$b" >&2 || true; fail "$lang: adopt.ps1 gerou árvore diferente"; }
  done
  for lang in go node java python rust dotnet; do
    a="$tmp/$lang-skel-sh" b="$tmp/$lang-skel-ps"; mkdir "$a" "$b"
    sh "$root/scripts/adopt.sh" --lang "$lang" --project demo --owner acme --repo demo --skeleton "$a" > /dev/null
    pwsh -NoProfile -File "$root/scripts/adopt.ps1" --lang "$lang" --project demo --owner acme --repo demo --skeleton "$b" > /dev/null
    [ "$(snapshot "$a")" = "$(snapshot "$b")" ] || { diff -r "$a" "$b" >&2 || true; fail "$lang: adopt.ps1 --skeleton gerou árvore diferente"; }
  done
  echo "tests/adopt.sh: adopt.ps1 equivalente"
else
  echo "tests/adopt.sh: pwsh ausente, equivalência com adopt.ps1 não verificada aqui (o CI verifica)"
fi
echo "tests/adopt.sh ok"
