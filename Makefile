# Pass IMAGE on the command line, e.g. make image IMAGE=youruser/kai
IMAGE     ?=
VERSION   ?= $(shell git describe --tags --always --dirty)
COMMIT    := $(shell git rev-parse --short HEAD)
DATE      := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PLATFORMS ?= linux/amd64,linux/arm64

.PHONY: build test lint image image-push

build:
	go build -o bin/kai ./cmd/kai

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run

# Build a local image for the current architecture.
image:
	@test -n "$(IMAGE)" || (echo "set IMAGE, e.g. make image IMAGE=youruser/kai"; exit 1)
	docker build \
	  --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg DATE=$(DATE) \
	  -t $(IMAGE):$(VERSION) .

# Build amd64 + arm64 and push to the registry.
image-push:
	@test -n "$(IMAGE)" || (echo "set IMAGE, e.g. make image-push IMAGE=youruser/kai"; exit 1)
	docker buildx build --platform $(PLATFORMS) \
	  --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg DATE=$(DATE) \
	  -t $(IMAGE):$(VERSION) -t $(IMAGE):latest --push .
