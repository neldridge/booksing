#!/bin/bash

function log {
    echo "> $(date +%T) $*"
}

log "deleting database"
if [ -f testdata/booksing.db ]; then
    rm -rf testdata/booksing.db
fi

log "building binary"
go build -o booksing .
cd testdata
export BOOKSING_LOGLEVEL=debug

log "running booksing"
../booksing
