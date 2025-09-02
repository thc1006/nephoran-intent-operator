package knowledge

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

// LazyKnowledgeLoader provides lazy loading and caching for telecom knowledge base.

type LazyKnowledgeLoader struct {
	// LRU cache for frequently accessed items.

	nfCache *lru.Cache[string, *telecom.NetworkFunctionSpec]

	interfaceCache *lru.Cache[string, *telecom.InterfaceSpec]

	qosCache *lru.Cache[string, *telecom.QosProfile]

	sliceCache *lru.Cache[string, *telecom.SliceTypeSpec]

	// Index for fast keyword lookup.

	keywordIndex map[string][]string // keyword -> [resource_ids]

	// Compressed data storage.

	compressedData map[string][]byte // resource_id -> compressed_data

	// Resource metadata for lazy loading decisions.

	metadata map[string]*ResourceMetadata

	// Mutex for thread-safe operations.

	mu sync.RWMutex

	// Cache statistics.

	stats *CacheStats

	// Configuration.

	config *LoaderConfig
}

// ResourceMetadata contains metadata about a resource.

type ResourceMetadata struct {
	ID string

	Type string // "nf", "interface", "qos", "slice", "kpi", "deployment"

	Name string

	Keywords []string

	Size int

	LastAccess time.Time

	AccessCount int
}

// CacheStats tracks cache performance.

type CacheStats struct {
	Hits int64

	Misses int64

	Evictions int64

	LoadTime time.Duration

	TotalLoads int64

	mu sync.RWMutex
}

// LoaderConfig contains configuration for the lazy loader.

type LoaderConfig struct {
	CacheSize int // Number of items to keep in LRU cache

	CompressData bool // Whether to compress stored data

	PreloadEssential bool // Preload essential network functions

	MaxMemoryMB int // Maximum memory usage in MB

	TTL time.Duration // Cache TTL
}

// DefaultLoaderConfig returns default configuration.

func DefaultLoaderConfig() *LoaderConfig {
	return &LoaderConfig{
		CacheSize: 50, // Keep 50 most recent items in memory

		CompressData: true,

		PreloadEssential: true,

		MaxMemoryMB: 100, // 100MB max memory

		TTL: 30 * time.Minute,
	}
}

// NewLazyKnowledgeLoader creates a new lazy loader with the given configuration.

