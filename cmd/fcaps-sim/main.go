package main

import (
	"bytes"
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

type Config struct {
	InputFile    string
	DelaySeconds int
	Target       string
	Namespace    string
	IntentURL    string
	Verbose      bool
}

func main() {
	config := parseFlags()
	
	if config.Verbose {
		log.Printf("FCAPS Simulator starting with config: %+v", config)
	}

	// Read FCAPS events from file
	events, err := loadFCAPSEvents(config.InputFile)
	if err != nil {
		log.Fatalf("Failed to load FCAPS events: %v", err)
	}

	log.Printf("Loaded %d FCAPS events from %s", len(events), config.InputFile)

	// Create processor
	processor := fcaps.NewProcessor(config.Target, config.Namespace)
	
	// Process events with configurable delay
	for i, event := range events {
		log.Printf("\n=== Processing Event %d/%d ===", i+1, len(events))
		
		// Process the event
		decision := processor.ProcessEvent(event)
		
		if decision.ShouldScale {
			log.Printf("Scaling decision: scale %s to %d replicas (reason: %s)", 
				config.Target, decision.NewReplicas, decision.Reason)
			
			// Generate intent
			intent, err := processor.GenerateIntent(decision)
			if err != nil {
				log.Printf("Failed to generate intent: %v", err)
				continue
			}

			// Send intent to the intent-ingest service
			if err := sendIntent(config.IntentURL, intent); err != nil {
				log.Printf("Failed to send intent: %v", err)
			} else {
				log.Printf("Successfully sent scaling intent to %s", config.IntentURL)
			}
		} else {
			log.Printf("No scaling action needed")
		}

		// Wait before processing next event (except for the last one)
		if i < len(events)-1 && config.DelaySeconds > 0 {
			log.Printf("Waiting %d seconds before next event...", config.DelaySeconds)
			time.Sleep(time.Duration(config.DelaySeconds) * time.Second)
		}
	}

	log.Printf("\nFCAPS simulation completed. Final replica count: %d", processor.GetCurrentReplicas())
}

func parseFlags() Config {
	var config Config
	
	// Get default input file path relative to repo root
	repoRoot, _ := os.Getwd()
	defaultInputFile := filepath.Join(repoRoot, "docs", "contracts", "fcaps.ves.examples.json")
	
	flag.StringVar(&config.InputFile, "input", defaultInputFile, "Path to FCAPS events JSON file")
	flag.IntVar(&config.DelaySeconds, "delay", 5, "Delay in seconds between processing events")
	flag.StringVar(&config.Target, "target", "nf-sim", "Target deployment name for scaling")
	flag.StringVar(&config.Namespace, "namespace", "ran-a", "Target namespace for scaling")
	flag.StringVar(&config.IntentURL, "intent-url", "http://localhost:8080/intent", "URL for posting scaling intents")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	return config
}

func loadFCAPSEvents(inputFile string) ([]fcaps.FCAPSEvent, error) {
	// Read the JSON file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", inputFile, err)
	}

	// Parse JSON structure - the file contains examples as a map
	var examples map[string]fcaps.FCAPSEvent
	if err := json.Unmarshal(data, &examples); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Convert map to slice for processing order
	var events []fcaps.FCAPSEvent
	
	// Process in a predictable order: fault, measurement, heartbeat
	orderedKeys := []string{"fault_example", "measurement_example", "heartbeat_example"}
	
	for _, key := range orderedKeys {
		if event, exists := examples[key]; exists {
			events = append(events, event)
		}
	}

	// Add any remaining events not in the ordered list
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

func sendIntent(intentURL string, intent *ingest.Intent) error {
	// Convert intent to JSON
	intentJSON, err := json.Marshal(intent)
	if err != nil {
		return fmt.Errorf("failed to marshal intent: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", intentURL, bytes.NewBuffer(intentJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "fcaps-sim/1.0")

	// Send request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("intent request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Log successful response
	log.Printf("Intent accepted by server: %s", strings.TrimSpace(string(responseBody)))
	
	return nil
}