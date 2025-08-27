// Package main implements the kpmgen tool for generating KPM measurement windows.
//
// This tool generates KPM (Key Performance Measurement) window JSON files according to the
// E2SM-KPM profile defined in docs/contracts/e2.kpm.profile.md. It supports different
// load profiles and outputs timestamped JSON files for testing and simulation purposes.
package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// KPMWindow represents a KPM measurement window with the three key metrics
// defined in the E2SM-KPM profile.
type KMPWindow struct {
	// P95LatencyMs represents the 95th percentile latency in milliseconds
	P95LatencyMs float64 `json:"kmp.p95_latency_ms"`

	// PRBUtilization represents Physical Resource Block utilization as a ratio [0..1]
	PRBUtilization float64 `json:"kmp.prb_utilization"`

	// UECount represents the number of connected User Equipment
	UECount int `json:"kmp.ue_count"`

	// Timestamp represents when this measurement was taken
	Timestamp time.Time `json:"timestamp"`

	// WindowID is a unique identifier for this measurement window
	WindowID string `json:"window_id"`
}

// ProfileConfig defines the ranges for different load profiles
type ProfileConfig struct {
	LatencyMin, LatencyMax         float64
	UtilizationMin, UtilizationMax float64
	UECountMin, UECountMax         int
}

var (
	// Profile configurations matching the specification
	profiles = map[string]ProfileConfig{
		"high": {
			LatencyMin: 200.0, LatencyMax: 500.0,
			UtilizationMin: 0.80, UtilizationMax: 0.95,
			UECountMin: 800, UECountMax: 1000,
		},
		"low": {
			LatencyMin: 10.0, LatencyMax: 50.0,
			UtilizationMin: 0.10, UtilizationMax: 0.30,
			UECountMin: 50, UECountMax: 200,
		},
		"random": {
			LatencyMin: 10.0, LatencyMax: 500.0,
			UtilizationMin: 0.10, UtilizationMax: 0.95,
			UECountMin: 50, UECountMax: 1000,
		},
	}
)

// Config holds the command-line configuration
type Config struct {
	OutDir  string
	Period  time.Duration
	Burst   int
	Profile string
}

// parseFlags parses command-line flags and returns the configuration
func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.OutDir, "out", "./kmp-windows", "Output directory for JSON files")
	flag.DurationVar(&config.Period, "period", time.Second, "Time between emissions")
	flag.IntVar(&config.Burst, "burst", 60, "Number of windows to generate")
	flag.StringVar(&config.Profile, "profile", "random", "Load profile: high|low|random")

	flag.Parse()

	// Input validation and sanitization
	if err := validateConfig(config); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	return config
}

// validateConfig performs comprehensive input validation
func validateConfig(config *Config) error {
	// Validate profile
	if _, exists := profiles[config.Profile]; !exists {
		return fmt.Errorf("invalid profile '%s'. Valid options: high, low, random", config.Profile)
	}

	// Validate and sanitize output directory path
	config.OutDir = filepath.Clean(config.OutDir)
	if strings.Contains(config.OutDir, "..") {
		return fmt.Errorf("output directory cannot contain relative path traversal")
	}

	// Validate burst count (bounds checking)
	if config.Burst < 1 {
		return fmt.Errorf("burst count must be at least 1")
	}
	if config.Burst > 10000 {
		return fmt.Errorf("burst count cannot exceed 10000 for safety")
	}

	// Validate period (bounds checking)
	if config.Period < 100*time.Millisecond {
		return fmt.Errorf("period must be at least 100ms")
	}
	if config.Period > 24*time.Hour {
		return fmt.Errorf("period cannot exceed 24 hours")
	}

	return nil
}

