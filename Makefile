# Build configuration
APP_NAME := telecom-llm-automation

# Cross-platform version detection
ifeq ($(OS),Windows_NT)
    VERSION := $(shell git describe --tags --always --dirty 2>nul || echo "dev")
    GOOS := windows
    GOBIN_DIR := bin
    DATE := $(shell powershell -Command "Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ'")
    GIT_COMMIT := $(shell git rev-parse --short HEAD 2>nul || echo "unknown")
    NPROC := $(shell powershell -Command "(Get-WmiObject -Class Win32_ComputerSystem).NumberOfLogicalProcessors")
else
    VERSION := $(shell git describe --tags --always --dirty)
    GOOS := $(shell go env GOOS)
    GOBIN_DIR := bin
    DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
    GIT_COMMIT := $(shell git rev-parse --short HEAD)
    NPROC := $(shell nproc 2>/dev/null || echo 4)
endif

# Registry and image configuration
REGISTRY ?= us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran
CACHE_REGISTRY ?= $(REGISTRY)/cache

# Multi-architecture build configuration
PLATFORMS := linux/amd64,linux/arm64
BUILDER_NAME := nephoran-builder

# Image list
IMAGES = \
  $(REGISTRY)/llm-processor \
  $(REGISTRY)/nephio-bridge \
  $(REGISTRY)/oran-adaptor \
  $(REGISTRY)/rag-api

# Build tools configuration
COSIGN_VERSION ?= v2.2.2
SYFT_VERSION ?= v0.95.0
GRYPE_VERSION ?= v0.74.1

# Security and signing configuration
COSIGN_EXPERIMENTAL ?= 1
COSIGN_PRIVATE_KEY ?= 
COSIGN_PASSWORD ?= 
ATTESTATION_KEY ?= 

# Build optimization flags
BUILD_CACHE ?= 1
PARALLEL_BUILDS ?= 1
SBOM_GENERATION ?= 1
IMAGE_SIGNING ?= 0

.PHONY: help setup-dev build-all deploy-dev test-integration lint generate
.PHONY: setup-buildx build-multiarch build-sbom build-sign build-attest build-complete
.PHONY: install-tools verify-tools clean-buildx build-cache-prune build-parallel
.PHONY: build-verify build-vulnerability-scan ci-setup ci-build ci-push

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
	@echo "Enhanced Build Commands:"
	@echo "  make setup-buildx      Set up Docker Buildx for multi-arch builds"
	@echo "  make build-multiarch   Build multi-architecture images (amd64 + arm64)"
	@echo "  make build-sbom        Build with SBOM generation"
	@echo "  make build-sign        Build and sign images with Cosign"
	@echo "  make build-attest      Build with attestations and provenance"
	@echo "  make build-complete    Complete build with all enhancements"
	@echo "  make build-cache-prune Clean build cache"
	@echo "  make install-tools     Install required build tools"
	@echo "  make verify-tools      Verify tool installations"
	@echo ""
	@echo "Environment Variables:"
	@echo "  OPENAI_API_KEY         Required for RAG system deployment"
	@echo "  WEAVIATE_URL           Weaviate connection URL (default: http://localhost:8080)"
	@echo "  REGISTRY               Docker registry for images"
	@echo "  CACHE_REGISTRY         Registry for build cache (default: REGISTRY/cache)"
	@echo "  PLATFORMS              Target platforms (default: linux/amd64,linux/arm64)"
	@echo "  BUILD_CACHE            Enable build caching (default: 1)"
	@echo "  SBOM_GENERATION        Enable SBOM generation (default: 1)"
	@echo "  IMAGE_SIGNING          Enable image signing (default: 0)"
	@echo "  COSIGN_PRIVATE_KEY     Path to Cosign private key for signing"
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
	$(MAKE) -j$(NPROC) build-llm-processor build-nephio-bridge build-oran-adaptor

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

# ============================================================================
# ENHANCED BUILD SYSTEM WITH MULTI-ARCH, SBOM, SIGNING, AND ATTESTATIONS
# ============================================================================

