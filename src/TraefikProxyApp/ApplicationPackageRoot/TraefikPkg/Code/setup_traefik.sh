#!/bin/bash

echo "Starting" > log.txt
set -e

#export TRAEFIK_HTTP_PORT=9999
#export TRAEFIK_ENABLE_DASHBOARD=yes
#export Fabric_Folder_App_Work=/home/dario/data

cp traefik-template.yaml ../traefik.yaml

vars=(TRAEFIK_HTTP_PORT TRAEFIK_ENABLE_DASHBOARD Fabric_Folder_App_Work)

for i in "${vars[@]}"; do sed -i 's#'\<$i\>'#'"${!i}"'#g' ../traefik.yaml; done

echo "Traefik config file:"
more ../traefik.yaml >> log.txt

cp dynConfig/dyn.yaml $Fabric_Folder_App_Work/dyn.yaml


#
# copy certificates in the work directory where the traefik can access it
#
#CONTAINER_TRAEFIK_CERT_DIR=traefik/certs
#if [ ! -z "$TRAEFIK_CERT_THUMBPRINT" ]; then
#    if [ ! -d "$CONTAINER_TRAEFIK_CERT_DIR" ]; then
#        echo "$CONTAINER_TRAEFIK_CERT_DIR does not exist. Creating directories..." >> $LOG_OUTPUT
#        mkdir -p $CONTAINER_TRAEFIK_CERT_DIR >> $LOG_OUTPUT 2>&1
#    else
#        echo "$CONTAINER_TRAEFIK_CERT_DIR exists." >> $LOG_OUTPUT
#    fi
#
#    cp -f ${TRAEFIK_CERT_DIR}/${TRAEFIK_CERT_THUMBPRINT}.crt ${CONTAINER_TRAEFIK_CERT_DIR}/traefik.crt >> $LOG_OUTPUT 2>&1
#    cp -f ${TRAEFIK_CERT_DIR}/${TRAEFIK_CERT_THUMBPRINT}.prv ${CONTAINER_TRAEFIK_CERT_DIR}/traefik.prv >> $LOG_OUTPUT 2>&1
#fi
