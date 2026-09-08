$ErrorActionPreference = 'Stop'

$projectName = 'skills-manager'
$repository = if ([string]::IsNullOrWhiteSpace($env:SKILLS_MANAGER_REPO)) { 'danielpavone/skills-manager' } else { $env:SKILLS_MANAGER_REPO }
$version = if ([string]::IsNullOrWhiteSpace($env:SKILLS_MANAGER_VERSION)) { 'latest' } else { $env:SKILLS_MANAGER_VERSION }
$installDir = $env:SKILLS_MANAGER_INSTALL_DIR
$downloadBase = $env:SKILLS_MANAGER_DOWNLOAD_BASE_URL
$curlBinary = if ([string]::IsNullOrWhiteSpace($env:SKILLS_MANAGER_CURL_BIN)) { 'curl.exe' } else { $env:SKILLS_MANAGER_CURL_BIN }
$testMode = $env:SKILLS_MANAGER_TEST_MODE -eq '1'

function Fail([string]$Message) {
    throw "skills-manager installer: $Message"
}

function Resolve-DownloadBase {
    if (-not [string]::IsNullOrWhiteSpace($script:downloadBase)) { return }
    if ($script:version -eq 'latest') {
        $script:downloadBase = "https://github.com/$($script:repository)/releases/latest/download"
        return
    }
    $script:downloadBase = "https://github.com/$($script:repository)/releases/download/$($script:version)"
}

function Validate-DownloadBase {
    if ($script:downloadBase -like 'https://*') { return }
    if ($script:testMode) { return }
    Fail "URL-base '$($script:downloadBase)' não é HTTPS; esperado uma URL https:// para downloads"
}

function Download-File([string]$Url, [string]$Destination) {
    & $script:curlBinary -fsSL $Url -o $Destination
    if ($LASTEXITCODE -ne 0) {
        Fail "falha ao baixar '$Url'; esperado um artefato acessível por HTTPS"
    }
}

function Verify-Checksum([string]$ChecksumPath, [string]$ArtifactPath, [string]$ArtifactName) {
    $entry = Get-Content -LiteralPath $ChecksumPath | Where-Object { $_ -match "\s$([regex]::Escape($ArtifactName))$" } | Select-Object -First 1
    if ([string]::IsNullOrWhiteSpace($entry)) {
        Fail "checksum de '$ArtifactName' ausente em '$ChecksumPath'; esperado SHA-256 publicado"
    }
    $expected = ($entry -split '\s+')[0]
    if ($expected -notmatch '^[0-9a-fA-F]{64}$') {
        Fail "checksum '$expected' para '$ArtifactName' tem formato inválido; esperado 64 caracteres hexadecimais"
    }
    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $ArtifactPath).Hash
    if ($actual -ne $expected) {
        Fail "checksum divergente para '$ArtifactName': obtido '$actual', esperado '$expected'; binário existente não foi substituído"
    }
}

function Resolve-InstallDirectory {
    if (-not [string]::IsNullOrWhiteSpace($script:installDir)) { return }
    $script:installDir = Join-Path ([Environment]::GetFolderPath('UserProfile')) '.local\bin'
}

function Install-Archive([string]$ArchivePath) {
    $stagingDir = Join-Path ([IO.Path]::GetTempPath()) "skills-manager-$([Guid]::NewGuid())"
    New-Item -ItemType Directory -Path $stagingDir -Force | Out-Null
    try {
        Expand-Archive -LiteralPath $ArchivePath -DestinationPath $stagingDir -Force
        $binaryPath = Join-Path $stagingDir "$script:projectName.exe"
        if (-not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) {
            Fail "arquivo '$($script:projectName).exe' não encontrado em '$ArchivePath'; esperado o binário na raiz do arquivo"
        }
        New-Item -ItemType Directory -Path $script:installDir -Force | Out-Null
        $targetPath = Join-Path $script:installDir "$script:projectName.exe"
        $temporaryTarget = Join-Path $script:installDir ".$($script:projectName).new.$PID.exe"
        Copy-Item -LiteralPath $binaryPath -Destination $temporaryTarget -Force
        Move-Item -LiteralPath $temporaryTarget -Destination $targetPath -Force
        Write-Output "skills-manager instalado em $targetPath"
    } finally {
        Remove-Item -LiteralPath $stagingDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Resolve-DownloadBase
Validate-DownloadBase
Resolve-InstallDirectory
$architecture = [Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
if ($architecture -eq 'X64') { $archiveArchitecture = 'x86_64' }
elseif ($architecture -eq 'Arm64') { $archiveArchitecture = 'arm64' }
else { Fail "arquitetura '$architecture' não suportada; esperado X64 ou Arm64" }
$archiveName = "$projectName`_Windows`_$archiveArchitecture.zip"
$checksumName = "${projectName}_checksums.txt"
$downloadDir = Join-Path ([IO.Path]::GetTempPath()) "skills-manager-download-$([Guid]::NewGuid())"
New-Item -ItemType Directory -Path $downloadDir -Force | Out-Null
try {
    $checksumPath = Join-Path $downloadDir $checksumName
    $archivePath = Join-Path $downloadDir $archiveName
    Download-File "$downloadBase/$checksumName" $checksumPath
    Download-File "$downloadBase/$archiveName" $archivePath
    Verify-Checksum $checksumPath $archivePath $archiveName
    Install-Archive $archivePath
} finally {
    Remove-Item -LiteralPath $downloadDir -Recurse -Force -ErrorAction SilentlyContinue
}
