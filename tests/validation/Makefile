# Makefile for Performance Testing Suite
# Target: 23/25 points for performance benchmarks

.PHONY: all test performance-test latency-test throughput-test scalability-test resource-test load-test comprehensive-test report clean help

# Variables
GINKGO := ginkgo
GO := go
K6 := k6
VEGETA := vegeta
TEST_TIMEOUT := 30m
LOAD_DURATION := 5m
TARGET_RPS := 45
CONCURRENCY := 200

# Color output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

# Default target
all: comprehensive-test

# Install dependencies
deps:
	@echo "$(GREEN)Installing test dependencies...$(NC)"
	go install github.com/onsi/ginkgo/v2/ginkgo@latest
	go install github.com/tsenart/vegeta@latest
	go get -u github.com/montanaflynn/stats
	go get -u github.com/prometheus/client_golang/prometheus
	@echo "$(GREEN)Dependencies installed successfully$(NC)"

# Run comprehensive performance test suite (23 points target)
comprehensive-test:
	@echo "$(GREEN)========================================$(NC)"
	@echo "$(GREEN)Running Comprehensive Performance Tests$(NC)"
	@echo "$(GREEN)Target: 23/25 points$(NC)"
	@echo "$(GREEN)========================================$(NC)"
	@$(GINKGO) -v --timeout=$(TEST_TIMEOUT) \
		--label-filter="performance" \
		--json-report=performance-report.json \
		--junit-report=performance-junit.xml \
		. -- -test.run="Performance"

# Latency testing (8 points)
latency-test:
	@echo "$(YELLOW)Running Latency Performance Tests (8 points)$(NC)"
	@echo "Target: P95 < 2s, P99 < 5s"
	@$(GO) test -v -timeout=$(TEST_TIMEOUT) \
		-run="TestLatencyPerformance" \
		-bench="BenchmarkLatency" \
		-benchtime=1m \
		./...

# Throughput testing (8 points)
throughput-test:
	@echo "$(YELLOW)Running Throughput Performance Tests (8 points)$(NC)"
	@echo "Target: >= 45 intents/minute"
	@$(GO) test -v -timeout=$(TEST_TIMEOUT) \
		-run="TestThroughputPerformance" \
		-bench="BenchmarkThroughput" \
		-benchtime=2m \
		./...

# Scalability testing (5 points)
scalability-test:
	@echo "$(YELLOW)Running Scalability Tests (5 points)$(NC)"
	@echo "Target: 200+ concurrent operations"
	@$(GO) test -v -timeout=$(TEST_TIMEOUT) \
		-run="TestScalability" \
		-bench="BenchmarkScalability" \
		-benchtime=3m \
		-parallel=$(CONCURRENCY) \
		./...

# Resource efficiency testing (2 points)
resource-test:
	@echo "$(YELLOW)Running Resource Efficiency Tests (2 points)$(NC)"
	@echo "Target: < 4GB memory, < 2 CPU cores"
	@$(GO) test -v -timeout=$(TEST_TIMEOUT) \
		-run="TestResourceEfficiency" \
		-bench="BenchmarkResource" \
		-benchmem \
		-cpuprofile=cpu.prof \
		-memprofile=mem.prof \
		./...

# Load testing with different patterns
load-test: load-constant load-ramp load-spike load-realistic

load-constant:
	@echo "$(YELLOW)Running Constant Load Test$(NC)"
	@$(GO) test -v -timeout=$(LOAD_DURATION) \
		-run="TestConstantLoad" \
		-args -rate=$(TARGET_RPS) \
		./...

load-ramp:
	@echo "$(YELLOW)Running Ramp-Up Load Test$(NC)"
	@$(GO) test -v -timeout=$(LOAD_DURATION) \
		-run="TestRampUpLoad" \
		-args -start-rate=10 -end-rate=100 \
		./...

