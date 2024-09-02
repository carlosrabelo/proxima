MAKEFLAGS += --no-print-directory

.DEFAULT_GOAL := help

.PHONY: build clean fmt-check help quality test vet

help: ## Show available targets
	@echo "proxima - Available targets"
	@echo ""
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*## "} {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build bin/proxima
	@mkdir -p bin
	@go build -o bin/proxima ./proxima/cmd/proxima/

test: ## Run unit tests
	@go test ./...

vet: ## Run go vet
	@go vet ./...

fmt-check: ## Fail if gofmt reports unformatted files
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "$$unformatted"; \
		exit 1; \
	fi

quality: ## Run fmt-check, vet, test, and build
	@$(MAKE) fmt-check
	@$(MAKE) vet
	@$(MAKE) test
	@$(MAKE) build

clean: ## Remove bin/proxima; keep bin/.gitkeep
	@rm -f bin/proxima
