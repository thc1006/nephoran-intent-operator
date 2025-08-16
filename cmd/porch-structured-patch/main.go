package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/thc1006/nephoran-intent-operator/internal/patch"
)

func main() {
	var intentPath string
	var outputDir string
	var apply bool

	flag.StringVar(&intentPath, "intent", "", "Path to the intent JSON file")
	flag.StringVar(&outputDir, "out", "", "Output directory for generated patches")
	flag.BoolVar(&apply, "apply", false, "Apply the patch using porch-direct after generation")
	flag.Parse()

	if intentPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --intent flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if outputDir == "" {
		outputDir = filepath.Join(".", "examples", "packages", "scaling")
	}

	if err := run(intentPath, outputDir, apply); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(intentPath, outputDir string, apply bool) error {
	// Load intent
	intent, err := patch.LoadIntent(intentPath)
	if err != nil {
		return fmt.Errorf("failed to load intent: %w", err)
	}

	// Generate patch
	generator := patch.NewGenerator(intent, outputDir)
	if err := generator.Generate(); err != nil {
		return fmt.Errorf("failed to generate patch: %w", err)
	}

	// Optionally apply using porch-direct
	if apply {
		fmt.Println("Calling porch-direct to apply patch...")
		cmd := exec.Command("porch-direct", "--package", outputDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: porch-direct failed: %v\n", err)
			fmt.Printf("You can manually apply with: porch-direct --package %s\n", outputDir)
		} else {
			fmt.Println("Patch applied successfully via porch-direct")
		}
	}

	return nil
}