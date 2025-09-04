# Security Migration Guide: internal/patch to internal/patchgen

## Executive Summary

This guide provides comprehensive instructions for migrating from the legacy `internal/patch` package to the new, security-enhanced `internal/patchgen` package. The migration addresses critical security vulnerabilities including path traversal, injection attacks, and improper input validation while maintaining backward compatibility where possible.

## Table of Contents

1. [Overview](#overview)
2. [Breaking Changes](#breaking-changes)
3. [Migration Prerequisites](#migration-prerequisites)
4. [Step-by-Step Migration](#step-by-step-migration)
5. [Code Migration Examples](#code-migration-examples)
6. [Testing and Validation](#testing-and-validation)
7. [Rollback Procedures](#rollback-procedures)
8. [Troubleshooting](#troubleshooting)

## Overview

### Why Migrate?

The legacy `internal/patch` package contained several security vulnerabilities:

1. **Path Traversal Vulnerabilities**: Unsanitized file paths allowed directory traversal attacks
2. **Injection Risks**: Insufficient input validation enabled various injection attacks
3. **Race Conditions**: Concurrent access without proper synchronization
4. **Memory Leaks**: Improper resource cleanup in error paths
5. **Weak Validation**: Inadequate input sanitization and validation

### What's New in internal/patchgen?

The `internal/patchgen` package provides:

- **Comprehensive Input Validation**: Multi-layer validation with type safety
- **Path Security**: Strict path sanitization preventing traversal attacks
- **Injection Prevention**: Pattern-based detection and blocking of injection attempts
- **Concurrency Safety**: Thread-safe operations with proper synchronization
- **Resource Management**: Automatic cleanup and resource limits
- **Audit Trail**: Complete logging of all operations
- **Performance Improvements**: Optimized algorithms and caching

## Breaking Changes

### API Changes

| Old API (internal/patch) | New API (internal/patchgen) | Notes |
|-------------------------|----------------------------|-------|
| `patch.Generate()` | `patchgen.NewPatchPackage().Generate()` | Constructor pattern |
| `patch.ApplyPatch()` | `patchgen.ApplyStructuredPatch()` | Structured approach |
| `patch.ValidatePatch()` | `patchgen.Validator.Validate()` | Dedicated validator |
| `patch.Config` | `patchgen.GeneratorConfig` | Enhanced configuration |
| Direct file writes | Sanitized file operations | Security layer added |

### Configuration Changes

#### Old Configuration (internal/patch)
```yaml
patch:
  output_dir: /tmp/patches
  allow_overwrite: true
  validation: minimal
```

#### New Configuration (internal/patchgen)
```yaml
patchgen:
  output_dir: /var/lib/nephoran/patches  # Restricted directory
  allow_overwrite: false                  # Secure default
  validation:
    level: strict                          # Enhanced validation
    max_size: 10485760                     # 10MB limit
    allowed_paths:                         # Whitelist approach
      - /var/lib/nephoran/patches
    forbidden_patterns:                    # Injection prevention
      - "../../"
      - "~/"
      - "${.*}"
```

### Behavioral Changes

1. **Stricter Validation**: All inputs undergo comprehensive validation
2. **Path Restrictions**: Only whitelisted paths are accessible
3. **Size Limits**: Maximum file and input sizes enforced
4. **Synchronous Operations**: Some async operations now synchronous for safety
5. **Error Handling**: More granular error types for better debugging

## Migration Prerequisites

### 1. System Requirements

```bash
# Check Go version (requires 1.21+)
go version

# Check available disk space (minimum 1GB)
df -h /var/lib/nephoran

# Verify permissions
ls -la /var/lib/nephoran/patches
```

### 2. Backup Current System

```bash
# Backup existing patches
tar -czf patches-backup-$(date +%Y%m%d).tar.gz /tmp/patches

# Backup configuration
cp -r /etc/nephoran /etc/nephoran.backup.$(date +%Y%m%d)

# Export current deployments
kubectl get deployments -o yaml > deployments-backup.yaml
```

### 3. Update Dependencies

```bash
# Update Go modules
go get github.com/thc1006/nephoran-intent-operator@latest

# Update vendor dependencies
go mod tidy
go mod vendor

# Verify no conflicts
go mod verify
```

## Step-by-Step Migration

### Phase 1: Preparation (Day 1)

#### Step 1.1: Analyze Current Usage
```go
// Find all imports of internal/patch
grep -r "internal/patch" --include="*.go" .

// Identify direct file operations
grep -r "os.Create\|os.OpenFile\|ioutil.WriteFile" --include="*.go" .

// Check for custom validators
grep -r "ValidatePatch\|validatePatch" --include="*.go" .
```

#### Step 1.2: Create Migration Branch
```bash
git checkout -b migration/patchgen
git push -u origin migration/patchgen
```

#### Step 1.3: Update Import Statements
```go
// Old imports
import (
    "github.com/thc1006/nephoran-intent-operator/internal/patch"
)

// New imports
import (
    "github.com/thc1006/nephoran-intent-operator/internal/patchgen"
    "github.com/thc1006/nephoran-intent-operator/internal/patchgen/generator"
)
```

### Phase 2: Code Migration (Day 2-3)

#### Step 2.1: Update Generator Code

**Before (internal/patch):**
```go
func GeneratePatch(intent *Intent) error {
    patch := &Patch{
        Target:    intent.Target,
        Namespace: intent.Namespace,
        Replicas:  intent.Replicas,
    }
    
    // Direct file write - INSECURE
    filePath := fmt.Sprintf("/tmp/patches/%s.yaml", intent.Name)
    data, _ := yaml.Marshal(patch)
    return ioutil.WriteFile(filePath, data, 0644)
}
```

**After (internal/patchgen):**
```go
func GeneratePatch(intent *patchgen.Intent) error {
    // Validate intent first
    validator := patchgen.NewValidator()
    if err := validator.ValidateIntent(intent); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // Create patch package with security controls
    pkg := patchgen.NewPatchPackage(intent, "/var/lib/nephoran/patches")
    
    // Generate with automatic sanitization
    if err := pkg.Generate(); err != nil {
        return fmt.Errorf("generation failed: %w", err)
    }
    
    // Audit log the operation
    audit.LogOperation("patch_generated", map[string]interface{}{
        "intent_id": intent.ID,
        "target":    intent.Target,
        "user":      getCurrentUser(),
    })
    
    return nil
}
```

#### Step 2.2: Update Validation Logic

**Before (internal/patch):**
```go
func ValidatePatch(patch *Patch) bool {
    return patch.Target != "" && patch.Replicas > 0
}
```

**After (internal/patchgen):**
```go
func ValidatePatch(patch *patchgen.PatchFile) error {
    validator := patchgen.NewValidator()
    
    // Comprehensive validation
    if err := validator.ValidatePatchFile(patch); err != nil {
        return fmt.Errorf("patch validation failed: %w", err)
    }
    
    // Additional security checks
    if err := validator.ValidateSecurityConstraints(patch); err != nil {
        return fmt.Errorf("security validation failed: %w", err)
    }
    
    return nil
}
```

#### Step 2.3: Update File Operations

**Before (internal/patch):**
```go
func SavePatch(name string, content []byte) error {
    // Direct concatenation - PATH TRAVERSAL RISK
    path := "/tmp/patches/" + name
    return ioutil.WriteFile(path, content, 0644)
}
```

**After (internal/patchgen):**
```go
func SavePatch(name string, content []byte) error {
    // Use sanitized generator
    sanitizer := generator.NewSanitizer()
    
    // Sanitize filename
    safeName, err := sanitizer.SanitizeFilename(name)
    if err != nil {
        return fmt.Errorf("invalid filename: %w", err)
    }
    
    // Validate content
    if err := sanitizer.ValidateContent(content); err != nil {
        return fmt.Errorf("invalid content: %w", err)
    }
    
    // Use secure write with atomic operations
    path := filepath.Join("/var/lib/nephoran/patches", safeName)
    return generator.SecureWriteFile(path, content, 0600)
}
```

### Phase 3: Configuration Migration (Day 4)

#### Step 3.1: Update Configuration Files

**config/patches.yaml:**
```yaml
# Old configuration
patches:
  enabled: true
  directory: /tmp/patches
  max_size: 0  # No limit
  
# New configuration
patchgen:
  enabled: true
  directory: /var/lib/nephoran/patches
  security:
    max_size: 10485760  # 10MB
    max_files: 1000
    allowed_extensions:
      - .yaml
      - .json
    forbidden_patterns:
      - "../"
      - "~/"
      - "${"
  validation:
    level: strict
    timeout: 30s
  audit:
    enabled: true
    retention_days: 90
```

#### Step 3.2: Update Environment Variables

```bash
# Old environment variables
export PATCH_DIR=/tmp/patches
export PATCH_VALIDATION=basic

# New environment variables
export PATCHGEN_DIR=/var/lib/nephoran/patches
export PATCHGEN_VALIDATION_LEVEL=strict
export PATCHGEN_SECURITY_ENABLED=true
export PATCHGEN_AUDIT_ENABLED=true
```

### Phase 4: Testing (Day 5-6)

#### Step 4.1: Unit Tests

```go
func TestPatchGeneration(t *testing.T) {
    // Test with valid input
    intent := &patchgen.Intent{
        ID:        "test-001",
        Target:    "nginx-deployment",
        Namespace: "default",
        Replicas:  3,
    }
    
    pkg := patchgen.NewPatchPackage(intent, t.TempDir())
    err := pkg.Generate()
    assert.NoError(t, err)
    
    // Test with malicious input
    maliciousIntent := &patchgen.Intent{
        ID:        "../../../etc/passwd",
        Target:    "'; DROP TABLE deployments; --",
        Namespace: "${SHELL}",
        Replicas:  -1,
    }
    
    pkg2 := patchgen.NewPatchPackage(maliciousIntent, t.TempDir())
    err = pkg2.Generate()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "validation failed")
}
```

#### Step 4.2: Integration Tests

```go
func TestEndToEndMigration(t *testing.T) {
    // Setup test environment
    testDir := t.TempDir()
    config := &patchgen.GeneratorConfig{
        OutputDir:        testDir,
        ValidationLevel:  patchgen.ValidationStrict,
        SecurityEnabled:  true,
        AuditEnabled:     true,
    }
    
    // Create generator
    gen := patchgen.NewGenerator(config)
    
    // Test complete workflow
    intent := createTestIntent()
    err := gen.ProcessIntent(intent)
    assert.NoError(t, err)
    
    // Verify output
    files, err := ioutil.ReadDir(testDir)
    assert.NoError(t, err)
    assert.NotEmpty(t, files)
    
    // Verify security constraints
    for _, file := range files {
        assert.NotContains(t, file.Name(), "..")
        assert.True(t, file.Mode().Perm() <= 0600)
    }
}
```

#### Step 4.3: Security Tests

```bash
# Run security scanner
gosec ./internal/patchgen/...

# Run dependency check
nancy sleuth

# Run fuzzing tests
go test -fuzz=FuzzPatchGeneration -fuzztime=10s ./internal/patchgen

# Run penetration tests
./scripts/security/pentest-patchgen.sh
```

### Phase 5: Deployment (Day 7)

#### Step 5.1: Canary Deployment

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: feature-flags
data:
  use_new_patchgen: "true"
  patchgen_percentage: "10"  # Start with 10% traffic
```

```go
// Feature flag check in code
func GeneratePatch(intent *Intent) error {
    if featureFlags.IsEnabled("use_new_patchgen") {
        return generatePatchNew(intent)  // New implementation
    }
    return generatePatchLegacy(intent)   // Old implementation
}
```

#### Step 5.2: Progressive Rollout

```bash
# Day 1: 10% traffic
kubectl patch configmap feature-flags -p '{"data":{"patchgen_percentage":"10"}}'

# Day 3: 50% traffic (after monitoring)
kubectl patch configmap feature-flags -p '{"data":{"patchgen_percentage":"50"}}'

# Day 5: 100% traffic (after validation)
kubectl patch configmap feature-flags -p '{"data":{"patchgen_percentage":"100"}}'
```

#### Step 5.3: Monitoring

```yaml
# Prometheus alerts
groups:
  - name: patchgen_migration
    rules:
      - alert: PatchGenHighErrorRate
        expr: rate(patchgen_errors_total[5m]) > 0.1
        for: 10m
        annotations:
          summary: High error rate in patchgen
          
      - alert: PatchGenValidationFailures
        expr: rate(patchgen_validation_failures_total[5m]) > 0.05
        for: 5m
        annotations:
          summary: Increased validation failures
```

## Code Migration Examples

### Example 1: Simple Patch Generation

**Legacy Code:**
```go
func CreateScalingPatch(deployment string, replicas int) error {
    patch := map[string]interface{}{
        "spec": map[string]interface{}{
            "replicas": replicas,
        },
    }
    
    data, _ := json.Marshal(patch)
    filename := fmt.Sprintf("%s-patch.json", deployment)
    return ioutil.WriteFile("/tmp/" + filename, data, 0644)
}
```

**Migrated Code:**
```go
func CreateScalingPatch(deployment string, replicas int) error {
    // Create intent with validation
    intent := &patchgen.Intent{
        Target:     deployment,
        Replicas:   replicas,
        IntentType: "scaling",
        Namespace:  "default",
    }
    
    // Validate intent
    validator := patchgen.NewValidator()
    if err := validator.ValidateIntent(intent); err != nil {
        return fmt.Errorf("invalid intent: %w", err)
    }
    
    // Generate patch with security controls
    pkg := patchgen.NewPatchPackage(intent, getSecureOutputDir())
    if err := pkg.Generate(); err != nil {
        return fmt.Errorf("patch generation failed: %w", err)
    }
    
    // Log the operation
    log.Info("Patch generated", 
        "deployment", deployment,
        "replicas", replicas,
        "package", pkg.Kptfile.Metadata.Name)
    
    return nil
}
```

### Example 2: Batch Processing

**Legacy Code:**
```go
func ProcessBatchPatches(patches []PatchRequest) {
    for _, p := range patches {
        go func(patch PatchRequest) {
            // Unsafe concurrent file writes
            GeneratePatch(patch)
        }(p)
    }
}
```

**Migrated Code:**
```go
func ProcessBatchPatches(patches []patchgen.Intent) error {
    // Use controlled concurrency
    sem := make(chan struct{}, 5) // Limit to 5 concurrent operations
    errChan := make(chan error, len(patches))
    var wg sync.WaitGroup
    
    for _, intent := range patches {
        wg.Add(1)
        sem <- struct{}{} // Acquire semaphore
        
        go func(i patchgen.Intent) {
            defer wg.Done()
            defer func() { <-sem }() // Release semaphore
            
            // Process with timeout
            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()
            
            pkg := patchgen.NewPatchPackageWithContext(ctx, &i, getSecureOutputDir())
            if err := pkg.Generate(); err != nil {
                errChan <- fmt.Errorf("failed to process %s: %w", i.ID, err)
                return
            }
        }(intent)
    }
    
    wg.Wait()
    close(errChan)
    
    // Collect errors
    var errs []error
    for err := range errChan {
        errs = append(errs, err)
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("batch processing failed: %v", errs)
    }
    
    return nil
}
```

### Example 3: Custom Validation

**Legacy Code:**
```go
func ValidateCustomPatch(patch interface{}) bool {
    // Minimal validation
    return patch != nil
}
```

**Migrated Code:**
```go
func ValidateCustomPatch(patch interface{}) error {
    // Convert to structured format
    patchFile, err := patchgen.ConvertToPatchFile(patch)
    if err != nil {
        return fmt.Errorf("invalid patch format: %w", err)
    }
    
    // Create validator with custom rules
    validator := patchgen.NewValidator()
    validator.AddCustomRule("namespace", func(v interface{}) error {
        ns, ok := v.(string)
        if !ok {
            return errors.New("namespace must be string")
        }
        if ns == "kube-system" || ns == "kube-public" {
            return errors.New("cannot patch system namespaces")
        }
        return nil
    })
    
    // Validate with all rules
    if err := validator.ValidatePatchFile(patchFile); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // Additional security validation
    if err := validator.ValidateSecurityConstraints(patchFile); err != nil {
        return fmt.Errorf("security validation failed: %w", err)
    }
    
    return nil
}
```

## Testing and Validation

### 1. Unit Test Migration

```go
// Test suite for migrated code
type PatchgenMigrationTestSuite struct {
    suite.Suite
    tempDir string
    config  *patchgen.GeneratorConfig
}

func (s *PatchgenMigrationTestSuite) SetupTest() {
    s.tempDir = s.T().TempDir()
    s.config = &patchgen.GeneratorConfig{
        OutputDir:       s.tempDir,
        ValidationLevel: patchgen.ValidationStrict,
    }
}

func (s *PatchgenMigrationTestSuite) TestSecurityValidation() {
    // Test path traversal prevention
    maliciousIntent := &patchgen.Intent{
        ID:     "../../../etc/passwd",
        Target: "test",
    }
    
    pkg := patchgen.NewPatchPackage(maliciousIntent, s.tempDir)
    err := pkg.Generate()
    s.Error(err)
    s.Contains(err.Error(), "invalid path")
}

func (s *PatchgenMigrationTestSuite) TestInjectionPrevention() {
    // Test injection prevention
    injectionIntent := &patchgen.Intent{
        Target: "'; DROP TABLE deployments; --",
    }
    
    validator := patchgen.NewValidator()
    err := validator.ValidateIntent(injectionIntent)
    s.Error(err)
    s.Contains(err.Error(), "invalid characters")
}
```

### 2. Integration Testing

```bash
#!/bin/bash
# Integration test script

# Setup test environment
export PATCHGEN_DIR=$(mktemp -d)
export PATCHGEN_VALIDATION_LEVEL=strict

# Run integration tests
go test -tags=integration ./internal/patchgen/... -v

# Verify no sensitive files accessed
if [ -f /etc/passwd.backup ]; then
    echo "ERROR: Path traversal vulnerability detected!"
    exit 1
fi

# Check file permissions
find $PATCHGEN_DIR -type f -perm /077 -exec echo "ERROR: Insecure file permissions: {}" \;

# Cleanup
rm -rf $PATCHGEN_DIR
```

### 3. Performance Testing

```go
func BenchmarkPatchGeneration(b *testing.B) {
    intent := &patchgen.Intent{
        Target:    "test-deployment",
        Namespace: "default",
        Replicas:  3,
    }
    
    tempDir := b.TempDir()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        pkg := patchgen.NewPatchPackage(intent, tempDir)
        _ = pkg.Generate()
    }
}

func BenchmarkConcurrentGeneration(b *testing.B) {
    b.RunParallel(func(pb *testing.PB) {
        tempDir := b.TempDir()
        for pb.Next() {
            intent := &patchgen.Intent{
                ID:        uuid.New().String(),
                Target:    "test-deployment",
                Namespace: "default",
                Replicas:  3,
            }
            pkg := patchgen.NewPatchPackage(intent, tempDir)
            _ = pkg.Generate()
        }
    })
}
```

## Rollback Procedures

### Immediate Rollback

```bash
#!/bin/bash
# Emergency rollback script

echo "Starting emergency rollback..."

# 1. Disable new patchgen
kubectl patch configmap feature-flags -p '{"data":{"use_new_patchgen":"false"}}'

# 2. Scale down affected deployments
kubectl scale deployment nephoran-operator --replicas=0

# 3. Restore old code
git checkout main
git pull origin main

# 4. Rebuild and deploy
make build
make deploy

# 5. Scale up with old code
kubectl scale deployment nephoran-operator --replicas=3

# 6. Verify rollback
kubectl logs -l app=nephoran-operator --tail=100

echo "Rollback completed"
```

### Gradual Rollback

```yaml
# Progressive rollback using feature flags
apiVersion: v1
kind: ConfigMap
metadata:
  name: rollback-stages
data:
  stage_1: |
    patchgen_percentage: 75
    legacy_percentage: 25
  stage_2: |
    patchgen_percentage: 50
    legacy_percentage: 50
  stage_3: |
    patchgen_percentage: 25
    legacy_percentage: 75
  stage_4: |
    patchgen_percentage: 0
    legacy_percentage: 100
```

## Troubleshooting

### Common Issues and Solutions

#### Issue 1: Permission Denied Errors
```bash
# Error: permission denied: /var/lib/nephoran/patches

# Solution: Fix directory permissions
sudo mkdir -p /var/lib/nephoran/patches
sudo chown -R nephoran:nephoran /var/lib/nephoran
sudo chmod 750 /var/lib/nephoran/patches
```

#### Issue 2: Validation Failures
```go
// Error: validation failed: invalid character in target name

// Solution: Update validation rules
validator := patchgen.NewValidator()
validator.SetValidationLevel(patchgen.ValidationPermissive) // Temporary
```

#### Issue 3: Performance Degradation
```yaml
# Symptom: Slower patch generation

# Solution: Enable caching
patchgen:
  cache:
    enabled: true
    ttl: 300s
    max_size: 100MB
```

#### Issue 4: Memory Leaks
```go
// Symptom: Increasing memory usage

// Solution: Ensure proper cleanup
defer func() {
    if err := pkg.Cleanup(); err != nil {
        log.Error(err, "cleanup failed")
    }
}()
```

### Debug Mode

```bash
# Enable debug logging
export PATCHGEN_DEBUG=true
export PATCHGEN_LOG_LEVEL=debug

# Enable trace logging for specific components
export PATCHGEN_TRACE_VALIDATION=true
export PATCHGEN_TRACE_SANITIZATION=true

# Run with debug output
./nephoran-operator --debug --trace-patchgen
```

### Health Checks

```go
// Health check endpoint for migration status
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    status := map[string]interface{}{
        "patchgen": map[string]interface{}{
            "enabled":    featureFlags.IsEnabled("use_new_patchgen"),
            "percentage": featureFlags.GetInt("patchgen_percentage"),
            "errors":     metrics.GetErrorCount("patchgen"),
            "latency":    metrics.GetP99Latency("patchgen"),
        },
        "legacy": map[string]interface{}{
            "enabled":    !featureFlags.IsEnabled("use_new_patchgen"),
            "percentage": 100 - featureFlags.GetInt("patchgen_percentage"),
            "errors":     metrics.GetErrorCount("patch"),
            "latency":    metrics.GetP99Latency("patch"),
        },
    }
    
    json.NewEncoder(w).Encode(status)
}
```

## Migration Checklist

### Pre-Migration
- [ ] Backup current system
- [ ] Review breaking changes
- [ ] Update dependencies
- [ ] Create migration branch
- [ ] Setup test environment

### During Migration
- [ ] Update import statements
- [ ] Migrate generator code
- [ ] Update validation logic
- [ ] Secure file operations
- [ ] Update configuration
- [ ] Write migration tests

### Post-Migration
- [ ] Run security scans
- [ ] Execute performance tests
- [ ] Deploy to staging
- [ ] Monitor metrics
- [ ] Progressive rollout
- [ ] Update documentation

### Validation
- [ ] All tests passing
- [ ] No security vulnerabilities
- [ ] Performance acceptable
- [ ] Audit logs working
- [ ] Rollback tested
- [ ] Documentation updated

## Support and Resources

### Documentation
- [Security Architecture](./SECURITY_ARCHITECTURE.md)
- [API Reference](../API_REFERENCE.md)
- [Best Practices](./SECURITY_BEST_PRACTICES.md)

### Contact
- Security Team: security@nephoran.io
- Migration Support: migration-support@nephoran.io
- Emergency Hotline: +1-555-SEC-HELP

### Tools and Scripts
- Migration scripts: `/scripts/migration/`
- Validation tools: `/tools/patchgen-validator/`
- Performance benchmarks: `/benchmarks/patchgen/`

---

**Document Classification**: INTERNAL USE ONLY  
**Last Updated**: 2025-08-19  
**Version**: 1.0.0  
**Migration Timeline**: 7-10 days  
**Risk Level**: Medium  
**Rollback Time**: < 5 minutes