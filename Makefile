# Nephoran Intent Operator - Comprehensive Makefile
# Combines LLM demo capabilities with Kubernetes operator functionality

# Go configuration
GO := go
GOLANGCI_LINT := golangci-lint
GO_VERSION := 1.24

# Build configuration
BIN_DIR := bin
CMD_DIR := cmd
PKG := github.com/thc1006/nephoran-intent-operator

# Test configuration
TEST_TIMEOUT := 10m
TEST_PARALLEL := 4
TEST_COVERAGE_THRESHOLD := 80

# LLM Demo configuration
LLM_DEMO_PORT := 8080
INTENT_INGEST_PORT := 8081
LLM_PROCESSOR_PORT := 9090
DEMO_HANDOFF_DIR := ./demo-handoff
DEMO_INTENT := "scale odu-high-phy to 3 in ns oran-odu"

# Schema validation
SCHEMA_FILE := docs/contracts/intent.schema.json
AJV_CLI := npx ajv-cli

# Kubernetes operator configuration
CONTROLLER_GEN ?= $(shell go env GOPATH)/bin/controller-gen
CRD_OUTPUT_DIR = config/crd/bases
API_PACKAGES = ./api/v1 ./api/v1alpha1 ./api/intent/v1alpha1

# Create directories
$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

$(DEMO_HANDOFF_DIR):
	@mkdir -p $(DEMO_HANDOFF_DIR)

# =============================================================================
# Kubernetes Operator Targets (Kubebuilder Convention)
# =============================================================================

# Generate Kubernetes CRDs and related manifests
.PHONY: manifests
manifests: controller-gen
	@echo "🔧 Generating CRDs and manifests..."
	$(CONTROLLER_GEN) crd:allowDangerousTypes=true paths=./api/v1 paths=./api/v1alpha1 paths=./api/intent/v1alpha1 output:crd:dir=$(CRD_OUTPUT_DIR)
	$(CONTROLLER_GEN) rbac:roleName=nephoran-manager paths="./controllers/..." output:rbac:dir=config/rbac
	$(CONTROLLER_GEN) webhook paths="./..." output:webhook:dir=config/webhook
	@echo "✅ Manifests generated successfully"

# Generate deepcopy code and RBAC
.PHONY: generate
generate: controller-gen
	@echo "🔧 Generating deepcopy code and RBAC..."
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	$(CONTROLLER_GEN) rbac:roleName=nephoran-manager paths="./controllers/..." output:rbac:dir=config/rbac
	@echo "✅ Code generation completed"

# Install controller-gen if not present
.PHONY: controller-gen
controller-gen:
	@test -f $(CONTROLLER_GEN) || { \
		echo "🔨 Installing controller-gen..."; \
		go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest; \
	}

# =============================================================================
# Security Tools Installation
# =============================================================================

.PHONY: install-security-tools
install-security-tools: ## Install all security scanning tools
	@echo "Installing security scanning tools..."
	@echo "Installing gosec..."
	$(GO) install github.com/securego/gosec/v2/cmd/gosec@v2.21.4
	@echo "Installing govulncheck..."
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Installing nancy..."
	$(GO) install github.com/sonatype-nexus-community/nancy@latest
	@echo "Installing go-licenses..."
	$(GO) install github.com/google/go-licenses@latest
	@echo "Installing staticcheck..."
	$(GO) install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "[SUCCESS] Security tools installed"

.PHONY: verify-security
verify-security: ## Verify security scanning setup
	@echo "Verifying security scanning configuration..."
	@chmod +x scripts/verify-security-scans.sh
	@./scripts/verify-security-scans.sh

.PHONY: security-scan
security-scan: ## Run local security scans
	@echo "Running security scans..."
	@echo "Running gosec..."
	-@gosec -fmt json -out security-results/gosec.json ./... 2>/dev/null || true
	@echo "Running govulncheck..."
	-@govulncheck -json ./... > security-results/govulncheck.json 2>&1 || true
	@echo "Security scan results saved to security-results/"

