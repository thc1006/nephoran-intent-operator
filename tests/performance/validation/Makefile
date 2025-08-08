# Nephoran Intent Operator Performance Validation Suite Makefile

.PHONY: help install test test-quick test-comprehensive test-ci validate-config trends regression report clean lint

# Default target
help: ## Show this help message
	@echo "Nephoran Intent Operator Performance Validation Suite"
	@echo "===================================================="
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Installation and setup
install: ## Install dependencies and setup validation suite
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Creating output directories..."
	mkdir -p test-results/validation
	mkdir -p test-data/archive
	@echo "Generating default configuration..."
	go run cmd.go config init validation-config.json
	@echo "Installation complete!"

# Testing targets
test: ## Run full performance validation suite (30-60 minutes)
	@echo "Running comprehensive performance validation..."
	go test -v ./... -timeout=90m -tags=validation

test-quick: ## Run quick validation for development (5 minutes)
	@echo "Running quick validation..."
	go run cmd.go run --preset quick --timeout 5m

test-comprehensive: ## Run comprehensive validation with maximum rigor (60+ minutes)
	@echo "Running comprehensive validation..."
	go run cmd.go run --preset comprehensive --timeout 120m

test-ci: ## Run CI-optimized validation (10 minutes)
	@echo "Running CI validation..."
	go run cmd.go run --preset ci --timeout 15m --ci-mode

test-stress: ## Run stress testing validation
	@echo "Running stress testing..."
	go run cmd.go run --preset stress --timeout 45m

test-regression: ## Run regression-focused validation
	@echo "Running regression testing..."
	go run cmd.go run --preset regression --timeout 30m

# Specific claim validation
test-latency: ## Test intent processing latency claim only
	go run cmd.go run --claims intent_latency_p95 --timeout 15m

test-concurrency: ## Test concurrent capacity claim only
	go run cmd.go run --claims concurrent_capacity --timeout 20m

test-throughput: ## Test throughput claim only
	go run cmd.go run --claims throughput_rate --timeout 25m

test-availability: ## Test system availability claim only
	go run cmd.go run --claims system_availability --timeout 30m

test-rag-latency: ## Test RAG retrieval latency claim only
	go run cmd.go run --claims rag_retrieval_latency_p95 --timeout 15m

test-cache-hit-rate: ## Test cache hit rate claim only
	go run cmd.go run --claims cache_hit_rate --timeout 20m

# Configuration management
validate-config: ## Validate validation configuration file
	@if [ -f validation-config.json ]; then \
		go run cmd.go config validate validation-config.json; \
	else \
		echo "Configuration file not found. Run 'make install' first."; \
		exit 1; \
	fi

config-presets: ## List available configuration presets
	go run cmd.go config presets

init-config: ## Initialize configuration template
	go run cmd.go config init validation-config.json

# Analysis and reporting
trends: ## Analyze performance trends (default: intent_latency_p95, last 30 days)
	go run cmd.go trends --claim intent_latency_p95 --days 30

trends-custom: ## Analyze trends for custom claim and timeframe
	@read -p "Enter claim name: " claim; \
	read -p "Enter number of days: " days; \
	go run cmd.go trends --claim $$claim --days $$days

regression: ## Detect performance regressions
	go run cmd.go regression --lookback-days 7

regression-alert: ## Run regression detection with automatic alerting
	go run cmd.go regression --auto-alert --lookback-days 7

report: ## Generate comprehensive validation report
	@if [ -z "$(RUN_ID)" ]; then \
		echo "Usage: make report RUN_ID=<run-id>"; \
		echo "Available runs:"; \
		ls test-results/validation/ | grep -E "val-[0-9]+" | head -5; \
	else \
		go run cmd.go report --run-id $(RUN_ID) --format html; \
	fi

report-latest: ## Generate report for latest validation run
	@latest_run=$$(ls -t test-results/validation/ | grep -E "val-[0-9]+" | head -1); \
	if [ -n "$$latest_run" ]; then \
		go run cmd.go report --run-id $$latest_run --format html; \
	else \
		echo "No validation runs found. Run a validation first."; \
	fi

# Data management
data-cleanup: ## Perform data cleanup based on retention policy
	go run cmd.go data cleanup

data-archive: ## Archive validation data
	go run cmd.go data archive

data-verify: ## Verify data integrity
	go run cmd.go data verify

baseline-create: ## Create new performance baseline
	go run cmd.go baseline create

baseline-list: ## List available performance baselines
	go run cmd.go baseline list

baseline-compare: ## Compare current results with baseline
	go run cmd.go baseline compare

# Development and testing
lint: ## Run linting and code quality checks
	@echo "Running linting..."
	go vet ./...
	go fmt ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

