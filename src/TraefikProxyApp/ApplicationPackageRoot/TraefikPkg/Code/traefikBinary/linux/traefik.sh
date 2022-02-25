#!/bin/bash

echo "Starting traefik... " > log1.txt
./traefik $@ >> log1.txt 2>&1
