# Copyright © 2017 Zlatko Čalušić
#
# Use of this source code is governed by an MIT-style license that can be found in the LICENSE file.
#

DOCKER_IMAGE ?= restic/rest-server

.PHONY: default rest-server install uninstall docker_build docker_push clean

default: rest-server

rest-server:
	@go run build.go

install: rest-server
	sudo /usr/bin/install -m 755 rest-server /usr/local/bin/rest-server

uninstall:
	sudo rm -f /usr/local/bin/rest-server

docker_build:
	docker pull golang:1.9.1-alpine
	docker run --rm -it \
		-v $(CURDIR):/go/src/github.com/restic/rest-server \
		-w /go/src/github.com/restic/rest-server \
		golang:1.9.1-alpine \
		go run build.go
	docker pull alpine:3.6
	docker build -t $(DOCKER_IMAGE) .

docker_push:
	docker push $(DOCKER_IMAGE):latest

clean:
	rm -f rest-server
