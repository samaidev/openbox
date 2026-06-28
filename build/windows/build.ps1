# OpenBox — Windows local build helper
# ----------------------------------------------------------------------------
# Prerequisites on a Windows dev machine:
#   1. Go 1.23+                https://go.dev/dl/
#   2. Inno Setup 6            https://jrsoftware.org/isdl.php
#   3. (Optional) Fyne CLI:    go install fyne.io/fyne/v2/cmd/fyne@latest
#
# Usage:
#   pwsh build\windows\build.ps1           # build + package installer
#   pwsh build\windows\build.ps1 -SkipInstaller   # just compile the exe
# ----------------------------------------------------------------------------
[CmdletBinding()]
param(
  [switch] $SkipInstaller,
  [string] $Version = "0.1.0"
)

$ErrorActionPreference = "Stop"
$root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
Set-Location $root

Write-Host "==> Building openbox.exe (windows/amd64)" -ForegroundColor Cyan
$env:CGO_ENABLED = "1"
$env:GOOS = "windows"
$env:GOARCH = "amd64"

$outDir = Join-Path $root "dist\openbox-windows-amd64"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

go build -ldflags "-H windowsgui -X main.Version=$Version" `
         -o (Join-Path $outDir "openbox.exe") `
         ./cmd/openbox
if ($LASTEXITCODE -ne 0) { throw "go build failed" }

Copy-Item (Join-Path $root "LICENSE")    $outDir -Force
Copy-Item (Join-Path $root "README.md")  $outDir -Force

Write-Host "==> Built: $outDir\openbox.exe" -ForegroundColor Green

if ($SkipInstaller) { return }

# Locate Inno Setup (ISCC). Try standard install paths first.
$iscc = $null
foreach ($candidate in @(
  "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
  "C:\Program Files\Inno Setup 6\ISCC.exe"
)) {
  if (Test-Path $candidate) { $iscc = $candidate; break }
}
if (-not $iscc) {
  $iscc = (Get-Command ISCC.exe -ErrorAction SilentlyContinue).Source
}
if (-not $iscc) {
  Write-Warning "Inno Setup (ISCC.exe) not found on PATH or standard locations."
  Write-Warning "Install it from https://jrsoftware.org/isdl.php and re-run."
  return
}

Write-Host "==> Compiling installer with: $iscc" -ForegroundColor Cyan
& $iscc /Q (Join-Path $root "build\windows\openbox.iss")
if ($LASTEXITCODE -ne 0) { throw "ISCC failed" }

$installer = Join-Path $root "dist\OpenBox-$Version-windows-amd64-setup.exe"
if (Test-Path $installer) {
  $size = [math]::Round((Get-Item $installer).Length / 1MB, 2)
  Write-Host "==> Installer: $installer  ($size MB)" -ForegroundColor Green
}
