# Makefile for go-mcp-file-context-server

APP_NAME := go-mcp-file-context-server
OUTPUT_DIR := bin
GO := go
LDFLAGS := -ldflags="-s -w"

# Detect OS for universal binary creation
UNAME_S := $(shell uname -s 2>/dev/null || echo Windows)

.PHONY: all clean build test darwin-universal darwin linux windows checksums help

# Default target
all: clean build checksums

help:
	@echo "Available targets:"
	@echo "  all             - Clean, build all platforms, generate checksums"
	@echo "  build           - Build for all platforms"
	@echo "  darwin          - Build for macOS (arm64 and amd64)"
	@echo "  darwin-universal- Build macOS universal binary (requires macOS)"
	@echo "  linux           - Build for Linux (amd64 and arm64)"
	@echo "  windows         - Build for Windows (amd64)"
	@echo "  test            - Run tests"
	@echo "  clean           - Remove build artifacts"
	@echo "  checksums       - Generate SHA256 checksums"

clean:
	@echo "Cleaning..."
	@rm -rf $(OUTPUT_DIR)
	@mkdir -p $(OUTPUT_DIR)

build: darwin linux windows
ifeq ($(UNAME_S),Darwin)
	@$(MAKE) darwin-universal
endif

# macOS builds
darwin: $(OUTPUT_DIR)/$(APP_NAME)-darwin-arm64 $(OUTPUT_DIR)/$(APP_NAME)-darwin-amd64

$(OUTPUT_DIR)/$(APP_NAME)-darwin-arm64:
	@echo "Building for darwin/arm64..."
	@GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $@ .

$(OUTPUT_DIR)/$(APP_NAME)-darwin-amd64:
	@echo "Building for darwin/amd64..."
	@GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $@ .

darwin-universal: darwin
ifeq ($(UNAME_S),Darwin)
	@echo "Creating macOS universal binary..."
	@lipo -create -output $(OUTPUT_DIR)/$(APP_NAME)-darwin-universal \
		$(OUTPUT_DIR)/$(APP_NAME)-darwin-arm64 \
		$(OUTPUT_DIR)/$(APP_NAME)-darwin-amd64
else
	@echo "Skipping universal binary (requires macOS)"
endif

# Linux builds
linux: $(OUTPUT_DIR)/$(APP_NAME)-linux-amd64 $(OUTPUT_DIR)/$(APP_NAME)-linux-arm64

$(OUTPUT_DIR)/$(APP_NAME)-linux-amd64:
	@echo "Building for linux/amd64..."
	@GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $@ .

$(OUTPUT_DIR)/$(APP_NAME)-linux-arm64:
	@echo "Building for linux/arm64..."
	@GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $@ .

# Windows builds
windows: $(OUTPUT_DIR)/$(APP_NAME)-windows-amd64.exe

$(OUTPUT_DIR)/$(APP_NAME)-windows-amd64.exe:
	@echo "Building for windows/amd64..."
	@GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $@ .

# Run tests
test:
	@echo "Running tests..."
	@$(GO) test -v ./...

# Generate checksums
checksums:
	@echo "Generating checksums..."
	@cd $(OUTPUT_DIR) && \
	if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum * > checksums.txt; \
	elif command -v shasum >/dev/null 2>&1; then \
		shasum -a 256 * > checksums.txt; \
	else \
		echo "sha256sum/shasum not available"; \
	fi

# Development helpers
run:
	@$(GO) run . -log-level debug

fmt:
	@$(GO) fmt ./...

vet:
	@$(GO) vet ./...

lint: fmt vet
