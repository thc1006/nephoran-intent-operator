module github.com/thc1006/nephoran-intent-operator

go 1.24.1

// Go 1.24+ optimization directives for better performance
//go:build go1.24

require (
	// === CORE DEPENDENCIES (Production Critical) ===

	// AWS SDK v2 - Updated to latest secure versions (Feb 2025)
	github.com/aws/aws-sdk-go-v2 v1.32.7
	github.com/aws/aws-sdk-go-v2/config v1.28.7
	github.com/aws/aws-sdk-go-v2/service/s3 v1.73.0

	// === FILE SYSTEM WATCHER ===
	github.com/fsnotify/fsnotify v1.9.0

	// Git operations for GitOps workflows
	github.com/go-git/go-git/v5 v5.16.2

	// Core logging and utilities
	github.com/go-logr/logr v1.4.3

	// Security and authentication
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/uuid v1.6.0

	// HTTP routing and API
	github.com/gorilla/mux v1.8.1

	// === CACHING ===
	github.com/hashicorp/golang-lru/v2 v2.0.7

	// Prometheus for metrics
	github.com/prometheus/client_golang v1.22.0

	// Redis client - UPDATED TO LATEST SECURE VERSION (CVE-2024-45337)
	github.com/redis/go-redis/v9 v9.12.0

	// === CIRCUIT BREAKER ===
	github.com/sony/gobreaker v1.0.0

	// === AI/ML DEPENDENCIES ===

	// Weaviate vector database client only
	github.com/weaviate/weaviate-go-client/v4 v4.16.1

	// === OBSERVABILITY STACK ===

	// OpenTelemetry - consolidated exports
	go.opentelemetry.io/otel v1.37.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.37.0
	go.opentelemetry.io/otel/metric v1.37.0
	go.opentelemetry.io/otel/sdk v1.37.0
	go.opentelemetry.io/otel/trace v1.37.0
	go.uber.org/zap v1.27.0
	
	// UPDATED SECURE CRYPTO - CRITICAL CVE-2024-45337 FIX
	golang.org/x/crypto v0.31.0
	golang.org/x/oauth2 v0.30.0

	// === ESSENTIAL UTILITIES ===
	golang.org/x/sync v0.16.0
	golang.org/x/time v0.12.0

	// === KUBERNETES ECOSYSTEM - UPDATED TO v0.31.12 STABLE ===

	// Kubernetes core APIs - Updated to latest stable v0.31.12 (January 2025)
	k8s.io/api v0.31.12
	k8s.io/apiextensions-apiserver v0.31.12
	k8s.io/apimachinery v0.31.12
	k8s.io/client-go v0.31.12
	k8s.io/klog/v2 v2.130.1
	sigs.k8s.io/controller-runtime v0.19.0

	// Configuration management - single YAML library
	sigs.k8s.io/yaml v1.6.0 // Replaces gopkg.in/yaml.v2
)

// === DEVELOPMENT DEPENDENCIES ===
// These are only used for development and testing - not included in production builds
require (
	// Security scanning and SBOM generation
	github.com/CycloneDX/cyclonedx-gomod v1.9.0
	github.com/onsi/ginkgo/v2 v2.23.4
	github.com/onsi/gomega v1.38.0
	// Testing essentials only
	github.com/stretchr/testify v1.10.0

	// API documentation
	github.com/swaggo/swag v1.16.6

	// JSON schema validation
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/vuln v1.1.4

	// gRPC for internal services - updated to latest secure version
	google.golang.org/grpc v1.75.0
	google.golang.org/protobuf v1.36.6 // indirect

	// Helm integration for package management
	helm.sh/helm/v3 v3.18.4

	// Code generation tools (build-time only)
	k8s.io/code-generator v0.31.12
	k8s.io/kube-openapi v0.0.0-20250318190949-c8a335a9a2ff // indirect
	sigs.k8s.io/controller-tools v0.18.0
)

require (
	cloud.google.com/go/compute v1.38.0
	cloud.google.com/go/container v1.43.0
	cloud.google.com/go/monitoring v1.24.2
	cloud.google.com/go/storage v1.56.0
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.8.2
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork v1.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage v1.8.1
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/Masterminds/sprig/v3 v3.3.0
	
	// Updated AWS SDK components to latest secure versions (Feb 2025)
	github.com/aws/aws-sdk-go-v2/credentials v1.17.67
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.63.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.47.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.241.0
	github.com/aws/aws-sdk-go-v2/service/ecs v1.62.0
	github.com/aws/aws-sdk-go-v2/service/eks v1.69.0
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.48.0
	github.com/aws/aws-sdk-go-v2/service/iam v1.45.0
	github.com/aws/aws-sdk-go-v2/service/rds v1.102.0
	github.com/aws/aws-sdk-go-v2/service/route53 v1.55.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.36.0
	
	github.com/bytedance/sonic v1.14.0
	github.com/cert-manager/cert-manager v1.18.2
	github.com/ghodss/yaml v1.0.0
	github.com/gin-gonic/gin v1.10.1
	github.com/go-ldap/ldap/v3 v3.4.11
	github.com/go-logr/zapr v1.3.0
	github.com/go-playground/validator/v10 v10.26.0
	// REMOVED VULNERABLE github.com/go-redis/redis/v8 - replaced with secure v9 above
	github.com/golang/mock v1.6.0
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6
	github.com/gophercloud/gophercloud v1.14.1
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674
	github.com/hashicorp/vault/api v1.20.0
	github.com/ledongthuc/pdf v0.0.0-20250511090121-5959a4027728
	github.com/lestrrat-go/jwx/v2 v2.1.6
	github.com/lib/pq v1.10.9
	github.com/microcosm-cc/bluemonday v1.0.27
	// Use standard mapstructure instead of problematic go-viper version
	github.com/montanaflynn/stats v0.7.1
	github.com/pkg/errors v0.9.1
	github.com/pquerna/otp v1.5.0
	github.com/prometheus/client_model v0.6.2
	github.com/prometheus/common v0.62.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rogpeppe/go-internal v1.14.1
	github.com/rs/cors v1.11.1
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
	github.com/shirou/gopsutil/v3 v3.24.5
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.9.1
	github.com/spf13/viper v1.20.1
	github.com/testcontainers/testcontainers-go v0.33.0
	github.com/tsenart/vegeta/v12 v12.12.0
	github.com/valyala/fasthttp v1.64.0
	github.com/valyala/fastjson v1.6.4
	github.com/vmware/govmomi v0.51.0
	github.com/weaviate/weaviate v1.27.0
	go.opentelemetry.io/otel/exporters/jaeger v1.17.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.37.0
	go.opentelemetry.io/otel/exporters/prometheus v0.54.0
	golang.org/x/mod v0.26.0
	golang.org/x/net v0.42.0
	gonum.org/v1/gonum v0.16.0
	google.golang.org/api v0.243.0
	gopkg.in/square/go-jose.v2 v2.6.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	honnef.co/go/tools v0.6.1
	k8s.io/metrics v0.31.12
	sigs.k8s.io/kustomize/api v0.19.0
	sigs.k8s.io/kustomize/kyaml v0.19.0
)

// Note: mapstructure replace removed due to module path conflict