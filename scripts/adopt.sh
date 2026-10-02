#!/bin/sh
# Adota o sdd-kit num repositório novo ou existente (spec 004): copia
# template/common e template/<lang> sem sobrescrever o que já existe.
#
# Uso: adopt.sh --lang go|node|java|python|rust|dotnet [--project NOME] [--owner DONO]
#               [--repo REPO] [--dry-run] [--force] [--skeleton] [DESTINO]
#
# Sem o template ao lado do script (ex.: curl … | sh -s -- …), baixa o da
# versão SDD_KIT_REF do GitHub.
set -eu

KIT_REF="${SDD_KIT_REF:-v1.1.0}"
LANGS="go node java python rust dotnet"

usage() {
  cat >&2 <<USAGE
uso: adopt.sh --lang go|node|java|python|rust|dotnet [opções] [DESTINO]

  --lang LING      linguagem do repositório (obrigatório): $LANGS
  --project NOME   nome do projeto (padrão: nome do diretório de destino)
  --owner DONO     dono no GitHub (padrão: deduzido do remote origin)
  --repo REPO      repositório no GitHub (padrão: deduzido do remote origin)
  --dry-run        mostra o que seria feito, sem escrever nada
  --force          sobrescreve arquivos que já existem
  --skeleton       num repositório vazio, cria um projeto mínimo com um teste para o CI
                   nascer verde (nunca sobrescreve, nem com --force)
  DESTINO          diretório do repositório (padrão: diretório atual)
USAGE
  exit 2
}

die() { echo "erro: $*" >&2; usage; }

lang="" project="" owner="" repo="" dry=0 force=0 skeleton=0 dest="."
while [ $# -gt 0 ]; do
  case "$1" in
    --lang) [ $# -ge 2 ] || die "--lang precisa de um valor"; lang="$2"; shift 2 ;;
    --project) [ $# -ge 2 ] || die "--project precisa de um valor"; project="$2"; shift 2 ;;
    --owner) [ $# -ge 2 ] || die "--owner precisa de um valor"; owner="$2"; shift 2 ;;
    --repo) [ $# -ge 2 ] || die "--repo precisa de um valor"; repo="$2"; shift 2 ;;
    --dry-run) dry=1; shift ;;
    --force) force=1; shift ;;
    --skeleton) skeleton=1; shift ;;
    -h | --help) usage ;;
    -*) die "opção desconhecida: $1" ;;
    *) dest="$1"; shift ;;
  esac
done

[ -n "$lang" ] || die "--lang é obrigatório"
case " $LANGS " in *" $lang "*) ;; *) die "linguagem inválida: $lang" ;; esac
[ -d "$dest" ] || die "destino não é um diretório: $dest"
dest="$(cd "$dest" && pwd)"

# Dono e repositório a partir do remote origin (https, ssh ou proxy).
if [ -z "$owner" ] || [ -z "$repo" ]; then
  url="$(git -C "$dest" remote get-url origin 2>/dev/null || true)"
  path="$(printf '%s\n' "$url" | sed -e 's#\.git$##' -e 's#^[a-z]*@[^:/]*:#/#' | awk -F/ 'NF >= 2 { print $(NF-1) "/" $NF }')"
  [ -n "$owner" ] || owner="${path%%/*}"
  [ -n "$repo" ] || repo="${path#*/}"
fi
[ -n "$owner" ] || die "não foi possível deduzir --owner do remote origin"
[ -n "$repo" ] || die "não foi possível deduzir --repo do remote origin"
[ -n "$project" ] || project="$(basename "$dest")"

# Template: ao lado do script ou baixado da versão do kit.
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
template=""
case "$0" in
  *adopt.sh) template="$(cd "$(dirname "$0")/.." && pwd)/template" ;;
esac
if [ -z "$template" ] || [ ! -d "$template/common" ]; then
  echo "baixando o template do sdd-kit $KIT_REF…" >&2
  curl -fsSL "https://github.com/fabiodrneles/sdd-kit/archive/$KIT_REF.tar.gz" | tar -xz -C "$tmp"
  template="$(find "$tmp" -mindepth 2 -maxdepth 2 -type d -name template | head -n1)"
  [ -d "$template/common" ] || { echo "erro: template não encontrado em $KIT_REF" >&2; exit 1; }
fi

# Valores escapados para o sed (/, & e \).
esc() { printf '%s\n' "$1" | sed 's/[\/&\\]/\\&/g'; }
s_project="$(esc "$project")" s_owner="$(esc "$owner")" s_repo="$(esc "$repo")"

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1"; else shasum -a 256 "$1"; fi | cut -d' ' -f1
}

