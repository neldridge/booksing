#!/bin/bash

function log {
    echo "> $(date +%T) $*"
}

log "Building binary"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags 'fts5' -o booksing ./cmd/ui


log "copying to sanny"
upx booksing
scp booksing sanny:/tmp/booksing

log "Sending restart trigger"
ssh sanny "sudo systemctl restart booksing"

log "Deployed app in ${SECONDS} seconds"
ssh sanny "sudo journalctl -u booksing -f"
