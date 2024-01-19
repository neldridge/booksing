FROM golang:1.21.6 as builder
WORKDIR /app

COPY . .

RUN go mod tidy && GOOS=linux GOARCH=arm64 go build -tags 'fts5' -o booksing ./cmd/ui

FROM neldridge/upx:latest as upx
COPY --from=builder /app/booksing /app/booksing
RUN upx --ultra-brute /app/booksing

FROM busybox:latest
ENV BOOKSING_BOOKDIR /books
ENV BOOKSING_DATABASEDIR /db
ENV BOOKSING_FAILDIR /failed
ENV BOOKSING_IMPORTDIR /import
ENV BOOKSING_TIMEZONE UTC

WORKDIR /
COPY ./docker-entrypoint.sh /docker-entrypoint.sh
COPY --from=builder /app/booksing /booksing

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/booksing"]
