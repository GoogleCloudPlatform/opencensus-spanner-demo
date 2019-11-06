FROM golang:1.13.1-buster

ENV GO111MODULE=on

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

ENTRYPOINT ["/app/metrics-load-generator"]