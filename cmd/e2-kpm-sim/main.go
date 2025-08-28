package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/kpm"
)

func main() {
	var (
		outputDir = flag.String("out", "metrics", "output directory for metrics")
		period    = flag.Duration("period", 1*time.Second, "metric generation period")
		nodeID    = flag.String("node", "e2-node-001", "E2 node identifier")
	)
	flag.Parse()

	if err := os.MkdirAll(*outputDir, 0o750); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	generator, err := kpm.NewGenerator(*nodeID, *outputDir)
	if err != nil {
		log.Fatalf("Failed to create generator: %v", err)
	}

	ticker := time.NewTicker(*period)
	defer ticker.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	log.Printf("E2 KPM Simulator started: node=%s, output=%s, period=%v", *nodeID, *outputDir, *period)

	var errorCount int
	const maxRetries = 3

	for {
		select {
		case <-ticker.C:
			// Retry logic for transient failures.
			var lastErr error
			for retry := range maxRetries {
				if err := generator.GenerateMetric(); err != nil {
					lastErr = err
					if retry < maxRetries-1 {
						time.Sleep(100 * time.Millisecond * time.Duration(retry+1))
						continue
					}
				} else {
					log.Printf("Metric generated for node %s", *nodeID)
					lastErr = nil
					break
				}
			}

			if lastErr != nil {
				errorCount++
				log.Printf("Failed to generate metric after %d retries: %v (total errors: %d)",
					maxRetries, lastErr, errorCount)
			}
		case <-sigCh:
			log.Printf("Shutting down E2 KPM Simulator (total errors: %d)", errorCount)
			return
		}
	}
}
