# Nephoran Intent Operator Makefile
# Production-ready Kubernetes Operator for O-RAN/5G Network Orchestration
# Target: Linux-only deployment on Ubuntu clusters (AWS EKS, Azure AKS, Google GKE)

# =============================================================================
# Core Configuration
# =============================================================================

# Version and Build Configuration
VERSION ?= 0.1.0
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
BUILD_USER := $(shell id -u -n 2>/dev/null || echo "unknown")

# Go Configuration
GO_VERSION := 1.24.6
GOPROXY := https://proxy.golang.org,direct
GOSUMDB := sum.golang.org

# Kubernetes and Controller Runtime Versions
KUBERNETES_VERSION := v1.32.1
CONTROLLER_RUNTIME_VERSION := v0.19.1
KUBEBUILDER_VERSION := 4.4.1
ENVTEST_K8S_VERSION := 1.32.1

# Project Configuration
PROJECT_NAME := nephoran-intent-operator
MODULE := github.com/thc1006/nephoran-intent-operator
IMAGE_REGISTRY ?= ghcr.io/nephio-project
IMAGE_TAG ?= $(VERSION)
PLATFORMS := linux/amd64,linux/arm64

# Build Configuration
CGO_ENABLED := 0
GOOS := linux
GOARCH := amd64

# Directory Structure
BIN_DIR := bin
BUILD_DIR := build
DIST_DIR := dist
CONFIG_DIR := config
CHARTS_DIR := charts
DOCS_DIR := docs

# Tool Versions
GOLANGCI_LINT_VERSION := v1.62.2
GINKGO_VERSION := v2.25.1
GOMEGA_VERSION := v1.38.1
MOCKGEN_VERSION := v0.6.0
TRIVY_VERSION := 0.58.4
HELM_VERSION := v3.18.6
YQ_VERSION := v4.44.6
KUSTOMIZE_VERSION := v5.6.0

# Security and Quality Gates
TRIVY_SEVERITY := HIGH,CRITICAL
GOSEC_SEVERITY := high
GOVULNCHECK_ENABLED := true

# =============================================================================
# Main Services Configuration
# =============================================================================

# Core operator and simulators
SERVICES := nephoran-operator porch-direct conductor-loop intent-ingest llm-processor
SIMULATORS := a1-sim e2-kmp-sim fcaps-sim conductor
TOOLS := webhook-manager oran-adaptor nephio-bridge

ALL_SERVICES := $(SERVICES) $(SIMULATORS) $(TOOLS)

# Service binary mappings
MANAGER_BIN := $(BIN_DIR)/nephoran-operator
PORCH_DIRECT_BIN := $(BIN_DIR)/porch-direct
CONDUCTOR_LOOP_BIN := $(BIN_DIR)/conductor-loop
INTENT_INGEST_BIN := $(BIN_DIR)/intent-ingest
LLM_PROCESSOR_BIN := $(BIN_DIR)/llm-processor
A1_SIM_BIN := $(BIN_DIR)/a1-sim
E2_KMP_SIM_BIN := $(BIN_DIR)/e2-kmp-sim
FCAPS_SIM_BIN := $(BIN_DIR)/fcaps-sim
CONDUCTOR_BIN := $(BIN_DIR)/conductor
WEBHOOK_MANAGER_BIN := $(BIN_DIR)/webhook-manager
ORAN_ADAPTOR_BIN := $(BIN_DIR)/oran-adaptor
NEPHIO_BRIDGE_BIN := $(BIN_DIR)/nephio-bridge

# =============================================================================
# Build Flags and Optimization
# =============================================================================

# Common build flags for production
COMMON_LDFLAGS := -w -s \
	-X $(MODULE)/pkg/version.Version=$(VERSION) \
	-X $(MODULE)/pkg/version.CommitHash=$(COMMIT_HASH) \
	-X $(MODULE)/pkg/version.BuildDate=$(BUILD_DATE) \
	-X $(MODULE)/pkg/version.GoVersion=$(shell go version | cut -d' ' -f3) \
	-X $(MODULE)/pkg/version.Platform=$(GOOS)/$(GOARCH)

