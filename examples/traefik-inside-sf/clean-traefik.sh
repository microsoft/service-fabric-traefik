#!/bin/bash
sfctl application delete --application-id Traefik
sfctl application unprovision --application-type-name TraefikType --application-type-version 1.0.0
sfctl store delete --content-path Traefik
