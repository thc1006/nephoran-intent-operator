// Package providers implements O-RAN O2 IMS (Infrastructure Management Services) providers
// for managing cloud infrastructure resources following O-RAN specifications.
//
// # Overview
//
// The providers package provides a unified interface for managing diverse cloud infrastructure
// resources in O-RAN environments. It abstracts the complexity of different cloud providers
// and infrastructure platforms behind a common Provider interface, enabling consistent
// resource management across hybrid and multi-cloud deployments.
//
// # Key Features
//
//   - Unified Provider interface for all infrastructure operations
//   - Support for multiple resource types (clusters, networks, storage, applications)
//   - Event-driven architecture with comprehensive monitoring
//   - Factory pattern for dynamic provider instantiation
//   - Registry pattern for managing multiple provider instances
//   - Mock implementation for testing and development
//
// # Core Interfaces
//
// The package defines several key interfaces:
//
//   - Provider: Core interface for basic resource lifecycle management
//   - ClusterProvider: Extended interface for Kubernetes cluster operations
//   - NetworkProvider: Extended interface for network resource management
//   - StorageProvider: Extended interface for storage resource management
//   - MonitoringProvider: Extended interface for observability operations
//
// # Resource Types
//
// Supported resource types include:
//   - Clusters: Kubernetes clusters and compute infrastructure
//   - Networks: Virtual networks, subnets, and security groups
//   - Storage: Persistent volumes and snapshots
//   - Applications: Deployments, services, and configurations
//
// # Usage Example
//
//	// Create a provider factory
//	factory := providers.NewDefaultProviderFactory()
//
//	// Register provider types
//	factory.RegisterProvider("aws", awsConstructor, awsSchema)
//	factory.RegisterProvider("azure", azureConstructor, azureSchema)
//
//	// Create provider instances
//	config := providers.ProviderConfig{
//		Type: "aws",
//		Config: json.RawMessage(`{}`),
//		Credentials: map[string]string{
//			"accessKey": "...",
//			"secretKey": "...",
//		},
//	}
//
//	provider, err := factory.CreateProvider("aws", config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer provider.Close() // #nosec G307 - Error handled in defer
//
//	// Initialize the provider
//	ctx := context.Background()
//	err = provider.Initialize(ctx, config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create a resource
//	req := providers.ResourceRequest{
//		Name: "my-cluster",
//		Type: providers.ResourceTypeCluster,
//		Spec: json.RawMessage(`{}`),
//	}
//
//	resource, err := provider.CreateResource(ctx, req)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Monitor resource status
//	status, err := provider.GetResourceStatus(ctx, resource.ID)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("Resource status: %s\n", status)
//
// # Event Handling
//
// The package supports event-driven operations through the EventHandler interface:
//
//	type MyEventHandler struct{}
//
//	func (h *MyEventHandler) HandleResourceEvent(ctx context.Context, event providers.ResourceEvent) error {
//		log.Printf("Resource event: %s for resource %s", event.Type, event.ResourceID)
//		return nil
//	}
//
//	func (h *MyEventHandler) HandleProviderEvent(ctx context.Context, event providers.ProviderEvent) error {
//		log.Printf("Provider event: %s from provider %s", event.Type, event.ProviderName)
//		return nil
//	}
//
// # Provider Registry
//
// For managing multiple provider instances:
//
//	registry := providers.NewProviderRegistry(factory)
//
//	// Register providers
//	err := registry.CreateAndRegisterProvider("aws-prod", "aws", prodConfig)
//	err = registry.CreateAndRegisterProvider("aws-dev", "aws", devConfig)
//
//	// Use providers
//	prodProvider, err := registry.GetProvider("aws-prod")
//	devProvider, err := registry.GetProvider("aws-dev")
//
//	// Clean up
//	defer registry.Close() // #nosec G307 - Error handled in defer
//
// # Testing
//
// The package includes a comprehensive MockProvider for testing:
//
//	provider := providers.NewMockProvider("test-provider")
//	err := provider.Initialize(ctx, mockConfig)
//
//	// Use provider for testing...
//
// # Error Handling
//
// All operations return appropriate errors following Go conventions. Providers
// should implement proper error handling and return meaningful error messages
// that can be used for debugging and monitoring.
//
// # Thread Safety
//
// All provider implementations must be thread-safe. The factory and registry
// implementations use appropriate synchronization mechanisms to ensure safe
// concurrent access.
//
// # Compliance
//
// This package follows O-RAN O2 IMS specifications for infrastructure management
// and integrates with the broader Nephio ecosystem for cloud-native network
// function management.
package providers
