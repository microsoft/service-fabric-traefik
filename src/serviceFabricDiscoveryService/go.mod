module github.com/microsoft/service-fabric-traefik/serviceFabricDiscoveryService

go 1.16

require (
	github.com/ghodss/yaml v1.0.0
	github.com/github/certstore v0.1.0
	github.com/gorilla/websocket v1.4.2
	github.com/jjcollinge/servicefabric v0.0.2-0.20180125130438-8eebe170fa1b
	github.com/labstack/echo/v4 v4.5.0
	github.com/onrik/logrus v0.9.0
	github.com/sirupsen/logrus v1.8.1
	github.com/traefik/genconf v0.0.0-20210122120711-a2bf09240729
	github.com/traefik/paerser v0.1.4
	github.com/urfave/cli/v2 v2.3.0
)

replace github.com/github/certstore => github.com/tg123/certstore v0.1.1-0.20210416194039-a3d5d6605185
