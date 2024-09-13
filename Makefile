MAKEFLAGS += --no-print-directory

.DEFAULT_GOAL := help

.PHONY: build clean deb fmt help install lint package quality test uninstall

BINARY_NAME := proxima

export BINARY_NAME

help: ## Show available targets
	@echo "proxima - Available targets"
	@echo ""
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*## "} {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build bin/proxima
	@./.make/build.sh

test: ## Run unit tests
	@./.make/test.sh

lint: ## Run go vet
	@go vet ./...

fmt: ## Format Go sources
	@go fmt ./...

quality: ## Run fmt-check, vet, test, and build
	@./.make/quality.sh

clean: ## Remove bin/proxima; keep bin/.gitkeep
	@./.make/clean.sh

install: ## Install to ~/.local/bin (requires prior make build)
	@./.make/install.sh

uninstall: ## Remove from ~/.local/bin (sudo only if needed)
	@./.make/uninstall.sh

package: ## Build a .deb into dist/ (requires prior make build)
	@./.make/package.sh

deb: package ## Alias for package
