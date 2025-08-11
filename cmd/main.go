package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	ingest "github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

func main() {
	repoRoot, _ := os.Getwd()
	schemaPath := filepath.Join(repoRoot, "docs", "contracts", "intent.schema.json")
	outDir := filepath.Join(repoRoot, "handoff")

	v, err := ingest.NewValidator(schemaPath)
	if err != nil {
		log.Fatalf("load schema failed: %v", err)
	}
	h := ingest.NewHandler(v, outDir)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/intent", h.HandleIntent)

	addr := ":8080"
	log.Printf("intent-ingest listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
