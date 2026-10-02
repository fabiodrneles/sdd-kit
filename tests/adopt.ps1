#!/usr/bin/env pwsh
# Testes da spec 004 (AC-1..AC-5) para scripts/adopt.ps1, rodados no Windows
# pelo CI. No Linux e no macOS, tests/adopt.sh cobre o mesmo e compara as duas
# implementações (AC-6).
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = Split-Path -Parent $PSScriptRoot
$adopt = Join-Path $root 'scripts/adopt.ps1'
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("sdd-kit-test-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
$opts = @('--project', 'Demo', '--owner', 'acme', '--repo', 'demo')

function Fail([string]$Message) { throw "FALHOU: $Message" }
function Invoke-Adopt([string[]]$ArgList) {
  $out = & pwsh -NoProfile -File $adopt @ArgList 2>&1
  return @{ Code = $LASTEXITCODE; Out = ($out -join "`n") }
}
function Get-Snapshot([string]$Dir) {
  Get-ChildItem -LiteralPath $Dir -Recurse -File -Force | Sort-Object FullName |
    ForEach-Object { $_.FullName + ' ' + (Get-FileHash -LiteralPath $_.FullName).Hash }
}

try {
  foreach ($lang in 'go', 'node', 'java', 'python') {
    # 004 AC-1
    $d = Join-Path $tmp "$lang-empty"; New-Item -ItemType Directory -Path $d | Out-Null
    $r = Invoke-Adopt (@('--lang', $lang) + $opts + @($d))
    if ($r.Code -ne 0) { Fail "$lang saiu com $($r.Code): $($r.Out)" }
    $want = @(Get-ChildItem (Join-Path $root 'template/common'), (Join-Path $root "template/$lang") -Recurse -File -Force).Count
    $want++ # .sdd-kit.json
    $got = @(Get-ChildItem $d -Recurse -File -Force).Count
    if ($want -ne $got) { Fail "${lang}: esperava $want arquivos, gerou $got" }
    if (Get-ChildItem $d -Recurse -File -Force | Select-String -Pattern '\{\{[A-Z_]*\}\}' -CaseSensitive) { Fail "${lang}: sobrou marcador" }
    $bytes = [IO.File]::ReadAllBytes((Join-Path $d 'CLAUDE.md'))
    if ($bytes -contains 13) { Fail "${lang}: CLAUDE.md com CRLF" }
    $st = Get-Content -Raw (Join-Path $d '.sdd-kit.json') | ConvertFrom-Json
    if ($st.lang -ne $lang -or @($st.files.PSObject.Properties).Count -ne ($want - 1)) { Fail "${lang}: .sdd-kit.json inválido" }
    # 004 AC-3
    $before = Get-Snapshot $d
    $r = Invoke-Adopt (@('--lang', $lang) + $opts + @($d))
    if (Compare-Object $before (Get-Snapshot $d)) { Fail "${lang}: segunda execução mudou arquivos" }
    if ($r.Out -notmatch ': 0 criados') { Fail "${lang}: segunda execução criou arquivos" }
  }

  # 004 AC-2
  $d = Join-Path $tmp 'existing'; New-Item -ItemType Directory -Path $d | Out-Null
  Set-Content -LiteralPath (Join-Path $d 'CLAUDE.md') -Value 'meu claude' -NoNewline
  $r = Invoke-Adopt (@('--lang', 'go') + $opts + @($d))
  if ((Get-Content -Raw (Join-Path $d 'CLAUDE.md')) -ne 'meu claude') { Fail 'CLAUDE.md existente foi sobrescrito' }
  if ($r.Out -notmatch '(?m)^ignorado \(já existe\): CLAUDE\.md$') { Fail 'CLAUDE.md não aparece como ignorado' }

  # 004 AC-4
  $d = Join-Path $tmp 'invalid'; New-Item -ItemType Directory -Path $d | Out-Null
  foreach ($argList in @((@('--lang', 'cobol') + $opts + @($d)), ($opts + @($d)), @('--lang'))) {
    $r = Invoke-Adopt $argList
    if ($r.Code -ne 2) { Fail "'$($argList -join ' ')' saiu com $($r.Code), esperava 2" }
    if (@(Get-ChildItem -Force $d).Count -ne 0) { Fail "'$($argList -join ' ')' escreveu no destino" }
  }

  # 004 AC-5
  $r = Invoke-Adopt (@('--lang', 'python', '--dry-run') + $opts + @($d))
  if (@(Get-ChildItem -Force $d).Count -ne 0) { Fail '--dry-run escreveu no destino' }
  if ($r.Out -notmatch '(?m)^\(dry-run\) ') { Fail '--dry-run sem resumo' }

  # 010 FR-2, AC-2
  $d = Join-Path $tmp 'skeleton'; New-Item -ItemType Directory -Path $d | Out-Null
  $null = Invoke-Adopt (@('--lang', 'python', '--skeleton') + $opts + @($d))
  if (-not (Test-Path -LiteralPath (Join-Path $d 'tests/test_greeting.py'))) { Fail '--skeleton não criou o teste' }
  if ((Get-Content -Raw (Join-Path $d '.sdd-kit.json')) -match 'test_greeting') { Fail '--skeleton registrou o esqueleto no estado' }
  Set-Content -LiteralPath (Join-Path $d 'app/__init__.py') -Value 'meu' -NoNewline
  $null = Invoke-Adopt (@('--lang', 'python', '--skeleton', '--force') + $opts + @($d))
  if ((Get-Content -Raw (Join-Path $d 'app/__init__.py')) -ne 'meu') { Fail '--skeleton --force sobrescreveu código do projeto' }

  Write-Output 'tests/adopt.ps1 ok'
}
finally {
  Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
