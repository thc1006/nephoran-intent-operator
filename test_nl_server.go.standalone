package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	
	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

// Simple test server for NL to Intent endpoint
func main() {
	http.HandleFunc("/nl/intent", nlToIntentHandler)
	
	port := ":8090"
	fmt.Printf("Starting NL to Intent server on %s\n", port)
	fmt.Println("Test with: curl -X POST http://localhost:8090/nl/intent -H \"Content-Type: text/plain\" -d \"scale nf-sim to 4 in ns ran-a\"")
	
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func nlToIntentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	
	text := string(body)
	if text == "" {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}
	
	// Parse intent
	parser := ingest.NewRuleBasedIntentParser()
	intent, err := parser.ParseIntent(text)
	if err != nil {
		errorResp := map[string]interface{}{
			"error": fmt.Sprintf("Failed to parse intent: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResp)
		return
	}
	
	// Validate intent with schema
	if err := ingest.ValidateIntentWithSchema(intent, ""); err != nil {
		errorResp := map[string]interface{}{
			"error": fmt.Sprintf("Invalid intent: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResp)
		return
	}
	
	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(intent)
}