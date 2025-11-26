# Main Makefile for Proxima project
# Usage: make [target]

# Variables
SRC_DIR = src
RUN_DIR = run
INSTALL_SCRIPT = $(RUN_DIR)/install.sh
UNINSTALL_SCRIPT = $(RUN_DIR)/uninstall.sh

# Default target
.PHONY: help
help:
	@echo "Proxima - Proxmox VM Manager"
	@echo ""
	@echo "Available targets:"
	@echo "  build     - Build the project (calls make build in src/)"
	@echo "  clean     - Clean build files (calls make clean in src/)"
	@echo "  install   - Install Proxima to $$HOME/.local/bin"
	@echo "  uninstall - Remove Proxima from $$HOME/.local/bin"
	@echo "  help      - Display this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make install"
	@echo "  make clean"
	@echo "  make uninstall"

.PHONY: build
build:
	@echo "Building Proxima..."
	@$(MAKE) -C $(SRC_DIR) build
	@echo "Build completed successfully!"

.PHONY: clean
clean:
	@echo "Cleaning build files..."
	@$(MAKE) -C $(SRC_DIR) clean
	@echo "Clean completed successfully!"

.PHONY: test
test:
	@echo "Running tests..."
	@$(MAKE) -C $(SRC_DIR) test
	@echo "Tests completed successfully!"

.PHONY: install
install:
	@echo "Installing Proxima..."
	@if [ ! -f "$(INSTALL_SCRIPT)" ]; then \
		echo "Error: Installation script not found at $(INSTALL_SCRIPT)"; \
		exit 1; \
	fi
	@$(INSTALL_SCRIPT)

.PHONY: uninstall
uninstall:
	@echo "Uninstalling Proxima..."
	@if [ ! -f "$(UNINSTALL_SCRIPT)" ]; then \
		echo "Error: Uninstallation script not found at $(UNINSTALL_SCRIPT)"; \
		exit 1; \
	fi
	@$(UNINSTALL_SCRIPT)