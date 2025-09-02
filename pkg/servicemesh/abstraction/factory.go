// Package abstraction provides service mesh factory.

package abstraction

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ProviderFactory is a function that creates a service mesh implementation.

type ProviderFactory func(kubernetes.Interface, client.Client, *rest.Config, *ServiceMeshConfig) (ServiceMeshInterface, error)

var (
	providerRegistry = make(map[ServiceMeshProvider]ProviderFactory)

	registryMutex sync.RWMutex
)

// RegisterProvider registers a service mesh provider factory.

func RegisterProvider(provider ServiceMeshProvider, factory ProviderFactory) {
	registryMutex.Lock()

	defer registryMutex.Unlock()

	providerRegistry[provider] = factory
}

// ServiceMeshFactory creates service mesh implementations.

type ServiceMeshFactory struct {
	kubeClient kubernetes.Interface

	dynamicClient client.Client

	config *rest.Config

	detector *ServiceMeshDetector

	logger logr.Logger
}

// NewServiceMeshFactory creates a new service mesh factory.

func NewServiceMeshFactory(
	kubeClient kubernetes.Interface,

	dynamicClient client.Client,

	config *rest.Config,
) *ServiceMeshFactory {
	return &ServiceMeshFactory{
		kubeClient: kubeClient,

		dynamicClient: dynamicClient,

		config: config,

		detector: NewServiceMeshDetector(kubeClient, config),

		logger: log.Log.WithName("service-mesh-factory"),
	}
}

// CreateServiceMesh creates a service mesh implementation based on configuration.

func (f *ServiceMeshFactory) CreateServiceMesh(ctx context.Context, config *ServiceMeshConfig) (ServiceMeshInterface, error) {
	provider := config.Provider

	// Auto-detect if provider not specified.

	if provider == "" || provider == "auto" {

		detectedProvider, err := f.detector.DetectServiceMesh(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to detect service mesh: %w", err)
		}

		provider = detectedProvider

		f.logger.Info("Auto-detected service mesh provider", "provider", provider)

	}

	// Create provider-specific implementation using registry.

	registryMutex.RLock()

	factory, exists := providerRegistry[ServiceMeshProvider(provider)]

	registryMutex.RUnlock()

	if !exists {

		if provider == ProviderNone {
			return f.createNoOpMesh(ctx, config)
		}

		return nil, fmt.Errorf("unsupported service mesh provider: %s", provider)

	}

	mesh, err := factory(f.kubeClient, f.dynamicClient, f.config, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create service mesh %s: %w", provider, err)
	}

	// Initialize the mesh.

	if err := mesh.Initialize(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to initialize service mesh %s: %w", provider, err)
	}

	return mesh, nil
}

// createNoOpMesh creates a no-op service mesh implementation for environments without service mesh.

func (f *ServiceMeshFactory) createNoOpMesh(ctx context.Context, config *ServiceMeshConfig) (ServiceMeshInterface, error) {
	f.logger.Info("Creating no-op service mesh implementation")

	return NewNoOpMesh(), nil
}

// GetAvailableProviders returns the list of available service mesh providers.

func (f *ServiceMeshFactory) GetAvailableProviders(ctx context.Context) ([]ProviderInfo, error) {
	providers := []ProviderInfo{}

	// Check each provider.

	for _, provider := range []ServiceMeshProvider{ProviderIstio, ProviderLinkerd, ProviderConsul} {

		info := ProviderInfo{
			Provider: provider,

			Available: false,
		}

		// Check if provider is installed.

		detectedProvider, err := f.detector.DetectServiceMesh(ctx)

		if err == nil && detectedProvider == provider {

			info.Available = true

			info.Version, _ = f.detector.GetServiceMeshVersion(ctx, provider)

			// Get capabilities.

			info.Capabilities = f.getProviderCapabilities(provider)

		}

		providers = append(providers, info)

	}

	return providers, nil
}

// getProviderCapabilities returns the capabilities of a service mesh provider.

func (f *ServiceMeshFactory) getProviderCapabilities(provider ServiceMeshProvider) []Capability {
	switch provider {

	case ProviderIstio:

		return []Capability{
			CapabilityMTLS,

			CapabilityTrafficManagement,

			CapabilityObservability,

			CapabilityMultiCluster,

			CapabilitySPIFFE,

			CapabilityWASM,
		}

	case ProviderLinkerd:

		return []Capability{
			CapabilityMTLS,

			CapabilityTrafficManagement,

			CapabilityObservability,

			CapabilitySPIFFE,
		}

	case ProviderConsul:

		return []Capability{
			CapabilityMTLS,

			CapabilityTrafficManagement,

			CapabilityObservability,

			CapabilityMultiCluster,
		}

	default:

		return []Capability{}

	}
}

// ProviderInfo contains information about a service mesh provider.

type ProviderInfo struct {
	Provider ServiceMeshProvider `json:"provider"`

	Available bool `json:"available"`

	Version string `json:"version,omitempty"`

	Capabilities []Capability `json:"capabilities,omitempty"`
}

// ValidateConfiguration validates service mesh configuration.

func (f *ServiceMeshFactory) ValidateConfiguration(config *ServiceMeshConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Validate provider.

	switch config.Provider {

	case ProviderIstio, ProviderLinkerd, ProviderConsul, ProviderNone, "auto", "":

		// Valid providers.

	default:

		return fmt.Errorf("invalid provider: %s", config.Provider)

	}

	// Validate certificate configuration.

	if config.CertificateConfig != nil {

		if config.CertificateConfig.CertLifetime <= 0 {
			return fmt.Errorf("certificate lifetime must be positive")
		}

		if config.CertificateConfig.RotationThreshold < 0 || config.CertificateConfig.RotationThreshold > 100 {
			return fmt.Errorf("rotation threshold must be between 0 and 100")
		}

		if config.CertificateConfig.SPIFFEEnabled && config.CertificateConfig.SPIREServerURL == "" {
			return fmt.Errorf("SPIRE server URL required when SPIFFE is enabled")
		}

	}

	// Validate policy defaults.

	if config.PolicyDefaults != nil {
		switch config.PolicyDefaults.MTLSMode {

		case "STRICT", "PERMISSIVE", "DISABLE", "":

			// Valid modes.

		default:

			return fmt.Errorf("invalid mTLS mode: %s", config.PolicyDefaults.MTLSMode)

		}
	}

	// Validate observability configuration.

	if config.ObservabilityConfig != nil {

		if config.ObservabilityConfig.EnableTracing && config.ObservabilityConfig.TracingBackend == "" {
			return fmt.Errorf("tracing backend required when tracing is enabled")
		}

		if config.ObservabilityConfig.MetricsPort < 0 || config.ObservabilityConfig.MetricsPort > 65535 {
			return fmt.Errorf("invalid metrics port: %d", config.ObservabilityConfig.MetricsPort)
		}

	}

	// Validate multi-cluster configuration.

	if config.MultiCluster != nil {

		if config.MultiCluster.ClusterName == "" {
			return fmt.Errorf("cluster name required for multi-cluster configuration")
		}

		if config.MultiCluster.Federation != nil && config.MultiCluster.Federation.Enabled {
			if len(config.MultiCluster.Federation.TrustDomains) == 0 {
				return fmt.Errorf("trust domains required for federation")
			}
		}

	}

	return nil
}
