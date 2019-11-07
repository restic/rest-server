PROJECT_NAME := rest-server
DOCKER_IMAGE ?= restic/$(PROJECT_NAME)
MAIN := ./cmd/$(PROJECT_NAME)/main.go
VERSION := $(shell git describe --tags --abbrev=0)

install:
	go install

build: install
	go build -o $(PROJECT_NAME) $(MAIN)

docker_build: install
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker_push: docker_build
	docker push $(DOCKER_IMAGE):$(REST_SERVER_VERSION)
	docker push $(DOCKER_IMAGE):latest
