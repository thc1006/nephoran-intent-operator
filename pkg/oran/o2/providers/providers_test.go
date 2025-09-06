package providers

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestProviderFactory(t *testing.T) {
	factory := NewDefaultProviderFactory()

	// Test registering a provider
	err := factory.RegisterProvider("test", MockProviderConstructor, GetMockProviderSchema())
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	// Test listing supported types
	types := factory.ListSupportedTypes()
	if len(types) != 1 || types[0] != "test" {
		t.Errorf("Expected [test], got %v", types)
	}

	// Test getting schema
	schema, err := factory.GetProviderSchema("test")
	if err != nil {
		t.Fatalf("Failed to get schema: %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema")
	}

	// Test creating provider
	config := ProviderConfig{
		Type:   "test",
		Config: json.RawMessage(`{}`),
	}

	provider, err := factory.CreateProvider("test", config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider == nil {
		t.Error("Expected non-nil provider")
	}

	// Clean up
	provider.Close()
}

func TestMockProvider(t *testing.T) {
	provider := NewMockProvider("test-provider")

	// Test GetInfo before initialization
	info := provider.GetInfo()
	if info.Name != "test-provider" {
		t.Errorf("Expected name 'test-provider', got %s", info.Name)
	}

	// Test operations before initialization (should fail)
	ctx := context.Background()
	_, err := provider.CreateResource(ctx, &CreateResourceRequest{
		Name: "test-resource",
		Type: string(ResourceTypeDeployment),
	})
	if err == nil {
		t.Error("Expected error for uninitialized provider")
	}

	// Initialize provider
	config := ProviderConfig{
		Type:   "mock",
		Config: json.RawMessage(`{}`),
	}

	err = provider.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}

	// Test double initialization (should fail)
	err = provider.Initialize(ctx, config)
	if err == nil {
		t.Error("Expected error for double initialization")
	}
}