test-unit: ## Run unit tests for the validation framework
	go test -v ./... -short -timeout=5m

test-integration: ## Run integration tests
	go test -v ./... -tags=integration -timeout=30m

benchmark: ## Run benchmarks for the validation framework
	go test -bench=. -benchmem ./...

coverage: ## Generate test coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Monitoring and maintenance
monitor: ## Monitor validation runs (requires running validation)
	@echo "Monitoring validation progress..."
	@while true; do \
		if pgrep -f "validation" > /dev/null; then \
			echo "$$(date): Validation running..."; \
		else \
			echo "$$(date): No validation running"; \
		fi; \
		sleep 30; \
	done

status: ## Show validation environment status
	@echo "Validation Environment Status"
	@echo "============================="
	@echo "Go version: $$(go version)"
	@echo "Working directory: $$(pwd)"
	@echo "Test results directory: $$(ls -la test-results/validation 2>/dev/null | wc -l) files"
	@echo "Archive directory: $$(ls -la test-data/archive 2>/dev/null | wc -l) files"
	@echo "Configuration: $$(if [ -f validation-config.json ]; then echo 'Present'; else echo 'Missing'; fi)"
	@echo ""
	@echo "Latest validation runs:"
	@ls -t test-results/validation/ 2>/dev/null | head -5 || echo "No runs found"

# Environment-specific targets
dev: test-quick ## Run development validation (alias for test-quick)

staging: ## Run staging environment validation
	go run cmd.go run --environment staging --timeout 20m

production: ## Run production environment validation
	go run cmd.go run --environment production --timeout 60m

# CI/CD integration helpers
ci-validate: ## Validate for CI/CD pipeline with quality gates
	@echo "Running CI validation with quality gates..."
	go run cmd.go run --preset ci --timeout 15m --ci-mode
	@if [ $$? -eq 0 ]; then \
		echo "✅ CI validation passed"; \
	else \
		echo "❌ CI validation failed"; \
		exit 1; \
	fi

ci-regression-check: ## Check for regressions in CI/CD
	go run cmd.go regression --lookback-days 3 --severity-filter critical
	@if [ $$? -eq 0 ]; then \
		echo "✅ No critical regressions detected"; \
	else \
		echo "❌ Critical regressions detected"; \
		exit 1; \
	fi

# Cleanup
clean: ## Clean up test results and temporary files
	@echo "Cleaning up test results and temporary files..."
	rm -rf test-results/validation/*
	rm -f *.out *.html
	rm -f coverage.out coverage.html
	@echo "Cleanup complete"

clean-all: clean ## Clean everything including archives and baselines
	@echo "Cleaning all data including archives..."
	rm -rf test-data/archive/*
	rm -rf test-data/baselines/*
	@echo "Complete cleanup finished"

# Quick reference
quick-reference: ## Show quick reference for common commands
	@echo "Nephoran Validation Quick Reference"
	@echo "================================="
	@echo ""
	@echo "🚀 Quick Start:"
	@echo "  make install          # Setup environment"
	@echo "  make test-quick       # 5-minute validation"
	@echo "  make test-ci          # CI/CD validation"
	@echo ""
	@echo "🧪 Comprehensive Testing:"
	@echo "  make test             # Full validation suite"
	@echo "  make test-comprehensive # Maximum rigor"
	@echo "  make test-stress      # Stress testing"
	@echo ""
	@echo "📊 Analysis:"
	@echo "  make trends           # Performance trends"
	@echo "  make regression       # Detect regressions"
	@echo "  make report-latest    # Generate report"
	@echo ""
	@echo "🔧 Maintenance:"
	@echo "  make data-cleanup     # Clean old data"
	@echo "  make status           # Environment status"
	@echo "  make clean            # Clean results"

# Help with examples
examples: ## Show usage examples
	@echo "Nephoran Validation Usage Examples"
	@echo "=================================="
	@echo ""
	@echo "1. Quick development validation:"
	@echo "   make test-quick"
	@echo ""
	@echo "2. Test specific performance claim:"
	@echo "   make test-latency"
	@echo "   make test-concurrency"
	@echo ""
	@echo "3. Full production validation:"
	@echo "   make production"
	@echo ""
	@echo "4. CI/CD integration:"
	@echo "   make ci-validate && make ci-regression-check"
	@echo ""
	@echo "5. Analyze trends for custom timeframe:"
	@echo "   go run cmd.go trends --claim throughput_rate --days 60"
	@echo ""
	@echo "6. Custom validation with overrides:"
	@echo "   go run cmd.go run --confidence 99 --min-samples 100 --timeout 45m"
	@echo ""
	@echo "7. Generate custom report:"
	@echo "   make report RUN_ID=val-20240108-143022"