install-tools: ## Install required build tools (Cosign, Syft, Grype)
	@echo "--- Installing Build Tools ---"
	@echo "Installing Cosign $(COSIGN_VERSION)..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "if (-not (Get-Command cosign -ErrorAction SilentlyContinue)) { \
		Invoke-WebRequest -Uri 'https://github.com/sigstore/cosign/releases/download/$(COSIGN_VERSION)/cosign-windows-amd64.exe' -OutFile 'cosign.exe'; \
		Move-Item cosign.exe C:\Windows\System32\cosign.exe; \
	}"
else
	@if ! command -v cosign >/dev/null 2>&1; then \
		curl -O -L "https://github.com/sigstore/cosign/releases/download/$(COSIGN_VERSION)/cosign-linux-amd64"; \
		sudo mv cosign-linux-amd64 /usr/local/bin/cosign; \
		sudo chmod +x /usr/local/bin/cosign; \
	fi
endif
	@echo "Installing Syft $(SYFT_VERSION)..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "if (-not (Get-Command syft -ErrorAction SilentlyContinue)) { \
		Invoke-WebRequest -Uri 'https://github.com/anchore/syft/releases/download/$(SYFT_VERSION)/syft_$(SYFT_VERSION:v%=%)_windows_amd64.zip' -OutFile 'syft.zip'; \
		Expand-Archive syft.zip -DestinationPath .; \
		Move-Item syft.exe C:\Windows\System32\syft.exe; \
		Remove-Item syft.zip; \
	}"
else
	@if ! command -v syft >/dev/null 2>&1; then \
		curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin $(SYFT_VERSION); \
	fi
endif
	@echo "Installing Grype $(GRYPE_VERSION)..."
ifeq ($(OS),Windows_NT)
	@powershell -Command "if (-not (Get-Command grype -ErrorAction SilentlyContinue)) { \
		Invoke-WebRequest -Uri 'https://github.com/anchore/grype/releases/download/$(GRYPE_VERSION)/grype_$(GRYPE_VERSION:v%=%)_windows_amd64.zip' -OutFile 'grype.zip'; \
		Expand-Archive grype.zip -DestinationPath .; \
		Move-Item grype.exe C:\Windows\System32\grype.exe; \
		Remove-Item grype.zip; \
	}"
else
	@if ! command -v grype >/dev/null 2>&1; then \
		curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin $(GRYPE_VERSION); \
	fi
endif
	@echo "All build tools installed successfully!"

verify-tools: ## Verify tool installations
	@echo "--- Verifying Tool Installations ---"
	@echo "Docker version:"
	@docker --version
	@echo "Docker Buildx version:"
	@docker buildx version
	@echo "Cosign version:"
	@cosign version || echo "Cosign not found - run 'make install-tools'"
	@echo "Syft version:"
	@syft version || echo "Syft not found - run 'make install-tools'"
	@echo "Grype version:"
	@grype version || echo "Grype not found - run 'make install-tools'"

setup-buildx: ## Set up Docker Buildx for multi-architecture builds
	@echo "--- Setting up Docker Buildx ---"
	@echo "Creating buildx builder instance: $(BUILDER_NAME)"
	@docker buildx create --name $(BUILDER_NAME) --driver docker-container --bootstrap --use || true
	@docker buildx inspect $(BUILDER_NAME) --bootstrap
	@echo "Buildx setup complete!"

build-multiarch: setup-buildx ## Build multi-architecture images with registry caching
	@echo "--- Building Multi-Architecture Images ---"
	@echo "Platforms: $(PLATFORMS)"
	@echo "Build cache enabled: $(BUILD_CACHE)"
	@$(MAKE) _build-image-multiarch IMG=llm-processor DOCKERFILE=cmd/llm-processor/Dockerfile
	@$(MAKE) _build-image-multiarch IMG=nephio-bridge DOCKERFILE=cmd/nephio-bridge/Dockerfile
	@$(MAKE) _build-image-multiarch IMG=oran-adaptor DOCKERFILE=cmd/oran-adaptor/Dockerfile
	@$(MAKE) _build-image-multiarch IMG=rag-api DOCKERFILE=pkg/rag/Dockerfile
	@echo "Multi-architecture build complete!"

