.DEFAULT_GOAL := help

BIN := bin/thmnzr
IMAGE := thmnzr

help: ## Show command list
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

fmt: ## Format Go source files
	gofmt -w cmd internal

fmt-check: ## Check Go formatting
	@test -z "$$(gofmt -l cmd internal)"

vet: ## Run go vet
	go vet ./...

test: ## Run tests
	go test ./...

build: ## Build the CLI binary
	go build -o $(BIN) ./cmd/thmnzr

check: fmt-check vet test build ## Run all checks

docker-build: ## Build the Docker image locally
	docker build -t $(IMAGE) .

docker-run: ## Show Docker CLI help in the image
	docker run --rm -i $(IMAGE) thmnzr --help

clean: ## Remove build artifacts
	rm -rf bin
