#!/usr/bin/env pwsh
# Instala o axyn no Windows com um comando (spec 021 FR-1):
#   irm https://raw.githubusercontent.com/fabiodrneles/sdd-kit/main/scripts/install-axyn.ps1 | iex
# Baixa axyn_windows_amd64.exe da release, confere o sha256 pelo axyn_checksums.txt e
# instala em $env:AXYN_BIN (padrão $env:LOCALAPPDATA\axyn). Num repositório git, roda
# `axyn install` no fim. Variáveis: AXYN_VERSION (padrão latest), AXYN_RELEASE_URL
# (base do download, para testes).
$ErrorActionPreference = 'Stop'

# Roda por `irm | iex`, dentro da sessão do próprio usuário: nunca `exit` (fecharia a
# janela); os erros usam throw, que também dá código diferente de zero no `pwsh -File`.
function Stop-Install([string]$Message) {
  throw "install-axyn: $Message"
}

# O Invoke-WebRequest do PowerShell 7 não aceita file:// (base local dos testes).
function Get-Asset([string]$Uri, [string]$OutFile) {
  $u = [Uri]$Uri
  if ($u.IsFile) { Copy-Item -LiteralPath $u.LocalPath -Destination $OutFile }
  else { Invoke-WebRequest -UseBasicParsing -Uri $Uri -OutFile $OutFile }
}

$name = 'axyn_windows_amd64.exe'
if ($env:AXYN_RELEASE_URL) { $base = $env:AXYN_RELEASE_URL }
elseif ($env:AXYN_VERSION) { $base = "https://github.com/fabiodrneles/sdd-kit/releases/download/$($env:AXYN_VERSION)" }
else { $base = 'https://github.com/fabiodrneles/sdd-kit/releases/latest/download' }

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("axyn-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  try {
    Get-Asset "$base/$name" (Join-Path $tmp $name)
    Get-Asset "$base/axyn_checksums.txt" (Join-Path $tmp 'sums')
  } catch { Stop-Install "não consegui baixar de ${base}: $($_.Exception.Message)" }

  $want = $null
  foreach ($line in Get-Content (Join-Path $tmp 'sums')) {
    $f = $line -split '\s+', 2
    if ($f.Count -eq 2 -and $f[1].TrimStart('*').Trim() -eq $name) { $want = $f[0]; break }
  }
  if (-not $want) { Stop-Install "$name não está no axyn_checksums.txt; nada foi instalado" }
  $got = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $tmp $name)).Hash.ToLower()
  if ($got -ne $want.ToLower()) { Stop-Install "sha256 de $name não confere (esperado $want, veio $got); nada foi instalado" }

  $bin = if ($env:AXYN_BIN) { $env:AXYN_BIN } else { Join-Path $env:LOCALAPPDATA 'axyn' }
  New-Item -ItemType Directory -Force -Path $bin | Out-Null
  $dest = Join-Path $bin 'axyn.exe'
  Copy-Item -LiteralPath (Join-Path $tmp $name) -Destination $dest -Force
} finally {
  Remove-Item -Recurse -Force -LiteralPath $tmp -ErrorAction SilentlyContinue
}

Write-Output "axyn instalado em $dest"
$onPath = ($env:PATH -split [IO.Path]::PathSeparator) -contains $bin
if (-not $onPath) {
  Write-Output "adicione ao PATH: [Environment]::SetEnvironmentVariable('Path', `"$bin;`" + [Environment]::GetEnvironmentVariable('Path', 'User'), 'User')"
}

if (Get-Command git -ErrorAction SilentlyContinue) {
  & git rev-parse --git-dir *> $null
  if ($LASTEXITCODE -eq 0) {
    & $dest install
    if ($LASTEXITCODE -ne 0) { Stop-Install "axyn install falhou (código $LASTEXITCODE)" }
    return
  }
}
Write-Output 'no repositório do projeto: axyn install'
