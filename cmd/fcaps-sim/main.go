package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/fcaps"
	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

// Config represents a config.

type Config struct {
	InputFile string

	DelaySeconds int

	Target string

	Namespace string

	IntentURL string

	Verbose bool

	// VES collector configuration.

	CollectorURL string

	Period int

	Burst int

	NFName string

	OutHandoff string // Local handoff directory for reducer mode
}

func main() {
	config := parseFlags()

	if config.Verbose {
		log.Printf("FCAPS Simulator starting with config: %+v", config)
	}

	// Ensure handoff directory exists if specified.

	if config.OutHandoff != "" {
		if err := os.MkdirAll(config.OutHandoff, 0o750); err != nil {
			log.Fatalf("Failed to create handoff directory: %v", err)
		}
	}

	// Read FCAPS events from file.

	events, err := loadFCAPSEvents(config.InputFile)
	if err != nil {
		log.Fatalf("Failed to load FCAPS events: %v", err)
	}

	log.Printf("Loaded %d FCAPS events from %s", len(events), config.InputFile)

	// Create processor.

	processor := fcaps.NewProcessor(config.Target, config.Namespace)

	// Create reducer if in local mode.

	var reducer *fcaps.Reducer

	if config.OutHandoff != "" {

		reducer = fcaps.NewReducer(config.Burst, config.OutHandoff)

		log.Printf("Local reducer enabled: burst=%d, handoff=%s", config.Burst, config.OutHandoff)

	}

	// If collector URL is provided, send VES events there as well.

	if config.CollectorURL != "" {
		log.Printf("VES collector enabled at %s", config.CollectorURL)
	}

	// Process events with configurable delay.

	for i, event := range events {

		log.Printf("\n=== Processing Event %d/%d ===", i+1, len(events))

		// Send to VES collector if configured.

		if config.CollectorURL != "" {
			if err := sendVESEvent(config.CollectorURL, event, config.Verbose); err != nil {
				log.Printf("Failed to send VES event to collector: %v", err)
			} else if config.Verbose {
				log.Printf("Sent VES event to collector %s", config.CollectorURL)
			}
		}

		// Process with local reducer if enabled.

		if reducer != nil {
			if intent := reducer.ProcessEvent(event); intent != nil {

				filename := writeReducerIntent(config.OutHandoff, intent)

				log.Printf("*** BURST DETECTED *** Intent written: %s", filename)

				log.Printf("    Scaling %s to %d replicas (reason: %s)",

					intent.Target, intent.Replicas, intent.Reason)

			}
		}

		// Process the event.

		decision := processor.ProcessEvent(event)

		if decision.ShouldScale {

			log.Printf("Scaling decision: scale %s to %d replicas (reason: %s)",

				config.Target, decision.NewReplicas, decision.Reason)

			// Generate intent.

			intent, err := processor.GenerateIntent(decision)
			if err != nil {

				log.Printf("Failed to generate intent: %v", err)

				continue

			}

			// Send intent to the intent-ingest service if not in local-only mode.

			if config.IntentURL != "" && config.OutHandoff == "" {
				if err := sendIntent(config.IntentURL, intent); err != nil {
					log.Printf("Failed to send intent: %v", err)
				} else {
					log.Printf("Successfully sent scaling intent to %s", config.IntentURL)
				}
			}

		} else if config.Verbose {
			log.Printf("No scaling action needed")
		}

		// Wait before processing next event (except for the last one).

		if i < len(events)-1 && config.DelaySeconds > 0 {

			log.Printf("Waiting %d seconds before next event...", config.DelaySeconds)

			time.Sleep(time.Duration(config.DelaySeconds) * time.Second)

		}

	}

	log.Printf("\nFCAPS simulation completed. Final replica count: %d", processor.GetCurrentReplicas())
}

