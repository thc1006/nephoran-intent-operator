# Makefile for CRD Reference Documentation

.PHONY: help install build serve deploy clean validate version test all

# Variables
PYTHON := python3
PIP := pip
MKDOCS := mkdocs
VENV := venv
SITE_DIR := site
DOCS_DIR := docs
PORT := 8000

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m # No Color

# Default target
help: ## Show this help message
	@echo "Nephoran Intent Operator - CRD Documentation"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "Usage: make [target]"

all: install validate build ## Run all steps: install, validate, and build

# Environment setup
venv: ## Create Python virtual environment
	@echo "$(YELLOW)Creating virtual environment...$(NC)"
	@$(PYTHON) -m venv $(VENV)
	@echo "$(GREEN)Virtual environment created$(NC)"

install: venv ## Install documentation dependencies
	@echo "$(YELLOW)Installing dependencies...$(NC)"
	@. $(VENV)/bin/activate && $(PIP) install --upgrade pip
	@. $(VENV)/bin/activate && $(PIP) install -r requirements.txt
	@echo "$(GREEN)Dependencies installed$(NC)"

# Documentation building
build: ## Build documentation site
	@echo "$(YELLOW)Building documentation...$(NC)"
	@. $(VENV)/bin/activate && $(MKDOCS) build --strict
	@echo "$(GREEN)Documentation built in $(SITE_DIR)/$(NC)"

serve: ## Serve documentation locally (default: http://localhost:8000)
	@echo "$(YELLOW)Starting documentation server...$(NC)"
	@echo "$(GREEN)Documentation available at http://localhost:$(PORT)$(NC)"
	@. $(VENV)/bin/activate && $(MKDOCS) serve --dev-addr localhost:$(PORT)

serve-all: ## Serve documentation on all interfaces
	@echo "$(YELLOW)Starting documentation server on all interfaces...$(NC)"
	@echo "$(GREEN)Documentation available at http://0.0.0.0:$(PORT)$(NC)"
	@. $(VENV)/bin/activate && $(MKDOCS) serve --dev-addr 0.0.0.0:$(PORT)

# Deployment
deploy: validate build ## Deploy documentation to GitHub Pages
	@echo "$(YELLOW)Deploying to GitHub Pages...$(NC)"
	@. $(VENV)/bin/activate && $(MKDOCS) gh-deploy --force
	@echo "$(GREEN)Documentation deployed$(NC)"

deploy-version: validate build ## Deploy versioned documentation
	@echo "$(YELLOW)Enter version (e.g., v1.0.0):$(NC)"
	@read VERSION; \
	. $(VENV)/bin/activate && mike deploy $$VERSION latest --push
	@echo "$(GREEN)Version $$VERSION deployed$(NC)"

# Validation and testing
validate: ## Validate documentation files
	@echo "$(YELLOW)Validating documentation...$(NC)"
	@echo "Checking YAML syntax in examples..."
	@find $(DOCS_DIR) -name '*.md' -exec grep -l '```yaml' {} \; | while read file; do \
		echo "  Checking $$file"; \
	done
	@echo "Checking for broken internal links..."
	@. $(VENV)/bin/activate && $(MKDOCS) build --strict 2>&1 | grep -E "WARNING|ERROR" || true
	@echo "$(GREEN)Validation complete$(NC)"

test: build ## Test built documentation
	@echo "$(YELLOW)Testing documentation...$(NC)"
	@echo "Checking for broken links..."
	@command -v linkchecker >/dev/null 2>&1 || (echo "Installing linkchecker..." && pip install linkchecker)
	@linkchecker $(SITE_DIR)/index.html --no-warnings --ignore-url="^mailto:" || true
	@echo "$(GREEN)Testing complete$(NC)"

lint: ## Lint markdown files
	@echo "$(YELLOW)Linting markdown files...$(NC)"
	@command -v markdownlint >/dev/null 2>&1 || npm install -g markdownlint-cli
	@markdownlint $(DOCS_DIR) --disable MD013 MD033 || true
	@echo "$(GREEN)Linting complete$(NC)"

# CRD generation
generate-crds: ## Generate CRD documentation from Go types
	@echo "$(YELLOW)Generating CRD documentation...$(NC)"
	@command -v controller-gen >/dev/null 2>&1 || (echo "$(RED)controller-gen not found. Install with: go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest$(NC)" && exit 1)
	@cd ../.. && controller-gen crd:maxDescLen=0 paths="./api/v1/..." output:crd:artifacts:config=docs/crd-reference/generated/crds
	@echo "$(GREEN)CRD documentation generated$(NC)"

