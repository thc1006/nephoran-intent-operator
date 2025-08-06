# Nephoran Intent Operator - Essential Makefile
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
REGISTRY ?= us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran
SERVICES := llm-processor nephio-bridge oran-adaptor rag-api

.PHONY: help build test lint generate docker-build docker-push deploy clean

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

build: ## Build all 4 core services in parallel
	@echo "Building core services in parallel..."
	@mkdir -p bin
	@$(MAKE) -j4 build-go build-rag

build-go: ## Build Go services
	@go build -ldflags "-X main.version=$(VERSION)" -o bin/llm-processor ./cmd/llm-processor & \
	 go build -ldflags "-X main.version=$(VERSION)" -o bin/nephio-bridge ./cmd/nephio-bridge & \
	 go build -ldflags "-X main.version=$(VERSION)" -o bin/oran-adaptor ./cmd/oran-adaptor & \
	 wait

build-rag: ## Build RAG API (placeholder - Python service)
	@echo "RAG API build complete (Python service)"

test: ## Run basic tests
	@go test -v -race -coverprofile=coverage.out ./...

test-comprehensive: ## Run comprehensive Ginkgo v2 test suite with 90%+ coverage
	@echo "Running comprehensive test suite..."
	@./scripts/run-comprehensive-tests.sh --type all

test-unit: ## Run unit tests only
	@./scripts/run-comprehensive-tests.sh --type unit

test-integration: ## Run integration tests only
	@./scripts/run-comprehensive-tests.sh --type integration

test-performance: ## Run performance tests only
	@./scripts/run-comprehensive-tests.sh --type performance

test-coverage: ## Generate coverage report
	@./scripts/run-comprehensive-tests.sh --type unit
	@echo "Coverage report generated in test-results/coverage/coverage.html"

lint: ## Run golangci-lint
	@golangci-lint run ./...

generate: ## Generate CRDs and code
	@go generate ./...
	@controller-gen crd rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

docker-build: ## Build Docker images
	@echo "Building Docker images in parallel..."
	@$(MAKE) -j4 $(addprefix docker-build-, $(SERVICES))

docker-build-llm-processor docker-build-nephio-bridge docker-build-oran-adaptor: ## Build Go service images
	@docker build -t $(REGISTRY)/$(patsubst docker-build-%,%,$@):$(VERSION) --target $(patsubst docker-build-%,%,$@) .

docker-build-rag-api: ## Build RAG API image
	@docker build -t $(REGISTRY)/rag-api:$(VERSION) -f rag-python/Dockerfile ./rag-python

docker-push: ## Push Docker images
	@for service in $(SERVICES); do docker push $(REGISTRY)/$$service:$(VERSION) & done; wait

deploy: ## Deploy to Kubernetes
	@kustomize build deployments/kustomize/overlays/dev | kubectl apply -f -

clean: ## Clean build artifacts
	@rm -rf bin/ coverage.out coverage.html
	@docker system prune -f --filter label=project=nephoran