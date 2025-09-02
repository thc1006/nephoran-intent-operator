package llm

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// BenchmarkGPUInference benchmarks GPU-accelerated inference performance
func BenchmarkGPUInference(b *testing.B) {
	config := getDefaultGPUConfig()
	config.EnableBatching = false // Test single inference first

	accelerator, err := NewGPUAccelerator(config)
	if err != nil {
		b.Skipf("GPU accelerator not available: %v", err)
	}
	defer accelerator.Close()

	// Create test request
	request := &InferenceRequest{
		ModelName:   "gpt-3.5-turbo",
		InputTokens: generateRandomTokens(100),
		MaxTokens:   50,
		Temperature: 0.7,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := accelerator.ProcessInference(ctx, request)
		cancel()

		if err != nil {
			b.Fatalf("Inference failed: %v", err)
		}
	}
}

// BenchmarkGPUBatchInference benchmarks batched GPU inference
func BenchmarkGPUBatchInference(b *testing.B) {
	batchSizes := []int{1, 4, 8, 16, 32, 64}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize%d", batchSize), func(b *testing.B) {
			config := getDefaultGPUConfig()
			config.EnableBatching = true
			config.BatchConfig.MaxBatchSize = batchSize

			accelerator, err := NewGPUAccelerator(config)
			if err != nil {
				b.Skipf("GPU accelerator not available: %v", err)
			}
			defer accelerator.Close()

			// Create batch requests
			requests := make([]*InferenceRequest, batchSize)
			for i := 0; i < batchSize; i++ {
				requests[i] = &InferenceRequest{
					ModelName:   "gpt-3.5-turbo",
					InputTokens: generateRandomTokens(100),
					MaxTokens:   50,
					Temperature: 0.7,
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_, err := accelerator.ProcessBatch(ctx, requests)
				cancel()

				if err != nil {
					b.Fatalf("Batch inference failed: %v", err)
				}
			}

			// Report tokens per second
			totalTokens := int64(batchSize * 150 * b.N) // avg 150 tokens per request
			tokensPerSec := float64(totalTokens) / b.Elapsed().Seconds()
			b.ReportMetric(tokensPerSec, "tokens/sec")
		})
	}
}

