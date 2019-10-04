#!/bin/bash

function log {
    echo "> $(date +%T) $*"
}

trap 'kill $(jobs -p)' EXIT

log "building binary"
go build -o booksing .

workingdir=$(mktemp -d)
log "Creating temp workspace in ${workingdir}"
cp -a testdata $workingdir

export BOOKSING_LOGLEVEL=debug
export BOOKSING_DATABASE="firestore://booksing-erwin-land"
export GOOGLE_APPLICATION_CREDENTIALS="booksing-creds.json"
export BOOKSING_IMPORTDIR="${workingdir}/testdata/import"
export BOOKSING_BOOKDIR="${workingdir}/testdata/"


log "running booksing"
./booksing &

log "starting live yarn"
cd web
yarn serve


wait
log "cleaning up workspace"
rm -rf ${workdingdir}
