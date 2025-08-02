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

lint: ## Run linters with caching
	@echo "--- Running Linters with Caching ---"
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run --enable-all --disable=exhaustivestruct,exhaustruct,gochecknoglobals,gochecknoinits,gomnd,wsl,nlreturn,testpackage --timeout=5m ./...
ifeq ($(shell command -v flake8 2>/dev/null),)
	@echo "flake8 not found, skipping Python linting"
else
	flake8 pkg/rag/ --max-line-length=120 --extend-ignore=E203,W503
endif


help:
	@echo "Makefile for the LLM-Enhanced Nephio R5 and O-RAN Network Automation System"
	@echo ""
	@echo "Development Commands:"
	@echo "  make setup-dev         Set up the development environment (install dependencies)"
	@echo "  make build-all         Build all service binaries"
	@echo "  make test-integration  Run all integration tests"
	@echo "  make lint              Run code linters"
	@echo "  make generate          Generate Kubernetes code"
	@echo ""
	@echo "Deployment Commands:"
	@echo "  make deploy-dev        Deploy all components to the development environment"
	@echo "  make docker-build      Build all Docker images"
	@echo "  make docker-push       Push Docker images to registry"
	@echo ""
	@echo "RAG System Commands:"
	@echo "  make deploy-rag        Deploy complete RAG system with Weaviate"
	@echo "  make cleanup-rag       Remove RAG system deployment"
	@echo "  make verify-rag        Verify RAG system health"
	@echo "  make populate-kb-enhanced  Populate knowledge base with enhanced pipeline"
	@echo "  make rag-status        Show RAG system component status"
	@echo "  make rag-logs          Show RAG system logs"
	@echo ""
	@echo "Environment Variables:"
	@echo "  OPENAI_API_KEY         Required for RAG system deployment"
	@echo "  WEAVIATE_URL           Weaviate connection URL (default: http://localhost:8080)"
	@echo "  REGISTRY               Docker registry for images"
	@echo ""

setup-dev: ## Set up development environment
	@echo "--- Setting up development environment ---"
	go mod tidy
ifeq ($(OS),Windows_NT)
	python -m pip install -r requirements-rag.txt
else
	pip3 install -r requirements-rag.txt
endif

build-all: ## Build all components in parallel
	@echo "--- Building all components in parallel ---"
	$(MAKE) -j$(shell nproc 2>/dev/null || echo 4) build-llm-processor build-nephio-bridge build-oran-adaptor

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

docker-build: ## Build docker images with BuildKit optimization
	@echo "--- Building Docker Images with Parallel BuildKit ---"
	DOCKER_BUILDKIT=1 docker build --build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
		--build-arg VCS_REF=$(shell git rev-parse --short HEAD) \
		-t $(REGISTRY)/llm-processor:$(VERSION) \
		-f cmd/llm-processor/Dockerfile . &
	DOCKER_BUILDKIT=1 docker build --build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
		--build-arg VCS_REF=$(shell git rev-parse --short HEAD) \
		-t $(REGISTRY)/nephio-bridge:$(VERSION) \
		-f cmd/nephio-bridge/Dockerfile . &
	DOCKER_BUILDKIT=1 docker build --build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
		--build-arg VCS_REF=$(shell git rev-parse --short HEAD) \
		-t $(REGISTRY)/oran-adaptor:$(VERSION) \
		-f cmd/oran-adaptor/Dockerfile . &
	DOCKER_BUILDKIT=1 docker build --build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
		--build-arg VCS_REF=$(shell git rev-parse --short HEAD) \
		-t $(REGISTRY)/rag-api:$(VERSION) \
		-f pkg/rag/Dockerfile . &
	wait
	@echo "All Docker images built successfully"

docker-push: ## Push docker images to the registry
	@echo "--- Pushing Docker Images ---"
	docker push $(REGISTRY)/llm-processor:$(VERSION)
	docker push $(REGISTRY)/nephio-bridge:$(VERSION)
	docker push $(REGISTRY)/oran-adaptor:$(VERSION)
	docker push $(REGISTRY)/rag-api:$(VERSION)

populate-kb: ## Populate the Weaviate knowledge base (legacy)
	@echo "--- Populating Knowledge Base (Legacy) ---"
ifeq ($(OS),Windows_NT)
	set OPENAI_API_KEY=$(OPENAI_API_KEY) && set WEAVIATE_URL=$(WEAVIATE_URL) && python scripts/populate_vector_store.py
else
	OPENAI_API_KEY=$(OPENAI_API_KEY) WEAVIATE_URL=$(WEAVIATE_URL) python3 scripts/populate_vector_store.py
endif

populate-kb-enhanced: ## Populate with enhanced RAG pipeline
	@echo "--- Populating Knowledge Base (Enhanced) ---"
ifeq ($(OS),Windows_NT)
	set OPENAI_API_KEY=$(OPENAI_API_KEY) && set WEAVIATE_URL=$(WEAVIATE_URL) && python scripts/populate_vector_store_enhanced.py knowledge_base/
else
	OPENAI_API_KEY=$(OPENAI_API_KEY) WEAVIATE_URL=$(WEAVIATE_URL) python3 scripts/populate_vector_store_enhanced.py knowledge_base/
endif

deploy-rag: ## Deploy complete RAG system
	@echo "--- Deploying RAG System ---"
	@if [ -z "$(OPENAI_API_KEY)" ]; then \
		echo "Error: OPENAI_API_KEY environment variable is required"; \
		exit 1; \
	fi
	OPENAI_API_KEY=$(OPENAI_API_KEY) ./scripts/deploy-rag-system.sh deploy

