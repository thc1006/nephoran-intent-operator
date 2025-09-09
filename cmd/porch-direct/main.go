package main

import (
<<<<<<< HEAD
	"encoding/json"
	"flag"
	"fmt"
	"log"
=======
	"flag"
	"fmt"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	"os"
	"path/filepath"

	"github.com/thc1006/nephoran-intent-operator/internal/generator"
	"github.com/thc1006/nephoran-intent-operator/internal/intent"
	"github.com/thc1006/nephoran-intent-operator/internal/writer"
	"github.com/thc1006/nephoran-intent-operator/pkg/porch"
)

func main() {
	var intentPath, outDir string
<<<<<<< HEAD

	var dryRun, minimal bool

	var repo, packageName, workspace, namespace, porchURL string

	var autoApprove bool

	flag.StringVar(&intentPath, "intent", "", "path to intent JSON (docs/contracts/intent.schema.json)")

	flag.StringVar(&outDir, "out", "examples/packages/direct", "output directory for generated KRM")

	flag.BoolVar(&dryRun, "dry-run", false, "show what would be generated without writing files")

	flag.BoolVar(&minimal, "minimal", false, "generate minimal package (Deployment + Kptfile only)")

	// Porch-specific flags.

	flag.StringVar(&repo, "repo", "", "Target Porch repository name")

	flag.StringVar(&packageName, "package", "", "Target package name")

	flag.StringVar(&workspace, "workspace", "default", "Workspace identifier")

	flag.StringVar(&namespace, "namespace", "default", "Target namespace")

	flag.StringVar(&porchURL, "porch", "http://localhost:9443", "Porch API base URL")

	flag.BoolVar(&autoApprove, "auto-approve", false, "Automatically approve package proposals")

	flag.Parse()

	if intentPath == "" {

		fmt.Fprintln(os.Stderr, "error: -intent is required")

		flag.Usage()

		log.Fatal(2)

	}

	cfg := &Config{
		IntentPath: intentPath,

		OutDir: outDir,

		DryRun: dryRun,

		Minimal: minimal,

		Repo: repo,

		Package: packageName,

		Workspace: workspace,

		Namespace: namespace,

		PorchURL: porchURL,

		AutoApprove: autoApprove,
	}

	if err := run(cfg); err != nil {

		fmt.Fprintf(os.Stderr, "error: %v\n", err)

		log.Fatal(1)

	}
}

// Config represents a config.

type Config struct {
	IntentPath string

	OutDir string

	DryRun bool

	Minimal bool

	Repo string

	Package string

	Workspace string

	Namespace string

	PorchURL string

	AutoApprove bool
}

func run(cfg *Config) error {
	// Determine project root by looking for go.mod.

=======
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
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	fmt.Printf("[porch-direct] Project root: %s\n", projectRoot)
<<<<<<< HEAD

	fmt.Printf("[porch-direct] Loading intent from: %s\n", cfg.IntentPath)

	fmt.Printf("[porch-direct] Output directory: %s\n", cfg.OutDir)

	if cfg.DryRun {
		fmt.Printf("[porch-direct] Mode: DRY RUN (no files will be written)\n")
	}

	if cfg.Minimal {
		fmt.Printf("[porch-direct] Package type: MINIMAL\n")
	}

=======
	fmt.Printf("[porch-direct] Loading intent from: %s\n", cfg.IntentPath)
	fmt.Printf("[porch-direct] Output directory: %s\n", cfg.OutDir)
	if cfg.DryRun {
		fmt.Printf("[porch-direct] Mode: DRY RUN (no files will be written)\n")
	}
	if cfg.Minimal {
		fmt.Printf("[porch-direct] Package type: MINIMAL\n")
	}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if cfg.PorchURL != "" && !cfg.DryRun {
		fmt.Printf("[porch-direct] Porch API: %s\n", cfg.PorchURL)
	}

<<<<<<< HEAD
	// Create loader.

=======
	// Create loader
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	loader, err := intent.NewLoader(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to create intent loader: %w", err)
	}

<<<<<<< HEAD
	// Load and validate intent.

=======
	// Load and validate intent
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	result, err := loader.LoadFromFile(cfg.IntentPath)
	if err != nil {
		return fmt.Errorf("failed to load intent file: %w", err)
	}

	if !result.IsValid {
<<<<<<< HEAD

		fmt.Fprintf(os.Stderr, "Intent validation failed:\n")

		for _, validationErr := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", validationErr.Field, validationErr.Message)
		}

		return fmt.Errorf("intent validation failed with %d errors", len(result.Errors))

=======
		fmt.Fprintf(os.Stderr, "Intent validation failed:\n")
		for _, validationErr := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", validationErr.Field, validationErr.Message)
		}
		return fmt.Errorf("intent validation failed with %d errors", len(result.Errors))
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	fmt.Printf("[porch-direct] Intent validated successfully: %s\n", result.Intent.String())

<<<<<<< HEAD
	// Resolve repository and package names if not provided.

	if cfg.Repo == "" {
		cfg.Repo = resolveRepo(result.Intent)
	}

=======
	// Resolve repository and package names if not provided
	if cfg.Repo == "" {
		cfg.Repo = resolveRepo(result.Intent)
	}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if cfg.Package == "" {
		cfg.Package = resolvePackage(result.Intent)
	}

<<<<<<< HEAD
	// Create generator.

	packageGen := generator.NewPackageGenerator()

	// Generate package.

	var pkg *generator.Package

=======
	// Create generator
	packageGen := generator.NewPackageGenerator()

	// Generate package
	var pkg *generator.Package
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if cfg.Minimal {
		pkg, err = packageGen.GenerateMinimalPackage(result.Intent, cfg.OutDir)
	} else {
		pkg, err = packageGen.GeneratePackage(result.Intent, cfg.OutDir)
	}
