FROM golang:1.20.2-alpine3.17 AS build
WORKDIR /
COPY . ./
ENV DB_PATH=/shop.db
RUN apk add --update gcc musl-dev sqlite
RUN cat /script.sql | sqlite3 $DB_PATH
RUN go mod download
RUN go build -tags "linux" -o /client

FROM alpine:3.17
WORKDIR /
ENV DB_PATH=/shop.db
COPY --from=build /client /client
COPY --from=build /shop.db /shop.db
ENTRYPOINT ["/client"]