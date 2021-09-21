package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	disco "github.com/dariopb/serviceFabricDiscoveryService/pkg/discovery"
	restapi "github.com/dariopb/serviceFabricDiscoveryService/pkg/restapi"
	version "github.com/dariopb/serviceFabricDiscoveryService/version"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func printVersion() {
	log.Info(fmt.Sprintf("DiscoveryService version: %v", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

// server
var port int
var loglevelstr string
var clusterEndpoint string
var clientCertificate string
var clientCertificatePK string
var certStoreSearchKey string
var insecuretls bool
var httpport int

func main() {

	app := &cli.App{
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "loglevel",
				Aliases:     []string{"l"},
				Value:       "info",
				Usage:       "debug level, one of: info, debug",
				EnvVars:     []string{"LOGLEVEL"},
				Destination: &loglevelstr,
			},
		},
		Commands: []*cli.Command{
			{
				Name: "run",
				//Aliases: []string{"server"},
				Usage:  "runs as a server",
				Action: server,

				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "clusterendpoint",
						Aliases:     []string{"e"},
						Value:       "",
						Usage:       "cluster endpoint [http://localhost:19080]",
						EnvVars:     []string{"CLUSTER_ENDPOINT"},
						Destination: &clusterEndpoint,
						Required:    true,
					},
					&cli.StringFlag{
						Name:        "clientCertificate",
						Value:       "",
						Usage:       "path or content for the client certificate",
						EnvVars:     []string{"CLIENT_CERT"},
						Destination: &clientCertificate,
						Required:    false,
					},
					&cli.StringFlag{
						Name:        "clientCertificatePK",
						Value:       "",
						Usage:       "path or content for the client certificate private key",
						EnvVars:     []string{"CLUSTER_CERT_SEARCH_KEY"},
						Destination: &clientCertificatePK,
						Required:    false,
					},
					&cli.StringFlag{
						Name:        "certStoreSearchKey",
						Aliases:     []string{"k"},
						Value:       "",
						Usage:       "keyword to look for searching the cluster certificate (windows cert store)",
						EnvVars:     []string{"CLUSTER_CERT_SEARCH_KEY"},
						Destination: &certStoreSearchKey,
						Required:    false,
					},
					&cli.IntFlag{
						Name:        "httpport",
						Aliases:     []string{"p"},
						Value:       0,
						Usage:       "port for the HTTP rest endpoint (server will be disabled if not provided)",
						EnvVars:     []string{"HTTP_PORT"},
						Destination: &httpport,
						Required:    true,
					},
					&cli.BoolFlag{
						Name:        "insecuretls",
						Aliases:     []string{"i"},
						Value:       false,
						Usage:       "allow skip checking server CA/hostname",
						EnvVars:     []string{"INSECURE_TLS"},
						Destination: &insecuretls,
						Required:    false,
					},
				},
			},
		},

		Name:  "discoveryService",
		Usage: "exposes service fabric application and service metadata over websockets",
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func server(ctx *cli.Context) error {
	printVersion()

	loglevel := log.InfoLevel
	if l, err := log.ParseLevel(loglevelstr); err == nil {
		loglevel = l
	}

	//log.AddHook(ProcessCounter)
	//log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetLevel(loglevel)
	log.SetOutput(os.Stdout)

	config := disco.CreateConfig()
	config.ClusterManagementURL = clusterEndpoint
	config.CertStoreSearchKey = certStoreSearchKey
	config.Certificate = clientCertificate
	config.CertificateKey = clientCertificatePK
	config.InsecureSkipVerify = insecuretls

	disco, err := disco.NewDiscoveryService(config, nil, httpport)
	if err != nil {
		log.Fatalf("failed to start new discovery service: ", err)
	}

	if httpport != 0 {
		restapi.NewRestApi(nil, httpport, "", disco)
	} else {
		log.Debug("httpport not provided, not starting HTTP server")
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c

	disco.Close()

	return err
}
