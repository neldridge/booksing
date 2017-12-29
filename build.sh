#!/bin/bash

if [[ "${1}" != "release" ]]; then
    go-bindata-assetfs static/...;
    go build -o booksing *.go
    exit 0
fi

for os in linux darwin windows; do
    for arch in 386 arm amd64; do
        echo "building $os $arch"
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o "~/dist/booksing-$os-$arch" -a -ldflags '-extldflags "-static"' .
    done
done
