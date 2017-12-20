#!/bin/bash

dep ensure
go-bindata-assetfs static/...;
go build -o booksing *.go
