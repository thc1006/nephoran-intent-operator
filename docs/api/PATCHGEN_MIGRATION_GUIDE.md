# Patchgen Migration Guide

## Executive Summary

This guide provides comprehensive instructions for migrating from the legacy `internal/patch` module to the new security-enhanced `internal/patchgen` module in the Nephoran Intent Operator. The new module addresses critical security vulnerabilities while providing improved functionality and maintainability.

## Table of Contents

- [Overview](#overview)
- [Security Improvements](#security-improvements)
- [Breaking Changes](#breaking-changes)
- [Migration Steps](#migration-steps)
- [Code Examples](#code-examples)
- [API Reference](#api-reference)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Overview

### Why Migrate?

The `internal/patchgen` module represents a complete rewrite of the patch generation system with:

- **Enhanced Security**: Comprehensive protection against path traversal, command injection, and timestamp collisions
- **Improved Validation**: JSON Schema 2020-12 based validation with strict type checking
- **Better Structure**: Clean separation of concerns with dedicated types and validators
- **Production Readiness**: Battle-tested security measures suitable for enterprise deployments
- **Compliance**: Meets O-RAN security requirements and industry best practices

### Key Differences

| Aspect | internal/patch (Legacy) | internal/patchgen (New) |
|--------|------------------------|-------------------------|
| **Security** | Basic file operations | Zero-trust validation, path sanitization |
| **Validation** | Manual field checks | JSON Schema 2020-12 validation |
| **Timestamps** | Unix seconds (collision-prone) | RFC3339Nano + random suffix |
| **Package Names** | Simple timestamp | Collision-resistant with entropy |
| **Error Handling** | Basic errors | Detailed, actionable error messages |
| **Testing** | Limited coverage | Comprehensive security test suite |

## Security Improvements

### 1. Path Traversal Prevention

The new module implements multiple layers of defense against path traversal attacks:

```go
// Old (Vulnerable)
packageDir := filepath.Join(g.OutputDir, packageName)
os.MkdirAll(packageDir, 0755) // No validation!

// New (Secure)
if info, err := os.Stat(p.OutputDir); os.IsNotExist(err) {
    return fmt.Errorf("output directory %s does not exist", p.OutputDir)
} else if !info.IsDir() {
    return fmt.Errorf("output path %s is not a directory", p.OutputDir)
}
// Additional validation layers applied
```

### 2. Timestamp Collision Resistance

Prevents race conditions and duplicate package names:

```go
// Old (Collision-prone)
packageName := fmt.Sprintf("%s-scaling-%d", g.Intent.Target, time.Now().Unix())

// New (Collision-resistant)
func generatePackageName(target string) string {
    now := time.Now().UTC()
    randomSuffix, _ := rand.Int(rand.Reader, big.NewInt(10000))
    timestamp := fmt.Sprintf("%s-%04d", now.Format("20060102-150405"), randomSuffix.Int64())
    return fmt.Sprintf("%s-scaling-patch-%s", target, timestamp)
}
```

### 3. Command Injection Protection

All user inputs are validated against strict schemas:

```go
// JSON Schema validation prevents injection
const IntentSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "target": {
      "type": "string",
      "minLength": 1,
      "pattern": "^[a-z0-9](?:[-a-z0-9]*[a-z0-9])?$"
    }
  }
}`
```

## Breaking Changes

### 1. Type Changes

```go
// Old
type Generator struct {
    Intent    *Intent
    OutputDir string
}

// New
type PatchPackage struct {
    Kptfile   *Kptfile
    PatchFile *PatchFile
    OutputDir string
    Intent    *Intent
}
```

### 2. Constructor Changes

```go
// Old
generator := patch.NewGenerator(intent, outputDir)

// New
package := patchgen.NewPatchPackage(intent, outputDir)
```

### 3. Method Signatures

```go
// Old
err := generator.Generate()

// New
err := package.Generate()
packagePath := package.GetPackagePath() // New method
```

### 4. Validation Requirements

The new module enforces strict validation:

```go
// New required validation
validator, err := patchgen.NewValidator(logger)
if err != nil {
    return err
}

intent, err := validator.ValidateIntent(intentData)
if err != nil {
    return fmt.Errorf("validation failed: %w", err)
}
```

## Migration Steps

### Step 1: Update Imports

Replace all imports of the old module:

```go
// Old
import "github.com/nephoran/internal/patch"

// New
import "github.com/nephoran/internal/patchgen"
```

### Step 2: Update Type References

Update all type references throughout your codebase:

```go
// Old
var generator *patch.Generator
var intent *patch.Intent

// New
var package *patchgen.PatchPackage
var intent *patchgen.Intent
```

### Step 3: Add Validation

Implement validation before processing:

```go
import (
    "github.com/go-logr/logr"
    "github.com/nephoran/internal/patchgen"
)

func processIntent(intentData []byte, outputDir string, logger logr.Logger) error {
    // Create validator
    validator, err := patchgen.NewValidator(logger)
    if err != nil {
        return fmt.Errorf("failed to create validator: %w", err)
    }
    
    // Validate intent
    intent, err := validator.ValidateIntent(intentData)
    if err != nil {
        return fmt.Errorf("intent validation failed: %w", err)
    }
    
    // Create package
    pkg := patchgen.NewPatchPackage(intent, outputDir)
    
    // Generate files
    if err := pkg.Generate(); err != nil {
        return fmt.Errorf("package generation failed: %w", err)
    }
    
    logger.Info("Package generated successfully", 
        "path", pkg.GetPackagePath())
    
    return nil
}
```

### Step 4: Update Error Handling

The new module provides more detailed errors:

```go
// Old
if err := generator.Generate(); err != nil {
    log.Fatal(err) // Generic error
}

// New
if err := pkg.Generate(); err != nil {
    switch {
    case strings.Contains(err.Error(), "does not exist"):
        // Handle missing directory
    case strings.Contains(err.Error(), "validation failed"):
        // Handle validation error
    default:
        // Handle other errors
    }
}
```

### Step 5: Update Tests

Update your test suites to use the new API:

```go
func TestPatchGeneration(t *testing.T) {
    // Create test intent
    intent := &patchgen.Intent{
        IntentType: "scaling",
        Target:     "test-deployment",
        Namespace:  "default",
        Replicas:   3,
    }
    
    // Create temporary directory
    tempDir := t.TempDir()
    
    // Generate package
    pkg := patchgen.NewPatchPackage(intent, tempDir)
    err := pkg.Generate()
    
    require.NoError(t, err)
    
    // Verify files exist
    packagePath := pkg.GetPackagePath()
    assert.DirExists(t, packagePath)
    assert.FileExists(t, filepath.Join(packagePath, "Kptfile"))
    assert.FileExists(t, filepath.Join(packagePath, "scaling-patch.yaml"))
    assert.FileExists(t, filepath.Join(packagePath, "README.md"))
}
```

## Code Examples

### Complete Migration Example

#### Before (Legacy)

```go
package main

import (
    "encoding/json"
    "log"
    "os"
    
    "github.com/nephoran/internal/patch"
)

func main() {
    // Read intent file
    data, err := os.ReadFile("intent.json")
    if err != nil {
        log.Fatal(err)
    }
    
    // Parse intent
    var intent patch.Intent
    if err := json.Unmarshal(data, &intent); err != nil {
        log.Fatal(err)
    }
    
    // Generate patch
    generator := patch.NewGenerator(&intent, "./output")
    if err := generator.Generate(); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Patch generated successfully")
}
```

#### After (New)

```go
package main

import (
    "log"
    "os"
    
    "github.com/go-logr/logr"
    "github.com/go-logr/zapr"
    "go.uber.org/zap"
    
    "github.com/nephoran/internal/patchgen"
)

func main() {
    // Setup logger
    zapLog, _ := zap.NewProduction()
    logger := zapr.NewLogger(zapLog)
    
    // Create validator
    validator, err := patchgen.NewValidator(logger)
    if err != nil {
        log.Fatalf("Failed to create validator: %v", err)
    }
    
    // Read and validate intent file
    data, err := os.ReadFile("intent.json")
    if err != nil {
        log.Fatalf("Failed to read intent file: %v", err)
    }
    
    intent, err := validator.ValidateIntent(data)
    if err != nil {
        log.Fatalf("Intent validation failed: %v", err)
    }
    
    // Ensure output directory exists
    outputDir := "./output"
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        log.Fatalf("Failed to create output directory: %v", err)
    }
    
    // Generate patch package
    pkg := patchgen.NewPatchPackage(intent, outputDir)
    if err := pkg.Generate(); err != nil {
        log.Fatalf("Package generation failed: %v", err)
    }
    
    logger.Info("Package generated successfully",
        "path", pkg.GetPackagePath(),
        "target", intent.Target,
        "replicas", intent.Replicas)
}
```

### Integration with Controllers

```go
package controllers

import (
    "context"
    "encoding/json"
    
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"
    
    "github.com/nephoran/api/v1"
    "github.com/nephoran/internal/patchgen"
)

type NetworkIntentReconciler struct {
    client.Client
    Scheme    *runtime.Scheme
    Validator *patchgen.Validator
}

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := log.FromContext(ctx)
    
    // Fetch NetworkIntent
    var networkIntent v1.NetworkIntent
    if err := r.Get(ctx, req.NamespacedName, &networkIntent); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    
    // Convert to patchgen Intent
    intentData, err := json.Marshal(map[string]interface{}{
        "intent_type": "scaling",
        "target":      networkIntent.Spec.Target,
        "namespace":   networkIntent.Namespace,
        "replicas":    networkIntent.Spec.Replicas,
        "source":      "k8s-controller",
    })
    if err != nil {
        return ctrl.Result{}, err
    }
    
    // Validate intent
    intent, err := r.Validator.ValidateIntent(intentData)
    if err != nil {
        logger.Error(err, "Intent validation failed")
        // Update status with validation error
        networkIntent.Status.State = "ValidationFailed"
        networkIntent.Status.Message = err.Error()
        r.Status().Update(ctx, &networkIntent)
        return ctrl.Result{}, nil
    }
    
    // Generate patch package
    outputDir := "/tmp/patches"
    pkg := patchgen.NewPatchPackage(intent, outputDir)
    if err := pkg.Generate(); err != nil {
        logger.Error(err, "Package generation failed")
        return ctrl.Result{}, err
    }
    
    // Apply to Porch/kpt
    if err := r.applyToPorch(ctx, pkg.GetPackagePath()); err != nil {
        return ctrl.Result{}, err
    }
    
    // Update status
    networkIntent.Status.State = "Applied"
    networkIntent.Status.PackagePath = pkg.GetPackagePath()
    r.Status().Update(ctx, &networkIntent)
    
    return ctrl.Result{}, nil
}
```

## API Reference

### Types

#### Intent

```go
type Intent struct {
    IntentType    string `json:"intent_type"`
    Target        string `json:"target"`
    Namespace     string `json:"namespace"`
    Replicas      int    `json:"replicas"`
    Reason        string `json:"reason,omitempty"`
    Source        string `json:"source,omitempty"`
    CorrelationID string `json:"correlation_id,omitempty"`
}
```

#### PatchPackage

```go
type PatchPackage struct {
    Kptfile   *Kptfile
    PatchFile *PatchFile
    OutputDir string
    Intent    *Intent
}
```

### Functions

#### NewValidator

```go
func NewValidator(logger logr.Logger) (*Validator, error)
```

Creates a new validator with JSON Schema 2020-12 support.

#### ValidateIntent

```go
func (v *Validator) ValidateIntent(intentData []byte) (*Intent, error)
```

Validates raw JSON data against the Intent schema.

#### NewPatchPackage

```go
func NewPatchPackage(intent *Intent, outputDir string) *PatchPackage
```

Creates a new patch package with collision-resistant naming.

#### Generate

```go
func (p *PatchPackage) Generate() error
```

Generates all package files with security validation.

#### GetPackagePath

```go
func (p *PatchPackage) GetPackagePath() string
```

Returns the full path to the generated package.

## Troubleshooting

### Common Issues

#### 1. Validation Errors

**Problem**: Intent validation fails with schema errors.

**Solution**:
```go
// Check the exact validation error
intent, err := validator.ValidateIntent(data)
if err != nil {
    log.Printf("Validation error: %v", err)
    // Common issues:
    // - Missing required fields
    // - Invalid field types
    // - Pattern violations (e.g., invalid Kubernetes names)
}
```

#### 2. Directory Permission Errors

**Problem**: Cannot create output directory.

**Solution**:
```go
// Ensure proper permissions
outputDir := "/var/lib/nephoran/patches"
if err := os.MkdirAll(outputDir, 0755); err != nil {
    if os.IsPermission(err) {
        log.Printf("Permission denied: %v", err)
        // Run with appropriate privileges or change directory
    }
}
```

#### 3. Package Name Collisions

**Problem**: Package names still colliding in high-concurrency scenarios.

**Solution**:
The new module uses cryptographically secure random numbers. If collisions still occur:

```go
// Implement retry logic
var pkg *patchgen.PatchPackage
var err error

for retries := 0; retries < 3; retries++ {
    pkg = patchgen.NewPatchPackage(intent, outputDir)
    err = pkg.Generate()
    if err == nil || !strings.Contains(err.Error(), "already exists") {
        break
    }
    time.Sleep(100 * time.Millisecond)
}
```

#### 4. Schema Compilation Errors

**Problem**: JSON Schema fails to compile.

**Solution**:
```go
// Ensure JSON Schema draft compatibility
compiler := jsonschema.NewCompiler()
compiler.DefaultDraft(jsonschema.Draft2020) // Must be 2020-12
```

### Migration Validation Checklist

Use this checklist to ensure successful migration:

- [ ] All imports updated from `internal/patch` to `internal/patchgen`
- [ ] Validator created and configured with appropriate logger
- [ ] All intent data validated before processing
- [ ] Error handling updated to handle detailed error messages
- [ ] Output directories validated for existence and permissions
- [ ] Package path retrieval using `GetPackagePath()` method
- [ ] Tests updated to use new API and validation
- [ ] Security test cases added for path traversal prevention
- [ ] Concurrent generation tested for collision resistance
- [ ] Documentation updated to reflect new module usage

## Best Practices

### 1. Always Validate Input

Never skip validation, even for trusted sources:

```go
// Always validate, even internal data
intent, err := validator.ValidateIntent(data)
if err != nil {
    return fmt.Errorf("validation failed: %w", err)
}
```

### 2. Use Structured Logging

Leverage the logger for better observability:

```go
logger.Info("Generating patch package",
    "intent_type", intent.IntentType,
    "target", intent.Target,
    "namespace", intent.Namespace,
    "replicas", intent.Replicas)
```

### 3. Handle Errors Gracefully

Provide meaningful error messages:

```go
if err := pkg.Generate(); err != nil {
    return fmt.Errorf("failed to generate package for %s/%s: %w", 
        intent.Namespace, intent.Target, err)
}
```

### 4. Implement Retry Logic

For production systems, implement retry with exponential backoff:

```go
func generateWithRetry(intent *patchgen.Intent, outputDir string, maxRetries int) error {
    var lastErr error
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        pkg := patchgen.NewPatchPackage(intent, outputDir)
        if err := pkg.Generate(); err == nil {
            return nil
        } else {
            lastErr = err
            backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
            time.Sleep(backoff)
        }
    }
    
    return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
```

### 5. Monitor Package Generation

Track metrics for operational visibility:

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    packageGenerations = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "patchgen_packages_generated_total",
            Help: "Total number of patch packages generated",
        },
        []string{"intent_type", "status"},
    )
)

func generatePackage(intent *patchgen.Intent, outputDir string) error {
    pkg := patchgen.NewPatchPackage(intent, outputDir)
    
    if err := pkg.Generate(); err != nil {
        packageGenerations.WithLabelValues(intent.IntentType, "failed").Inc()
        return err
    }
    
    packageGenerations.WithLabelValues(intent.IntentType, "success").Inc()
    return nil
}
```

## Security Considerations

### 1. Directory Restrictions

Always restrict output directories:

```go
var allowedOutputDirs = []string{
    "/var/lib/nephoran/patches",
    "/tmp/nephoran-patches",
}

func isAllowedOutputDir(dir string) bool {
    absDir, _ := filepath.Abs(dir)
    for _, allowed := range allowedOutputDirs {
        if strings.HasPrefix(absDir, allowed) {
            return true
        }
    }
    return false
}
```

### 2. Input Sanitization

The module automatically sanitizes inputs, but additional checks can be added:

```go
// Additional sanitization for paranoid mode
if strings.Contains(intent.Target, "..") || 
   strings.Contains(intent.Target, "/") ||
   strings.Contains(intent.Target, "\\") {
    return fmt.Errorf("invalid target name: %s", intent.Target)
}
```

### 3. Audit Logging

Implement audit logging for compliance:

```go
type AuditLogger struct {
    logger logr.Logger
}

func (a *AuditLogger) LogPackageGeneration(intent *patchgen.Intent, path string, userID string) {
    a.logger.Info("AUDIT: Package generated",
        "user_id", userID,
        "intent_type", intent.IntentType,
        "target", intent.Target,
        "namespace", intent.Namespace,
        "replicas", intent.Replicas,
        "package_path", path,
        "timestamp", time.Now().UTC().Format(time.RFC3339Nano))
}
```

## Performance Considerations

### Concurrent Generation

The module is thread-safe for concurrent use:

```go
func processBatch(intents []*patchgen.Intent, outputDir string) error {
    var wg sync.WaitGroup
    errors := make(chan error, len(intents))
    
    for _, intent := range intents {
        wg.Add(1)
        go func(i *patchgen.Intent) {
            defer wg.Done()
            
            pkg := patchgen.NewPatchPackage(i, outputDir)
            if err := pkg.Generate(); err != nil {
                errors <- err
            }
        }(intent)
    }
    
    wg.Wait()
    close(errors)
    
    // Collect errors
    var errs []error
    for err := range errors {
        errs = append(errs, err)
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("batch processing had %d errors", len(errs))
    }
    
    return nil
}
```

### Memory Usage

The module is memory-efficient, but for large-scale operations:

```go
// Use streaming for large batches
func streamingProcessor(intentChan <-chan *patchgen.Intent, outputDir string) {
    for intent := range intentChan {
        pkg := patchgen.NewPatchPackage(intent, outputDir)
        if err := pkg.Generate(); err != nil {
            log.Printf("Failed to generate package: %v", err)
            continue
        }
        
        // Immediately process or upload the package
        processPackage(pkg.GetPackagePath())
        
        // Clean up if needed
        os.RemoveAll(pkg.GetPackagePath())
    }
}
```

## Conclusion

The migration to `internal/patchgen` provides critical security improvements while maintaining backward compatibility where possible. The new module's zero-trust approach to validation, combined with collision-resistant naming and comprehensive error handling, makes it suitable for production deployments in security-sensitive environments.

For additional support or questions about the migration, please consult the security team or open an issue in the repository.

## Appendix: Quick Reference

### Migration Checklist

```bash
# 1. Update go.mod dependencies
go get -u github.com/nephoran/internal/patchgen

# 2. Update imports
find . -type f -name "*.go" -exec sed -i 's/internal\/patch/internal\/patchgen/g' {} \;

# 3. Run tests
go test ./... -v

# 4. Update CI/CD pipelines
# Ensure new validation steps are included

# 5. Deploy to staging
kubectl apply -f deployment/staging/

# 6. Monitor for errors
kubectl logs -f deployment/nephoran-operator -n nephoran-system

# 7. Deploy to production
kubectl apply -f deployment/production/
```

### Common Patterns

```go
// Pattern: Validate and Generate
func validateAndGenerate(data []byte, outputDir string) (string, error) {
    validator, _ := patchgen.NewValidator(logger)
    intent, err := validator.ValidateIntent(data)
    if err != nil {
        return "", err
    }
    
    pkg := patchgen.NewPatchPackage(intent, outputDir)
    if err := pkg.Generate(); err != nil {
        return "", err
    }
    
    return pkg.GetPackagePath(), nil
}

// Pattern: Batch Processing with Error Collection
func batchProcess(intents [][]byte, outputDir string) ([]string, []error) {
    validator, _ := patchgen.NewValidator(logger)
    var paths []string
    var errors []error
    
    for _, data := range intents {
        intent, err := validator.ValidateIntent(data)
        if err != nil {
            errors = append(errors, err)
            continue
        }
        
        pkg := patchgen.NewPatchPackage(intent, outputDir)
        if err := pkg.Generate(); err != nil {
            errors = append(errors, err)
            continue
        }
        
        paths = append(paths, pkg.GetPackagePath())
    }
    
    return paths, errors
}
```
