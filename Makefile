# Build configuration
APP_NAME := telecom-llm-automation
# Cross-platform version detection
ifeq ($(OS),Windows_NT)
    VERSION := $(shell git describe --tags --always --dirty 2>nul || echo "dev")
    GOOS := windows
    GOBIN_DIR := bin
else
    VERSION := $(shell git describe --tags --always --dirty)
    GOOS := $(shell go env GOOS)
    GOBIN_DIR := bin
endif
REGISTRY ?= us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran
IMAGES = \
  $(REGISTRY)/llm-processor \
  $(REGISTRY)/nephio-bridge

.PHONY: help setup-dev build-all deploy-dev test-integration lint generate

generate: ## Generate code
	@echo "--- Generating Code ---"
	controller-gen object:headerFile=hack/boilerplate.go.txt paths="github.com/thc1006/nephoran-intent-operator/api/v1"

lint: ## Run linters
	@echo "--- Running Linters ---"
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...
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
ifeq ($(OS),Windows_NT)
	python -m pip install -r requirements-rag.txt
else
	pip3 install -r requirements-rag.txt
endif

build-all: ## Build all components
	@echo "--- Building all components ---"
	$(MAKE) build-llm-processor
	$(MAKE) build-nephio-bridge  
	$(MAKE) build-oran-adaptor

deploy-dev: ## Deploy to development environment
	@echo "--- Deploying to development environment ---"
	kustomize build deployments/kustomize/overlays/dev | kubectl apply -f -
	kubectl apply -f deployments/crds/networkintent_crd.yaml
	kubectl apply -f deployments/crds/managedelement_crd.yaml

test-integration: ## Run integration tests
	@echo "--- Running integration tests ---"
	@echo "--- Setting up envtest ---"
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
ifeq ($(OS),Windows_NT)
	$(shell go env GOPATH)\bin\setup-envtest.exe use 1.29.0 --bin-dir $(shell go env GOPATH)\bin
	set KUBEBUILDER_ASSETS=$(shell $(shell go env GOPATH)\bin\setup-envtest.exe use 1.29.0 --bin-dir $(shell go env GOPATH)\bin -p path) && go test -v ./pkg/controllers/...
	-python -m pytest
else
	$(shell go env GOPATH)/bin/setup-envtest use 1.29.0 --bin-dir $(shell go env GOPATH)/bin
	KUBEBUILDER_ASSETS=$(shell $(shell go env GOPATH)/bin/setup-envtest use 1.29.0 --bin-dir $(shell go env GOPATH)/bin -p path) go test -v ./pkg/controllers/...
	-python3 -m pytest
endif

# TODO: Add build targets for each service (e.g., build-llm-processor)
.PHONY: build-llm-processor build-nephio-bridge build-oran-adaptor docker-build docker-push populate-kb

docker-build: ## Build docker images for all services
	@echo "--- Building Docker Images ---"
	docker build -t $(REGISTRY)/llm-processor:$(VERSION) -f cmd/llm-processor/Dockerfile .
	docker build -t $(REGISTRY)/nephio-bridge:$(VERSION) -f cmd/nephio-bridge/Dockerfile .
	docker build -t $(REGISTRY)/oran-adaptor:$(VERSION) -f cmd/oran-adaptor/Dockerfile .
	docker build -t $(REGISTRY)/rag-api:$(VERSION) -f pkg/rag/Dockerfile .

docker-push: ## Push docker images to the registry
	@echo "--- Pushing Docker Images ---"
	docker push $(REGISTRY)/llm-processor:$(VERSION)
	docker push $(REGISTRY)/nephio-bridge:$(VERSION)
	docker push $(REGISTRY)/oran-adaptor:$(VERSION)
	docker push $(REGISTRY)/rag-api:$(VERSION)

populate-kb: ## Populate the Weaviate knowledge base
	@echo "--- Populating Knowledge Base ---"
ifeq ($(OS),Windows_NT)
	set OPENAI_API_KEY=$(OPENAI_API_KEY) && set WEAVIATE_URL=$(WEAVIATE_URL) && python scripts/populate_vector_store.py
else
	OPENAI_API_KEY=$(OPENAI_API_KEY) WEAVIATE_URL=$(WEAVIATE_URL) python3 scripts/populate_vector_store.py
endif

build-llm-processor:
	@echo "Building llm-processor..."
ifeq ($(OS),Windows_NT)
	go build -o bin\llm-processor.exe cmd\llm-processor\main.go
else
	go build -o bin/llm-processor cmd/llm-processor/main.go
endif

build-nephio-bridge:
	@echo "Building nephio-bridge..."
ifeq ($(OS),Windows_NT)
	go build -o bin\nephio-bridge.exe cmd\nephio-bridge\main.go
else
	go build -o bin/nephio-bridge cmd/nephio-bridge/main.go
endif

build-oran-adaptor:
	@echo "Building oran-adaptor..."
ifeq ($(OS),Windows_NT)
	go build -o bin\oran-adaptor.exe cmd\oran-adaptor\main.go
else
	go build -o bin/oran-adaptor cmd/oran-adaptor/main.go
endif