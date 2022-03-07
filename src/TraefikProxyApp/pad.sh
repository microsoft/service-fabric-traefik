# Linux
sfctl cluster select --endpoint http://localhost:19080 #--pem /var/lib/sfcerts/cert.pem --no-verify

sfctl application delete --application-id traefik --force-remove FORCE_REMOVE
sfctl application unprovision --application-type-name TraefikType --application-type-version 0.1.0-beta

sfctl application upload --path ./traefik #--show-progress
sfctl application provision --application-type-build-path traefik

p='{
    "ReverseProxy_InstanceCount":"3",
    "ReverseProxy_FetcherEndpoint":"7777",
    "ReverseProxy_HttpPort":"8080",
    "ReverseProxy_CertificateSearchKeyword":"",
    "ClusterEndpoint":"https://localhost:19080",
    "ClientCertificate":"/var/lib/sfcerts/64E6BDA1F105807DF290F69C52D3D3A93D2449E2.pem",
    "ClientCertificatePK":"/var/lib/sfcerts/64E6BDA1F105807DF290F69C52D3D3A93D2449E2.prv",
    "ReverseProxy_EnableDashboard":"true"
}'
    #"ReverseProxy_PlacementConstraints":"NodeType == NT2"


sfctl application create --app-name fabric:/traefik --app-type TraefikType --app-version 0.1.0-beta --parameters "$p"


# Sample pinger app for validating (navidate to /pinger/PingerService/id)
sfctl application delete --application-id pinger7000 --force-remove FORCE_REMOVE
sfctl application unprovision --application-type-name PingerApplicationType --application-type-version 1.0

sfctl application upload --path ./pinger-traefik #--show-progress
sfctl application provision --application-type-build-path pinger-traefik
sfctl application create --app-name fabric:/pinger7000 --app-type PingerApplicationType --app-version 1.0 --parameters '{"Pinger_Port":"7000"}'


# Go to SF app dir
# /mnt/sfroot 
more FabricHostSettings.xml | grep Cert|grep Clus
    <Parameter Name="ClusterCertificateAclingInterval" Value="86400" />
    <Parameter Name="ClusterCertThumbprints" Value="CB93FC5CAAB491B83570089F54F0CDDF872C9B82,5197317B5263E5ED8790053B1EFF34C0456F2B4F" />

dir /var/lib/sfcerts/CB*
/var/lib/sfcerts/CB93FC5CAAB491B83570089F54F0CDDF872C9B82.crt
/var/lib/sfcerts/CB93FC5CAAB491B83570089F54F0CDDF872C9B82.pem
/var/lib/sfcerts/CB93FC5CAAB491B83570089F54F0CDDF872C9B82.pfx
/var/lib/sfcerts/CB93FC5CAAB491B83570089F54F0CDDF872C9B82.prv

more /datadrive/svcfab/data/FabricHostSettings.xml | grep Cert|grep Clus

sfctl cluster select --endpoint https://localhost:19080 --no-verify --pem /var/lib/sfcerts/CB93FC5CAAB491B83570089F54F0CDDF872C9B82.pem
