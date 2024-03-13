#!/bin/bash
set -e

export filePath=Traefik
export appName=Traefik
export TRAEFIK_VERSION="2.11.0"

go version

if [[ ! -f ./Traefik/TraefikPkg/Fetcher.Code/certs/sf.key || ! -f ./Traefik/TraefikPkg/Fetcher.Code/certs/sf.crt ]] ; then
  echo "You need to extract Service Fabric cert and key for traefik. See deploy-sf.sh"
  exit 1
fi

# Download traefik
mkdir -p ./tmp
curl -L -o ./tmp/traefik_v${TRAEFIK_VERSION}_linux_amd64.tar.gz https://github.com/traefik/traefik/releases/download/v${TRAEFIK_VERSION}/traefik_v${TRAEFIK_VERSION}_linux_amd64.tar.gz
tar -xzvf ./tmp/traefik_v${TRAEFIK_VERSION}_linux_amd64.tar.gz -C ./tmp
mv ./tmp/traefik ./Traefik/TraefikPkg/Traefik.Code/traefik
rm -rf ./tmp

# Build binary for fetcher
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -C ../../src/serviceFabricDiscoveryService -o ../../examples/traefik-inside-sf/Traefik/TraefikPkg/Fetcher.Code/fetcher ./cmd

echo Uploading Application Files
sfctl application upload --path ${filePath} --show-progress

echo Provisioning Application Type
sfctl application provision --application-type-build-path ${filePath}

echo Creating Application
sfctl application create --app-name fabric:/${appName} --app-type ${appName}Type --app-version 1.0.0
