module github.com/microsoft/service-fabric-traefik/serviceFabricDiscoveryService

go 1.20

require (
	github.com/ghodss/yaml v1.0.0
	github.com/github/certstore v0.2.0
	github.com/gorilla/websocket v1.5.1
	github.com/jjcollinge/servicefabric v0.0.2-0.20231030111952-b30eba315e44
	github.com/labstack/echo/v4 v4.11.4
	github.com/onrik/logrus v0.11.0
	github.com/sirupsen/logrus v1.9.3
	github.com/traefik/genconf v0.5.2
	github.com/traefik/paerser v0.2.0
	github.com/urfave/cli/v2 v2.27.1
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/github/certstore => github.com/tg123/certstore v0.1.1-0.20210416194039-a3d5d6605185
