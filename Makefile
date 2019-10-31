# Copyright © 2017 Zlatko Čalušić
#
# Use of this source code is governed by an MIT-style license that can be found in the LICENSE file.
#

DOCKER_IMAGE ?= restic/rest-server

REST_SERVER_VERSION := $(strip $(shell cat VERSION))

.PHONY: default rest-server install uninstall docker_build docker_push clean

default: rest-server

rest-server:
	@go run build.go

docker_build:
	docker pull golang:alpine
	docker run --rm -it \
		-v $(CURDIR):/go/src/github.com/restic/rest-server \
		-w /go/src/github.com/restic/rest-server \
		golang:alpine \
		go run build.go
	docker pull alpine
	docker build -t $(DOCKER_IMAGE):$(REST_SERVER_VERSION) .
	docker tag $(DOCKER_IMAGE):$(REST_SERVER_VERSION) $(DOCKER_IMAGE):latest

docker_push:
	docker push $(DOCKER_IMAGE):$(REST_SERVER_VERSION)
	docker push $(DOCKER_IMAGE):latest

clean:
	rm -f rest-server