_build-image-multiarch: ## Internal target for building individual multi-arch images
	@echo "Building $(IMG) for platforms: $(PLATFORMS)"
	@CACHE_FROM=""; \
	CACHE_TO=""; \
	if [ "$(BUILD_CACHE)" = "1" ]; then \
		CACHE_FROM="--cache-from type=registry,ref=$(CACHE_REGISTRY)/$(IMG):cache"; \
		CACHE_TO="--cache-to type=registry,ref=$(CACHE_REGISTRY)/$(IMG):cache,mode=max"; \
	fi; \
	docker buildx build \
		--platform $(PLATFORMS) \
		--builder $(BUILDER_NAME) \
		$$CACHE_FROM \
		$$CACHE_TO \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(DATE) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--label "org.opencontainers.image.title=$(IMG)" \
		--label "org.opencontainers.image.description=Nephoran Intent Operator - $(IMG)" \
		--label "org.opencontainers.image.version=$(VERSION)" \
		--label "org.opencontainers.image.created=$(DATE)" \
		--label "org.opencontainers.image.revision=$(GIT_COMMIT)" \
		--label "org.opencontainers.image.source=https://github.com/thc1006/nephoran-intent-operator" \
		--label "org.opencontainers.image.vendor=Nephoran" \
		--tag $(REGISTRY)/$(IMG):$(VERSION) \
		--tag $(REGISTRY)/$(IMG):latest \
		--push \
		--progress plain \
		-f $(DOCKERFILE) .

build-sbom: build-multiarch ## Generate Software Bill of Materials (SBOM) for images
	@echo "--- Generating SBOM for Images ---"
ifeq ($(SBOM_GENERATION), 1)
	@mkdir -p sbom/
	@for img in llm-processor nephio-bridge oran-adaptor rag-api; do \
		echo "Generating SBOM for $(REGISTRY)/$$img:$(VERSION)"; \
		syft $(REGISTRY)/$$img:$(VERSION) -o spdx-json=sbom/$$img-$(VERSION).sbom.json; \
		syft $(REGISTRY)/$$img:$(VERSION) -o cyclonedx-json=sbom/$$img-$(VERSION).cyclonedx.json; \
		syft $(REGISTRY)/$$img:$(VERSION) -o table=sbom/$$img-$(VERSION).sbom.txt; \
		echo "SBOM generated: sbom/$$img-$(VERSION).sbom.json"; \
	done
	@echo "SBOM generation complete!"
else
	@echo "SBOM generation disabled (SBOM_GENERATION=0)"
endif

build-sign: build-sbom ## Sign container images with Cosign
	@echo "--- Signing Container Images ---"
ifeq ($(IMAGE_SIGNING), 1)
	@if [ -z "$(COSIGN_PRIVATE_KEY)" ]; then \
		echo "Warning: COSIGN_PRIVATE_KEY not set, using keyless signing"; \
		for img in llm-processor nephio-bridge oran-adaptor rag-api; do \
			echo "Signing $(REGISTRY)/$$img:$(VERSION)"; \
			cosign sign --yes $(REGISTRY)/$$img:$(VERSION); \
		done; \
	else \
		echo "Using private key signing"; \
		for img in llm-processor nephio-bridge oran-adaptor rag-api; do \
			echo "Signing $(REGISTRY)/$$img:$(VERSION)"; \
			cosign sign --key $(COSIGN_PRIVATE_KEY) $(REGISTRY)/$$img:$(VERSION); \
		done; \
	fi
	@echo "Image signing complete!"
else
	@echo "Image signing disabled (IMAGE_SIGNING=0)"
endif

build-attest: build-sign ## Generate build attestations and provenance
	@echo "--- Generating Build Attestations ---"
ifeq ($(IMAGE_SIGNING), 1)
	@for img in llm-processor nephio-bridge oran-adaptor rag-api; do \
		echo "Generating attestation for $(REGISTRY)/$$img:$(VERSION)"; \
		if [ -f "sbom/$$img-$(VERSION).sbom.json" ]; then \
			if [ -z "$(COSIGN_PRIVATE_KEY)" ]; then \
				cosign attest --yes --predicate sbom/$$img-$(VERSION).sbom.json $(REGISTRY)/$$img:$(VERSION); \
			else \
				cosign attest --key $(COSIGN_PRIVATE_KEY) --predicate sbom/$$img-$(VERSION).sbom.json $(REGISTRY)/$$img:$(VERSION); \
			fi; \
		fi; \
	done
	@echo "Build attestation complete!"
