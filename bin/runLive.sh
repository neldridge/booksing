#!/bin/bash

ulimit -S -n 1024

workingdir=$(mktemp -d)
mkdir "${workingdir}/import"

function log {
    echo "> $(date +%T) $*"
}

function cleanup {
  ##  log "Stopping meili container"
  ##  sudo docker stop meili
    log "Removing old workdir"
    rm -rf workingdir
    log "Done. ✅"
}

trap 'cleanup' EXIT

## log "Starting meilisearch in docker"
## sudo docker run --name meili -d  --rm -p 7701:7700 getmeili/meilisearch:latest ./meilisearch --master-key=masterKey
## 
## log "Waiting for meili availability"
## until curl --max-time 0.3 --output /dev/null --silent --head http://localhost:7701; do
##     printf '.'
##     sleep 0.1
## done

log "Creating temp workspace in ${workingdir}"
cp -a testdata/import $workingdir/import/

source .env

export BOOKSING_LOGLEVEL=debug
export BOOKSING_ADMINUSER='erwin@gnur.nl'
export BOOKSING_DATABASEDIR="${workingdir}/db"
export BOOKSING_IMPORTDIR="${workingdir}/import"
export BOOKSING_FAILDIR="${workingdir}/failed"
export BOOKSING_BOOKDIR="${workingdir}/"
export BOOKSING_SAVEINTERVAL="20s"
export BOOKSING_SECURE="false"
export GOOGLE_KEY="$GKEY"
export GOOGLE_SECRET="$GSECRET"
export BOOKSING_SECRET="vJbh7i6tMWNN7BNYQ"
export BOOKSING_MQTTENABLED=true
export BOOKSING_MQTTHOST="tcp://sanny.egdk.nl:1883"
export BOOKSING_MQTTTOPIC="events"
export BOOKSING_MQTTCLIENTID="booksing"
export BOOKSING_BINDADDRESS=":7133"


air
