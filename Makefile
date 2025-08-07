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

# Kubernetes configuration
NAMESPACE ?= nephoran-system
KUBECONFIG ?= ~/.kube/config

# Excellence framework configuration
REPORTS_DIR = .excellence-reports
EXCELLENCE_THRESHOLD = 75

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
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping..."; \
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

.PHONY: test-all
test-all: test test-integration test-e2e test-excellence ## Run all test suites

.PHONY: coverage
coverage: test ## Generate and view test coverage report
	@echo "Generating coverage report..."
	go tool cover -html=$(REPORTS_DIR)/coverage.out -o $(REPORTS_DIR)/coverage.html
	@echo "Coverage report generated: $(REPORTS_DIR)/coverage.html"

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
build: generate manifests fmt vet ## Build the operator binary
	@echo "Building operator binary..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o bin/manager cmd/main.go

.PHONY: build-debug
build-debug: generate manifests fmt vet ## Build the operator binary with debug info
	@echo "Building operator binary with debug info..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -gcflags="all=-N -l" -o bin/manager-debug cmd/main.go

.PHONY: cross-build
cross-build: generate manifests fmt vet ## Build binaries for multiple platforms
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
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_NAME):latest

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

##@ Shortcuts and Aliases

.PHONY: dev
dev: deps generate manifests fmt test ## Quick development workflow

.PHONY: ci
ci: deps generate manifests fmt vet test lint ## CI workflow

.PHONY: validate
validate: excellence-score ## Alias for excellence-score

.PHONY: dashboard  
dashboard: excellence-dashboard ## Alias for excellence-dashboard

.PHONY: gate
gate: excellence-gate ## Alias for excellence-gate

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