# Utility targets
clean: ## Clean generated files
	@echo "$(YELLOW)Cleaning generated files...$(NC)"
	@rm -rf $(SITE_DIR)
	@rm -rf .cache
	@find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	@echo "$(GREEN)Cleaned$(NC)"

clean-all: clean ## Clean everything including virtual environment
	@echo "$(YELLOW)Cleaning everything...$(NC)"
	@rm -rf $(VENV)
	@rm -rf generated/
	@echo "$(GREEN)All cleaned$(NC)"

version: ## Show MkDocs and theme versions
	@echo "$(YELLOW)Version information:$(NC)"
	@. $(VENV)/bin/activate && $(MKDOCS) --version
	@. $(VENV)/bin/activate && $(PYTHON) -c "import mkdocs_material; print(f'Material theme: {mkdocs_material.__version__}')"
	@. $(VENV)/bin/activate && $(PIP) list | grep mkdocs

stats: ## Show documentation statistics
	@echo "$(YELLOW)Documentation statistics:$(NC)"
	@echo "Markdown files: $$(find $(DOCS_DIR) -name '*.md' | wc -l)"
	@echo "Total lines: $$(find $(DOCS_DIR) -name '*.md' -exec wc -l {} + | tail -1 | awk '{print $$1}')"
	@echo "YAML examples: $$(find $(DOCS_DIR) -name '*.md' -exec grep -c '```yaml' {} + | awk '{sum+=$$1} END {print sum}')"
	@echo "Images: $$(find $(DOCS_DIR) -name '*.png' -o -name '*.jpg' -o -name '*.svg' | wc -l)"

# Development helpers
watch: ## Watch for changes and auto-rebuild
	@echo "$(YELLOW)Watching for changes...$(NC)"
	@. $(VENV)/bin/activate && $(MKDOCS) serve --dirtyreload

new-crd: ## Create template for new CRD documentation
	@echo "$(YELLOW)Enter CRD name (lowercase):$(NC)"
	@read CRD_NAME; \
	mkdir -p $(DOCS_DIR)/crds/$$CRD_NAME; \
	cp templates/crd-template.md $(DOCS_DIR)/crds/$$CRD_NAME/overview.md; \
	cp templates/crd-spec-template.md $(DOCS_DIR)/crds/$$CRD_NAME/spec.md; \
	cp templates/crd-examples-template.md $(DOCS_DIR)/crds/$$CRD_NAME/examples.md; \
	echo "$(GREEN)Created templates in $(DOCS_DIR)/crds/$$CRD_NAME/$(NC)"

update-deps: ## Update all dependencies to latest versions
	@echo "$(YELLOW)Updating dependencies...$(NC)"
	@. $(VENV)/bin/activate && $(PIP) install --upgrade pip
	@. $(VENV)/bin/activate && $(PIP) list --outdated
	@. $(VENV)/bin/activate && $(PIP) install --upgrade -r requirements.txt
	@echo "$(GREEN)Dependencies updated$(NC)"

# Docker support
docker-build: ## Build documentation in Docker
	@echo "$(YELLOW)Building Docker image...$(NC)"
	@docker build -t nephoran-docs .
	@docker run --rm -v $(PWD):/docs nephoran-docs build
	@echo "$(GREEN)Documentation built in Docker$(NC)"

docker-serve: ## Serve documentation using Docker
	@echo "$(YELLOW)Starting Docker container...$(NC)"
	@docker run --rm -p $(PORT):8000 -v $(PWD):/docs nephoran-docs serve --dev-addr 0.0.0.0:8000

# CI/CD helpers
ci-build: install validate build test ## CI pipeline build
	@echo "$(GREEN)CI build complete$(NC)"

ci-deploy: ci-build deploy ## CI pipeline deploy
	@echo "$(GREEN)CI deployment complete$(NC)"

# Print configuration
info: ## Show current configuration
	@echo "$(YELLOW)Current Configuration:$(NC)"
	@echo "  Python: $(PYTHON)"
	@echo "  Virtual Env: $(VENV)"
	@echo "  Docs Directory: $(DOCS_DIR)"
	@echo "  Site Directory: $(SITE_DIR)"
	@echo "  Server Port: $(PORT)"
	@echo ""
	@echo "$(YELLOW)Environment:$(NC)"
	@echo "  OS: $$(uname -s)"
	@echo "  Python: $$($(PYTHON) --version)"
	@echo "  Current Dir: $$(pwd)"

.DEFAULT_GOAL := help