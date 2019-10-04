FROM node as jsbuilder
WORKDIR /workspace
COPY web/package.json /workspace/
COPY web/package-lock.json /workspace/
RUN npm install
COPY web/src /workspace/src
COPY web/babel.config.js /workspace/
COPY web/vue.config.js /workspace/
RUN npm run build


FROM golang:1.12 as builder
WORKDIR /go/src/github.com/gnur/booksing/
#RUN apt-get update && apt-get install -y git
RUN go get github.com/jteeuwen/go-bindata/...
RUN go get github.com/elazarl/go-bindata-assetfs/...
ENV GO111MODULE=on
COPY --from=jsbuilder /workspace/dist web/dist
RUN go-bindata-assetfs -prefix web web/dist/...
COPY epub epub
COPY go.mod go.mod
RUN go mod download
COPY *.go ./
RUN go build -ldflags "-linkmode external -extldflags -static" -o booksing *.go

FROM gnur/calibre:2018-07-10
COPY /testdata /books
COPY --from=builder /go/src/github.com/gnur/booksing/booksing /
COPY .idea/booksing-erwin-land-1e12c348495e.json /creds.json
CMD [ "/booksing" ]
