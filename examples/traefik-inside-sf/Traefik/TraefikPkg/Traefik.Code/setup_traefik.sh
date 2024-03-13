#!/bin/bash

echo "Starting" > log.txt
set -e

#echo "Traefik config file:"
#more ../traefik.yaml >> log.txt

mkdir -p ../dynamic-config/

cp traefik-dynamic.yaml ../dynamic-config/traefik-dynamic.yaml
