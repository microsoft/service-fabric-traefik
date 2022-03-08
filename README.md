# ServiceFabricTraefik 1.0.0

The reverse proxy is an application, supplied out of band from the service fabric distribution, that customers deploy to their clusters and handles proxying traffic to backend services. The service, that potentially runs on every node in the cluster, takes care of handling endpoint resolution, automatic retry, and other connection failures on behalf of the clients. The reverse proxy can be configured to apply various policies as it handles requests from client services.

Using a reverse proxy allows the client service to use any client-side HTTP communication libraries and does not require special resolution and retry logic in the service. The reverse proxy is mostly a terminating endpoint for the TLS connections unless the TCP option is used.

>Note that, at this time, this is a reverse proxy built-in replacement and not a generic service fabric “gateway” able to handle partition queries, but that might be added (via customer written plugins or similar) in the future.


## How it works 
As of this release, the services need to be explicitly exposed via [service extension labels](), enabling the proxying (HTTP/TCP) functionality for a particular service and endpoint. With the right labels’ setup, the reverse proxy will expose one or more endpoints on the local nodes for client services to use. The ports can then be exposed to the load balancer in order to get the services available outside of the cluster. The required certificates needed should be already deployed to the nodes where the proxy is running as is the case with any other Service Fabric application.

## Using the application  

