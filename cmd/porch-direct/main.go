
package main



import (

	"flag"

	"fmt"

	"log"

	"os"

	"path/filepath"



	"github.com/nephio-project/nephoran-intent-operator/internal/generator"

	"github.com/nephio-project/nephoran-intent-operator/internal/intent"

	"github.com/nephio-project/nephoran-intent-operator/internal/writer"

	"github.com/nephio-project/nephoran-intent-operator/pkg/porch"

)



func main() {

	var intentPath, outDir string

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

		IntentPath:  intentPath,

		OutDir:      outDir,

		DryRun:      dryRun,

		Minimal:     minimal,

		Repo:        repo,

		Package:     packageName,

		Workspace:   workspace,

		Namespace:   namespace,

		PorchURL:    porchURL,

		AutoApprove: autoApprove,

	}



	if err := run(cfg); err != nil {

		fmt.Fprintf(os.Stderr, "error: %v\n", err)

		log.Fatal(1)

	}

}



// Config represents a config.

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

	AutoApprove bool

}



func run(cfg *Config) error {

	// Determine project root by looking for go.mod.

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



	// Create loader.

	loader, err := intent.NewLoader(projectRoot)

	if err != nil {

		return fmt.Errorf("failed to create intent loader: %w", err)

	}



	// Load and validate intent.

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



	// Resolve repository and package names if not provided.

	if cfg.Repo == "" {

		cfg.Repo = resolveRepo(result.Intent)

	}

	if cfg.Package == "" {

		cfg.Package = resolvePackage(result.Intent)

	}



	// Create generator.

	packageGen := generator.NewPackageGenerator()



	// Generate package.

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



	// If Porch URL is provided, use Porch API (handles both dry-run and live modes).

	if cfg.PorchURL != "" {

		return submitToPorch(cfg, result.Intent, pkg)

	}



	// Otherwise, write to filesystem.

	fsWriter := writer.NewFilesystemWriter(cfg.OutDir, cfg.DryRun)



	// Write package.

	writeResult, err := fsWriter.WritePackage(pkg)

	if err != nil {

		return fmt.Errorf("failed to write package: %w", err)

	}



	// Report results.

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



// submitToPorch submits the generated package to Porch API.

func submitToPorch(cfg *Config, intent *intent.NetworkIntent, pkg *generator.Package) error {

	client := porch.NewClient(cfg.PorchURL, cfg.DryRun)



	// Create package request.

	packageReq := &porch.PackageRequest{

		Repository: cfg.Repo,

		Package:    cfg.Package,

		Workspace:  cfg.Workspace,

		Namespace:  cfg.Namespace,

		Intent:     intent,

		Files:      pkg.GetFiles(),

	}



	// Create or update package revision.

	revision, err := client.CreateOrUpdatePackage(packageReq)

	if err != nil {

		return fmt.Errorf("failed to create/update package: %w", err)

	}



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

	}



	return nil

}



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