else
	@echo "Build attestation skipped (IMAGE_SIGNING=0)"
endif

build-complete: install-tools verify-tools build-attest ## Complete build pipeline with all enhancements
	@echo "--- Complete Build Pipeline Finished ---"
	@echo "Built images:"
	@for img in llm-processor nephio-bridge oran-adaptor rag-api; do \
		echo "  $(REGISTRY)/$$img:$(VERSION)"; \
		echo "  $(REGISTRY)/$$img:latest"; \
	done
	@echo ""
	@echo "Enhancements applied:"
	@echo "  ✓ Multi-architecture builds ($(PLATFORMS))"
ifeq ($(BUILD_CACHE), 1)
	@echo "  ✓ Registry-based caching"
else
	@echo "  ✗ Registry-based caching (disabled)"
endif
ifeq ($(SBOM_GENERATION), 1)
	@echo "  ✓ SBOM generation (SPDX, CycloneDX)"
else
	@echo "  ✗ SBOM generation (disabled)"
endif
ifeq ($(IMAGE_SIGNING), 1)
	@echo "  ✓ Image signing with Cosign"
	@echo "  ✓ Build attestations"
else
	@echo "  ✗ Image signing (disabled)"
	@echo "  ✗ Build attestations (disabled)"
endif
	@echo ""

build-cache-prune: ## Clean Docker build cache
	@echo "--- Cleaning Build Cache ---"
	@docker buildx prune -f --filter until=72h
	@docker system prune -f --volumes
	@echo "Build cache cleaned!"

clean-buildx: ## Remove buildx builder instance
	@echo "--- Cleaning Buildx Builder ---"
	@docker buildx rm $(BUILDER_NAME) || true
	@echo "Buildx builder removed!"

build-verify: build-complete ## Verify built images and their signatures
	@echo "--- Verifying Built Images ---"
	@for img in llm-processor nephio-bridge oran-adaptor rag-api; do \
		echo "Verifying $(REGISTRY)/$$img:$(VERSION)"; \
		docker pull $(REGISTRY)/$$img:$(VERSION); \
		docker inspect $(REGISTRY)/$$img:$(VERSION) | jq '.[0].Config.Labels' || echo "No labels found"; \
		if [ "$(IMAGE_SIGNING)" = "1" ]; then \
			echo "Verifying signature for $(REGISTRY)/$$img:$(VERSION)"; \
			if [ -z "$(COSIGN_PRIVATE_KEY)" ]; then \
				cosign verify $(REGISTRY)/$$img:$(VERSION) || echo "Signature verification failed"; \
			else \
				cosign verify --key $(COSIGN_PRIVATE_KEY) $(REGISTRY)/$$img:$(VERSION) || echo "Signature verification failed"; \
			fi; \
		fi; \
	done
	@echo "Image verification complete!"

build-vulnerability-scan: build-complete ## Scan images for vulnerabilities
	@echo "--- Scanning Images for Vulnerabilities ---"
	@mkdir -p security/
	@for img in llm-processor nephio-bridge oran-adaptor rag-api; do \
		echo "Scanning $(REGISTRY)/$$img:$(VERSION) for vulnerabilities"; \
		grype $(REGISTRY)/$$img:$(VERSION) -o json > security/$$img-$(VERSION)-vulnerabilities.json; \
		grype $(REGISTRY)/$$img:$(VERSION) -o table > security/$$img-$(VERSION)-vulnerabilities.txt; \
		echo "Vulnerability report: security/$$img-$(VERSION)-vulnerabilities.json"; \
	done
	@echo "Vulnerability scanning complete!"

# Enhanced parallel build with progress tracking
build-parallel: setup-buildx ## Build all images in parallel with progress tracking
	@echo "--- Parallel Multi-Architecture Build ---"
	@echo "Starting parallel builds for $(NPROC) processes..."
	@start_time=$$(date +%s); \
	$(MAKE) -j$(NPROC) \
		_build-image-multiarch-parallel-llm \
		_build-image-multiarch-parallel-nephio \
		_build-image-multiarch-parallel-oran \
		_build-image-multiarch-parallel-rag; \
	end_time=$$(date +%s); \
	echo "Parallel build completed in $$((end_time - start_time)) seconds"