// generateWindow creates a KPM window with cryptographically secure random values
func generateWindow(profile ProfileConfig, windowID string) (*KMPWindow, error) {
	// Generate cryptographically secure random values
	latencyRand, err := cryptoRandFloat64()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random latency: %w", err)
	}
	latency := profile.LatencyMin + latencyRand*(profile.LatencyMax-profile.LatencyMin)

	utilizationRand, err := cryptoRandFloat64()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random utilization: %w", err)
	}
	utilization := profile.UtilizationMin + utilizationRand*(profile.UtilizationMax-profile.UtilizationMin)

	ueCountRange := profile.UECountMax - profile.UECountMin + 1
	ueCountRand, err := cryptoRandInt(ueCountRange)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random UE count: %w", err)
	}
	ueCount := profile.UECountMin + ueCountRand

	return &KMPWindow{
		P95LatencyMs:   latency,
		PRBUtilization: utilization,
		UECount:        ueCount,
		Timestamp:      time.Now().UTC(),
		WindowID:       sanitizeWindowID(windowID),
	}, nil
}

// cryptoRandFloat64 generates a cryptographically secure random float64 in [0, 1)
func cryptoRandFloat64() (float64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<53))
	if err != nil {
		return 0, err
	}
	return float64(n.Int64()) / (1 << 53), nil
}

// cryptoRandInt generates a cryptographically secure random int in [0, max)
func cryptoRandInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

// sanitizeWindowID ensures the window ID is safe for use in filenames
func sanitizeWindowID(id string) string {
	// Remove any potentially dangerous characters
	id = strings.ReplaceAll(id, "..", "")
	id = strings.ReplaceAll(id, "/", "")
	id = strings.ReplaceAll(id, "\\", "")
	id = strings.ReplaceAll(id, ":", "")
	id = strings.ReplaceAll(id, "*", "")
	id = strings.ReplaceAll(id, "?", "")
	id = strings.ReplaceAll(id, "\"", "")
	id = strings.ReplaceAll(id, "<", "")
	id = strings.ReplaceAll(id, ">", "")
	id = strings.ReplaceAll(id, "|", "")

	// Limit length
	if len(id) > 100 {
		id = id[:100]
	}

	return id
}

// writeWindow writes a KPM window to a JSON file with the specified format
func writeWindow(window *KMPWindow, outDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename: YYYYMMDDTHHMMSSZ_kmp-window.json
	filename := fmt.Sprintf("%s_kmp-window.json", window.Timestamp.Format("20060102T150405Z"))
	filepath := filepath.Join(outDir, filename)

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(window, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filepath, err)
	}

	log.Printf("Generated KPM window: %s", filename)
	return nil
}

// runGenerator runs the main generation loop
func runGenerator(ctx context.Context, config *Config) error {
	profile := profiles[config.Profile]
	log.Printf("Starting KPM generator with profile '%s' (burst=%d, period=%v)",
		config.Profile, config.Burst, config.Period)

	ticker := time.NewTicker(config.Period)
	defer ticker.Stop()

	windowCount := 0

	for {
		select {
		case <-ctx.Done():
			log.Printf("Generator stopped after %d windows", windowCount)
			return ctx.Err()

		case <-ticker.C:
			if windowCount >= config.Burst {
				log.Printf("Generated %d windows, stopping", windowCount)
				return nil
			}

			windowID := fmt.Sprintf("window-%d", windowCount+1)
			window, err := generateWindow(profile, windowID)
			if err != nil {
				log.Printf("Error generating window: %v", err)
				continue
			}

			if err := writeWindow(window, config.OutDir); err != nil {
				log.Printf("Error writing window: %v", err)
				continue
			}

			windowCount++
		}
	}
}

func main() {
	// Note: Using crypto/rand for secure random number generation
	// No seed initialization required for crypto/rand

	// Parse command-line flags with validation
	config := parseFlags()

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down gracefully...", sig)
		cancel()
	}()

	// Run the generator
	if err := runGenerator(ctx, config); err != nil && err != context.Canceled {
		log.Fatalf("Generator error: %v", err)
	}

	log.Println("KPM generator finished")
}
