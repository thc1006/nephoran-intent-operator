package main

import (
	"encoding/json"
	"fmt"
	"log"
	
	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

func main() {
	// Create parser
	parser := ingest.NewRuleBasedIntentParser()
	
	// Test cases
	testCases := []string{
		"scale nf-sim to 4 in ns ran-a",
		"scale my-app to 3",
		"deploy nginx in ns production",
		"delete old-app from ns staging",
		"update myapp set replicas=5 in ns prod",
		"invalid command",
	}
	
	fmt.Println("Testing NL to Intent Parser:")
	fmt.Println("=============================")
	
	for _, testCase := range testCases {
		fmt.Printf("\nInput: %s\n", testCase)
		
		// Parse intent
		intent, err := parser.ParseIntent(testCase)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		// Validate intent with schema
		if err := ingest.ValidateIntentWithSchema(intent, ""); err != nil {
			fmt.Printf("Validation Error: %v\n", err)
		}
		
		// Convert to JSON
		jsonData, err := json.MarshalIndent(intent, "", "  ")
		if err != nil {
			fmt.Printf("JSON Error: %v\n", err)
			continue
		}
		
		fmt.Printf("Output:\n%s\n", jsonData)
	}
	
	// Test schema validation specifically for scaling intent
	fmt.Println("\n\nTesting Schema Validation for Scaling Intent:")
	fmt.Println("==============================================")
	
	scalingIntent := map[string]interface{}{
		"intent_type": "scaling",
		"target":      "nf-sim",
		"namespace":   "ran-a",
		"replicas":    4,
	}
	
	// Try to validate with schema
	validator, err := ingest.NewIntentSchemaValidator("")
	if err != nil {
		log.Printf("Warning: Could not load schema validator: %v", err)
		log.Println("Using basic validation instead")
		if err := ingest.ValidateIntent(scalingIntent); err != nil {
			fmt.Printf("Basic Validation Error: %v\n", err)
		} else {
			fmt.Println("Basic validation passed!")
		}
	} else {
		if err := validator.ValidateIntent(scalingIntent); err != nil {
			fmt.Printf("Schema Validation Error: %v\n", err)
		} else {
			fmt.Println("Schema validation passed!")
		}
		
		// Test with invalid replicas (exceeds max)
		scalingIntent["replicas"] = 200
		fmt.Println("\nTesting with replicas=200 (exceeds max of 100):")
		if err := validator.ValidateIntent(scalingIntent); err != nil {
			fmt.Printf("Expected validation error: %v\n", err)
		}
	}
}