package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	ingest "github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

func main() {
	// Command-line flags
	var (
		addr       = flag.String("addr", ":8080", "HTTP server address")
		handoffDir = flag.String("handoff", filepath.Join(".", "handoff"), "Directory for handoff files")
		schemaFile = flag.String("schema", "", "Path to intent schema file (default: docs/contracts/intent.schema.json)")
		mode       = flag.String("mode", "", "Intent parsing mode: rules|llm (overrides MODE env var)")
		provider   = flag.String("provider", "", "LLM provider: mock (overrides PROVIDER env var)")
	)
	flag.Parse()

	// Check environment variables (command-line flags take precedence)
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

	// Determine schema path
	var schemaPath string
	if *schemaFile != "" {
		schemaPath = *schemaFile
	} else {
		repoRoot, _ := os.Getwd()
		schemaPath = filepath.Join(repoRoot, "docs", "contracts", "intent.schema.json")
	}

	// Initialize validator
	v, err := ingest.NewValidator(schemaPath)
	if err != nil {
		log.Fatalf("Failed to load schema: %v", err)
	}

	// Create provider based on mode
	intentProvider, err := ingest.NewProvider(*mode, *provider)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// Create handler with provider
	h := ingest.NewHandler(v, *handoffDir, intentProvider)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { 
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/intent", h.HandleIntent)

	// Start server
	log.Printf("intent-ingest starting...")
	log.Printf("  Address: %s", *addr)
	log.Printf("  Mode: %s", *mode)
	if *mode == "llm" {
		log.Printf("  Provider: %s", *provider)
	}
	log.Printf("  Handoff directory: %s", *handoffDir)
	log.Printf("  Schema: %s", schemaPath)
	
	fmt.Printf("\nReady to accept intents at http://localhost%s/intent\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