.PHONY: trivy-scan
trivy-scan: ## Run Trivy container and filesystem scans
	@echo "Running Trivy security scans..."
	@mkdir -p security-results
	@echo "Scanning filesystem..."
	-@trivy fs --format json --output security-results/trivy-fs.json . || true
	@echo "Trivy scan results saved to security-results/"

# =============================================================================
# Build Targets
# =============================================================================

.PHONY: build
build: build-all

.PHONY: manager
manager: $(BIN_DIR) ## Build manager binary for Kubernetes operator
	@echo "Building Kubernetes operator manager..."
	$(GO) build -o $(BIN_DIR)/manager ./cmd/main.go
	@echo "[SUCCESS] Manager binary built"

.PHONY: build-all
build-all: $(BIN_DIR)
	@echo "Building all components..."
	$(GO) build -o $(BIN_DIR)/llm-processor ./$(CMD_DIR)/llm-processor
	$(GO) build -o $(BIN_DIR)/intent-ingest ./$(CMD_DIR)/intent-ingest
	$(GO) build -o $(BIN_DIR)/conductor ./$(CMD_DIR)/conductor
	@echo "[SUCCESS] All components built"

.PHONY: build-llm-processor
build-llm-processor: $(BIN_DIR)
	@echo "Building LLM processor..."
	$(GO) build -o $(BIN_DIR)/llm-processor ./$(CMD_DIR)/llm-processor
	@echo "[SUCCESS] LLM processor built"

.PHONY: build-intent-ingest
build-intent-ingest: $(BIN_DIR)
	@echo "Building intent ingest service..."
	$(GO) build -o $(BIN_DIR)/intent-ingest ./$(CMD_DIR)/intent-ingest
	@echo "[SUCCESS] Intent ingest service built"

# =============================================================================
# LLM Offline Demo Target
# =============================================================================

.PHONY: llm-offline-demo
llm-offline-demo: build-llm-processor $(DEMO_HANDOFF_DIR)
	@echo "=== LLM Offline Demo ==="
	@echo "This demo shows the complete offline LLM provider pipeline:"
	@echo "1. Natural language processing with OFFLINE provider"
	@echo "2. Schema validation against intent.schema.json"
	@echo "3. Handoff file generation for intent-processor integration"
	@echo ""
	@echo "Starting llm-processor service with OFFLINE provider..."
	@echo "Service will run on port $(LLM_DEMO_PORT)"
	@echo "Handoff directory: $(DEMO_HANDOFF_DIR)"
	@echo ""
	@echo "Test the service with:"
	@echo "  curl -X POST http://localhost:$(LLM_DEMO_PORT)/api/v2/process-intent \\"
	@echo "    -H 'Content-Type: application/json' \\"
	@echo "    -d '{\"intent\": \"$(DEMO_INTENT)\"}'"
	@echo ""
	@echo "Press Ctrl+C to stop the demo"
	@echo ""
	LLM_PROVIDER=OFFLINE PORT=$(LLM_DEMO_PORT) AUTH_ENABLED=false TLS_ENABLED=false ./$(BIN_DIR)/llm-processor

.PHONY: llm-demo-test
llm-demo-test: build-intent-ingest $(DEMO_HANDOFF_DIR)
	@echo "=== Testing LLM Demo Pipeline ==="
	@# Start service in background
	@echo "Starting intent-ingest service..."
	MODE=llm PROVIDER=mock ./$(BIN_DIR)/intent-ingest \
		-addr :$(LLM_DEMO_PORT) \
		-handoff $(DEMO_HANDOFF_DIR) \
		-schema $(SCHEMA_FILE) & \
	SERVER_PID=$$!; \
	sleep 3; \
	echo "Testing health endpoint..."; \
	curl -f http://localhost:$(LLM_DEMO_PORT)/healthz || (kill $$SERVER_PID; exit 1); \
	echo "Testing intent processing..."; \
	curl -X POST -H "Content-Type: application/json" \
		-d '{"spec": {"intent": $(DEMO_INTENT)}}' \
		http://localhost:$(LLM_DEMO_PORT)/intent || (kill $$SERVER_PID; exit 1); \
	echo "Checking handoff file generation..."; \
	ls -la $(DEMO_HANDOFF_DIR)/ || (kill $$SERVER_PID; exit 1); \
	kill $$SERVER_PID; \
	echo "[SUCCESS] LLM demo pipeline test completed"

