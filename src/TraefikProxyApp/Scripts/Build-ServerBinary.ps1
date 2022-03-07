#
# Simple script to set Go env variables to cross compile Go Programs
# It is assumed Go has been installed and Go commands can be used under /script folder 
# Specify the go OS and ARCH to be used in the go ENV
# 

param(
    [string]$goOS,
    [string]$goARCH
)

while (!($goOS)) {
    Write-Host "Review current suported GO OS:" -foregroundcolor Green
    Write-Host "https://go.dev/doc/install/source"
    Write-Host "Please provide the GO OS (e.g. 'linux' or 'windows') that you wish to use in the GO env to compile the Go program: " -foregroundcolor Green -NoNewline
    $goOS = Read-Host 
}

while (!($goARCH)) {
    Write-Host "Review current suported GO ARCH:" -foregroundcolor Green
    Write-Host "https://go.dev/doc/install/source"
    Write-Host "Please provide the GO ARCH (e.g. 'amd64' or 'arm') that you wish to use in the GO env to compile the Go program: " -foregroundcolor Green -NoNewline
    $goARCH = Read-Host 
}

Set-Location -Path "./src/serviceFabricDiscoveryService/cmd"
Get-ChildItem

$Env:GOOS = $goOS
$Env:GOARCH = $goARCH
Write-Host "Current Go env variables: " -foregroundcolor Green
Write-Host go env

Write-Host $PSScriptRoot/$serverPath
$serverPath = "/../ApplicationPackageRoot/TraefikPkg/Fetcher.Code"
Move-Item "server.exe" -Destination $PSScriptRoot/$serverPath -Force
Get-ChildItem $PSScriptRoot/$serverPath

