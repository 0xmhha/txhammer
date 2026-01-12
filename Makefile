.PHONY: build test lint fmt clean run help

# Variables
BINARY_NAME=txhammer
BUILD_DIR=build
GO=go
GOFLAGS=-v
LDFLAGS=-s -w

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

## help: Show this help
help:
	@echo "TxHammer - StableNet Stress Testing Tool"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