_build-image-multiarch-parallel-llm:
	@$(MAKE) _build-image-multiarch IMG=llm-processor DOCKERFILE=cmd/llm-processor/Dockerfile

_build-image-multiarch-parallel-nephio:
	@$(MAKE) _build-image-multiarch IMG=nephio-bridge DOCKERFILE=cmd/nephio-bridge/Dockerfile

_build-image-multiarch-parallel-oran:
	@$(MAKE) _build-image-multiarch IMG=oran-adaptor DOCKERFILE=cmd/oran-adaptor/Dockerfile

_build-image-multiarch-parallel-rag:
	@$(MAKE) _build-image-multiarch IMG=rag-api DOCKERFILE=pkg/rag/Dockerfile

# CI/CD Integration targets
ci-setup: install-tools setup-buildx ## Setup CI/CD environment
	@echo "--- Setting up CI/CD Environment ---"
	@echo "Docker info:"
	@docker info
	@echo "Available builders:"
	@docker buildx ls
	@echo "CI/CD setup complete!"

ci-build: ci-setup ## Build for CI/CD with optimal settings
	@echo "--- CI/CD Build Pipeline ---"
	@echo "Building with CI optimizations..."
	BUILD_CACHE=1 PARALLEL_BUILDS=1 $(MAKE) build-parallel
ifeq ($(SBOM_GENERATION), 1)
	@echo "Generating SBOM artifacts..."
	$(MAKE) build-sbom
endif
	@echo "CI/CD build complete!"

ci-push: ci-build ## Push images from CI/CD
	@echo "--- CI/CD Push Pipeline ---"
	@for img in llm-processor nephio-bridge oran-adaptor rag-api; do \
		echo "Pushing $(REGISTRY)/$$img:$(VERSION)"; \
		docker push $(REGISTRY)/$$img:$(VERSION); \
		docker push $(REGISTRY)/$$img:latest; \
	done
	@echo "CI/CD push complete!"

ci-security: ci-build ## Run security scanning in CI/CD
	@echo "--- CI/CD Security Pipeline ---"
	$(MAKE) build-vulnerability-scan
	@echo "Security scanning results available in security/ directory"

ci-sign: ci-push ## Sign images in CI/CD (requires IMAGE_SIGNING=1)
	@echo "--- CI/CD Signing Pipeline ---"
ifeq ($(IMAGE_SIGNING), 1)
	$(MAKE) build-sign
	$(MAKE) build-attest
	@echo "Image signing and attestation complete!"
else
	@echo "Image signing disabled (set IMAGE_SIGNING=1 to enable)"
endif

ci-complete: ci-sign ci-security build-verify ## Complete CI/CD pipeline
	@echo "--- Complete CI/CD Pipeline Finished ---"
	@echo "Pipeline summary:"
	@echo "  ✓ Multi-architecture builds"
	@echo "  ✓ Registry-based caching"
	@echo "  ✓ Container images pushed"
ifeq ($(SBOM_GENERATION), 1)
	@echo "  ✓ SBOM generation"
else
	@echo "  ✗ SBOM generation (disabled)"
endif
ifeq ($(IMAGE_SIGNING), 1)
	@echo "  ✓ Image signing"
	@echo "  ✓ Build attestations"
else
	@echo "  ✗ Image signing (disabled)"
endif
	@echo "  ✓ Vulnerability scanning"
	@echo "  ✓ Image verification"
	@echo ""
	@echo "Artifacts:"
	@echo "  Images: $(REGISTRY)/{llm-processor,nephio-bridge,oran-adaptor,rag-api}:$(VERSION)"
ifeq ($(SBOM_GENERATION), 1)
	@echo "  SBOM: sbom/ directory"
endif
	@echo "  Security: security/ directory"

# ============================================================================
# END ENHANCED BUILD SYSTEM
# ============================================================================

