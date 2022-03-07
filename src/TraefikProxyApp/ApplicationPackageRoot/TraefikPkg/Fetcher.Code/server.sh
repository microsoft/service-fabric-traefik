#!/bin/bash

echo "Starting fetcher... " > log.txt
./server $@ > log.txt 2>&1