func parseFlags() Config {
	var config Config

	// Get default input file path relative to repo root.

	repoRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}

	defaultInputFile := filepath.Join(repoRoot, "docs", "contracts", "fcaps.ves.examples.json")

	flag.StringVar(&config.InputFile, "input", defaultInputFile, "Path to FCAPS events JSON file")

	flag.IntVar(&config.DelaySeconds, "delay", 5, "Delay in seconds between processing events")

	flag.StringVar(&config.Target, "target", "nf-sim", "Target deployment name for scaling")

	flag.StringVar(&config.Namespace, "namespace", "ran-a", "Target namespace for scaling")

	flag.StringVar(&config.IntentURL, "intent-url", "http://localhost:8080/intent", "URL for posting scaling intents")

	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")

	// VES collector flags.

	flag.StringVar(&config.CollectorURL, "collector-url", "http://localhost:9999/eventListener/v7", "VES collector URL")

	flag.IntVar(&config.Period, "period", 10, "Period in seconds between event batches")

	flag.IntVar(&config.Burst, "burst", 1, "Number of events per burst")

	flag.StringVar(&config.NFName, "nf-name", "nf-sim", "Network Function name")

	flag.StringVar(&config.OutHandoff, "out-handoff", "", "Local handoff directory for reducer mode")

	flag.Parse()

	return config
}

func loadFCAPSEvents(inputFile string) ([]fcaps.FCAPSEvent, error) {
	// Read the JSON file.

	data, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", inputFile, err)
	}

	// Parse JSON structure - the file contains examples as a map.

	var examples map[string]fcaps.FCAPSEvent

	if err := json.Unmarshal(data, &examples); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Convert map to slice for processing order.

	orderedKeys := []string{"fault_example", "measurement_example", "heartbeat_example"}
	events := make([]fcaps.FCAPSEvent, 0, len(orderedKeys))

	// Process in a predictable order: fault, measurement, heartbeat.

	for _, key := range orderedKeys {
		if event, exists := examples[key]; exists {
			events = append(events, event)
		}
	}

	// Add any remaining events not in the ordered list.

	for key, event := range examples {

		found := false

		for _, orderedKey := range orderedKeys {
			if key == orderedKey {

				found = true

				break

			}
		}

		if !found {
			events = append(events, event)
		}

	}

	if len(events) == 0 {
		return nil, fmt.Errorf("no valid FCAPS events found in file")
	}

	return events, nil
}

func sendVESEvent(collectorURL string, event fcaps.FCAPSEvent, verbose bool) error {
	// Convert event to JSON.

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal VES event: %w", err)
	}

	if verbose {
		log.Printf("Sending VES event: %s", string(eventJSON))
	}

	// Create HTTP request with context.

	req, err := http.NewRequestWithContext(context.Background(), "POST", collectorURL, bytes.NewBuffer(eventJSON))
	if err != nil {
		return fmt.Errorf("failed to create VES request: %w", err)
	}

	// Set headers according to VES specification.

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("User-Agent", "fcaps-sim/1.0")

	req.Header.Set("X-MinorVersion", "1")

	req.Header.Set("X-PatchVersion", "0")

	req.Header.Set("X-LatestVersion", "7.3")

	// Send request.

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send VES request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }() // #nosec G307 - Error handled in defer

	// Read response.

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read VES response: %w", err)
	}

	// Check response status (VES typically returns 202 Accepted).

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("VES collector request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	if verbose {
		log.Printf("VES collector response: %s", strings.TrimSpace(string(responseBody)))
	}

	return nil
}

func writeReducerIntent(outDir string, intent *fcaps.ScalingIntent) string {
	timestamp := time.Now().UTC().Format("20060102T150405Z")

	filename := filepath.Join(outDir, fmt.Sprintf("intent-%s.json", timestamp))

	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {

		log.Printf("Failed to marshal reducer intent: %v", err)

		return ""

	}

	if err := os.WriteFile(filename, data, 0o640); err != nil {

		log.Printf("Failed to write reducer intent: %v", err)

		return ""

	}

	return filename
}

func sendIntent(intentURL string, intent *ingest.Intent) error {
	// Convert intent to JSON.

	intentJSON, err := json.Marshal(intent)
	if err != nil {
		return fmt.Errorf("failed to marshal intent: %w", err)
	}

	// Create HTTP request with context.

	req, err := http.NewRequestWithContext(context.Background(), "POST", intentURL, bytes.NewBuffer(intentJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers.

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("User-Agent", "fcaps-sim/1.0")

	// Send request.

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }() // #nosec G307 - Error handled in defer

	// Read response.

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status.

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("intent request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Log successful response.

	log.Printf("Intent accepted by server: %s", strings.TrimSpace(string(responseBody)))

	return nil
}
