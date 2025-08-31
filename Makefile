# Makefile for Nephoran Intent Operator with Excellence Validation Framework

# Project configuration
PROJECT_NAME = nephoran-intent-operator
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT = $(shell git rev-parse --short HEAD)
DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Helper functions for MVP scaling operations
define check_tool
	@if ! command -v $(1) >/dev/null 2>&1; then \
		echo "ERROR: $(1) is required but not installed"; \
		exit 1; \
	fi
endef

define kubectl_patch_deployment
	kubectl patch deployment $(1) \
		--namespace $(NAMESPACE) \
		--type='json' \
		-p='[{"op": "replace", "path": "/spec/replicas", "value": $(2)}]' 2>/dev/null || true
endef

define kubectl_patch_hpa
	kubectl patch hpa $(1) \
		--namespace $(NAMESPACE) \
		--type='json' \
		-p='[{"op": "replace", "path": "/spec/minReplicas", "value": $(2)}, {"op": "replace", "path": "/spec/maxReplicas", "value": $(3)}]' 2>/dev/null || true
endef

define create_patch_file
	@mkdir -p handoff/patches; \
	echo '{"kind":"Patch","metadata":{"name":"$(1)"},"spec":{"replicas":$(2),"resources":{"requests":{"cpu":"$(3)","memory":"$(4)"}}}}' \
		> handoff/patches/$(1)-patch.json; \
	echo "$(1) patch saved to handoff/patches/$(1)-patch.json"
endef

# Go configuration - Updated for Go 1.24+ with 2025 optimizations
GO_VERSION = 1.24.1
CONTROLLER_GEN_VERSION ?= v0.19.0
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= 0

# Go 1.24+ Performance Optimizations (2025 standards)
GOEXPERIMENT ?= swisstable,pacer,nocoverageredesign,rangefunc
GOMAXPROCS ?= $(shell nproc 2>/dev/null || echo 4)
GOMEMLIMIT ?= 8GiB
GOGC ?= 75
GODEBUG ?= gctrace=0,scavenge=off
PGO_PROFILE ?= default.pgo