load-spike:
	@echo "$(YELLOW)Running Spike Load Test$(NC)"
	@$(GO) test -v -timeout=$(LOAD_DURATION) \
		-run="TestSpikeLoad" \
		-args -base-rate=30 -spike-rate=200 \
		./...

load-realistic:
	@echo "$(YELLOW)Running Realistic Telecom Load Test$(NC)"
	@$(GO) test -v -timeout=15m \
		-run="TestRealisticTelecomLoad" \
		./...

# Vegeta load testing
vegeta-test:
	@echo "$(YELLOW)Running Vegeta Load Test$(NC)"
	@echo "GET http://localhost:8080/api/v1/intents" | \
		$(VEGETA) attack -duration=$(LOAD_DURATION) -rate=$(TARGET_RPS)/s | \
		$(VEGETA) report -type=text

# K6 load testing (if k6 scripts are available)
k6-test:
	@echo "$(YELLOW)Running K6 Load Test$(NC)"
	@if [ -f k6-performance-test.js ]; then \
		$(K6) run --duration=$(LOAD_DURATION) --vus=$(CONCURRENCY) k6-performance-test.js; \
	else \
		echo "$(RED)K6 test script not found$(NC)"; \
	fi

# Component-specific tests
component-test: test-llm test-porch test-multicluster test-oran

test-llm:
	@echo "$(YELLOW)Testing LLM/RAG Pipeline Performance$(NC)"
	@$(GO) test -v -run="TestLLMPipeline" -bench="BenchmarkLLM" ./...

test-porch:
	@echo "$(YELLOW)Testing Porch Package Management Performance$(NC)"
	@$(GO) test -v -run="TestPorchPerformance" -bench="BenchmarkPorch" ./...

test-multicluster:
	@echo "$(YELLOW)Testing Multi-Cluster Propagation Performance$(NC)"
	@$(GO) test -v -run="TestMultiCluster" -bench="BenchmarkPropagation" ./...

test-oran:
	@echo "$(YELLOW)Testing O-RAN Interface Performance$(NC)"
	@$(GO) test -v -run="TestORANInterfaces" -bench="BenchmarkORAN" ./...

# Performance profiling
profile:
	@echo "$(GREEN)Running Performance Profiling$(NC)"
	@$(GO) test -v -run="TestPerformance" \
		-cpuprofile=cpu.prof \
		-memprofile=mem.prof \
		-blockprofile=block.prof \
		-mutexprofile=mutex.prof \
		./...
	@echo "$(GREEN)Profiles generated. Use 'go tool pprof' to analyze$(NC)"

# Analyze CPU profile
analyze-cpu:
	@$(GO) tool pprof -http=:8080 cpu.prof

# Analyze memory profile
analyze-mem:
	@$(GO) tool pprof -http=:8081 mem.prof

# Generate performance report
report:
	@echo "$(GREEN)Generating Performance Report$(NC)"
	@$(GO) run ../scripts/generate-performance-report.go \
		-input=performance-report.json \
		-output=performance-report.html
	@echo "$(GREEN)Report generated: performance-report.html$(NC)"

# Performance regression check
regression-check:
	@echo "$(YELLOW)Checking for Performance Regressions$(NC)"
	@$(GO) test -v -run="TestRegressionDetection" ./...
	@if [ $$? -eq 0 ]; then \
		echo "$(GREEN)No performance regressions detected$(NC)"; \
	else \
		echo "$(RED)Performance regression detected!$(NC)"; \
		exit 1; \
	fi

# Continuous performance monitoring
monitor:
	@echo "$(GREEN)Starting Performance Monitoring$(NC)"
	@while true; do \
		$(MAKE) quick-test; \
		sleep 300; \
	done

# Quick performance test (for CI/CD)
quick-test:
	@echo "$(YELLOW)Running Quick Performance Test$(NC)"
	@$(GO) test -v -timeout=5m \
		-run="TestQuickPerformance" \
		-short \
		./...

