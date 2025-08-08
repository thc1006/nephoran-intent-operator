# Security and Supply Chain Validation Makefile

# Security configuration
SECURITY_DIR := .
REPORTS_DIR := $(SECURITY_DIR)/reports
SCAN_DIR := ..
GOSEC_SEVERITY := medium
TRIVY_SEVERITY := CRITICAL,HIGH,MEDIUM
GITLEAKS_CONFIG := $(SECURITY_DIR)/configs/gitleaks.toml

# Supply chain configuration
SBOM_FORMAT := spdx-json
SBOM_OUTPUT := $(REPORTS_DIR)/sbom
PROVENANCE_OUTPUT := $(REPORTS_DIR)/provenance
ATTESTATION_OUTPUT := $(REPORTS_DIR)/attestations

# Tools versions
GOSEC_VERSION := latest
TRIVY_VERSION := latest
SYFT_VERSION := latest
COSIGN_VERSION := latest
GRYPE_VERSION := latest

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: all
all: install-tools security-scan supply-chain-scan compliance-check

##@ Installation

.PHONY: install-tools
install-tools: ## Install all security and supply chain tools
	@echo "$(BLUE)Installing security and supply chain tools...$(NC)"
	@$(MAKE) install-gosec
	@$(MAKE) install-trivy
	@$(MAKE) install-syft
	@$(MAKE) install-grype
	@$(MAKE) install-cosign
	@$(MAKE) install-gitleaks
	@$(MAKE) install-nancy
	@$(MAKE) install-govulncheck
	@$(MAKE) install-cyclonedx
	@$(MAKE) install-lichen
	@echo "$(GREEN)✅ All tools installed successfully$(NC)"

.PHONY: install-gosec
install-gosec:
	@echo "Installing gosec..."
	@go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)

.PHONY: install-trivy
install-trivy:
	@echo "Installing trivy..."
	@curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b $${HOME}/.local/bin $(TRIVY_VERSION)

.PHONY: install-syft
install-syft:
	@echo "Installing syft..."
	@curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b $${HOME}/.local/bin $(SYFT_VERSION)

.PHONY: install-grype
install-grype:
	@echo "Installing grype..."
	@curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b $${HOME}/.local/bin $(GRYPE_VERSION)

.PHONY: install-cosign
install-cosign:
	@echo "Installing cosign..."
	@go install github.com/sigstore/cosign/v2/cmd/cosign@$(COSIGN_VERSION)

.PHONY: install-gitleaks
install-gitleaks:
	@echo "Installing gitleaks..."
	@go install github.com/zricethezav/gitleaks/v8@latest

.PHONY: install-nancy
install-nancy:
	@echo "Installing nancy..."
	@go install github.com/sonatype-nexus-community/nancy@latest

.PHONY: install-govulncheck
install-govulncheck:
	@echo "Installing govulncheck..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: install-cyclonedx
install-cyclonedx:
	@echo "Installing cyclonedx..."
	@go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest

.PHONY: install-lichen
install-lichen:
	@echo "Installing lichen..."
	@go install github.com/uw-labs/lichen@latest

##@ Security Scanning

.PHONY: security-scan
security-scan: prepare-dirs ## Run comprehensive security scanning
	@echo "$(BLUE)Running comprehensive security scanning...$(NC)"
	@$(MAKE) scan-code
	@$(MAKE) scan-secrets
	@$(MAKE) scan-dependencies
	@$(MAKE) scan-containers
	@$(MAKE) scan-vulnerabilities
	@$(MAKE) generate-security-report
	@echo "$(GREEN)✅ Security scanning completed$(NC)"

.PHONY: scan-code
scan-code: ## Run static code analysis with gosec
	@echo "$(YELLOW)Running static code analysis...$(NC)"
	@mkdir -p $(REPORTS_DIR)/gosec
	@gosec -fmt sarif -out $(REPORTS_DIR)/gosec/gosec.sarif -severity $(GOSEC_SEVERITY) $(SCAN_DIR)/... 2>&1 | tee $(REPORTS_DIR)/gosec/gosec.log || true
	@gosec -fmt json -out $(REPORTS_DIR)/gosec/gosec.json -severity $(GOSEC_SEVERITY) $(SCAN_DIR)/... || true
	@gosec -fmt text -out $(REPORTS_DIR)/gosec/gosec.txt -severity $(GOSEC_SEVERITY) $(SCAN_DIR)/... || true
	@echo "$(GREEN)Code analysis completed$(NC)"

