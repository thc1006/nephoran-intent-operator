package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	ingest "github.com/thc1006/nephoran-intent-operator/internal/ingest"
	"github.com/thc1006/nephoran-intent-operator/pkg/middleware"
)

func main() {
	// repoRoot 假設是目前工作樹根目錄
	repoRoot, _ := os.Getwd()
	schemaPath := filepath.Join(repoRoot, "docs", "contracts", "intent.schema.json")
	outDir := filepath.Join(repoRoot, "handoff") // 與 porch 分支用同一個協作目錄名，後面會給你路徑參數

	v, err := ingest.NewValidator(schemaPath)
	if err != nil {
		log.Fatalf("load schema failed: %v", err)
	}
	h := ingest.NewHandler(v, outDir)

	// SECURITY FIX: Set up comprehensive security middleware
	securityConfig := middleware.DefaultSecuritySuiteConfig()
	securityConfig.RateLimit.QPS = 10        // 10 requests per second per IP
	securityConfig.RateLimit.Burst = 20      // Allow bursts up to 20 requests
	securityConfig.RequireAuth = false       // No authentication required for this service
	
	securitySuite, err := middleware.NewSecuritySuite(securityConfig, nil)
	if err != nil {
		log.Fatalf("failed to create security suite: %v", err)
	}

	mux := http.NewServeMux()
	
	// Health endpoint with basic security
	healthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { 
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) 
	})
	mux.Handle("/healthz", securitySuite.Middleware(healthHandler))
	
	// Intent endpoint with full security middleware
	intentHandler := http.HandlerFunc(h.HandleIntent)
	mux.Handle("/intent", securitySuite.Middleware(intentHandler))

	addr := ":8080"
	log.Printf("intent-ingest listening on %s with rate limiting and security headers enabled", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
