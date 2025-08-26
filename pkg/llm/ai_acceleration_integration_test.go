package llm

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestGPUAcceleratorIntegration tests the complete GPU accelerator integration
func TestGPUAcceleratorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GPU integration test in short mode")
	}

	config := getDefaultGPUConfig()
	config.EnableBatching = true
	config.EnableModelCache = true

	accelerator, err := NewGPUAccelerator(config)
	if err != nil {
		t.Skipf("GPU accelerator not available: %v", err)
	}
	defer accelerator.Close()

	t.Run("SingleInference", func(t *testing.T) {
		request := &InferenceRequest{
			ModelName:   "gpt-3.5-turbo",
			InputTokens: []int{1, 2, 3, 4, 5},
			MaxTokens:   10,
			Temperature: 0.7,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		response, err := accelerator.ProcessInference(ctx, request)
		if err != nil {
			t.Fatalf("Inference failed: %v", err)
		}

		if response == nil {
			t.Fatal("Response is nil")
		}

		// Basic validation
		if len(response.OutputTokens) == 0 {
			t.Error("No output tokens generated")
		}
	})

	t.Run("BatchInference", func(t *testing.T) {
		requests := make([]*InferenceRequest, 4)
		for i := range requests {
			requests[i] = &InferenceRequest{
				ModelName:   "gpt-3.5-turbo",
				InputTokens: generateRandomTokens(50),
				MaxTokens:   20,
				Temperature: 0.7,
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		responses, err := accelerator.ProcessBatch(ctx, requests)
		if err != nil {
			t.Fatalf("Batch inference failed: %v", err)
		}

		if len(responses) != len(requests) {
			t.Errorf("Expected %d responses, got %d", len(requests), len(responses))
		}

		for i, response := range responses {
			if response == nil {
				t.Errorf("Response %d is nil", i)
			}
		}
	})

	t.Run("DeviceInfo", func(t *testing.T) {
		deviceInfo := accelerator.GetDeviceInfo()
		if len(deviceInfo) == 0 {
			t.Error("No device info returned")
		}

		for _, info := range deviceInfo {
			if info.Name == "" {
				t.Error("Device name is empty")
			}
			if info.TotalMemory <= 0 {
				t.Error("Invalid total memory")
			}
		}
	})

	t.Run("PerformanceOptimization", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := accelerator.OptimizePerformance(ctx)
		if err != nil {
			t.Errorf("Performance optimization failed: %v", err)
		}
	})
}

// TestEnhancedModelCacheIntegration tests the enhanced model cache
func TestEnhancedModelCacheIntegration(t *testing.T) {
	config := getDefaultEnhancedCacheConfig()
	config.EnablePrediction = true
	config.EnablePreloading = true
	config.EnableOptimization = true

	cache, err := NewEnhancedModelCache(config)
	if err != nil {
		t.Fatalf("Failed to create enhanced model cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	modelName := "test-model"
	modelPath := "/tmp/test-model"
	deviceID := 0

	t.Run("ModelLoading", func(t *testing.T) {
		err := cache.LoadModel(ctx, modelName, modelPath, deviceID)
		if err != nil {
			t.Fatalf("Failed to load model: %v", err)
		}
	})

	t.Run("ModelRetrieval", func(t *testing.T) {
		model, found, err := cache.GetModel(ctx, modelName, deviceID)
		if err != nil {
			t.Fatalf("Failed to get model: %v", err)
		}

		if !found {
			t.Error("Model not found in cache")
		}

		if model == nil {
			t.Error("Retrieved model is nil")
		}
	})

	t.Run("PredictivePreloading", func(t *testing.T) {
		err := cache.PredictivePreload(ctx)
		if err != nil {
			t.Errorf("Predictive preloading failed: %v", err)
		}
	})

	t.Run("CacheOptimization", func(t *testing.T) {
		err := cache.OptimizeCache(ctx)
		if err != nil {
			t.Errorf("Cache optimization failed: %v", err)
		}
	})
}

// TestVectorSearchAcceleratorIntegration tests vector search acceleration
func TestVectorSearchAcceleratorIntegration(t *testing.T) {
	config := getDefaultVectorSearchConfig()
	config.VectorDimensions = 768
	config.EnableSearchCache = true

	accelerator, err := NewVectorSearchAccelerator(config)
	if err != nil {
		t.Fatalf("Failed to create vector search accelerator: %v", err)
	}
	defer accelerator.Close()

	ctx := context.Background()
	dimensions := config.VectorDimensions

	t.Run("EmbeddingGeneration", func(t *testing.T) {
		text := "This is a test document for embedding generation"
		modelName := "text-embedding-3-small"

		embedding, err := accelerator.GenerateEmbedding(ctx, text, modelName)
		if err != nil {
			t.Fatalf("Embedding generation failed: %v", err)
		}

		if len(embedding) != dimensions {
			t.Errorf("Expected embedding dimension %d, got %d", dimensions, len(embedding))
		}

		// Check if embedding is normalized (for cosine similarity)
		var norm float64
		for _, val := range embedding {
			norm += float64(val * val)
		}
		if norm < 0.9 || norm > 1.1 {
			t.Errorf("Embedding appears not normalized, norm: %f", norm)
		}
	})

	t.Run("BatchEmbeddingGeneration", func(t *testing.T) {
		texts := []string{
			"First test document",
			"Second test document",
			"Third test document",
		}
		modelName := "text-embedding-3-small"

		embeddings, err := accelerator.GenerateEmbeddingBatch(ctx, texts, modelName)
		if err != nil {
			t.Fatalf("Batch embedding generation failed: %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
		}

		for i, embedding := range embeddings {
			if len(embedding) != dimensions {
				t.Errorf("Embedding %d has dimension %d, expected %d", i, len(embedding), dimensions)
			}
		}
	})

	t.Run("VectorIndexingAndSearch", func(t *testing.T) {
		// Index some test vectors
		testVectors := []struct {
			id       string
			vector   []float32
			metadata map[string]interface{}
		}{
			{
				id:     "doc1",
				vector: generateRandomVector(dimensions),
				metadata: map[string]interface{}{
					"category": "technology",
					"score":    0.95,
				},
			},
			{
				id:     "doc2",
				vector: generateRandomVector(dimensions),
				metadata: map[string]interface{}{
					"category": "science",
					"score":    0.87,
				},
			},
			{
				id:     "doc3",
				vector: generateRandomVector(dimensions),
				metadata: map[string]interface{}{
					"category": "technology",
					"score":    0.92,
				},
			},
		}

		// Index vectors
		for _, tv := range testVectors {
			err := accelerator.IndexVector(ctx, tv.id, tv.vector, tv.metadata)
			if err != nil {
				t.Fatalf("Failed to index vector %s: %v", tv.id, err)
			}
		}

		// Perform similarity search
		queryVector := testVectors[0].vector // Use first vector as query
		topK := 2

		result, err := accelerator.SearchSimilar(ctx, queryVector, topK, nil)
		if err != nil {
			t.Fatalf("Similarity search failed: %v", err)
		}

		if len(result.Results) == 0 {
			t.Error("No search results returned")
		}

		if len(result.Results) > topK {
			t.Errorf("Expected at most %d results, got %d", topK, len(result.Results))
		}

		// Check result quality
		if len(result.Results) > 0 {
			firstResult := result.Results[0]
			if firstResult.VectorID == "" {
				t.Error("First result has empty vector ID")
			}
			if firstResult.Score <= 0 {
				t.Error("First result has invalid score")
			}
		}
	})

	t.Run("SearchByText", func(t *testing.T) {
		query := "technology innovation"
		topK := 3
		modelName := "text-embedding-3-small"

		result, err := accelerator.SearchByText(ctx, query, topK, modelName, nil)
		if err != nil {
			t.Fatalf("Search by text failed: %v", err)
		}

		if result == nil {
			t.Error("Search result is nil")
		}

		if result.SearchTimeMs <= 0 {
			t.Error("Invalid search time")
		}
	})

	t.Run("PerformanceOptimization", func(t *testing.T) {
		err := accelerator.PerformanceOptimization(ctx)
		if err != nil {
			t.Errorf("Performance optimization failed: %v", err)
		}
	})
}

// TestGPUMemoryManagerIntegration tests GPU memory management
func TestGPUMemoryManagerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GPU memory test in short mode")
	}

	config := getDefaultGPUMemoryConfig()
	config.EnableMemoryPools = true
	config.EnableAutoDefrag = true

	manager, err := NewGPUMemoryManager(config, []int{0})
	if err != nil {
		t.Skipf("GPU memory manager not available: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	t.Run("MemoryAllocation", func(t *testing.T) {
		size := int64(10 * 1024 * 1024) // 10MB
		purpose := AllocationPurposeModelWeights
		deviceID := 0

		allocation, err := manager.AllocateMemory(ctx, size, purpose, deviceID)
		if err != nil {
			t.Fatalf("Memory allocation failed: %v", err)
		}

		if allocation == nil {
			t.Fatal("Allocation is nil")
		}

		if allocation.AllocatedSize < size {
			t.Errorf("Allocated size %d is less than requested %d", allocation.AllocatedSize, size)
		}

		// Test deallocation
		err = manager.DeallocateMemory(ctx, allocation)
		if err != nil {
			t.Fatalf("Memory deallocation failed: %v", err)
		}
	})

	t.Run("MultipleAllocations", func(t *testing.T) {
		var allocations []*MemoryAllocation
		sizes := []int64{
			1 * 1024 * 1024,   // 1MB
			5 * 1024 * 1024,   // 5MB
			10 * 1024 * 1024,  // 10MB
			50 * 1024 * 1024,  // 50MB
		}

		// Allocate multiple blocks
		for i, size := range sizes {
			allocation, err := manager.AllocateMemory(ctx, size, AllocationPurposeCache, 0)
			if err != nil {
				t.Errorf("Allocation %d failed: %v", i, err)
				continue
			}
			allocations = append(allocations, allocation)
		}

		// Get memory stats
		stats := manager.GetMemoryStats()
		if stats.TotalAllocations == 0 {
			t.Error("No allocations recorded in stats")
		}

		// Deallocate all
		for i, allocation := range allocations {
			err := manager.DeallocateMemory(ctx, allocation)
			if err != nil {
				t.Errorf("Deallocation %d failed: %v", i, err)
			}
		}
	})

	t.Run("MemoryOptimization", func(t *testing.T) {
		err := manager.OptimizeMemory(ctx, 0)
		if err != nil {
			t.Errorf("Memory optimization failed: %v", err)
		}
	})

	t.Run("MemoryStats", func(t *testing.T) {
		stats := manager.GetMemoryStats()
		
		if stats == nil {
			t.Fatal("Memory stats is nil")
		}

		if len(stats.DeviceStats) == 0 {
			t.Error("No device stats available")
		}

		for deviceID, deviceStats := range stats.DeviceStats {
			if deviceStats.TotalMemory <= 0 {
				t.Errorf("Device %d has invalid total memory: %d", deviceID, deviceStats.TotalMemory)
			}
		}
	})
}

// TestConcurrentOperations tests thread safety and concurrent operations
func TestConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	// Test concurrent GPU inference
	t.Run("ConcurrentInference", func(t *testing.T) {
		config := getDefaultGPUConfig()
		accelerator, err := NewGPUAccelerator(config)
		if err != nil {
			t.Skipf("GPU accelerator not available: %v", err)
		}
		defer accelerator.Close()

		concurrency := 10
		requestsPerWorker := 5

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*requestsPerWorker)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < requestsPerWorker; j++ {
					request := &InferenceRequest{
						ModelName:   "gpt-3.5-turbo",
						InputTokens: generateRandomTokens(50),
						MaxTokens:   25,
						Temperature: 0.7,
					}

					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					_, err := accelerator.ProcessInference(ctx, request)
					cancel()

					if err != nil {
						errors <- fmt.Errorf("worker %d request %d failed: %w", workerID, j, err)
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			t.Logf("Concurrent error: %v", err)
			errorCount++
		}

		if errorCount > concurrency*requestsPerWorker/10 { // Allow up to 10% errors
			t.Errorf("Too many errors in concurrent test: %d/%d", errorCount, concurrency*requestsPerWorker)
		}
	})

	// Test concurrent vector operations
	t.Run("ConcurrentVectorOperations", func(t *testing.T) {
		config := getDefaultVectorSearchConfig()
		accelerator, err := NewVectorSearchAccelerator(config)
		if err != nil {
			t.Fatalf("Failed to create vector search accelerator: %v", err)
		}
		defer accelerator.Close()

		concurrency := 5
		operationsPerWorker := 10

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*operationsPerWorker)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				ctx := context.Background()

				for j := 0; j < operationsPerWorker; j++ {
					// Mix of indexing and searching operations
					if j%2 == 0 {
						// Index operation
						vectorID := fmt.Sprintf("worker_%d_vec_%d", workerID, j)
						vector := generateRandomVector(config.VectorDimensions)
						metadata := map[string]interface{}{
							"worker": workerID,
							"index":  j,
						}

						err := accelerator.IndexVector(ctx, vectorID, vector, metadata)
						if err != nil {
							errors <- fmt.Errorf("worker %d index %d failed: %w", workerID, j, err)
						}
					} else {
						// Search operation
						queryVector := generateRandomVector(config.VectorDimensions)
						_, err := accelerator.SearchSimilar(ctx, queryVector, 5, nil)
						if err != nil {
							errors <- fmt.Errorf("worker %d search %d failed: %w", workerID, j, err)
						}
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		errorCount := 0
		for err := range errors {
			t.Logf("Concurrent vector error: %v", err)
			errorCount++
		}

		if errorCount > concurrency*operationsPerWorker/5 { // Allow up to 20% errors
			t.Errorf("Too many errors in concurrent vector test: %d/%d", errorCount, concurrency*operationsPerWorker)
		}
	})
}

// TestErrorHandling tests error handling and recovery
func TestErrorHandling(t *testing.T) {
	t.Run("InvalidGPUConfig", func(t *testing.T) {
		config := getDefaultGPUConfig()
		config.EnabledDevices = []int{999} // Non-existent device

		_, err := NewGPUAccelerator(config)
		if err == nil {
			t.Error("Expected error for invalid device configuration")
		}
	})

	t.Run("InvalidMemoryAllocation", func(t *testing.T) {
		config := getDefaultGPUMemoryConfig()
		manager, err := NewGPUMemoryManager(config, []int{0})
		if err != nil {
			t.Skipf("GPU memory manager not available: %v", err)
		}
		defer manager.Close()

		ctx := context.Background()

		// Test zero size allocation
		_, err = manager.AllocateMemory(ctx, 0, AllocationPurposeCache, 0)
		if err == nil {
			t.Error("Expected error for zero size allocation")
		}

		// Test negative size allocation
		_, err = manager.AllocateMemory(ctx, -1, AllocationPurposeCache, 0)
		if err == nil {
			t.Error("Expected error for negative size allocation")
		}

		// Test invalid device ID
		_, err = manager.AllocateMemory(ctx, 1024, AllocationPurposeCache, 999)
		if err == nil {
			t.Error("Expected error for invalid device ID")
		}
	})

	t.Run("InvalidVectorDimensions", func(t *testing.T) {
		config := getDefaultVectorSearchConfig()
		accelerator, err := NewVectorSearchAccelerator(config)
		if err != nil {
			t.Fatalf("Failed to create vector search accelerator: %v", err)
		}
		defer accelerator.Close()

		ctx := context.Background()

		// Test mismatched vector dimensions
		wrongDimensionVector := make([]float32, config.VectorDimensions+100)
		for i := range wrongDimensionVector {
			wrongDimensionVector[i] = 0.1
		}

		err = accelerator.IndexVector(ctx, "test", wrongDimensionVector, nil)
		if err == nil {
			t.Error("Expected error for wrong vector dimensions")
		}

		_, err = accelerator.SearchSimilar(ctx, wrongDimensionVector, 5, nil)
		if err == nil {
			t.Error("Expected error for wrong query vector dimensions")
		}
	})

	t.Run("TimeoutHandling", func(t *testing.T) {
		config := getDefaultGPUConfig()
		accelerator, err := NewGPUAccelerator(config)
		if err != nil {
			t.Skipf("GPU accelerator not available: %v", err)
		}
		defer accelerator.Close()

		request := &InferenceRequest{
			ModelName:   "gpt-3.5-turbo",
			InputTokens: generateRandomTokens(100),
			MaxTokens:   50,
			Temperature: 0.7,
		}

		// Very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()

		_, err = accelerator.ProcessInference(ctx, request)
		if err == nil {
			t.Error("Expected timeout error")
		}
	})
}

// TestResourceCleanup tests proper resource cleanup
func TestResourceCleanup(t *testing.T) {
	t.Run("GPUAcceleratorCleanup", func(t *testing.T) {
		config := getDefaultGPUConfig()
		accelerator, err := NewGPUAccelerator(config)
		if err != nil {
			t.Skipf("GPU accelerator not available: %v", err)
		}

		// Close should not error
		err = accelerator.Close()
		if err != nil {
			t.Errorf("GPU accelerator close failed: %v", err)
		}

		// Double close should not error
		err = accelerator.Close()
		if err != nil {
			t.Errorf("GPU accelerator double close failed: %v", err)
		}
	})

	t.Run("MemoryManagerCleanup", func(t *testing.T) {
		config := getDefaultGPUMemoryConfig()
		manager, err := NewGPUMemoryManager(config, []int{0})
		if err != nil {
			t.Skipf("GPU memory manager not available: %v", err)
		}

		err = manager.Close()
		if err != nil {
			t.Errorf("Memory manager close failed: %v", err)
		}
	})

	t.Run("CacheCleanup", func(t *testing.T) {
		config := getDefaultEnhancedCacheConfig()
		cache, err := NewEnhancedModelCache(config)
		if err != nil {
			t.Fatalf("Failed to create enhanced model cache: %v", err)
		}

		err = cache.Close()
		if err != nil {
			t.Errorf("Cache close failed: %v", err)
		}
	})
}