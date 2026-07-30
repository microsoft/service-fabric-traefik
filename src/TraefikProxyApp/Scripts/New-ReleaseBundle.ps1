<#
.SYNOPSIS
    Builds, packages, and assembles a deployable ServiceFabricTraefik release bundle.

.DESCRIPTION
    End-to-end release helper. Unless -SkipBuild is given, it:
      1. Builds server.exe with the version injected via -ldflags.
      2. Stages server.exe into TraefikPkg\Fetcher.Code.
      3. Optionally downloads traefik.exe (when -TraefikVersion/-TraefikFileName are given).
      4. Restores NuGet packages (needed for the .sfproj Package target).
      5. Runs the msbuild Package target.
    It then assembles the packaged output into a versioned release layout. With -WindowsOnly it
    prunes the files that are inert on a Windows cluster (Linux manifests/scripts and unused
    Windows batch files) so the drop matches the official Windows release:
    <OutputPath>\windows\TraefikProxyApp\...

    Note: official releases are additionally code-signed by an internal Azure DevOps pipeline;
    this script does not sign binaries.

.PARAMETER Version
    Release version. Injected into server.exe and used for the zip name. E.g. 1.2.0

.PARAMETER Configuration
    Build configuration (Release or Debug). Default: Release.

.PARAMETER Platform
    Build platform. Default: x64.

.PARAMETER TraefikVersion
    Traefik release tag to download. Default: v2.11.53. Pass an empty string ('') to skip the
    download and reuse the already-staged traefik.exe.

.PARAMETER TraefikFileName
    Traefik release asset name. Default: traefik_v2.11.53_windows_amd64.zip.

.PARAMETER SkipBuild
    Skip the go/nuget/msbuild steps and just assemble an already-built package. Default: off.

.PARAMETER Clean
    Delete previous build outputs (pkg, obj, bin, dist) before building. Ignored with -SkipBuild.
    Default: on (override with -Clean:$false).

.PARAMETER WindowsOnly
    Prune the Linux/unused files and lay the package out under a windows\ folder. Default: on
    (override with -WindowsOnly:$false).

.PARAMETER Zip
    Also produce a zip of the assembled bundle. Default: on (override with -Zip:$false).

.PARAMETER PackagePath
    Path to the msbuild Package output. Default: pkg\<Configuration>.

.PARAMETER OutputPath
    Root output directory for the assembled bundle. Default: <repo>\out.

.EXAMPLE
    # Full clean build + Windows-only bundle + zip (all defaults)
    .\New-ReleaseBundle.ps1 -Version 1.2.0

.EXAMPLE
    # Just re-assemble an already-built package
    .\New-ReleaseBundle.ps1 -SkipBuild -WindowsOnly
#>
param(
    [string]$Version = '1.2.0',
    [string]$Configuration = 'Release',
    [string]$Platform = 'x64',
    [string]$TraefikVersion = 'v2.11.53',
    [string]$TraefikFileName = 'traefik_v2.11.53_windows_amd64.zip',
    [switch]$SkipBuild = $false,
    [switch]$Clean = $true,
    [switch]$WindowsOnly = $true,
    [switch]$Zip = $true,
    [string]$PackagePath = "$PSScriptRoot\..\pkg\$Configuration",
    [string]$OutputPath = "$PSScriptRoot\..\..\..\out"
)

$ErrorActionPreference = 'Stop'

# --- Paths -------------------------------------------------------------------
$repoRoot   = (Resolve-Path "$PSScriptRoot\..\..\..").Path
$appProj    = Join-Path $repoRoot 'src\TraefikProxyApp\TraefikProxyApp.sfproj'
$sln        = Join-Path $repoRoot 'TraefikSF.sln'
$svcDir     = Join-Path $repoRoot 'src\serviceFabricDiscoveryService'
$fetcherDir = Join-Path $repoRoot 'src\TraefikProxyApp\ApplicationPackageRoot\TraefikPkg\Fetcher.Code'
$goModule   = 'github.com/microsoft/service-fabric-traefik/serviceFabricDiscoveryService'

function Resolve-MSBuild {
    $cmd = Get-Command msbuild -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio\Installer\vswhere.exe'
    if (Test-Path $vswhere) {
        $p = & $vswhere -latest -requires Microsoft.Component.MSBuild -find 'MSBuild\**\Bin\MSBuild.exe' |
             Select-Object -First 1
        if ($p) { return $p }
    }
    throw "MSBuild not found. Open a 'Developer PowerShell for VS', or install VS with MSBuild."
}

