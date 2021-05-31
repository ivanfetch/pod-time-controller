DOCKER_NAME=pod-time-controller
DOCKER_REGISTRY=ivanfetch
VERSION= $(shell git describe --tags)
GIT_COMMIT=$(shell git rev-parse HEAD)
LDFLAGS="-s -w -X podTimeController.Version=$(VERSION) -X podTimeController.GitCommit=$(GIT_COMMIT)"

all: build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:fmt vet
	go test

.PHONY: build
build:test
	go build -ldflags $(LDFLAGS) -o $(DOCKER_NAME) cmd/main.go

# While we don't need a local build to succeed as a prerequisite,
# its build-time catches build errors faster than a build in Docker.
.PHONY: docker-build
docker-build: build
	docker build -t $(DOCKER_NAME) .

# This requires the container runs in a Docker net that can reach Kube.
# .PHONY: docker-run
docker-run:
	docker run -it --rm $(DOCKER_NAME)

.PHONY: docker-push
docker-push:
	docker tag $(DOCKER_NAME) $(DOCKER_REGISTRY)/$(DOCKER_NAME):$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_NAME):$(VERSION)