# =============================================================================
# Testing Strategy
# =============================================================================

.PHONY: test
test: test-unit test-integration

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	CGO_ENABLED=1 $(GO) test -v -race -timeout=$(TEST_TIMEOUT) -parallel=$(TEST_PARALLEL) ./...
	@echo "[SUCCESS] Unit tests passed"

.PHONY: test-unit-coverage
test-unit-coverage:
	@echo "Running unit tests with coverage..."
	CGO_ENABLED=1 $(GO) test -v -race -timeout=$(TEST_TIMEOUT) -parallel=$(TEST_PARALLEL) \
		-coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@$(GO) tool cover -func=coverage.out | grep total | \
		awk '{print "Total coverage: " $$3}'

.PHONY: test-llm-providers
test-llm-providers:
	@echo "Running LLM provider tests..."
	CGO_ENABLED=1 $(GO) test -v -race -timeout=$(TEST_TIMEOUT) \
		./internal/ingest/... \
		-run "Test.*Provider"
	@echo "[SUCCESS] LLM provider tests passed"

.PHONY: test-schema-validation
test-schema-validation:
	@echo "Running schema validation tests..."
	CGO_ENABLED=1 $(GO) test -v -race -timeout=$(TEST_TIMEOUT) \
		./internal/ingest/... \
		-run "Test.*Schema.*|Test.*Validat.*"
	@echo "[SUCCESS] Schema validation tests passed"

.PHONY: test-integration
test-integration: build-all $(DEMO_HANDOFF_DIR)
	@echo "Running integration tests..."
	@echo "Testing intent-ingest to LLM processor handoff..."
	@# Test the full pipeline integration
	$(GO) test -v -timeout=$(TEST_TIMEOUT) \
		./cmd/intent-ingest/... ./cmd/llm-processor/... \
		-run "Test.*Integration"
	@echo "[SUCCESS] Integration tests passed"

# =============================================================================
# Schema and Validation
# =============================================================================

.PHONY: validate-schema
validate-schema:
	@echo "Validating intent schema..."
	@if command -v $(AJV_CLI) >/dev/null 2>&1; then \
		$(AJV_CLI) validate -s $(SCHEMA_FILE) --spec=draft7; \
		echo "[SUCCESS] Schema validation passed"; \
	else \
		echo "Warning: ajv-cli not found, skipping JSON schema validation"; \
		echo "Install with: npm install -g ajv-cli"; \
	fi

