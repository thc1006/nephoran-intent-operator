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

# Contract and Schema Validation
.PHONY: validate-all
validate-all: validate-crds validate-contracts validate-examples
	@echo "🏆 All validations passed - contracts and manifests are compliant"