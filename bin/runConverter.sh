#!/bin/bash

function log {
    echo "> $(date +%T) $*"
}

if [[ -f ~/.global-env ]]; then
    source ~/.global-env
fi


log "building binary"
go build -o converter ./cmd/converter/ || exit 1

export CONVERTER_TOPICNAME="convert-book"
export GOOGLE_APPLICATION_CREDENTIALS="booksing-converter-creds.json"
export CONVERTER_SUBSCRIPTIONNAME="converter"
export CONVERTER_GOOGLEPROJECT="booksing"
export CONVERTER_BOOKSINGHOST="$BOOKSING_HOST"
export CONVERTER_BOOKSINGAPIKEY="$BOOKSING_API_KEY"


./converter
