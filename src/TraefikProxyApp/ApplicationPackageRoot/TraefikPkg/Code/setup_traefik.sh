#!/bin/bash

set -e

LOG_OUTPUT=/tmp/setup_traefik.out
# 
# setup rules folder
# Note: The Traefik image assumes the rules will be created under the subdirectory traefik/rules of the mounted folder. 
#
TRAEFIK_RULES_DIR=traefik/rules
if [ ! -d "$TRAEFIK_RULES_DIR" ]; then
    echo "$TRAEFIK_RULES_DIR does not exist. Creating directories..." >> $LOG_OUTPUT
    mkdir -p $TRAEFIK_RULES_DIR >> $LOG_OUTPUT 2>&1
else
    echo "$TRAEFIK_RULES_DIR exists." >> $LOG_OUTPUT
fi

#
# copy certificates in the work directory where the container can access it
# Note: The Traefik image assumes the certs will be created under the subdirectory traefik/certs of the mounted folder. 
#
CONTAINER_TRAEFIK_CERT_DIR=traefik/certs
if [ ! -z "$TRAEFIK_CERT_THUMBPRINT" ]; then
    if [ ! -d "$CONTAINER_TRAEFIK_CERT_DIR" ]; then
        echo "$CONTAINER_TRAEFIK_CERT_DIR does not exist. Creating directories..." >> $LOG_OUTPUT
        mkdir -p $CONTAINER_TRAEFIK_CERT_DIR >> $LOG_OUTPUT 2>&1
    else
        echo "$CONTAINER_TRAEFIK_CERT_DIR exists." >> $LOG_OUTPUT
    fi

    cp -f ${TRAEFIK_CERT_DIR}/${TRAEFIK_CERT_THUMBPRINT}.crt ${CONTAINER_TRAEFIK_CERT_DIR}/traefik.crt >> $LOG_OUTPUT 2>&1
    cp -f ${TRAEFIK_CERT_DIR}/${TRAEFIK_CERT_THUMBPRINT}.prv ${CONTAINER_TRAEFIK_CERT_DIR}/traefik.prv >> $LOG_OUTPUT 2>&1
fi
