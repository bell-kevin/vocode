param(
  [string]$Msys2Root = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$innerScript = Join-Path $scriptRoot "setup-portaudio.ps1"

if (-not (Test-Path $innerScript)) {
  throw "Expected script not found: $innerScript"
}

if ([string]::IsNullOrWhiteSpace($Msys2Root)) {
  & $innerScript
} else {
  & $innerScript -Msys2Root $Msys2Root
}

