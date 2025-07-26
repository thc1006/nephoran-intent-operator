package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	ragAPIURL string
)

// Defines the structure of the incoming request from the controller
type networkIntentRequest struct {
	Spec struct {
		Intent string `json:"intent"`
	} `json:"spec"`
}

// Defines the structure of the request sent to the RAG API
type ragAPIRequest struct {
	Intent string `json:"intent"`
}

func main() {
	ragAPIURL = os.Getenv("RAG_API_URL")
	if ragAPIURL == "" {
		// Default to the Kubernetes service name for the RAG API
		ragAPIURL = "http://rag-api.default.svc.cluster.local:5001/process_intent"
	}

	http.HandleFunc("/process", processHandler)
	log.Println("Starting LLM Processor server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func processHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Decode the incoming NetworkIntent resource from the controller
	var intentReq networkIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&intentReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	log.Printf("Received intent: %s", intentReq.Spec.Intent)

	// 2. Forward the extracted intent string to the RAG API
	ragRequestBody := ragAPIRequest{Intent: intentReq.Spec.Intent}
	ragRespBody, err := callRAGAPI(r.Context(), ragRequestBody)
	if err != nil {
		log.Printf("Error calling RAG API: %v", err)
		http.Error(w, "Failed to process intent via RAG API", http.StatusInternalServerError)
		return
	}
	log.Printf("Received structured response from RAG API")

	// 3. The RAG API response is the structured JSON.
	// Forward it directly back to the controller.
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(ragRespBody); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func callRAGAPI(ctx context.Context, reqBody ragAPIRequest) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	requestBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body for RAG API: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", ragAPIURL, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request for RAG API: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to RAG API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("RAG API returned non-200 status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}