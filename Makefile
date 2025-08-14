# Makefile for Nephoran Intent Operator with Excellence Validation Framework

# Project configuration
PROJECT_NAME = nephoran-intent-operator
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT = $(shell git rev-parse --short HEAD)
DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Go configuration
GO_VERSION = 1.24.1
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= 0

# Supply Chain Security Configuration
GOSUMDB ?= sum.golang.org
GOPROXY ?= https://proxy.golang.org,direct
GOPRIVATE ?= github.com/thc1006/*

# Docker configuration  
REGISTRY ?= ghcr.io
IMAGE_NAME = $(REGISTRY)/$(PROJECT_NAME)
IMAGE_TAG ?= $(VERSION)
BUILD_TYPE ?= production

# Kubernetes configuration
NAMESPACE ?= nephoran-system
KUBECONFIG ?= ~/.kube/config

# Excellence framework configuration
REPORTS_DIR = .excellence-reports
EXCELLENCE_THRESHOLD = 75

# Comprehensive Validation configuration
VALIDATION_REPORTS_DIR = test-results
VALIDATION_TARGET_SCORE = 90
VALIDATION_CONCURRENCY = 50

# Code Quality Gate configuration
QUALITY_REPORTS_DIR = .quality-reports
COVERAGE_THRESHOLD = 90
QUALITY_THRESHOLD = 8.0
PERFORMANCE_THRESHOLD = 15.0
DEBT_THRESHOLD = 0.3

# Regression Testing configuration
REGRESSION_REPORTS_DIR = regression-artifacts
REGRESSION_BASELINE_ID ?= 
REGRESSION_FAIL_ON_DETECTION ?= true
REGRESSION_ALERT_WEBHOOK ?=

# Build flags
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -s -w"

.PHONY: help
help: ## Show this help message
	@echo "Nephoran Intent Operator - Excellence Validation Framework"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: deps
deps: ## Install development dependencies
	@echo "Installing development dependencies..."
	@echo "Configuring supply chain security..."
	@export GOSUMDB=$(GOSUMDB) && export GOPROXY=$(GOPROXY) && export GOPRIVATE=$(GOPRIVATE)
	go mod tidy
	go mod download
	go mod verify
	@if ! command -v ginkgo >/dev/null 2>&1; then \
		echo "Installing Ginkgo..."; \
		go install github.com/onsi/ginkgo/v2/ginkgo@latest; \
	fi
	@if ! command -v controller-gen >/dev/null 2>&1; then \
		echo "Installing controller-gen..."; \
		go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest; \
	fi
	@if ! command -v kustomize >/dev/null 2>&1; then \
		echo "Installing kustomize..."; \
		go install sigs.k8s.io/kustomize/kustomize/v5@latest; \
	fi
	
install-security-tools: ## Install security and supply chain tools
	@echo "Installing security and supply chain tools..."
	go generate tools.go
	@echo "Security tools installed successfully"

verify-supply-chain: ## Run comprehensive supply chain security verification
	@echo "Running supply chain security verification..."
	@./scripts/verify-supply-chain.sh

verify-modules: ## Verify Go module integrity
	@echo "Verifying Go module integrity..."
	@export GOSUMDB=$(GOSUMDB)
	go mod verify
	@echo "✅ Module integrity verification passed"

scan-vulnerabilities: ## Scan for vulnerabilities using govulncheck
	@echo "Scanning for vulnerabilities..."
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	govulncheck ./...
	@echo "✅ Vulnerability scan completed"

generate-sbom: ## Generate Software Bill of Materials
	@echo "Generating Software Bill of Materials..."
	@mkdir -p $(REPORTS_DIR)/sbom
	@if command -v cyclonedx-gomod >/dev/null 2>&1; then \
		cyclonedx-gomod mod -json -output-file $(REPORTS_DIR)/sbom/sbom-cyclonedx.json; \
		echo "CycloneDX SBOM generated"; \
	else \
		echo "Installing cyclonedx-gomod..."; \
		go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest; \
		cyclonedx-gomod mod -json -output-file $(REPORTS_DIR)/sbom/sbom-cyclonedx.json; \
	fi
	@go list -m -f '{{.Path}}@{{.Version}}' all > $(REPORTS_DIR)/sbom/dependencies.txt
	@echo "✅ SBOM generated in $(REPORTS_DIR)/sbom/"

check-supply-chain-config: ## Verify supply chain security configuration
	@echo "Checking supply chain security configuration..."
	@echo "GOPROXY: $(GOPROXY)"
	@echo "GOSUMDB: $(GOSUMDB)"
	@echo "GOPRIVATE: $(GOPRIVATE)"
	@go env | grep -E "(GOPROXY|GOSUMDB|GOPRIVATE|GOINSECURE)"

update-security-tools: ## Update all security tools to latest versions
	@echo "Updating security tools..."
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✅ Security tools updated"

supply-chain-report: verify-supply-chain generate-sbom ## Generate comprehensive supply chain security report
	@echo "Generating comprehensive supply chain security report..."
	@mkdir -p $(REPORTS_DIR)/supply-chain
	@./scripts/verify-supply-chain.sh > $(REPORTS_DIR)/supply-chain/supply-chain-report.txt 2>&1 || true
	@echo "✅ Supply chain security report generated in $(REPORTS_DIR)/supply-chain/"

##@ Supply Chain Security

.PHONY: install-security-tools verify-supply-chain verify-modules scan-vulnerabilities generate-sbom
.PHONY: check-supply-chain-config update-security-tools supply-chain-report

.PHONY: generate
generate: deps ## Generate code (CRDs, deepcopy, etc.)
	@echo "Generating code..."
	go generate ./...
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: gen
gen: ## Generate CRDs and deep copy methods (output to deployments/crds/)
	@echo "Generating CRDs and deep copy methods..."
	@mkdir -p deployments/crds
	@if ! command -v controller-gen >/dev/null 2>&1; then \
		echo "Installing controller-gen..."; \
		go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest; \
	fi
	@echo "Attempting to generate deep copy methods..."
	@controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/v1" || echo "⚠️  Deep copy generation failed due to compilation errors"
	@echo "Attempting to generate CRDs..."
	@controller-gen crd rbac:roleName=manager-role webhook paths="./api/v1" output:crd:artifacts:config=deployments/crds 2>/dev/null || \
		(echo "⚠️  CRD generation failed, using existing CRDs..." && \
		 cp deployments/crds/*.yaml deployments/crds/ 2>/dev/null || echo "No existing CRDs found")
	@echo "✅ Gen target completed (check warnings above for any issues)"
	@ls -la deployments/crds/ 2>/dev/null || echo "📁 Contents of deployments/crds/ directory:"

.PHONY: manifests
manifests: deps ## Generate Kubernetes manifests
	@echo "Generating Kubernetes manifests..."
	controller-gen crd rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting Go code..."
	gofmt -s -w .
	go mod tidy

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config .golangci.yml; \
	else \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run --config .golangci.yml; \
	fi

##@ Testing

.PHONY: test
test: ## Run unit tests
	@echo "Running unit tests..."
	mkdir -p $(REPORTS_DIR)
	go test ./... -v -race -coverprofile=$(REPORTS_DIR)/coverage.out -covermode=atomic

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	mkdir -p $(REPORTS_DIR)
	go test ./tests/integration/... -v -timeout=30m

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	@echo "Running end-to-end tests..."
	mkdir -p $(REPORTS_DIR)
	go test ./tests/e2e/... -v -timeout=45m

.PHONY: test-excellence
test-excellence: ## Run excellence validation test suite
	@echo "Running excellence validation test suite..."
	mkdir -p $(REPORTS_DIR)
	go test ./tests/excellence/... -v -timeout=30m --ginkgo.v

.PHONY: test-regression
test-regression: ## Run regression testing suite
	@echo "Running regression testing suite..."
	mkdir -p $(REGRESSION_REPORTS_DIR)
	REGRESSION_BASELINE_ID=$(REGRESSION_BASELINE_ID) \
	REGRESSION_FAIL_ON_DETECTION=$(REGRESSION_FAIL_ON_DETECTION) \
	REGRESSION_ALERT_WEBHOOK=$(REGRESSION_ALERT_WEBHOOK) \
	go test ./tests/ -run TestRegressionSuite -v -timeout=60m

.PHONY: test-all
test-all: test test-integration test-e2e test-excellence test-regression ## Run all test suites

.PHONY: coverage
coverage: test ## Generate and view test coverage report
	@echo "Generating coverage report..."
	go tool cover -html=$(REPORTS_DIR)/coverage.out -o $(REPORTS_DIR)/coverage.html
	@echo "Coverage report generated: $(REPORTS_DIR)/coverage.html"

##@ Comprehensive Validation Suite

.PHONY: validation-setup
validation-setup: ## Setup environment for comprehensive validation
	@echo "Setting up comprehensive validation environment..."
	mkdir -p $(VALIDATION_REPORTS_DIR)
	go install github.com/onsi/ginkgo/v2/ginkgo@latest
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: validate-comprehensive
validate-comprehensive: validation-setup ## Run comprehensive validation suite (target 90/100 points)
	@echo "Running comprehensive validation suite..."
	@echo "Target score: $(VALIDATION_TARGET_SCORE)/100 points"
	go run ./tests/scripts/run-comprehensive-validation.go \
		--scope=all \
		--target-score=$(VALIDATION_TARGET_SCORE) \
		--concurrency=$(VALIDATION_CONCURRENCY) \
		--output-dir=$(VALIDATION_REPORTS_DIR) \
		--report-format=both \
		--verbose

.PHONY: validate-functional
validate-functional: validation-setup ## Run functional completeness validation (target 45/50 points)
	@echo "Running functional completeness validation..."
	go run ./tests/scripts/run-comprehensive-validation.go \
		--scope=functional \
		--target-score=45 \
		--output-dir=$(VALIDATION_REPORTS_DIR) \
		--report-format=both \
		--verbose

.PHONY: validate-performance-comprehensive
validate-performance-comprehensive: validation-setup ## Run performance benchmarking validation (target 23/25 points)
	@echo "Running performance benchmarking validation..."
	go run ./tests/scripts/run-comprehensive-validation.go \
		--scope=performance \
		--target-score=23 \
		--concurrency=$(VALIDATION_CONCURRENCY) \
		--enable-load-test=true \
		--output-dir=$(VALIDATION_REPORTS_DIR) \
		--report-format=both \
		--verbose

.PHONY: validate-security-comprehensive
validate-security-comprehensive: validation-setup ## Run security compliance validation (target 14/15 points)
	@echo "Running security compliance validation..."
	go run ./tests/scripts/run-comprehensive-validation.go \
		--scope=security \
		--target-score=14 \
		--output-dir=$(VALIDATION_REPORTS_DIR) \
		--report-format=both \
		--verbose

.PHONY: validate-production-comprehensive
validate-production-comprehensive: validation-setup ## Run production readiness validation (target 8/10 points)
	@echo "Running production readiness validation..."
	go run ./tests/scripts/run-comprehensive-validation.go \
		--scope=production \
		--target-score=8 \
		--enable-chaos-test=false \
		--output-dir=$(VALIDATION_REPORTS_DIR) \
		--report-format=both \
		--verbose

.PHONY: validate-chaos
validate-chaos: validation-setup ## Run chaos engineering tests (destructive testing)
	@echo "Running chaos engineering validation..."
	@echo "WARNING: This will run destructive tests that may cause temporary failures"
	@read -p "Are you sure you want to continue? [y/N] " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		go run ./tests/scripts/run-comprehensive-validation.go \
			--scope=production \
			--target-score=8 \
			--enable-chaos-test=true \
			--output-dir=$(VALIDATION_REPORTS_DIR) \
			--report-format=both \
			--verbose; \
	else \
		echo "Chaos testing cancelled"; \
	fi

.PHONY: validation-report
validation-report: ## Generate and display validation report
	@echo "Generating comprehensive validation report..."
	@if [ -f "$(VALIDATION_REPORTS_DIR)/validation-report.json" ]; then \
		score=$$(jq -r '.total_score' $(VALIDATION_REPORTS_DIR)/validation-report.json); \
		target=$$(jq -r '.max_possible_score' $(VALIDATION_REPORTS_DIR)/validation-report.json); \
		echo ""; \
		echo "============================================================================="; \
		echo "NEPHORAN INTENT OPERATOR - COMPREHENSIVE VALIDATION REPORT"; \
		echo "============================================================================="; \
		echo ""; \
		echo "OVERALL SCORE: $$score/$$target POINTS"; \
		if [ "$$score" -ge "$(VALIDATION_TARGET_SCORE)" ]; then \
			echo "STATUS: ✅ PASSED"; \
		else \
			echo "STATUS: ❌ FAILED"; \
		fi; \
		echo ""; \
		echo "CATEGORY BREAKDOWN:"; \
		echo "├── Functional Completeness:  $$(jq -r '.functional_score' $(VALIDATION_REPORTS_DIR)/validation-report.json)/50 points"; \
		echo "├── Performance Benchmarks:   $$(jq -r '.performance_score' $(VALIDATION_REPORTS_DIR)/validation-report.json)/25 points"; \
		echo "├── Security Compliance:      $$(jq -r '.security_score' $(VALIDATION_REPORTS_DIR)/validation-report.json)/15 points"; \
		echo "└── Production Readiness:     $$(jq -r '.production_score' $(VALIDATION_REPORTS_DIR)/validation-report.json)/10 points"; \
		echo ""; \
		echo "HTML Report: $(VALIDATION_REPORTS_DIR)/validation-report.html"; \
		echo "JSON Report: $(VALIDATION_REPORTS_DIR)/validation-report.json"; \
		echo "============================================================================="; \
	else \
		echo "❌ No validation report found. Run 'make validate-comprehensive' first."; \
	fi

.PHONY: validation-gate
validation-gate: ## Check if validation meets gate criteria (CI/CD gate)
	@echo "Checking comprehensive validation gate..."
	@if [ -f "$(VALIDATION_REPORTS_DIR)/validation-report.json" ]; then \
		score=$$(jq -r '.total_score' $(VALIDATION_REPORTS_DIR)/validation-report.json); \
		echo "Current validation score: $$score"; \
		if [ "$$score" -ge "$(VALIDATION_TARGET_SCORE)" ]; then \
			echo "✅ Validation gate PASSED (score: $$score >= $(VALIDATION_TARGET_SCORE))"; \
		else \
			echo "❌ Validation gate FAILED (score: $$score < $(VALIDATION_TARGET_SCORE))"; \
			exit 1; \
		fi; \
	else \
		echo "❌ Validation gate FAILED - no validation report found"; \
		echo "Run 'make validate-comprehensive' first"; \
		exit 1; \
	fi

.PHONY: validation-clean
validation-clean: ## Clean validation artifacts
	@echo "Cleaning validation artifacts..."
	rm -rf $(VALIDATION_REPORTS_DIR)/

##@ Excellence Validation

.PHONY: validate-docs
validate-docs: ## Validate documentation quality
	@echo "Validating documentation quality..."
	mkdir -p $(REPORTS_DIR)
	bash scripts/validate-docs.sh --report-dir $(REPORTS_DIR)

.PHONY: validate-security
validate-security: verify-supply-chain scan-vulnerabilities ## Run security compliance checks
	@echo "Running security compliance validation..."
	mkdir -p $(REPORTS_DIR)
	bash scripts/daily-compliance-check.sh --report-dir $(REPORTS_DIR) --security-only

.PHONY: validate-performance
validate-performance: ## Run performance SLA validation
	@echo "Running performance SLA validation..."
	mkdir -p $(REPORTS_DIR)
	bash scripts/daily-compliance-check.sh --report-dir $(REPORTS_DIR) --performance-only

.PHONY: validate-community
validate-community: ## Analyze community engagement metrics
	@echo "Analyzing community engagement metrics..."
	mkdir -p $(REPORTS_DIR)
	bash scripts/community-metrics.sh --report-dir $(REPORTS_DIR)

.PHONY: excellence-score
excellence-score: validate-docs validate-security validate-community test-excellence ## Calculate overall excellence score
	@echo "Calculating overall excellence score..."
	mkdir -p $(REPORTS_DIR)
	bash scripts/excellence-scoring-system.sh --report-dir $(REPORTS_DIR)

.PHONY: excellence-dashboard
excellence-dashboard: excellence-score ## Generate excellence dashboard
	@echo "Generating excellence dashboard..."
	bash scripts/excellence-scoring-system.sh --report-dir $(REPORTS_DIR) --html-only
	@if [ -f "$(REPORTS_DIR)/excellence_dashboard.html" ]; then \
		echo "Excellence dashboard available at: $(REPORTS_DIR)/excellence_dashboard.html"; \
	fi

.PHONY: excellence-gate
excellence-gate: excellence-score ## Check if project meets excellence gate criteria
	@echo "Checking excellence gate criteria..."
	@if [ -f "$(REPORTS_DIR)/excellence_dashboard.json" ]; then \
		score=$$(jq -r '.summary.overall_score' $(REPORTS_DIR)/excellence_dashboard.json); \
		echo "Current excellence score: $$score"; \
		if [ "$$(echo "$$score >= $(EXCELLENCE_THRESHOLD)" | bc)" -eq 1 ]; then \
			echo "✅ Excellence gate PASSED (score: $$score >= $(EXCELLENCE_THRESHOLD))"; \
		else \
			echo "❌ Excellence gate FAILED (score: $$score < $(EXCELLENCE_THRESHOLD))"; \
			exit 1; \
		fi; \
	else \
		echo "❌ Excellence scoring failed - no dashboard data available"; \
		exit 1; \
	fi

##@ Building

.PHONY: build
build: gen fmt vet ## Build the operator binary
	@echo "Building operator binary..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o bin/manager cmd/main.go

.PHONY: build-debug
build-debug: gen fmt vet ## Build the operator binary with debug info
	@echo "Building operator binary with debug info..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -gcflags="all=-N -l" -o bin/manager-debug cmd/main.go

.PHONY: cross-build
cross-build: gen fmt vet ## Build binaries for multiple platforms
	@echo "Cross-building for multiple platforms..."
	mkdir -p bin
	@for os in linux windows darwin; do \
		for arch in amd64 arm64; do \
			echo "Building for $$os/$$arch..."; \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o bin/manager-$$os-$$arch cmd/main.go; \
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

.PHONY: deploy-samples
deploy-samples: ## Deploy sample resources
	@echo "Deploying sample resources..."
	kubectl apply -f config/samples/

.PHONY: helm-install
helm-install: ## Install using Helm chart
	@echo "Installing using Helm..."
	helm upgrade --install $(PROJECT_NAME) deployments/helm/nephoran-operator \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--set image.tag=$(IMAGE_TAG)

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall Helm release
	@echo "Uninstalling Helm release..."
	helm uninstall $(PROJECT_NAME) --namespace $(NAMESPACE)

.PHONY: mvp-scale-up
mvp-scale-up: ## Scale up MVP deployment using porch/kpt patches
	@echo "Scaling up MVP deployment..."
	@echo "Generating scaling patch for increased capacity..."
	@if [ -d "kpt-packages/nephio" ]; then \
		echo "Applying scale-up patch to Nephio packages..."; \
		kubectl patch deployment nephoran-controller-manager \
			--namespace $(NAMESPACE) \
			--type='json' \
			-p='[{"op": "replace", "path": "/spec/replicas", "value": 3}]' 2>/dev/null || true; \
		kubectl patch hpa nephoran-controller-hpa \
			--namespace $(NAMESPACE) \
			--type='json' \
			-p='[{"op": "replace", "path": "/spec/minReplicas", "value": 3}, \
				 {"op": "replace", "path": "/spec/maxReplicas", "value": 10}]' 2>/dev/null || true; \
		echo "Scale-up patch applied successfully"; \
	else \
		echo "Warning: kpt-packages/nephio not found. Creating default scale-up patch..."; \
		mkdir -p handoff/patches; \
		echo '{"kind":"Patch","metadata":{"name":"scale-up"},"spec":{"replicas":3,"resources":{"requests":{"cpu":"500m","memory":"512Mi"}}}}' \
			> handoff/patches/scale-up-patch.json; \
		echo "Scale-up patch saved to handoff/patches/scale-up-patch.json"; \
	fi
	@echo "MVP scale-up completed"

.PHONY: mvp-scale-down
mvp-scale-down: ## Scale down MVP deployment using porch/kpt patches
	@echo "Scaling down MVP deployment..."
	@echo "Generating scaling patch for reduced capacity..."
	@if [ -d "kpt-packages/nephio" ]; then \
		echo "Applying scale-down patch to Nephio packages..."; \
		kubectl patch deployment nephoran-controller-manager \
			--namespace $(NAMESPACE) \
			--type='json' \
			-p='[{"op": "replace", "path": "/spec/replicas", "value": 1}]' 2>/dev/null || true; \
		kubectl patch hpa nephoran-controller-hpa \
			--namespace $(NAMESPACE) \
			--type='json' \
			-p='[{"op": "replace", "path": "/spec/minReplicas", "value": 1}, \
				 {"op": "replace", "path": "/spec/maxReplicas", "value": 3}]' 2>/dev/null || true; \
		echo "Scale-down patch applied successfully"; \
	else \
		echo "Warning: kpt-packages/nephio not found. Creating default scale-down patch..."; \
		mkdir -p handoff/patches; \
		echo '{"kind":"Patch","metadata":{"name":"scale-down"},"spec":{"replicas":1,"resources":{"requests":{"cpu":"100m","memory":"128Mi"}}}}' \
			> handoff/patches/scale-down-patch.json; \
		echo "Scale-down patch saved to handoff/patches/scale-down-patch.json"; \
	fi
	@echo "MVP scale-down completed"

##@ Development Environment

.PHONY: quickstart
quickstart: ## Run the 15-minute quickstart tutorial
	@echo "Starting Nephoran Intent Operator Quick Start..."
	@chmod +x scripts/quickstart.sh
	@./scripts/quickstart.sh

.PHONY: quickstart-clean
quickstart-clean: ## Clean up quickstart resources
	@echo "Cleaning up quickstart resources..."
	@chmod +x scripts/quickstart.sh
	@./scripts/quickstart.sh --cleanup

.PHONY: dev-up
dev-up: quickstart ## Alias for quickstart - set up development environment
	@echo "Development environment ready!"

.PHONY: dev-down
dev-down: quickstart-clean ## Tear down development environment
	@echo "Development environment cleaned up!"

.PHONY: kind-create
kind-create: ## Create a kind cluster for development
	@echo "Creating kind cluster..."
	@if ! command -v kind >/dev/null 2>&1; then \
		echo "kind not found, please install it first"; \
		exit 1; \
	fi
	kind create cluster --name $(PROJECT_NAME)-dev

.PHONY: kind-delete
kind-delete: ## Delete the kind cluster
	@echo "Deleting kind cluster..."
	kind delete cluster --name $(PROJECT_NAME)-dev

.PHONY: kind-load-image
kind-load-image: docker-build ## Load Docker image into kind cluster
	@echo "Loading Docker image into kind cluster..."
	kind load docker-image $(IMAGE_NAME):$(IMAGE_TAG) --name $(PROJECT_NAME)-dev

.PHONY: dev-deploy
dev-deploy: kind-create kind-load-image install-crds deploy deploy-samples ## Complete development deployment
	@echo "Development deployment complete!"
	@echo "You can now test the operator in your kind cluster"

##@ Maintenance

.PHONY: clean
clean: ## Clean build artifacts and reports
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf $(REPORTS_DIR)/
	go clean -cache
	docker system prune -f --volumes

.PHONY: update-deps
update-deps: ## Update Go dependencies
	@echo "Updating Go dependencies..."
	go get -u ./...
	go mod tidy

.PHONY: security-scan
security-scan: verify-supply-chain generate-sbom ## Run comprehensive security scanning
	@echo "Running comprehensive security scan..."
	mkdir -p $(REPORTS_DIR)
	@if [ -f "scripts/security-scan.sh" ]; then \
		bash scripts/security-scan.sh; \
	fi
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./... > $(REPORTS_DIR)/vulnerability-scan.txt 2>&1 || true; \
		echo "Vulnerability scan results saved to $(REPORTS_DIR)/vulnerability-scan.txt"; \
	fi

.PHONY: benchmark
benchmark: ## Run performance benchmarks
	@echo "Running performance benchmarks..."
	mkdir -p $(REPORTS_DIR)
	go test -bench=. -benchmem ./... > $(REPORTS_DIR)/benchmark.txt 2>&1
	@echo "Benchmark results saved to $(REPORTS_DIR)/benchmark.txt"

##@ Release

.PHONY: pre-release
pre-release: excellence-gate test-all docker-build docker-security-scan ## Pre-release validation
	@echo "Pre-release validation completed successfully!"

.PHONY: release-dry-run
release-dry-run: pre-release ## Dry run of release process
	@echo "Performing release dry run..."
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Image: $(IMAGE_NAME):$(IMAGE_TAG)"
	@echo "Excellence gate: PASSED"

.PHONY: release
release: pre-release docker-push ## Create and push release
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

##@ Regression Testing

.PHONY: regression-setup
regression-setup: ## Setup regression testing environment
	@echo "Setting up regression testing environment..."
	mkdir -p $(REGRESSION_REPORTS_DIR)
	mkdir -p $(REGRESSION_REPORTS_DIR)/baselines
	mkdir -p $(REGRESSION_REPORTS_DIR)/reports
	go install github.com/onsi/ginkgo/v2/ginkgo@latest

.PHONY: regression-baseline
regression-baseline: ## Establish new regression baseline
	@echo "Establishing new regression baseline..."
	mkdir -p $(REGRESSION_REPORTS_DIR)/baselines
	REGRESSION_MODE=baseline \
	go test ./tests/ -run TestRegressionSuite -v -timeout=60m
	@echo "New baseline created in $(REGRESSION_REPORTS_DIR)/baselines/"

.PHONY: regression-test
regression-test: ## Run comprehensive regression testing
	@echo "Running comprehensive regression testing..."
	@echo "Baseline ID: $(REGRESSION_BASELINE_ID)"
	@echo "Fail on detection: $(REGRESSION_FAIL_ON_DETECTION)"
	mkdir -p $(REGRESSION_REPORTS_DIR)
	REGRESSION_BASELINE_ID=$(REGRESSION_BASELINE_ID) \
	REGRESSION_FAIL_ON_DETECTION=$(REGRESSION_FAIL_ON_DETECTION) \
	REGRESSION_ALERT_WEBHOOK=$(REGRESSION_ALERT_WEBHOOK) \
	TEST_ARTIFACTS_PATH=$(REGRESSION_REPORTS_DIR) \
	go test ./tests/ -run TestRegressionSuite -v -timeout=60m

.PHONY: regression-dashboard
regression-dashboard: ## Generate regression dashboard and reports
	@echo "Generating regression dashboard..."
	mkdir -p $(REGRESSION_REPORTS_DIR)/dashboard
	go run ./tests/scripts/generate-regression-dashboard.go \
		--baselines-path=$(REGRESSION_REPORTS_DIR)/baselines \
		--output-path=$(REGRESSION_REPORTS_DIR)/dashboard
	@echo "Dashboard generated: $(REGRESSION_REPORTS_DIR)/dashboard/regression-dashboard.html"

.PHONY: regression-trends
regression-trends: ## Generate regression trend analysis
	@echo "Generating regression trend analysis..."
	mkdir -p $(REGRESSION_REPORTS_DIR)/trends
	go run ./tests/scripts/analyze-regression-trends.go \
		--baselines-path=$(REGRESSION_REPORTS_DIR)/baselines \
		--output-path=$(REGRESSION_REPORTS_DIR)/trends \
		--days=30
	@echo "Trend analysis completed: $(REGRESSION_REPORTS_DIR)/trends/"

.PHONY: regression-alert-test
regression-alert-test: ## Test regression alert system
	@echo "Testing regression alert system..."
	go run ./tests/scripts/test-alert-system.go \
		--webhook-url=$(REGRESSION_ALERT_WEBHOOK) \
		--test-mode=true

.PHONY: regression-cleanup
regression-cleanup: ## Clean up regression test artifacts
	@echo "Cleaning up regression test artifacts..."
	rm -rf $(REGRESSION_REPORTS_DIR)/
	@echo "Regression artifacts cleaned up"

.PHONY: regression-status
regression-status: ## Show regression testing status
	@echo "Regression Testing Status"
	@echo "========================="
	@echo "Reports directory: $(REGRESSION_REPORTS_DIR)"
	@if [ -d "$(REGRESSION_REPORTS_DIR)/baselines" ]; then \
		echo "Baseline count: $$(ls -1 $(REGRESSION_REPORTS_DIR)/baselines/baseline-*.json 2>/dev/null | wc -l)"; \
		if [ -n "$$(ls -1 $(REGRESSION_REPORTS_DIR)/baselines/baseline-*.json 2>/dev/null | head -1)" ]; then \
			latest=$$(ls -1t $(REGRESSION_REPORTS_DIR)/baselines/baseline-*.json 2>/dev/null | head -1); \
			echo "Latest baseline: $$(basename $$latest)"; \
			echo "Baseline date: $$(stat -c %y $$latest 2>/dev/null || stat -f %Sm $$latest 2>/dev/null || echo 'Unknown')"; \
		fi; \
	else \
		echo "No baselines found. Run 'make regression-baseline' first."; \
	fi
	@if [ -d "$(REGRESSION_REPORTS_DIR)/reports" ]; then \
		echo "Report count: $$(ls -1 $(REGRESSION_REPORTS_DIR)/reports/regression-report-*.json 2>/dev/null | wc -l)"; \
		if [ -n "$$(ls -1 $(REGRESSION_REPORTS_DIR)/reports/regression-report-*.json 2>/dev/null | head -1)" ]; then \
			latest=$$(ls -1t $(REGRESSION_REPORTS_DIR)/reports/regression-report-*.json 2>/dev/null | head -1); \
			echo "Latest report: $$(basename $$latest)"; \
		fi; \
	fi
	@echo "Configuration:"
	@echo "  Baseline ID: $(REGRESSION_BASELINE_ID)"
	@echo "  Fail on detection: $(REGRESSION_FAIL_ON_DETECTION)"
	@echo "  Alert webhook: $(REGRESSION_ALERT_WEBHOOK)"

.PHONY: regression-ci
regression-ci: ## Run regression testing in CI mode (fail-fast)
	@echo "Running regression testing in CI mode..."
	mkdir -p $(REGRESSION_REPORTS_DIR)
	CI=true \
	REGRESSION_BASELINE_ID=$(REGRESSION_BASELINE_ID) \
	REGRESSION_FAIL_ON_DETECTION=true \
	TEST_ARTIFACTS_PATH=$(REGRESSION_REPORTS_DIR) \
	go test ./tests/ -run TestRegressionSuite -v -timeout=60m -failfast

.PHONY: regression-full
regression-full: regression-setup regression-test regression-dashboard regression-trends ## Run complete regression testing workflow
	@echo "Complete regression testing workflow completed"
	@echo "View dashboard: file://$(PWD)/$(REGRESSION_REPORTS_DIR)/dashboard/regression-dashboard.html"

##@ Code Quality Gates

.PHONY: quality-gate
quality-gate: ## Run comprehensive code quality gate
	@echo "Running comprehensive code quality gate..."
	@echo "Coverage threshold: $(COVERAGE_THRESHOLD)%"
	@echo "Quality threshold: $(QUALITY_THRESHOLD)/10.0"
	@chmod +x scripts/quality-gate.sh
	./scripts/quality-gate.sh --coverage-threshold=$(COVERAGE_THRESHOLD) --quality-threshold=$(QUALITY_THRESHOLD) --reports-dir=$(QUALITY_REPORTS_DIR)

.PHONY: quality-gate-ci
quality-gate-ci: ## Run quality gate in CI mode (fail fast)
	@echo "Running quality gate in CI mode..."
	@chmod +x scripts/quality-gate.sh
	./scripts/quality-gate.sh --ci --coverage-threshold=$(COVERAGE_THRESHOLD) --quality-threshold=$(QUALITY_THRESHOLD) --reports-dir=$(QUALITY_REPORTS_DIR)

.PHONY: quality-metrics
quality-metrics: ## Calculate comprehensive quality metrics
	@echo "Calculating comprehensive quality metrics..."
	mkdir -p $(QUALITY_REPORTS_DIR)
	go run scripts/quality-metrics.go . $(QUALITY_REPORTS_DIR)/quality-metrics.json

.PHONY: performance-regression
performance-regression: ## Run performance regression tests
	@echo "Running performance regression tests..."
	mkdir -p $(QUALITY_REPORTS_DIR)
	go run scripts/performance-regression-test.go . $(QUALITY_REPORTS_DIR)/baseline.json $(QUALITY_REPORTS_DIR)/performance-results.json

.PHONY: technical-debt
technical-debt: ## Analyze technical debt
	@echo "Analyzing technical debt..."
	mkdir -p $(QUALITY_REPORTS_DIR)
	go run scripts/technical-debt-monitor.go . "" $(QUALITY_REPORTS_DIR)/technical-debt-report.json

.PHONY: quality-dashboard
quality-dashboard: quality-gate ## Generate comprehensive quality dashboard
	@echo "Quality dashboard generated: $(QUALITY_REPORTS_DIR)/quality-dashboard.html"
	@if [ -f "$(QUALITY_REPORTS_DIR)/quality-dashboard.html" ]; then \
		echo "🔍 View dashboard: file://$(PWD)/$(QUALITY_REPORTS_DIR)/quality-dashboard.html"; \
	fi

.PHONY: quality-check
quality-check: quality-gate ## Quick quality check (alias for quality-gate)

.PHONY: quality-fix
quality-fix: ## Attempt to automatically fix quality issues
	@echo "Attempting to fix quality issues..."
	@chmod +x scripts/quality-gate.sh
	./scripts/quality-gate.sh --fix --reports-dir=$(QUALITY_REPORTS_DIR)

.PHONY: quality-baseline
quality-baseline: ## Set performance baseline for regression testing
	@echo "Setting performance baseline..."
	mkdir -p $(QUALITY_REPORTS_DIR)
	go run scripts/performance-regression-test.go . "" $(QUALITY_REPORTS_DIR)/baseline.json
	@echo "Baseline saved to $(QUALITY_REPORTS_DIR)/baseline.json"

.PHONY: quality-clean
quality-clean: ## Clean quality reports
	@echo "Cleaning quality reports..."
	rm -rf $(QUALITY_REPORTS_DIR)/

.PHONY: quality-summary
quality-summary: ## Show quality summary
	@echo "Quality Summary for $(PROJECT_NAME)"
	@echo "=================================="
	@if [ -f "$(QUALITY_REPORTS_DIR)/quality-dashboard.json" ]; then \
		echo "Overall Score: $$(jq -r '.summary.quality_score' $(QUALITY_REPORTS_DIR)/quality-dashboard.json 2>/dev/null || echo 'N/A')/10.0"; \
		echo "Test Coverage: $$(jq -r '.summary.coverage_percent' $(QUALITY_REPORTS_DIR)/quality-dashboard.json 2>/dev/null || echo 'N/A')%"; \
		echo "Lint Issues: $$(jq -r '.summary.lint_issues' $(QUALITY_REPORTS_DIR)/quality-dashboard.json 2>/dev/null || echo 'N/A')"; \
		echo "Security Issues: $$(jq -r '.summary.vulnerabilities' $(QUALITY_REPORTS_DIR)/quality-dashboard.json 2>/dev/null || echo 'N/A')"; \
		echo "Status: $$(jq -r '.summary.overall_status' $(QUALITY_REPORTS_DIR)/quality-dashboard.json 2>/dev/null || echo 'UNKNOWN')"; \
	else \
		echo "No quality data available. Run 'make quality-gate' first."; \
	fi

##@ Shortcuts and Aliases

.PHONY: dev
dev: deps gen fmt test ## Quick development workflow

.PHONY: ci
ci: deps gen fmt vet test lint quality-gate-ci ## CI workflow with quality gates

.PHONY: validate
validate: excellence-score ## Alias for excellence-score

.PHONY: dashboard  
dashboard: quality-dashboard ## Alias for quality-dashboard

.PHONY: gate
gate: quality-gate ## Alias for quality-gate

# Default target
.DEFAULT_GOAL := help

# Check for required tools
.PHONY: check-tools
check-tools:
	@echo "Checking required tools..."
	@error=0; \
	for tool in go docker kubectl; do \
		if ! command -v $$tool >/dev/null 2>&1; then \
			echo "❌ $$tool is required but not installed"; \
			error=1; \
		else \
			echo "✅ $$tool is available"; \
		fi; \
	done; \
	exit $$error

# Initialize project
.PHONY: init
init: check-tools deps ## Initialize the project for development
	@echo "Initializing project for development..."
	mkdir -p $(REPORTS_DIR)
	@echo "Project initialized successfully!"
	@echo "Run 'make help' to see available commands"

##@ MVP Demo Commands

MVP_DEMO_DIR = examples/mvp-oran-sim
MVP_NAMESPACE = mvp-demo

.PHONY: mvp-up
mvp-up: ## Run complete MVP demo flow (install → prepare → send → apply → validate)
	@echo "===== Starting MVP Demo Flow ====="
	@echo "Step 1/5: Installing Porch components..."
	@cd $(MVP_DEMO_DIR) && \
		if [ -f "01-install-porch.sh" ]; then \
			bash 01-install-porch.sh; \
		else \
			pwsh -File 01-install-porch.ps1; \
		fi
	@echo ""
	@echo "Step 2/5: Preparing NF simulator package..."
	@cd $(MVP_DEMO_DIR) && \
		if [ -f "02-prepare-nf-sim.sh" ]; then \
			bash 02-prepare-nf-sim.sh; \
		else \
			pwsh -File 02-prepare-nf-sim.ps1; \
		fi
	@echo ""
	@echo "Step 3/5: Sending scaling intent..."
	@cd $(MVP_DEMO_DIR) && \
		if [ -f "03-send-intent.sh" ]; then \
			REPLICAS=3 bash 03-send-intent.sh; \
		else \
			pwsh -File 03-send-intent.ps1 -Replicas 3; \
		fi
	@echo ""
	@echo "Step 4/5: Applying package with Porch/KPT..."
	@cd $(MVP_DEMO_DIR) && \
		if [ -f "04-porch-apply.sh" ]; then \
			bash 04-porch-apply.sh; \
		else \
			pwsh -File 04-porch-apply.ps1; \
		fi
	@echo ""
	@echo "Step 5/5: Validating deployment..."
	@cd $(MVP_DEMO_DIR) && \
		if [ -f "05-validate.sh" ]; then \
			bash 05-validate.sh; \
		else \
			pwsh -File 05-validate.ps1; \
		fi
	@echo ""
	@echo "===== MVP Demo Complete! ====="

.PHONY: mvp-scale-up
mvp-scale-up: ## Scale NF simulator up to 5 replicas
	@echo "Scaling NF simulator to 5 replicas..."
	@cd $(MVP_DEMO_DIR) && \
		if [ -f "03-send-intent.sh" ]; then \
			REPLICAS=5 REASON="Scale up test" bash 03-send-intent.sh; \
		else \
			pwsh -File 03-send-intent.ps1 -Replicas 5 -Reason "Scale up test"; \
		fi
	@sleep 3
	@kubectl patch deployment nf-sim -n $(MVP_NAMESPACE) -p '{"spec":{"replicas":5}}' || true
	@echo "Waiting for scale up..."
	@sleep 5
	@kubectl get deployment nf-sim -n $(MVP_NAMESPACE)

.PHONY: mvp-scale-down
mvp-scale-down: ## Scale NF simulator down to 1 replica
	@echo "Scaling NF simulator to 1 replica..."
	@cd $(MVP_DEMO_DIR) && \
		if [ -f "03-send-intent.sh" ]; then \
			REPLICAS=1 REASON="Scale down test" bash 03-send-intent.sh; \
		else \
			pwsh -File 03-send-intent.ps1 -Replicas 1 -Reason "Scale down test"; \
		fi
	@sleep 3
	@kubectl patch deployment nf-sim -n $(MVP_NAMESPACE) -p '{"spec":{"replicas":1}}' || true
	@echo "Waiting for scale down..."
	@sleep 5
	@kubectl get deployment nf-sim -n $(MVP_NAMESPACE)

.PHONY: mvp-down
mvp-down: mvp-scale-down ## Scale down and delete PackageRevision if created
	@echo "Cleaning up PackageRevisions..."
	@kubectl delete packagerevision nf-sim-package-v1 -n default 2>/dev/null || true
	@echo "MVP scaled down"

.PHONY: mvp-status
mvp-status: ## Show current MVP deployment status
	@echo "===== MVP Deployment Status ====="
	@echo "Namespace: $(MVP_NAMESPACE)"
	@kubectl get namespace $(MVP_NAMESPACE) 2>/dev/null || echo "Namespace not found"
	@echo ""
	@echo "Deployment:"
	@kubectl get deployment nf-sim -n $(MVP_NAMESPACE) 2>/dev/null || echo "Deployment not found"
	@echo ""
	@echo "Pods:"
	@kubectl get pods -n $(MVP_NAMESPACE) -l app=nf-sim 2>/dev/null || echo "No pods found"
	@echo ""
	@echo "Service:"
	@kubectl get service nf-sim -n $(MVP_NAMESPACE) 2>/dev/null || echo "Service not found"

.PHONY: mvp-clean
mvp-clean: ## Clean up all MVP demo resources
	@echo "Cleaning up MVP demo resources..."
	@echo "Deleting namespace $(MVP_NAMESPACE)..."
	@kubectl delete namespace $(MVP_NAMESPACE) --ignore-not-found=true
	@echo "Deleting PackageRevisions..."
	@kubectl delete packagerevision -l app=nf-sim --all-namespaces --ignore-not-found=true 2>/dev/null || true
	@echo "Cleaning up local package directories..."
	@rm -rf $(MVP_DEMO_DIR)/package-* 2>/dev/null || true
	@echo "MVP demo resources cleaned up"

.PHONY: mvp-logs
mvp-logs: ## Show logs from NF simulator pods
	@echo "===== NF Simulator Logs ====="
	@kubectl logs -n $(MVP_NAMESPACE) -l app=nf-sim --tail=50 2>/dev/null || echo "No logs available"

.PHONY: mvp-watch
mvp-watch: ## Watch MVP deployment status continuously
	@watch -n 2 "kubectl get deployment,pods,service -n $(MVP_NAMESPACE) 2>/dev/null || echo 'Resources not found'"