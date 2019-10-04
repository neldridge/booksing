#!/bin/bash

function log {
    echo "> $(date +%T) $*"
}

trap "exit" INT TERM
trap "removeContainers; kill 0" EXIT

function removeContainers {
    log "removing old containers"
    docker-compose rm -s -f -v
}

function startContainers {
    log "building and starting containers"
    docker-compose up --build --force-recreate
}

removeContainers


startContainers
