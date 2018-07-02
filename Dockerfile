FROM golang:1.10-alpine as builder
WORKDIR /go/src/github.com/gnur/booksing/
RUN apk add --no-cache git
RUN go get github.com/jteeuwen/go-bindata/...
RUN go get github.com/elazarl/go-bindata-assetfs/...
COPY static static
RUN go-bindata-assetfs static/...
COPY vendor vendor
COPY *.go ./
RUN go build -o app *.go

FROM alpine
WORKDIR /app
COPY --from=builder /go/src/github.com/gnur/booksing/app .
CMD [ "./app" ]