func TestResourceOperations(t *testing.T) {
	provider := NewMockProvider("test-provider")
	ctx := context.Background()

	// Initialize provider
	config := ProviderConfig{
		Type:   "mock",
		Config: json.RawMessage(`{}`),
	}

	err := provider.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}
	defer provider.Close() // #nosec G307 - Error handled in defer

	// Test CreateResource
	req := &CreateResourceRequest{
		Name: "test-deployment",
		Type: string(ResourceTypeDeployment),
		Specification: json.RawMessage(`{}`),
		Labels: map[string]string{
			"env": "test",
		},
	}

	resource, err := provider.CreateResource(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create resource: %v", err)
	}

	if resource.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, resource.Name)
	}

	if resource.Type != req.Type {
		t.Errorf("Expected type %s, got %s", req.Type, resource.Type)
	}

	// Wait for async creation to complete
	time.Sleep(200 * time.Millisecond)

	// Test GetResource
	retrievedResource, err := provider.GetResource(ctx, resource.ID)
	if err != nil {
		t.Fatalf("Failed to get resource: %v", err)
	}

	if retrievedResource.Status != string(StatusReady) {
		t.Errorf("Expected status %s, got %s", StatusReady, retrievedResource.Status)
	}

	// Test GetResourceStatus
	status, err := provider.GetResourceStatus(ctx, resource.ID)
	if err != nil {
		t.Fatalf("Failed to get resource status: %v", err)
	}

	if status != StatusReady {
		t.Errorf("Expected status %s, got %s", StatusReady, status)
	}

	// Test UpdateResource
	updateReq := &UpdateResourceRequest{
		Specification: json.RawMessage(`{}`),
		Labels: map[string]string{
			"env":     "test",
			"updated": "true",
		},
	}

	updatedResource, err := provider.UpdateResource(ctx, resource.ID, updateReq)
	if err != nil {
		t.Fatalf("Failed to update resource: %v", err)
	}

	if updatedResource.Name != updateReq.Name {
		t.Errorf("Expected updated name %s, got %s", updateReq.Name, updatedResource.Name)
	}

	// Test ListResources
	resources, err := provider.ListResources(ctx, &ResourceFilter{})
	if err != nil {
		t.Fatalf("Failed to list resources: %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resources))
	}

	// Test ListResources with type filter
	resources, err = provider.ListResources(ctx, &ResourceFilter{
		Types: []string{string(ResourceTypeDeployment)},
	})
	if err != nil {
		t.Fatalf("Failed to list resources with filter: %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("Expected 1 deployment resource, got %d", len(resources))
	}

	// Test ListResources with label filter
	resources, err = provider.ListResources(ctx, &ResourceFilter{
		Labels: map[string]string{
			"env": "test",
		},
	})
	if err != nil {
		t.Fatalf("Failed to list resources with label filter: %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("Expected 1 resource with label filter, got %d", len(resources))
	}

	// Test DeleteResource
	err = provider.DeleteResource(ctx, resource.ID)
	if err != nil {
		t.Fatalf("Failed to delete resource: %v", err)
	}

	// Wait for async deletion to complete
	time.Sleep(100 * time.Millisecond)

	// Verify resource is deleted
	_, err = provider.GetResource(ctx, resource.ID)
	if err == nil {
		t.Error("Expected error when getting deleted resource")
	}
}

func TestProviderRegistry(t *testing.T) {
	factory := NewDefaultProviderFactory()
	registry := NewProviderRegistry()

	// Register mock provider type
	err := factory.RegisterProvider("mock", MockProviderConstructor, GetMockProviderSchema())
	if err != nil {
		t.Fatalf("Failed to register provider type: %v", err)
	}

	// Test CreateAndRegisterProvider
	config := ProviderConfig{
		Type:   "mock",
		Config: json.RawMessage(`{}`),
	}

	err = registry.CreateAndRegisterProvider("test-provider", "mock", config)
	if err != nil {
		t.Fatalf("Failed to create and register provider: %v", err)
	}

	// Test GetProvider
	provider, err := registry.GetProvider("test-provider")
	if err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if provider == nil {
		t.Error("Expected non-nil provider")
	}

	// Test ListProviders
	providers := registry.ListProviders()
	if len(providers) != 1 || providers[0] != "test-provider" {
		t.Errorf("Expected [test-provider], got %v", providers)
	}

	// Test UnregisterProvider
	err = registry.UnregisterProvider("test-provider")
	if err != nil {
		t.Fatalf("Failed to unregister provider: %v", err)
	}

	// Verify provider is unregistered
	_, err = registry.GetProvider("test-provider")
	if err == nil {
		t.Error("Expected error when getting unregistered provider")
	}

	// Test Close
	err = registry.Close()
	if err != nil {
		t.Fatalf("Failed to close registry: %v", err)
	}
}

func TestResourceTypes(t *testing.T) {
	// Test that all resource types are defined
	types := []ResourceType{
		ResourceTypeCluster,
		ResourceTypeNode,
		ResourceTypeNetwork,
		ResourceTypeStorage,
		ResourceTypeDeployment,
		ResourceTypeService,
		ResourceTypeConfigMap,
		ResourceTypeSecret,
	}

	for _, resourceType := range types {
		if string(resourceType) == "" {
			t.Errorf("Resource type is empty")
		}
	}
}

func TestResourceStatuses(t *testing.T) {
	// Test that all resource statuses are defined
	statuses := []ResourceStatus{
		StatusPending,
		StatusCreating,
		StatusReady,
		StatusUpdating,
		StatusDeleting,
		StatusError,
		StatusUnknown,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("Resource status is empty")
		}
	}
}

func TestEventTypes(t *testing.T) {
	// Test that all event types are defined
	eventTypes := []EventType{
		EventTypeResourceCreated,
		EventTypeResourceUpdated,
		EventTypeResourceDeleted,
		EventTypeResourceFailed,
		EventTypeProviderStarted,
		EventTypeProviderStopped,
		EventTypeProviderError,
		EventTypeScaleUp,
		EventTypeScaleDown,
		EventTypeHealthCheck,
		EventTypeAlertTriggered,
	}

	for _, eventType := range eventTypes {
		if string(eventType) == "" {
			t.Errorf("Event type is empty")
		}
	}
}

func TestGlobalFactory(t *testing.T) {
	// Test that global factory works
	factory := GetGlobalFactory()
	if factory == nil {
		t.Error("Expected non-nil global factory")
	}

	// Mock provider should be registered during init
	types := factory.ListSupportedTypes()
	found := false
	for _, providerType := range types {
		if providerType == "mock" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Mock provider not found in global factory")
	}

	// Test creating provider using global factory
	config := ProviderConfig{
		Type:   "mock",
		Config: json.RawMessage(`{}`),
	}

	provider, err := CreateGlobalProvider("mock", config)
	if err != nil {
		t.Fatalf("Failed to create provider using global factory: %v", err)
	}

	if provider == nil {
		t.Error("Expected non-nil provider")
	}

	// Clean up
	provider.Close()
}
