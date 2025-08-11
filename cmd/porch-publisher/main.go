package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type Intent struct {
	IntentType string `json:"intent_type"`
	Target     string `json:"target"`
	Namespace  string `json:"namespace"`
	Replicas   int    `json:"replicas"`
}

func main() {
	var intentPath string
	var outDir string
	flag.StringVar(&intentPath, "intent", "", "path to intent json (from ingest handoff)")
	flag.StringVar(&outDir, "out", "examples/packages/scaling", "output package directory")
	flag.Parse()
	if intentPath == "" {
		fmt.Println("usage: porch-publisher -intent <path-to-intent.json> [-out examples/packages/scaling]")
		os.Exit(2)
	}

	b, err := os.ReadFile(intentPath)
	if err != nil {
		panic(err)
	}
	var in Intent
	if err := json.Unmarshal(b, &in); err != nil {
		panic(err)
	}
	if in.IntentType != "scaling" {
		panic("only scaling intent is supported in MVP")
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}
	patch := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: %d
`, in.Target, in.Namespace, in.Replicas)

	dst := filepath.Join(outDir, "scaling-patch.yaml")
	if err := os.WriteFile(dst, []byte(patch), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote:", dst)
	fmt.Println("next: (optional) kpt live init/apply under", outDir)
}
