#!/bin/bash

echo "Starting fetcher: "

../TraefikPkg.Fetcher.Code.0.1.0-beta/server $@ > log.txt 2>&1