# --- Build (unless -SkipBuild) ----------------------------------------------
if (-not $SkipBuild) {
    # Make sure `go` and `nuget` are reachable even in a fresh shell.
    $env:Path = [Environment]::GetEnvironmentVariable('Path','Machine') + ';' +
                [Environment]::GetEnvironmentVariable('Path','User')

    if ($Clean) {
        Write-Host "==> Cleaning previous build outputs" -ForegroundColor Cyan
        @(
            (Join-Path $repoRoot 'src\TraefikProxyApp\pkg'),
            (Join-Path $repoRoot 'src\TraefikProxyApp\obj'),
            (Join-Path $repoRoot 'src\TraefikProxyApp\bin'),
            (Join-Path $svcDir 'dist')
        ) | Where-Object { Test-Path $_ } | ForEach-Object {
            Remove-Item $_ -Recurse -Force
            Write-Host "  removed $_" -ForegroundColor DarkGray
        }
    }

    Write-Host "==> Building server.exe (version $Version)" -ForegroundColor Cyan
    $ldflags = "-X $goModule/version.Version=$Version"
    & go build -C $svcDir -ldflags $ldflags -o 'dist\server.exe' '.\cmd'
    if ($LASTEXITCODE) { throw "go build failed ($LASTEXITCODE)" }
    Copy-Item (Join-Path $svcDir 'dist\server.exe') (Join-Path $fetcherDir 'server.exe') -Force

    if ($TraefikVersion -and $TraefikFileName) {
        Write-Host "==> Downloading Traefik $TraefikVersion ($TraefikFileName)" -ForegroundColor Cyan
        & (Join-Path $PSScriptRoot 'Get-TraefikBinary.ps1') -version $TraefikVersion -fileName $TraefikFileName
    } else {
        Write-Host "==> Skipping Traefik download (reusing already-staged traefik.exe)" -ForegroundColor DarkYellow
    }

    Write-Host "==> Restoring NuGet packages" -ForegroundColor Cyan
    & nuget restore $sln
    if ($LASTEXITCODE) { throw "nuget restore failed ($LASTEXITCODE)" }

    Write-Host "==> Packaging TraefikProxyApp ($Configuration|$Platform)" -ForegroundColor Cyan
    $msbuild = Resolve-MSBuild
    & $msbuild $appProj /t:Package "/p:Configuration=$Configuration" "/p:Platform=$Platform" /verbosity:minimal
    if ($LASTEXITCODE) { throw "msbuild package failed ($LASTEXITCODE)" }
}

# Files that are inert on a Windows cluster and are excluded from the official Windows drop.
$windowsExclusions = @(
    'ApplicationManifestLinux.xml',
    'TraefikPkg\ServiceManifestLinux.xml',
    'TraefikPkg\Code\setup_traefik.sh',
    'TraefikPkg\Code\traefik.sh',
    'TraefikPkg\Fetcher.Code\server.sh',
    'TraefikPkg\Code\setup_traefik.bat',
    'TraefikPkg\Code\startproxy.bat'
)

if (-not (Test-Path (Join-Path $PackagePath 'ApplicationManifest.xml'))) {
    throw "No package found at '$PackagePath'. Build it first: " +
          "msbuild TraefikProxyApp.sfproj /t:Package /p:Configuration=Release /p:Platform=x64"
}
$PackagePath = (Resolve-Path $PackagePath).Path

$osFolder = if ($WindowsOnly) { 'windows' } else { 'any' }
$appDest  = Join-Path $OutputPath "$osFolder\TraefikProxyApp"

if (Test-Path $appDest) { Remove-Item $appDest -Recurse -Force }
New-Item -ItemType Directory -Force -Path $appDest | Out-Null

Copy-Item (Join-Path $PackagePath '*') $appDest -Recurse -Force

if ($WindowsOnly) {
    foreach ($rel in $windowsExclusions) {
        $f = Join-Path $appDest $rel
        if (Test-Path $f) {
            Remove-Item $f -Force
            Write-Host "  pruned $rel" -ForegroundColor DarkGray
        }
    }
}

Write-Host "Assembled '$osFolder' bundle at: $appDest" -ForegroundColor Green

if ($Zip) {
    $osRoot  = Join-Path $OutputPath $osFolder
    $zipPath = Join-Path $OutputPath "service-fabric-traefik-$Version-$osFolder.zip"
    if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
    Compress-Archive -Path $osRoot -DestinationPath $zipPath
    Write-Host "Zipped: $zipPath" -ForegroundColor Green
}
