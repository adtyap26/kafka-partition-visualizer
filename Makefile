# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOTEST=$(GOCMD) test
GOLIST=$(GOCMD) list
GOFMT=gofmt -w

# Target binary name (change if needed)
BINARY_NAME=kafka-viz
# Output directory for builds
OUTPUT_DIR=bin

# Default target executed when you just run `make`
all: build

# Build the binary for the current OS/ARCH
build: fmt
	@echo "Building $(BINARY_NAME) for $(shell go env GOOS)/$(shell go env GOARCH)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"

# Build for specific platforms
build-linux: fmt
	@echo "Building $(BINARY_NAME) for linux/amd64..."
	@mkdir -p $(OUTPUT_DIR)/linux_amd64
	@GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/linux_amd64/$(BINARY_NAME) .
	@echo "Build complete: $(OUTPUT_DIR)/linux_amd64/$(BINARY_NAME)"

build-macos-amd64: fmt
	@echo "Building $(BINARY_NAME) for darwin/amd64..."
	@mkdir -p $(OUTPUT_DIR)/darwin_amd64
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/darwin_amd64/$(BINARY_NAME) .
	@echo "Build complete: $(OUTPUT_DIR)/darwin_amd64/$(BINARY_NAME)"

build-macos-arm64: fmt
	@echo "Building $(BINARY_NAME) for darwin/arm64 (Apple Silicon)..."
	@mkdir -p $(OUTPUT_DIR)/darwin_arm64
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/darwin_arm64/$(BINARY_NAME) .
	@echo "Build complete: $(OUTPUT_DIR)/darwin_arm64/$(BINARY_NAME)"

build-windows: fmt
	@echo "Building $(BINARY_NAME) for windows/amd64..."
	@mkdir -p $(OUTPUT_DIR)/windows_amd64
	@$(GOBUILD) -o $(OUTPUT_DIR)/windows_amd64/$(BINARY_NAME).exe .
	@echo "Build complete: $(OUTPUT_DIR)/windows_amd64/$(BINARY_NAME).exe"

# Run the application
run:
	$(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME) .
	./$(OUTPUT_DIR)/$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(OUTPUT_DIR)
	@echo "Clean complete."

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) $$(go list -f '{{.Dir}}' ./...)
	@echo "Formatting complete."

# Get dependencies
deps:
	$(GOGET) ./...

# Help message
help:
	@echo "Available commands:"
	@echo "  make build            - Build the binary for the current OS/ARCH"
	@echo "  make build-linux      - Build for Linux (amd64)"
	@echo "  make build-macos-amd64- Build for macOS (amd64)"
	@echo "  make build-macos-arm64- Build for macOS (arm64)"
	@echo "  make build-windows    - Build for Windows (amd64)"
	@echo "  make run              - Build and run the application"
	@echo "  make clean            - Remove build artifacts"
	@echo "  make fmt              - Format Go source code"
	@echo "  make deps             - Install dependencies"
	@echo "  make help             - Show this help message"

.PHONY: all build build-linux build-macos-amd64 build-macos-arm64 build-windows run clean fmt deps help