# Benchmark comparison
benchmark-compare:
	@echo "$(YELLOW)Comparing Benchmarks$(NC)"
	@if [ -f old.bench ]; then \
		$(GO) test -bench=. -benchmem ./... > new.bench; \
		benchstat old.bench new.bench; \
		mv new.bench old.bench; \
	else \
		$(GO) test -bench=. -benchmem ./... > old.bench; \
		echo "$(GREEN)Baseline benchmark saved$(NC)"; \
	fi

# Export metrics to Prometheus
export-metrics:
	@echo "$(YELLOW)Exporting Metrics to Prometheus$(NC)"
	@$(GO) test -v -run="TestMetricsExport" ./...

# Generate Grafana dashboard
generate-dashboard:
	@echo "$(YELLOW)Generating Grafana Dashboard$(NC)"
	@$(GO) run ../scripts/generate-dashboard.go \
		-output=performance-dashboard.json

# Clean up test artifacts
clean:
	@echo "$(YELLOW)Cleaning up test artifacts$(NC)"
	@rm -f *.prof *.bench *.json *.xml *.html
	@rm -f performance-report.* performance-baseline.*
	@echo "$(GREEN)Cleanup complete$(NC)"

# Help target
help:
	@echo "$(GREEN)Performance Testing Suite - Available Targets$(NC)"
	@echo ""
	@echo "Main Targets:"
	@echo "  $(YELLOW)comprehensive-test$(NC)  - Run full performance test suite (23/25 points target)"
	@echo "  $(YELLOW)latency-test$(NC)        - Test latency performance (8 points)"
	@echo "  $(YELLOW)throughput-test$(NC)     - Test throughput performance (8 points)"
	@echo "  $(YELLOW)scalability-test$(NC)    - Test scalability (5 points)"
	@echo "  $(YELLOW)resource-test$(NC)       - Test resource efficiency (2 points)"
	@echo ""
	@echo "Load Testing:"
	@echo "  $(YELLOW)load-test$(NC)           - Run all load test scenarios"
	@echo "  $(YELLOW)vegeta-test$(NC)         - Run Vegeta load test"
	@echo "  $(YELLOW)k6-test$(NC)             - Run K6 load test"
	@echo ""
	@echo "Component Testing:"
	@echo "  $(YELLOW)component-test$(NC)      - Test all components"
	@echo "  $(YELLOW)test-llm$(NC)            - Test LLM/RAG pipeline"
	@echo "  $(YELLOW)test-porch$(NC)          - Test Porch package management"
	@echo "  $(YELLOW)test-multicluster$(NC)   - Test multi-cluster propagation"
	@echo "  $(YELLOW)test-oran$(NC)           - Test O-RAN interfaces"
	@echo ""
	@echo "Analysis & Monitoring:"
	@echo "  $(YELLOW)profile$(NC)             - Generate performance profiles"
	@echo "  $(YELLOW)analyze-cpu$(NC)         - Analyze CPU profile"
	@echo "  $(YELLOW)analyze-mem$(NC)         - Analyze memory profile"
	@echo "  $(YELLOW)report$(NC)              - Generate HTML report"
	@echo "  $(YELLOW)regression-check$(NC)    - Check for performance regressions"
	@echo "  $(YELLOW)monitor$(NC)             - Continuous performance monitoring"
	@echo ""
	@echo "Utilities:"
	@echo "  $(YELLOW)deps$(NC)                - Install test dependencies"
	@echo "  $(YELLOW)clean$(NC)               - Clean up test artifacts"
	@echo "  $(YELLOW)help$(NC)                - Show this help message"

# Performance validation for CI/CD
ci-test: deps quick-test regression-check
	@echo "$(GREEN)CI Performance Tests Completed$(NC)"

# Full validation suite integration
validate: comprehensive-test report
	@echo "$(GREEN)========================================$(NC)"
	@echo "$(GREEN)Performance Validation Complete$(NC)"
	@echo "$(GREEN)Check performance-report.html for details$(NC)"
	@echo "$(GREEN)========================================$(NC)"

.DEFAULT_GOAL := help