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

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	generator := kpm.NewGenerator(*nodeID, *outputDir)

	ticker := time.NewTicker(*period)
	defer ticker.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	log.Printf("E2 KPM Simulator started: node=%s, output=%s, period=%v", *nodeID, *outputDir, *period)

	for {
		select {
		case <-ticker.C:
			if err := generator.GenerateMetric(); err != nil {
				log.Printf("Failed to generate metric: %v", err)
			} else {
				log.Printf("Metric generated for node %s", *nodeID)
			}
		case <-sigCh:
			log.Println("Shutting down E2 KPM Simulator")
			return
		}
	}
}
