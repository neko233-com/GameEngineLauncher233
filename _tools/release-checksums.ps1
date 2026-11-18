param([string]$OutputDir = 'build/windows-amd64')
$ErrorActionPreference = 'Stop'
if (-not [Environment]::Is64BitProcess) {
    $native = Get-Command pwsh.exe -All | Where-Object { $_.Source -notmatch '\(x86\)' } | Select-Object -First 1 -ExpandProperty Source
    if (-not $native) { throw 'A 64-bit PowerShell 7 is required.' }
    & $native -NoProfile -File $PSCommandPath -OutputDir $OutputDir
    exit $LASTEXITCODE
}
$root = Split-Path -Parent $PSScriptRoot
$version = (Get-Content (Join-Path $root 'mygo.json') -Raw | ConvertFrom-Json).version
$dir = [IO.Path]::GetFullPath((Join-Path $root $OutputDir))
$names = @('GameEngineLauncher233.exe', "gameenginelauncher233-setup-$version-amd64.exe", "gameenginelauncher233-$version-amd64.msi")
$lines = foreach ($name in $names) {
    $path = Join-Path $dir $name
    if (-not (Test-Path -LiteralPath $path)) { throw "Missing release file: $path" }
    '{0}  {1}' -f (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant(), $name
}
Set-Content -LiteralPath (Join-Path $dir 'SHA256SUMS.txt') -Value $lines -Encoding ascii
