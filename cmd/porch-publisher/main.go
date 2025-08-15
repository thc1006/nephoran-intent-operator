package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

// Intent represents a network intent - kept for public API compatibility
type Intent struct {
	IntentType string `json:"intent_type"`
	Target     string `json:"target"`
	Namespace  string `json:"namespace"`
	Replicas   int    `json:"replicas"`
}

func main() {
	var intentPath string
	var outDir string
	var format string
	flag.StringVar(&intentPath, "intent", "", "path to intent json (from ingest handoff)")
	flag.StringVar(&outDir, "out", "examples/packages/scaling", "output package directory")
	flag.StringVar(&format, "format", "full", "output format: 'full' (default) or 'smp' (Strategic Merge Patch)")
	flag.Parse()
	
	if intentPath == "" {
		fmt.Println("usage: porch-publisher -intent <path-to-intent.json> [-out examples/packages/scaling] [-format full|smp]")
		os.Exit(2)
	}

	b, err := os.ReadFile(intentPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	
	var in Intent
	if err := json.Unmarshal(b, &in); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}
	
	// Use the internal package to write the intent
	if err := porch.WriteIntent(in, outDir, format); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}