# Makefile for Nephoran Intent Operator mTLS Security Testing

# Configuration
PROJECT_ROOT := $(shell cd ../.. && pwd)
TESTS_DIR := $(CURDIR)
REPORTS_DIR := $(TESTS_DIR)/reports
LOG_DIR := $(TESTS_DIR)/logs
BIN_DIR := $(PROJECT_ROOT)/bin

# Test configuration
GINKGO_PARALLEL ?= 4
GINKGO_TIMEOUT ?= 30m
TEST_NAMESPACE ?= nephoran-security-tests
VERBOSE ?= false

# Tools
GINKGO := $(BIN_DIR)/ginkgo
KUBECTL := kubectl

# Colors
GREEN := \033[0;32m
RED := \033[0;31m
BLUE := \033[0;34m
YELLOW := \033[1;33m
NC := \033[0m

.PHONY: help
help: ## Display this help message
	@echo "Nephoran Intent Operator - mTLS Security Testing"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: setup
setup: ## Setup test environment and dependencies
	@echo -e "$(BLUE)[INFO]$(NC) Setting up security test environment..."
	@mkdir -p $(REPORTS_DIR) $(LOG_DIR)
	@$(MAKE) install-tools
	@$(MAKE) setup-namespace
	@echo -e "$(GREEN)[SUCCESS]$(NC) Test environment ready"

.PHONY: install-tools
install-tools: ## Install required testing tools
	@echo -e "$(BLUE)[INFO]$(NC) Installing testing tools..."
	@mkdir -p $(BIN_DIR)
	@if [ ! -f $(GINKGO) ]; then \
		echo -e "$(YELLOW)[INFO]$(NC) Installing Ginkgo..."; \
		cd $(PROJECT_ROOT) && go install github.com/onsi/ginkgo/v2/ginkgo@latest; \
		cp $$(go env GOPATH)/bin/ginkgo $(GINKGO); \
	fi
	@echo -e "$(GREEN)[SUCCESS]$(NC) Tools installed"

.PHONY: setup-namespace
setup-namespace: ## Create test namespace
	@echo -e "$(BLUE)[INFO]$(NC) Setting up test namespace: $(TEST_NAMESPACE)"
	@$(KUBECTL) create namespace $(TEST_NAMESPACE) --dry-run=client -o yaml | $(KUBECTL) apply -f -
	@$(KUBECTL) label namespace $(TEST_NAMESPACE) security-testing=true --overwrite

.PHONY: test
test: test-all ## Run all security tests (alias for test-all)

.PHONY: test-all
test-all: setup ## Run complete security test suite
	@echo -e "$(BLUE)[INFO]$(NC) Running complete security test suite..."
	@$(MAKE) test-security
	@$(MAKE) test-integration
	@$(MAKE) test-compliance
	@echo -e "$(GREEN)[SUCCESS]$(NC) All security tests completed"

.PHONY: test-security
test-security: setup ## Run mTLS security tests
	@echo -e "$(BLUE)[INFO]$(NC) Running mTLS security tests..."
	@cd $(TESTS_DIR) && $(GINKGO) \
		--junit-report=$(REPORTS_DIR)/security-test-report.xml \
		--timeout=$(GINKGO_TIMEOUT) \
		-p --procs=$(GINKGO_PARALLEL) \
		$(if $(filter true,$(VERBOSE)),-v) \
		./mtls_security_test.go
	@echo -e "$(GREEN)[SUCCESS]$(NC) Security tests completed"

.PHONY: test-integration
test-integration: setup ## Run mTLS integration tests
	@echo -e "$(BLUE)[INFO]$(NC) Running mTLS integration tests..."
	@cd $(TESTS_DIR) && $(GINKGO) \
		--junit-report=$(REPORTS_DIR)/integration-test-report.xml \
		--timeout=$(GINKGO_TIMEOUT) \
		-p --procs=$(GINKGO_PARALLEL) \
		$(if $(filter true,$(VERBOSE)),-v) \
		./mtls_integration_test.go
	@echo -e "$(GREEN)[SUCCESS]$(NC) Integration tests completed"

.PHONY: test-performance
test-performance: setup ## Run mTLS performance tests
	@echo -e "$(BLUE)[INFO]$(NC) Running mTLS performance tests..."
	@cd $(TESTS_DIR) && $(GINKGO) \
		--junit-report=$(REPORTS_DIR)/performance-test-report.xml \
		--timeout=45m \
		-p --procs=$(GINKGO_PARALLEL) \
		$(if $(filter true,$(VERBOSE)),-v) \
		./mtls_performance_test.go
	@$(MAKE) benchmark
	@echo -e "$(GREEN)[SUCCESS]$(NC) Performance tests completed"

.PHONY: test-compliance
test-compliance: setup ## Run compliance validation tests
	@echo -e "$(BLUE)[INFO]$(NC) Running compliance validation tests..."
	@cd $(TESTS_DIR) && $(GINKGO) \
		--junit-report=$(REPORTS_DIR)/compliance-test-report.xml \
		--timeout=$(GINKGO_TIMEOUT) \
		-p --procs=$(GINKGO_PARALLEL) \
		$(if $(filter true,$(VERBOSE)),-v) \
		./mtls_compliance_test.go
	@$(MAKE) compliance-report
	@echo -e "$(GREEN)[SUCCESS]$(NC) Compliance tests completed"

.PHONY: test-chaos
test-chaos: setup ## Run chaos engineering tests
	@echo -e "$(BLUE)[INFO]$(NC) Running chaos engineering tests..."
	@echo -e "$(YELLOW)[WARNING]$(NC) Chaos tests will inject failures - ensure this is a test environment"
	@cd $(TESTS_DIR) && $(GINKGO) \
		--junit-report=$(REPORTS_DIR)/chaos-test-report.xml \
		--timeout=60m \
		-p --procs=$(GINKGO_PARALLEL) \
		$(if $(filter true,$(VERBOSE)),-v) \
		./mtls_chaos_test.go
	@echo -e "$(GREEN)[SUCCESS]$(NC) Chaos engineering tests completed"

.PHONY: benchmark
benchmark: setup ## Run performance benchmarks
	@echo -e "$(BLUE)[INFO]$(NC) Running performance benchmarks..."
	@cd $(TESTS_DIR) && go test -bench=. -benchmem -count=3 -timeout=30m \
		./mtls_performance_test.go \
		> $(REPORTS_DIR)/benchmark-results.txt 2> $(LOG_DIR)/benchmarks.log
	@echo -e "$(GREEN)[SUCCESS]$(NC) Benchmarks completed - results in $(REPORTS_DIR)/benchmark-results.txt"

.PHONY: test-quick
test-quick: setup ## Run essential security tests quickly
	@echo -e "$(BLUE)[INFO]$(NC) Running quick security validation..."
	@cd $(TESTS_DIR) && $(GINKGO) \
		--focus="Certificate Validation|mTLS Handshake|Compliance" \
		--junit-report=$(REPORTS_DIR)/quick-test-report.xml \
		--timeout=10m \
		-p --procs=$(GINKGO_PARALLEL) \
		./mtls_security_test.go ./mtls_compliance_test.go
	@echo -e "$(GREEN)[SUCCESS]$(NC) Quick security tests completed"

.PHONY: test-comprehensive
test-comprehensive: setup ## Run comprehensive test suite including slow tests
	@echo -e "$(BLUE)[INFO]$(NC) Running comprehensive security test suite..."
	@$(MAKE) test-security VERBOSE=true
	@$(MAKE) test-integration VERBOSE=true  
	@$(MAKE) test-performance VERBOSE=true
	@$(MAKE) test-compliance VERBOSE=true
	@$(MAKE) test-chaos VERBOSE=true
	@$(MAKE) generate-reports
	@echo -e "$(GREEN)[SUCCESS]$(NC) Comprehensive testing completed"

.PHONY: compliance-report
compliance-report: ## Generate compliance summary report
	@echo -e "$(BLUE)[INFO]$(NC) Generating compliance report..."
	@$(TESTS_DIR)/generate_compliance_report.sh
	@echo -e "$(GREEN)[SUCCESS]$(NC) Compliance report generated"

.PHONY: generate-reports
generate-reports: ## Generate consolidated test reports
	@echo -e "$(BLUE)[INFO]$(NC) Generating consolidated reports..."
	@$(TESTS_DIR)/generate_test_summary.sh
	@echo -e "$(GREEN)[SUCCESS]$(NC) Reports generated in $(REPORTS_DIR)"

.PHONY: clean
clean: ## Clean test artifacts and reports
	@echo -e "$(BLUE)[INFO]$(NC) Cleaning test artifacts..."
	@rm -rf $(REPORTS_DIR)/* $(LOG_DIR)/*
	@find $(TESTS_DIR) -name "*.log" -delete
	@echo -e "$(GREEN)[SUCCESS]$(NC) Cleanup completed"

.PHONY: clean-namespace
clean-namespace: ## Delete test namespace and resources
	@echo -e "$(BLUE)[INFO]$(NC) Cleaning test namespace: $(TEST_NAMESPACE)"
	@$(KUBECTL) delete namespace $(TEST_NAMESPACE) --ignore-not-found=true
	@echo -e "$(GREEN)[SUCCESS]$(NC) Test namespace cleaned"

.PHONY: deep-clean
deep-clean: clean clean-namespace ## Deep clean all test resources
	@echo -e "$(GREEN)[SUCCESS]$(NC) Deep clean completed"

.PHONY: lint
lint: ## Run linting on test files
	@echo -e "$(BLUE)[INFO]$(NC) Linting test files..."
	@cd $(PROJECT_ROOT) && golangci-lint run ./tests/security/...
	@echo -e "$(GREEN)[SUCCESS]$(NC) Linting completed"

.PHONY: format
format: ## Format test files
	@echo -e "$(BLUE)[INFO]$(NC) Formatting test files..."
	@cd $(TESTS_DIR) && gofmt -w -s *.go
	@echo -e "$(GREEN)[SUCCESS]$(NC) Formatting completed"

.PHONY: validate
validate: setup ## Validate test configuration
	@echo -e "$(BLUE)[INFO]$(NC) Validating test configuration..."
	@if [ ! -f $(TESTS_DIR)/test-config.yaml ]; then \
		echo -e "$(RED)[ERROR]$(NC) test-config.yaml not found"; \
		exit 1; \
	fi
	@$(KUBECTL) cluster-info > /dev/null || (echo -e "$(RED)[ERROR]$(NC) Cannot connect to Kubernetes cluster" && exit 1)
	@echo -e "$(GREEN)[SUCCESS]$(NC) Configuration validation passed"

.PHONY: status
status: ## Show test environment status
	@echo -e "$(BLUE)[INFO]$(NC) Test Environment Status"
	@echo "=================================="
	@echo "Project Root: $(PROJECT_ROOT)"
	@echo "Tests Directory: $(TESTS_DIR)"
	@echo "Reports Directory: $(REPORTS_DIR)"
	@echo "Log Directory: $(LOG_DIR)"
	@echo "Test Namespace: $(TEST_NAMESPACE)"
	@echo ""
	@echo "Kubernetes Context: $$($(KUBECTL) config current-context 2>/dev/null || echo 'Not configured')"
	@echo "Cluster Status: $$($(KUBECTL) cluster-info --request-timeout=5s > /dev/null 2>&1 && echo 'Connected' || echo 'Disconnected')"
	@echo ""
	@echo "Tool Status:"
	@echo "  Ginkgo: $$($(GINKGO) version 2>/dev/null || echo 'Not installed')"
	@echo "  Kubectl: $$($(KUBECTL) version --client --short 2>/dev/null || echo 'Not installed')"
	@echo ""
	@echo "Recent Reports:"
	@ls -la $(REPORTS_DIR)/ 2>/dev/null || echo "  No reports found"

.PHONY: watch
watch: setup ## Watch for changes and run tests continuously
	@echo -e "$(BLUE)[INFO]$(NC) Watching for changes..."
	@cd $(TESTS_DIR) && $(GINKGO) watch \
		--focus="Certificate Validation" \
		-p --procs=2 \
		./mtls_security_test.go

.PHONY: debug
debug: setup ## Run tests in debug mode with verbose output
	@echo -e "$(BLUE)[INFO]$(NC) Running tests in debug mode..."
	@cd $(TESTS_DIR) && $(GINKGO) \
		-v --trace \
		--focus="$(FOCUS)" \
		./mtls_security_test.go

.PHONY: ci
ci: setup ## Run tests suitable for CI environment
	@echo -e "$(BLUE)[INFO]$(NC) Running CI test suite..."
	@$(MAKE) test-security GINKGO_PARALLEL=2 GINKGO_TIMEOUT=20m
	@$(MAKE) test-compliance GINKGO_PARALLEL=2 GINKGO_TIMEOUT=15m
	@echo -e "$(GREEN)[SUCCESS]$(NC) CI tests completed"

# Test targets with specific focus patterns
.PHONY: test-handshake
test-handshake: setup ## Test mTLS handshake scenarios
	@cd $(TESTS_DIR) && $(GINKGO) --focus="Handshake" ./mtls_security_test.go

.PHONY: test-certificates
test-certificates: setup ## Test certificate validation
	@cd $(TESTS_DIR) && $(GINKGO) --focus="Certificate" ./mtls_security_test.go

.PHONY: test-attacks
test-attacks: setup ## Test attack simulation scenarios  
	@cd $(TESTS_DIR) && $(GINKGO) --focus="Attack" ./mtls_security_test.go

.PHONY: test-oran
test-oran: setup ## Test O-RAN compliance requirements
	@cd $(TESTS_DIR) && $(GINKGO) --focus="O-RAN" ./mtls_compliance_test.go

.PHONY: test-nist
test-nist: setup ## Test NIST compliance requirements
	@cd $(TESTS_DIR) && $(GINKGO) --focus="NIST" ./mtls_compliance_test.go

.PHONY: test-tls13
test-tls13: setup ## Test TLS 1.3 compliance
	@cd $(TESTS_DIR) && $(GINKGO) --focus="TLS.*1.3" ./mtls_compliance_test.go

# Utility targets
.PHONY: logs
logs: ## Show recent test logs
	@echo -e "$(BLUE)[INFO]$(NC) Recent test logs:"
	@find $(LOG_DIR) -name "*.log" -mtime -1 -exec echo "=== {} ===" \; -exec tail -20 {} \;

.PHONY: reports
reports: ## Show available reports
	@echo -e "$(BLUE)[INFO]$(NC) Available reports:"
	@ls -la $(REPORTS_DIR)/

.PHONY: open-reports
open-reports: ## Open HTML reports in browser
	@if command -v xdg-open > /dev/null; then \
		find $(REPORTS_DIR) -name "*.html" -exec xdg-open {} \; ; \
	elif command -v open > /dev/null; then \
		find $(REPORTS_DIR) -name "*.html" -exec open {} \; ; \
	else \
		echo -e "$(YELLOW)[INFO]$(NC) HTML reports available in $(REPORTS_DIR)"; \
	fi

# Default target
.DEFAULT_GOAL := help