func NewLazyKnowledgeLoader(config *LoaderConfig) (*LazyKnowledgeLoader, error) {
	if config == nil {
		config = DefaultLoaderConfig()
	}

	// Create LRU caches.

	nfCache, err := lru.New[string, *telecom.NetworkFunctionSpec](config.CacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create NF cache: %w", err)
	}

	interfaceCache, err := lru.New[string, *telecom.InterfaceSpec](config.CacheSize / 2)
	if err != nil {
		return nil, fmt.Errorf("failed to create interface cache: %w", err)
	}

	qosCache, err := lru.New[string, *telecom.QosProfile](config.CacheSize / 4)
	if err != nil {
		return nil, fmt.Errorf("failed to create QoS cache: %w", err)
	}

	sliceCache, err := lru.New[string, *telecom.SliceTypeSpec](config.CacheSize / 4)
	if err != nil {
		return nil, fmt.Errorf("failed to create slice cache: %w", err)
	}

	loader := &LazyKnowledgeLoader{
		nfCache: nfCache,

		interfaceCache: interfaceCache,

		qosCache: qosCache,

		sliceCache: sliceCache,

		keywordIndex: make(map[string][]string),

		compressedData: make(map[string][]byte),

		metadata: make(map[string]*ResourceMetadata),

		stats: &CacheStats{},

		config: config,
	}

	// Initialize with minimal data.

	if err := loader.initializeMinimal(); err != nil {
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	// Preload essential items if configured.

	if config.PreloadEssential {
		loader.preloadEssentials()
	}

	return loader, nil
}

// initializeMinimal initializes only the index and metadata.

func (l *LazyKnowledgeLoader) initializeMinimal() error {
	// Build keyword index for common network functions.

	essentialNFs := []string{"amf", "smf", "upf", "pcf", "udm", "ausf", "nrf", "nssf"}

	for _, nfName := range essentialNFs {

		metadata := &ResourceMetadata{
			ID: nfName,

			Type: "nf",

			Name: strings.ToUpper(nfName),

			Keywords: l.extractKeywords(nfName),
		}

		l.metadata[nfName] = metadata

		// Update keyword index.

		for _, keyword := range metadata.Keywords {
			l.keywordIndex[keyword] = append(l.keywordIndex[keyword], nfName)
		}

	}

	// Add interface metadata.

	interfaces := []string{"n1", "n2", "n3", "n4", "namf", "nsmf"}

	for _, ifName := range interfaces {

		metadata := &ResourceMetadata{
			ID: ifName,

			Type: "interface",

			Name: strings.ToUpper(ifName),

			Keywords: []string{ifName, "interface", "5g"},
		}

		l.metadata[ifName] = metadata

		for _, keyword := range metadata.Keywords {
			l.keywordIndex[keyword] = append(l.keywordIndex[keyword], ifName)
		}

	}

	// Add slice types.

	sliceTypes := []string{"embb", "urllc", "mmtc"}

	for _, sliceType := range sliceTypes {

		metadata := &ResourceMetadata{
			ID: sliceType,

			Type: "slice",

			Name: strings.ToUpper(sliceType),

			Keywords: []string{sliceType, "slice", "network"},
		}

		l.metadata[sliceType] = metadata

		for _, keyword := range metadata.Keywords {
			l.keywordIndex[keyword] = append(l.keywordIndex[keyword], sliceType)
		}

	}

	return nil
}

// preloadEssentials preloads essential network functions.

func (l *LazyKnowledgeLoader) preloadEssentials() {
	essentials := []string{"amf", "smf", "upf"}

	for _, nf := range essentials {
		// This will trigger lazy loading.

		l.GetNetworkFunction(nf)
	}
}

// extractKeywords extracts keywords from a resource name.

func (l *LazyKnowledgeLoader) extractKeywords(name string) []string {
	keywords := []string{name}

	// Add common variations.

	keywords = append(keywords, strings.ToLower(name))

	keywords = append(keywords, strings.ToUpper(name))

	// Add domain-specific keywords.

	switch strings.ToLower(name) {

	case "amf":

		keywords = append(keywords, "access", "mobility", "management", "registration")

	case "smf":

		keywords = append(keywords, "session", "management", "pdu")

	case "upf":

		keywords = append(keywords, "user", "plane", "data", "forwarding")

	case "pcf":

		keywords = append(keywords, "policy", "control", "qos")

	case "udm":

		keywords = append(keywords, "data", "management", "subscription")

	}

	return keywords
}

// GetNetworkFunction retrieves a network function with lazy loading.

func (l *LazyKnowledgeLoader) GetNetworkFunction(name string) (*telecom.NetworkFunctionSpec, bool) {
	l.mu.RLock()

	// Check cache first.

	if nf, ok := l.nfCache.Get(strings.ToLower(name)); ok {

		l.stats.recordHit()

		l.mu.RUnlock()

		return nf, true

	}

	l.stats.recordMiss()

	l.mu.RUnlock()

	// Load from compressed storage or generate on-demand.

	l.mu.Lock()

	defer l.mu.Unlock()

	nf := l.loadNetworkFunction(strings.ToLower(name))

	if nf == nil {
		return nil, false
	}

	// Add to cache.

	l.nfCache.Add(strings.ToLower(name), nf)

	// Update metadata.

	if meta, ok := l.metadata[strings.ToLower(name)]; ok {

		meta.LastAccess = time.Now()

		meta.AccessCount++

	}

	return nf, true
}

// loadNetworkFunction loads or generates a network function specification.

func (l *LazyKnowledgeLoader) loadNetworkFunction(name string) *telecom.NetworkFunctionSpec {
	// Check if we have compressed data.

	if data, ok := l.compressedData[name]; ok {

		nf := &telecom.NetworkFunctionSpec{}

		if err := l.decompress(data, nf); err == nil {
			return nf
		}

	}

	// Generate on-demand based on the function name.

	switch name {

	case "amf":

		return l.generateAMFSpec()

	case "smf":

		return l.generateSMFSpec()

	case "upf":

		return l.generateUPFSpec()

	case "pcf":

		return l.generatePCFSpec()

	case "udm":

		return l.generateUDMSpec()

	case "ausf":

		return l.generateAUSFSpec()

	case "nrf":

		return l.generateNRFSpec()

	case "nssf":

		return l.generateNSSFSpec()

	default:

		return nil

	}
}

// generateAMFSpec generates AMF specification on-demand.

func (l *LazyKnowledgeLoader) generateAMFSpec() *telecom.NetworkFunctionSpec {
	return &telecom.NetworkFunctionSpec{
		Name: "AMF",

		Type: "5gc-control-plane",

		Description: "Access and Mobility Management Function",

		Version: "R17",

		Vendor: "multi-vendor",

		Specification3GPP: []string{"TS 23.501", "TS 23.502"},

		Interfaces: []string{"N1", "N2", "N8", "N11"},

		ServiceInterfaces: []string{"Namf_Communication", "Namf_EventExposure"},

		Dependencies: []string{"AUSF", "UDM", "PCF", "SMF"},

		Resources: telecom.ResourceRequirements{
			MinCPU: "2",

			MinMemory: "4Gi",

			MaxCPU: "8",

			MaxMemory: "16Gi",

			Storage: "100Gi",

			NetworkBW: "10Gbps",
		},

		Scaling: telecom.ScalingParameters{
			MinReplicas: 3,

			MaxReplicas: 20,

			TargetCPU: 70,

			TargetMemory: 80,

			ScaleUpThreshold: 80,

			ScaleDownDelay: 300,
		},

		Performance: telecom.PerformanceBaseline{
			MaxThroughputRPS: 10000,

			AvgLatencyMs: 50,

			P95LatencyMs: 100,

			P99LatencyMs: 200,

			MaxConcurrentSessions: 100000,
		},
	}
}

// generateSMFSpec generates SMF specification on-demand.

func (l *LazyKnowledgeLoader) generateSMFSpec() *telecom.NetworkFunctionSpec {
	return &telecom.NetworkFunctionSpec{
		Name: "SMF",

		Type: "5gc-control-plane",

		Description: "Session Management Function",

		Version: "R17",

		Vendor: "multi-vendor",

		Specification3GPP: []string{"TS 23.501", "TS 23.502"},

		Interfaces: []string{"N4", "N7", "N10", "N11"},

		ServiceInterfaces: []string{"Nsmf_PDUSession", "Nsmf_EventExposure"},

		Dependencies: []string{"UPF", "PCF", "UDM", "AMF"},

		Resources: telecom.ResourceRequirements{
			MinCPU: "2",

			MinMemory: "4Gi",

			MaxCPU: "6",

			MaxMemory: "12Gi",

			Storage: "50Gi",

			NetworkBW: "5Gbps",
		},

		Scaling: telecom.ScalingParameters{
			MinReplicas: 2,

			MaxReplicas: 15,

			TargetCPU: 70,

			TargetMemory: 75,

			ScaleUpThreshold: 75,

			ScaleDownDelay: 300,
		},

		Performance: telecom.PerformanceBaseline{
			MaxThroughputRPS: 5000,

			AvgLatencyMs: 100,

			P95LatencyMs: 200,

			P99LatencyMs: 400,

			MaxConcurrentSessions: 200000,
		},
	}
}

// generateUPFSpec generates UPF specification on-demand.

func (l *LazyKnowledgeLoader) generateUPFSpec() *telecom.NetworkFunctionSpec {
	return &telecom.NetworkFunctionSpec{
		Name: "UPF",

		Type: "5gc-user-plane",

		Description: "User Plane Function",

		Version: "R17",

		Vendor: "multi-vendor",

		Specification3GPP: []string{"TS 23.501", "TS 23.502"},

		Interfaces: []string{"N3", "N4", "N6", "N9"},

		Dependencies: []string{"SMF"},

		Resources: telecom.ResourceRequirements{
			MinCPU: "4",

			MinMemory: "8Gi",

			MaxCPU: "16",

			MaxMemory: "32Gi",

			Storage: "200Gi",

			NetworkBW: "100Gbps",

			Accelerator: "intel.com/qat",
		},

		Scaling: telecom.ScalingParameters{
			MinReplicas: 2,

			MaxReplicas: 10,

			TargetCPU: 60,

			TargetMemory: 70,

			ScaleUpThreshold: 70,

			ScaleDownDelay: 600,
		},

		Performance: telecom.PerformanceBaseline{
			MaxThroughputRPS: 100000,

			AvgLatencyMs: 10,

			P95LatencyMs: 20,

			P99LatencyMs: 50,

			MaxConcurrentSessions: 1000000,
		},
	}
}

// generatePCFSpec generates PCF specification on-demand.

func (l *LazyKnowledgeLoader) generatePCFSpec() *telecom.NetworkFunctionSpec {
	return &telecom.NetworkFunctionSpec{
		Name: "PCF",

		Type: "5gc-control-plane",

		Description: "Policy Control Function",

		Version: "R17",

		Specification3GPP: []string{"TS 23.501", "TS 23.503"},

		Interfaces: []string{"N5", "N7", "N15"},

		ServiceInterfaces: []string{"Npcf_PolicyControl"},

		Dependencies: []string{"UDR"},

		Resources: telecom.ResourceRequirements{
			MinCPU: "1",

			MinMemory: "2Gi",

			MaxCPU: "4",

			MaxMemory: "8Gi",

			Storage: "50Gi",
		},

		Performance: telecom.PerformanceBaseline{
			MaxThroughputRPS: 2000,

			AvgLatencyMs: 75,

			P95LatencyMs: 150,
		},
	}
}

// generateUDMSpec generates UDM specification on-demand.

func (l *LazyKnowledgeLoader) generateUDMSpec() *telecom.NetworkFunctionSpec {
	return &telecom.NetworkFunctionSpec{
		Name: "UDM",

		Type: "5gc-control-plane",

		Description: "Unified Data Management",

		Version: "R17",

		Specification3GPP: []string{"TS 23.501", "TS 29.503"},

		Interfaces: []string{"N8", "N10", "N13"},

		ServiceInterfaces: []string{"Nudm_SubscriberDataManagement"},

		Dependencies: []string{"UDR"},

		Resources: telecom.ResourceRequirements{
			MinCPU: "2",

			MinMemory: "4Gi",

			MaxCPU: "6",

			MaxMemory: "12Gi",

			Storage: "100Gi",
		},

		Performance: telecom.PerformanceBaseline{
			MaxThroughputRPS: 8000,

			AvgLatencyMs: 60,

			P95LatencyMs: 120,
		},
	}
}

// generateAUSFSpec generates AUSF specification on-demand.

func (l *LazyKnowledgeLoader) generateAUSFSpec() *telecom.NetworkFunctionSpec {
	return &telecom.NetworkFunctionSpec{
		Name: "AUSF",

		Type: "5gc-control-plane",

		Description: "Authentication Server Function",

		Version: "R17",

		Specification3GPP: []string{"TS 23.501", "TS 33.501"},

		Interfaces: []string{"N12", "N13"},

		ServiceInterfaces: []string{"Nausf_UEAuthentication"},

		Dependencies: []string{"UDM"},

		Resources: telecom.ResourceRequirements{
			MinCPU: "1",

			MinMemory: "2Gi",

			MaxCPU: "4",

			MaxMemory: "8Gi",

			Storage: "20Gi",
		},

		Performance: telecom.PerformanceBaseline{
			MaxThroughputRPS: 5000,

			AvgLatencyMs: 80,

			P95LatencyMs: 160,
		},
	}
}

// generateNRFSpec generates NRF specification on-demand.

func (l *LazyKnowledgeLoader) generateNRFSpec() *telecom.NetworkFunctionSpec {
	return &telecom.NetworkFunctionSpec{
		Name: "NRF",

		Type: "5gc-control-plane",

		Description: "Network Repository Function",

		Version: "R17",

		Specification3GPP: []string{"TS 23.501", "TS 29.510"},

		ServiceInterfaces: []string{"Nnrf_NFManagement", "Nnrf_NFDiscovery"},

		Resources: telecom.ResourceRequirements{
			MinCPU: "1",

			MinMemory: "2Gi",

			MaxCPU: "3",

			MaxMemory: "6Gi",

			Storage: "50Gi",
		},

		Performance: telecom.PerformanceBaseline{
			MaxThroughputRPS: 15000,

			AvgLatencyMs: 30,

			P95LatencyMs: 60,
		},
	}
}

// generateNSSFSpec generates NSSF specification on-demand.

func (l *LazyKnowledgeLoader) generateNSSFSpec() *telecom.NetworkFunctionSpec {
	return &telecom.NetworkFunctionSpec{
		Name: "NSSF",

		Type: "5gc-control-plane",

		Description: "Network Slice Selection Function",

		Version: "R17",

		Specification3GPP: []string{"TS 23.501", "TS 29.531"},

		Interfaces: []string{"N22"},

		ServiceInterfaces: []string{"Nnssf_NSSelection"},

		Dependencies: []string{"NRF"},

		Resources: telecom.ResourceRequirements{
			MinCPU: "1",

			MinMemory: "1Gi",

			MaxCPU: "2",

			MaxMemory: "4Gi",

			Storage: "20Gi",
		},

		Performance: telecom.PerformanceBaseline{
			MaxThroughputRPS: 3000,

			AvgLatencyMs: 40,

			P95LatencyMs: 80,
		},
	}
}

// GetInterface retrieves an interface specification with lazy loading.

func (l *LazyKnowledgeLoader) GetInterface(name string) (*telecom.InterfaceSpec, bool) {
	l.mu.RLock()

	// Check cache first.

	if iface, ok := l.interfaceCache.Get(strings.ToLower(name)); ok {

		l.stats.recordHit()

		l.mu.RUnlock()

		return iface, true

	}

	l.stats.recordMiss()

	l.mu.RUnlock()

	// Load on-demand.

	l.mu.Lock()

	defer l.mu.Unlock()

	iface := l.loadInterface(strings.ToLower(name))

	if iface == nil {
		return nil, false
	}

	l.interfaceCache.Add(strings.ToLower(name), iface)

	return iface, true
}

// loadInterface loads or generates an interface specification.

func (l *LazyKnowledgeLoader) loadInterface(name string) *telecom.InterfaceSpec {
	switch name {

	case "n1":

		return &telecom.InterfaceSpec{
			Name: "N1",

			Type: "reference-point",

			Protocol: []string{"NAS"},

			Specification: "3GPP TS 24.501",

			Description: "Interface between UE and AMF",
		}

	case "n2":

		return &telecom.InterfaceSpec{
			Name: "N2",

			Type: "reference-point",

			Protocol: []string{"NGAP"},

			Specification: "3GPP TS 38.413",

			Description: "Interface between gNB and AMF",
		}

	case "n3":

		return &telecom.InterfaceSpec{
			Name: "N3",

			Type: "reference-point",

			Protocol: []string{"GTP-U"},

			Specification: "3GPP TS 29.281",

			Description: "Interface between gNB and UPF",
		}

	case "n4":

		return &telecom.InterfaceSpec{
			Name: "N4",

			Type: "reference-point",

			Protocol: []string{"PFCP"},

			Specification: "3GPP TS 29.244",

			Description: "Interface between SMF and UPF",
		}

	default:

		return nil

	}
}

// GetQosProfile retrieves a QoS profile with lazy loading.

func (l *LazyKnowledgeLoader) GetQosProfile(name string) (*telecom.QosProfile, bool) {
	l.mu.RLock()

	if qos, ok := l.qosCache.Get(strings.ToLower(name)); ok {

		l.stats.recordHit()

		l.mu.RUnlock()

		return qos, true

	}

	l.stats.recordMiss()

	l.mu.RUnlock()

	l.mu.Lock()

	defer l.mu.Unlock()

	qos := l.loadQosProfile(strings.ToLower(name))

	if qos == nil {
		return nil, false
	}

	l.qosCache.Add(strings.ToLower(name), qos)

	return qos, true
}

// loadQosProfile loads or generates a QoS profile.

func (l *LazyKnowledgeLoader) loadQosProfile(name string) *telecom.QosProfile {
	switch name {

	case "5qi_1":

		return &telecom.QosProfile{
			QCI: 1,

			QFI: 1,

			Resource: "GBR",

			Priority: 2,

			DelayBudget: 100,

			ErrorRate: 0.01,

			MaxBitrateUL: "150",

			MaxBitrateDL: "150",

			GuaranteedBR: "64",
		}

	case "5qi_9":

		return &telecom.QosProfile{
			QCI: 9,

			QFI: 9,

			Resource: "Non-GBR",

			Priority: 9,

			DelayBudget: 300,

			ErrorRate: 0.000001,

			MaxBitrateUL: "unlimited",

			MaxBitrateDL: "unlimited",
		}

	default:

		return nil

	}
}

// GetSliceType retrieves a slice type with lazy loading.

func (l *LazyKnowledgeLoader) GetSliceType(name string) (*telecom.SliceTypeSpec, bool) {
	l.mu.RLock()

	if slice, ok := l.sliceCache.Get(strings.ToLower(name)); ok {

		l.stats.recordHit()

		l.mu.RUnlock()

		return slice, true

	}

	l.stats.recordMiss()

	l.mu.RUnlock()

	l.mu.Lock()

	defer l.mu.Unlock()

	slice := l.loadSliceType(strings.ToLower(name))

	if slice == nil {
		return nil, false
	}

	l.sliceCache.Add(strings.ToLower(name), slice)

	return slice, true
}

// loadSliceType loads or generates a slice type specification.

func (l *LazyKnowledgeLoader) loadSliceType(name string) *telecom.SliceTypeSpec {
	switch name {

	case "embb":

		return &telecom.SliceTypeSpec{
			SST: 1,

			Description: "Enhanced Mobile Broadband",

			UseCase: "High-speed data services",

			QosProfile: "5qi_9",

			NetworkFunctions: []string{"amf", "smf", "upf", "pcf", "udm"},
		}

	case "urllc":

		return &telecom.SliceTypeSpec{
			SST: 2,

			Description: "Ultra-Reliable Low Latency Communications",

			UseCase: "Industrial automation",

			QosProfile: "5qi_82",

			NetworkFunctions: []string{"amf", "smf", "upf", "pcf"},
		}

	case "mmtc":

		return &telecom.SliceTypeSpec{
			SST: 3,

			Description: "Massive Machine Type Communications",

			UseCase: "IoT sensors",

			NetworkFunctions: []string{"amf", "smf", "upf"},
		}

	default:

		return nil

	}
}

// FindResourcesByKeywords finds resources matching the given keywords.

func (l *LazyKnowledgeLoader) FindResourcesByKeywords(keywords []string) []string {
	l.mu.RLock()

	defer l.mu.RUnlock()

	resourceSet := make(map[string]bool)

	for _, keyword := range keywords {

		keyword = strings.ToLower(keyword)

		if resources, ok := l.keywordIndex[keyword]; ok {
			for _, resource := range resources {
				resourceSet[resource] = true
			}
		}

	}

	// Convert set to slice.

	var results []string

	for resource := range resourceSet {
		results = append(results, resource)
	}

	return results
}

// compress compresses data using gzip.

func (l *LazyKnowledgeLoader) compress(data interface{}) ([]byte, error) {
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)

	enc := gob.NewEncoder(gz)

	if err := enc.Encode(data); err != nil {

		gz.Close()

		return nil, err

	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompress decompresses data.

func (l *LazyKnowledgeLoader) decompress(data []byte, target interface{}) error {
	buf := bytes.NewBuffer(data)

	gz, err := gzip.NewReader(buf)
	if err != nil {
		return err
	}

	defer gz.Close()

	dec := gob.NewDecoder(gz)

	return dec.Decode(target)
}

// GetStats returns cache statistics.

func (l *LazyKnowledgeLoader) GetStats() map[string]interface{} {
	l.stats.mu.RLock()

	defer l.stats.mu.RUnlock()

	hitRate := float64(0)

	if total := l.stats.Hits + l.stats.Misses; total > 0 {
		hitRate = float64(l.stats.Hits) / float64(total) * 100
	}

	return json.RawMessage(`{}`)
}

// ClearCache clears all caches.

func (l *LazyKnowledgeLoader) ClearCache() {
	l.mu.Lock()

	defer l.mu.Unlock()

	l.nfCache.Purge()

	l.interfaceCache.Purge()

	l.qosCache.Purge()

	l.sliceCache.Purge()
}

// recordHit records a cache hit.

func (s *CacheStats) recordHit() {
	s.mu.Lock()

	defer s.mu.Unlock()

	s.Hits++
}

// recordMiss records a cache miss.

func (s *CacheStats) recordMiss() {
	s.mu.Lock()

	defer s.mu.Unlock()

	s.Misses++
}

// ListNetworkFunctions returns available network function names.

func (l *LazyKnowledgeLoader) ListNetworkFunctions() []string {
	l.mu.RLock()

	defer l.mu.RUnlock()

	var names []string

	for id, meta := range l.metadata {
		if meta.Type == "nf" {
			names = append(names, id)
		}
	}

	return names
}

// PreloadByIntent preloads resources based on intent keywords.

func (l *LazyKnowledgeLoader) PreloadByIntent(intent string) {
	// Extract keywords from intent.

	keywords := strings.Fields(strings.ToLower(intent))

	// Find matching resources.

	resources := l.FindResourcesByKeywords(keywords)

	// Preload found resources.

	for _, resource := range resources {
		if meta, ok := l.metadata[resource]; ok {
			switch meta.Type {

			case "nf":

				l.GetNetworkFunction(resource)

			case "interface":

				l.GetInterface(resource)

			case "slice":

				l.GetSliceType(resource)

			case "qos":

				l.GetQosProfile(resource)

			}
		}
	}
}

// GetMemoryUsage estimates current memory usage.

func (l *LazyKnowledgeLoader) GetMemoryUsage() int64 {
	l.mu.RLock()

	defer l.mu.RUnlock()

	// Rough estimation based on cache sizes.

	nfCacheSize := l.nfCache.Len() * 10240 // ~10KB per NF

	interfaceCacheSize := l.interfaceCache.Len() * 2048 // ~2KB per interface

	qosCacheSize := l.qosCache.Len() * 1024 // ~1KB per QoS profile

	sliceCacheSize := l.sliceCache.Len() * 5120 // ~5KB per slice

	compressedSize := 0

	for _, data := range l.compressedData {
		compressedSize += len(data)
	}

	return int64(nfCacheSize + interfaceCacheSize + qosCacheSize + sliceCacheSize + compressedSize)
}

// IsInitialized returns whether the loader is initialized.

func (l *LazyKnowledgeLoader) IsInitialized() bool {
	return len(l.metadata) > 0
}

// SaveToFile saves compressed knowledge base to a file.

func (l *LazyKnowledgeLoader) SaveToFile(filename string) error {
	l.mu.RLock()

	defer l.mu.RUnlock()

	data := json.RawMessage(`{}`)

	compressed, err := l.compress(data)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	// In production, write to file.

	// For now, just return nil.

	_ = compressed

	return nil
}

// LoadFromFile loads compressed knowledge base from a file.

func (l *LazyKnowledgeLoader) LoadFromFile(filename string) error {
	// In production, implement file loading.

	// For now, just return nil.

	return nil
}
