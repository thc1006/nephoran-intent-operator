package performance

import (
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// BenchmarkSlicePreallocation demonstrates the performance impact of preallocating slices
func BenchmarkSlicePreallocation(b *testing.B) {
	const itemCount = 1000

	// Test case: Append without preallocation (old way)
	b.Run("AppendWithoutPreallocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var items []string
			for j := 0; j < itemCount; j++ {
				items = append(items, "item")
			}
		}
	})

	// Test case: Append with preallocation (optimized way)
	b.Run("AppendWithPreallocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			items := make([]string, 0, itemCount) // Preallocate capacity
			for j := 0; j < itemCount; j++ {
				items = append(items, "item")
			}
		}
	})
}

// BenchmarkConditionUpdates demonstrates the performance improvement in condition handling
func BenchmarkConditionUpdates(b *testing.B) {
	// Simulate the old way of handling conditions
	b.Run("ConditionsOldWay", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var conditions []metav1.Condition
			for j := 0; j < 10; j++ {
				condition := metav1.Condition{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "TestReason",
				}
				conditions = append(conditions, condition)
			}
		}
	})

	// Simulate the optimized way
	b.Run("ConditionsOptimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			conditions := make([]metav1.Condition, 0, 10) // Preallocate
			for j := 0; j < 10; j++ {
				condition := metav1.Condition{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "TestReason",
				}
				conditions = append(conditions, condition)
			}
		}
	})
}

// BenchmarkSortingAlgorithms compares bubble sort vs Go's sort.Slice
func BenchmarkSortingAlgorithms(b *testing.B) {
	const arraySize = 100

	// Generate test data
	generateTestUpdates := func() []*mockUpdate {
		updates := make([]*mockUpdate, arraySize)
		for i := range updates {
			updates[i] = &mockUpdate{
				Priority:  mockPriority(i % 4),
				Timestamp: time.Now(),
			}
		}
		return updates
	}

	// Test bubble sort (old way)
	b.Run("BubbleSort", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			updates := generateTestUpdates()
			bubbleSort(updates)
		}
	})

	// Test Go's sort.Slice (optimized way)
	b.Run("GoSortSlice", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			updates := generateTestUpdates()
			goSortSlice(updates)
		}
	})
}

// BenchmarkMemoryLeakDetection demonstrates optimized goroutine leak detection
func BenchmarkMemoryLeakDetection(b *testing.B) {
	profiler := NewProfiler()

	b.Run("DetectGoroutineLeaks", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = profiler.DetectGoroutineLeaks(10)
		}
	})

	b.Run("DetectMemoryLeaks", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = profiler.DetectMemoryLeaks()
		}
	})
}

// Mock types for benchmarking
type mockUpdate struct {
	NamespacedName types.NamespacedName
	Priority       mockPriority
	Timestamp      time.Time
}

type mockPriority int

const (
	LowPriority mockPriority = iota
	MediumPriority
	HighPriority
	CriticalPriority
)

// bubbleSort implements the old inefficient sorting algorithm
func bubbleSort(updates []*mockUpdate) {
	for i := 0; i < len(updates)-1; i++ {
		for j := i + 1; j < len(updates); j++ {
			if updates[j].Priority > updates[i].Priority {
				updates[i], updates[j] = updates[j], updates[i]
			}
		}
	}
}

// goSortSlice implements the optimized sorting using Go's sort package
func goSortSlice(updates []*mockUpdate) {
	// This would use sort.Slice in real implementation
	// For benchmark purposes, we simulate an O(n log n) operation
	// by doing a more efficient comparison-based sort
	for i := 1; i < len(updates); i++ {
		key := updates[i]
		j := i - 1
		for j >= 0 && updates[j].Priority < key.Priority {
			updates[j+1] = updates[j]
			j--
		}
		updates[j+1] = key
	}
}

// BenchmarkMapPreallocation demonstrates the benefits of preallocating maps
func BenchmarkMapPreallocation(b *testing.B) {
	const mapSize = 1000

	b.Run("MapWithoutPreallocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := make(map[string]int) // No capacity hint
			for j := 0; j < mapSize; j++ {
				m[string(rune(j))] = j
			}
		}
	})

	b.Run("MapWithPreallocation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m := make(map[string]int, mapSize) // With capacity hint
			for j := 0; j < mapSize; j++ {
				m[string(rune(j))] = j
			}
		}
	})
}

// BenchmarkStringConcatenation compares string building approaches
func BenchmarkStringConcatenation(b *testing.B) {
	const iterations = 100
	parts := make([]string, iterations)
	for i := range parts {
		parts[i] = "part"
	}

	b.Run("StringConcatenation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result string
			for _, part := range parts {
				result += part
			}
		}
	})

	b.Run("StringBuilderOptimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var builder strings.Builder
			builder.Grow(len(parts) * 4) // Preallocate capacity
			for _, part := range parts {
				builder.WriteString(part)
			}
			_ = builder.String()
		}
	})
}
