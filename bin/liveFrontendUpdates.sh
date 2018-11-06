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
    docker-compose up -d --build --force-recreate
}

function refreshBooks {
    log "sleeping"
    sleep 5
    log "refreshing books"
    curl localhost:7132/refresh
}

removeContainers

startContainers

refreshBooks &

cd web
npm run serve