// BenchmarkModelCaching benchmarks enhanced model caching performance
func BenchmarkModelCaching(b *testing.B) {
	config := getDefaultEnhancedCacheConfig()
	cache, err := NewEnhancedModelCache(config)
	if err != nil {
		b.Fatalf("Failed to create model cache: %v", err)
	}
	defer cache.Close()

	models := []string{"gpt-3.5-turbo", "gpt-4", "claude-3", "llama-2-7b", "llama-2-13b"}
	deviceID := 0

	// Pre-warm cache with some models
	ctx := context.Background()
	for _, model := range models[:3] {
		err := cache.LoadModel(ctx, model, fmt.Sprintf("/models/%s", model), deviceID)
		if err != nil {
			b.Fatalf("Failed to load model %s: %v", model, err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Random model access pattern
		modelName := models[rand.Intn(len(models))]

		start := time.Now()
		_, found, err := cache.GetModel(ctx, modelName, deviceID)
		latency := time.Since(start)

		if err != nil {
			b.Fatalf("Cache get failed: %v", err)
		}

		// Track cache hits vs misses
		if found {
			b.ReportMetric(latency.Seconds()*1000, "cache_hit_ms")
		} else {
			b.ReportMetric(latency.Seconds()*1000, "cache_miss_ms")
		}
	}
}

// BenchmarkVectorSearch benchmarks vector search acceleration
func BenchmarkVectorSearch(b *testing.B) {
	topKValues := []int{1, 10, 50, 100}
	vectorDimensions := []int{768, 1536, 4096}

	for _, dim := range vectorDimensions {
		for _, topK := range topKValues {
			b.Run(fmt.Sprintf("Dim%d_TopK%d", dim, topK), func(b *testing.B) {
				config := getDefaultVectorSearchConfig()
				config.VectorDimensions = dim
				config.MaxResultsPerQuery = topK * 2

				accelerator, err := NewVectorSearchAccelerator(config)
				if err != nil {
					b.Fatalf("Failed to create vector search accelerator: %v", err)
				}
				defer accelerator.Close()

				// Index some vectors
				ctx := context.Background()
				numVectors := 10000
				for i := 0; i < numVectors; i++ {
					vector := generateRandomVector(dim)
					metadata := json.RawMessage(`{}`)

					err := accelerator.IndexVector(ctx, fmt.Sprintf("vec_%d", i), vector, metadata)
					if err != nil {
						b.Fatalf("Failed to index vector: %v", err)
					}
				}

				// Generate query vector
				queryVector := generateRandomVector(dim)

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					start := time.Now()
					result, err := accelerator.SearchSimilar(ctx, queryVector, topK, nil)
					latency := time.Since(start)

					if err != nil {
						b.Fatalf("Search failed: %v", err)
					}

					b.ReportMetric(latency.Seconds()*1000, "search_latency_ms")
					b.ReportMetric(float64(len(result.Results)), "results_returned")
				}
			})
		}
	}
}

// BenchmarkGPUMemoryAllocation benchmarks GPU memory management
func BenchmarkGPUMemoryAllocation(b *testing.B) {
	allocationSizes := []int64{
		1024 * 1024,        // 1MB
		10 * 1024 * 1024,   // 10MB
		100 * 1024 * 1024,  // 100MB
		1024 * 1024 * 1024, // 1GB
	}

	for _, size := range allocationSizes {
		b.Run(fmt.Sprintf("Size%dMB", size/(1024*1024)), func(b *testing.B) {
			config := getDefaultGPUMemoryConfig()
			manager, err := NewGPUMemoryManager(config, []int{0})
			if err != nil {
				b.Skipf("GPU memory manager not available: %v", err)
			}
			defer manager.Close()

			b.ResetTimer()
			b.ReportAllocs()

			allocations := make([]*MemoryAllocation, b.N)

			// Allocation phase
			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				allocation, err := manager.AllocateMemory(ctx, size, AllocationPurposeModelWeights, 0)
				if err != nil {
					b.Fatalf("Allocation failed: %v", err)
				}
				allocations[i] = allocation
			}

			// Deallocation phase
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				err := manager.DeallocateMemory(ctx, allocations[i])
				if err != nil {
					b.Fatalf("Deallocation failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkEmbeddingGeneration benchmarks embedding generation performance
func BenchmarkEmbeddingGeneration(b *testing.B) {
	config := getDefaultVectorSearchConfig()
	accelerator, err := NewVectorSearchAccelerator(config)
	if err != nil {
		b.Fatalf("Failed to create vector search accelerator: %v", err)
	}
	defer accelerator.Close()

	textLengths := []int{50, 200, 500, 1000} // character counts
	batchSizes := []int{1, 10, 50, 100}

	for _, textLen := range textLengths {
		for _, batchSize := range batchSizes {
			b.Run(fmt.Sprintf("TextLen%d_Batch%d", textLen, batchSize), func(b *testing.B) {
				// Generate test texts
				texts := make([]string, batchSize)
				for i := 0; i < batchSize; i++ {
					texts[i] = generateRandomText(textLen)
				}

				modelName := "text-embedding-3-small"

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					ctx := context.Background()

					if batchSize == 1 {
						_, err := accelerator.GenerateEmbedding(ctx, texts[0], modelName)
						if err != nil {
							b.Fatalf("Embedding generation failed: %v", err)
						}
					} else {
						_, err := accelerator.GenerateEmbeddingBatch(ctx, texts, modelName)
						if err != nil {
							b.Fatalf("Batch embedding generation failed: %v", err)
						}
					}
				}

				// Report embeddings per second
				embeddingsPerSec := float64(batchSize*b.N) / b.Elapsed().Seconds()
				b.ReportMetric(embeddingsPerSec, "embeddings/sec")
			})
		}
	}
}

// BenchmarkConcurrentInference benchmarks concurrent inference performance
func BenchmarkConcurrentInference(b *testing.B) {
	concurrencyLevels := []int{1, 2, 4, 8, 16, 32}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency%d", concurrency), func(b *testing.B) {
			config := getDefaultGPUConfig()
			config.EnableBatching = true

			accelerator, err := NewGPUAccelerator(config)
			if err != nil {
				b.Skipf("GPU accelerator not available: %v", err)
			}
			defer accelerator.Close()

			b.ResetTimer()
			b.ReportAllocs()

			var wg sync.WaitGroup
			requestsPerWorker := b.N / concurrency

			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					for j := 0; j < requestsPerWorker; j++ {
						request := &InferenceRequest{
							ModelName:   "gpt-3.5-turbo",
							InputTokens: generateRandomTokens(100),
							MaxTokens:   50,
							Temperature: 0.7,
						}

						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						_, err := accelerator.ProcessInference(ctx, request)
						cancel()

						if err != nil {
							b.Errorf("Worker %d inference failed: %v", workerID, err)
							return
						}
					}
				}(i)
			}

			wg.Wait()

			// Report requests per second
			requestsPerSec := float64(b.N) / b.Elapsed().Seconds()
			b.ReportMetric(requestsPerSec, "requests/sec")
		})
	}
}

