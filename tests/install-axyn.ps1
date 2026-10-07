#!/usr/bin/env pwsh
# Testes da spec 021 FR-1 para scripts/install-axyn.ps1, rodados no Windows pelo CI
# (tests/install-axyn.sh cobre o equivalente em sh).
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = Split-Path -Parent $PSScriptRoot
$install = Join-Path $root 'scripts/install-axyn.ps1'
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("axyn-test-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null

function Fail([string]$Message) { throw "FALHOU: $Message" }
function Invoke-Install([string]$Dir, [hashtable]$EnvVars) {
  Push-Location $Dir
  try {
    foreach ($k in $EnvVars.Keys) { Set-Item "Env:$k" $EnvVars[$k] }
    $out = & pwsh -NoProfile -File $install 2>&1
    return @{ Code = $LASTEXITCODE; Out = ($out -join "`n") }
  } finally { Pop-Location }
}

try {
  $dist = Join-Path $tmp 'dist'
  New-Item -ItemType Directory -Path $dist | Out-Null
  $exe = Join-Path $dist 'axyn_windows_amd64.exe'
  Push-Location (Join-Path $root 'axyn')
  try {
    $env:CGO_ENABLED = '0'
    & go build -ldflags '-X main.version=v9.9.9' -o $exe .
    if ($LASTEXITCODE -ne 0) { Fail 'go build falhou' }
  } finally { Pop-Location }
  $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $exe).Hash.ToLower()
  Set-Content -Path (Join-Path $dist 'axyn_checksums.txt') -Value "$hash  axyn_windows_amd64.exe"

  $repo = Join-Path $tmp 'repo'
  New-Item -ItemType Directory -Path $repo | Out-Null
  & git -C $repo init -q
  $bin = Join-Path $tmp 'bin'
  $r = Invoke-Install $repo @{ AXYN_RELEASE_URL = ([Uri]$dist).AbsoluteUri; AXYN_BIN = $bin }
  if ($r.Code -ne 0) { Fail "instalação saiu com $($r.Code): $($r.Out)" }
  $dest = Join-Path $bin 'axyn.exe'
  if (-not (Test-Path $dest)) { Fail "axyn.exe não foi instalado: $($r.Out)" }
  if ((& $dest version) -notmatch 'v9\.9\.9') { Fail 'axyn version não traz a versão do build' }
  if (-not (Test-Path (Join-Path $repo 'opencode.json'))) { Fail "axyn install não rodou no repositório git: $($r.Out)" }

  # checksum errado: para com o motivo e não instala nada
  $bad = Join-Path $tmp 'bad'
  Copy-Item -Recurse $dist $bad
  Set-Content -Path (Join-Path $bad 'axyn_checksums.txt') -Value ('0' * 64 + '  axyn_windows_amd64.exe')
  $bin2 = Join-Path $tmp 'bin2'
  $r = Invoke-Install $repo @{ AXYN_RELEASE_URL = ([Uri]$bad).AbsoluteUri; AXYN_BIN = $bin2 }
  if ($r.Code -eq 0) { Fail 'checksum errado deveria falhar' }
  if ($r.Out -notmatch 'sha256') { Fail "sem o motivo do checksum: $($r.Out)" }
  if (Test-Path (Join-Path $bin2 'axyn.exe')) { Fail 'instalou com checksum errado' }

  Write-Output 'ok: install-axyn.ps1'
} finally {
  Remove-Item -Recurse -Force -LiteralPath $tmp -ErrorAction SilentlyContinue
}
# O último pwsh filho (checksum errado) saiu com 1 de propósito; o passo do Actions
# devolve o $LASTEXITCODE, então o sucesso precisa ser explícito.
exit 0
