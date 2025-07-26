# Build configuration
APP_NAME := telecom-llm-automation
VERSION := $(shell git describe --tags --always --dirty)
REGISTRY ?= gcr.io/your-project

.PHONY: help setup-dev build-all deploy-dev test-integration lint

lint: ## Run linters
	@echo "--- Running Linters ---"
	golangci-lint run ./...
	flake8 pkg/rag/


help:
	@echo "Makefile for the LLM-Enhanced Nephio R5 and O-RAN Network Automation System"
	@echo ""
	@echo "Usage:"
	@echo "  make setup-dev         Set up the development environment (install dependencies)"
	@echo "  make build-all         Build all service binaries"
	@echo "  make deploy-dev        Deploy all components to the development environment"
	@echo "  make test-integration  Run all integration tests"
	@echo ""

setup-dev: ## Set up development environment
	@echo "--- Setting up development environment ---"
	go mod tidy
	pip3 install -r requirements-rag.txt

build-all: ## Build all components
	@echo "--- Building all components ---"
	$(MAKE) build-llm-processor
	$(MAKE) build-nephio-bridge  
	$(MAKE) build-oran-adaptor

deploy-dev: ## Deploy to development environment
	@echo "--- Deploying to development environment ---"
	kustomize build deployments/kustomize/dev | kubectl apply -f -

test-integration: ## Run integration tests
	@echo "--- Running integration tests ---"
	go test -v ./...
	python3 -m pytest

# TODO: Add build targets for each service (e.g., build-llm-processor)
.PHONY: build-llm-processor build-nephio-bridge build-oran-adaptor
build-llm-processor:
	@echo "Building llm-processor..."
	# Add build command here

build-nephio-bridge:
	@echo "Building nephio-bridge..."
	# Add build command here

build-oran-adaptor:
	@echo "Building oran-adaptor..."
	# Add build command here