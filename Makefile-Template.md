# Build configuration
APP_NAME := telecom-llm-automation
VERSION := $(shell git describe --tags --always --dirty)
REGISTRY ?= gcr.io/your-project

.PHONY: setup-dev
setup-dev: ## Set up development environment
	go mod download && go mod tidy
	pip3 install -r requirements-rag.txt
	$(MAKE) install-tools

.PHONY: build-all
build-all: ## Build all components
	$(MAKE) build-llm-processor
	$(MAKE) build-nephio-bridge  
	$(MAKE) build-oran-adaptor

.PHONY: deploy-dev
deploy-dev: ## Deploy to development environment
	kustomize build deployments/dev | kubectl apply -f -
	$(MAKE) kpt-apply

.PHONY: test-integration
test-integration: ## Run integration tests
	go test -v ./tests/integration/...
	python3 -m pytest tests/rag/ -v
