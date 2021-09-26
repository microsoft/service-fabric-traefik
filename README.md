# serviceFabricDiscoveryService
serviceFabricDiscoveryService is a service that connects to a Service Fabric cluster and exposes discovery data and changes [async] over websockets or, locally, via a file. Changes on names (applications/services) and endpoint mapping information is sent as messages over the websocket as they happen, the client doesn't have to poll the server.

The service exposes several websocket routes that have specific functionality.

## Running the server
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
# Supported routes

## ws://{hostname:port}/api/traefik:

This route exposes a stream of Traefik 2.x compatible yaml data that can be fed directly into the Traefik *file* provider. The returned data maps routing rules for service instances running on the cluster, taking into account the Health and Status of each of the services in order to ensure requests are only routed to healthy service instances.

## Example configuration

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
          <Label Key="traefik.http.defaultEP.loadbalancer.passhostheader">true</Label>
          <Label Key="traefik.http.defaultEP.loadbalancer.healthcheck.path">/</Label>
          <Label Key="traefik.http.defaultEP.loadbalancer.healthcheck.interval">10s</Label>
          <Label Key="traefik.http.defaultEP.loadbalancer.healthcheck.scheme">http</Label>
        </Labels>
        </Extension>
      </Extensions>
    </StatelessServiceType>
  </ServiceTypes>
  ...
```

---

The only required label to expose a service via the reverse proxy is the **traefik.http.[endpointName]** one. Setting only this label will expose the service on a well known path and handle the basic scenarios.

```
http(s)://<Cluster FQDN | internal IP>:Port/ApplicationInstanceName/ServiceInstanceName/{PartitionGuid}/<Suffix path>
```

If you need to change the routes or add middleware then you can add different labels configuring them.


## Supported Labels (since 0.2.x)

*Http enable section*

* **traefik.http.[endpointName]**    Enables exposing an http service via the reverse proxy.

Rule section

* **traefik.http.[endpointName].rule**    Traefik rule to apply [PathPrefix(`/dario`))]. This rule is added on top of the default path generation. If this is set, you **have** to define a middleware to remove the prefix for the service to receive the stripped path.

*Loadbalancer section*

* **traefik.http.[endpointName].loadbalancer.passhostheader**          passhostheaders ['true'/'false']
* **traefik.http.[endpointName].loadbalancer.healthcheck.path**        Healthcheck endpoint path ['/healtz']
* **traefik.http.[endpointName].loadbalancer.healthcheck.interval**    Healthcheck interval ['10s']
* **traefik.http.[endpointName].loadbalancer.healthcheck.scheme**      Healthcheck scheme ['http']

*Middleware section*

* **traefik.http.[endpointName].middleware.stripprefix**    prefix to strip ['/dario']

## License

This software is released under the MIT License