You can clone the repo, build, and deploy or simply grab the latest [ZIP/SFPKG application](https://github.com/microsoft/service-fabric-traefik/releases/latest) from Releases section, modify configs, and deploy.

![alt text](/Documentation/Images/traefik-cluster-view.png "Cluster View UI")

![alt text](/Documentation/Images/traefik-service-view.png "Cluster Service View UI")


## Deploy it using PowerShell  

After either downloading the sfapp package from the releases or cloning the repo and building (code will be up shortly), you need to adjust the configuration settings to meet to your needs (this means changing settings in Settings.xml, ApplicationManifest.xml and any other changes needed for the traefik-template.yaml configuration).

>If you need a quick test cluster, you can deploy a test Service Fabric managed cluster following the instructions from here: [SFMC](https://docs.microsoft.com/en-us/azure/service-fabric/quickstart-managed-cluster-template), or via this template if you already have a client certificate and thumbprint available: [Deploy](https://portal.azure.com/#create/Microsoft.Template/uri/https%3A%2F%2Fraw.githubusercontent.com%2FAzure-Samples%2Fservice-fabric-cluster-templates%2Fmaster%2FSF-Managed-Basic-SKU-1-NT%2Fazuredeploy.json)

>Retrieve the cluster certificate TP using:  $serverThumbprint = (Get-AzResource -ResourceId /subscriptions/$SUBSCRIPTION/resourceGroups/$RESOURCEGROUP/providers/Microsoft.ServiceFabric/managedclusters/$CLUSTERNAME).Properties.clusterCertificateThumbprints

```PowerShell

#cd to the top level directory where you downloaded the package zip
cd \downloads

#Expand the zip file

Expand-Archive .\service-fabric-traefik.zip -Force

#cd to the directory that holds the application package

cd .\service-fabric-traefik\windows\

#create a $appPath variable that points to the application location:
#E.g., for Windows deployments:

$appPath = "C:\downloads\service-fabric-traefik\windows\TraefikProxyApp"

#For Linux deployments:

#$appPath = "C:\downloads\service-fabric-traefik\linux\TraefikProxyApp"

#Connect to target cluster, for example:

Connect-ServiceFabricCluster -ConnectionEndpoint @('sf-win-cluster.westus2.cloudapp.azure.com:19000') -X509Credential -FindType FindByThumbprint -FindValue '[Client_TP]' -StoreLocation LocalMachine -StoreName 'My' # -ServerCertThumbprint [Server_TP]

# Use this to remove a previous Traefik Application
#Remove-ServiceFabricApplication -ApplicationName fabric:/traefik -Force
#Unregister-ServiceFabricApplicationType -ApplicationTypeName TraefikType -ApplicationTypeVersion 1.0.0 -Force

#Copy and register and run the Traefik Application
Copy-ServiceFabricApplicationPackage -CompressPackage -ApplicationPackagePath $appPath # -ApplicationPackagePathInImageStore traefik
Register-ServiceFabricApplicationType -ApplicationPathInImageStore traefik

#Fill the right values that are suitable for your cluster and application (the default ones below will work without modification if you used a Service Fabric managed cluster Quickstart template with one node type. Adjust the placement constraints to use other node types)
$p = @{
    ReverseProxy_InstanceCount="1"
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
New-ServiceFabricApplication -ApplicationName fabric:/traefik -ApplicationTypeName TraefikType -ApplicationTypeVersion 1.0.0 -ApplicationParameter $p


#OR if updating existing version:  

Start-ServiceFabricApplicationUpgrade -ApplicationName fabric:/traefik -ApplicationTypeVersion 1.0.0 -Monitored -FailureAction rollback
```  

## Add the right labels to your services

### ServiceManifest file

This is a sample SF enabled service showing the currently supported labels. If the sf name is fabric:/pinger/PingerService, the endpoint [endpointName] will be expose at that prefix: '/pinger/PingerService/'

```xml
  ...
  <ServiceTypes>
    <StatelessServiceType ServiceTypeName="PingerServiceType" UseImplicitHost="true">
      <Extensions>
        <Extension Name="traefik">
        <Labels xmlns="http://schemas.microsoft.com/2015/03/fabact-no-schema">
          <Label Key="traefik.http.defaultEP">true</Label>
          <Label Key="traefik.http.defaultEP.service.loadbalancer.passhostheader">true</Label>
          <Label Key="traefik.http.defaultEP.service.loadbalancer.healthcheck.path">/</Label>
          <Label Key="traefik.http.defaultEP.service.loadbalancer.healthcheck.interval">10s</Label>
          <Label Key="traefik.http.defaultEP.service.loadbalancer.healthcheck.scheme">http</Label>
        </Labels>
        </Extension>
      </Extensions>
    </StatelessServiceType>
  </ServiceTypes>
  ...
```

---

The only required label to expose a service via the reverse proxy is the **traefik.http.[endpointName]** set to true. Setting only this label will expose the service on a well known path and handle the basic scenarios.

```
http(s)://<Cluster FQDN | internal IP>:Port/ApplicationInstanceName/ServiceInstanceName?PartitionGuid=xxxxx
```

If you need to change the routes or add middleware then you can add different labels configuring them.


## Supported Labels

*Http enable section*

* **traefik.http.[endpointName]**    Enables exposing an http service via the reverse proxy ['true']

Router section

* **traefik.http.[endpointName].router.rule**    Traefik rule to apply [PathPrefix(`/api`))]. This rule is added on top of the default path generation. If this is set, you **have** to define a middleware to remove the prefix for the service to receive the stripped path.
* **traefik.http.[endpointName].router.tls.options**    Enable TLS on the route ['true'/'false']. T

*Loadbalancer section*

* **traefik.http.[endpointName].loadbalancer.passhostheader**          passhostheaders ['true'/'false']
* **traefik.http.[endpointName].loadbalancer.healthcheck.path**        Healthcheck endpoint path ['/healtz']
* **traefik.http.[endpointName].loadbalancer.healthcheck.interval**    Healthcheck interval ['10s']
* **traefik.http.[endpointName].loadbalancer.healthcheck.scheme**      Healthcheck scheme ['http']

*Middleware section*

* **traefik.http.[endpointName].middlewares.[Yourt_Middleware_Name].stripPrefix.prefixes**    prefix to strip ['/api']

## Sample Test application

A sample test application, that is included in the release, can be deployed to test everything is working alright. After deployment, you should be able to hit it at:

https://your-cluster:8080/pinger0/PingerService/id

>Note that the service is going to be exposed on https since the service has a label for the route.tls option. You can explore that looking at the service manifest for this app.


```Powershell

# Sample pinger app for validating (navidate to /pinger0/PingerService/id on https)
#Remove-ServiceFabricApplication -ApplicationName fabric:/pinger$i -Force
#Unregister-ServiceFabricApplicationType -ApplicationTypeName PingerApplicationType -ApplicationTypeVersion 1.0 -Force

$appPath = "C:\downloads\service-fabric-traefik\windows\pinger-traefik"

Copy-ServiceFabricApplicationPackage -CompressPackage -ApplicationPackagePath $appPath -ApplicationPackagePathInImageStore pinger-traefik
Register-ServiceFabricApplicationType -ApplicationPathInImageStore pinger-traefik

$p = @{
    "Pinger_Instance_Count"="3"
    "Pinger_Port"="7000"
    #"Pinger_PlacementConstraints"= "NodeType == NT2"
}

New-ServiceFabricApplication -ApplicationName fabric:/pinger0 -ApplicationTypeName PingerApplicationType -ApplicationTypeVersion 1.0 -ApplicationParameter $p


```

# Traefik Reverse Proxy for Service Fabric Integration

## Project Structure

This repo includes:

* `TraefikProxyApp`: an example Service Fabric application, consisting of:
  * `server.exe`: The guest executable that fetches endpoint information from Service Fabric that are configured to use Traefik via Service Manifest Extensions, and exposes the summarized configurations for `traefik.exe` to consume in real-time
  * `traefik.exe`: The guest executable that implements a Reverse Proxy using Traefik. It reads configuration fetched from server.exe

## Building & Getting latest Executable

1. At the moment application package comes with server.exe, which includes latest serviceFabricDiscovery changes. Can also add changes to serviceFabricDiscoveryService and then manually build binary. Use similar steps mentioned under method 2:Using go [Building and Testing - Traefik](https://doc.traefik.io/traefik/contributing/building-testing/#build-traefik), to build server (e.g "go build .\cmd\server"). Once the server executable has been built, replace the default executable under ./src/TraefikProxyApp/ApplicationPackageRoot/TraefikPkg/Fetcher.Code

2. Get latest traefik.exe from [traefik github page](https://github.com/traefik/traefik/releases) and place under ./src/TraefikProxyApp/ApplicationPackageRoot/TraefikPkg/Code


## Updating manifest file
TraefikProxyApp comes with an ApplicationManifest and ServiceManifest for both windows and linux. By default the .xml files contains the content for a windows cluster deployment. For a linux deployment need to replace with the linux xml content.

Also, you can open `TraefikSF.sln` at the root of the repo with Visual Studio 2019.
Running builds and deploying to local or remote SF clusters instead of using local powershell prompt.

## serviceFabricDiscoveryService
serviceFabricDiscoveryService is a service that connects to a Service Fabric cluster and exposes discovery data and changes [async] over websockets or, locally, via a file. Changes on names (applications/services) and endpoint mapping information is sent as messages over the websocket as they happen, the client doesn't have to poll the server.

The service exposes several websocket routes that have specific functionality.

# Running the server
```
NAME:
   discoveryService - exposes service fabric application and service metadata over websockets

USAGE:
   server.exe [global options] command [command options] [arguments...]

COMMANDS:
   run      runs as a server
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --loglevel value, -l value  debug level, one of: info, debug (default: "info") [%LOGLEVEL%]
   --help, -h                  show help (default: false)
```

```
NAME:
   server.exe run - runs as a server

USAGE:
   server.exe run [command options] [arguments...]

OPTIONS:
   --clusterEndpoint value, -e value     cluster endpoint [http://localhost:19080] [%CLUSTER_ENDPOINT%]
   --clientCertificate value             path or content for the client certificate [%CLIENT_CERT%]
   --clientCertificatePK value           path or content for the client certificate private key [%CLIENT_CERT_PK%]
   --certStoreSearchKey value, -k value  keyword to look for searching the cluster certificate (windows cert store) [%CLUSTER_CERT_SEARCH_KEY%]
   --httpport value, -p value            port for the HTTP rest endpoint (server will be disabled if not provided) (default: 0) [%HTTP_PORT%]
   --insecureTLS, -i                     allow skip checking server CA/hostname (default: false) [%INSECURE_TLS%]
   --publishFilePath value, -f value     filename to write to, empty won't write anywhere [%PUBLISH_FILE_PATH%]
   --help, -h                            show help (default: false)
```
## Supported routes

## ws://{hostname:port}/api/traefik:

This route exposes a stream of Traefik 2.x compatible yaml data that can be fed directly into the Traefik *file* provider. The returned data maps routing rules for service instances running on the cluster, taking into account the Health and Status of each of the services in order to ensure requests are only routed to healthy service instances.


## Running Locally

* Deploy `TraefikProxyApp` to the local cluster
* Deploy the pinger test application mentioned in [Sample-Test-Application](#sample-test-application). Using a browser, access `https://localhost/pinger0/PingerService`. If all works, you should get a `200 OK` response with contents resembling the following:

   ```json
   {
     "Pinger: I'm alive on ... "
   }
   ```



## License

This software is released under the MIT License

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Trademarks

This project may contain trademarks or logos for projects, products, or services. Authorized use of Microsoft 
trademarks or logos is subject to and must follow 
[Microsoft's Trademark & Brand Guidelines](https://www.microsoft.com/en-us/legal/intellectualproperty/trademarks/usage/general).
Use of Microsoft trademarks or logos in modified versions of this project must not cause confusion or imply Microsoft sponsorship.
Any use of third-party trademarks or logos are subject to those third-party's policies.
