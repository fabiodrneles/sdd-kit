#!/bin/sh
# Testes da spec 003: estrutura, marcadores e workflows do template (AC-1, AC-2).
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root/template"
fail=0
err() { echo "FALHOU: $*" >&2; fail=1; }

for lang in go node java python rust dotnet; do
  for f in Makefile .github/workflows/ci.yml .github/dependabot.yml .claude/hooks/session-start.sh; do
    [ -f "$lang/$f" ] || err "template/$lang/$f não existe"
  done
  grep -q '^ci:' "$lang/Makefile" || err "template/$lang/Makefile sem alvo ci"
  grep -q 'make ci' "$lang/.github/workflows/ci.yml" || err "template/$lang: o CI não roda make ci"
done

# Outros agentes leem AGENTS.md (spec 006 FR-5).
# 010 AC-3: o template Java também atende projetos Gradle.
[ -f java/scripts/jacoco.init.gradle ] || err "template/java/scripts/jacoco.init.gradle não existe"
grep -q 'build.gradle' java/Makefile || err "template/java/Makefile não detecta Gradle"

# 010 AC-4: o workflow de release vai para toda linguagem e aceita release-as e ref.
rt=common/.github/workflows/release-tag.yml
if [ -f "$rt" ]; then
  for input in release-as ref; do
    grep -q "^      $input:" "$rt" || err "$rt sem o input $input"
  done
  grep -q 'gh release create' "$rt" || err "$rt não publica a release"
else
  err "$rt não existe"
fi

# 010 AC-5: o CI de um projeto Node adotado (validado no qa-portfolio) roda lint,
# testes com cobertura mínima e build.
# shellcheck disable=SC2016 # o $(COVERAGE_MIN) é literal, como no Makefile
for step in 'npm run lint' 'check-coverage --lines $(COVERAGE_MIN)' 'npm test' 'npm run build'; do
  grep -qF "$step" node/Makefile || err "template/node/Makefile: o make ci não roda '$step'"
done

# 015 AC-4/AC-7: o PR de fechamento abre quando uma issue fecha, desligável por SDD_ENGINE.
pd=common/.github/workflows/sdd-on-phase-done.yml
[ -f "$pd" ] || err "$pd não existe"
for t in 'issues:' 'types: [closed]' "vars.SDD_ENGINE != 'off'" 'sdd-on-phase-done.sh' 'SDD_ENGINE_TOKEN || github.token'; do
  grep -qF "$t" "$pd" || err "$pd sem '$t'"
done
if grep -q 'pull_request_target' "$pd"; then err "$pd usa pull_request_target"; fi

# 015 AC-3/AC-7: o resumo do CI roda por workflow_run, só para PRs do próprio repositório,
# e é desligável por SDD_ENGINE.
cs=common/.github/workflows/sdd-ci-summary.yml
[ -f "$cs" ] || err "$cs não existe"
for t in 'workflow_run:' 'SDD_ENGINE' 'head_repository.full_name == github.repository' 'sdd-ci-comment.sh'; do
  grep -qF "$t" "$cs" || err "$cs sem '$t'"
done
if grep -q 'pull_request_target' "$cs"; then err "$cs usa pull_request_target"; fi
if grep -q 'ref:' "$cs"; then err "$cs faz checkout de outra ref (código do PR)"; fi

[ -f common/AGENTS.md ] || err "template/common/AGENTS.md não existe"
grep -q 'make ci' common/AGENTS.md || err "template/common/AGENTS.md não cita make ci"

# O template habilita a skill pelo plugin do kit (spec 003 FR-1).
jq -e '.extraKnownMarketplaces["sdd-kit"].source.repo == "fabiodrneles/sdd-kit"
  and .enabledPlugins["sdd-delivery@sdd-kit"] == true' common/.claude/settings.json >/dev/null 2>&1 ||
  err "template/common/.claude/settings.json não habilita sdd-delivery@sdd-kit"

# 003 AC-2: nada específico do projeto de origem.
if grep -rniI 'cv-craft' . >/dev/null; then
  err "o template cita cv-craft: $(grep -rlniI 'cv-craft' . | tr '\n' ' ')"
fi

# Só os marcadores conhecidos (spec 003 FR-3).
unknown="$(grep -rhoI '{{[A-Z_]*}}' . | sort -u | grep -vxE '\{\{(PROJECT|OWNER|REPO)\}\}' || true)"
[ -z "$unknown" ] || err "marcadores desconhecidos: $unknown"

# 011 AC-3: blocos marcados do README rodam no CI de docs; os outros, não.
grep -q 'sh scripts/doc-commands.sh' common/.github/workflows/docs.yml || err "docs.yml não roda o doc-commands.sh"
dc="$(mktemp -d)"
# shellcheck disable=SC2016 # crases literais do markdown
printf '<!-- doc-commands -->\n```bash\ntouch ran\n```\n\n```bash\nexit 1\n```\n' > "$dc/README.md"
(cd "$dc" && sh "$root/template/common/scripts/doc-commands.sh" > out 2>&1) || err "doc-commands falhou com blocos bons: $(cat "$dc/out")"
[ -f "$dc/ran" ] || err "doc-commands não rodou o bloco marcado"
# shellcheck disable=SC2016 # crases literais do markdown
printf '\n<!-- doc-commands -->\n```bash\nfalse\n```\n' >> "$dc/README.md"
if (cd "$dc" && sh "$root/template/common/scripts/doc-commands.sh" > out 2>&1); then
  err "doc-commands aceitou um bloco que falha"
fi
grep -q '^FALHOU README.md:12' "$dc/out" || err "doc-commands não apontou o bloco que falhou: $(cat "$dc/out")"
# 011 AC-3: make linkcheck em toda linguagem; com o lychee, um link quebrado reprova.
for lang in go node java python rust dotnet; do
  grep -q '^linkcheck:' "$lang/Makefile" || err "template/$lang/Makefile sem alvo linkcheck"
done
if command -v lychee >/dev/null; then
  sed -e 's/{{OWNER}}/acme/g' -e 's/{{REPO}}/demo/g' common/lychee.toml > "$dc/lychee.toml"
  cp go/Makefile "$dc/"
  rm "$dc/README.md"; printf '# X\n\n[ok](ok.md)\n' > "$dc/README.md"; printf '# Ok\n' > "$dc/ok.md"
  (cd "$dc" && make -s linkcheck > out 2>&1) || err "make linkcheck reprovou links bons: $(cat "$dc/out")"
  printf '\n[quebrado](nao-existe.md)\n' >> "$dc/README.md"
  if (cd "$dc" && make -s linkcheck > out 2>&1); then err "make linkcheck aceitou um link quebrado"; fi
else
  echo "tests/template.sh: lychee ausente, make linkcheck não executado aqui"
fi
rm -rf "$dc"

# 003 AC-1: workflows válidos.
command -v actionlint >/dev/null || { echo "actionlint não instalado (veja .claude/hooks/session-start.sh)" >&2; exit 1; }
# shellcheck disable=SC2046 # lista de arquivos sem espaços
actionlint $(find . -path '*/.github/workflows/*.yml') || fail=1

[ "$fail" -eq 0 ] && echo "tests/template.sh ok"
exit "$fail"
