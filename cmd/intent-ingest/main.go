// Package main provides an HTTP server for ingesting network intents and converting them to structured NetworkIntent CRDs.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	ingest "github.com/thc1006/nephoran-intent-operator/internal/ingest"
	"github.com/thc1006/nephoran-intent-operator/pkg/porch"
)

func main() {
	// Command-line flags.

	var (
		addr = flag.String("addr", ":8080", "HTTP server address")

		handoffDir = flag.String("handoff", filepath.Join(".", "handoff"), "Directory for handoff files")

		schemaFile = flag.String("schema", "", "Path to intent schema file (default: docs/contracts/intent.schema.json)")

		mode = flag.String("mode", "", "Intent parsing mode: rules|llm (overrides MODE env var)")

		provider = flag.String("provider", "", "LLM provider: mock (overrides PROVIDER env var)")

		porchEnabled = flag.Bool("porch-enabled", false, "Enable Porch integration for package creation")

		porchURL = flag.String("porch-url", "http://porch-server:8080", "Porch server URL")

		porchDryRun = flag.Bool("porch-dry-run", false, "Porch dry-run mode (write to ./out instead of actual API calls)")
	)

	flag.Parse()

	// Check environment variables (command-line flags take precedence).

	if *mode == "" {

		*mode = os.Getenv("MODE")

		if *mode == "" {
			*mode = "rules" // default to rules mode
		}

	}

	if *provider == "" {

		*provider = os.Getenv("PROVIDER")

		if *provider == "" {
			*provider = "mock" // default to mock provider for LLM mode
		}

	}

	// Determine schema path.

	var schemaPath string

	if *schemaFile != "" {
		schemaPath = *schemaFile
	} else {

		repoRoot, err := os.Getwd()
		if err != nil {
			log.Fatalf("failed to get working directory: %v", err)
		}

		schemaPath = filepath.Join(repoRoot, "docs", "contracts", "intent.schema.json")

	}

	// Initialize validator.

	v, err := ingest.NewValidator(schemaPath)
	if err != nil {
		log.Fatalf("Failed to load schema: %v", err)
	}

	// Create provider based on mode.

	intentProvider, err := ingest.NewProvider(*mode, *provider)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// Create Porch client if enabled.

	var porchClient ingest.PorchClient
	if *porchEnabled {
		porchClient, err = porch.NewClient(*porchURL, *porchDryRun)
		if err != nil {
			log.Fatalf("Failed to create Porch client: %v", err)
		}
		log.Printf("Porch integration enabled (URL: %s, dry-run: %v)", *porchURL, *porchDryRun)
	} else {
		log.Printf("Porch integration disabled (filesystem-only mode)")
	}

	// Create handler with provider and optional Porch client.

	h := ingest.NewHandler(v, *handoffDir, intentProvider, porchClient)

	// Setup HTTP routes.

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		if _, err := w.Write([]byte("ok\n")); err != nil {
			// Log error but continue since response may have already been sent.

			log.Printf("Failed to write health check response: %v", err)
		}
	})

	mux.HandleFunc("/intent", h.HandleIntent)

	// Start server.

	log.Printf("intent-ingest starting...")

	log.Printf("  Address: %s", *addr)

	log.Printf("  Mode: %s", *mode)

	if *mode == "llm" {
		log.Printf("  Provider: %s", *provider)
	}

	log.Printf("  Handoff directory: %s", *handoffDir)

	log.Printf("  Schema: %s", schemaPath)

	if *porchEnabled {
		log.Printf("  Porch URL: %s", *porchURL)
		log.Printf("  Porch dry-run: %v", *porchDryRun)
	}

	fmt.Printf("\nReady to accept intents at http://localhost%s/intent\n", *addr)

	// Use http.Server with timeouts to fix G114 security warning.

	server := &http.Server{
		Addr: *addr,

		Handler: mux,

		ReadTimeout: 15 * time.Second,

		WriteTimeout: 15 * time.Second,

		IdleTimeout: 60 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
