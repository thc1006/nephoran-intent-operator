# =============================================================================
# KUBERNETES OPERATOR MAKEFILE - 2025 BEST PRACTICES
# =============================================================================
# Modern Kubernetes operator development with:
# - Controller-runtime v0.19.1
# - Kubernetes v1.32.1 
# - Enhanced security scanning
# - Advanced testing patterns
# - RBAC validation
# - CRD lifecycle management
# =============================================================================

SHELL := /bin/bash
.DEFAULT_GOAL := help

# =============================================================================
# 2025 VERSION MATRIX
# =============================================================================
GO_VERSION ?= 1.24.6
KUBERNETES_VERSION ?= 1.32.1
CONTROLLER_RUNTIME_VERSION ?= v0.19.1
KUBEBUILDER_VERSION ?= 4.4.1
ENVTEST_K8S_VERSION ?= 1.32.1

# =============================================================================
# PROJECT CONFIGURATION
# =============================================================================
# Image URL to use all building/pushing image targets
IMG ?= nephoran-operator:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = $(ENVTEST_K8S_VERSION)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# =============================================================================
# BUILD OPTIMIZATION FOR 2025
# =============================================================================
export CGO_ENABLED := 0
export GOOS := linux
export GOARCH := amd64
export GOMAXPROCS := 4
export GOMEMLIMIT := 4GiB
export GOGC := 100

# Build flags for production
LDFLAGS := -s -w -X main.version=$(shell git describe --tags --always --dirty)
BUILD_FLAGS := -ldflags "$(LDFLAGS)" -trimpath

# Critical paths for the operator
BIN_DIR := bin
CONFIG_DIR := config
CRD_DIR := $(CONFIG_DIR)/crd/bases
RBAC_DIR := $(CONFIG_DIR)/rbac
WEBHOOK_DIR := $(CONFIG_DIR)/webhook
SAMPLES_DIR := $(CONFIG_DIR)/samples

.PHONY: help
help: ## Display this help.
	@echo "🎯 Nephoran Kubernetes Operator - 2025 Edition"
	@echo "==============================================="
	@awk 'BEGIN {FS = ":.*##"}; /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	@echo "📋 Generating Kubernetes manifests..."
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	@echo "🔄 Generating deepcopy code..."
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	@echo "🎨 Running go fmt..."
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	@echo "🔍 Running go vet..."
	go vet ./...

##@ Testing - 2025 Enhanced Patterns

.PHONY: test
test: manifests generate fmt vet envtest ## Run comprehensive tests with envtest.
	@echo "🧪 Running comprehensive tests with envtest..."
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
	USE_EXISTING_CLUSTER=false \
	go test -v -timeout=10m -coverprofile=coverage.out ./...
	@echo "✅ All tests passed!"

.PHONY: test-controllers
test-controllers: manifests generate envtest ## Run controller tests with enhanced envtest setup.
	@echo "🎮 Running controller tests with envtest..."
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
	USE_EXISTING_CLUSTER=false \
	KUBEBUILDER_CONTROLPLANE_START_TIMEOUT=60s \
	KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT=60s \
	go test -v -timeout=15m -p=1 ./controllers/... -coverprofile=controller-coverage.out
	@echo "✅ Controller tests completed!"

.PHONY: test-webhooks
test-webhooks: manifests generate envtest ## Test admission webhooks with proper envtest setup.
	@echo "🪝 Running webhook tests..."
	@if [[ -d "webhooks" ]]; then \
		KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		USE_EXISTING_CLUSTER=false \
		go test -v -timeout=10m -p=1 ./webhooks/... -coverprofile=webhook-coverage.out; \
	else \
		echo "ℹ️ No dedicated webhook directory found, checking controllers for webhook tests..."; \
		KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		USE_EXISTING_CLUSTER=false \
		go test -v -timeout=10m -p=1 ./controllers/... -run=".*Webhook.*" -coverprofile=webhook-coverage.out || echo "No webhook tests found"; \
	fi
	@echo "✅ Webhook tests completed!"

.PHONY: test-api
test-api: manifests generate envtest ## Test API validation logic.
	@echo "🔌 Running API validation tests..."
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
	USE_EXISTING_CLUSTER=false \
	go test -v -timeout=10m ./api/... -coverprofile=api-coverage.out
	@echo "✅ API tests completed!"

