#!/bin/sh
# Spec 003 AC-4, 007 AC-4 e 010 AC-1: adota o template num projeto mínimo da linguagem e roda o
# `make ci` gerado (o mesmo comando que o ci.yml gerado roda). Precisa da
# linguagem instalada; o CI roda um job por linguagem.
# Uso: scripts/e2e-template.sh go|node|java|java-gradle|python|rust|dotnet [diretório-de-trabalho]
set -eu

lang="${1:?uso: e2e-template.sh go|node|java|java-gradle|python|rust|dotnet}"
# java-gradle: o template Java num projeto Gradle (spec 010 AC-3).
adopt_lang="${lang%-gradle}"
root="$(cd "$(dirname "$0")/.." && pwd)"
work="${2:-$(mktemp -d)}"
d="$work/$lang"
rm -rf "$d"
cp -R "$root/tests/fixtures/$lang" "$d"
sh "$root/scripts/adopt.sh" --lang "$adopt_lang" --project demo --owner acme --repo demo "$d" > /dev/null
cd "$d"
if grep -q '^deps:' Makefile; then make deps; fi
make ci
# 011 AC-6: com A11Y_PAGES, uma violação de acessibilidade derruba o make ci e
# aparece na saída; corrigida, passa. O script do axe não conta na cobertura (o
# make ci acima passou com ele no projeto).
if [ "$lang" = node ]; then
  printf '<!doctype html>\n<html lang="pt">\n<head><title>Demo</title></head>\n<body><main><h1>Demo</h1><img src="logo.png"></main></body>\n</html>\n' > page.html
  if make ci A11Y_PAGES=page.html > a11y.log 2>&1; then
    echo "FALHOU: make ci passou com uma imagem sem alt (axe)" >&2; tail -20 a11y.log >&2; exit 1
  fi
  grep -q 'page.html: image-alt' a11y.log || { echo "FALHOU: make ci não mostrou a violação" >&2; tail -20 a11y.log >&2; exit 1; }
  sed 's/<img src="logo.png">/<img src="logo.png" alt="Logo">/' page.html > page.ok && mv page.ok page.html
  make ci A11Y_PAGES=page.html > a11y.log 2>&1 || { echo "FALHOU: make ci reprovou a página corrigida" >&2; tail -20 a11y.log >&2; exit 1; }
  rm -f page.html a11y.log
fi

# Spec 010 AC-1: um arquivo sem nenhum teste derruba a cobertura abaixo do mínimo,
# e o make ci falha mostrando a cobertura (arquivos não carregados também contam).
case "$lang" in
  go) printf 'package demo\n\n// Untested has no test.\nfunc Untested(n int) int {\n\tfor i := 0; i < n; i++ {\n\t\tn += i\n\t}\n\tif n > 10 {\n\t\treturn n * 2\n\t}\n\treturn n\n}\n' > untested.go ;;
  node) printf 'export function untested(n) {\n  let t = n;\n  for (let i = 0; i < n; i++) t += i;\n  if (t > 10) return t * 2;\n  return t;\n}\n' > untested.js ;;
  python) printf 'def untested(n: int) -> int:\n    t = n\n    for i in range(n):\n        t += i\n    if t > 10:\n        return t * 2\n    return t\n' > demo/untested.py ;;
  rust) # antes do módulo de teste, que o clippy exige no fim do arquivo
    awk '/^#\[cfg\(test\)\]/ && !done { print "pub fn untested(n: i32) -> i32 {\n    let mut t = n;\n    for i in 0..n {\n        t += i;\n    }\n    if t > 10 {\n        return t * 2;\n    }\n    t\n}\n"; done = 1 } { print }' \
      src/lib.rs > src/lib.rs.new && mv src/lib.rs.new src/lib.rs ;;
  dotnet) printf 'namespace Demo;\n\npublic static class Untested\n{\n    public static int Run(int n)\n    {\n        var t = n;\n        for (var i = 0; i < n; i++)\n        {\n            t += i;\n        }\n\n        if (t > 10)\n        {\n            return t * 2;\n        }\n\n        return t;\n    }\n}\n' > src/Demo/Untested.cs ;;
  java | java-gradle) printf 'package demo;\n\npublic final class Untested {\n  private Untested() {}\n\n  public static int run(int n) {\n    int t = n;\n    for (int i = 0; i < n; i++) {\n      t += i;\n    }\n    if (t > 10) {\n      return t * 2;\n    }\n    return t;\n  }\n}\n' > src/main/java/demo/Untested.java ;;
