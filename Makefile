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

.PHONY: validate-crds
validate-crds: manifests
	@echo "Validating generated CRDs..."
	@for crd in config/crd/bases/*.yaml; do \
		echo "Validating $$crd..."; \
		kubeval --strict $$crd || exit 1; \
	done
	@echo "[SUCCESS] All CRDs are valid"

# Build manager binary
.PHONY: build
build: generate
	@echo "🔨 Building manager binary..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/manager cmd/main.go
	@echo "✅ Manager binary built successfully"

# Build docker image
.PHONY: docker-build
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t ${IMG} .
	@echo "✅ Docker image built successfully"

# Deploy to Kubernetes cluster
.PHONY: deploy
deploy: manifests
	@echo "⚙️ Deploying to Kubernetes cluster..."
	kustomize build config/default | kubectl apply -f -
	@echo "✅ Deployment completed"

# Install CRDs into cluster
.PHONY: install
install: manifests
	@echo "📝 Installing CRDs..."
	kustomize build config/crd | kubectl apply -f -
	@echo "✅ CRDs installed successfully"

# Set default image if not provided
IMG ?= nephoran-operator:latest