package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thc1006/nephoran-intent-operator/internal/generator"
	"github.com/thc1006/nephoran-intent-operator/internal/intent"
	"github.com/thc1006/nephoran-intent-operator/internal/writer"
	"github.com/thc1006/nephoran-intent-operator/pkg/porch"
)

func main() {
	var intentPath, outDir string
	var dryRun, minimal bool
	var repo, packageName, workspace, namespace, porchURL, tokenFile string
	var autoApprove, autoPublish bool
	
	flag.StringVar(&intentPath, "intent", "", "path to intent JSON (docs/contracts/intent.schema.json)")
	flag.StringVar(&outDir, "out", "examples/packages/direct", "output directory for generated KRM")
	flag.BoolVar(&dryRun, "dry-run", false, "show what would be generated without writing files")
	flag.BoolVar(&minimal, "minimal", false, "generate minimal package (Deployment + Kptfile only)")
	
	// Porch-specific flags
	flag.StringVar(&repo, "repo", "", "Target Porch repository name")
	flag.StringVar(&packageName, "package", "", "Target package name")
	flag.StringVar(&workspace, "workspace", "default", "Workspace identifier")
	flag.StringVar(&namespace, "namespace", "default", "Target namespace")
	flag.StringVar(&porchURL, "porch-api", "http://localhost:9443", "Porch API base URL")
	flag.StringVar(&tokenFile, "token", "", "Path to service account token file")
	flag.BoolVar(&autoApprove, "auto-approve", false, "Automatically approve package proposals")
	flag.BoolVar(&autoPublish, "auto-publish", false, "Automatically publish approved packages")
	
	flag.Parse()

	if intentPath == "" {
		fmt.Fprintln(os.Stderr, "error: -intent is required")
		flag.Usage()
		os.Exit(2)
	}

	cfg := &Config{
		IntentPath:  intentPath,
		OutDir:      outDir,
		DryRun:      dryRun,
		Minimal:     minimal,
		Repo:        repo,
		Package:     packageName,
		Workspace:   workspace,
		Namespace:   namespace,
		PorchURL:    porchURL,
		TokenFile:   tokenFile,
		AutoApprove: autoApprove,
		AutoPublish: autoPublish,
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type Config struct {
	IntentPath  string
	OutDir      string
	DryRun      bool
	Minimal     bool
	Repo        string
	Package     string
	Workspace   string
	Namespace   string
	PorchURL    string
	TokenFile   string
	AutoApprove bool
	AutoPublish bool
}

func run(cfg *Config) error {
	// Determine project root by looking for go.mod
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	fmt.Printf("[porch-direct] Project root: %s\n", projectRoot)
	fmt.Printf("[porch-direct] Loading intent from: %s\n", cfg.IntentPath)
	fmt.Printf("[porch-direct] Output directory: %s\n", cfg.OutDir)
	if cfg.DryRun {
		fmt.Printf("[porch-direct] Mode: DRY RUN (no files will be written)\n")
	}
	if cfg.Minimal {
		fmt.Printf("[porch-direct] Package type: MINIMAL\n")
	}
	if cfg.PorchURL != "" && !cfg.DryRun {
		fmt.Printf("[porch-direct] Porch API: %s\n", cfg.PorchURL)
	}

	// Create loader
	loader, err := intent.NewLoader(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to create intent loader: %w", err)
	}

	// Load and validate intent
	result, err := loader.LoadFromFile(cfg.IntentPath)
	if err != nil {
		return fmt.Errorf("failed to load intent file: %w", err)
	}

	if !result.IsValid {
		fmt.Fprintf(os.Stderr, "Intent validation failed:\n")
		for _, validationErr := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", validationErr.Field, validationErr.Message)
		}
		return fmt.Errorf("intent validation failed with %d errors", len(result.Errors))
	}

	fmt.Printf("[porch-direct] Intent validated successfully: %s\n", result.Intent.String())

	// Resolve repository and package names if not provided
	if cfg.Repo == "" {
		cfg.Repo = resolveRepo(result.Intent)
	}
	if cfg.Package == "" {
		cfg.Package = resolvePackage(result.Intent)
	}

	// Create generator
	packageGen := generator.NewPackageGenerator()

	// Generate package
	var pkg *generator.Package
	if cfg.Minimal {
		pkg, err = packageGen.GenerateMinimalPackage(result.Intent, cfg.OutDir)
	} else {
		pkg, err = packageGen.GeneratePackage(result.Intent, cfg.OutDir)
	}
	if err != nil {
		return fmt.Errorf("failed to generate package: %w", err)
	}

	fmt.Printf("[porch-direct] Generated package: %s (%d files)\n", pkg.Name, pkg.GetFileCount())

	// If Porch URL is provided, use Porch API (handles both dry-run and live modes)
	if cfg.PorchURL != "" {
		return submitToPorch(cfg, result.Intent, pkg)
	}

	// Otherwise, write to filesystem
	fsWriter := writer.NewFilesystemWriter(cfg.OutDir, cfg.DryRun)

	// Write package
	writeResult, err := fsWriter.WritePackage(pkg)
	if err != nil {
		return fmt.Errorf("failed to write package: %w", err)
	}

	// Report results
	fmt.Printf("[porch-direct] Package written to: %s\n", writeResult.PackagePath)
	
	if len(writeResult.FilesWritten) > 0 {
		fmt.Printf("[porch-direct] Files written: %v\n", writeResult.FilesWritten)
	}
	if len(writeResult.FilesUpdated) > 0 {
		fmt.Printf("[porch-direct] Files updated: %v\n", writeResult.FilesUpdated)
	}
	if len(writeResult.FilesSkipped) > 0 {
		fmt.Printf("[porch-direct] Files skipped (unchanged): %v\n", writeResult.FilesSkipped)
	}

	if writeResult.WasIdempotent {
		fmt.Printf("[porch-direct] Operation was idempotent (no changes needed)\n")
	}

	fmt.Printf("[porch-direct] Success! KRM package generated for %s scaling to %d replicas\n", 
		result.Intent.Target, result.Intent.Replicas)

	return nil
}

// submitToPorch submits the generated package to Porch API with proper lifecycle management
func submitToPorch(cfg *Config, intent *intent.NetworkIntent, pkg *generator.Package) error {
	// Read authentication token if provided
	var token string
	if cfg.TokenFile != "" {
		tokenBytes, err := os.ReadFile(cfg.TokenFile)
		if err != nil {
			return fmt.Errorf("failed to read token file: %w", err)
		}
		token = string(tokenBytes)
	}

	// Create Porch client with auth
	client := porch.NewClientWithAuth(cfg.PorchURL, token, cfg.DryRun)

	// Create package request
	packageReq := &porch.PackageRequest{
		Repository: cfg.Repo,
		Package:    cfg.Package,
		Workspace:  cfg.Workspace,
		Namespace:  cfg.Namespace,
		Intent:     intent,
		Files:      pkg.GetFiles(),
	}

	// Step 1: Create or update package revision (starts in DRAFT state)
	revision, err := client.CreateOrUpdatePackage(packageReq)
	if err != nil {
		return fmt.Errorf("failed to create/update package: %w", err)
	}
	
	// Format: mvp-repo/ran-scale/nf-sim@vNNN
	packageRef := fmt.Sprintf("%s/%s@%s", cfg.Repo, cfg.Package, revision.Revision)
	if revision.Revision == "" {
		packageRef = fmt.Sprintf("%s/%s@v001", cfg.Repo, cfg.Package) // fallback
	}
	fmt.Printf("Created packagerevision %s (DRAFT", packageRef)

	// Step 2: Submit for review (DRAFT → REVIEW)
	revision, err = client.SubmitForReview(revision)
	if err != nil {
		return fmt.Errorf("failed to submit for review: %w", err)
	}
	fmt.Printf("→REVIEW")

	// Step 3: Approve package (REVIEW → APPROVED) if requested
	if cfg.AutoApprove {
		revision, err = client.ApprovePackage(revision)
		if err != nil {
			return fmt.Errorf("failed to approve package: %w", err)
		}
		fmt.Printf("→APPROVED")
		
		// Step 4: Publish package (APPROVED → PUBLISHED) if requested
		if cfg.AutoPublish {
			revision, err = client.PublishPackage(revision)
			if err != nil {
				return fmt.Errorf("failed to publish package: %w", err)
			}
			fmt.Printf("→PUBLISHED)\n")
		} else {
			fmt.Printf(")\n")
			fmt.Printf("Package ready for publishing. Use --auto-publish flag to publish automatically\n")
		}
	} else {
		fmt.Printf(")\n")
		fmt.Printf("Package is in review state. Approve manually or use --auto-approve flag\n")
	}

	return nil
}

// resolveRepo determines the repository name based on intent
func resolveRepo(intent *intent.NetworkIntent) string {
	// Default to mvp-repo as specified in requirements
	return "mvp-repo"
}

// resolvePackage generates a package name from intent
func resolvePackage(intent *intent.NetworkIntent) string {
	// Default to ran-scale/nf-sim as specified in requirements
	return "ran-scale/nf-sim"
}

// findProjectRoot walks up the directory tree to find the project root (containing go.mod)
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gomodPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(gomodPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached filesystem root
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found in any parent directory")
}