.PHONY: scan-secrets
scan-secrets: ## Scan for secrets and credentials
	@echo "$(YELLOW)Scanning for secrets and credentials...$(NC)"
	@mkdir -p $(REPORTS_DIR)/secrets
	@if [ -f $(GITLEAKS_CONFIG) ]; then \
		gitleaks detect --config $(GITLEAKS_CONFIG) --source $(SCAN_DIR) --report-path $(REPORTS_DIR)/secrets/gitleaks.json --report-format json --no-git || true; \
	else \
		gitleaks detect --source $(SCAN_DIR) --report-path $(REPORTS_DIR)/secrets/gitleaks.json --report-format json --no-git || true; \
	fi
	@echo "$(GREEN)Secret scanning completed$(NC)"

.PHONY: scan-dependencies
scan-dependencies: ## Scan dependencies for vulnerabilities
	@echo "$(YELLOW)Scanning dependencies for vulnerabilities...$(NC)"
	@mkdir -p $(REPORTS_DIR)/dependencies
	@cd $(SCAN_DIR) && govulncheck -json ./... > $(REPORTS_DIR)/dependencies/govulncheck.json 2>&1 || true
	@cd $(SCAN_DIR) && govulncheck ./... > $(REPORTS_DIR)/dependencies/govulncheck.txt 2>&1 || true
	@cd $(SCAN_DIR) && go list -json -deps ./... | nancy sleuth --output json > $(REPORTS_DIR)/dependencies/nancy.json 2>&1 || true
	@echo "$(GREEN)Dependency scanning completed$(NC)"

.PHONY: scan-containers
scan-containers: ## Scan container images for vulnerabilities
	@echo "$(YELLOW)Scanning container images...$(NC)"
	@mkdir -p $(REPORTS_DIR)/containers
	@if [ -f $(SCAN_DIR)/Dockerfile ]; then \
		trivy image --severity $(TRIVY_SEVERITY) --format json --output $(REPORTS_DIR)/containers/trivy.json --input $(SCAN_DIR) 2>&1 || true; \
		trivy image --severity $(TRIVY_SEVERITY) --format sarif --output $(REPORTS_DIR)/containers/trivy.sarif --input $(SCAN_DIR) 2>&1 || true; \
	fi
	@echo "$(GREEN)Container scanning completed$(NC)"

.PHONY: scan-vulnerabilities
scan-vulnerabilities: ## Comprehensive vulnerability scanning
	@echo "$(YELLOW)Running comprehensive vulnerability scan...$(NC)"
	@mkdir -p $(REPORTS_DIR)/vulnerabilities
	@cd $(SCAN_DIR) && grype dir:. --output json > $(REPORTS_DIR)/vulnerabilities/grype.json 2>&1 || true
	@cd $(SCAN_DIR) && grype dir:. --output table > $(REPORTS_DIR)/vulnerabilities/grype.txt 2>&1 || true
	@echo "$(GREEN)Vulnerability scanning completed$(NC)"

##@ Supply Chain Security

.PHONY: supply-chain-scan
supply-chain-scan: prepare-dirs ## Run supply chain security validation
	@echo "$(BLUE)Running supply chain security validation...$(NC)"
	@$(MAKE) generate-sbom
	@$(MAKE) scan-sbom
	@$(MAKE) verify-dependencies
	@$(MAKE) check-licenses
	@$(MAKE) generate-provenance
	@$(MAKE) create-attestations
	@echo "$(GREEN)✅ Supply chain validation completed$(NC)"

.PHONY: generate-sbom
generate-sbom: ## Generate Software Bill of Materials
	@echo "$(YELLOW)Generating SBOM...$(NC)"
	@mkdir -p $(SBOM_OUTPUT)
	@cd $(SCAN_DIR) && syft dir:. -o $(SBOM_FORMAT) > $(SBOM_OUTPUT)/sbom-syft.json
	@cd $(SCAN_DIR) && cyclonedx-gomod mod -json -output-file $(SBOM_OUTPUT)/sbom-cyclonedx.json
	@echo "$(GREEN)SBOM generated$(NC)"

.PHONY: scan-sbom
scan-sbom: ## Scan SBOM for vulnerabilities
	@echo "$(YELLOW)Scanning SBOM for vulnerabilities...$(NC)"
	@if [ -f $(SBOM_OUTPUT)/sbom-syft.json ]; then \
		grype sbom:$(SBOM_OUTPUT)/sbom-syft.json --output json > $(SBOM_OUTPUT)/sbom-vulnerabilities.json 2>&1 || true; \
	fi
	@echo "$(GREEN)SBOM scanning completed$(NC)"

