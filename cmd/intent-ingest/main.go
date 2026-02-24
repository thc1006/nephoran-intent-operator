// Package main provides an HTTP server for ingesting network intents and converting them to structured NetworkIntent CRDs.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	ingest "github.com/thc1006/nephoran-intent-operator/internal/ingest"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/porch"
)

func main() {
	// Initialize structured logging
	logLevel := logging.GetLogLevel()
	logging.InitGlobalLogger(logLevel)
	logger := logging.NewLogger(logging.ComponentIngest)

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
			logger.ErrorEvent(err, "Failed to get working directory")
			os.Exit(1)
		}

		schemaPath = filepath.Join(repoRoot, "docs", "contracts", "intent.schema.json")

	}

	// Initialize validator.

	v, err := ingest.NewValidator(schemaPath)
	if err != nil {
		logger.ErrorEvent(err, "Failed to load schema", "schemaPath", schemaPath)
		os.Exit(1)
	}

	// Create provider based on mode.

	intentProvider, err := ingest.NewProvider(*mode, *provider)
	if err != nil {
		logger.ErrorEvent(err, "Failed to create provider", "mode", *mode, "provider", *provider)
		os.Exit(1)
	}

	// Create Porch client if enabled.

	var porchClient ingest.PorchClient
	if *porchEnabled {
		porchClient, err = porch.NewClient(*porchURL, *porchDryRun)
		if err != nil {
			logger.ErrorEvent(err, "Failed to create Porch client", "porchURL", *porchURL, "dryRun", *porchDryRun)
			os.Exit(1)
		}
		logger.InfoEvent("Porch integration enabled", "porchURL", *porchURL, "dryRun", *porchDryRun)
	} else {
		logger.InfoEvent("Porch integration disabled (filesystem-only mode)")
	}

	// Create handler with provider and optional Porch client.

	h := ingest.NewHandler(v, *handoffDir, intentProvider, porchClient)

	// Setup HTTP routes with middleware for request logging.

	mux := http.NewServeMux()

	// Health check endpoint with request logging
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := uuid.New().String()
		reqLogger := logger.WithRequestID(requestID)

		reqLogger.DebugEvent("Health check request received", "method", r.Method, "path", r.URL.Path)

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Request-ID", requestID)

		if _, err := w.Write([]byte("ok\n")); err != nil {
			// Log error but continue since response may have already been sent.
			reqLogger.ErrorEvent(err, "Failed to write health check response")
			duration := time.Since(start).Seconds()
			reqLogger.HTTPError(r.Method, r.URL.Path, http.StatusInternalServerError, err, duration)
		} else {
			duration := time.Since(start).Seconds()
			reqLogger.HTTPRequest(r.Method, r.URL.Path, http.StatusOK, duration)
		}
	})

	// Intent endpoint with request logging wrapper
	mux.HandleFunc("/intent", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := uuid.New().String()
		reqLogger := logger.WithRequestID(requestID)

		reqLogger.InfoEvent("Intent request received", "method", r.Method, "path", r.URL.Path, "contentType", r.Header.Get("Content-Type"))

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Set request ID in header for client tracking
		rw.Header().Set("X-Request-ID", requestID)

		// Call the actual handler
		h.HandleIntent(rw, r)

		// Log the completed request
		duration := time.Since(start).Seconds()
		if rw.statusCode >= 400 {
			reqLogger.HTTPError(r.Method, r.URL.Path, rw.statusCode, fmt.Errorf("request failed with status %d", rw.statusCode), duration)
		} else {
			reqLogger.HTTPRequest(r.Method, r.URL.Path, rw.statusCode, duration)
		}
	})

	// Log server startup configuration
	logger.InfoEvent("Intent-ingest server starting",
		"address", *addr,
		"mode", *mode,
		"provider", *provider,
		"handoffDir", *handoffDir,
		"schemaPath", schemaPath,
		"logLevel", logLevel,
	)

	fmt.Printf("\nReady to accept intents at http://localhost%s/intent\n", *addr)

	// Use http.Server with timeouts to fix G114 security warning.

	server := &http.Server{
		Addr: *addr,

		Handler: mux,

		ReadTimeout: 15 * time.Second,

		WriteTimeout: 15 * time.Second,

		IdleTimeout: 60 * time.Second,
	}

	logger.InfoEvent("Starting HTTP server", "addr", *addr)
	if err := server.ListenAndServe(); err != nil {
		logger.ErrorEvent(err, "Server failed", "addr", *addr)
		os.Exit(1)
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