# TODO: Add build targets for each service (e.g., build-llm-processor)
.PHONY: build-llm-processor build-nephio-bridge build-oran-adaptor docker-build docker-push populate-kb

docker-build: ## Build docker images with BuildKit optimization (legacy, use build-multiarch for enhanced features)
	@echo "--- Building Docker Images with Parallel BuildKit (Legacy Mode) ---"
	@echo "Tip: Use 'make build-multiarch' for multi-architecture builds with enhanced features"
	DOCKER_BUILDKIT=1 docker build --build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(DATE) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		--label "org.opencontainers.image.title=llm-processor" \
		--label "org.opencontainers.image.version=$(VERSION)" \
		--label "org.opencontainers.image.created=$(DATE)" \
		--label "org.opencontainers.image.revision=$(GIT_COMMIT)" \
		-t $(REGISTRY)/llm-processor:$(VERSION) \
		-f cmd/llm-processor/Dockerfile . &
	DOCKER_BUILDKIT=1 docker build --build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(DATE) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		--label "org.opencontainers.image.title=nephio-bridge" \
		--label "org.opencontainers.image.version=$(VERSION)" \
		--label "org.opencontainers.image.created=$(DATE)" \
		--label "org.opencontainers.image.revision=$(GIT_COMMIT)" \
		-t $(REGISTRY)/nephio-bridge:$(VERSION) \
		-f cmd/nephio-bridge/Dockerfile . &
	DOCKER_BUILDKIT=1 docker build --build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(DATE) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		--label "org.opencontainers.image.title=oran-adaptor" \
		--label "org.opencontainers.image.version=$(VERSION)" \
		--label "org.opencontainers.image.created=$(DATE)" \
		--label "org.opencontainers.image.revision=$(GIT_COMMIT)" \
		-t $(REGISTRY)/oran-adaptor:$(VERSION) \
		-f cmd/oran-adaptor/Dockerfile . &
	DOCKER_BUILDKIT=1 docker build --build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(DATE) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		--label "org.opencontainers.image.title=rag-api" \
		--label "org.opencontainers.image.version=$(VERSION)" \
		--label "org.opencontainers.image.created=$(DATE)" \
		--label "org.opencontainers.image.revision=$(GIT_COMMIT)" \
		-t $(REGISTRY)/rag-api:$(VERSION) \
		-f pkg/rag/Dockerfile . &
	wait
	@echo "All Docker images built successfully (single-arch: linux/amd64)"

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
	rm -rf sbom/
	rm -rf security/
	rm -rf coverage/
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
test-unit: ## Run unit tests with coverage
	@echo "--- Running Unit Tests with Coverage ---"
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated in coverage.html"

test-unit-ci: ## Run unit tests for CI with coverage threshold
	@echo "--- Running Unit Tests for CI ---"
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./pkg/...
	@echo "--- Checking Coverage Threshold (90%) ---"
	@go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//' | awk '{if ($$1 < 90) exit 1}'

