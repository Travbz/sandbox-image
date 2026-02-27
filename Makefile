.PHONY: build test lint vet fmt clean image help

BINARY := entrypoint
BUILD_DIR := build
IMAGE_NAME := sandbox-image
IMAGE_TAG := latest

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'

## build: Build the entrypoint binary
build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY) ./entrypoint/

## test: Run all tests
test:
	go test ./... -v

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## vet: Run go vet
vet:
	go vet ./...

## fmt: Format code with goimports
fmt:
	goimports -w .

## image: Build the Docker image (multi-arch)
image:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE_NAME):$(IMAGE_TAG) .

## image-local: Build the Docker image for the local platform only
image-local:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
