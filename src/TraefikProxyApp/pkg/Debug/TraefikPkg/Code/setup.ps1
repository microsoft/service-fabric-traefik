
$vars = @("TRAEFIK_HTTP_PORT", "TRAEFIK_ENABLE_DASHBOARD", "Fabric_Folder_App_Work")

$template = Get-Content traefik-template.yaml -Raw

foreach ($i in $vars) { 
    $v = [System.Environment]::GetEnvironmentVariable($i)
    Write-Host "Replacing: [$i] with [$v]"
    $template=$template.Replace("<$i>",$v)
}

Write-Host "Traefik config file:"
$template

Set-Content ..\traefik.yaml -Value $template

$workFolder = $env:Fabric_Folder_App_Work

mkdir -Force ..\dynConfig
copy dynConfig\dyn.yaml $workFolder\dyn.yaml