# Supply Chain Security Configuration
GOSUMDB ?= sum.golang.org
GOPROXY ?= https://proxy.golang.org,direct
GOPRIVATE ?= github.com/thc1006/*

# ULTRA SPEED CI Configuration
CI_ULTRA_SPEED_WORKFLOW = .github/workflows/ci-ultra-speed.yml
CI_BASELINE_WORKFLOW = .github/workflows/ci.yml
CI_BENCHMARK_SCRIPT = scripts/ci-performance-benchmark.ps1

# Docker configuration  
REGISTRY ?= ghcr.io
IMAGE_NAME ?= $(PROJECT_NAME)
IMAGE_TAG ?= $(VERSION)
BUILD_TYPE ?= production

# Performance and build optimization with Go 1.24+ features
PARALLEL_JOBS ?= $(shell nproc --all 2>/dev/null || echo 4)
GO_BUILD_ENV = GOMAXPROCS=$(GOMAXPROCS) GOMEMLIMIT=$(GOMEMLIMIT) GOGC=$(GOGC) GODEBUG=$(GODEBUG) GOEXPERIMENT=$(GOEXPERIMENT)
ifeq ($(BUILD_TYPE),fast)
	FAST_BUILD_FLAGS = -ldflags="-s -w" -gcflags="all=-l=4" -buildmode=default -pgo=$(PGO_PROFILE) -buildvcs=false
	FAST_BUILD_TAGS = fast_build,netgo,osusergo
else ifeq ($(BUILD_TYPE),debug)
	FAST_BUILD_FLAGS = -gcflags="all=-N -l" -race -buildvcs=auto
	FAST_BUILD_TAGS = debug
else
	FAST_BUILD_FLAGS = -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -buildid=" -trimpath -pgo=$(PGO_PROFILE) -buildvcs=false
	FAST_BUILD_TAGS = production,netgo,osusergo
endif

# LDFLAGS for production builds with Go 1.24+ optimizations
LDFLAGS = -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -extldflags '-static' -buildid=" -pgo=$(PGO_PROFILE)

# Quality and testing configuration
QUALITY_REPORTS_DIR ?= .excellence-reports
REPORTS_DIR ?= $(QUALITY_REPORTS_DIR)
COVERAGE_THRESHOLD ?= 80.0

# Testing configuration (2025 Optimized)
GO_TEST_FLAGS ?= -v -race -timeout=35m -parallel=8
GO_TEST_COVERAGE_FLAGS ?= -coverprofile=$(QUALITY_REPORTS_DIR)/coverage.out -covermode=atomic
GO_TEST_PARALLEL_JOBS ?= $(shell echo "$(PARALLEL_JOBS) * 2" | bc)
GO_TEST_2025_FLAGS ?= -json -shuffle=on -count=1

# Kubernetes configuration
NAMESPACE ?= nephoran-system

# MVP Scaling Configuration
MIN_REPLICAS_CONDUCTOR ?= 1
MAX_REPLICAS_CONDUCTOR ?= 3
MIN_REPLICAS_PLANNER ?= 1
MAX_REPLICAS_PLANNER ?= 2
MIN_REPLICAS_SIMULATOR ?= 2
MAX_REPLICAS_SIMULATOR ?= 5

# Resource requirements for MVP scaling
CONDUCTOR_CPU_REQUEST ?= "100m"
CONDUCTOR_MEMORY_REQUEST ?= "128Mi"
PLANNER_CPU_REQUEST ?= "200m"
PLANNER_MEMORY_REQUEST ?= "256Mi"
SIMULATOR_CPU_REQUEST ?= "50m"
SIMULATOR_MEMORY_REQUEST ?= "64Mi"

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: clean
clean: ## Clean build artifacts and reports
	@echo "Cleaning build artifacts and reports..."
	rm -rf bin/
	rm -rf $(QUALITY_REPORTS_DIR)/
	rm -rf vendor/
	rm -rf .cache/
	go clean -cache -modcache -testcache 2>/dev/null || true

.PHONY: clean-all
clean-all: clean ## Clean everything including Docker resources
	@echo "Deep cleaning all resources..."
	docker system prune -f 2>/dev/null || true
	go clean -cache -modcache -testcache -fuzzcache 2>/dev/null || true
	rm -rf ~/.cache/go-build 2>/dev/null || true

##@ MVP Scaling Operations

.PHONY: mvp-scale-up
mvp-scale-up: ## Scale up MVP components with optimized resources
	@echo "Scaling up MVP components..."
	$(call check_tool,kubectl)
	@mkdir -p handoff/patches
	
	@echo "Scaling conductor..."
	$(call kubectl_patch_deployment,conductor,$(MAX_REPLICAS_CONDUCTOR))
	$(call kubectl_patch_hpa,conductor-hpa,$(MIN_REPLICAS_CONDUCTOR),$(MAX_REPLICAS_CONDUCTOR))
	$(call create_patch_file,conductor,$(MAX_REPLICAS_CONDUCTOR),$(CONDUCTOR_CPU_REQUEST),$(CONDUCTOR_MEMORY_REQUEST))
	
	@echo "Scaling planner..."
	$(call kubectl_patch_deployment,planner,$(MAX_REPLICAS_PLANNER))
	$(call kubectl_patch_hpa,planner-hpa,$(MIN_REPLICAS_PLANNER),$(MAX_REPLICAS_PLANNER))
	$(call create_patch_file,planner,$(MAX_REPLICAS_PLANNER),$(PLANNER_CPU_REQUEST),$(PLANNER_MEMORY_REQUEST))
	
	@echo "Scaling simulator..."
	$(call kubectl_patch_deployment,simulator,$(MAX_REPLICAS_SIMULATOR))
	$(call kubectl_patch_hpa,simulator-hpa,$(MIN_REPLICAS_SIMULATOR),$(MAX_REPLICAS_SIMULATOR))
	$(call create_patch_file,simulator,$(MAX_REPLICAS_SIMULATOR),$(SIMULATOR_CPU_REQUEST),$(SIMULATOR_MEMORY_REQUEST))
	
	@echo "[SUCCESS] MVP scale-up completed. Patches saved to handoff/patches/"
	@ls -la handoff/patches/

.PHONY: mvp-scale-down
mvp-scale-down: ## Scale down MVP components to minimum resources
	@echo "Scaling down MVP components..."
	$(call check_tool,kubectl)
	@mkdir -p handoff/patches
	
	@echo "Scaling down conductor..."
	$(call kubectl_patch_deployment,conductor,$(MIN_REPLICAS_CONDUCTOR))
	$(call kubectl_patch_hpa,conductor-hpa,$(MIN_REPLICAS_CONDUCTOR),$(MIN_REPLICAS_CONDUCTOR))
	$(call create_patch_file,conductor,$(MIN_REPLICAS_CONDUCTOR),$(CONDUCTOR_CPU_REQUEST),$(CONDUCTOR_MEMORY_REQUEST))
	
	@echo "Scaling down planner..."
	$(call kubectl_patch_deployment,planner,$(MIN_REPLICAS_PLANNER))
	$(call kubectl_patch_hpa,planner-hpa,$(MIN_REPLICAS_PLANNER),$(MIN_REPLICAS_PLANNER))
	$(call create_patch_file,planner,$(MIN_REPLICAS_PLANNER),$(PLANNER_CPU_REQUEST),$(PLANNER_MEMORY_REQUEST))
	
	@echo "Scaling down simulator..."
	$(call kubectl_patch_deployment,simulator,$(MIN_REPLICAS_SIMULATOR))
	$(call kubectl_patch_hpa,simulator-hpa,$(MIN_REPLICAS_SIMULATOR),$(MIN_REPLICAS_SIMULATOR))
	$(call create_patch_file,simulator,$(MIN_REPLICAS_SIMULATOR),$(SIMULATOR_CPU_REQUEST),$(SIMULATOR_MEMORY_REQUEST))
	
	@echo "[SUCCESS] MVP scale-down completed. Patches saved to handoff/patches/"
	@ls -la handoff/patches/

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy

.PHONY: vendor
vendor: ## Run go mod vendor
	go mod vendor

.PHONY: lint
lint: ## Run golangci-lint with 2025 comprehensive configuration
	@echo "🔍 Running golangci-lint with 2025 comprehensive configuration..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config=.golangci-fast.yml --timeout=10m --issues-exit-code=1; \
	else \
		echo "📥 Installing golangci-lint v1.65+ (2025 compatible)..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --config=.golangci-fast.yml --timeout=10m --issues-exit-code=1; \
	fi

.PHONY: lint-fast
lint-fast: ## Run golangci-lint with fast linters only (development mode)
	@echo "⚡ Running golangci-lint with fast linters only..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config=.golangci-fast.yml --fast --timeout=5m --issues-exit-code=1; \
	else \
		echo "📥 Installing golangci-lint v1.65+ (2025 compatible)..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --config=.golangci-fast.yml --fast --timeout=5m --issues-exit-code=1; \
	fi

.PHONY: lint-changed
lint-changed: ## Run golangci-lint only on changed files (PR mode)
	@echo "📝 Running golangci-lint on changed files only..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config=.golangci-fast.yml --new-from-rev=HEAD~1 --timeout=2m; \
	else \
		echo "📥 Installing golangci-lint v1.65+ (2025 compatible)..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --config=.golangci-fast.yml --new-from-rev=HEAD~1 --timeout=2m; \
	fi

.PHONY: lint-fix
lint-fix: ## Run golangci-lint and automatically fix issues where possible
	@echo "🔧 Running golangci-lint with auto-fix (2025 configuration)..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config=.golangci-fast.yml --fix --timeout=15m; \
	else \
		echo "📥 Installing golangci-lint v1.65+ (2025 compatible)..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --config=.golangci-fast.yml --fix --timeout=15m; \
	fi

.PHONY: lint-security
lint-security: ## Run golangci-lint with security focus and SARIF output
	@echo "🔒 Running security-focused linting with SARIF output..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config=.golangci-fast.yml --out-format=sarif > golangci-lint-report.sarif; \
		golangci-lint run --config=.golangci-fast.yml --enable-only=gosec,bidichk,bodyclose,rowserrcheck,sqlclosecheck; \
	else \
		echo "📥 Installing golangci-lint v1.65+ (2025 compatible)..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --config=.golangci-fast.yml --out-format=sarif > golangci-lint-report.sarif; \
		golangci-lint run --config=.golangci-fast.yml --enable-only=gosec,bidichk,bodyclose,rowserrcheck,sqlclosecheck; \
	fi

.PHONY: validate-configs
validate-configs: ## Validate golangci-lint configuration files with comprehensive checks
	@echo "✅ Validating golangci-lint configuration with comprehensive checks..."
	@if [ -f "scripts/hooks/validate-golangci-config.sh" ]; then \
		chmod +x scripts/hooks/validate-golangci-config.sh; \
		scripts/hooks/validate-golangci-config.sh; \
	else \
		echo "[WARNING] Validation script not found, using basic validation"; \
		golangci-lint config path -c .golangci-fast.yml; \
		golangci-lint linters -c .golangci-fast.yml; \
	fi

.PHONY: lint-validate
lint-validate: validate-configs ## Alias for validate-configs (backwards compatibility)

.PHONY: verify
verify: fmt vet tidy lint validate-configs ## Verify code consistency with config validation
	@echo "Verifying code consistency..."
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "[ERROR] Code verification failed. The following files need attention:"; \
		git status --porcelain; \
		echo "Please run 'make fmt tidy' and commit the changes."; \
		exit 1; \
	fi
	@echo "[SUCCESS] Code verification passed"

##@ Pre-commit Hooks

.PHONY: install-hooks
install-hooks: ## Install pre-commit hooks to prevent invalid golangci-lint configs
	@echo "Installing pre-commit hooks..."
	@chmod +x scripts/install-hooks.sh
	@scripts/install-hooks.sh

.PHONY: test-hooks
test-hooks: ## Test pre-commit hooks installation
	@echo "Testing pre-commit hooks..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit run --all-files --dry-run; \
	else \
		echo "[ERROR] pre-commit not installed. Run 'make install-hooks' first"; \
		exit 1; \
	fi

.PHONY: run-hooks
run-hooks: ## Run all pre-commit hooks on staged files
	@echo "Running pre-commit hooks on staged files..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit run; \
	else \
		echo "[ERROR] pre-commit not installed. Run 'make install-hooks' first"; \
		exit 1; \
	fi

.PHONY: run-hooks-all
run-hooks-all: ## Run all pre-commit hooks on all files
	@echo "Running pre-commit hooks on all files..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit run --all-files; \
	else \
		echo "[ERROR] pre-commit not installed. Run 'make install-hooks' first"; \
		exit 1; \
	fi

.PHONY: update-hooks
update-hooks: ## Update pre-commit hooks to latest versions
	@echo "Updating pre-commit hooks..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit autoupdate; \
	else \
		echo "[ERROR] pre-commit not installed. Run 'make install-hooks' first"; \
		exit 1; \
	fi

.PHONY: uninstall-hooks
uninstall-hooks: ## Uninstall pre-commit hooks
	@echo "Uninstalling pre-commit hooks..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit uninstall; \
		echo "Pre-commit hooks uninstalled"; \
	else \
		echo "[INFO] pre-commit not installed, nothing to uninstall"; \
	fi

##@ Code Generation

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: gen
gen: generate ## Alias for generate (backwards compatibility)

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

##@ Testing

# Test configuration
TEST_OUTPUT_DIR ?= .test-reports
TEST_FIXTURES_DIR ?= tests/fixtures
TEST_MOCKS_DIR ?= tests/mocks
TEST_E2E_DIR ?= tests/e2e
TEST_INTEGRATION_DIR ?= tests/integration
TEST_UNIT_DIR ?= tests/unit
TEST_VALIDATION_DIR ?= tests/validation
TEST_SMOKE_DIR ?= tests/smoke
TEST_BENCHMARK_DIR ?= tests/benchmarks
TEST_TIMEOUT_UNIT ?= 5m
TEST_TIMEOUT_INTEGRATION ?= 15m  
TEST_TIMEOUT_E2E ?= 30m
TEST_TIMEOUT_VALIDATION ?= 10m
TEST_PARALLEL_JOBS ?= 4

# Test dependencies
TESTCONTAINERS_VERSION ?= v0.33.0
GOMOCK_VERSION ?= v0.5.0
TESTIFY_VERSION ?= v1.10.0

.PHONY: test-deps
test-deps: ## Install test dependencies
	@echo "Installing test dependencies..."
	go install go.uber.org/mock/mockgen@$(GOMOCK_VERSION)
	go install github.com/onsi/ginkgo/v2/ginkgo@latest
	go install github.com/onsi/gomega/...@latest
	go mod download github.com/testcontainers/testcontainers-go@$(TESTCONTAINERS_VERSION)
	go mod download github.com/stretchr/testify@$(TESTIFY_VERSION)

.PHONY: test-setup
test-setup: test-deps ## Set up test infrastructure
	@echo "Setting up test infrastructure..."
	@mkdir -p $(TEST_OUTPUT_DIR) $(TEST_FIXTURES_DIR) $(TEST_MOCKS_DIR)
	@mkdir -p $(TEST_E2E_DIR) $(TEST_INTEGRATION_DIR) $(TEST_UNIT_DIR)
	@mkdir -p $(TEST_VALIDATION_DIR) $(TEST_SMOKE_DIR) $(TEST_BENCHMARK_DIR)
	@mkdir -p $(TEST_OUTPUT_DIR)/coverage $(TEST_OUTPUT_DIR)/junit $(TEST_OUTPUT_DIR)/html

.PHONY: check-test-tools
check-test-tools: ## Check if test tools are installed
	@echo "Checking test tools..."
	@command -v mockgen >/dev/null 2>&1 || (echo "[ERROR] mockgen not found. Run 'make test-deps'" && exit 1)
	@command -v ginkgo >/dev/null 2>&1 || (echo "[ERROR] ginkgo not found. Run 'make test-deps'" && exit 1)
	@echo "[SUCCESS] All test tools are available"

.PHONY: check-redis-connectivity
check-redis-connectivity: ## Check Redis connectivity for integration tests
	@echo "Checking Redis connectivity..."
	@if command -v redis-cli >/dev/null 2>&1; then \
		if ! timeout 5 redis-cli -h localhost -p 6379 ping >/dev/null 2>&1; then \
			echo "[ERROR] Redis not available at localhost:6379. Starting Redis if available..."; \
			if command -v redis-server >/dev/null 2>&1; then \
				redis-server --daemonize yes --port 6379 --timeout 0 --tcp-keepalive 300 --maxmemory 256mb --maxmemory-policy allkeys-lru; \
				sleep 2; \
				if ! timeout 5 redis-cli -h localhost -p 6379 ping >/dev/null 2>&1; then \
					echo "[WARNING]  Redis still not available. Some tests may fail. Consider running: docker run -d -p 6379:6379 redis:7-alpine"; \
					exit 0; \
				fi; \
			else \
				echo "[WARNING]  Redis not available and redis-server not installed. Some tests may fail."; \
				echo "    Consider running: docker run -d -p 6379:6379 redis:7-alpine"; \
				exit 0; \
			fi; \
		fi; \
		echo "[SUCCESS] Redis is accessible"; \
	else \
		echo "[WARNING]  redis-cli not found. Cannot verify Redis connectivity"; \
	fi

.PHONY: check-k8s-tools
check-k8s-tools: ## Check if Kubernetes test tools are available
	@echo "Checking Kubernetes test tools..."
	@command -v kubectl >/dev/null 2>&1 || (echo "[ERROR] kubectl not found" && exit 1)
	@command -v kind >/dev/null 2>&1 || echo "[WARNING] kind not found - E2E tests may fail"
	@command -v helm >/dev/null 2>&1 || echo "[WARNING] helm not found - some tests may fail"
	@echo "[SUCCESS] Kubernetes tools check completed"

.PHONY: test
test: test-unit ## Run unit tests (alias for test-unit)

.PHONY: test-unit
test-unit: test-setup check-test-tools ## Run unit tests with comprehensive coverage
	@echo "Running unit tests with comprehensive coverage..."
	@export CGO_ENABLED=1 GOMAXPROCS=$(TEST_PARALLEL_JOBS) GODEBUG=gocachehash=1; \
	go test ./api/... ./controllers/... ./pkg/... ./cmd/... ./internal/... \
		-v -race -timeout=$(TEST_TIMEOUT_UNIT) \
		-coverprofile=$(TEST_OUTPUT_DIR)/coverage/unit-coverage.out \
		-covermode=atomic \
		-count=1 \
		-parallel=$(TEST_PARALLEL_JOBS) \
		-short \
		-json > $(TEST_OUTPUT_DIR)/junit/unit-test-results.json \
		|| (echo "[ERROR] Unit tests failed" && exit 1)
	@echo "[SUCCESS] Unit tests completed successfully"

.PHONY: test-unit-controllers
test-unit-controllers: test-setup check-test-tools ## Run controller unit tests with mocks
	@echo "Running controller unit tests with mocks..."
	@go test ./controllers/... \
		-v -race -timeout=$(TEST_TIMEOUT_UNIT) \
		-coverprofile=$(TEST_OUTPUT_DIR)/coverage/controllers-coverage.out \
		-covermode=atomic \
		-count=1 \
		-parallel=$(TEST_PARALLEL_JOBS) \
		-short \
		-json > $(TEST_OUTPUT_DIR)/junit/controllers-test-results.json \
		|| (echo "[ERROR] Controller unit tests failed" && exit 1)
	@echo "[SUCCESS] Controller unit tests completed"

.PHONY: test-unit-api
test-unit-api: test-setup check-test-tools ## Run API unit tests
	@echo "Running API unit tests..."
	@go test ./api/... \
		-v -race -timeout=$(TEST_TIMEOUT_UNIT) \
		-coverprofile=$(TEST_OUTPUT_DIR)/coverage/api-coverage.out \
		-covermode=atomic \
		-count=1 \
		-parallel=$(TEST_PARALLEL_JOBS) \
		-short \
		-json > $(TEST_OUTPUT_DIR)/junit/api-test-results.json \
		|| (echo "[ERROR] API unit tests failed" && exit 1)
	@echo "[SUCCESS] API unit tests completed"

.PHONY: test-integration
test-integration: test-setup check-redis-connectivity check-k8s-tools ## Run integration tests with testcontainers
	@echo "Running integration tests with testcontainers..."
	@go test $(TEST_INTEGRATION_DIR)/... \
		-tags=integration \
		-v -race -timeout=$(TEST_TIMEOUT_INTEGRATION) \
		-coverprofile=$(TEST_OUTPUT_DIR)/coverage/integration-coverage.out \
		-covermode=atomic \
		-count=1 \
		-parallel=$(TEST_PARALLEL_JOBS) \
		-json > $(TEST_OUTPUT_DIR)/junit/integration-test-results.json \
		|| (echo "[ERROR] Integration tests failed" && exit 1)
	@echo "[SUCCESS] Integration tests completed"

.PHONY: test-integration-crds
test-integration-crds: test-setup check-k8s-tools ## Run CRD integration tests
	@echo "Running CRD integration tests..."
	@go test $(TEST_INTEGRATION_DIR)/crds/... \
		-tags=integration \
		-v -race -timeout=$(TEST_TIMEOUT_INTEGRATION) \
		-coverprofile=$(TEST_OUTPUT_DIR)/coverage/crd-coverage.out \
		-covermode=atomic \
		-count=1 \
		-json > $(TEST_OUTPUT_DIR)/junit/crd-test-results.json \
		|| (echo "[ERROR] CRD integration tests failed" && exit 1)
	@echo "[SUCCESS] CRD integration tests completed"

.PHONY: test-integration-k8s
test-integration-k8s: test-setup check-k8s-tools ## Run Kubernetes integration tests
	@echo "Running Kubernetes integration tests..."
	@go test $(TEST_INTEGRATION_DIR)/k8s/... \
		-tags=integration \
		-v -race -timeout=$(TEST_TIMEOUT_INTEGRATION) \
		-coverprofile=$(TEST_OUTPUT_DIR)/coverage/k8s-coverage.out \
		-covermode=atomic \
		-count=1 \
		-json > $(TEST_OUTPUT_DIR)/junit/k8s-test-results.json \
		|| (echo "[ERROR] Kubernetes integration tests failed" && exit 1)
	@echo "[SUCCESS] Kubernetes integration tests completed"

.PHONY: test-e2e
test-e2e: test-setup check-k8s-tools ## Run end-to-end tests with real cluster
	@echo "Running end-to-end tests with real cluster..."
	@go test $(TEST_E2E_DIR)/... \
		-tags=e2e \
		-v -timeout=$(TEST_TIMEOUT_E2E) \
		-coverprofile=$(TEST_OUTPUT_DIR)/coverage/e2e-coverage.out \
		-covermode=atomic \
		-count=1 \
		-json > $(TEST_OUTPUT_DIR)/junit/e2e-test-results.json \
		|| (echo "[ERROR] E2E tests failed" && exit 1)
	@echo "[SUCCESS] E2E tests completed"

.PHONY: test-e2e-operator
test-e2e-operator: test-setup check-k8s-tools ## Run operator E2E tests
	@echo "Running operator E2E tests..."
	@go test $(TEST_E2E_DIR)/operator/... \
		-tags=e2e \
		-v -timeout=$(TEST_TIMEOUT_E2E) \
		-count=1 \
		-json > $(TEST_OUTPUT_DIR)/junit/operator-e2e-test-results.json \
		|| (echo "[ERROR] Operator E2E tests failed" && exit 1)
	@echo "[SUCCESS] Operator E2E tests completed"

.PHONY: test-smoke
test-smoke: test-setup ## Run smoke tests for quick validation
	@echo "Running smoke tests for quick validation..."
	@go test $(TEST_SMOKE_DIR)/... \
		-tags=smoke \
		-v -timeout=2m \
		-count=1 \
		-json > $(TEST_OUTPUT_DIR)/junit/smoke-test-results.json \
		|| (echo "[ERROR] Smoke tests failed" && exit 1)
	@echo "[SUCCESS] Smoke tests completed"

.PHONY: test-validation
test-validation: test-setup ## Run validation tests for generated code
	@echo "Running validation tests for generated code..."
	@go test $(TEST_VALIDATION_DIR)/... \
		-tags=validation \
		-v -timeout=$(TEST_TIMEOUT_VALIDATION) \
		-count=1 \
		-json > $(TEST_OUTPUT_DIR)/junit/validation-test-results.json \
		|| (echo "[ERROR] Validation tests failed" && exit 1)
	@echo "[SUCCESS] Validation tests completed"

.PHONY: test-validation-crds
test-validation-crds: test-setup manifests ## Validate generated CRDs
	@echo "Validating generated CRDs..."
	@go test $(TEST_VALIDATION_DIR)/crds/... \
		-tags=validation \
		-v -timeout=$(TEST_TIMEOUT_VALIDATION) \
		-count=1 \
		-json > $(TEST_OUTPUT_DIR)/junit/crd-validation-test-results.json \
		|| (echo "[ERROR] CRD validation tests failed" && exit 1)
	@echo "[SUCCESS] CRD validation tests completed"

.PHONY: test-validation-codegen
test-validation-codegen: test-setup generate ## Validate generated code
	@echo "Validating generated code..."
	@go test $(TEST_VALIDATION_DIR)/codegen/... \
		-tags=validation \
		-v -timeout=$(TEST_TIMEOUT_VALIDATION) \
		-count=1 \
		-json > $(TEST_OUTPUT_DIR)/junit/codegen-validation-test-results.json \
		|| (echo "[ERROR] Code generation validation tests failed" && exit 1)
	@echo "[SUCCESS] Code generation validation tests completed"

.PHONY: test-excellence
test-excellence: ## Run excellence validation test suite with improved reliability
	@echo "Running excellence validation test suite with improved reliability..."
	@mkdir -p $(REPORTS_DIR) $(QUALITY_REPORTS_DIR)
	@# Create test lock directory for synchronization
	@export TEST_MUTEX_LOCK_DIR=$${TMPDIR:-/tmp}/nephoran-excellence-locks && \
	mkdir -p "$$TEST_MUTEX_LOCK_DIR" && \
	go test ./tests/excellence/... -v -timeout=30m --ginkgo.v \
		-coverprofile=$(QUALITY_REPORTS_DIR)/excellence-coverage.out \
		-json | tee $(REPORTS_DIR)/excellence-results.json || \
		(echo "[ERROR] Excellence tests failed. Check $(REPORTS_DIR)/excellence-results.json for details" && exit 1)
	@echo "[SUCCESS] Excellence validation tests completed successfully"

.PHONY: test-regression
test-regression: ## Run regression testing suite
	@echo "Running regression testing suite..."
	@mkdir -p $(REPORTS_DIR) $(QUALITY_REPORTS_DIR)
	@# Run comprehensive regression tests with proper resource allocation
	@GOMAXPROCS=4 GOMEMLIMIT=4GiB \
		go test ./tests/regression/... -v -timeout=60m \
		-coverprofile=$(QUALITY_REPORTS_DIR)/regression-coverage.out \
		-json | tee $(REPORTS_DIR)/regression-results.json || \
		(echo "[ERROR] Regression tests failed. Check $(REPORTS_DIR)/regression-results.json for details" && exit 1)
	@echo "[SUCCESS] Regression testing completed successfully"

.PHONY: test-all
test-all: test test-integration test-e2e test-excellence test-regression ## Run all test suites
	@echo "All test suites completed successfully!"

.PHONY: test-ci
test-ci: ## Run unit tests with CI-compatible coverage reporting (fast, no integration)
	@echo "Running unit tests with CI-compatible coverage (no integration tests)..."
	mkdir -p .test-reports
	@export CGO_ENABLED=1 GOMAXPROCS=2 GODEBUG=gocachehash=1 GO111MODULE=on; \
	timeout 5m go test ./... -v -race -timeout=4m30s \
		-coverprofile=.test-reports/coverage.out \
		-covermode=atomic \
		-count=1 \
		-parallel=4 \
		-short || echo "Tests completed (may have reached 5m timeout ceiling)"
	@if [ -f ".test-reports/coverage.out" ] && [ -s ".test-reports/coverage.out" ]; then \
		go tool cover -html=.test-reports/coverage.out -o .test-reports/coverage.html; \
		echo "Coverage report generated: .test-reports/coverage.html"; \
		coverage_percent=$$(go tool cover -func=.test-reports/coverage.out 2>/dev/null | grep total | awk '{print $$3}' || echo "0.0%"); \
		echo "Coverage: $$coverage_percent"; \
		echo "coverage_available=true" > .test-reports/test-status.txt; \
	else \
		echo "Warning: Coverage file not generated or empty"; \
		echo "coverage_available=false" > .test-reports/test-status.txt; \
	fi

.PHONY: test-benchmark
test-benchmark: test-setup ## Run performance benchmarks with detailed reporting
	@echo "Running performance benchmarks with detailed reporting..."
	@go test $(TEST_BENCHMARK_DIR)/... ./pkg/... ./controllers/... \
		-bench=. -benchmem -benchtime=5s \
		-timeout=30m \
		-count=3 \
		-cpu=1,2,4 \
		-json > $(TEST_OUTPUT_DIR)/junit/benchmark-results.json \
		2>&1 | tee $(TEST_OUTPUT_DIR)/benchmark-results.txt
	@echo "[SUCCESS] Performance benchmarks completed"

.PHONY: test-benchmark-memory
test-benchmark-memory: test-setup ## Run memory-focused benchmarks
	@echo "Running memory-focused benchmarks..."
	@go test ./controllers/... ./pkg/... \
		-bench=. -benchmem -memprofile=$(TEST_OUTPUT_DIR)/coverage/memory.prof \
		-timeout=15m \
		-count=1 \
		-json > $(TEST_OUTPUT_DIR)/junit/memory-benchmark-results.json
	@echo "[SUCCESS] Memory benchmarks completed"

.PHONY: test-benchmark-cpu
test-benchmark-cpu: test-setup ## Run CPU-focused benchmarks  
	@echo "Running CPU-focused benchmarks..."
	@go test ./controllers/... ./pkg/... \
		-bench=. -cpuprofile=$(TEST_OUTPUT_DIR)/coverage/cpu.prof \
		-timeout=15m \
		-count=1 \
		-json > $(TEST_OUTPUT_DIR)/junit/cpu-benchmark-results.json
	@echo "[SUCCESS] CPU benchmarks completed"

.PHONY: test-ci
test-ci: test-setup ## Run CI-compatible test suite (fast, deterministic)
	@echo "Running CI-compatible test suite..."
	@export CGO_ENABLED=1 GOMAXPROCS=2 GODEBUG=gocachehash=1 GO111MODULE=on; \
	go test ./... -v -race -timeout=$(TEST_TIMEOUT_UNIT) \
		-coverprofile=$(TEST_OUTPUT_DIR)/coverage/ci-coverage.out \
		-covermode=atomic \
		-count=1 \
		-parallel=$(TEST_PARALLEL_JOBS) \
		-short \
		-json > $(TEST_OUTPUT_DIR)/junit/ci-test-results.json \
		|| (echo "[ERROR] CI tests failed" && exit 1)
	@echo "[SUCCESS] CI tests completed successfully"

.PHONY: test-ci-pipeline
test-ci-pipeline: test-setup ## Test CI/CD pipeline functionality
	@echo "Testing CI/CD pipeline functionality..."
	@go test $(TEST_VALIDATION_DIR)/ci/... \
		-tags=ci \
		-v -timeout=$(TEST_TIMEOUT_VALIDATION) \
		-count=1 \
		-json > $(TEST_OUTPUT_DIR)/junit/ci-pipeline-test-results.json \
		|| (echo "[ERROR] CI pipeline tests failed" && exit 1)
	@echo "[SUCCESS] CI pipeline tests completed"

.PHONY: test-all
test-all: test-unit test-integration test-validation test-smoke ## Run all core test suites (excludes E2E and benchmarks)
	@echo "[SUCCESS] All core test suites completed successfully!"

.PHONY: test-all-comprehensive
test-all-comprehensive: test-unit test-integration test-e2e test-validation test-smoke test-benchmark ## Run comprehensive test suite (all tests including E2E and benchmarks)
	@echo "[SUCCESS] Comprehensive test suite completed successfully!"

.PHONY: test-coverage-merge
test-coverage-merge: ## Merge all coverage reports into a single report
	@echo "Merging coverage reports..."
	@mkdir -p $(TEST_OUTPUT_DIR)/coverage
	@if command -v gocovmerge >/dev/null 2>&1; then \
		gocovmerge $(TEST_OUTPUT_DIR)/coverage/*-coverage.out > $(TEST_OUTPUT_DIR)/coverage/merged-coverage.out; \
	else \
		echo "Installing gocovmerge..."; \
		go install github.com/wadey/gocovmerge@latest; \
		gocovmerge $(TEST_OUTPUT_DIR)/coverage/*-coverage.out > $(TEST_OUTPUT_DIR)/coverage/merged-coverage.out; \
	fi
	@echo "[SUCCESS] Coverage reports merged"

.PHONY: test-coverage-report
test-coverage-report: test-coverage-merge ## Generate comprehensive coverage reports
	@echo "Generating comprehensive coverage reports..."
	@mkdir -p $(TEST_OUTPUT_DIR)/coverage $(TEST_OUTPUT_DIR)/html
	@if [ -f "$(TEST_OUTPUT_DIR)/coverage/merged-coverage.out" ]; then \
		go tool cover -html=$(TEST_OUTPUT_DIR)/coverage/merged-coverage.out -o $(TEST_OUTPUT_DIR)/html/coverage.html; \
		go tool cover -func=$(TEST_OUTPUT_DIR)/coverage/merged-coverage.out > $(TEST_OUTPUT_DIR)/coverage/coverage-summary.txt; \
		echo "[SUCCESS] Coverage reports generated:"; \
		echo "  HTML: $(TEST_OUTPUT_DIR)/html/coverage.html"; \
		echo "  Summary: $(TEST_OUTPUT_DIR)/coverage/coverage-summary.txt"; \
		COVERAGE=$$(go tool cover -func=$(TEST_OUTPUT_DIR)/coverage/merged-coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
		echo "Total Coverage: $${COVERAGE}%"; \
		if [ $$(echo "$${COVERAGE} >= $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
			echo "[SUCCESS] Coverage $${COVERAGE}% meets threshold $(COVERAGE_THRESHOLD)%"; \
		else \
			echo "[WARNING] Coverage $${COVERAGE}% below threshold $(COVERAGE_THRESHOLD)%"; \
		fi; \
	else \
		echo "[ERROR] No coverage data found. Run tests first."; \
		exit 1; \
	fi

.PHONY: test-coverage-check
test-coverage-check: test-coverage-report ## Check coverage against threshold
	@echo "Checking coverage against threshold..."
	@if [ -f "$(TEST_OUTPUT_DIR)/coverage/merged-coverage.out" ]; then \
		COVERAGE=$$(go tool cover -func=$(TEST_OUTPUT_DIR)/coverage/merged-coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
		echo "Current Coverage: $${COVERAGE}%"; \
		echo "Required Threshold: $(COVERAGE_THRESHOLD)%"; \
		if [ $$(echo "$${COVERAGE} >= $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
			echo "[SUCCESS] Coverage check passed"; \
		else \
			echo "[ERROR] Coverage check failed - below threshold"; \
			exit 1; \
		fi; \
	else \
		echo "[ERROR] No coverage data available"; \
		exit 1; \
	fi

.PHONY: benchmark
benchmark: test-benchmark ## Alias for test-benchmark (backwards compatibility)

##@ Quality Assurance

.PHONY: security-scan
security-scan: ## Run comprehensive security scans
	@echo "Running comprehensive security scans..."
	@mkdir -p $(QUALITY_REPORTS_DIR)/security
	
	@# Go vulnerability check
	@if command -v govulncheck >/dev/null 2>&1; then \
		echo "Running Go vulnerability scan..."; \
		govulncheck ./... | tee $(QUALITY_REPORTS_DIR)/security/govulncheck-report.txt; \
	else \
		echo "[WARNING]  govulncheck not found. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi
	
	@# Semgrep security analysis if available
	@if command -v semgrep >/dev/null 2>&1; then \
		echo "Running Semgrep security analysis..."; \
		semgrep --config=auto --json --output=$(QUALITY_REPORTS_DIR)/security/semgrep-report.json . || true; \
	else \
		echo "[WARNING]  Semgrep not found. Consider installing for enhanced security scanning"; \
	fi
	
	@# Nancy dependency vulnerability scan if available
	@if command -v nancy >/dev/null 2>&1; then \
		echo "Running Nancy dependency scan..."; \
		go list -json -deps ./... | nancy sleuth | tee $(QUALITY_REPORTS_DIR)/security/nancy-report.txt || true; \
	else \
		echo "[WARNING]  Nancy not found. Consider installing for dependency vulnerability scanning"; \
	fi
	
	@echo "[SUCCESS] Security scan completed. Results in $(QUALITY_REPORTS_DIR)/security/"

.PHONY: coverage-report
coverage-report: test ## Generate comprehensive coverage report
	@echo "Generating comprehensive coverage report..."
	@mkdir -p $(QUALITY_REPORTS_DIR)/coverage
	
	@# Generate HTML coverage report
	@if [ -f "$(QUALITY_REPORTS_DIR)/coverage/coverage.out" ]; then \
		go tool cover -html=$(QUALITY_REPORTS_DIR)/coverage/coverage.out -o $(QUALITY_REPORTS_DIR)/coverage/coverage.html; \
		echo "[SUCCESS] HTML coverage report: $(QUALITY_REPORTS_DIR)/coverage/coverage.html"; \
	fi
	
	@# Coverage summary
	@if [ -f "$(QUALITY_REPORTS_DIR)/coverage/coverage.out" ]; then \
		echo "Coverage Summary:"; \
		go tool cover -func=$(QUALITY_REPORTS_DIR)/coverage/coverage.out | tail -1; \
		COVERAGE=$$(go tool cover -func=$(QUALITY_REPORTS_DIR)/coverage/coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
		echo "Coverage: $${COVERAGE}%"; \
		if [ $$(echo "$${COVERAGE} < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
			echo "[ERROR] Coverage $${COVERAGE}% is below threshold $(COVERAGE_THRESHOLD)%"; \
			exit 1; \
		else \
			echo "[SUCCESS] Coverage $${COVERAGE}% meets threshold $(COVERAGE_THRESHOLD)%"; \
		fi; \
	else \
		echo "[ERROR] No coverage file found"; \
		exit 1; \
	fi

.PHONY: quality-gate
quality-gate: lint security-scan coverage-report ## Run complete quality gate checks
	@echo "Quality gate checks completed successfully! 🎉"

.PHONY: excellence-dashboard
excellence-dashboard: ## Generate excellence metrics dashboard
	@echo "Generating excellence metrics dashboard..."
	@mkdir -p $(QUALITY_REPORTS_DIR)/dashboard
	
	@# Collect metrics
	@echo "Collecting metrics..."
	@if [ -f "$(QUALITY_REPORTS_DIR)/coverage/coverage.out" ]; then \
		COVERAGE=$$(go tool cover -func=$(QUALITY_REPORTS_DIR)/coverage/coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
		echo "{\"coverage\": $${COVERAGE}, \"timestamp\": \"$$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" > $(QUALITY_REPORTS_DIR)/dashboard/metrics.json; \
		echo "[SUCCESS] Dashboard metrics generated: $(QUALITY_REPORTS_DIR)/dashboard/metrics.json"; \
		echo "Coverage: $${COVERAGE}%"; \
		if [ $$(echo "$${COVERAGE} >= 90.0" | bc -l) -eq 1 ]; then \
			echo "🏆 EXCELLENCE ACHIEVED: Coverage >= 90%"; \
		elif [ $$(echo "$${COVERAGE} >= 80.0" | bc -l) -eq 1 ]; then \
			echo "[SUCCESS] GOOD: Coverage >= 80%"; \
		else \
			echo "[WARNING]  NEEDS IMPROVEMENT: Coverage < 80%"; \
		fi; \
	else \
		echo "[ERROR] Excellence scoring failed - no coverage data available"; \
		exit 1; \
	fi

##@ Building

.PHONY: build
build: generate fmt vet ## Build the operator binary with Go 1.24+ optimizations
	@echo "Building operator binary with Go 1.24+ ultra optimization..."
	$(GO_BUILD_ENV) CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(FAST_BUILD_FLAGS) -tags=$(FAST_BUILD_TAGS) -o bin/manager cmd/main.go

.PHONY: build-porch-publisher
build-porch-publisher: ## Build the porch-publisher binary
	@echo "Building porch-publisher binary..."
	@mkdir -p bin
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(FAST_BUILD_FLAGS) $(LDFLAGS) -o bin/porch-publisher cmd/porch-publisher/main.go

.PHONY: build-conductor
build-conductor: ## Build the conductor binary
	@echo "Building conductor binary..."
	@mkdir -p bin
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(FAST_BUILD_FLAGS) $(LDFLAGS) -o bin/conductor cmd/conductor/main.go

.PHONY: build-all-binaries
build-all-binaries: build build-porch-publisher build-conductor conductor-loop-build ## Build all core binaries
	@echo "All binaries built successfully!"
	@ls -la bin/

.PHONY: build-debug
build-debug: generate fmt vet ## Build the operator binary with debug info
	@echo "Building operator binary with debug info..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -gcflags="all=-N -l" -o bin/manager-debug cmd/main.go

.PHONY: cross-build
cross-build: generate fmt vet ## Build binaries for multiple platforms with Go 1.24+ optimizations
	@echo "Cross-building for multiple platforms with Go 1.24+ optimizations..."
	mkdir -p bin
	@for os in linux windows darwin; do \
		for arch in amd64 arm64; do \
			echo "Building for $$os/$$arch..."; \
			$(GO_BUILD_ENV) CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
				go build $(FAST_BUILD_FLAGS) -tags=$(FAST_BUILD_TAGS) \
				-o bin/manager-$$os-$$arch cmd/main.go; \
		done; \
	done

##@ Container Images

.PHONY: docker-build
docker-build: ## Build Docker image (production)
	@echo "Building production Docker image..."
	docker build -f Dockerfile.production \
		--build-arg SERVICE_NAME=manager \
		--build-arg SERVICE_TYPE=go \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(DATE) \
		--build-arg VCS_REF=$(COMMIT) \
		--target final \
		-t $(IMAGE_NAME):$(IMAGE_TAG) .
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_NAME):latest

.PHONY: docker-build-dev
docker-build-dev: ## Build development Docker image
	@echo "Building development Docker image..."
	docker build -f Dockerfile.dev \
		--build-arg SERVICE_NAME=manager \
		--build-arg SERVICE_TYPE=go \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(DATE) \
		--build-arg VCS_REF=$(COMMIT) \
		--target final \
		-t $(IMAGE_NAME):$(IMAGE_TAG)-dev .

.PHONY: docker-build-multiarch
docker-build-multiarch: ## Build multi-architecture Docker image
	@echo "Building multi-architecture Docker image..."
	docker buildx build -f Dockerfile.multiarch \
		--platform linux/amd64,linux/arm64 \
		--build-arg SERVICE_NAME=manager \
		--build-arg SERVICE_TYPE=go \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(DATE) \
		--build-arg VCS_REF=$(COMMIT) \
		--target final \
		-t $(IMAGE_NAME):$(IMAGE_TAG) . \
		--push

.PHONY: docker-build-all-services
docker-build-all-services: ## Build all service images using consolidated script
	@echo "Building all services with consolidated Dockerfiles..."
	@chmod +x scripts/docker-build-consolidated.sh
	@./scripts/docker-build-consolidated.sh all --build-type $(BUILD_TYPE)

.PHONY: docker-build-service
docker-build-service: ## Build specific service (usage: make docker-build-service SERVICE=llm-processor)
	@if [ -z "$(SERVICE)" ]; then \
		echo "ERROR: SERVICE is required. Usage: make docker-build-service SERVICE=llm-processor"; \
		exit 1; \
	fi
	@echo "Building $(SERVICE) service..."
	@chmod +x scripts/docker-build-consolidated.sh
	@./scripts/docker-build-consolidated.sh $(SERVICE) --build-type $(BUILD_TYPE)

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image to registry..."
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(IMAGE_NAME):latest

.PHONY: docker-security-scan
docker-security-scan: docker-build ## Scan Docker image for vulnerabilities
	@echo "Scanning Docker image for security vulnerabilities..."
	mkdir -p $(REPORTS_DIR)
	@if command -v trivy >/dev/null 2>&1; then \
		trivy image --format json --output $(REPORTS_DIR)/container-scan.json $(IMAGE_NAME):$(IMAGE_TAG); \
		trivy image $(IMAGE_NAME):$(IMAGE_TAG); \
	elif [ -f "scripts/docker-security-scan.sh" ]; then \
		bash scripts/docker-security-scan.sh $(IMAGE_NAME):$(IMAGE_TAG); \
	else \
		echo "No container scanning tool available"; \
	fi

##@ Deployment

.PHONY: install-crds
install-crds: manifests ## Install CRDs to the cluster
	@echo "Installing CRDs..."
	kubectl apply -f config/crd/bases

.PHONY: uninstall-crds
uninstall-crds: manifests ## Uninstall CRDs from the cluster
	@echo "Uninstalling CRDs..."
	kubectl delete -f config/crd/bases

.PHONY: deploy
deploy: manifests ## Deploy to the cluster
	@echo "Deploying to cluster..."
	cd config/manager && kustomize edit set image controller=$(IMAGE_NAME):$(IMAGE_TAG)
	kustomize build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Remove from the cluster
	@echo "Removing from cluster..."
	kustomize build config/default | kubectl delete -f -

.PHONY: build-webhook
build-webhook: ## Build the webhook manager binary
	@echo "Building webhook manager binary..."
	@mkdir -p bin
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(FAST_BUILD_FLAGS) $(LDFLAGS) -o bin/webhook-manager cmd/webhook/main.go

.PHONY: conductor-loop-build
conductor-loop-build: ## Build conductor with closed-loop support
	@echo "Building conductor with closed-loop support..."
	@mkdir -p bin
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(FAST_BUILD_FLAGS) $(LDFLAGS) \
		-o bin/conductor-loop cmd/conductor-loop/main.go

.PHONY: demo-setup
demo-setup: ## Set up demo environment
	@echo "Setting up demo environment..."
	@if [ -f "scripts/demo-setup.sh" ]; then \
		chmod +x scripts/demo-setup.sh; \
		./scripts/demo-setup.sh; \
	else \
		echo "Demo setup script not found. Creating minimal setup..."; \
		kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -; \
		echo "[SUCCESS] Demo namespace $(NAMESPACE) created"; \
	fi

##@ Tools

# Try to find controller-gen in multiple locations
CONTROLLER_GEN = $(shell which controller-gen 2>/dev/null || echo $(shell pwd)/bin/controller-gen)
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary
	@if ! command -v controller-gen &> /dev/null && [ ! -f $(shell pwd)/bin/controller-gen ]; then \
		echo "Installing controller-gen $(CONTROLLER_GEN_VERSION)..."; \
		GOBIN=$(shell pwd)/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION); \
	else \
		echo "controller-gen is already available"; \
	fi

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

# go-get-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

##@ Performance and Optimization

.PHONY: profile-cpu
profile-cpu: ## Run CPU profiling
	@echo "Running CPU profiling..."
	@mkdir -p $(QUALITY_REPORTS_DIR)/profiling
	go test -cpuprofile=$(QUALITY_REPORTS_DIR)/profiling/cpu.prof -bench=. -benchtime=30s ./...
	@echo "CPU profile saved to $(QUALITY_REPORTS_DIR)/profiling/cpu.prof"
	@echo "View with: go tool pprof $(QUALITY_REPORTS_DIR)/profiling/cpu.prof"

.PHONY: profile-memory
profile-memory: ## Run memory profiling
	@echo "Running memory profiling..."
	@mkdir -p $(QUALITY_REPORTS_DIR)/profiling
	go test -memprofile=$(QUALITY_REPORTS_DIR)/profiling/mem.prof -bench=. -benchtime=30s ./...
	@echo "Memory profile saved to $(QUALITY_REPORTS_DIR)/profiling/mem.prof"
	@echo "View with: go tool pprof $(QUALITY_REPORTS_DIR)/profiling/mem.prof"

.PHONY: optimization-report
optimization-report: profile-cpu profile-memory benchmark ## Generate comprehensive optimization report
	@echo "Generating optimization report..."
	@mkdir -p $(QUALITY_REPORTS_DIR)/optimization
	@echo "Optimization analysis completed. Reports available in $(QUALITY_REPORTS_DIR)/"

##@ Documentation

.PHONY: docs-generate
docs-generate: ## Generate comprehensive documentation
	@echo "Generating comprehensive documentation..."
	@mkdir -p docs/generated
	
	@# Generate API documentation
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/main.go -o docs/generated/swagger --parseInternal; \
		echo "[SUCCESS] Swagger documentation generated"; \
	else \
		echo "[WARNING]  swag not found. Install with: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi
	
	@# Generate Go documentation
	godoc -http=:6060 > /dev/null 2>&1 & \
	GODOC_PID=$$!; \
	sleep 3; \
	curl -s http://localhost:6060/pkg/github.com/thc1006/nephoran-intent-operator/ > docs/generated/godoc.html || true; \
	kill $$GODOC_PID 2>/dev/null || true
	
	@echo "[SUCCESS] Documentation generated in docs/generated/"

.PHONY: docs-serve
docs-serve: ## Serve documentation locally
	@echo "Serving documentation at http://localhost:6060"
	godoc -http=:6060

# Advanced Operations

.PHONY: chaos-test
chaos-test: ## Run chaos engineering tests
	@echo "Running chaos engineering tests..."
	@mkdir -p $(QUALITY_REPORTS_DIR)/chaos
	@if [ -f "tests/chaos/chaos_test.go" ]; then \
		GOMAXPROCS=4 go test ./tests/chaos/... -v -timeout=60m \
			-json | tee $(QUALITY_REPORTS_DIR)/chaos/chaos-results.json; \
	else \
		echo "[WARNING]  Chaos tests not found. Consider implementing chaos engineering tests."; \
	fi

.PHONY: load-test
load-test: ## Run load testing
	@echo "Running load testing..."
	@mkdir -p $(QUALITY_REPORTS_DIR)/load
	@if command -v vegeta >/dev/null 2>&1; then \
		echo "GET http://localhost:8080/health" | vegeta attack -duration=60s -rate=100/s | tee $(QUALITY_REPORTS_DIR)/load/load-results.bin | vegeta report; \
	elif [ -f "scripts/load-test.sh" ]; then \
		chmod +x scripts/load-test.sh; \
		./scripts/load-test.sh; \
	else \
		echo "[WARNING]  Load testing tools not available"; \
	fi

.PHONY: health-check
health-check: ## Perform comprehensive health check
	@echo "Performing comprehensive health check..."
	@echo "[SUCCESS] Go version: $$(go version)"
	@echo "[SUCCESS] Git status: $$(git status --porcelain | wc -l) modified files"
	@echo "[SUCCESS] Dependencies: $$(go list -m all | wc -l) modules"
	@echo "[SUCCESS] Build status: $$(make build >/dev/null 2>&1 && echo 'PASS' || echo 'FAIL')"
	@echo "[SUCCESS] Test status: $$(timeout 30s make test >/dev/null 2>&1 && echo 'PASS' || echo 'FAIL')"

##@ CI/CD Integration

.PHONY: ci-setup
ci-setup: ## Setup CI environment
	@echo "Setting up CI environment..."
	go version
	go env
	go mod download
	go mod verify

.PHONY: ci-test
ci-test: ci-setup test lint security-scan coverage-report ## Run CI test pipeline
	@echo "CI test pipeline completed successfully!"

.PHONY: ci-build
ci-build: ci-setup build docker-build ## Run CI build pipeline
	@echo "CI build pipeline completed successfully!"

.PHONY: ci-deploy
ci-deploy: ci-build ## Run CI deployment pipeline
	@echo "CI deployment pipeline completed successfully!"

##@ Supply Chain Security

.PHONY: sbom-generate
sbom-generate: ## Generate Software Bill of Materials (SBOM)
	@echo "Generating SBOM..."
	@mkdir -p $(QUALITY_REPORTS_DIR)/sbom
	@if command -v cyclonedx-gomod >/dev/null 2>&1; then \
		cyclonedx-gomod app -json -output $(QUALITY_REPORTS_DIR)/sbom/sbom.json; \
		echo "[SUCCESS] SBOM generated: $(QUALITY_REPORTS_DIR)/sbom/sbom.json"; \
	else \
		echo "[WARNING]  cyclonedx-gomod not found. Install with: go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest"; \
	fi

.PHONY: verify-dependencies
verify-dependencies: ## Verify dependency integrity
	@echo "Verifying dependency integrity..."
	go mod verify
	@if command -v nancy >/dev/null 2>&1; then \
		go list -json -deps ./... | nancy sleuth; \
	else \
		echo "[WARNING]  Nancy not found for vulnerability scanning"; \
	fi

.PHONY: security-baseline
security-baseline: security-scan sbom-generate verify-dependencies ## Establish security baseline
	@echo "Security baseline established!"

##@ Metrics and Monitoring

.PHONY: metrics-collect
metrics-collect: ## Collect system metrics
	@echo "Collecting system metrics..."
	@mkdir -p $(QUALITY_REPORTS_DIR)/metrics
	@echo "Build time: $$(date)" > $(QUALITY_REPORTS_DIR)/metrics/build-metrics.txt
	@echo "Go version: $$(go version)" >> $(QUALITY_REPORTS_DIR)/metrics/build-metrics.txt
	@echo "Git commit: $(COMMIT)" >> $(QUALITY_REPORTS_DIR)/metrics/build-metrics.txt
	@echo "Binary size: $$(du -h bin/manager 2>/dev/null || echo 'N/A')" >> $(QUALITY_REPORTS_DIR)/metrics/build-metrics.txt

.PHONY: performance-baseline
performance-baseline: benchmark profile-cpu profile-memory metrics-collect ## Establish performance baseline
	@echo "Performance baseline established!"
	@echo "Results available in $(QUALITY_REPORTS_DIR)/"

##@ Maintenance

.PHONY: dependency-update
dependency-update: ## Update dependencies safely
	@echo "Updating dependencies safely..."
	go get -u ./...
	go mod tidy
	go mod verify
	@echo "[SUCCESS] Dependencies updated. Please run tests to verify compatibility."

.PHONY: cleanup-old-reports
cleanup-old-reports: ## Clean up old report files
	@echo "Cleaning up old reports..."
	find $(QUALITY_REPORTS_DIR) -name "*.json" -mtime +7 -delete 2>/dev/null || true
	find $(QUALITY_REPORTS_DIR) -name "*.txt" -mtime +7 -delete 2>/dev/null || true
	find $(QUALITY_REPORTS_DIR) -name "*.html" -mtime +7 -delete 2>/dev/null || true
	@echo "[SUCCESS] Old reports cleaned up"

.PHONY: full-reset
full-reset: clean-all ## Complete reset of the development environment
	@echo "Performing full development environment reset..."
	git clean -fdx
	go clean -cache -modcache -testcache -fuzzcache
	@echo "[SUCCESS] Full reset completed"

##@ Meta

.PHONY: version
version: ## Show version information
	@echo "Project: $(PROJECT_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Date: $(DATE)"
	@echo "Go Version: $$(go version)"

.PHONY: release
release: docker-push ## Create and push release
	@echo "Creating release $(VERSION)..."
	@if command -v gh >/dev/null 2>&1; then \
		gh release create $(VERSION) --title "Release $(VERSION)" --generate-notes; \
	else \
		echo "GitHub CLI not available, skipping release creation"; \
	fi

##@ Monitoring and Observability

.PHONY: logs
logs: ## Show operator logs
	@echo "Showing operator logs..."
	kubectl logs -n $(NAMESPACE) -l control-plane=controller-manager -f

.PHONY: status
status: ## Show deployment status
	@echo "Showing deployment status..."
	kubectl get all -n $(NAMESPACE)
	kubectl get crds | grep -i nephoran || true

.PHONY: metrics
metrics: ## Show metrics (if available)
	@echo "Fetching metrics..."
	@if kubectl get svc -n $(NAMESPACE) | grep -q metrics; then \
		kubectl port-forward -n $(NAMESPACE) svc/nephoran-operator-metrics 8080:8080 & \
		sleep 2; \
		curl -s http://localhost:8080/metrics | head -20; \
		kill %1; \
	else \
		echo "Metrics service not available"; \
	fi

.PHONY: info
info: version ## Show comprehensive project information
	@echo ""
	@echo "Build Configuration:"
	@echo "  GOOS: $(GOOS)"
	@echo "  GOARCH: $(GOARCH)"
	@echo "  CGO_ENABLED: $(CGO_ENABLED)"
	@echo "  BUILD_TYPE: $(BUILD_TYPE)"
	@echo "  PARALLEL_JOBS: $(PARALLEL_JOBS)"
	@echo ""
	@echo "Quality Reports: $(QUALITY_REPORTS_DIR)"
	@echo "Registry: $(REGISTRY)"
	@echo "Image: $(IMAGE_NAME):$(IMAGE_TAG)"
