package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/thc1006/nephoran-intent-operator/internal/loop"
)

func main() {
	log.SetPrefix("[conductor-loop] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Get the handoff directory path (relative to current working directory)
	handoffDir := filepath.Join(".", "handoff")
	
	// Ensure handoff directory exists
	if err := os.MkdirAll(handoffDir, 0755); err != nil {
		log.Fatalf("Failed to create handoff directory: %v", err)
	}

	// Convert to absolute path for consistency
	absHandoffDir, err := filepath.Abs(handoffDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	log.Printf("Starting conductor-loop, watching directory: %s", absHandoffDir)

	// Create and start the watcher
	watcher, err := loop.NewWatcher(absHandoffDir)
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start watching
	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	// Wait for either error or interrupt signal
	select {
	case err := <-done:
		if err != nil {
			log.Fatalf("Watcher error: %v", err)
		}
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully", sig)
		watcher.Close()
	}

	log.Println("Conductor-loop stopped")
}