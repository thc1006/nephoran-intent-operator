
package main



import (

	"flag"

	"log"

	"net/url"

	"os"

	"os/signal"

	"path/filepath"

	"syscall"

	"time"



	"github.com/nephio-project/nephoran-intent-operator/internal/watch"

)



func main() {

	// Parse command-line flags.

	handoffDir := flag.String("handoff", "./handoff", "Directory to watch for intent files")

	postURL := flag.String("post-url", "", "Optional HTTP endpoint to POST valid intents")

	schemaPath := flag.String("schema", "./docs/contracts/intent.schema.json", "Path to intent schema file")

	debounceMs := flag.Int("debounce-ms", 300, "Debounce delay in milliseconds to handle partial writes")



	// Security flags.

	bearerToken := flag.String("bearer-token", "", "Bearer token for authentication")

	apiKey := flag.String("api-key", "", "API key for authentication")

	apiKeyHeader := flag.String("api-key-header", "X-API-Key", "Custom API key header")

	insecureSkipVerify := flag.Bool("insecure-skip-verify", false, "Skip TLS certificate verification (for development only)")



	flag.Parse()



	// Validate POST URL if provided.

	if *postURL != "" {

		if _, err := url.Parse(*postURL); err != nil {

			log.Fatalf("Invalid POST URL '%s': %v", *postURL, err)

		}

		// Ensure it's HTTP or HTTPS.

		if parsedURL, err := url.Parse(*postURL); err != nil {

			log.Fatalf("Failed to re-parse POST URL '%s': %v", *postURL, err)

		} else if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {

			log.Fatalf("POST URL must use HTTP or HTTPS scheme, got: %s", *postURL)

		}

	}



	// Set up structured logging.

	log.SetPrefix("[conductor-watch] ")

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)



	// Convert to absolute paths for Windows compatibility.

	absHandoffDir, err := filepath.Abs(*handoffDir)

	if err != nil {

		log.Fatalf("Failed to get absolute path for handoff dir: %v", err)

	}



	absSchemaPath, err := filepath.Abs(*schemaPath)

	if err != nil {

		log.Fatalf("Failed to get absolute path for schema: %v", err)

	}



	// Ensure handoff directory exists.

	if err := os.MkdirAll(absHandoffDir, 0o750); err != nil {

		log.Fatalf("Failed to create handoff directory %s: %v", absHandoffDir, err)

	}



	// Verify schema file exists.

	if _, err := os.Stat(absSchemaPath); err != nil {

		log.Fatalf("Schema file not found at %s: %v", absSchemaPath, err)

	}



	log.Printf("Starting conductor-watch:")

	log.Printf("  Watching: %s", absHandoffDir)

	log.Printf("  Schema: %s", absSchemaPath)

	log.Printf("  Debounce: %dms", *debounceMs)

	if *postURL != "" {

		log.Printf("  POST URL: %s", *postURL)

		if *bearerToken != "" {

			log.Printf("  Auth: Bearer token configured")

		}

		if *apiKey != "" {

			log.Printf("  Auth: API key configured (header: %s)", *apiKeyHeader)

		}

		if *insecureSkipVerify {

			log.Printf("  TLS: Certificate verification DISABLED (development only)")

		}

	}



	// Create watcher configuration.

	config := &watch.Config{

		HandoffDir:         absHandoffDir,

		SchemaPath:         absSchemaPath,

		PostURL:            *postURL,

		DebounceDelay:      time.Duration(*debounceMs) * time.Millisecond,

		BearerToken:        *bearerToken,

		APIKey:             *apiKey,

		APIKeyHeader:       *apiKeyHeader,

		InsecureSkipVerify: *insecureSkipVerify,

	}



	// Create and start watcher.

	watcher, err := watch.NewWatcher(config)

	if err != nil {

		log.Fatalf("Failed to create watcher: %v", err)

	}



	// Setup signal handling for graceful shutdown.

	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)



	// Start watching in a goroutine.

	done := make(chan error, 1)

	go func() {

		done <- watcher.Start()

	}()



	// Wait for either error or interrupt signal.

	select {

	case err := <-done:

		if err != nil {

			log.Fatalf("Watcher error: %v", err)

		}

	case sig := <-sigChan:

		log.Printf("Received signal %v, shutting down gracefully", sig)

		if err := watcher.Stop(); err != nil {

			log.Printf("Error stopping watcher: %v", err)

		}

	}



	log.Println("Conductor-watch stopped")

}

