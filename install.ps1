# hackfetch installer (Windows / PowerShell)
# usage:
#   irm https://raw.githubusercontent.com/xerneas3318/hackfetch/main/install.ps1 | iex
#
# Override install dir with $env:HACKFETCH_INSTALL_DIR before piping into iex, e.g.:
#   $env:HACKFETCH_INSTALL_DIR='C:\tools\hackfetch'; irm https://raw.githubusercontent.com/xerneas3318/hackfetch/main/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$Repo = 'xerneas3318/hackfetch'

function Write-Status($msg, $color = 'Gray') {
    Write-Host "  $msg" -ForegroundColor $color
}

Write-Host ''
Write-Status '✦ hackfetch installer' 'DarkYellow'
Write-Host ''

# --- detect arch
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    'x86'   { throw '32-bit Windows is not supported' }
    default { throw "unsupported arch: $env:PROCESSOR_ARCHITECTURE" }
}
$platform = "windows-$arch"
Write-Status "✓ detected: $platform" 'Green'

# --- pick install dir
$installDir = if ($env:HACKFETCH_INSTALL_DIR) {
    $env:HACKFETCH_INSTALL_DIR
} else {
    Join-Path $env:LOCALAPPDATA 'Programs\hackfetch'
}
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

# --- latest release tag
Write-Status '→ checking latest release...' 'DarkGray'
$release = Invoke-RestMethod -UseBasicParsing -Uri "https://api.github.com/repos/$Repo/releases/latest"
$tag = $release.tag_name
if (-not $tag) { throw "couldn't fetch latest release tag" }
Write-Status "✓ version: $tag" 'Green'

$assetName = "hackfetch-$platform.zip"
$url = "https://github.com/$Repo/releases/download/$tag/$assetName"

# --- download + extract
$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("hackfetch-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
try {
    Write-Status "↓ downloading $url" 'DarkGray'
    $zip = Join-Path $tmp 'hackfetch.zip'
    Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $zip

    Expand-Archive -Force -Path $zip -DestinationPath $tmp

    $src = Join-Path $tmp 'hackfetch.exe'
    if (-not (Test-Path $src)) { throw "archive missing hackfetch.exe" }

    $dst = Join-Path $installDir 'hackfetch.exe'
    Write-Status "↓ installing to $dst" 'DarkGray'
    Copy-Item -Force $src $dst
}
finally {
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $tmp
}

Write-Host ''
Write-Status "✓ installed  $dst" 'Green'
Write-Host ''

# --- PATH check / persist for current user
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$pathParts = if ($userPath) { $userPath -split ';' } else { @() }
$alreadyOnPath = $pathParts | Where-Object { $_ -and ([IO.Path]::GetFullPath($_) -ieq [IO.Path]::GetFullPath($installDir)) }

if (-not $alreadyOnPath) {
    Write-Status "⚠ $installDir is not on your user PATH — adding it" 'DarkYellow'
    $newPath = if ($userPath) { "$userPath;$installDir" } else { $installDir }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    $env:Path = "$env:Path;$installDir"
    Write-Status '  open a new terminal for PATH changes to take effect in other apps' 'DarkGray'
    Write-Host ''
}

Write-Status 'next:  hackfetch -setup' 'DarkGray'
Write-Host ''
