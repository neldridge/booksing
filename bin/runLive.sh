#!/bin/bash

workingdir=$(mktemp -d)
mkdir "${workingdir}/import"

function log {
    echo "> $(date +%T) $*"
}

function cleanup {
    log "Removing old workdir"
    rm -rf workingdir
    log "Done. ✅"
}

trap 'cleanup' EXIT

log "Creating temp workspace in ${workingdir}"
cp -a testdata/import/gutenberg/* $workingdir/import/

export BOOKSING_LOGLEVEL=debug
export BOOKSING_ADMINUSER='erwin@gnur.nl'
export BOOKSING_DATABASE="file://${workingdir}/booksing.db"
export BOOKSING_IMPORTDIR="${workingdir}/import"
export BOOKSING_BOOKDIR="${workingdir}/"


air
