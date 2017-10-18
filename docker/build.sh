#!/bin/sh

set -e

echo "Build binary using golang docker image"
docker run --rm -ti \
    -v `pwd`:/go/src/github.com/restic/rest-server \
    -w /go/src/github.com/restic/rest-server golang:1.9.1-alpine go run build.go

echo "Build docker image restic/rest-server:latest"
docker build --rm -t restic/rest-server:latest -f docker/Dockerfile .