# Production build flags (size optimized)
PROD_LDFLAGS := $(COMMON_LDFLAGS) -buildmode=pie
PROD_GCFLAGS := -trimpath
PROD_ASMFLAGS := -trimpath

# Debug build flags
DEBUG_LDFLAGS := $(COMMON_LDFLAGS)
DEBUG_GCFLAGS := -N -l

# Build tags
BUILD_TAGS := netgo,osusergo,static_build

# Go build flags
GOFLAGS := -mod=readonly -trimpath
GO_BUILD_FLAGS := -ldflags="$(PROD_LDFLAGS)" -gcflags="$(PROD_GCFLAGS)" -asmflags="$(PROD_ASMFLAGS)" -tags="$(BUILD_TAGS)"
GO_TEST_FLAGS := -race -coverprofile=coverage.out -covermode=atomic

# =============================================================================
# Tool Configuration
# =============================================================================

# Local tool binaries
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# Tool binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
GINKGO ?= $(LOCALBIN)/ginkgo-$(GINKGO_VERSION)
MOCKGEN ?= $(LOCALBIN)/mockgen-$(MOCKGEN_VERSION)
YQ ?= $(LOCALBIN)/yq-$(YQ_VERSION)
KUSTOMIZE ?= $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)
TRIVY ?= $(LOCALBIN)/trivy-$(TRIVY_VERSION)

# Tool versions
CONTROLLER_TOOLS_VERSION ?= v0.19.0
ENVTEST_VERSION ?= v0.19.1

# =============================================================================
# Default Target
# =============================================================================

.DEFAULT_GOAL := help

## help: Show this help message
.PHONY: help
help:
	@echo "Nephoran Intent Operator - Production Makefile"
	@echo "Target: Linux-only Kubernetes operator for O-RAN/5G orchestration"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Main services: $(ALL_SERVICES)"
	@echo "Build configuration: Go $(GO_VERSION), Kubernetes $(KUBERNETES_VERSION)"

# =============================================================================
# Development Environment
# =============================================================================

## setup: Initialize development environment
.PHONY: setup
setup: tools deps ## Setup development environment
	@echo "Setting up development environment..."
	@go mod tidy
	@go mod verify
	@echo "Development environment ready"

## deps: Download and verify dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@go mod verify

## tools: Install development tools
.PHONY: tools
tools: $(CONTROLLER_GEN) $(ENVTEST) $(GOLANGCI_LINT) $(GINKGO) $(MOCKGEN) $(YQ) $(KUSTOMIZE) $(TRIVY) ## Install development tools

$(CONTROLLER_GEN): $(LOCALBIN)
	@test -s $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION) && \
	mv $(LOCALBIN)/controller-gen $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)

$(ENVTEST): $(LOCALBIN)
	@test -s $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION) && \
	mv $(LOCALBIN)/setup-envtest $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)

$(GOLANGCI_LINT): $(LOCALBIN)
	@test -s $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION) || \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION) && \
	mv $(LOCALBIN)/golangci-lint $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)

$(GINKGO): $(LOCALBIN)
	@test -s $(LOCALBIN)/ginkgo-$(GINKGO_VERSION) || \
	GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION) && \
	mv $(LOCALBIN)/ginkgo $(LOCALBIN)/ginkgo-$(GINKGO_VERSION)

$(MOCKGEN): $(LOCALBIN)
	@test -s $(LOCALBIN)/mockgen-$(MOCKGEN_VERSION) || \
	GOBIN=$(LOCALBIN) go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION) && \
	mv $(LOCALBIN)/mockgen $(LOCALBIN)/mockgen-$(MOCKGEN_VERSION)

