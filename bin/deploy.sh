#!/bin/bash

function log {
    echo "> $(date +%T) $*"
}

log "Generating static assets"

log "binpacking static assets"
pkger

log "Building binary"
GOOS=linux GOARCH=amd64 go build -o booksing ./cmd/ui


log "copying to sanny"
scp booksing pit:/tmp/
mv booksing /tmp/booksing

log "Sending restart trigger"
curl https://booksing.erwin.land/kill

log "Deployed app in ${SECONDS} seconds"
