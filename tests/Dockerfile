# Dockerfile to setup nri-mssql binary in a container

FROM golang:1.23.4-bookworm AS builder
ARG CGO_ENABLED=0
WORKDIR /go/src/github.com/newrelic/nri-mssql
COPY . .
RUN make clean compile

FROM alpine:latest
COPY --from=builder /go/src/github.com/newrelic/nri-mssql/bin /
CMD ["sleep", "1h"]