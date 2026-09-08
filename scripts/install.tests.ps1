$ErrorActionPreference = 'Stop'

$testRoot = Join-Path ([IO.Path]::GetTempPath()) "skills-manager-installer-test-$([Guid]::NewGuid())"
$fixtureDir = Join-Path $testRoot 'release'
$archiveRoot = Join-Path $testRoot 'archive'
$installDir = Join-Path $testRoot 'bin'
New-Item -ItemType Directory -Path $fixtureDir, $archiveRoot, $installDir -Force | Out-Null

try {
    $binaryPath = Join-Path $archiveRoot 'skills-manager.exe'
    Set-Content -LiteralPath $binaryPath -Value 'fixture binary' -NoNewline
    $archiveName = 'skills-manager_Windows_x86_64.zip'
    $archivePath = Join-Path $fixtureDir $archiveName
    Compress-Archive -LiteralPath $binaryPath -DestinationPath $archivePath -Force
    $checksum = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
    Set-Content -LiteralPath (Join-Path $fixtureDir 'skills-manager_checksums.txt') -Value "$checksum  $archiveName" -NoNewline
    $installed = Join-Path $installDir 'skills-manager.exe'
    Set-Content -LiteralPath $installed -Value 'binário anterior' -NoNewline

    $fakeCurl = Join-Path $testRoot 'fake-curl.cmd'
    @("@echo off", 'set "source_url="', 'set "destination="', ':loop', 'if "%~1"=="" goto copy', 'if /I "%~1"=="-o" (set "destination=%~2" & shift & shift & goto loop)', 'set "source_url=%~1"', 'shift', 'goto loop', ':copy', 'for %%F in ("%source_url%") do set "source_name=%%~nxF"', 'copy /Y "%SKILLS_MANAGER_FIXTURE_DIR%\%source_name%" "%destination%" >NUL') | Set-Content -LiteralPath $fakeCurl -Encoding ASCII

    $env:SKILLS_MANAGER_TEST_MODE = '1'
    $env:SKILLS_MANAGER_DOWNLOAD_BASE_URL = 'fixture://release'
    $env:SKILLS_MANAGER_CURL_BIN = $fakeCurl
    $env:SKILLS_MANAGER_FIXTURE_DIR = $fixtureDir
    $env:SKILLS_MANAGER_INSTALL_DIR = $installDir
    $env:SKILLS_MANAGER_VERSION = 'v0.1.0'
    $badChecksum = ('0' * 64) -join ''
    Set-Content -LiteralPath (Join-Path $fixtureDir 'skills-manager_checksums.txt') -Value "$badChecksum  $archiveName" -NoNewline
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'install.ps1') 2>$null
    if ($LASTEXITCODE -eq 0) { throw 'teste do instalador: checksum divergente deveria falhar' }
    if ((Get-Content -LiteralPath $installed -Raw) -ne 'binário anterior') { throw 'teste do instalador: checksum divergente substituiu o binário' }
    Set-Content -LiteralPath (Join-Path $fixtureDir 'skills-manager_checksums.txt') -Value "$checksum  $archiveName" -NoNewline
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'install.ps1')
    if (-not (Test-Path -LiteralPath $installed -PathType Leaf)) { throw 'teste do instalador: binário não instalado' }
    Write-Output 'instalador Windows validado com fixtures locais'
} finally {
    Remove-Item -LiteralPath $testRoot -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item Env:SKILLS_MANAGER_TEST_MODE, Env:SKILLS_MANAGER_DOWNLOAD_BASE_URL, Env:SKILLS_MANAGER_CURL_BIN, Env:SKILLS_MANAGER_FIXTURE_DIR, Env:SKILLS_MANAGER_INSTALL_DIR, Env:SKILLS_MANAGER_VERSION -ErrorAction SilentlyContinue
}
