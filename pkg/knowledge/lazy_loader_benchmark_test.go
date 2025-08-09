package knowledge

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

// BenchmarkOriginalKnowledgeBase benchmarks the original implementation
func BenchmarkOriginalKnowledgeBase(b *testing.B) {
	b.ReportAllocs()

	// Measure initial memory
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialAlloc := m.Alloc

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		kb := telecom.NewTelecomKnowledgeBase()
		_ = kb.IsInitialized()
	}

	b.StopTimer()

	// Measure final memory
	runtime.ReadMemStats(&m)
	finalAlloc := m.Alloc
	memoryUsed := finalAlloc - initialAlloc

	b.ReportMetric(float64(memoryUsed)/1024/1024, "MB/op")
}

// BenchmarkLazyLoadingKnowledgeBase benchmarks the lazy loading implementation
func BenchmarkLazyLoadingKnowledgeBase(b *testing.B) {
	b.ReportAllocs()

	// Measure initial memory
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialAlloc := m.Alloc

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		loader, err := NewLazyKnowledgeLoader(DefaultLoaderConfig())
		if err != nil {
			b.Fatal(err)
		}
		_ = loader.IsInitialized()
	}

	b.StopTimer()

	// Measure final memory
	runtime.ReadMemStats(&m)
	finalAlloc := m.Alloc
	memoryUsed := finalAlloc - initialAlloc

	b.ReportMetric(float64(memoryUsed)/1024/1024, "MB/op")
}

// BenchmarkNetworkFunctionAccess benchmarks accessing network functions
func BenchmarkNetworkFunctionAccess(b *testing.B) {
	b.Run("Original", func(b *testing.B) {
		kb := telecom.NewTelecomKnowledgeBase()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = kb.GetNetworkFunction("amf")
			_, _ = kb.GetNetworkFunction("smf")
			_, _ = kb.GetNetworkFunction("upf")
		}
	})

	b.Run("LazyLoading", func(b *testing.B) {
		loader, _ := NewLazyKnowledgeLoader(DefaultLoaderConfig())
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = loader.GetNetworkFunction("amf")
			_, _ = loader.GetNetworkFunction("smf")
			_, _ = loader.GetNetworkFunction("upf")
		}
	})
}

// BenchmarkMemoryUsageUnderLoad simulates memory usage under load
func BenchmarkMemoryUsageUnderLoad(b *testing.B) {
	b.Run("Original", func(b *testing.B) {
		kb := telecom.NewTelecomKnowledgeBase()

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		initialAlloc := m.Alloc

		// Simulate accessing all network functions
		for _, nf := range kb.ListNetworkFunctions() {
			_, _ = kb.GetNetworkFunction(nf)
		}

		runtime.ReadMemStats(&m)
		memoryUsed := m.Alloc - initialAlloc

		b.ReportMetric(float64(memoryUsed)/1024/1024, "MB")
	})

	b.Run("LazyLoading", func(b *testing.B) {
		loader, _ := NewLazyKnowledgeLoader(DefaultLoaderConfig())

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		initialAlloc := m.Alloc

		// Simulate accessing all network functions
		for _, nf := range loader.ListNetworkFunctions() {
			_, _ = loader.GetNetworkFunction(nf)
		}

		runtime.ReadMemStats(&m)
		memoryUsed := m.Alloc - initialAlloc

		b.ReportMetric(float64(memoryUsed)/1024/1024, "MB")
	})
}

// TestMemoryOptimization tests memory optimization effectiveness
func TestMemoryOptimization(t *testing.T) {
	// Create original knowledge base
	originalKB := telecom.NewTelecomKnowledgeBase()

	// Create lazy loading knowledge base
	lazyLoader, err := NewLazyKnowledgeLoader(DefaultLoaderConfig())
	if err != nil {
		t.Fatalf("Failed to create lazy loader: %v", err)
	}

	// Force garbage collection to get accurate measurements
	runtime.GC()
	runtime.GC()

	// Measure original memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	originalMemory := m.Alloc

	// Access some functions from original
	_, _ = originalKB.GetNetworkFunction("amf")
	_, _ = originalKB.GetNetworkFunction("smf")

	runtime.ReadMemStats(&m)
	originalUsed := m.Alloc - originalMemory

	// Force garbage collection again
	runtime.GC()
	runtime.GC()

	// Measure lazy loading memory usage
	runtime.ReadMemStats(&m)
	lazyMemory := m.Alloc

	// Access same functions from lazy loader
	_, _ = lazyLoader.GetNetworkFunction("amf")
	_, _ = lazyLoader.GetNetworkFunction("smf")

	runtime.ReadMemStats(&m)
	lazyUsed := m.Alloc - lazyMemory

	// Calculate reduction
	reduction := float64(originalUsed-lazyUsed) / float64(originalUsed) * 100

	t.Logf("Original memory usage: %.2f MB", float64(originalUsed)/1024/1024)
	t.Logf("Lazy loading memory usage: %.2f MB", float64(lazyUsed)/1024/1024)
	t.Logf("Memory reduction: %.2f%%", reduction)

	// Verify lazy loading uses less memory
	if lazyUsed >= originalUsed {
		t.Errorf("Lazy loading should use less memory. Original: %d, Lazy: %d", originalUsed, lazyUsed)
	}
}