.PHONY: test-integration
test-integration: ## Run integration tests (requires actual cluster).
	@echo "🔗 Running integration tests..."
	@if [[ -z "$$KUBECONFIG" ]]; then \
		echo "❌ KUBECONFIG not set. Integration tests require a real cluster."; \
		exit 1; \
	fi
	USE_EXISTING_CLUSTER=true go test -v -timeout=20m -tags=integration ./test/integration/...
	@echo "✅ Integration tests completed!"

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests (requires cluster with operator deployed).
	@echo "🌐 Running end-to-end tests..."
	@if [[ -z "$$KUBECONFIG" ]]; then \
		echo "❌ KUBECONFIG not set. E2E tests require a real cluster."; \
		exit 1; \
	fi
	go test -v -timeout=30m -tags=e2e ./test/e2e/...
	@echo "✅ E2E tests completed!"

.PHONY: coverage
coverage: test ## Generate and view coverage report.
	@echo "📊 Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "📈 Coverage report generated: coverage.html"

##@ Kubernetes Validation - 2025 Standards

.PHONY: validate-crds
validate-crds: manifests ## Validate generated CRDs using kubeval.
	@echo "🔍 Validating generated CRDs..."
	@if ! command -v kubeval &> /dev/null; then \
		echo "❌ kubeval not found. Installing..."; \
		curl -L https://github.com/instrumenta/kubeval/releases/latest/download/kubeval-linux-amd64.tar.gz | tar xz; \
		sudo mv kubeval /usr/local/bin/; \
	fi
	@for crd in $(CRD_DIR)/*.yaml; do \
		echo "  Validating $$crd..."; \
		kubeval --strict "$$crd" || exit 1; \
	done
	@echo "✅ All CRDs are valid!"

.PHONY: validate-rbac
validate-rbac: generate ## Validate RBAC configurations for security compliance.
	@echo "🔐 Validating RBAC configurations..."
	@if ! command -v kubeval &> /dev/null; then \
		echo "❌ kubeval not found. Installing..."; \
		curl -L https://github.com/instrumenta/kubeval/releases/latest/download/kubeval-linux-amd64.tar.gz | tar xz; \
		sudo mv kubeval /usr/local/bin/; \
	fi
	@for rbac in $(RBAC_DIR)/*.yaml; do \
		echo "  Validating $$rbac..."; \
		kubeval --strict "$$rbac" || exit 1; \
	done
	@echo "🔍 Checking for dangerous RBAC permissions..."
	@if grep -r "apiGroups.*\*" $(RBAC_DIR)/ 2>/dev/null; then \
		echo "⚠️ Warning: Found wildcard API groups in RBAC"; \
	fi
	@if grep -r "resources.*\*" $(RBAC_DIR)/ 2>/dev/null; then \
		echo "⚠️ Warning: Found wildcard resources in RBAC"; \
	fi
	@if grep -r "verbs.*\*" $(RBAC_DIR)/ 2>/dev/null; then \
		echo "❌ Error: Found wildcard verbs in RBAC - this is dangerous!"; \
		exit 1; \
	fi
	@echo "✅ RBAC configurations are secure!"

.PHONY: validate-webhooks
validate-webhooks: manifests ## Validate webhook configurations.
	@echo "🪝 Validating webhook configurations..."
	@if [[ -d "$(WEBHOOK_DIR)" ]]; then \
		if ! command -v kubeval &> /dev/null; then \
			echo "❌ kubeval not found. Installing..."; \
			curl -L https://github.com/instrumenta/kubeval/releases/latest/download/kubeval-linux-amd64.tar.gz | tar xz; \
			sudo mv kubeval /usr/local/bin/; \
		fi; \
		for webhook in $(WEBHOOK_DIR)/*.yaml; do \
			echo "  Validating $$webhook..."; \
			kubeval --strict "$$webhook" || exit 1; \
		done; \
		echo "✅ Webhook configurations are valid!"; \
	else \
		echo "ℹ️ No webhook configurations found"; \
	fi

.PHONY: validate-samples
validate-samples: ## Validate sample resource configurations.
	@echo "📋 Validating sample resources..."
	@if [[ -d "$(SAMPLES_DIR)" ]]; then \
		if ! command -v kubeval &> /dev/null; then \
			echo "❌ kubeval not found. Installing..."; \
			curl -L https://github.com/instrumenta/kubeval/releases/latest/download/kubeval-linux-amd64.tar.gz | tar xz; \
			sudo mv kubeval /usr/local/bin/; \
		fi; \
		for sample in $(SAMPLES_DIR)/*.yaml; do \
			echo "  Validating $$sample..."; \
			kubeval --strict "$$sample" || exit 1; \
		done; \
		echo "✅ Sample resources are valid!"; \
	else \
		echo "ℹ️ No sample resources found"; \
	fi

.PHONY: validate-kustomize
validate-kustomize: ## Validate kustomize configurations.
	@echo "📋 Validating kustomize configurations..."
	@if ! command -v kustomize &> /dev/null; then \
		echo "❌ kustomize not found. Installing..."; \
		curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash; \
		sudo mv kustomize /usr/local/bin/; \
	fi
	@if ! command -v kubeval &> /dev/null; then \
		echo "❌ kubeval not found. Installing..."; \
		curl -L https://github.com/instrumenta/kubeval/releases/latest/download/kubeval-linux-amd64.tar.gz | tar xz; \
		sudo mv kubeval /usr/local/bin/; \
	fi
	@echo "  Testing default kustomization..."
	@kustomize build config/default > /tmp/default-manifest.yaml
	@kubeval --strict /tmp/default-manifest.yaml
	@echo "✅ Kustomize configurations are valid!"

.PHONY: validate-all
validate-all: validate-crds validate-rbac validate-webhooks validate-samples validate-kustomize ## Run all validation checks.
	@echo "🎯 All Kubernetes configurations validated successfully!"

##@ Security Scanning - 2025 Standards

.PHONY: security-scan
security-scan: security-scan-go security-scan-container security-scan-k8s ## Run comprehensive security scanning.

.PHONY: security-scan-go
security-scan-go: ## Scan Go code for vulnerabilities.
	@echo "🔍 Scanning Go modules for vulnerabilities..."
	@if ! command -v govulncheck &> /dev/null; then \
		echo "📦 Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	govulncheck ./...
	@echo "✅ Go vulnerability scan completed!"

.PHONY: security-scan-container
security-scan-container: docker-build ## Scan container image for vulnerabilities.
	@echo "🐳 Scanning container image for vulnerabilities..."
	@if ! command -v trivy &> /dev/null; then \
		echo "📦 Installing Trivy..."; \
		curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin; \
	fi
	trivy image --severity HIGH,CRITICAL --ignore-unfixed $(IMG)
	@echo "✅ Container security scan completed!"

.PHONY: security-scan-k8s
security-scan-k8s: ## Scan Kubernetes configurations for security issues.
	@echo "☸️ Scanning Kubernetes configurations for security issues..."
	@if ! command -v trivy &> /dev/null; then \
		echo "📦 Installing Trivy..."; \
		curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin; \
	fi
	trivy config config/ --severity HIGH,CRITICAL
	@echo "✅ Kubernetes security scan completed!"

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	@echo "🔨 Building manager binary..."
	@mkdir -p $(BIN_DIR)
	go build $(BUILD_FLAGS) -o $(BIN_DIR)/manager cmd/main.go
	@echo "✅ Manager binary built: $(BIN_DIR)/manager"

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	@echo "🚀 Starting controller manager..."
	go run ./cmd/main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	@echo "🐳 Building container image: $(IMG)"
	$(CONTAINER_TOOL) build -t $(IMG) .
	@echo "✅ Container image built: $(IMG)"

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	@echo "📤 Pushing container image: $(IMG)"
	$(CONTAINER_TOOL) push $(IMG)

.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	@echo "🔀 Building multi-platform container image: $(IMG)"
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag $(IMG) .
	- $(CONTAINER_TOOL) buildx rm project-v3-builder

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	@echo "☸️ Installing CRDs into cluster..."
	$(KUSTOMIZE) build config/crd | kubectl apply -f -
	@echo "✅ CRDs installed successfully!"

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	@echo "🗑️ Uninstalling CRDs from cluster..."
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	@echo "🚀 Deploying controller to cluster..."
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -
	@echo "✅ Controller deployed successfully!"

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	@echo "🗑️ Undeploying controller from cluster..."
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
KUSTOMIZE_VERSION ?= v5.5.0
CONTROLLER_TOOLS_VERSION ?= v0.19.0
GOLANGCI_LINT_VERSION ?= v1.63.3

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "🔄 Updating kustomize to $(KUSTOMIZE_VERSION)"; \
		rm -f $(LOCALBIN)/kustomize; \
	fi
	@test -s $(LOCALBIN)/kustomize || \
	{ curl -Ss "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	@test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	@test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	@test -s $(LOCALBIN)/golangci-lint && $(LOCALBIN)/golangci-lint --version | grep -q $(GOLANGCI_LINT_VERSION) || \
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter & yamllint
	@echo "🔍 Running golangci-lint..."
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	@echo "🔧 Running golangci-lint with fixes..."
	$(GOLANGCI_LINT) run --fix

##@ Cleanup

.PHONY: clean
clean: ## Clean build artifacts and temporary files.
	@echo "🧹 Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	rm -f *-coverage.out
	rm -f *.sarif
	@echo "✅ Clean completed!"

.PHONY: clean-all
clean-all: clean ## Clean everything including downloaded tools.
	@echo "🧹 Cleaning everything including tools..."
	rm -rf $(LOCALBIN)
	@echo "✅ Deep clean completed!"

##@ CI/CD Integration

.PHONY: ci-test
ci-test: fmt vet lint test validate-all ## Run all CI tests and validations.
	@echo "🎯 CI test suite completed successfully!"

.PHONY: ci-build
ci-build: build docker-build ## Build all CI artifacts.
	@echo "📦 CI build completed successfully!"

.PHONY: ci-security
ci-security: security-scan ## Run CI security checks.
	@echo "🔒 CI security checks completed successfully!"

.PHONY: ci-full
ci-full: ci-test ci-build ci-security ## Run complete CI pipeline.
	@echo "🚀 Full CI pipeline completed successfully!"

##@ Release

.PHONY: release-manifest
release-manifest: manifests kustomize ## Generate release manifest.
	@echo "📄 Generating release manifest..."
	@mkdir -p dist
	$(KUSTOMIZE) build config/default > dist/nephoran-operator.yaml
	@echo "✅ Release manifest generated: dist/nephoran-operator.yaml"

.PHONY: release-crds
release-crds: manifests kustomize ## Generate CRD-only manifest for release.
	@echo "📄 Generating CRD manifest..."
	@mkdir -p dist
	$(KUSTOMIZE) build config/crd > dist/nephoran-crds.yaml
	@echo "✅ CRD manifest generated: dist/nephoran-crds.yaml"

.PHONY: release-rbac
release-rbac: manifests kustomize ## Generate RBAC-only manifest for release.
	@echo "📄 Generating RBAC manifest..."
	@mkdir -p dist
	$(KUSTOMIZE) build config/rbac > dist/nephoran-rbac.yaml
	@echo "✅ RBAC manifest generated: dist/nephoran-rbac.yaml"

.PHONY: release-all
release-all: release-manifest release-crds release-rbac ## Generate all release artifacts.
	@echo "📦 All release artifacts generated in dist/ directory"

##@ Development Helpers

.PHONY: debug
debug: ## Show debug information about the build environment.
	@echo "🔍 Build Environment Debug Information"
	@echo "====================================="
	@echo "Go Version: $(shell go version)"
	@echo "Kubernetes Version: $(KUBERNETES_VERSION)"
	@echo "Controller-Runtime Version: $(CONTROLLER_RUNTIME_VERSION)"
	@echo "Kubebuilder Version: $(KUBEBUILDER_VERSION)"
	@echo "Image: $(IMG)"
	@echo "GOBIN: $(GOBIN)"
	@echo "LOCALBIN: $(LOCALBIN)"
	@echo "GOPATH: $(shell go env GOPATH)"
	@echo "GOCACHE: $(shell go env GOCACHE)"
	@echo "GOMODCACHE: $(shell go env GOMODCACHE)"
	@echo ""
	@echo "Installed Tools:"
	@echo "---------------"
	@ls -la $(LOCALBIN)/ 2>/dev/null || echo "No tools installed"

.PHONY: dev-setup
dev-setup: controller-gen kustomize envtest golangci-lint ## Setup development environment.
	@echo "🛠️ Development environment setup completed!"
	@echo "📋 Available tools:"
	@echo "  - controller-gen: $(CONTROLLER_GEN)"
	@echo "  - kustomize: $(KUSTOMIZE)"
	@echo "  - envtest: $(ENVTEST)"
	@echo "  - golangci-lint: $(GOLANGCI_LINT)"

.PHONY: status
status: ## Show current development status.
	@echo "📊 Development Status"
	@echo "===================="
	@echo "Git Branch: $(shell git branch --show-current 2>/dev/null || echo 'unknown')"
	@echo "Git Status: $(shell git status --porcelain | wc -l) modified files"
	@echo "Go Modules: $(shell go list -m all | wc -l) modules"
	@echo "Test Coverage: $(shell test -f coverage.out && go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' || echo 'not available')"
	@echo ""
	@echo "Project Structure:"
	@echo "-----------------"
	@find . -name "*.go" -not -path "./vendor/*" | head -10
	@echo "... and $(shell find . -name "*.go" -not -path "./vendor/*" | wc -l) Go files total"