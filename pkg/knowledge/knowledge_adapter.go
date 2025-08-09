package knowledge

import (
	"fmt"
	"strings"
	"sync"

	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

// KnowledgeBaseAdapter adapts the lazy loader to the existing TelecomKnowledgeBase interface
type KnowledgeBaseAdapter struct {
	loader *LazyKnowledgeLoader
	mu     sync.RWMutex
}

// NewKnowledgeBaseAdapter creates a new adapter with lazy loading
func NewKnowledgeBaseAdapter(config *LoaderConfig) (*KnowledgeBaseAdapter, error) {
	loader, err := NewLazyKnowledgeLoader(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create lazy loader: %w", err)
	}

	return &KnowledgeBaseAdapter{
		loader: loader,
	}, nil
}

// NewOptimizedTelecomKnowledgeBase creates an optimized knowledge base with lazy loading
func NewOptimizedTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase {
	// Create a minimal knowledge base that uses lazy loading
	kb := &telecom.TelecomKnowledgeBase{
		NetworkFunctions: make(map[string]*telecom.NetworkFunctionSpec),
		Interfaces:       make(map[string]*telecom.InterfaceSpec),
		QosProfiles:      make(map[string]*telecom.QosProfile),
		SliceTypes:       make(map[string]*telecom.SliceTypeSpec),
		PerformanceKPIs:  make(map[string]*telecom.KPISpec),
		DeploymentTypes:  make(map[string]*telecom.DeploymentPattern),
	}

	// Only initialize essential metadata, not the full data
	kb.NetworkFunctions["amf"] = nil // Placeholder, will be loaded on demand
	kb.NetworkFunctions["smf"] = nil
	kb.NetworkFunctions["upf"] = nil

	return kb
}

// GetNetworkFunction retrieves a network function with lazy loading
func (a *KnowledgeBaseAdapter) GetNetworkFunction(name string) (*telecom.NetworkFunctionSpec, bool) {
	return a.loader.GetNetworkFunction(name)
}

// GetInterface retrieves an interface with lazy loading
func (a *KnowledgeBaseAdapter) GetInterface(name string) (*telecom.InterfaceSpec, bool) {
	return a.loader.GetInterface(name)
}

// GetQosProfile retrieves a QoS profile with lazy loading
func (a *KnowledgeBaseAdapter) GetQosProfile(name string) (*telecom.QosProfile, bool) {
	return a.loader.GetQosProfile(name)
}

// GetSliceType retrieves a slice type with lazy loading
func (a *KnowledgeBaseAdapter) GetSliceType(name string) (*telecom.SliceTypeSpec, bool) {
	return a.loader.GetSliceType(name)
}

// ListNetworkFunctions returns available network function names
func (a *KnowledgeBaseAdapter) ListNetworkFunctions() []string {
	return a.loader.ListNetworkFunctions()
}

// PreloadForIntent preloads relevant resources based on intent
func (a *KnowledgeBaseAdapter) PreloadForIntent(intent string) {
	a.loader.PreloadByIntent(intent)
}

// GetStats returns cache statistics
func (a *KnowledgeBaseAdapter) GetStats() map[string]interface{} {
	return a.loader.GetStats()
}

// ClearCache clears all caches
func (a *KnowledgeBaseAdapter) ClearCache() {
	a.loader.ClearCache()
}

// GetMemoryUsage returns estimated memory usage
func (a *KnowledgeBaseAdapter) GetMemoryUsage() int64 {
	return a.loader.GetMemoryUsage()
}

// IsInitialized returns whether the knowledge base is initialized
func (a *KnowledgeBaseAdapter) IsInitialized() bool {
	return a.loader.IsInitialized()
}

// ConvertToTelecomKnowledgeBase converts the adapter to a TelecomKnowledgeBase for compatibility
func (a *KnowledgeBaseAdapter) ConvertToTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase {
	kb := &telecom.TelecomKnowledgeBase{
		NetworkFunctions: make(map[string]*telecom.NetworkFunctionSpec),
		Interfaces:       make(map[string]*telecom.InterfaceSpec),
		QosProfiles:      make(map[string]*telecom.QosProfile),
		SliceTypes:       make(map[string]*telecom.SliceTypeSpec),
		PerformanceKPIs:  make(map[string]*telecom.KPISpec),
		DeploymentTypes:  make(map[string]*telecom.DeploymentPattern),
	}

	// Only load essential functions to keep memory usage low
	essentialFunctions := []string{"amf", "smf", "upf"}
	for _, name := range essentialFunctions {
		if nf, ok := a.GetNetworkFunction(name); ok {
			kb.NetworkFunctions[strings.ToLower(name)] = nf
		}
	}

	// Load basic interfaces
	essentialInterfaces := []string{"n1", "n2", "n3", "n4"}
	for _, name := range essentialInterfaces {
		if iface, ok := a.GetInterface(name); ok {
			kb.Interfaces[strings.ToLower(name)] = iface
		}
	}

	// Load basic QoS profiles
	essentialQos := []string{"5qi_1", "5qi_9"}
	for _, name := range essentialQos {
		if qos, ok := a.GetQosProfile(name); ok {
			kb.QosProfiles[strings.ToLower(name)] = qos
		}
	}

	// Load slice types
	sliceTypes := []string{"embb", "urllc", "mmtc"}
	for _, name := range sliceTypes {
		if slice, ok := a.GetSliceType(name); ok {
			kb.SliceTypes[strings.ToLower(name)] = slice
		}
	}

	// Add minimal KPIs
	kb.PerformanceKPIs["registration_success_rate"] = &telecom.KPISpec{
		Name:        "Registration Success Rate",
		Type:        "gauge",
		Unit:        "percentage",
		Description: "Percentage of successful UE registrations",
		Category:    "reliability",
		Thresholds: telecom.Thresholds{
			Critical: 95.0,
			Warning:  98.0,
			Target:   99.5,
		},
	}

	// Add minimal deployment patterns
	kb.DeploymentTypes["high-availability"] = &telecom.DeploymentPattern{
		Name:        "high-availability",
		Description: "High availability deployment",
		UseCase:     []string{"production"},
		Architecture: telecom.DeploymentArchitecture{
			Type:       "multi-region",
			Redundancy: "active-active",
		},
		Scaling: telecom.ScalingPattern{
			Horizontal: true,
			Predictive: true,
		},
		Resilience: telecom.ResiliencePattern{
			CircuitBreaker: true,
			Retry:          true,
			HealthCheck:    true,
		},
	}

	return kb
}

// LazyTelecomKnowledgeBase wraps TelecomKnowledgeBase with lazy loading
type LazyTelecomKnowledgeBase struct {
	*telecom.TelecomKnowledgeBase
	adapter *KnowledgeBaseAdapter
	mu      sync.RWMutex
}

// NewLazyTelecomKnowledgeBase creates a new lazy-loading knowledge base
func NewLazyTelecomKnowledgeBase() (*LazyTelecomKnowledgeBase, error) {
	config := DefaultLoaderConfig()
	adapter, err := NewKnowledgeBaseAdapter(config)
	if err != nil {
		return nil, err
	}

	// Create a minimal base knowledge base
	base := &telecom.TelecomKnowledgeBase{
		NetworkFunctions: make(map[string]*telecom.NetworkFunctionSpec),
		Interfaces:       make(map[string]*telecom.InterfaceSpec),
		QosProfiles:      make(map[string]*telecom.QosProfile),
		SliceTypes:       make(map[string]*telecom.SliceTypeSpec),
		PerformanceKPIs:  make(map[string]*telecom.KPISpec),
		DeploymentTypes:  make(map[string]*telecom.DeploymentPattern),
	}

	return &LazyTelecomKnowledgeBase{
		TelecomKnowledgeBase: base,
		adapter:              adapter,
	}, nil
}

// GetNetworkFunction overrides the base method with lazy loading
func (l *LazyTelecomKnowledgeBase) GetNetworkFunction(name string) (*telecom.NetworkFunctionSpec, bool) {
	l.mu.RLock()

	// Check if already loaded in base
	if nf, ok := l.TelecomKnowledgeBase.NetworkFunctions[strings.ToLower(name)]; ok && nf != nil {
		l.mu.RUnlock()
		return nf, true
	}
	l.mu.RUnlock()

	// Load from adapter
	nf, ok := l.adapter.GetNetworkFunction(name)
	if !ok {
		return nil, false
	}

	// Cache in base for future access
	l.mu.Lock()
	l.TelecomKnowledgeBase.NetworkFunctions[strings.ToLower(name)] = nf
	l.mu.Unlock()

	return nf, true
}

// GetInterface overrides the base method with lazy loading
func (l *LazyTelecomKnowledgeBase) GetInterface(name string) (*telecom.InterfaceSpec, bool) {
	l.mu.RLock()

	if iface, ok := l.TelecomKnowledgeBase.Interfaces[strings.ToLower(name)]; ok && iface != nil {
		l.mu.RUnlock()
		return iface, true
	}
	l.mu.RUnlock()

	iface, ok := l.adapter.GetInterface(name)
	if !ok {
		return nil, false
	}

	l.mu.Lock()
	l.TelecomKnowledgeBase.Interfaces[strings.ToLower(name)] = iface
	l.mu.Unlock()

	return iface, true
}

// GetQosProfile overrides the base method with lazy loading
func (l *LazyTelecomKnowledgeBase) GetQosProfile(name string) (*telecom.QosProfile, bool) {
	l.mu.RLock()

	if qos, ok := l.TelecomKnowledgeBase.QosProfiles[strings.ToLower(name)]; ok && qos != nil {
		l.mu.RUnlock()
		return qos, true
	}
	l.mu.RUnlock()

	qos, ok := l.adapter.GetQosProfile(name)
	if !ok {
		return nil, false
	}

	l.mu.Lock()
	l.TelecomKnowledgeBase.QosProfiles[strings.ToLower(name)] = qos
	l.mu.Unlock()

	return qos, true
}

// GetSliceType overrides the base method with lazy loading
func (l *LazyTelecomKnowledgeBase) GetSliceType(name string) (*telecom.SliceTypeSpec, bool) {
	l.mu.RLock()

	if slice, ok := l.TelecomKnowledgeBase.SliceTypes[strings.ToLower(name)]; ok && slice != nil {
		l.mu.RUnlock()
		return slice, true
	}
	l.mu.RUnlock()

	slice, ok := l.adapter.GetSliceType(name)
	if !ok {
		return nil, false
	}

	l.mu.Lock()
	l.TelecomKnowledgeBase.SliceTypes[strings.ToLower(name)] = slice
	l.mu.Unlock()

	return slice, true
}

// PreloadForIntent preloads relevant resources based on intent
func (l *LazyTelecomKnowledgeBase) PreloadForIntent(intent string) {
	l.adapter.PreloadForIntent(intent)
}

// GetStats returns cache statistics
func (l *LazyTelecomKnowledgeBase) GetStats() map[string]interface{} {
	return l.adapter.GetStats()
}

// GetMemoryUsage returns estimated memory usage
func (l *LazyTelecomKnowledgeBase) GetMemoryUsage() int64 {
	return l.adapter.GetMemoryUsage()
}
