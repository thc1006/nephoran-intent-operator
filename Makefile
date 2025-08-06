# Nephoran Intent Operator - Simplified Makefile
# Essential build targets for core development workflow

# Build configuration
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
REGISTRY ?= us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran

# Core services
SERVICES := llm-processor nephio-bridge oran-adaptor

.PHONY: help build test lint generate docker-build docker-push deploy clean

help: ## Show available targets
	@echo "Nephoran Intent Operator - Available Make targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build all core services in parallel
	@echo "Building core services..."
	@$(MAKE) -j $(words $(SERVICES)) $(addprefix build-, $(SERVICES))

build-llm-processor: ## Build LLM processor service
	@echo "Building llm-processor..."
	@go build -ldflags "-X main.version=$(VERSION)" -o bin/llm-processor ./cmd/llm-processor

build-nephio-bridge: ## Build Nephio bridge service  
	@echo "Building nephio-bridge..."
	@go build -ldflags "-X main.version=$(VERSION)" -o bin/nephio-bridge ./cmd/nephio-bridge

build-oran-adaptor: ## Build O-RAN adaptor service
	@echo "Building oran-adaptor..."
	@go build -ldflags "-X main.version=$(VERSION)" -o bin/oran-adaptor ./cmd/oran-adaptor

test: ## Run tests with coverage
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run ./...

generate: ## Generate CRDs and mocks
	@echo "Generating code..."
	@go generate ./...
	@controller-gen crd:maxDescLen=0 rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

docker-build: ## Build Docker images for all services
	@echo "Building Docker images..."
	@$(MAKE) -j $(words $(SERVICES)) $(addprefix docker-build-, $(SERVICES))
	@$(MAKE) docker-build-rag-api

docker-build-llm-processor: ## Build LLM processor Docker image
	@docker build -t $(REGISTRY)/llm-processor:$(VERSION) --target llm-processor .

docker-build-nephio-bridge: ## Build Nephio bridge Docker image  
	@docker build -t $(REGISTRY)/nephio-bridge:$(VERSION) --target nephio-bridge .

docker-build-oran-adaptor: ## Build O-RAN adaptor Docker image
	@docker build -t $(REGISTRY)/oran-adaptor:$(VERSION) --target oran-adaptor .

docker-build-rag-api: ## Build RAG API Docker image (Python)
	@docker build -t $(REGISTRY)/rag-api:$(VERSION) -f rag-python/Dockerfile ./rag-python

docker-push: ## Push Docker images to registry
	@echo "Pushing Docker images..."
	@for service in $(SERVICES) rag-api; do \
		echo "Pushing $$service..."; \
		docker push $(REGISTRY)/$$service:$(VERSION); \
	done

deploy: ## Deploy to Kubernetes using Kustomize
	@echo "Deploying to Kubernetes..."
	@kustomize build deployments/kustomize/overlays/dev | kubectl apply -f -

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@docker system prune -f --filter label=project=nephoran