$(YQ): $(LOCALBIN)
	@test -s $(LOCALBIN)/yq-$(YQ_VERSION) || \
	curl -sSfL -o $(LOCALBIN)/yq-$(YQ_VERSION) "https://github.com/mikefarah/yq/releases/download/$(YQ_VERSION)/yq_linux_amd64" && \
	chmod +x $(LOCALBIN)/yq-$(YQ_VERSION)

$(KUSTOMIZE): $(LOCALBIN)
	@test -s $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION) || \
	curl -sSfL "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F$(KUSTOMIZE_VERSION)/kustomize_$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz" | \
	tar -xz -C $(LOCALBIN) && mv $(LOCALBIN)/kustomize $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)

$(TRIVY): $(LOCALBIN)
	@test -s $(LOCALBIN)/trivy-$(TRIVY_VERSION) || \
	curl -sSfL "https://github.com/aquasecurity/trivy/releases/download/v$(TRIVY_VERSION)/trivy_$(TRIVY_VERSION)_Linux-64bit.tar.gz" | \
	tar -xz -C $(LOCALBIN) trivy && mv $(LOCALBIN)/trivy $(LOCALBIN)/trivy-$(TRIVY_VERSION)

# =============================================================================
# Code Generation
# =============================================================================

## generate: Generate code (CRDs, clients, mocks)
.PHONY: generate
generate: $(CONTROLLER_GEN) $(MOCKGEN) ## Generate code
	@echo "Generating code..."
	@$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	@go generate ./...
	@echo "Code generation complete"

## manifests: Generate Kubernetes manifests
.PHONY: manifests
manifests: $(CONTROLLER_GEN) ## Generate Kubernetes manifests
	@echo "Generating Kubernetes manifests..."
	@$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	@echo "Manifests generated"

# =============================================================================
# Build Targets
# =============================================================================

## build: Build all services
.PHONY: build
build: generate manifests build-all ## Build all services

## build-all: Build all service binaries
.PHONY: build-all
build-all: $(foreach service,$(ALL_SERVICES),build-$(service))

## build-manager: Build the main operator manager
.PHONY: build-manager build-nephoran-operator
build-manager build-nephoran-operator: $(MANAGER_BIN)

$(MANAGER_BIN): $(LOCALBIN)
	@echo "Building nephoran-operator..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(GOFLAGS) $(GO_BUILD_FLAGS) -o $@ ./cmd/main.go

## build-porch-direct: Build porch-direct service
.PHONY: build-porch-direct
build-porch-direct: $(PORCH_DIRECT_BIN)

$(PORCH_DIRECT_BIN): $(LOCALBIN)
	@echo "Building porch-direct..."
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(GOFLAGS) $(GO_BUILD_FLAGS) -o $@ ./cmd/porch-direct

# =============================================================================
# Testing
# =============================================================================

## test: Run unit tests
.PHONY: test
test: $(ENVTEST) ## Run unit tests
	@echo "Running unit tests..."
	@KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		go test $(GO_TEST_FLAGS) ./...

## test-integration: Run integration tests
.PHONY: test-integration
test-integration: $(ENVTEST) $(GINKGO) ## Run integration tests
	@echo "Running integration tests..."
	@KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		$(GINKGO) -r --randomize-all --randomize-suites --fail-on-pending \
		--keep-going --cover --coverprofile=coverage.out \
		--race --trace --progress --tags integration ./test/...

## coverage: Generate test coverage report
.PHONY: coverage
coverage: test ## Generate coverage report
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# =============================================================================
# Code Quality and Security
# =============================================================================

## lint: Run code linters
.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run golangci-lint
	@echo "Running linters..."
	@$(GOLANGCI_LINT) run --verbose --timeout=10m

## security-scan: Run security scans
.PHONY: security-scan
security-scan: gosec govulncheck trivy-fs ## Run all security scans

## gosec: Run gosec security scanner
.PHONY: gosec
gosec: ## Run gosec security scanner
	@echo "Running gosec security scan..."
	@command -v gosec >/dev/null 2>&1 || go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@gosec -severity $(GOSEC_SEVERITY) -confidence medium -quiet ./...

