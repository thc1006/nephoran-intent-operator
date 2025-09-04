# Nephoran Intent Operator - Makefile for Kubernetes Operator

# Variables
CONTROLLER_GEN ?= $(shell go env GOPATH)/bin/controller-gen
CRD_OUTPUT_DIR = config/crd/bases
API_PACKAGES = ./api/v1 ./api/v1alpha1 ./api/intent/v1alpha1

# Generate Kubernetes CRDs and related manifests
.PHONY: manifests
manifests: controller-gen
	@echo "🔧 Generating CRDs and manifests..."
	$(CONTROLLER_GEN) crd:allowDangerousTypes=true paths=./api/v1 paths=./api/v1alpha1 paths=./api/intent/v1alpha1 output:crd:dir=$(CRD_OUTPUT_DIR)
	$(CONTROLLER_GEN) rbac:roleName=nephoran-manager paths="./controllers/..." output:rbac:dir=config/rbac
	$(CONTROLLER_GEN) webhook paths="./..." output:webhook:dir=config/webhook
	@echo "✅ Manifests generated successfully"

# Install controller-gen if not present
.PHONY: controller-gen
controller-gen:
	@which $(CONTROLLER_GEN) > /dev/null 2>&1 || { \
		echo "🔨 Installing controller-gen..."; \
		go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest; \
	}

.PHONY: validate-crds
validate-crds: manifests
	@echo "Validating generated CRDs..."
	@for crd in config/crd/bases/*.yaml; do \
		echo "Validating $$crd..."; \
		kubeval --strict $$crd || exit 1; \
	done
	@echo "[SUCCESS] All CRDs are valid"