# Makefile for nac

BINARY_NAME=nac
GO_VERSION=1.24

.PHONY: all build test clean lint fmt vet install help

all: fmt vet test build

## Build:
build: ## Build the binary
	go build -o $(BINARY_NAME) ./cmd/nac

install: ## Install the binary to $GOPATH/bin
	go install ./cmd/nac

## Development:
test: ## Run unit and integration tests
	go test ./... -v -count=1

fmt: ## Format the code
	go fmt ./...

vet: ## Vet the code
	go vet ./...

lint: ## Run linter (requires golangci-lint)
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping..."; \
	fi

clean: ## Clean build artifacts
	go clean
	rm -f $(BINARY_NAME)
	rm -rf test-debug/ debug-project/ test-project/ backups/

## Help:
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
