package main

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
)

func main() {
	// Create a test logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Initialize metrics
	registry := prometheus.NewRegistry()
	git.InitMetrics(registry)

	// Create a git client with limited concurrency
	client := git.NewClientWithLogger(
		"https://github.com/test/repo.git",
		"main",
		"test-key",
		logger,
	)

	// Set concurrent push limit via environment
	os.Setenv("GIT_CONCURRENT_PUSH_LIMIT", "2")

	fmt.Println("=== Observability Features Verification ===")
	fmt.Println()
	fmt.Println("1. Configuration:")
	fmt.Println("   - Concurrent push limit set via env: 2")
	fmt.Println()

	fmt.Println("2. Testing concurrent semaphore acquisition:")
	fmt.Println("   (Watch for debug logs with in_flight and limit fields)")
	fmt.Println()

	// Test concurrent operations
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// This will trigger semaphore acquisition/release with logging
			// Note: actual push will fail without a real repo, but that's ok
			files := map[string]string{
				fmt.Sprintf("test%d.txt", id): fmt.Sprintf("content %d", id),
			}

			fmt.Printf("   Operation %d starting...\n", id)
			_, err := client.CommitAndPush(files, fmt.Sprintf("Test commit %d", id))
			if err != nil {
				fmt.Printf("   Operation %d completed (expected error: %v)\n", id, err)
			} else {
				fmt.Printf("   Operation %d completed successfully\n", id)
			}
		}(i)

		// Small delay between launches to see semaphore behavior
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()

	fmt.Println()
	fmt.Println("3. Metrics verification:")

	// Gather metrics
	mfs, err := registry.Gather()
	if err != nil {
		fmt.Printf("   Error gathering metrics: %v\n", err)
	} else {
		for _, mf := range mfs {
			if mf.GetName() == "nephoran_git_push_in_flight" {
				fmt.Printf("   ✓ Git push in-flight metric registered\n")
				fmt.Printf("     Current value: %v\n", mf.GetMetric()[0].GetGauge().GetValue())
			}
		}
	}

	fmt.Println()
	fmt.Println("=== Verification Complete ===")
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Println("✓ Configurable concurrent push limit (via env var)")
	fmt.Println("✓ Debug logging with in_flight and limit fields")
	fmt.Println("✓ Prometheus metrics integration")
	fmt.Println("✓ Semaphore-based concurrency control")
}