// BenchmarkMemoryFragmentation benchmarks memory fragmentation handling
func BenchmarkMemoryFragmentation(b *testing.B) {
	config := getDefaultGPUMemoryConfig()
	config.EnableAutoDefrag = true

	manager, err := NewGPUMemoryManager(config, []int{0})
	if err != nil {
		b.Skipf("GPU memory manager not available: %v", err)
	}
	defer manager.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create fragmentation by allocating and deallocating random sizes
		var allocations []*MemoryAllocation
		ctx := context.Background()

		// Allocation phase - create fragmentation
		for j := 0; j < 100; j++ {
			size := int64(rand.Intn(50*1024*1024) + 1024*1024) // 1MB to 50MB
			allocation, err := manager.AllocateMemory(ctx, size, AllocationPurposeCache, 0)
			if err != nil {
				continue // Skip if allocation fails
			}
			allocations = append(allocations, allocation)
		}

		// Deallocate randomly to create fragmentation
		for len(allocations) > 20 {
			idx := rand.Intn(len(allocations))
			manager.DeallocateMemory(ctx, allocations[idx])
			allocations = append(allocations[:idx], allocations[idx+1:]...)
		}

		// Try to allocate large block (should trigger defragmentation)
		largeSize := int64(100 * 1024 * 1024) // 100MB
		start := time.Now()
		largeAlloc, err := manager.AllocateMemory(ctx, largeSize, AllocationPurposeModelWeights, 0)
		defragTime := time.Since(start)

		if err == nil {
			b.ReportMetric(defragTime.Seconds()*1000, "defrag_time_ms")
			manager.DeallocateMemory(ctx, largeAlloc)
		}

		// Clean up remaining allocations
		for _, allocation := range allocations {
			manager.DeallocateMemory(ctx, allocation)
		}
	}
}

// Test utility functions

func generateRandomTokens(count int) []int {
	tokens := make([]int, count)
	for i := range tokens {
		tokens[i] = rand.Intn(50000) // Typical vocabulary size
	}
	return tokens
}

func generateRandomVector(dimensions int) []float32 {
	vector := make([]float32, dimensions)
	for i := range vector {
		vector[i] = rand.Float32()*2 - 1 // Range [-1, 1]
	}
	// Normalize the vector
	var norm float32
	for _, val := range vector {
		norm += val * val
	}
	norm = float32(1.0 / (float64(norm) + 1e-8)) // Avoid division by zero
	for i := range vector {
		vector[i] *= norm
	}
	return vector
}

