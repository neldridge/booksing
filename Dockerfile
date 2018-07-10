FROM golang:1.10 as builder
WORKDIR /go/src/github.com/gnur/booksing/
#RUN apt-get update && apt-get install -y git
RUN go get github.com/jteeuwen/go-bindata/...
RUN go get github.com/elazarl/go-bindata-assetfs/...
COPY static static
RUN go-bindata-assetfs static/...
COPY vendor vendor
COPY *.go ./
RUN go build -ldflags "-linkmode external -extldflags -static" -o booksing *.go

FROM debian
RUN apt-get update && apt-get install -y calibre python
COPY --from=builder /go/src/github.com/gnur/booksing/booksing /
COPY testdata /books
CMD [ "/booksing" ]
