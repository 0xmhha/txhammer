.PHONY: build test lint fmt clean run help docker docker-build docker-run docker-clean

# Variables
BINARY_NAME=txhammer
BUILD_DIR=build
GO=go
GOFLAGS=-v
LDFLAGS=-s -w

# Docker variables
DOCKER_IMAGE=txhammer
DOCKER_TAG=latest
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
TARGETARCH=$(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd

## test: Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -shuffle=on -coverprofile=coverage.out ./...

## test-short: Run tests without race detection
test-short:
	@echo "Running tests (short)..."
	$(GO) test -v -shuffle=on ./...

## lint: Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	gofumpt -l -w .

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

## run: Run with default parameters (requires RPC_URL and PRIVATE_KEY env vars)
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME) \
		--url $${RPC_URL:-http://localhost:8545} \
		--private-key $${PRIVATE_KEY} \
		--transactions 100

## docker: Build Docker image
docker: docker-build

## docker-build: Build Docker image with version info
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG) for $(TARGETARCH)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) .

## docker-run: Run Docker container (requires RPC_URL and PRIVATE_KEY env vars)
docker-run:
	@echo "Running $(DOCKER_IMAGE) container..."
	docker run --rm -it \
		-v $(PWD)/reports:/app/reports \
		-e RPC_URL=$${RPC_URL:-http://host.docker.internal:8545} \
		$(DOCKER_IMAGE):$(DOCKER_TAG) \
		--url $${RPC_URL:-http://host.docker.internal:8545} \
		--private-key $${PRIVATE_KEY} \
		--transactions 100

## docker-clean: Remove Docker image
docker-clean:
	@echo "Removing Docker image..."
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true

## compose-up: Start docker-compose services (anvil node)
compose-up:
	docker-compose up -d anvil

## compose-down: Stop docker-compose services
compose-down:
	docker-compose down

## compose-test: Run txhammer in docker-compose with anvil
compose-test: docker-build compose-up
	@echo "Waiting for anvil to start..."
	@sleep 3
	docker-compose run --rm txhammer \
		--url http://anvil:8545 \
		--private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
		--transactions 50 \
		--sub-accounts 5

## help: Show this help
help:
	@echo "TxHammer - StableNet Stress Testing Tool"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