# Spec 011 FR-4: com uma tag vX.Y.Z no repositório, as fases do ROADMAP criado
# começam na versão seguinte (Fase 1 e 2: próximas minors; Fase 3: v1.0.0 se
# ainda for 0.x, senão a terceira minor). Sem tag, ficam v0.1.0, v0.2.0, v1.0.0.
tag="$(git -C "$dest" tag --list 'v[0-9]*' --sort=-v:refname 2>/dev/null | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -n 1 || true)"
roadmap_sed=""
if [ -n "$tag" ]; then
  major="$(echo "${tag#v}" | cut -d. -f1)" minor="$(echo "${tag#v}" | cut -d. -f2)"
  f3="v1.0.0"; [ "$major" -eq 0 ] || f3="v$major.$((minor + 3)).0"
  roadmap_sed="/^## Fase 1 /s/\`v[0-9.]*\`/\`v$major.$((minor + 1)).0\`/;/^## Fase 2 /s/\`v[0-9.]*\`/\`v$major.$((minor + 2)).0\`/;/^## Fase 3 /s/\`v[0-9.]*\`/\`$f3\`/"
fi

created=0 skipped=0 overwritten=0
: > "$tmp/state"
parts="common $lang"
# Spec 010 FR-2: o esqueleto é código do projeto, não arquivo gerenciado pelo kit.
[ "$skeleton" -eq 0 ] || parts="$parts skeleton/$lang"
for part in $parts; do
  # Ordenado para o relatório ser estável entre execuções e sistemas.
  (cd "$template/$part" && find . -type f | sed 's#^\./##' | LC_ALL=C sort) > "$tmp/files"
  while IFS= read -r rel; do
    src="$template/$part/$rel"
    dst="$dest/$rel"
    if [ -e "$dst" ] && { [ "$force" -eq 0 ] || [ "$part" != "${part#skeleton/}" ]; }; then
      echo "ignorado (já existe): $rel"
      skipped=$((skipped + 1))
      continue
    fi
    if [ -e "$dst" ]; then
      action="sobrescrito"; overwritten=$((overwritten + 1))
    else
      action="criado"; created=$((created + 1))
    fi
    echo "$action: $rel"
    [ "$dry" -eq 0 ] || continue
    mkdir -p "$(dirname "$dst")"
    sed -e "s/{{PROJECT}}/$s_project/g" -e "s/{{OWNER}}/$s_owner/g" -e "s/{{REPO}}/$s_repo/g" "$src" > "$dst"
    if [ -x "$src" ]; then chmod +x "$dst"; fi
    [ "$part" != "${part#skeleton/}" ] || echo "$rel $(sha256 "$dst")" >> "$tmp/state"
    # O estado guarda o hash do ROADMAP do kit, para a sincronização (que gera o
    # template sem tags) não tomar a numeração ajustada por mudança do kit.
    if [ "$rel" = specs/ROADMAP.md ] && [ -n "$roadmap_sed" ]; then
      sed "$roadmap_sed" "$dst" > "$tmp/roadmap" && cat "$tmp/roadmap" > "$dst"
    fi
  done < "$tmp/files"
done

# Arquivo de estado (spec 005 FR-2): versão do kit, linguagem e o hash de cada
# arquivo gerenciado. Como qualquer outro arquivo, só é reescrito com --force.
if [ -e "$dest/.sdd-kit.json" ] && [ "$force" -eq 0 ]; then
  echo "ignorado (já existe): .sdd-kit.json"
  skipped=$((skipped + 1))
else
  if [ -e "$dest/.sdd-kit.json" ]; then
    echo "sobrescrito: .sdd-kit.json"; overwritten=$((overwritten + 1))
  else
    echo "criado: .sdd-kit.json"; created=$((created + 1))
  fi
  if [ "$dry" -eq 0 ]; then
    {
      printf '{\n  "kit": "sdd-kit",\n  "version": "%s",\n  "lang": "%s",\n  "project": "%s",\n  "owner": "%s",\n  "repo": "%s",\n  "files": {' \
        "$KIT_REF" "$lang" "$project" "$owner" "$repo"
      sep=""
      while read -r rel hash; do
        printf '%s\n    "%s": "%s"' "$sep" "$rel" "$hash"
        sep=","
      done < "$tmp/state"
      printf '\n  }\n}\n'
    } > "$dest/.sdd-kit.json"
  fi
fi

prefix=""
[ "$dry" -eq 0 ] || prefix="(dry-run) "
echo "${prefix}sdd-kit $lang em $dest: $created criados, $skipped ignorados, $overwritten sobrescritos"

# Spec 011 FR-3: a licença é decisão do dono; a adoção só avisa que falta.
lic=""
for f in "$dest"/LICENSE* "$dest"/LICENCE* "$dest"/COPYING*; do
  [ ! -f "$f" ] || lic="$f"
done
[ -n "$lic" ] || echo "aviso: sem LICENSE: escolha uma licença (https://choosealicense.com) e crie o arquivo"

# Spec 011 FR-1: avisa o que faria o CI Node nascer vermelho, sem falhar.
if [ "$lang" = node ] && [ -f "$dest/package.json" ]; then
  if ! command -v node >/dev/null 2>&1; then
    echo "aviso: node ausente: lockfile e script de teste não verificados"
  else
    (cd "$dest" && node -e '
      const fs = require("fs");
      const p = JSON.parse(fs.readFileSync("package.json", "utf8"));
      const test = (p.scripts || {}).test;
      if (!test || /no test specified/.test(test))
        console.log("aviso: package.json sem script \"test\": o make ci falha até o projeto ter testes");
      if (!fs.existsSync("package-lock.json")) {
        console.log("aviso: sem package-lock.json: o npm ci do CI falha; rode npm install e versione o lockfile");
      } else {
        const root = (JSON.parse(fs.readFileSync("package-lock.json", "utf8")).packages || {})[""];
        const norm = (o) => JSON.stringify(Object.keys(o || {}).sort().map((k) => [k, o[k]]));
        const fields = ["dependencies", "devDependencies", "optionalDependencies", "peerDependencies"];
        if (root && fields.some((f) => norm(p[f]) !== norm(root[f])))
          console.log("aviso: package-lock.json fora de sincronia com o package.json: o npm ci do CI falha; rode npm install");
      }')
  fi
fi
