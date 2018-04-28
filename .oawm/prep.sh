#!/bin/bash

export PROJECT=booksing

mkdir -p /go/src/github.com/gnur/
ln -s /src /go/src/github.com/gnur/${PROJECT}
cd /go/src/github.com/gnur/${PROJECT}/

apk add --no-cache git
go get github.com/jteeuwen/go-bindata/...
go get github.com/elazarl/go-bindata-assetfs/...

go-bindata-assetfs static/...