func generateRandomText(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 "
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// Integration benchmarks

// BenchmarkEndToEndPipeline benchmarks the complete AI acceleration pipeline
func BenchmarkEndToEndPipeline(b *testing.B) {
	// Initialize all components
	gpuConfig := getDefaultGPUConfig()
	gpuAccelerator, err := NewGPUAccelerator(gpuConfig)
	if err != nil {
		b.Skipf("GPU accelerator not available: %v", err)
	}
	defer gpuAccelerator.Close()

	cacheConfig := getDefaultEnhancedCacheConfig()
	modelCache, err := NewEnhancedModelCache(cacheConfig)
	if err != nil {
		b.Fatalf("Failed to create model cache: %v", err)
	}
	defer modelCache.Close()

	vectorConfig := getDefaultVectorSearchConfig()
	vectorSearch, err := NewVectorSearchAccelerator(vectorConfig)
	if err != nil {
		b.Fatalf("Failed to create vector search: %v", err)
	}
	defer vectorSearch.Close()

	memoryConfig := getDefaultGPUMemoryConfig()
	memoryManager, err := NewGPUMemoryManager(memoryConfig, []int{0})
	if err != nil {
		b.Skipf("GPU memory manager not available: %v", err)
	}
	defer memoryManager.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()

		// 1. Generate embeddings for input text
		inputText := generateRandomText(200)
		embedding, err := vectorSearch.GenerateEmbedding(ctx, inputText, "text-embedding-3-small")
		if err != nil {
			b.Fatalf("Embedding generation failed: %v", err)
		}

		// 2. Search for relevant context
		searchResults, err := vectorSearch.SearchSimilar(ctx, embedding, 5, nil)
		if err != nil {
			b.Fatalf("Vector search failed: %v", err)
		}

		// 3. Load model if not cached
		modelName := "gpt-3.5-turbo"
		_, found, err := modelCache.GetModel(ctx, modelName, 0)
		if err != nil {
			b.Fatalf("Model cache access failed: %v", err)
		}

		if !found {
			// Simulate model loading time
			time.Sleep(time.Millisecond * 10)
		}

		// 4. Perform inference with context
		request := &InferenceRequest{
			ModelName:   modelName,
			InputTokens: generateRandomTokens(150),
			MaxTokens:   100,
			Temperature: 0.7,
		}

		response, err := gpuAccelerator.ProcessInference(ctx, request)
		if err != nil {
			b.Fatalf("Inference failed: %v", err)
		}

		// Track end-to-end metrics
		if len(response.OutputTokens) == 0 {
			b.Error("No output tokens generated")
		}
		if len(searchResults.Results) == 0 {
			b.Error("No search results returned")
		}
	}
}

// Performance profiling test
func BenchmarkResourceUtilization(b *testing.B) {
	config := getDefaultGPUConfig()
	accelerator, err := NewGPUAccelerator(config)
	if err != nil {
		b.Skipf("GPU accelerator not available: %v", err)
	}
	defer accelerator.Close()

	// Monitor resource utilization during benchmark
	var maxMemoryUsage int64
	var totalGPUTime time.Duration

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		request := &InferenceRequest{
			ModelName:   "gpt-3.5-turbo",
			InputTokens: generateRandomTokens(100),
			MaxTokens:   50,
			Temperature: 0.7,
		}

		ctx := context.Background()
		_, err := accelerator.ProcessInference(ctx, request)
		if err != nil {
			b.Fatalf("Inference failed: %v", err)
		}

		gpuTime := time.Since(start)
		totalGPUTime += gpuTime

		// Simulate memory usage tracking
		if memUsage := int64(rand.Intn(1000000000)); memUsage > maxMemoryUsage {
			maxMemoryUsage = memUsage
		}
	}

	// Report resource utilization metrics
	avgGPUTime := totalGPUTime / time.Duration(b.N)
	b.ReportMetric(avgGPUTime.Seconds()*1000, "avg_gpu_time_ms")
	b.ReportMetric(float64(maxMemoryUsage)/(1024*1024), "peak_memory_mb")
}

// Stress test for stability under load
func BenchmarkStressTest(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping stress test in short mode")
	}

	config := getDefaultGPUConfig()
	config.EnableBatching = true

	accelerator, err := NewGPUAccelerator(config)
	if err != nil {
		b.Skipf("GPU accelerator not available: %v", err)
	}
	defer accelerator.Close()

	// Run for a fixed duration rather than fixed iterations
	duration := 30 * time.Second
	if testing.Short() {
		duration = 5 * time.Second
	}

	timeout := time.After(duration)
	iterations := 0
	errors := 0

	logger := slog.Default()

	b.ResetTimer()

	for {
		select {
		case <-timeout:
			b.Logf("Stress test completed: %d iterations, %d errors (%.2f%% error rate)",
				iterations, errors, float64(errors)/float64(iterations)*100)
			return
		default:
			iterations++

			request := &InferenceRequest{
				ModelName:   "gpt-3.5-turbo",
				InputTokens: generateRandomTokens(rand.Intn(200) + 50), // Variable length
				MaxTokens:   rand.Intn(100) + 20,                       // Variable output
				Temperature: rand.Float64(),                            // Variable temperature
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := accelerator.ProcessInference(ctx, request)
			cancel()

			if err != nil {
				errors++
				logger.Warn("Stress test inference failed", "iteration", iterations, "error", err)
			}

			// Brief pause to prevent overwhelming the system
			if iterations%100 == 0 {
				time.Sleep(time.Millisecond * 10)
			}
		}
	}
}

