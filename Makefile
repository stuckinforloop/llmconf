# llmconf Makefile

# Variables
BINARY_NAME := llmconf
MODULE_PATH := github.com/stuckinforloop/llmconf
BUILD_DIR := ./bin
CMD_PATH := ./cmd/llmconf

# Go settings
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Build flags
LDFLAGS := -ldflags "-s -w -X '$(MODULE_PATH)/internal/version.Version=$(shell git describe --tags --always 2>/dev/null || echo dev)' -X '$(MODULE_PATH)/internal/version.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'"

# Default target
.DEFAULT_GOAL := all

.PHONY: all build clean test test-verbose test-coverage test-snapshots lint fmt vet install uninstall deps tidy help

## all: Build the binary
all: build

## build: Build the binary for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

## build-all: Build for all platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	@echo "All builds complete"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned"

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## test-short: Run short tests (skip integration)
test-short:
	@echo "Running short tests..."
	$(GOTEST) -v -short ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-snapshots: Update snapshot tests
test-snapshots:
	@echo "Updating snapshots..."
	UPDATE_SNAPS=true $(GOTEST) -v ./...

## lint: Run linter
lint:
	@echo "Running linter..."
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w -s .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## tidy: Tidy go modules
tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

## install: Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

## uninstall: Remove binary from GOPATH/bin
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Uninstalled"

## run: Build and run (for development)
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

## dev: Run with hot reload (requires air)
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Install with: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi

## version: Show version info
version:
	@echo "Version: $(shell git describe --tags --always 2>/dev/null || echo dev)"
	@echo "Module: $(MODULE_PATH)"

## generate: Run go generate
generate:
	@echo "Running go generate..."
	$(GOCMD) generate ./...

## check: Run all checks (fmt, vet, test, lint)
check: fmt vet test lint
	@echo "All checks passed!"

## ci: CI pipeline (build + test)
ci: deps build test
	@echo "CI pipeline complete"

## help: Show this help
help:
	@echo "Available targets:"
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //g' | column -t -s ':'

# Release targets (for maintainers)
.PHONY: release-tag release-build release

## release-tag: Tag a new release (usage: make release-tag VERSION=v1.0.0)
release-tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION not set. Use: make release-tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Tagging $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Tagged $(VERSION)"
	@echo "Push with: git push origin $(VERSION)"

## release-build: Build release binaries
release-build: clean build-all
	@echo "Creating release archives..."
	@mkdir -p $(BUILD_DIR)/release
	@cd $(BUILD_DIR) && for f in $(BINARY_NAME)-*; do \
		if [ -f "$$f" ]; then \
			tar czf release/$$f.tar.gz $$f; \
			echo "Created release/$$f.tar.gz"; \
		fi; \
	done
	@echo "Release archives created in $(BUILD_DIR)/release/"

## snapshot: Create snapshot build
snapshot: clean build test
	@echo "Creating snapshot build..."
	@mkdir -p $(BUILD_DIR)/snapshot
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/snapshot/$(BINARY_NAME)-$(shell git rev-parse --short HEAD)
	@echo "Snapshot: $(BUILD_DIR)/snapshot/$(BINARY_NAME)-$(shell git rev-parse --short HEAD)"