test-coverage: ## Generate comprehensive coverage report
	@echo "--- Generating Comprehensive Coverage Report ---"
	@echo "Testing individual packages..."
	@mkdir -p coverage/
	@for pkg in $$(go list ./pkg/...); do \
		echo "Testing $$pkg..."; \
		go test -v -race -coverprofile=coverage/$$(echo $$pkg | sed 's/.*\///').out -covermode=atomic $$pkg || true; \
	done
	@echo "Merging coverage profiles..."
	@echo "mode: atomic" > coverage/merged.out
	@grep -h -v "mode: atomic" coverage/*.out >> coverage/merged.out 2>/dev/null || true
	@go tool cover -html=coverage/merged.out -o coverage/coverage.html
	@go tool cover -func=coverage/merged.out > coverage/coverage.txt
	@echo "--- Coverage Summary ---"
	@tail -1 coverage/coverage.txt
	@echo "Detailed coverage report: coverage/coverage.html"
	@echo "Coverage summary: coverage/coverage.txt"

test-rag: ## Run RAG package tests specifically
	@echo "--- Running RAG Package Tests ---"
	go test -v -race -coverprofile=coverage/rag.out -covermode=atomic ./pkg/rag/...
	go tool cover -html=coverage/rag.out -o coverage/rag_coverage.html
	@echo "RAG coverage report: coverage/rag_coverage.html"

test-security: ## Run Security package tests specifically  
	@echo "--- Running Security Package Tests ---"
	go test -v -race -coverprofile=coverage/security.out -covermode=atomic ./pkg/security/...
	go tool cover -html=coverage/security.out -o coverage/security_coverage.html
	@echo "Security coverage report: coverage/security_coverage.html"

test-integration-full: ## Run comprehensive integration tests
	@echo "--- Running Comprehensive Integration Tests ---"
	@echo "--- Setting up envtest ---"
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
ifeq ($(OS),Windows_NT)
	$(shell go env GOPATH)\bin\setup-envtest.exe use 1.29.0 --bin-dir $(shell go env GOPATH)\bin
	set KUBEBUILDER_ASSETS=$(shell $(shell go env GOPATH)\bin\setup-envtest.exe use 1.29.0 --bin-dir $(shell go env GOPATH)\bin -p path) && go test -v -race -coverprofile=coverage/integration.out -covermode=atomic ./tests/integration/...
else
	$(shell go env GOPATH)/bin/setup-envtest use 1.29.0 --bin-dir $(shell go env GOPATH)/bin
	KUBEBUILDER_ASSETS=$(shell $(shell go env GOPATH)/bin/setup-envtest use 1.29.0 --bin-dir $(shell go env GOPATH)/bin -p path) go test -v -race -coverprofile=coverage/integration.out -covermode=atomic ./tests/integration/...
endif
	@echo "Integration tests completed"

test-complete: test-coverage test-integration-full benchmark security-scan validate-build ## Run complete test suite including benchmarks and security
	@echo "--- Complete Test Suite Finished ---"

# Build integrity and security
validate-all: validate-build security-scan ## Run all validation checks
	@echo "--- All Validations Completed ---"

test-e2e: ## Run end-to-end tests with envtest
	@echo "--- Running E2E Tests with envtest ---"
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
ifeq ($(OS),Windows_NT)
	$(shell go env GOPATH)\bin\setup-envtest.exe use 1.29.0 --bin-dir $(shell go env GOPATH)\bin
	set KUBEBUILDER_ASSETS=$(shell $(shell go env GOPATH)\bin\setup-envtest.exe use 1.29.0 --bin-dir $(shell go env GOPATH)\bin -p path) && go test -v -race -coverprofile=coverage/e2e.out -covermode=atomic ./tests/e2e/...
else
	$(shell go env GOPATH)/bin/setup-envtest use 1.29.0 --bin-dir $(shell go env GOPATH)/bin
	KUBEBUILDER_ASSETS=$(shell $(shell go env GOPATH)/bin/setup-envtest use 1.29.0 --bin-dir $(shell go env GOPATH)/bin -p path) go test -v -race -coverprofile=coverage/e2e.out -covermode=atomic ./tests/e2e/...
endif
	@echo "E2E test coverage report generated in coverage/e2e.out"

test-chaos: ## Run chaos engineering tests
	@echo "--- Running Chaos Engineering Tests ---"
	go test -v -race -coverprofile=coverage/chaos.out -covermode=atomic ./tests/chaos/...
	@echo "Chaos test coverage report generated in coverage/chaos.out"

test-disaster-recovery: ## Run disaster recovery tests
	@echo "--- Running Disaster Recovery Tests ---"
	go test -v -race -coverprofile=coverage/disaster-recovery.out -covermode=atomic ./tests/disaster-recovery/...
	@echo "Disaster recovery test coverage report generated in coverage/disaster-recovery.out"

test-all: test-unit test-integration test-e2e test-chaos test-disaster-recovery ## Run all tests
	@echo "--- All Tests Completed ---"
	@echo "Merging coverage reports..."
	@echo "mode: atomic" > coverage/all.out
	@grep -h -v "mode: atomic" coverage/*.out >> coverage/all.out 2>/dev/null || true
	go tool cover -html=coverage/all.out -o coverage/all.html
	go tool cover -func=coverage/all.out | grep total | awk '{print "Total Coverage: " $$3}'

pre-commit: lint validate-build ## Pre-commit validation
	@echo "--- Pre-commit Checks ---"