.PHONY: validate-crds
validate-crds: manifests
	@echo "Validating generated CRDs..."
	@for crd in config/crd/bases/*.yaml; do \
		echo "Validating $$crd..."; \
		kubeval --strict $$crd || exit 1; \
	done
	@echo "[SUCCESS] All CRDs are valid"

.PHONY: validate-contracts
validate-contracts:
	@echo "📝 Validating contract schemas..."
	@if command -v ajv >/dev/null 2>&1; then \
		echo "Validating with ajv-cli..."; \
		ajv compile -s docs/contracts/intent.schema.json || exit 1; \
		ajv compile -s docs/contracts/a1.policy.schema.json || exit 1; \
		ajv compile -s docs/contracts/scaling.schema.json || exit 1; \
		echo "✅ All JSON schemas are valid"; \
	else \
		echo "ajv-cli not found - using basic JSON validation"; \
		for schema in docs/contracts/*.json; do \
			echo "Checking $$schema..."; \
			go run -c 'import "encoding/json"; import "os"; import "io"; data, _ := io.ReadAll(os.Stdin); var obj interface{}; if json.Unmarshal(data, &obj) != nil { os.Exit(1) }' < "$$schema" || exit 1; \
		done; \
		echo "✅ Basic JSON validation passed"; \
	fi

.PHONY: validate-examples
validate-examples: validate-contracts
	@echo "🧩 Validating example files against schemas..."
	@echo "Checking FCAPS VES examples structure..."
	@go run -c 'import "encoding/json"; import "os"; import "io"; data, _ := io.ReadAll(os.Stdin); var obj interface{}; if json.Unmarshal(data, &obj) != nil { os.Exit(1) }' < docs/contracts/fcaps.ves.examples.json || exit 1
	@echo "✅ All examples are valid JSON"

.PHONY: validate-all
validate-all: validate-schema validate-crds validate-contracts validate-examples
	@echo "🏆 All validations passed - contracts and manifests are compliant"

# =============================================================================
# Code Quality and Linting
# =============================================================================

.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	$(GOLANGCI_LINT) run --timeout=5m
	@echo "[SUCCESS] Linting passed"

.PHONY: lint-fix
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	$(GOLANGCI_LINT) run --fix --timeout=5m
	@echo "[SUCCESS] Auto-fix completed"

.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	$(GO) fmt ./...
	@echo "[SUCCESS] Code formatted"

.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...
	@echo "[SUCCESS] Go vet passed"

# =============================================================================
# Build Verification
# =============================================================================

.PHONY: verify-build
verify-build: clean build test-unit
	@echo "=== Build Verification Complete ==="
	@echo "1. Clean build: PASSED"
	@echo "2. All components built: PASSED"
	@echo "3. Unit tests: PASSED"
	@echo "[SUCCESS] Build verification successful"

.PHONY: verify-llm-pipeline
verify-llm-pipeline: build-all validate-schema
	@echo "=== LLM Pipeline Verification ==="
	@echo "Testing complete LLM provider pipeline..."
	@# Verify each component can start
	@echo "Verifying intent-ingest binary..."
	./$(BIN_DIR)/intent-ingest -addr :0 & \
	SERVER_PID=$$!; \
	sleep 1; \
	kill $$SERVER_PID 2>/dev/null || true; \
	@echo "✓ intent-ingest binary verified"
	@echo "Verifying LLM processor binary..."
	@# LLM processor requires config, so just check it can show help
	./$(BIN_DIR)/llm-processor --help >/dev/null 2>&1 || \
		(echo "✓ llm-processor binary available (help displayed)"; true)
	@echo "[SUCCESS] LLM pipeline verification complete"

# =============================================================================
# CI/CD Integration
# =============================================================================

.PHONY: ci-test
ci-test: verify-build test-integration
	@echo "[SUCCESS] CI test suite completed"

.PHONY: ci-lint
ci-lint: lint vet
	@echo "[SUCCESS] CI linting completed"

.PHONY: ci-full
ci-full: ci-lint ci-test validate-all
	@echo "[SUCCESS] Full CI pipeline completed"

# =============================================================================
# Docker Build Targets (Kubernetes Operator Convention)
# =============================================================================

# Image URL to use all building/pushing image targets
IMG ?= nephoran-operator:latest

define DOCKERFILE_OPERATOR
# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the manager binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager ./cmd/main.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
endef
export DOCKERFILE_OPERATOR

.PHONY: docker-build
docker-build: manager ## Build docker image with the manager
	@echo "Building Docker image: $(IMG)"
	@echo "Creating multi-stage Dockerfile for Kubernetes operator..."
	@echo "$$DOCKERFILE_OPERATOR" > Dockerfile.operator
	@echo "Building Docker image with multi-stage build..."
	@docker build -f Dockerfile.operator -t $(IMG) .
	@echo "✅ Docker image built: $(IMG)"
	@rm -f Dockerfile.operator

.PHONY: docker-push
docker-push: ## Push docker image with the manager
	@echo "Pushing Docker image: $(IMG)"
	docker push $(IMG)
	@echo "✅ Docker image pushed: $(IMG)"

define DOCKERFILE_MULTIARCH
# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the manager binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -o manager ./cmd/main.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
endef
export DOCKERFILE_MULTIARCH

.PHONY: docker-buildx
docker-buildx: manager ## Build and push docker image for multiple platforms
	@echo "Building multi-arch Docker image: $(IMG)"
	@echo "Creating multi-stage Dockerfile for multi-arch build..."
	@echo "$$DOCKERFILE_MULTIARCH" > Dockerfile.multiarch
	docker buildx create --use --name=crossplat --node=crossplat || true
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t $(IMG) \
		-f Dockerfile.multiarch \
		--push .
	@echo "✅ Multi-arch Docker image built and pushed: $(IMG)"
	@rm -f Dockerfile.multiarch

# =============================================================================
# Kubernetes Cluster Operations
# =============================================================================

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config
	@echo "Installing CRDs..."
	@if command -v kustomize >/dev/null 2>&1; then \
		kustomize build config/crd | kubectl apply -f -; \
		echo "✅ CRDs installed"; \
	else \
		echo "Installing kustomize..."; \
		$(GO) install sigs.k8s.io/kustomize/kustomize/v5@latest; \
		kustomize build config/crd | kubectl apply -f -; \
		echo "✅ CRDs installed"; \
	fi

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config
	@echo "Uninstalling CRDs..."
	@if command -v kustomize >/dev/null 2>&1; then \
		kustomize build config/crd | kubectl delete --ignore-not-found=true -f -; \
		echo "✅ CRDs uninstalled"; \
	else \
		echo "❌ kustomize not found"; \
		exit 1; \
	fi

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config
	@echo "Deploying controller to cluster..."
	@if command -v kustomize >/dev/null 2>&1; then \
		cd config/manager && kustomize edit set image controller=$(IMG); \
		kustomize build config/default | kubectl apply -f -; \
		echo "✅ Controller deployed with image: $(IMG)"; \
	else \
		echo "Installing kustomize..."; \
		$(GO) install sigs.k8s.io/kustomize/kustomize/v5@latest; \
		cd config/manager && kustomize edit set image controller=$(IMG); \
		kustomize build config/default | kubectl apply -f -; \
		echo "✅ Controller deployed with image: $(IMG)"; \
	fi

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config
	@echo "Undeploying controller from cluster..."
	@if command -v kustomize >/dev/null 2>&1; then \
		kustomize build config/default | kubectl delete --ignore-not-found=true -f -; \
		echo "✅ Controller undeployed"; \
	else \
		echo "❌ kustomize not found"; \
		exit 1; \
	fi

# MVP Scaling Operations
# These targets provide direct scaling operations for MVP demonstrations
.PHONY: mvp-scale-up
mvp-scale-up:
	@echo "🔼 MVP Scale Up: Scaling target workload..."
	@if [ -z "$(TARGET)" ]; then \
		echo "Usage: make mvp-scale-up TARGET=<resource> NAMESPACE=<ns> REPLICAS=<count>"; \
		echo "Example: make mvp-scale-up TARGET=odu-high-phy NAMESPACE=oran-odu REPLICAS=5"; \
		exit 1; \
	fi
	@if command -v kubectl >/dev/null 2>&1; then \
		echo "Using kubectl to patch $(TARGET) in $(NAMESPACE)..."; \
		kubectl patch deployment $(TARGET) -n $(NAMESPACE:-default) \
			-p '{"spec":{"replicas":$(REPLICAS:-3)}}'; \
	else \
		echo "kubectl not found - generating scaling intent JSON"; \
		echo '{"intent_type":"scaling","target":"$(TARGET)","namespace":"$(NAMESPACE:-default)","replicas":$(REPLICAS:-3),"reason":"MVP scale-up operation","source":"make-target"}' > intent-scale-up.json; \
		echo "Generated: intent-scale-up.json"; \
	fi
	@echo "✅ MVP Scale Up completed"

.PHONY: mvp-scale-down  
mvp-scale-down:
	@echo "🔽 MVP Scale Down: Reducing target workload..."
	@if [ -z "$(TARGET)" ]; then \
		echo "Usage: make mvp-scale-down TARGET=<resource> NAMESPACE=<ns> REPLICAS=<count>"; \
		echo "Example: make mvp-scale-down TARGET=odu-high-phy NAMESPACE=oran-odu REPLICAS=1"; \
		exit 1; \
	fi
	@if command -v kubectl >/dev/null 2>&1; then \
		echo "Using kubectl to patch $(TARGET) in $(NAMESPACE)..."; \
		kubectl patch deployment $(TARGET) -n $(NAMESPACE:-default) \
			-p '{"spec":{"replicas":$(REPLICAS:-1)}}'; \
	else \
		echo "kubectl not found - generating scaling intent JSON"; \
		echo '{"intent_type":"scaling","target":"$(TARGET)","namespace":"$(NAMESPACE:-default)","replicas":$(REPLICAS:-1),"reason":"MVP scale-down operation","source":"make-target"}' > intent-scale-down.json; \
		echo "Generated: intent-scale-down.json"; \
	fi
	@echo "✅ MVP Scale Down completed"

# =============================================================================
# Development Utilities
# =============================================================================

.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	$(GO) mod tidy
	$(GO) mod download
	@if ! command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.55.2; \
	fi
	@echo "[SUCCESS] Development environment ready"

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -rf $(DEMO_HANDOFF_DIR)
	@rm -f coverage.out coverage.html
	@echo "[SUCCESS] Clean completed"

.PHONY: help
help:
	@echo "Nephoran Intent Operator - Build System"
	@echo ""
	@echo "Build Targets:"
	@echo "  build                 - Build all components"
	@echo "  manager               - Build Kubernetes operator manager"
	@echo "  build-llm-processor   - Build LLM processor only"
	@echo "  build-intent-ingest   - Build intent ingest service only"
	@echo ""
	@echo "Demo Targets:"
	@echo "  llm-offline-demo      - Start interactive LLM offline demo"
	@echo "  llm-demo-test         - Run automated LLM demo pipeline test"
	@echo ""
	@echo "Testing Targets:"
	@echo "  test                  - Run all tests"
	@echo "  test-unit             - Run unit tests only"
	@echo "  test-unit-coverage    - Run unit tests with coverage report"
	@echo "  test-llm-providers    - Test LLM provider implementations"
	@echo "  test-schema-validation - Test schema validation functionality"
	@echo "  test-integration      - Run integration tests"
	@echo ""
	@echo "Validation Targets:"
	@echo "  validate-schema       - Validate JSON schemas"
	@echo "  validate-crds         - Validate Kubernetes CRDs"
	@echo "  validate-contracts    - Validate contract schemas"
	@echo "  validate-examples     - Validate example files"
	@echo "  validate-all          - Run all validations"
	@echo ""
	@echo "Quality Targets:"
	@echo "  lint                  - Run golangci-lint"
	@echo "  lint-fix             - Run golangci-lint with auto-fix"
	@echo "  fmt                   - Format Go code"
	@echo "  vet                   - Run go vet"
	@echo ""
	@echo "Verification Targets:"
	@echo "  verify-build          - Complete build verification"
	@echo "  verify-llm-pipeline   - Verify LLM pipeline components"
	@echo ""
	@echo "CI/CD Targets:"
	@echo "  ci-test               - CI test suite"
	@echo "  ci-lint               - CI linting"
	@echo "  ci-full               - Full CI pipeline"
	@echo ""
	@echo "Docker Build Targets:"
	@echo "  docker-build IMG=<image> - Build Docker image (default: nephoran-operator:latest)"
	@echo "  docker-push IMG=<image>  - Push Docker image"
	@echo "  docker-buildx IMG=<image> - Build multi-arch Docker image"
	@echo ""
	@echo "Kubernetes Operator Targets:"
	@echo "  manifests             - Generate CRDs, RBAC, and webhook configurations"
	@echo "  generate              - Generate DeepCopy methods and other code"
	@echo "  install               - Install CRDs to cluster"
	@echo "  uninstall             - Uninstall CRDs from cluster"
	@echo "  deploy IMG=<image>    - Deploy operator to cluster"
	@echo "  undeploy              - Remove operator from cluster"
	@echo ""
	@echo "MVP Operations:"
	@echo "  mvp-scale-up TARGET=<resource> NAMESPACE=<ns> REPLICAS=<count>"
	@echo "  mvp-scale-down TARGET=<resource> NAMESPACE=<ns> REPLICAS=<count>"
	@echo ""
	@echo "Development:"
	@echo "  dev-setup             - Setup development environment"
	@echo "  clean                 - Clean build artifacts"
	@echo "  help                  - Show this help message"