esac
if make ci > coverage-gate.log 2>&1; then
  echo "FALHOU: make ci passou com um arquivo sem testes (a cobertura mínima não é exigida)" >&2
  tail -20 coverage-gate.log >&2
  exit 1
fi
grep -qiE 'cobertura|coverage' coverage-gate.log || { echo "FALHOU: make ci não mostrou a cobertura" >&2; tail -20 coverage-gate.log >&2; exit 1; }

# Spec 010 AC-2: num diretório vazio, --skeleton cria um projeto cujo make ci passa
# sem edição; rodar de novo não sobrescreve nada (nem com --force).
if [ "$lang" = "$adopt_lang" ]; then
  sk="$work/$lang-skeleton"
  rm -rf "$sk"
  mkdir -p "$sk"
  sh "$root/scripts/adopt.sh" --lang "$lang" --project demo --owner acme --repo demo --skeleton "$sk" > /dev/null
  (cd "$sk" && { if grep -q '^deps:' Makefile; then make deps; fi; } && make ci)
  again="$(sh "$root/scripts/adopt.sh" --lang "$lang" --project demo --owner acme --repo demo --skeleton --force "$sk")"
  for f in $(cd "$root/template/skeleton/$lang" && find . -type f | sed 's#^\./##'); do
    printf '%s\n' "$again" | grep -qx "ignorado (já existe): $f" || { echo "FALHOU: --skeleton sobrescreveu $f" >&2; exit 1; }
  done
  echo "e2e-template.sh $lang --skeleton ok"
fi

# 010 FR-1 (#172): com o cache quente, editar o package main (as linhas mudam de posição)
# não pode deixar no perfil blocos da versão antiga, que contam como não cobertos: o
# resultado em cache de outro pacote testado reaproveita metadados velhos do -coverpkg=./...
# (Go 1.25 ou mais novo; GOTOOLCHAIN=auto baixa). A cobertura tem de ser a de um cache frio.
# GOCACHE próprio, para não tocar o do runner. MUTANTE=1 tira o -count=1 do Makefile.
if [ "$lang" = go ]; then
  cv="$work/$lang-cache172"
  rm -rf "$cv"
  cp -R "$d" "$cv"
  rm -f "$cv/untested.go" "$cv/demo.go" "$cv/demo_test.go"
  sed -i.bak 's/^go .*/go 1.25.1/' "$cv/go.mod" && rm "$cv/go.mod.bak"
  mkdir -p "$cv/internal/lib"
  printf 'package lib\n\n// Two returns 2.\nfunc Two() int { return 2 }\n' > "$cv/internal/lib/lib.go"
  printf 'package lib\n\nimport "testing"\n\nfunc TestTwo(t *testing.T) {\n\tif Two() != 2 {\n\t\tt.Fatal("two")\n\t}\n}\n' > "$cv/internal/lib/lib_test.go"
  printf 'package main\n\nimport "example.com/demo/internal/lib"\n\nfunc run() int { return lib.Two() - 1 }\n\nfunc main() { _ = run() }\n' > "$cv/main.go"
  printf 'package main\n\nimport "testing"\n\nfunc TestRun(t *testing.T) {\n\tif run() != 1 {\n\t\tt.Fatal("run")\n\t}\n}\n' > "$cv/main_test.go"
  if [ -n "${MUTANTE:-}" ]; then
    sed 's/ -count=1//' "$cv/Makefile" > "$cv/Makefile.new" && mv "$cv/Makefile.new" "$cv/Makefile"
  fi
  cover_total() {
    if ! (cd "$cv" && GOTOOLCHAIN=auto GOCACHE="$1" make test COVERAGE_MIN=0 > "$work/cache172.log" 2>&1); then
      cat "$work/cache172.log" >&2
      return 1
    fi
    (cd "$cv" && GOTOOLCHAIN=auto go tool cover -func=coverage.out | awk '/^total:/ { print $3 }')
  }
  cover_total "$work/gocache" > /dev/null
  { printf '// shifted\n// shifted\n'; cat "$cv/main.go"; } > "$cv/main.go.new" && mv "$cv/main.go.new" "$cv/main.go"
  warm="$(cover_total "$work/gocache")"
  rm -rf "$work/gocache-cold"
  cold="$(cover_total "$work/gocache-cold")"
  if [ -z "$warm" ] || [ "$warm" != "$cold" ]; then
    echo "FALHOU: cobertura com cache quente ($warm) difere da com cache frio ($cold)" >&2
    exit 1
  fi
  echo "e2e-template.sh go: cobertura com cache quente ok ($warm)"
  # A válvula (sdd-cover-guard.sh) sozinha: sem o -count=1, o perfil quente sai misturado;
  # ela tem de perceber, refazer com um cache frio e chegar à mesma cobertura.
  sed 's/ -count=1//' "$cv/Makefile" > "$cv/Makefile.new" && mv "$cv/Makefile.new" "$cv/Makefile"
  rm -rf "$work/gocache-valve"
  cover_total "$work/gocache-valve" > /dev/null
  { printf '// shifted\n'; cat "$cv/main.go"; } > "$cv/main.go.new" && mv "$cv/main.go.new" "$cv/main.go"
  valve="$(cover_total "$work/gocache-valve")"
  rm -rf "$work/gocache-cold2"
  cold="$(cover_total "$work/gocache-cold2")"
  if [ -z "$valve" ] || [ "$valve" != "$cold" ]; then
    echo "FALHOU: sem -count=1, a válvula não corrigiu a cobertura ($valve, cache frio $cold)" >&2
    exit 1
  fi
  echo "e2e-template.sh go: válvula do cache ok ($valve)"
