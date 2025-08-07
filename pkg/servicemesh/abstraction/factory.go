// Package abstraction provides service mesh factory
package abstraction

import (
	"context"
	"fmt"

	"github.com/nephoran-operator/pkg/servicemesh/consul"
	"github.com/nephoran-operator/pkg/servicemesh/istio"
	"github.com/nephoran-operator/pkg/servicemesh/linkerd"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ServiceMeshFactory creates service mesh implementations
type ServiceMeshFactory struct {
	kubeClient     kubernetes.Interface
	dynamicClient  client.Client
	config         *rest.Config
	detector       *ServiceMeshDetector
	logger         log.Logger
}

// NewServiceMeshFactory creates a new service mesh factory
func NewServiceMeshFactory(
	kubeClient kubernetes.Interface,
	dynamicClient client.Client,
	config *rest.Config,
) *ServiceMeshFactory {
	return &ServiceMeshFactory{
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
		config:        config,
		detector:      NewServiceMeshDetector(kubeClient, config),
		logger:        log.Log.WithName("service-mesh-factory"),
	}
}

// CreateServiceMesh creates a service mesh implementation based on configuration
func (f *ServiceMeshFactory) CreateServiceMesh(ctx context.Context, config *ServiceMeshConfig) (ServiceMeshInterface, error) {
	provider := config.Provider

	// Auto-detect if provider not specified
	if provider == "" || provider == "auto" {
		detectedProvider, err := f.detector.DetectServiceMesh(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to detect service mesh: %w", err)
		}
		provider = detectedProvider
		f.logger.Info("Auto-detected service mesh provider", "provider", provider)
	}

	// Create provider-specific implementation
	switch provider {
	case ProviderIstio:
		return f.createIstioMesh(ctx, config)
	case ProviderLinkerd:
		return f.createLinkerdMesh(ctx, config)
	case ProviderConsul:
		return f.createConsulMesh(ctx, config)
	case ProviderNone:
		return f.createNoOpMesh(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported service mesh provider: %s", provider)
	}
}

// createIstioMesh creates an Istio service mesh implementation
func (f *ServiceMeshFactory) createIstioMesh(ctx context.Context, config *ServiceMeshConfig) (ServiceMeshInterface, error) {
	f.logger.Info("Creating Istio service mesh implementation")
	
	// Create Istio-specific configuration
	istioConfig := &istio.Config{
		Namespace:           config.Namespace,
		TrustDomain:        config.TrustDomain,
		ControlPlaneURL:    config.ControlPlaneURL,
		CertificateConfig:  config.CertificateConfig,
		PolicyDefaults:     config.PolicyDefaults,
		ObservabilityConfig: config.ObservabilityConfig,
		MultiCluster:       config.MultiCluster,
	}

	// Extract Istio-specific settings from custom config
	if config.CustomConfig != nil {
		if pilotURL, ok := config.CustomConfig["pilotURL"].(string); ok {
			istioConfig.PilotURL = pilotURL
		}
		if meshID, ok := config.CustomConfig["meshID"].(string); ok {
			istioConfig.MeshID = meshID
		}
		if network, ok := config.CustomConfig["network"].(string); ok {
			istioConfig.Network = network
		}
	}

	mesh, err := istio.NewIstioMesh(f.kubeClient, f.dynamicClient, f.config, istioConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Istio mesh: %w", err)
	}

	// Initialize the mesh
	if err := mesh.Initialize(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to initialize Istio mesh: %w", err)
	}

	return mesh, nil
}

// createLinkerdMesh creates a Linkerd service mesh implementation
func (f *ServiceMeshFactory) createLinkerdMesh(ctx context.Context, config *ServiceMeshConfig) (ServiceMeshInterface, error) {
	f.logger.Info("Creating Linkerd service mesh implementation")
	
	// Create Linkerd-specific configuration
	linkerdConfig := &linkerd.Config{
		Namespace:           config.Namespace,
		TrustDomain:        config.TrustDomain,
		ControlPlaneURL:    config.ControlPlaneURL,
		CertificateConfig:  config.CertificateConfig,
		PolicyDefaults:     config.PolicyDefaults,
		ObservabilityConfig: config.ObservabilityConfig,
	}

	// Extract Linkerd-specific settings from custom config
	if config.CustomConfig != nil {
		if identityTrustDomain, ok := config.CustomConfig["identityTrustDomain"].(string); ok {
			linkerdConfig.IdentityTrustDomain = identityTrustDomain
		}
		if proxyLogLevel, ok := config.CustomConfig["proxyLogLevel"].(string); ok {
			linkerdConfig.ProxyLogLevel = proxyLogLevel
		}
	}

	mesh, err := linkerd.NewLinkerdMesh(f.kubeClient, f.dynamicClient, f.config, linkerdConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Linkerd mesh: %w", err)
	}

	// Initialize the mesh
	if err := mesh.Initialize(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to initialize Linkerd mesh: %w", err)
	}

	return mesh, nil
}

// createConsulMesh creates a Consul Connect service mesh implementation
func (f *ServiceMeshFactory) createConsulMesh(ctx context.Context, config *ServiceMeshConfig) (ServiceMeshInterface, error) {
	f.logger.Info("Creating Consul Connect service mesh implementation")
	
	// Create Consul-specific configuration
	consulConfig := &consul.Config{
		Namespace:           config.Namespace,
		TrustDomain:        config.TrustDomain,
		CertificateConfig:  config.CertificateConfig,
		PolicyDefaults:     config.PolicyDefaults,
		ObservabilityConfig: config.ObservabilityConfig,
	}

	// Extract Consul-specific settings from custom config
	if config.CustomConfig != nil {
		if datacenter, ok := config.CustomConfig["datacenter"].(string); ok {
			consulConfig.Datacenter = datacenter
		}
		if gossipKey, ok := config.CustomConfig["gossipKey"].(string); ok {
			consulConfig.GossipKey = gossipKey
		}
		if aclEnabled, ok := config.CustomConfig["aclEnabled"].(bool); ok {
			consulConfig.ACLEnabled = aclEnabled
		}
	}

	mesh, err := consul.NewConsulMesh(f.kubeClient, f.dynamicClient, f.config, consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul mesh: %w", err)
	}

	// Initialize the mesh
	if err := mesh.Initialize(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to initialize Consul mesh: %w", err)
	}

	return mesh, nil
}

// createNoOpMesh creates a no-op service mesh implementation for environments without service mesh
func (f *ServiceMeshFactory) createNoOpMesh(ctx context.Context, config *ServiceMeshConfig) (ServiceMeshInterface, error) {
	f.logger.Info("Creating no-op service mesh implementation")
	return NewNoOpMesh(), nil
}

// GetAvailableProviders returns the list of available service mesh providers
func (f *ServiceMeshFactory) GetAvailableProviders(ctx context.Context) ([]ProviderInfo, error) {
	providers := []ProviderInfo{}

	// Check each provider
	for _, provider := range []ServiceMeshProvider{ProviderIstio, ProviderLinkerd, ProviderConsul} {
		info := ProviderInfo{
			Provider: provider,
			Available: false,
		}

		// Check if provider is installed
		detectedProvider, err := f.detector.DetectServiceMesh(ctx)
		if err == nil && detectedProvider == provider {
			info.Available = true
			info.Version, _ = f.detector.GetServiceMeshVersion(ctx, provider)
			
			// Get capabilities
			info.Capabilities = f.getProviderCapabilities(provider)
		}

		providers = append(providers, info)
	}

	return providers, nil
}

// getProviderCapabilities returns the capabilities of a service mesh provider
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

// ProviderInfo contains information about a service mesh provider
type ProviderInfo struct {
	Provider     ServiceMeshProvider `json:"provider"`
	Available    bool                `json:"available"`
	Version      string              `json:"version,omitempty"`
	Capabilities []Capability        `json:"capabilities,omitempty"`
}

// ValidateConfiguration validates service mesh configuration
func (f *ServiceMeshFactory) ValidateConfiguration(config *ServiceMeshConfig) error {
	if config == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Validate provider
	switch config.Provider {
	case ProviderIstio, ProviderLinkerd, ProviderConsul, ProviderNone, "auto", "":
		// Valid providers
	default:
		return fmt.Errorf("invalid provider: %s", config.Provider)
	}

	// Validate certificate configuration
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

	// Validate policy defaults
	if config.PolicyDefaults != nil {
		switch config.PolicyDefaults.MTLSMode {
		case "STRICT", "PERMISSIVE", "DISABLE", "":
			// Valid modes
		default:
			return fmt.Errorf("invalid mTLS mode: %s", config.PolicyDefaults.MTLSMode)
		}
	}

	// Validate observability configuration
	if config.ObservabilityConfig != nil {
		if config.ObservabilityConfig.EnableTracing && config.ObservabilityConfig.TracingBackend == "" {
			return fmt.Errorf("tracing backend required when tracing is enabled")
		}
		if config.ObservabilityConfig.MetricsPort < 0 || config.ObservabilityConfig.MetricsPort > 65535 {
			return fmt.Errorf("invalid metrics port: %d", config.ObservabilityConfig.MetricsPort)
		}
	}

	// Validate multi-cluster configuration
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