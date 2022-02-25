Connect-ServiceFabricCluster

# Register and run the Traefik Application
Remove-ServiceFabricApplication -ApplicationName fabric:/traefik -Force
Unregister-ServiceFabricApplicationType -ApplicationTypeName TraefikType -ApplicationTypeVersion 0.1.0-beta -Force

Copy-ServiceFabricApplicationPackage -ApplicationPackagePath .\traefik\ # -ApplicationPackagePathInImageStore traefik
Register-ServiceFabricApplicationType -ApplicationPathInImageStore traefik
$p = @{
    ReverseProxy_FetcherEndpoint="7777"
    ReverseProxy_HttpPort="8080"
    ReverseProxy_CertificateSearchKeyword=""
    ClusterEndpoint="https://localhost:19080"
    CertStoreSearchKey="sfmc"
    ClientCertificate=""
    ClientCertificatePK=""
    ReverseProxy_EnableDashboard="true"
    #ReverseProxy_PlacementConstraints="NodeType == NT2"
}
$p
New-ServiceFabricApplication -ApplicationName fabric:/traefik -ApplicationTypeName TraefikType -ApplicationTypeVersion 0.1.0-beta -ApplicationParameter $p


# Sample pinger app for validating (navidate to /pinger7000/PingerService/id)
for ($i=7000; $i -le 7000; $i++) {
    Remove-ServiceFabricApplication -ApplicationName fabric:/pinger$i -Force
}
Unregister-ServiceFabricApplicationType -ApplicationTypeName PingerApplicationType -ApplicationTypeVersion 1.0 -Force

Copy-ServiceFabricApplicationPackage -ApplicationPackagePath .\pinger-traefik\ #-ApplicationPackagePathInImageStore pinger
Register-ServiceFabricApplicationType -ApplicationPathInImageStore pinger-traefik

$pp = @{
    "Pinger_Instance_Count"="-1"
    #"Pinger_PlacementConstraints"= "NodeType == NT2"
    #"Pinger_Port"="7000"
}
for ($i=7000; $i -le 7000; $i++) {
    $p = $pp + @{Pinger_Port="$i"}
    New-ServiceFabricApplication -ApplicationName fabric:/pinger$i -ApplicationTypeName PingerApplicationType -ApplicationTypeVersion 1.0 -ApplicationParameter $p
}
