// Package providers contains usage examples for the LLM providers.
// This file demonstrates how to use the offline provider for intent parsing.
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// ExampleOfflineProvider demonstrates how to use the offline provider
// for parsing natural language scaling intents.
func ExampleOfflineProvider() {
	// Create configuration for offline provider
	config := &Config{
		Type:    ProviderTypeOffline,
		Timeout: 30 * time.Second,
	}

	// Create the offline provider
	provider, err := NewOfflineProvider(config)
	if err != nil {
		log.Fatalf("Failed to create offline provider: %v", err)
	}
	defer provider.Close()

	// Example inputs and their expected parsing
	testInputs := []string{
		"scale nf-sim to 3",
		"scale nf-sim to 5 in ns ran-a",
		"scale to 7 cu-cp-service",
		"set cu-up-service replicas to 4",
		"increase amf-service to 6",
		"decrease upf-service to 2 in namespace core5g",
	}

	ctx := context.Background()

	for _, input := range testInputs {
		// Process the intent
		resp, err := provider.ProcessIntent(ctx, input)
		if err != nil {
			log.Printf("Error processing '%s': %v", input, err)
			continue
		}

		// Parse the JSON response
		var intent map[string]interface{}
		if err := json.Unmarshal(resp.JSON, &intent); err != nil {
			log.Printf("Error unmarshaling JSON for '%s': %v", input, err)
			continue
		}

		// Display results
		fmt.Printf("Input: %s\n", input)
		fmt.Printf("Target: %s, Replicas: %.0f, Namespace: %s\n",
			intent["target"], intent["replicas"], intent["namespace"])
		fmt.Printf("Provider: %s, Confidence: %.1f\n\n",
			resp.Metadata.Provider, resp.Metadata.Confidence)
	}
}

// ExampleFactoryUsage demonstrates how to use the factory to create providers
func ExampleFactoryUsage() {
	// Create factory
	factory := NewFactory()

	// Create configuration from environment (defaults to OFFLINE if not set)
	config, err := ConfigFromEnvironment()
	if err != nil {
		log.Fatalf("Failed to create config: %v", err)
	}

	// Create provider using factory
	provider, err := factory.CreateProvider(config)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Get provider information
	info := provider.GetProviderInfo()
	fmt.Printf("Using provider: %s v%s\n", info.Name, info.Version)
	fmt.Printf("Description: %s\n", info.Description)
	fmt.Printf("Requires Auth: %v\n", info.RequiresAuth)

	// Process a scaling intent
	ctx := context.Background()
	resp, err := provider.ProcessIntent(ctx, "scale nf-sim to 3 in ns ran-a")
	if err != nil {
		log.Fatalf("Failed to process intent: %v", err)
	}

	// Display the generated JSON
	var intent map[string]interface{}
	json.Unmarshal(resp.JSON, &intent)
	
	prettyJSON, _ := json.MarshalIndent(intent, "", "  ")
	fmt.Printf("Generated Intent JSON:\n%s\n", string(prettyJSON))
}