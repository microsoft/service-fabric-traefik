package restapi

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	disco "github.com/microsoft/service-fabric-traefik/serviceFabricDiscoveryService/pkg/discovery"
	echolog "github.com/onrik/logrus/echo"
	log "github.com/sirupsen/logrus"
)

type RestAPI struct {
	echo  *echo.Echo
	disco *disco.DiscoveryService

	pingWait, pongWait, writeWait time.Duration
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func NewRestApi(cert *tls.Certificate, port int, token string, disco *disco.DiscoveryService) (RestAPI, error) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	api := RestAPI{
		echo:      e,
		disco:     disco,
		pingWait:  30 * time.Second,
		pongWait:  40 * time.Second,
		writeWait: 10 * time.Second,
	}

	// Routes
	e.GET("/help", help)
	e.GET("/api/traefik", api.getTraefikRoutes)

	e.Logger = echolog.NewLogger(log.StandardLogger(), "")
	//e.Use(middleware.Logger())

	log.Infof("Starting REST API on port %d", port)

	go func() {
		//err := e.StartTLS(fmt.Sprintf(":%d", port), cert.Certificate, xxxx)
		err := e.Start(fmt.Sprintf(":%d", port))
		if err != nil {
			log.Errorf("Start echo failed with [%s]", err.Error())
			panic(err.Error())
		}
	}()

	return api, nil
}

func (r RestAPI) Close() {
	if r.echo != nil {
		r.echo.Close()
		r.echo = nil
	}
}

func help(c echo.Context) error {
	hostname, _ := os.Hostname()
	banner := fmt.Sprintf("serviceFabricDiscoveryService alive on [%s] (%s on %s/%s). Available routes: ",
		hostname, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	routes := c.Echo().Routes()
	for _, route := range routes {
		banner = banner + "<li>" + route.Path + "</li> "
	}

	return c.HTML(http.StatusOK, banner)
}

func (api *RestAPI) getTraefikRoutes(c echo.Context) error {
	r := c.Request()
	l := log.WithField("remoteaddr", r.RemoteAddr)
	l.Infof("Websocket connection request")

	conn, err := upgrader.Upgrade(c.Response().Writer, r, nil)
	if err != nil {
		l.WithError(err).Error("Unable to upgrade connection")
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Faied handling WS"))
	}

	connError := make(chan error)
	discoveryData := api.disco.Subscribe(r.RemoteAddr)

	// Setup pong timeouts and read loop
	go api.readLoop(conn, connError, l)
	ticker := time.NewTicker(api.pingWait)
	defer ticker.Stop()

loop:
	for {
		select {
		case data := <-discoveryData:
			conn.SetWriteDeadline(time.Now().Add(api.writeWait))
			if err = conn.WriteMessage(websocket.TextMessage, data); err != nil {
				break loop
			}

		case <-ticker.C:
			l.Debugf("sending PING")
			conn.SetWriteDeadline(time.Now().Add(api.writeWait))
			if err = conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				break loop
			}
		case err = <-connError:
			break loop
		}
	}

	l.Infof("Websocket connection finishing [err: %v]", err)
	api.disco.Unsubscribe(r.RemoteAddr)
	conn.Close()

	return nil
}

func (api *RestAPI) readLoop(conn *websocket.Conn, connError chan error, l *log.Entry) {
	defer func() {
		conn.Close()
	}()

	//conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(api.pongWait))
	conn.SetPongHandler(func(string) error {
		l.Debugf("PONG received")
		conn.SetReadDeadline(time.Now().Add(api.pongWait))
		return nil
	})

	var err error
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		msg = msg
	}

	connError <- err
}