<<<<<<< HEAD

=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if err != nil {
		return fmt.Errorf("failed to generate package: %w", err)
	}

	fmt.Printf("[porch-direct] Generated package: %s (%d files)\n", pkg.Name, pkg.GetFileCount())

<<<<<<< HEAD
	// If Porch URL is provided, use Porch API (handles both dry-run and live modes).

=======
	// If Porch URL is provided, use Porch API (handles both dry-run and live modes)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if cfg.PorchURL != "" {
		return submitToPorch(cfg, result.Intent, pkg)
	}

<<<<<<< HEAD
	// Otherwise, write to filesystem.

	fsWriter := writer.NewFilesystemWriter(cfg.OutDir, cfg.DryRun)

	// Write package.

=======
	// Otherwise, write to filesystem
	fsWriter := writer.NewFilesystemWriter(cfg.OutDir, cfg.DryRun)

	// Write package
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	writeResult, err := fsWriter.WritePackage(pkg)
	if err != nil {
		return fmt.Errorf("failed to write package: %w", err)
	}

<<<<<<< HEAD
	// Report results.

	fmt.Printf("[porch-direct] Package written to: %s\n", writeResult.PackagePath)

	if len(writeResult.FilesWritten) > 0 {
		fmt.Printf("[porch-direct] Files written: %v\n", writeResult.FilesWritten)
	}

	if len(writeResult.FilesUpdated) > 0 {
		fmt.Printf("[porch-direct] Files updated: %v\n", writeResult.FilesUpdated)
	}

=======
	// Report results
	fmt.Printf("[porch-direct] Package written to: %s\n", writeResult.PackagePath)
	
	if len(writeResult.FilesWritten) > 0 {
		fmt.Printf("[porch-direct] Files written: %v\n", writeResult.FilesWritten)
	}
	if len(writeResult.FilesUpdated) > 0 {
		fmt.Printf("[porch-direct] Files updated: %v\n", writeResult.FilesUpdated)
	}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if len(writeResult.FilesSkipped) > 0 {
		fmt.Printf("[porch-direct] Files skipped (unchanged): %v\n", writeResult.FilesSkipped)
	}

	if writeResult.WasIdempotent {
		fmt.Printf("[porch-direct] Operation was idempotent (no changes needed)\n")
	}

<<<<<<< HEAD
	fmt.Printf("[porch-direct] Success! KRM package generated for %s scaling to %d replicas\n",

=======
	fmt.Printf("[porch-direct] Success! KRM package generated for %s scaling to %d replicas\n", 
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		result.Intent.Target, result.Intent.Replicas)

	return nil
}

<<<<<<< HEAD
// submitToPorch submits the generated package to Porch API.

func submitToPorch(cfg *Config, intent *intent.NetworkIntent, pkg *generator.Package) error {
	client := porch.NewClient(cfg.PorchURL, cfg.DryRun)

	// Create package request.

	packageReq := &porch.PackageRequest{
		Repository: cfg.Repo,

		Package: cfg.Package,

		Workspace: cfg.Workspace,

		Namespace: cfg.Namespace,

		Intent: intent,

		Files: func() json.RawMessage {
			files := pkg.GetFiles()
			if filesBytes, err := json.Marshal(files); err == nil {
				return filesBytes
			}
			return json.RawMessage(`{}`)
		}(),
	}

	// Create or update package revision.

=======
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
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	revision, err := client.CreateOrUpdatePackage(packageReq)
	if err != nil {
		return fmt.Errorf("failed to create/update package: %w", err)
	}
<<<<<<< HEAD

	fmt.Printf("[porch-direct] Package revision created: %s\n", revision.Name)

	// Submit proposal or auto-approve.

	if cfg.AutoApprove {

		if err := client.ApprovePackage(revision); err != nil {
			return fmt.Errorf("failed to approve package: %w", err)
		}

		fmt.Printf("[porch-direct] Package approved successfully\n")

	} else {

		proposal, err := client.SubmitProposal(revision)
		if err != nil {
			return fmt.Errorf("failed to submit proposal: %w", err)
		}

		fmt.Printf("[porch-direct] Proposal submitted: %s\n", proposal.ID)

		fmt.Printf("[porch-direct] Review and approve the proposal in Porch UI or use --auto-approve flag\n")

=======
	
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
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	return nil
}

<<<<<<< HEAD
// resolveRepo determines the repository name based on intent.

func resolveRepo(intent *intent.NetworkIntent) string {
	// Default mapping logic based on intent type.

	if intent.Target == "ran" || intent.Target == "gnb" || intent.Target == "du" || intent.Target == "cu" {
		return "ran-packages"
	}

	if intent.Target == "core" || intent.Target == "smf" || intent.Target == "upf" || intent.Target == "amf" {
		return "core-packages"
	}

	return "nephio-packages"
}

// resolvePackage generates a package name from intent.

func resolvePackage(intent *intent.NetworkIntent) string {
	return fmt.Sprintf("%s-scaling-%d", intent.Target, intent.Replicas)
}

// findProjectRoot walks up the directory tree to find the project root (containing go.mod).

=======
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
>>>>>>> 6835433495e87288b95961af7173d866977175ff
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
<<<<<<< HEAD

		gomodPath := filepath.Join(dir, "go.mod")

=======
		gomodPath := filepath.Join(dir, "go.mod")
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		if _, err := os.Stat(gomodPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
<<<<<<< HEAD

		if parent == dir {
			break // Reached filesystem root
		}

		dir = parent

=======
		if parent == dir {
			break // Reached filesystem root
		}
		dir = parent
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	return "", fmt.Errorf("go.mod not found in any parent directory")
}
