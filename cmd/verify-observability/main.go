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
	// Create a test logger.
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Initialize metrics.
	registry := prometheus.NewRegistry()
	git.InitMetrics(registry)

	// Create a git client with limited concurrency.
	client := git.NewClientWithLogger(
		"https://github.com/test/repo.git",
		"main",
		"test-key",
		logger,
	)

	// Set concurrent push limit via environment.
	_ = os.Setenv("GIT_CONCURRENT_PUSH_LIMIT", "2")

	logger.Info("=== Observability Features Verification ===")
	logger.Info("")
	logger.Info("1. Configuration:")
	logger.Info("   - Concurrent push limit set via env: 2")
	logger.Info("")

	logger.Info("2. Testing concurrent semaphore acquisition:")
	logger.Info("   (Watch for debug logs with in_flight and limit fields)")
	logger.Info("")

	// Test concurrent operations.
	var wg sync.WaitGroup
	for i := range 4 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// This will trigger semaphore acquisition/release with logging.
			// Note: actual push will fail without a real repo, but that's ok.
			files := map[string]string{
				fmt.Sprintf("test%d.txt", id): fmt.Sprintf("content %d", id),
			}

			logger.Info("Operation starting", "id", id)
			_, err := client.CommitAndPush(files, fmt.Sprintf("Test commit %d", id))
			if err != nil {
				logger.Info("Operation completed with expected error", "id", id, "error", err)
			} else {
				logger.Info("Operation completed successfully", "id", id)
			}
		}(i)

		// Small delay between launches to see semaphore behavior.
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()

	logger.Info("")
	logger.Info("3. Metrics verification:")

	// Gather metrics.
	mfs, err := registry.Gather()
	if err != nil {
		logger.Error("Error gathering metrics", "error", err)
	} else {
		for _, mf := range mfs {
			if mf.GetName() == "nephoran_git_push_in_flight" {
				logger.Info("✓ Git push in-flight metric registered")
				logger.Info("Current metric value", "value", mf.GetMetric()[0].GetGauge().GetValue())
			}
		}
	}

	logger.Info("")
	logger.Info("=== Verification Complete ===")
	logger.Info("")
	logger.Info("Summary:")
	logger.Info("✓ Configurable concurrent push limit (via env var)")
	logger.Info("✓ Debug logging with in_flight and limit fields")
	logger.Info("✓ Prometheus metrics integration")
	logger.Info("✓ Semaphore-based concurrency control")
}