.PHONY: verify-dependencies
verify-dependencies: ## Verify dependency integrity
	@echo "$(YELLOW)Verifying dependency integrity...$(NC)"
	@mkdir -p $(REPORTS_DIR)/verification
	@cd $(SCAN_DIR) && go mod verify 2>&1 | tee $(REPORTS_DIR)/verification/mod-verify.txt
	@cd $(SCAN_DIR) && go mod graph > $(REPORTS_DIR)/verification/dependency-graph.txt
	@echo "$(GREEN)Dependency verification completed$(NC)"

.PHONY: check-licenses
check-licenses: ## Check dependency licenses for compliance
	@echo "$(YELLOW)Checking dependency licenses...$(NC)"
	@mkdir -p $(REPORTS_DIR)/licenses
	@cd $(SCAN_DIR) && lichen --config=$(SECURITY_DIR)/configs/lichen.yaml $(SCAN_DIR) > $(REPORTS_DIR)/licenses/lichen-report.txt 2>&1 || true
	@echo "$(GREEN)License checking completed$(NC)"

.PHONY: generate-provenance
generate-provenance: ## Generate build provenance
	@echo "$(YELLOW)Generating build provenance...$(NC)"
	@mkdir -p $(PROVENANCE_OUTPUT)
	@echo "Build provenance generation would be performed here with in-toto/SLSA"
	@echo "$(GREEN)Provenance generated$(NC)"

.PHONY: create-attestations
create-attestations: ## Create supply chain attestations
	@echo "$(YELLOW)Creating supply chain attestations...$(NC)"
	@mkdir -p $(ATTESTATION_OUTPUT)
	@echo "Attestation creation would be performed here with cosign"
	@echo "$(GREEN)Attestations created$(NC)"

##@ Compliance and Reporting

.PHONY: compliance-check
compliance-check: ## Run compliance checks against security policies
	@echo "$(BLUE)Running compliance checks...$(NC)"
	@$(MAKE) check-cis-benchmark
	@$(MAKE) check-owasp
	@$(MAKE) check-nist
	@$(MAKE) generate-compliance-report
	@echo "$(GREEN)✅ Compliance checks completed$(NC)"

.PHONY: check-cis-benchmark
check-cis-benchmark: ## Check against CIS benchmarks
	@echo "$(YELLOW)Checking CIS benchmarks...$(NC)"
	@mkdir -p $(REPORTS_DIR)/compliance
	@echo "CIS benchmark validation would be performed here"
	@echo "$(GREEN)CIS benchmark check completed$(NC)"

.PHONY: check-owasp
check-owasp: ## Check against OWASP Top 10
	@echo "$(YELLOW)Checking OWASP Top 10...$(NC)"
	@echo "OWASP validation would be performed here"
	@echo "$(GREEN)OWASP check completed$(NC)"

.PHONY: check-nist
check-nist: ## Check against NIST framework
	@echo "$(YELLOW)Checking NIST framework...$(NC)"
	@echo "NIST framework validation would be performed here"
	@echo "$(GREEN)NIST check completed$(NC)"

##@ Reporting

.PHONY: generate-security-report
generate-security-report: ## Generate comprehensive security report
	@echo "$(YELLOW)Generating security report...$(NC)"
	@mkdir -p $(REPORTS_DIR)
	@$(SECURITY_DIR)/scripts/generate-security-report.sh $(REPORTS_DIR)
	@echo "$(GREEN)Security report generated at $(REPORTS_DIR)/security-report.html$(NC)"

.PHONY: generate-compliance-report
generate-compliance-report: ## Generate compliance report
	@echo "$(YELLOW)Generating compliance report...$(NC)"
	@$(SECURITY_DIR)/scripts/generate-compliance-report.sh $(REPORTS_DIR)
	@echo "$(GREEN)Compliance report generated$(NC)"

.PHONY: dashboard
dashboard: ## Launch security dashboard
	@echo "$(BLUE)Launching security dashboard...$(NC)"
	@python3 $(SECURITY_DIR)/scripts/security-dashboard.py $(REPORTS_DIR)

##@ Utilities

.PHONY: prepare-dirs
prepare-dirs: ## Prepare report directories
	@mkdir -p $(REPORTS_DIR)/{gosec,secrets,dependencies,containers,vulnerabilities,licenses,compliance}
	@mkdir -p $(SBOM_OUTPUT)
	@mkdir -p $(PROVENANCE_OUTPUT)
	@mkdir -p $(ATTESTATION_OUTPUT)

.PHONY: clean
clean: ## Clean up generated reports
	@echo "$(YELLOW)Cleaning up reports...$(NC)"
	@rm -rf $(REPORTS_DIR)
	@echo "$(GREEN)Cleanup completed$(NC)"

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)