fi

# 011 AC-2: com o go.mod pedindo outra versão, o hook de sessão deixa a toolchain
# baixada pelo GOTOOLCHAIN com o covdata, e a cobertura roda num pacote sem testes.
# GOTOOLCHAIN=auto como numa sessão na web (o CI fixa local).
if [ "$lang" = go ]; then
  want=1.25.1
  if [ "$(printf '%s\n' "go$want" "$(GOTOOLCHAIN=local go env GOVERSION)" | sort -V | tail -n 1)" = "go$want" ] &&
    [ "$(GOTOOLCHAIN=local go env GOVERSION)" != "go$want" ]; then
    hk="$work/$lang-hook"
    cp -R "$d" "$hk"
    cd "$hk"
    sed -i.bak "s/^go .*/go $want/" go.mod && rm go.mod.bak
    mkdir -p notest && printf 'package notest\n\n// N has no test.\nfunc N() int { return 1 }\n' > notest/notest.go
    gp="$work/gopath"; mkdir -p "$gp/bin"
    # golangci-lint falso na versão do Makefile: o teste é só da toolchain.
    v="$(sed -n 's/^GOLANGCI_LINT_VERSION *:= *v//p' Makefile)"
    printf '#!/bin/sh\necho "golangci-lint has version %s built"\n' "$v" > "$gp/bin/golangci-lint"
    chmod +x "$gp/bin/golangci-lint"
    mc="$(go env GOMODCACHE)"
    CLAUDE_CODE_REMOTE=true CLAUDE_PROJECT_DIR="$hk" GOTOOLCHAIN=auto GOPATH="$gp" GOMODCACHE="$mc" bash .claude/hooks/session-start.sh
    GOTOOLCHAIN=auto GOPATH="$gp" GOMODCACHE="$mc" go test -coverprofile=coverage.out ./... > "$work/hook.log" 2>&1 ||
      { cat "$work/hook.log" >&2; echo "FALHOU: cobertura com a toolchain do go.mod ($want)" >&2; exit 1; }
    echo "e2e-template.sh go: hook com a toolchain go$want ok"
  else
    echo "e2e-template.sh go: Go instalado já é go$want ou mais novo; hook não verificado"
  fi
fi
echo "e2e-template.sh $lang ok"
