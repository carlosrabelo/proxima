# Makefile for Proxima project
# Usage: make [target]

# Variables
BINARY_NAME = proxima
BUILD_DIR = ../bin
SOURCE_DIR = .
MAIN_FILE = cmd/proxima/main.go
GO_FILES = $(shell find $(SOURCE_DIR) -name "*.go")

# Default target
.PHONY: help
help:
	@echo "Proxima - Proxmox VM Manager"
	@echo ""
	@echo "Available targets:"
	@echo "  build    - Build binary to $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "  clean    - Remove compiled binary"
	@echo "  help     - Display this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make clean"

.PHONY: build
build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES)
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: clean
clean:
	@echo "Cleaning build files..."
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Clean completed"

.PHONY: test
test:
	@echo "Running tests..."
	@go test ./...
	@echo "Tests completed"