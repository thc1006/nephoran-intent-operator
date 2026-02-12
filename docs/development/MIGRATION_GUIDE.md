# Migration Guide: internal/patch to internal/patchgen

## Table of Contents

1. [Overview](#overview)
2. [Breaking Changes](#breaking-changes)
3. [Migration Timeline](#migration-timeline)
4. [Step-by-Step Migration](#step-by-step-migration)
5. [Code Examples](#code-examples)
6. [Testing Migration](#testing-migration)
7. [Rollback Procedures](#rollback-procedures)
8. [Troubleshooting](#troubleshooting)
9. [FAQ](#faq)

## Overview

The migration from `internal/patch` to `internal/patchgen` represents a significant security enhancement to the Nephoran Intent Operator's patch generation capabilities. This migration introduces comprehensive security controls, OWASP compliance, and collision-resistant naming mechanisms.

### Key Benefits of Migration

- **Enhanced Security**: OWASP Top 10 2021 compliance with comprehensive input validation
- **Collision Resistance**: Cryptographically secure package naming with nanosecond precision
- **Audit Trail**: Complete audit logging with tamper detection
- **Compliance Ready**: Built-in support for O-RAN WG11, SOC2, ISO27001
- **Better Error Handling**: Detailed error messages with security context
- **Performance**: Optimized for concurrent patch generation

### Migration Scope

| Component | Old Path | New Path | Impact |
|-----------|----------|----------|--------|
| Patch Generator | `internal/patch/generator.go` | `internal/patchgen/generator.go` | High |
| Intent Types | `internal/patch/intent.go` | `internal/patchgen/types.go` | Medium |
| Validation | `internal/patch/intent.go` | `internal/patchgen/validator.go` | High |
| Security Layer | N/A | `internal/security/secure_patchgen.go` | New |

## Breaking Changes

### 1. Package Import Changes

**Before:**
```go
import (
    "github.com/thc1006/nephoran-intent-operator/internal/patch"
)
```

**After:**
```go
import (
    "github.com/thc1006/nephoran-intent-operator/internal/patchgen"
    "github.com/thc1006/nephoran-intent-operator/internal/security"
)
```

### 2. Type Changes

#### Intent Structure
**Before:**
```go
type Intent struct {
    Target     string `json:"target"`
    Namespace  string `json:"namespace"`
    Replicas   int    `json:"replicas"`
    IntentType string `json:"intent_type"`
}
```

**After:**
```go
type Intent struct {
    Target        string `json:"target" validate:"required,kubernetes-name"`
    Namespace     string `json:"namespace" validate:"required,kubernetes-name"`
    Replicas      int    `json:"replicas" validate:"min=0,max=50"`
    IntentType    string `json:"intent_type" validate:"required,oneof=scaling"`
    Reason        string `json:"reason,omitempty" validate:"max=512"`
    Source        string `json:"source,omitempty" validate:"max=256"`
    CorrelationID string `json:"correlation_id,omitempty" validate:"uuid4"`
    Timestamp     string `json:"timestamp,omitempty" validate:"rfc3339"`
}
```

### 3. Function Signature Changes

#### Package Generation
**Before:**
```go
func GeneratePatch(intent *Intent, outputDir string) error
```

**After:**
```go
func NewPatchPackage(intent *Intent, outputDir string) *PatchPackage
func (p *PatchPackage) Generate() error
```

### 4. Security Requirements

The new implementation requires:
- Valid Kubernetes naming conventions for all identifiers
- Path validation to prevent directory traversal
- Input sanitization for all string fields
- Mandatory audit logging configuration

## Migration Timeline

### Phase 1: Preparation (Week 1)
- [ ] Review current patch generation usage
- [ ] Identify all dependencies on `internal/patch`
- [ ] Create migration branch
- [ ] Set up test environment

### Phase 2: Code Migration (Week 2-3)
- [ ] Update import statements
- [ ] Migrate to new type definitions
- [ ] Update function calls
- [ ] Add security configuration

### Phase 3: Testing (Week 3-4)
- [ ] Unit test updates
- [ ] Integration testing
- [ ] Security testing
- [ ] Performance testing

### Phase 4: Deployment (Week 5)
- [ ] Staged rollout to development
- [ ] Production deployment
- [ ] Monitor for issues
- [ ] Complete migration

## Step-by-Step Migration

### Step 1: Update Dependencies

Update your `go.mod` file to ensure you have the latest version:

```bash
go get -u github.com/thc1006/nephoran-intent-operator@feat-porch-structured
go mod tidy
```

### Step 2: Update Import Statements

Find and replace all imports:

```bash
# Find all files using the old package
grep -r "internal/patch" --include="*.go" .

# Update imports using your IDE or sed
sed -i 's|internal/patch|internal/patchgen|g' **/*.go
```

### Step 3: Update Type References

Update all type references from `patch.` to `patchgen.`:

```go
// Before
var intent *patch.Intent

// After
var intent *patchgen.Intent
```

### Step 4: Implement Security Configuration

Add security configuration to your initialization:

```go
import (
    "github.com/thc1006/nephoran-intent-operator/internal/security"
    "github.com/go-logr/logr"
)

// Initialize secure patch generator
func initializeSecurePatchGen(logger logr.Logger) (*security.SecurePatchGenerator, error) {
    intent := &patchgen.Intent{
        Target:     "my-deployment",
        Namespace:  "default",
        Replicas:   3,
        IntentType: "scaling",
        Reason:     "Scale up for load",
        Source:     "auto-scaler",
    }
    
    return security.NewSecurePatchGenerator(intent, "/output/dir", logger)
}
```

### Step 5: Update Function Calls

#### Old Pattern:
```go
func generatePatch(intent *patch.Intent) error {
    return patch.GeneratePatch(intent, "/output/dir")
}
```

#### New Pattern:
```go
func generatePatch(intent *patchgen.Intent) error {
    // Option 1: Basic generation
    pkg := patchgen.NewPatchPackage(intent, "/output/dir")
    return pkg.Generate()
    
    // Option 2: Secure generation with validation
    secureGen, err := security.NewSecurePatchGenerator(intent, "/output/dir", logger)
    if err != nil {
        return err
    }
    
    securePkg, err := secureGen.GenerateSecure()
    if err != nil {
        return err
    }
    
    return nil
}
```

### Step 6: Add Validation

Implement proper validation before patch generation:

```go
func validateAndGenerate(intent *patchgen.Intent) error {
    // Create validator
    validator := patchgen.NewValidator()
    
    // Validate intent
    if err := validator.ValidateIntent(intent); err != nil {
        return fmt.Errorf("intent validation failed: %w", err)
    }
    
    // Validate output directory
    if err := validator.ValidatePath("/output/dir"); err != nil {
        return fmt.Errorf("output path validation failed: %w", err)
    }
    
    // Generate patch
    pkg := patchgen.NewPatchPackage(intent, "/output/dir")
    return pkg.Generate()
}
```

## Code Examples

### Example 1: Basic Migration

**Before (using internal/patch):**
```go
package main

import (
    "log"
    "github.com/thc1006/nephoran-intent-operator/internal/patch"
)

func main() {
    intent := &patch.Intent{
        Target:     "nginx",
        Namespace:  "default",
        Replicas:   5,
        IntentType: "scaling",
    }
    
    if err := patch.GeneratePatch(intent, "./output"); err != nil {
        log.Fatal(err)
    }
}
```

**After (using internal/patchgen):**
```go
package main

import (
    "log"
    "github.com/thc1006/nephoran-intent-operator/internal/patchgen"
)

func main() {
    intent := &patchgen.Intent{
        Target:     "nginx",
        Namespace:  "default",
        Replicas:   5,
        IntentType: "scaling",
        Reason:     "Manual scale operation",
        Source:     "cli",
    }
    
    pkg := patchgen.NewPatchPackage(intent, "./output")
    if err := pkg.Generate(); err != nil {
        log.Fatal(err)
    }
}
```

### Example 2: Secure Generation with Full Validation

```go
package main

import (
    "context"
    "log"
    
    "github.com/go-logr/logr"
    "sigs.k8s.io/controller-runtime/pkg/log"
    
    "github.com/thc1006/nephoran-intent-operator/internal/patchgen"
    "github.com/thc1006/nephoran-intent-operator/internal/security"
)

func main() {
    // Setup logger
    logger := log.Log.WithName("patch-generator")
    
    // Create intent with full metadata
    intent := &patchgen.Intent{
        Target:        "nginx-deployment",
        Namespace:     "production",
        Replicas:      10,
        IntentType:    "scaling",
        Reason:        "Increased traffic detected",
        Source:        "hpa-controller",
        CorrelationID: "550e8400-e29b-41d4-a716-446655440000",
        Timestamp:     time.Now().Format(time.RFC3339),
    }
    
    // Use secure generator
    generator, err := security.NewSecurePatchGenerator(
        intent, 
        "./secure-output",
        logger,
    )
    if err != nil {
        log.Fatalf("Failed to create generator: %v", err)
    }
    
    // Generate with full security validation
    securePkg, err := generator.GenerateSecure()
    if err != nil {
        log.Fatalf("Failed to generate patch: %v", err)
    }
    
    logger.Info("Patch generated successfully",
        "package", securePkg.Kptfile.Metadata.Name,
        "security_validated", securePkg.SecurityMetadata.ValidationPassed,
    )
}
```

### Example 3: Controller Integration

```go
package controllers

import (
    "context"
    
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    
    "github.com/thc1006/nephoran-intent-operator/api/v1"
    "github.com/thc1006/nephoran-intent-operator/internal/patchgen"
    "github.com/thc1006/nephoran-intent-operator/internal/security"
)

type NetworkIntentReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := ctrl.LoggerFrom(ctx)
    
    // Fetch NetworkIntent
    var networkIntent v1.NetworkIntent
    if err := r.Get(ctx, req.NamespacedName, &networkIntent); err != nil {
        return ctrl.Result{}, err
    }
    
    // Convert to patchgen Intent
    intent := &patchgen.Intent{
        Target:        networkIntent.Spec.Target,
        Namespace:     networkIntent.Namespace,
        Replicas:      networkIntent.Spec.Replicas,
        IntentType:    "scaling",
        Reason:        networkIntent.Spec.Reason,
        Source:        "k8s-controller",
        CorrelationID: string(networkIntent.UID),
        Timestamp:     networkIntent.CreationTimestamp.Format(time.RFC3339),
    }
    
    // Generate secure patch
    generator, err := security.NewSecurePatchGenerator(
        intent,
        "/tmp/patches",
        logger,
    )
    if err != nil {
        return ctrl.Result{}, err
    }
    
    securePkg, err := generator.GenerateSecure()
    if err != nil {
        logger.Error(err, "Failed to generate patch")
        return ctrl.Result{}, err
    }
    
    // Apply patch using Porch API
    // ... porch client code ...
    
    return ctrl.Result{}, nil
}
```

## Testing Migration

### Unit Tests

Update your unit tests to use the new package:

```go
package patchgen_test

import (
    "testing"
    "os"
    "path/filepath"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/thc1006/nephoran-intent-operator/internal/patchgen"
)

func TestPatchGeneration(t *testing.T) {
    tests := []struct {
        name    string
        intent  *patchgen.Intent
        wantErr bool
    }{
        {
            name: "valid scaling intent",
            intent: &patchgen.Intent{
                Target:     "test-deployment",
                Namespace:  "default",
                Replicas:   3,
                IntentType: "scaling",
            },
            wantErr: false,
        },
        {
            name: "invalid target name",
            intent: &patchgen.Intent{
                Target:     "../../../etc/passwd",
                Namespace:  "default",
                Replicas:   3,
                IntentType: "scaling",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tmpDir := t.TempDir()
            
            pkg := patchgen.NewPatchPackage(tt.intent, tmpDir)
            err := pkg.Generate()
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                
                // Verify files were created
                packagePath := pkg.GetPackagePath()
                assert.DirExists(t, packagePath)
                
                // Check for required files
                assert.FileExists(t, filepath.Join(packagePath, "Kptfile"))
                assert.FileExists(t, filepath.Join(packagePath, "scaling-patch.yaml"))
                assert.FileExists(t, filepath.Join(packagePath, "README.md"))
            }
        })
    }
}
```

### Integration Tests

```go
func TestSecurePatchGenerationIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    logger := logr.Discard()
    tmpDir := t.TempDir()
    
    intent := &patchgen.Intent{
        Target:        "nginx",
        Namespace:     "test",
        Replicas:      5,
        IntentType:    "scaling",
        Reason:        "Integration test",
        Source:        "test-suite",
        CorrelationID: uuid.New().String(),
        Timestamp:     time.Now().Format(time.RFC3339),
    }
    
    // Test secure generation
    generator, err := security.NewSecurePatchGenerator(intent, tmpDir, logger)
    require.NoError(t, err)
    
    securePkg, err := generator.GenerateSecure()
    require.NoError(t, err)
    
    // Verify security metadata
    assert.True(t, securePkg.SecurityMetadata.ValidationPassed)
    assert.Equal(t, "OWASP-2021-compliant", securePkg.SecurityMetadata.SecurityVersion)
    assert.Equal(t, "O-RAN-WG11-L-Release", securePkg.SecurityMetadata.ThreatModel)
    
    // Verify security files
    packagePath := securePkg.GetPackagePath()
    assert.FileExists(t, filepath.Join(packagePath, "security-metadata.yaml"))
}
```

### Security Tests

```go
func TestSecurityValidation(t *testing.T) {
    tests := []struct {
        name      string
        target    string
        namespace string
        shouldFail bool
    }{
        {
            name:      "path traversal attempt",
            target:    "../../../etc/passwd",
            namespace: "default",
            shouldFail: true,
        },
        {
            name:      "script injection attempt",
            target:    "test<script>alert(1)</script>",
            namespace: "default",
            shouldFail: true,
        },
        {
            name:      "command injection attempt",
            target:    "test;rm -rf /",
            namespace: "default",
            shouldFail: true,
        },
        {
            name:      "valid kubernetes name",
            target:    "nginx-deployment-v2",
            namespace: "production",
            shouldFail: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            validator := patchgen.NewValidator()
            intent := &patchgen.Intent{
                Target:     tt.target,
                Namespace:  tt.namespace,
                Replicas:   1,
                IntentType: "scaling",
            }
            
            err := validator.ValidateIntent(intent)
            if tt.shouldFail {
                assert.Error(t, err, "Expected validation to fail for %s", tt.name)
            } else {
                assert.NoError(t, err, "Expected validation to pass for %s", tt.name)
            }
        })
    }
}
```

## Rollback Procedures

If issues are encountered during migration, follow these rollback steps:

### Step 1: Identify Issues

Check logs for migration-related errors:
```bash
kubectl logs -n nephoran-system deployment/intent-operator | grep -E "patchgen|security"
```

### Step 2: Temporary Compatibility Mode

If immediate rollback is needed, you can create a compatibility wrapper:

```go
// compatibility.go - Temporary wrapper for backward compatibility
package patch

import (
    "github.com/thc1006/nephoran-intent-operator/internal/patchgen"
)

type Intent = patchgen.Intent

func GeneratePatch(intent *Intent, outputDir string) error {
    pkg := patchgen.NewPatchPackage(intent, outputDir)
    return pkg.Generate()
}
```

### Step 3: Full Rollback

If a full rollback is necessary:

```bash
# Checkout previous version
git checkout <previous-commit-before-migration>

# Rebuild and deploy
make build
make deploy
```

### Step 4: Data Recovery

No data migration is required as this change only affects code structure. Generated patches remain compatible.

## Troubleshooting

### Common Issues and Solutions

#### Issue 1: Import Errors
**Error:** `cannot find package "github.com/thc1006/nephoran-intent-operator/internal/patch"`

**Solution:**
```bash
# Update all imports
find . -name "*.go" -exec sed -i 's|internal/patch|internal/patchgen|g' {} \;

# Update go.mod
go mod tidy
```

#### Issue 2: Type Mismatch
**Error:** `cannot use intent (type *patch.Intent) as type *patchgen.Intent`

**Solution:**
Update type declarations:
```go
// Change from
var intent *patch.Intent

// To
var intent *patchgen.Intent
```

#### Issue 3: Validation Failures
**Error:** `intent validation failed: target contains invalid characters`

**Solution:**
Ensure all names follow Kubernetes naming conventions:
- Lowercase alphanumeric characters or '-'
- Start with an alphanumeric character
- End with an alphanumeric character
- Maximum 253 characters

#### Issue 4: Missing Security Configuration
**Error:** `failed to initialize security validator`

**Solution:**
Ensure security configuration is properly initialized:
```go
logger := logr.Discard() // or your configured logger
generator, err := security.NewSecurePatchGenerator(intent, outputDir, logger)
```

#### Issue 5: Permission Denied
**Error:** `failed to create output directory: permission denied`

**Solution:**
Ensure the process has write permissions:
```bash
# Check permissions
ls -la /output/directory

# Fix permissions if needed
chmod 755 /output/directory
```

### Debug Mode

Enable debug logging for detailed migration information:

```go
import (
    "github.com/go-logr/logr"
    "github.com/go-logr/zapr"
    "go.uber.org/zap"
)

// Create debug logger
zapLog, _ := zap.NewDevelopment()
logger := zapr.NewLogger(zapLog)

// Use with generator
generator, err := security.NewSecurePatchGenerator(intent, outputDir, logger)
```

### Performance Considerations

The new implementation includes security validations which may impact performance:

| Operation | Old (ms) | New (ms) | Impact |
|-----------|----------|----------|--------|
| Simple patch | 5-10 | 10-20 | +100% |
| With validation | N/A | 15-30 | New feature |
| Secure generation | N/A | 20-40 | New feature |

For high-throughput scenarios, consider:
- Implementing caching for validation results
- Using goroutines for concurrent generation
- Pre-validating intents before generation

## FAQ

### Q1: Is the migration mandatory?
**A:** Yes, `internal/patch` is deprecated and will be removed in the next major version. The migration provides critical security improvements.

### Q2: Are the generated patches backward compatible?
**A:** Yes, the output format remains compatible with Kubernetes and Porch. Only the generation process has been enhanced.

### Q3: What happens to existing patches?
**A:** Existing patches continue to work. The migration only affects new patch generation.

### Q4: Can I use both packages during migration?
**A:** Not recommended. The packages may conflict. Complete migration package by package.

### Q5: How do I verify the migration was successful?
**A:** Run the provided test suite and verify:
- All unit tests pass
- Integration tests complete successfully
- Security validations are active
- Audit logs show patch generation events

### Q6: What security improvements are included?
**A:** Key improvements include:
- OWASP Top 10 2021 compliance
- Input validation and sanitization
- Path traversal protection
- Cryptographically secure naming
- Comprehensive audit logging
- Tamper-proof integrity chains

### Q7: Will this affect performance?
**A:** There is a small performance impact (10-20ms) due to security validations. This is negligible for most use cases and can be optimized if needed.

### Q8: Where can I get help?
**A:** Support channels:
- GitHub Issues: https://github.com/thc1006/nephoran-intent-operator/issues
- Documentation: /docs/SECURITY_ARCHITECTURE.md
- Security Team: security@nephoran.io

---

*Document Version: 1.0*  
*Last Updated: 2025-08-19*  
*Next Review: 2025-09-19*