## govulncheck: Run Go vulnerability check
.PHONY: govulncheck
govulncheck: ## Run govulncheck
	@echo "Running Go vulnerability check..."
	@command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck ./...

## trivy-fs: Run Trivy filesystem scan
.PHONY: trivy-fs
trivy-fs: $(TRIVY) ## Run Trivy filesystem scan
	@echo "Running Trivy filesystem scan..."
	@$(TRIVY) fs --severity $(TRIVY_SEVERITY) --no-progress .

## fmt: Format Go code
.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting Go code..."
	@go fmt ./...

## vet: Run go vet
.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

# =============================================================================
# Docker Build
# =============================================================================

## docker-build: Build Docker image for manager
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build --platform=$(PLATFORMS) \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMAGE_REGISTRY)/nephoran-operator:$(IMAGE_TAG) \
		-t $(IMAGE_REGISTRY)/nephoran-operator:latest .

# =============================================================================
# Kubernetes Deployment
# =============================================================================

## install: Install CRDs into the cluster
.PHONY: install
install: manifests $(KUSTOMIZE) ## Install CRDs
	@echo "Installing CRDs..."
	@$(KUSTOMIZE) build config/crd | kubectl apply -f -

## uninstall: Uninstall CRDs from the cluster
.PHONY: uninstall
uninstall: manifests $(KUSTOMIZE) ## Uninstall CRDs
	@echo "Uninstalling CRDs..."
	@$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found -f -

## deploy: Deploy operator to the cluster
.PHONY: deploy
deploy: manifests $(KUSTOMIZE) ## Deploy operator
	@echo "Deploying operator..."
	@cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE_REGISTRY)/nephoran-operator:$(IMAGE_TAG)
	@$(KUSTOMIZE) build config/default | kubectl apply -f -

## undeploy: Remove operator from the cluster
.PHONY: undeploy
undeploy: $(KUSTOMIZE) ## Remove operator from cluster
	@echo "Removing operator from cluster..."
	@$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found -f -

# =============================================================================
# MVP Scaling Operations
# =============================================================================

## mvp-scale-up: Scale up MVP deployment
.PHONY: mvp-scale-up
mvp-scale-up: ## Scale up MVP services
	@echo "Scaling up MVP deployment..."
	@kubectl scale deployment nephoran-operator --replicas=2 -n nephoran-system || echo "Deployment not found, continuing..."

## mvp-scale-down: Scale down MVP deployment
.PHONY: mvp-scale-down
mvp-scale-down: ## Scale down MVP services
	@echo "Scaling down MVP deployment..."
	@kubectl scale deployment nephoran-operator --replicas=1 -n nephoran-system || echo "Deployment not found, continuing..."

# =============================================================================
# Development and Debug
# =============================================================================

## run: Run operator locally
.PHONY: run
run: manifests generate fmt vet ## Run operator locally
	@echo "Running operator locally..."
	@go run ./cmd/main.go

## logs: View operator logs
.PHONY: logs
logs: ## View operator logs
	@kubectl logs -f deployment/nephoran-operator -n nephoran-system || echo "Deployment not found"

# =============================================================================
# Cleanup
# =============================================================================

## clean: Clean build artifacts
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(BUILD_DIR) $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@go clean -cache -testcache -modcache

## clean-tools: Clean installed tools
.PHONY: clean-tools
clean-tools: ## Clean installed tools
	@echo "Cleaning tools..."
	@rm -rf $(LOCALBIN)

## clean-all: Clean everything
.PHONY: clean-all
clean-all: clean clean-tools ## Clean everything

# =============================================================================
# Version and Release
# =============================================================================

## version: Display version information
.PHONY: version
version: ## Display version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Kubernetes Version: $(KUBERNETES_VERSION)"
	@echo "Controller Runtime: $(CONTROLLER_RUNTIME_VERSION)"

# Ensure directories exist
$(BIN_DIR) $(BUILD_DIR) $(DIST_DIR):
	@mkdir -p $@

.PHONY: all
all: setup build test lint ## Build everything