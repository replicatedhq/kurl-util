FROM golang:1.13-buster

WORKDIR /go/src/github.com/replicatedhq/kurl
COPY Makefile Makefile
COPY LICENSE LICENSE
COPY go.mod go.mod
COPY go.sum go.sum
COPY cmd cmd

RUN make build

FROM debian:buster-slim

COPY --from=0 /go/src/github.com/replicatedhq/kurl/bin/* /usr/local/bin/
COPY cmd/proxy/assets /assets

ARG commit=unknown
ENV COMMIT=$commit
