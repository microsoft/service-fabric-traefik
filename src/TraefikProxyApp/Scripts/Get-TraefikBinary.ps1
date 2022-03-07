#
# Simple script to pull down the Traefik Binary for deployment. 
# Specify the Traefik version and fileName to retrieve from releases
# 

param(
    [string]$version,
    [string]$fileName
)

while (!($version)) {
    Write-Host "Review current Traefik releases:" -foregroundcolor Green
    Write-Host "https://github.com/containous/traefik/releases"
    Write-Host "Please provide the release tag (e.g. 'v1.6.0-rc6' or 'v1.5.4') of the Traefik release you wish to download: " -foregroundcolor Green -NoNewline
    $version = Read-Host 
}

while (!($fileName)) {
    Write-Host "Review current Traefik OS and Architecture support for Traefik $version :" -foregroundcolor Green
    Write-Host "https://github.com/containous/traefik/releases/$version"
    Write-Host "Please provide the file name (e.g. 'traefik_v2.6.1_windows_amd64.zip' or 'traefik_v2.6.1_linux_amd64.tar.gz') of the Traefik release you wish to download: " -foregroundcolor Green -NoNewline
    $fileName = Read-Host 
}

$isWindows = If ($fileName.Contains("windows")) {$true} Else {$false}

#Github and other sites now require tls1.2 without this line the script will fail with an SSL error. 
[Net.ServicePointManager]::SecurityProtocol = "tls12, tls11, tls"

$traefikBaseUrl = "https://github.com/traefik/traefik/releases/download/"
$url = $traefikBaseUrl + $version + "/" + $fileName

Write-Host "Downloading Traefik Binary from: " -foregroundcolor Green
Write-Host $url

$traefikPath = "/../ApplicationPackageRoot/TraefikPkg/Code"
$outfile = $PSScriptRoot + "/" + $fileName

Write-Host "Downloading zip file" -foregroundcolor Green
Invoke-WebRequest -Uri $url -OutFile $outfile -UseBasicParsing
Write-Host "Download complete, files:" -foregroundcolor Green
Write-Host $outfile

Write-Host Extracting release files -foregroundcolor Green

#Files to delete after expanding zip file
$changeLog = "$PSScriptRoot" + "/" + "CHANGELOG.md"
$license = "$PSScriptRoot" + "/" + "LICENSE.md"
if ($isWindows){
    Expand-Archive -Path $PSScriptRoot/$fileName -DestinationPath $PSScriptRoot  -Force
    #$name = $fileName.Replace(".zip","")
    #$traefikExePath = $PSScriptRoot + "/" + $name + "/" + "traefik.exe"
    $traefikExePath = $PSScriptRoot + "/" + "traefik.exe"
    Move-Item $traefikExePath -Destination $PSScriptRoot/$traefikPath -Force
  
    #Removing temp files
    Remove-Item $PSScriptRoot/$fileName -Force
    #Remove-Item $PSScriptRoot/$name -Recurse -Force
    Remove-Item $changeLog -Force
    Remove-Item $license -Force
} else{
    tar -xvzf $PSScriptRoot/$fileName -C $PSScriptRoot
    #$name = $fileName.Replace(".tar.gz","")
    $traefikExePath = $PSScriptRoot + "/" + "traefik"
    Move-Item $traefikExePath -Destination $PSScriptRoot/$traefikPath -Force
    # Removing temp files
    Remove-Item $PSScriptRoot/$fileName -Force
    Remove-Item $changeLog -Force
    Remove-Item $license -Force
}
