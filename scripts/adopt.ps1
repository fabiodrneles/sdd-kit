#!/usr/bin/env pwsh
# Adota o sdd-kit num repositório novo ou existente (spec 004): copia
# template/common e template/<lang> sem sobrescrever o que já existe, e os
# modelos do projeto (template/seed) só quando faltam.
# Equivalente a scripts/adopt.sh, com as mesmas opções e a mesma saída.
#
# Uso: adopt.ps1 --lang go|node|java|python|rust|dotnet [--project NOME] [--owner DONO]
#                [--repo REPO] [--dry-run] [--force] [--skeleton] [DESTINO]
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$KitRef = if ($env:SDD_KIT_REF) { $env:SDD_KIT_REF } else { 'v1.7.0' }
$Langs = @('go', 'node', 'java', 'python', 'rust', 'dotnet')

function Show-Usage {
  [Console]::Error.WriteLine(@"
uso: adopt.ps1 --lang go|node|java|python|rust|dotnet [opções] [DESTINO]

  --lang LING      linguagem do repositório (obrigatório): $($Langs -join ' ')
  --project NOME   nome do projeto (padrão: nome do diretório de destino)
  --owner DONO     dono no GitHub (padrão: deduzido do remote origin)
  --repo REPO      repositório no GitHub (padrão: deduzido do remote origin)
  --dry-run        mostra o que seria feito, sem escrever nada
  --force          sobrescreve arquivos que já existem
  --skeleton       num repositório vazio, cria um projeto mínimo com um teste para o CI
                   nascer verde (nunca sobrescreve, nem com --force)
  DESTINO          diretório do repositório (padrão: diretório atual)
"@)
  exit 2
}

function Stop-WithUsage([string]$Message) {
  [Console]::Error.WriteLine("erro: $Message")
  Show-Usage
}

$lang = ''; $project = ''; $owner = ''; $repo = ''; $dry = $false; $force = $false; $skeleton = $false; $dest = '.'
for ($i = 0; $i -lt $args.Count; $i++) {
  $a = [string]$args[$i]
  switch -CaseSensitive ($a) {
    { $_ -in '--lang', '--project', '--owner', '--repo' } {
      if ($i + 1 -ge $args.Count) { Stop-WithUsage "$a precisa de um valor" }
      $i++
      $v = [string]$args[$i]
      switch ($a) {
        '--lang' { $lang = $v }
        '--project' { $project = $v }
        '--owner' { $owner = $v }
        '--repo' { $repo = $v }
      }
      break
    }
    '--dry-run' { $dry = $true; break }
    '--force' { $force = $true; break }
    '--skeleton' { $skeleton = $true; break }
    { $_ -in '-h', '--help' } { Show-Usage }
    { $_.StartsWith('-') } { Stop-WithUsage "opção desconhecida: $a" }
    default { $dest = $a }
  }
}

if (-not $lang) { Stop-WithUsage '--lang é obrigatório' }
if ($lang -cnotin $Langs) { Stop-WithUsage "linguagem inválida: $lang" }
if (-not (Test-Path -LiteralPath $dest -PathType Container)) { Stop-WithUsage "destino não é um diretório: $dest" }
$dest = (Resolve-Path -LiteralPath $dest).Path

# Dono e repositório a partir do remote origin (https, ssh ou proxy).
if (-not $owner -or -not $repo) {
  $url = ''
  try { $url = (& git -C $dest remote get-url origin 2>$null) } catch { $url = '' }
  if ($LASTEXITCODE -ne 0 -or -not $url) { $url = '' }
  $url = ($url -replace '\.git$', '') -replace '^[a-z]*@[^:/]*:', '/'
  $parts = @($url -split '/' | Where-Object { $_ -ne '' })
  if ($parts.Count -ge 2) {
    if (-not $owner) { $owner = $parts[-2] }
    if (-not $repo) { $repo = $parts[-1] }
  }
}
if (-not $owner) { Stop-WithUsage 'não foi possível deduzir --owner do remote origin' }
if (-not $repo) { Stop-WithUsage 'não foi possível deduzir --repo do remote origin' }
if (-not $project) { $project = Split-Path -Leaf $dest }

# Template: ao lado do script ou baixado da versão do kit.
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("sdd-kit-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  $template = Join-Path (Split-Path -Parent $PSScriptRoot) 'template'
  if (-not $PSScriptRoot -or -not (Test-Path (Join-Path $template 'common'))) {
    [Console]::Error.WriteLine("baixando o template do sdd-kit $KitRef…")
    $tgz = Join-Path $tmp 'kit.tar.gz'
    Invoke-WebRequest -Uri "https://github.com/fabiodrneles/sdd-kit/archive/$KitRef.tar.gz" -OutFile $tgz
    & tar -xzf $tgz -C $tmp
    $template = Get-ChildItem -Path $tmp -Directory | ForEach-Object { Join-Path $_.FullName 'template' } |
      Where-Object { Test-Path (Join-Path $_ 'common') } | Select-Object -First 1
    if (-not $template) { [Console]::Error.WriteLine("erro: template não encontrado em $KitRef"); exit 1 }
  }

  $utf8 = New-Object System.Text.UTF8Encoding($false)
  # Spec 011 FR-4: com uma tag vX.Y.Z no repositório, as fases do ROADMAP criado
  # começam na versão seguinte (Fase 1 e 2: próximas minors; Fase 3: v1.0.0 se
  # ainda for 0.x, senão a terceira minor). Sem tag, ficam v0.1.0, v0.2.0, v1.0.0.
  $phaseVersions = $null
  $tags = @()
  try { $tags = @(& git -C $dest tag --list 'v[0-9]*' --sort=-v:refname 2>$null) } catch { $tags = @() }
  $tag = $tags | Where-Object { $_ -cmatch '^v[0-9]+\.[0-9]+\.[0-9]+$' } | Select-Object -First 1
  if ($tag) {
    $nums = $tag.Substring(1).Split('.')
    $major = [int]$nums[0]; $minor = [int]$nums[1]
    $f3 = if ($major -eq 0) { 'v1.0.0' } else { "v$major.$($minor + 3).0" }
    $phaseVersions = @{ '1' = "v$major.$($minor + 1).0"; '2' = "v$major.$($minor + 2).0"; '3' = $f3 }
  }

  $created = 0; $skipped = 0; $overwritten = 0
  $state = [System.Collections.Generic.List[string]]::new()
  $parts = @('common', $lang, 'seed')
  # Spec 010 FR-2: o esqueleto é código do projeto, não arquivo gerenciado pelo kit.
  if ($skeleton) { $parts += "skeleton/$lang" }
  foreach ($part in $parts) {
    # Spec 012 FR-1: template/seed (specs e CHANGELOG) e o esqueleto (spec 010 FR-2)
    # são do projeto: criados quando faltam, nunca sobrescritos nem registrados no estado.
    $isOwned = $part.StartsWith('skeleton/') -or $part -ceq 'seed'
    $base = (Resolve-Path (Join-Path $template $part)).Path
    # Mesma ordem do sh (LC_ALL=C sort): ordinal.
    $files = [string[]]@(Get-ChildItem -LiteralPath $base -Recurse -File -Force |
        ForEach-Object { $_.FullName.Substring($base.Length + 1).Replace('\', '/') })
    [Array]::Sort($files, [StringComparer]::Ordinal)
    foreach ($rel in $files) {
      $src = Join-Path $base $rel
      $dst = Join-Path $dest $rel
      $exists = Test-Path -LiteralPath $dst
      if ($exists -and (-not $force -or $isOwned)) {
        Write-Output "ignorado (já existe): $rel"
        $skipped++
        continue
      }
      if ($exists) { $action = 'sobrescrito'; $overwritten++ } else { $action = 'criado'; $created++ }
      Write-Output "${action}: $rel"
      if ($dry) { continue }
      New-Item -ItemType Directory -Force -Path (Split-Path -Parent $dst) | Out-Null
      $text = [IO.File]::ReadAllText($src, $utf8)
      $text = $text.Replace('{{PROJECT}}', $project).Replace('{{OWNER}}', $owner).Replace('{{REPO}}', $repo)
      [IO.File]::WriteAllText($dst, $text, $utf8)
      if (-not $IsWindows) {
        $mode = [IO.File]::GetUnixFileMode($src)
        if ($mode -band [IO.UnixFileMode]::UserExecute) { [IO.File]::SetUnixFileMode($dst, $mode) }
      }
      if ($rel -ceq 'specs/ROADMAP.md' -and $phaseVersions) {
        $lines = [IO.File]::ReadAllText($dst, $utf8).Split("`n")
        for ($j = 0; $j -lt $lines.Count; $j++) {
          if ($lines[$j] -cmatch '^## Fase ([123]) ') {
            $lines[$j] = $lines[$j] -creplace '`v[0-9.]*`', ('`' + $phaseVersions[$Matches[1]] + '`')
          }
        }
        [IO.File]::WriteAllText($dst, ($lines -join "`n"), $utf8)
      }
      if ($isOwned) { continue }
      $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $dst).Hash.ToLowerInvariant()
      $state.Add('    "' + $rel + '": "' + $hash + '"')
    }
  }

  # Arquivo de estado (spec 005 FR-2), igual byte a byte ao do adopt.sh.
  $statePath = Join-Path $dest '.sdd-kit.json'
  $stateExists = Test-Path -LiteralPath $statePath
  if ($stateExists -and -not $force) {
    Write-Output 'ignorado (já existe): .sdd-kit.json'
    $skipped++
  }
  else {
    if ($stateExists) { Write-Output 'sobrescrito: .sdd-kit.json'; $overwritten++ }
    else { Write-Output 'criado: .sdd-kit.json'; $created++ }
    if (-not $dry) {
      $json = "{`n  `"kit`": `"sdd-kit`",`n  `"version`": `"$KitRef`",`n  `"lang`": `"$lang`",`n  `"project`": `"$project`",`n  `"owner`": `"$owner`",`n  `"repo`": `"$repo`",`n  `"files`": {"
      if ($state.Count -gt 0) { $json += "`n" + ($state -join ",`n") }
      $json += "`n  }`n}`n"
      [IO.File]::WriteAllText($statePath, $json, $utf8)
    }
  }

  $prefix = if ($dry) { '(dry-run) ' } else { '' }
  Write-Output "${prefix}sdd-kit $lang em ${dest}: $created criados, $skipped ignorados, $overwritten sobrescritos"

  # Spec 011 FR-3: a licença é decisão do dono; a adoção só avisa que falta.
  if (-not (Get-ChildItem -LiteralPath $dest -File | Where-Object { $_.Name -cmatch '^(LICENSE|LICENCE|COPYING)' })) {
    Write-Output 'aviso: sem LICENSE: escolha uma licença (https://choosealicense.com) e crie o arquivo'
  }

  # Spec 011 FR-1: avisa o que faria o CI Node nascer vermelho, sem falhar.
  $pkgPath = Join-Path $dest 'package.json'
  if ($lang -eq 'node' -and (Test-Path -LiteralPath $pkgPath)) {
    # -AsHashtable: o lockfile tem a chave vazia "", que um PSCustomObject não aceita.
    function Get-Field($Obj, [string]$Name) {
      if ($Obj -is [Collections.IDictionary] -and $Obj.Contains($Name)) { return $Obj[$Name] }
      return $null
    }
    function Get-Norm($Obj) {
      if ($Obj -isnot [Collections.IDictionary]) { return '' }
      return (@($Obj.Keys | Sort-Object -CaseSensitive | ForEach-Object { $_ + '=' + $Obj[$_] }) -join "`n")
    }
    $pkg = [IO.File]::ReadAllText($pkgPath, $utf8) | ConvertFrom-Json -AsHashtable
    $test = Get-Field (Get-Field $pkg 'scripts') 'test'
    if (-not $test -or $test -match 'no test specified') {
      Write-Output 'aviso: package.json sem script "test": o make ci falha até o projeto ter testes'
    }
    $lockPath = Join-Path $dest 'package-lock.json'
    if (-not (Test-Path -LiteralPath $lockPath)) {
      Write-Output 'aviso: sem package-lock.json: o npm ci do CI falha; rode npm install e versione o lockfile'
    }
    else {
      $lock = [IO.File]::ReadAllText($lockPath, $utf8) | ConvertFrom-Json -AsHashtable
      $root = Get-Field (Get-Field $lock 'packages') ''
      if ($null -ne $root) {
        foreach ($f in 'dependencies', 'devDependencies', 'optionalDependencies', 'peerDependencies') {
          if ((Get-Norm (Get-Field $pkg $f)) -cne (Get-Norm (Get-Field $root $f))) {
            Write-Output 'aviso: package-lock.json fora de sincronia com o package.json: o npm ci do CI falha; rode npm install'
            break
          }
        }
      }
    }
  }
}
finally {
  Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
