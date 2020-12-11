#!/bin/bash

function log {
    echo "> $(date +%T) $*"
}

log "binpacking static assets"
pkger

log "Building binary"
GOOS=linux GOARCH=amd64 go build -o booksing ./cmd/ui


log "copying to sanny"
upx booksing
mv booksing /tmp/booksing

log "Sending restart trigger"
curl localhost:7132/kill

log "Deployed app in ${SECONDS} seconds"
