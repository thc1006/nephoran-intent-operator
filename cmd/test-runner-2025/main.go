// Package main provides a 2025 Go testing demonstration runner.
//
// This demonstrates how to run the modern testing patterns programmatically
// and shows integration between different test components.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/thc1006/nephoran-intent-operator/hack/testtools"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	logger := logging.NewLogger("test-runner-2025", "info")

	fmt.Println("=== 2025 Go Testing Best Practices Demo ===")
	fmt.Println()

	// Demonstrate envtest setup
	if err := demonstrateEnvtestSetup(ctx, logger); err != nil {
		log.Fatalf("Failed to demonstrate envtest setup: %v", err)
	}

	// Demonstrate O2 API server with modern constructor
	if err := demonstrateO2APIServer(ctx, logger); err != nil {
		log.Fatalf("Failed to demonstrate O2 API server: %v", err)
	}

	fmt.Println("??All 2025 Go testing patterns demonstrated successfully!")
}

func demonstrateEnvtestSetup(ctx context.Context, logger *logging.StructuredLogger) error {
	fmt.Println("?”§ Demonstrating envtest environment setup...")

	// Get recommended options based on environment
	opts := testtools.GetRecommendedOptions()
	opts.VerboseLogging = true
	opts.CIMode = testtools.IsRunningInCI()

	fmt.Printf("   - CI Mode: %v\n", opts.CIMode)
	fmt.Printf("   - Verbose Logging: %v\n", opts.VerboseLogging)
	fmt.Printf("   - Memory Limit: %s\n", opts.MemoryLimit)
	fmt.Printf("   - CPU Limit: %s\n", opts.CPULimit)

	// For demonstration purposes, we won't actually start the full environment
	// as it requires Kubernetes binaries, but we show the proper patterns
	fmt.Println("   - Environment configuration completed ??)

	return nil
}

func demonstrateO2APIServer(ctx context.Context, logger *logging.StructuredLogger) error {
	fmt.Println("?? Demonstrating O2 API Server with 2025 constructor patterns...")

	// Create O2 IMS configuration
	config := &o2.O2IMSConfig{
		ServerAddress: "127.0.0.1",
		ServerPort:    0, // Use dynamic port
		TLSEnabled:    false,
		DatabaseConfig: json.RawMessage(`{}`),
		ComplianceMode:       true,
		SpecificationVersion: "O-RAN.WG6.O2ims-Interface-v01.01",
	}

	fmt.Printf("   - Server Address: %s\n", config.ServerAddress)
	fmt.Printf("   - TLS Enabled: %v\n", config.TLSEnabled)
	fmt.Printf("   - Compliance Mode: %v\n", config.ComplianceMode)
	fmt.Printf("   - Specification Version: %s\n", config.SpecificationVersion)

	// Demonstrate new constructor signature (2025 pattern)
	o2Server, err := o2.NewO2APIServer(config, logger, nil)
	if err != nil {
		return fmt.Errorf("failed to create O2 API server: %w", err)
	}

	fmt.Println("   - O2 API Server created with modern constructor ??)

	// Demonstrate backward compatibility
	o2ServerCompat, err := o2.NewO2APIServerWithConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create O2 API server (compat): %w", err)
	}

	fmt.Println("   - Backward compatibility confirmed ??)

	// Demonstrate proper cleanup with context
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()

	if err := o2Server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown O2 server", "error", err)
	}

	if err := o2ServerCompat.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown O2 server (compat)", "error", err)
	}

	fmt.Println("   - Proper cleanup with context demonstrated ??)

	return nil
}

func init() {
	// Set environment variables for demonstration
	os.Setenv("TEST_ENV", "true")
	os.Setenv("LOG_LEVEL", "info")
}