// TestCacheEffectiveness tests cache hit rate
func TestCacheEffectiveness(t *testing.T) {
	loader, err := NewLazyKnowledgeLoader(DefaultLoaderConfig())
	if err != nil {
		t.Fatalf("Failed to create lazy loader: %v", err)
	}

	// Access the same function multiple times
	for i := 0; i < 100; i++ {
		_, _ = loader.GetNetworkFunction("amf")
	}

	stats := loader.GetStats()

	hitRate := stats["hit_rate"].(float64)
	t.Logf("Cache hit rate: %.2f%%", hitRate)

	// Should have high hit rate (>95%)
	if hitRate < 95.0 {
		t.Errorf("Cache hit rate too low: %.2f%% (expected >95%%)", hitRate)
	}
}

// TestLazyLoadingCorrectness verifies lazy loading returns correct data
func TestLazyLoadingCorrectness(t *testing.T) {
	originalKB := telecom.NewTelecomKnowledgeBase()
	lazyLoader, err := NewLazyKnowledgeLoader(DefaultLoaderConfig())
	if err != nil {
		t.Fatalf("Failed to create lazy loader: %v", err)
	}

	testCases := []string{"amf", "smf", "upf", "pcf", "udm", "ausf", "nrf", "nssf"}

	for _, nfName := range testCases {
		t.Run(fmt.Sprintf("NetworkFunction_%s", nfName), func(t *testing.T) {
			origNF, origExists := originalKB.GetNetworkFunction(nfName)
			lazyNF, lazyExists := lazyLoader.GetNetworkFunction(nfName)

			if origExists != lazyExists {
				t.Errorf("Existence mismatch for %s: original=%v, lazy=%v", nfName, origExists, lazyExists)
			}

			if origExists && lazyExists {
				// Compare key fields
				if origNF.Name != lazyNF.Name {
					t.Errorf("Name mismatch for %s: original=%s, lazy=%s", nfName, origNF.Name, lazyNF.Name)
				}
				if origNF.Type != lazyNF.Type {
					t.Errorf("Type mismatch for %s: original=%s, lazy=%s", nfName, origNF.Type, lazyNF.Type)
				}
				if origNF.Version != lazyNF.Version {
					t.Errorf("Version mismatch for %s: original=%s, lazy=%s", nfName, origNF.Version, lazyNF.Version)
				}
			}
		})
	}
}

// TestPreloadByIntent tests intent-based preloading
func TestPreloadByIntent(t *testing.T) {
	loader, err := NewLazyKnowledgeLoader(DefaultLoaderConfig())
	if err != nil {
		t.Fatalf("Failed to create lazy loader: %v", err)
	}

	// Clear cache first
	loader.ClearCache()

	// Preload based on intent
	intent := "Deploy AMF and SMF with high availability"
	loader.PreloadByIntent(intent)

	// Check if AMF and SMF are in cache (should be cache hits)
	_, _ = loader.GetNetworkFunction("amf")
	_, _ = loader.GetNetworkFunction("smf")

	stats := loader.GetStats()
	hits := stats["hits"].(int64)

	if hits < 2 {
		t.Errorf("Expected at least 2 cache hits after preloading, got %d", hits)
	}
}

// BenchmarkIntentBasedLoading benchmarks intent-based loading
func BenchmarkIntentBasedLoading(b *testing.B) {
	loader, _ := NewLazyKnowledgeLoader(DefaultLoaderConfig())

	intents := []string{
		"Deploy AMF with high availability",
		"Configure SMF for production",
		"Setup UPF with auto-scaling",
		"Create URLLC slice",
		"Deploy edge computing infrastructure",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		intent := intents[i%len(intents)]
		loader.PreloadByIntent(intent)
	}
}

// TestMemoryLimit tests that memory usage stays within configured limits
func TestMemoryLimit(t *testing.T) {
	config := DefaultLoaderConfig()
	config.MaxMemoryMB = 10 // Set low limit for testing
	config.CacheSize = 5    // Small cache

	loader, err := NewLazyKnowledgeLoader(config)
	if err != nil {
		t.Fatalf("Failed to create lazy loader: %v", err)
	}

	// Access many items to trigger eviction
	for i := 0; i < 20; i++ {
		nfName := fmt.Sprintf("test_nf_%d", i%8)
		_, _ = loader.GetNetworkFunction(nfName)
	}

	// Check memory usage
	memoryUsage := loader.GetMemoryUsage()
	memoryMB := memoryUsage / 1024 / 1024

	t.Logf("Memory usage: %d MB (limit: %d MB)", memoryMB, config.MaxMemoryMB)

	// Memory usage should be reasonable (allowing some overhead)
	if memoryMB > int64(config.MaxMemoryMB*2) {
		t.Errorf("Memory usage exceeds limit: %d MB > %d MB", memoryMB, config.MaxMemoryMB*2)
	}
}
