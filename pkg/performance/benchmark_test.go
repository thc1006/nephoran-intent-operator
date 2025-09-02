package performance

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// BenchmarkStringOperations tests string building performance optimizations
func BenchmarkStringOperations(b *testing.B) {
	testStrings := []string{
		"backend",
		"model",
		"intent-type",
		"this is a test intent that we want to process",
	}

	b.Run("StringBuilder", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var builder strings.Builder
			totalLen := 0
			for _, s := range testStrings {
				totalLen += len(s)
			}
			totalLen += len(testStrings) - 1 // separators

			builder.Grow(totalLen)
			for j, s := range testStrings {
				if j > 0 {
					builder.WriteByte(':')
				}
				builder.WriteString(s)
			}
			_ = builder.String()
		}
	})

	b.Run("StringJoin", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = strings.Join(testStrings, ":")
		}
	})
}

// PERFORMANCE OPTIMIZATION BENCHMARKS

// Sample data structures for performance benchmarking
type ComplexStruct struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    json.RawMessage `json:"metadata"`
	Items       []Item                 `json:"items"`
	Status      string                 `json:"status"`
	Version     int                    `json:"version"`
	Config2     Config2                `json:"config"`
	Permissions []string               `json:"permissions"`
}

type Item struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Value    float64                `json:"value"`
	Tags     []string               `json:"tags"`
	Settings json.RawMessage `json:"settings"`
}

type Config2 struct {
	Enabled     bool              `json:"enabled"`
	Timeout     time.Duration     `json:"timeout"`
	RetryCount  int               `json:"retry_count"`
	Options     map[string]string `json:"options"`
	Thresholds  []float64         `json:"thresholds"`
	Environment string            `json:"environment"`
}

// createSampleData creates a complex data structure for benchmarking
func createSampleData(size int) *ComplexStruct {
	items := make([]Item, size)
	for i := 0; i < size; i++ {
		items[i] = Item{
			ID:    "item-" + string(rune('a'+i%26)),
			Type:  "type-" + string(rune('A'+i%10)),
			Value: float64(i) * 3.14159,
			Tags:  []string{"tag1", "tag2", "tag3"},
			Settings: json.RawMessage(`{}`),
		}
	}

	return &ComplexStruct{
		ID:        "test-struct-123",
		Name:      "Test Complex Structure",
		Timestamp: time.Now(),
		Metadata: json.RawMessage(`{}`),
		Items:   items,
		Status:  "active",
		Version: 42,
		Config2: Config2{
			Enabled:    true,
			Timeout:    30 * time.Second,
			RetryCount: 3,
			Options: map[string]string{
				"format": "json",
				"level":  "debug",
			},
			Thresholds:  []float64{0.1, 0.5, 0.9, 1.0},
			Environment: "production",
		},
		Permissions: []string{"read", "write", "execute", "admin"},
	}
}

// Benchmark SONIC vs standard JSON marshaling
func BenchmarkSONICMarshal_Small(b *testing.B) {
	data := createSampleData(10)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := JSONMarshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSONICMarshal_Medium(b *testing.B) {
	data := createSampleData(100)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := JSONMarshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSONICMarshal_Large(b *testing.B) {
	data := createSampleData(1000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := JSONMarshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStandardMarshal_Small(b *testing.B) {
	data := createSampleData(10)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStandardMarshal_Medium(b *testing.B) {
	data := createSampleData(100)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStandardMarshal_Large(b *testing.B) {
	data := createSampleData(1000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark SONIC vs standard JSON unmarshaling
func BenchmarkSONICUnmarshal_Small(b *testing.B) {
	data := createSampleData(10)
	jsonData, _ := json.Marshal(data)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result ComplexStruct
		err := JSONUnmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSONICUnmarshal_Medium(b *testing.B) {
	data := createSampleData(100)
	jsonData, _ := json.Marshal(data)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result ComplexStruct
		err := JSONUnmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSONICUnmarshal_Large(b *testing.B) {
	data := createSampleData(1000)
	jsonData, _ := json.Marshal(data)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result ComplexStruct
		err := JSONUnmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark metrics collection overhead
func BenchmarkMetricsCollection(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		metrics.RecordRequestDuration("GET", "/api/v1/intents", "200", time.Millisecond*10)
		metrics.RecordJSONMarshal(time.Microsecond*100, nil)
		metrics.RecordJSONUnmarshal(time.Microsecond*150, nil)
		metrics.RecordCacheHit("intent-cache")
		metrics.RecordIntentProcessing("scaling", "success", time.Millisecond*50)
	}
}

// Benchmark HTTP middleware overhead
func BenchmarkHTTPMiddlewareOverhead(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)
	middleware := NewHTTPMiddleware(metrics)

	// Create a simple handler
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// Test data to verify SONIC compatibility
func TestSONICCompatibility(t *testing.T) {
	data := createSampleData(50)

	// Marshal with SONIC
	sonicJSON, err := JSONMarshal(data)
	if err != nil {
		t.Fatalf("SONIC marshal failed: %v", err)
	}

	// Marshal with standard library
	stdJSON, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Standard marshal failed: %v", err)
	}

	// Unmarshal SONIC JSON with standard library
	var stdResult ComplexStruct
	err = json.Unmarshal(sonicJSON, &stdResult)
	if err != nil {
		t.Fatalf("Standard unmarshal of SONIC JSON failed: %v", err)
	}

	// Unmarshal standard JSON with SONIC
	var sonicResult ComplexStruct
	err = JSONUnmarshal(stdJSON, &sonicResult)
	if err != nil {
		t.Fatalf("SONIC unmarshal of standard JSON failed: %v", err)
	}

	// Verify both results match original
	if stdResult.ID != data.ID || sonicResult.ID != data.ID {
		t.Error("SONIC and standard JSON are not compatible")
	}

	if len(stdResult.Items) != len(data.Items) || len(sonicResult.Items) != len(data.Items) {
		t.Error("Item count mismatch between SONIC and standard JSON")
	}
}

// Benchmark configuration parsing
func BenchmarkConfigLoad(b *testing.B) {
	config := DefaultConfig()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate config operations
		if config.UseSONIC {
			data := createSampleData(10)
			_, _ = JSONMarshal(data)
		}

		if config.EnableHTTP2 {
			_ = SetupOptimizedTransport(config)
		}
	}
}

// Benchmark concurrent JSON processing
func BenchmarkConcurrentJSONProcessing(b *testing.B) {
	data := createSampleData(100)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			jsonData, err := JSONMarshal(data)
			if err != nil {
				b.Fatal(err)
			}

			var result ComplexStruct
			err = JSONUnmarshal(jsonData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Benchmark HTTP2 transport setup
func BenchmarkHTTP2TransportSetup(b *testing.B) {
	config := DefaultConfig()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		transport := SetupOptimizedTransport(config)
		if transport == nil {
			b.Fatal("Failed to create transport")
		}
	}
}

// Benchmark metrics registry operations
func BenchmarkMetricsRegistryOperations(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		// Perform some metric operations
		metrics.RecordRequestDuration("GET", "/test", "200", time.Millisecond)
		metrics.RecordCacheHit("test-cache")
		metrics.RecordIntentProcessing("test-intent", "success", time.Millisecond*10)
	}
}

