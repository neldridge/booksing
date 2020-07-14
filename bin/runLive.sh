#!/bin/bash

ulimit -S -n 1024

workingdir=$(mktemp -d)
mkdir "${workingdir}/import"

function log {
    echo "> $(date +%T) $*"
}

function cleanup {
    log "Stopping meili container"
    docker stop meili
    log "Removing old workdir"
    rm -rf workingdir
    log "Done. ✅"
}

trap 'cleanup' EXIT

log "Starting meilisearch in docker"
docker run --name meili -d  --rm -p 7700:7700 getmeili/meilisearch:latest ./meilisearch --master-key=masterKey

log "Waiting for meili availability"
until curl --max-time 0.3 --output /dev/null --silent --head http://localhost:7700; do
    printf '.'
    sleep 0.1
done

log "Creating temp workspace in ${workingdir}"
cp -a testdata/import/gutenberg/* $workingdir/import/

source .env

export BOOKSING_LOGLEVEL=debug
export BOOKSING_ADMINUSER='erwin@gnur.nl'
export BOOKSING_DATABASE="file://${workingdir}/booksing.db"
export BOOKSING_IMPORTDIR="${workingdir}/import"
export BOOKSING_FAILDIR="${workingdir}/failed"
export BOOKSING_BOOKDIR="${workingdir}/"
export BOOKSING_MEILI_HOST="http://localhost:7700"
export BOOKSING_MEILI_INDEX="books"
export BOOKSING_MEILI_KEY="masterKey"
export BOOKSING_SAVEINTERVAL="120s"
export BOOKSING_SECURE="false"
export GOOGLE_KEY="$GKEY"
export GOOGLE_SECRET="$GSECRET"
export SESSION_SECRET="vJbh7i6tMWNN7BNYQ"


air