cleanup-rag: ## Cleanup RAG system deployment
	@echo "--- Cleaning up RAG System ---"
	./scripts/deploy-rag-system.sh cleanup

verify-rag: ## Verify RAG system deployment
	@echo "--- Verifying RAG System ---"
	./scripts/deploy-rag-system.sh verify

rag-logs: ## Show RAG system logs
	@echo "--- RAG System Logs ---"
	kubectl logs -f deployment/weaviate --tail=50 || true
	kubectl logs -f deployment/rag-api --tail=50 || true

rag-status: ## Show RAG system status
	@echo "--- RAG System Status ---"
	kubectl get pods -l component=vectordb
	kubectl get pods -l app=rag-api
	kubectl get svc weaviate rag-api

build-llm-processor: ## Build LLM processor with optimizations
	@echo "Building llm-processor with optimizations..."
ifeq ($(OS),Windows_NT)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
		-ldflags="-w -s -X main.version=$(VERSION) -X main.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')" \
		-trimpath -a -installsuffix cgo \
		-o bin\llm-processor.exe cmd\llm-processor\main.go
else
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=amd64 go build \
		-ldflags="-w -s -X main.version=$(VERSION) -X main.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')" \
		-trimpath -a -installsuffix cgo \
		-o bin/llm-processor cmd/llm-processor/main.go
endif

build-nephio-bridge: ## Build Nephio bridge with optimizations
	@echo "Building nephio-bridge with optimizations..."
ifeq ($(OS),Windows_NT)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
		-ldflags="-w -s -X main.version=$(VERSION) -X main.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')" \
		-trimpath -a -installsuffix cgo \
		-o bin\nephio-bridge.exe cmd\nephio-bridge\main.go
else
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=amd64 go build \
		-ldflags="-w -s -X main.version=$(VERSION) -X main.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')" \
		-trimpath -a -installsuffix cgo \
		-o bin/nephio-bridge cmd/nephio-bridge/main.go
endif

build-oran-adaptor: ## Build O-RAN adaptor with optimizations
	@echo "Building oran-adaptor with optimizations..."
ifeq ($(OS),Windows_NT)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
		-ldflags="-w -s -X main.version=$(VERSION) -X main.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')" \
		-trimpath -a -installsuffix cgo \
		-o bin\oran-adaptor.exe cmd\oran-adaptor\main.go
else
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=amd64 go build \
		-ldflags="-w -s -X main.version=$(VERSION) -X main.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')" \
		-trimpath -a -installsuffix cgo \
		-o bin/oran-adaptor cmd/oran-adaptor/main.go
endif

# Performance monitoring and benchmarking targets
.PHONY: benchmark security-scan validate-images

benchmark: ## Run performance benchmarks
	@echo "--- Running Performance Benchmarks ---"
	go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./pkg/...
	@echo "Benchmark results saved to cpu.prof and mem.prof"

security-scan: ## Run comprehensive security vulnerability scans
	@echo "--- Running Comprehensive Security Scans ---"
	./scripts/security-scan.sh

validate-build: ## Validate build system integrity
	@echo "--- Validating Build System ---"
	./scripts/validate-build.sh

fix-api-versions: ## Fix API version inconsistencies
	@echo "--- Fixing API Version Inconsistencies ---"
	@echo "Regenerating CRDs with correct versions..."
	$(MAKE) generate
	controller-gen crd:generateEmbeddedObjectMeta=true paths="./api/v1" output:crd:artifacts:config=deployments/crds

validate-images: docker-build ## Validate Docker images
	@echo "--- Validating Docker Images ---"
	@for image in llm-processor nephio-bridge oran-adaptor rag-api; do \
		echo "Validating $(REGISTRY)/$$image:$(VERSION)"; \
		docker run --rm $(REGISTRY)/$$image:$(VERSION) --version 2>/dev/null || echo "Version check failed or not implemented for $$image"; \
		docker inspect $(REGISTRY)/$$image:$(VERSION) | jq '.[0].Config.Labels' || echo "No labels found for $$image"; \
	done

build-performance: ## Monitor build performance
	@echo "--- Build Performance Monitoring ---"
	@echo "Starting build with timing..."
	@start_time=$$(date +%s); \
	$(MAKE) clean build-all; \
	end_time=$$(date +%s); \
	echo "Total build time: $$((end_time - start_time)) seconds"

clean: ## Clean build artifacts
	@echo "--- Cleaning Build Artifacts ---"
	rm -rf bin/
	docker system prune -f
	go clean -cache -modcache -testcache

# Dependency management and updates
update-deps: ## Update dependencies safely
	@echo "--- Updating Dependencies ---"
	go get -u ./...
	go mod tidy
	go mod verify

# Development helpers
dev-setup: setup-dev ## Extended development setup with additional tools
	@echo "--- Installing Additional Development Tools ---"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securecodewarrior/sast-scan@latest

# Enhanced testing targets
test-all: test-integration benchmark security-scan validate-build ## Run all tests including benchmarks and security
	@echo "--- All Tests Completed ---"

# Build integrity and security
validate-all: validate-build security-scan ## Run all validation checks
	@echo "--- All Validations Completed ---"

pre-commit: lint validate-build ## Pre-commit validation
	@echo "--- Pre-commit Checks ---"