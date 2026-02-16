# Patchgen Troubleshooting Guide

## Overview

This guide provides comprehensive troubleshooting information for common issues encountered when migrating to or using the `internal/patchgen` module. It includes diagnostic procedures, resolution steps, and preventive measures.

## Table of Contents

- [Quick Diagnosis](#quick-diagnosis)
- [Common Migration Issues](#common-migration-issues)
- [Validation Errors](#validation-errors)
- [File System Issues](#file-system-issues)
- [Security-Related Issues](#security-related-issues)
- [Performance Issues](#performance-issues)
- [Integration Problems](#integration-problems)
- [Debugging Techniques](#debugging-techniques)
- [Error Reference](#error-reference)
- [Health Checks](#health-checks)
- [Support Escalation](#support-escalation)

## Quick Diagnosis

### Diagnostic Script

Use this script to quickly diagnose common issues:

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "runtime"
    
    "github.com/go-logr/zapr"
    "go.uber.org/zap"
    
    "github.com/nephoran/internal/patchgen"
)

func diagnose() {
    fmt.Println("=== Patchgen Diagnostic Report ===\n")
    
    // System Information
    fmt.Printf("Go Version: %s\n", runtime.Version())
    fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
    fmt.Printf("NumCPU: %d\n\n", runtime.NumCPU())
    
    // Check module availability
    fmt.Println("Module Check:")
    if _, err := os.Stat("go.mod"); err == nil {
        fmt.Println("✓ go.mod found")
    } else {
        fmt.Println("✗ go.mod not found")
    }
    
    // Test validator creation
    fmt.Println("\nValidator Check:")
    zapLog, _ := zap.NewProduction()
    logger := zapr.NewLogger(zapLog)
    
    validator, err := patchgen.NewValidator(logger)
    if err != nil {
        fmt.Printf("✗ Validator creation failed: %v\n", err)
    } else {
        fmt.Println("✓ Validator created successfully")
        
        // Test validation
        testIntent := []byte(`{
            "intent_type": "scaling",
            "target": "test-app",
            "namespace": "default",
            "replicas": 3
        }`)
        
        if _, err := validator.ValidateIntent(testIntent); err != nil {
            fmt.Printf("✗ Validation failed: %v\n", err)
        } else {
            fmt.Println("✓ Validation successful")
        }
    }
    
    // Test file system
    fmt.Println("\nFile System Check:")
    tempDir := os.TempDir()
    testDir := fmt.Sprintf("%s/patchgen-test-%d", tempDir, os.Getpid())
    
    if err := os.MkdirAll(testDir, 0755); err != nil {
        fmt.Printf("✗ Cannot create directory: %v\n", err)
    } else {
        fmt.Println("✓ Directory creation successful")
        defer os.RemoveAll(testDir)
        
        // Test package generation
        intent := &patchgen.Intent{
            IntentType: "scaling",
            Target:     "test",
            Namespace:  "default",
            Replicas:   1,
        }
        
        pkg := patchgen.NewPatchPackage(intent, testDir)
        if err := pkg.Generate(); err != nil {
            fmt.Printf("✗ Package generation failed: %v\n", err)
        } else {
            fmt.Println("✓ Package generation successful")
            fmt.Printf("  Package path: %s\n", pkg.GetPackagePath())
        }
    }
    
    fmt.Println("\n=== End Diagnostic Report ===")
}

func main() {
    diagnose()
}
```

## Common Migration Issues

### Issue 1: Import Path Not Found

**Symptom:**
```
cannot find module providing package github.com/nephoran/internal/patch
```

**Cause:** Old import path still being used.

**Solution:**
```bash
# Update all import statements
find . -type f -name "*.go" -exec sed -i 's/internal\/patch/internal\/patchgen/g' {} \;

# Update go.mod
go mod tidy
```

### Issue 2: Type Mismatch Errors

**Symptom:**
```
cannot use generator (type *patch.Generator) as type *patchgen.PatchPackage
```

**Cause:** Types have changed between versions.

**Solution:**
```go
// Old code
var generator *patch.Generator
generator = patch.NewGenerator(intent, outputDir)

// New code
var pkg *patchgen.PatchPackage
pkg = patchgen.NewPatchPackage(intent, outputDir)
```

### Issue 3: Missing Validation

**Symptom:**
```
panic: validation required but not performed
```

**Cause:** New module requires explicit validation.

**Solution:**
```go
// Add validation before processing
validator, err := patchgen.NewValidator(logger)
if err != nil {
    return err
}

intent, err := validator.ValidateIntent(intentData)
if err != nil {
    return fmt.Errorf("validation failed: %w", err)
}

// Now safe to use intent
pkg := patchgen.NewPatchPackage(intent, outputDir)
```

### Issue 4: Method Not Found

**Symptom:**
```
pkg.GetPackagePath undefined (type *PatchPackage has no field or method GetPackagePath)
```

**Cause:** Using old version of the module.

**Solution:**
```bash
# Update to latest version
go get -u github.com/nephoran/internal/patchgen@latest
go mod tidy
```

## Validation Errors

### Schema Validation Failed

**Symptom:**
```
schema validation failed: value does not match pattern "^[a-z0-9](?:[-a-z0-9]*[a-z0-9])?$"
```

**Diagnosis:**
```go
func diagnoseValidationError(intentData []byte) {
    var raw map[string]interface{}
    json.Unmarshal(intentData, &raw)
    
    // Check each field
    if target, ok := raw["target"].(string); ok {
        if !isValidKubernetesName(target) {
            fmt.Printf("Invalid target name: %s\n", target)
            fmt.Println("Must be lowercase alphanumeric with hyphens")
        }
    }
    
    if namespace, ok := raw["namespace"].(string); ok {
        if len(namespace) > 63 {
            fmt.Printf("Namespace too long: %d chars (max 63)\n", len(namespace))
        }
    }
    
    if replicas, ok := raw["replicas"].(float64); ok {
        if replicas < 0 || replicas > 100 {
            fmt.Printf("Replicas out of range: %.0f (must be 0-100)\n", replicas)
        }
    }
}
```

**Common Fixes:**

1. **Invalid Characters in Names:**
   ```go
   // Bad
   intent.Target = "MyApp_v2"
   
   // Good
   intent.Target = "myapp-v2"
   ```

2. **Missing Required Fields:**
   ```go
   // Bad
   intent := &patchgen.Intent{
       Target: "app",
       Replicas: 3,
   }
   
   // Good
   intent := &patchgen.Intent{
       IntentType: "scaling",  // Required
       Target:     "app",
       Namespace:  "default",   // Required
       Replicas:   3,
   }
   ```

3. **Wrong Intent Type:**
   ```go
   // Bad
   intent.IntentType = "scale"
   
   // Good
   intent.IntentType = "scaling"
   ```

### JSON Parsing Errors

**Symptom:**
```
invalid JSON: unexpected end of JSON input
```

**Diagnosis:**
```go
func validateJSON(data []byte) error {
    var js json.RawMessage
    if err := json.Unmarshal(data, &js); err != nil {
        // Find the error position
        if syntaxErr, ok := err.(*json.SyntaxError); ok {
            start := max(0, syntaxErr.Offset-20)
            end := min(len(data), syntaxErr.Offset+20)
            
            fmt.Printf("JSON error at position %d:\n", syntaxErr.Offset)
            fmt.Printf("Context: ...%s...\n", string(data[start:end]))
            fmt.Printf("         %s^\n", strings.Repeat(" ", int(syntaxErr.Offset-start)))
        }
        return err
    }
    return nil
}
```

**Common Fixes:**

1. **Missing Commas:**
   ```json
   // Bad
   {
       "intent_type": "scaling"
       "target": "app"
   }
   
   // Good
   {
       "intent_type": "scaling",
       "target": "app"
   }
   ```

2. **Trailing Commas:**
   ```json
   // Bad
   {
       "intent_type": "scaling",
       "target": "app",
   }
   
   // Good
   {
       "intent_type": "scaling",
       "target": "app"
   }
   ```

## File System Issues

### Directory Does Not Exist

**Symptom:**
```
output directory /var/lib/patches does not exist
```

**Solution:**
```go
func ensureDirectory(path string) error {
    // Check if exists
    if info, err := os.Stat(path); err == nil {
        if !info.IsDir() {
            return fmt.Errorf("%s exists but is not a directory", path)
        }
        return nil
    }
    
    // Create with proper permissions
    if err := os.MkdirAll(path, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }
    
    // Verify creation
    if _, err := os.Stat(path); err != nil {
        return fmt.Errorf("directory created but not accessible: %w", err)
    }
    
    return nil
}
```

### Permission Denied

**Symptom:**
```
failed to create package directory: permission denied
```

**Diagnosis:**
```bash
# Check directory permissions
ls -la /var/lib/patches

# Check current user
whoami

# Check effective permissions
namei -l /var/lib/patches
```

**Solutions:**

1. **Change Directory Ownership:**
   ```bash
   sudo chown -R $(whoami):$(whoami) /var/lib/patches
   ```

2. **Use Different Directory:**
   ```go
   // Use user-writable directory
   homeDir, _ := os.UserHomeDir()
   outputDir := filepath.Join(homeDir, ".nephoran", "patches")
   os.MkdirAll(outputDir, 0755)
   ```

3. **Run with Appropriate Permissions:**
   ```bash
   # If running in container
   docker run --user $(id -u):$(id -g) -v /var/lib/patches:/patches ...
   ```

### Disk Space Issues

**Symptom:**
```
failed to write file: no space left on device
```

**Diagnosis:**
```go
func checkDiskSpace(path string) error {
    var stat syscall.Statfs_t
    if err := syscall.Statfs(path, &stat); err != nil {
        return err
    }
    
    available := stat.Bavail * uint64(stat.Bsize)
    required := uint64(1024 * 1024) // 1MB minimum
    
    if available < required {
        return fmt.Errorf("insufficient disk space: %d bytes available, need %d",
            available, required)
    }
    
    return nil
}
```

## Security-Related Issues

### Path Traversal Detected

**Symptom:**
```
path traversal attempt detected: contains '..'
```

**Cause:** Security validation preventing directory escape.

**Solution:**
```go
// Clean and validate paths
func sanitizePath(base, userPath string) (string, error) {
    // Remove any path traversal attempts
    cleaned := filepath.Clean(userPath)
    
    // Ensure it's relative and safe
    if filepath.IsAbs(cleaned) {
        cleaned = filepath.Base(cleaned)
    }
    
    // Join with base
    final := filepath.Join(base, cleaned)
    
    // Verify still within base
    absBase, _ := filepath.Abs(base)
    absFinal, _ := filepath.Abs(final)
    
    if !strings.HasPrefix(absFinal, absBase) {
        return "", fmt.Errorf("path escapes base directory")
    }
    
    return final, nil
}
```

### Invalid Kubernetes Names

**Symptom:**
```
validation failed: target does not match required pattern
```

**Solution:**
```go
func sanitizeKubernetesName(name string) string {
    // Convert to lowercase
    name = strings.ToLower(name)
    
    // Replace invalid characters with hyphens
    name = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "-")
    
    // Remove leading/trailing hyphens
    name = strings.Trim(name, "-")
    
    // Ensure not empty
    if name == "" {
        name = "unnamed"
    }
    
    // Truncate if too long
    if len(name) > 253 {
        name = name[:253]
    }
    
    return name
}
```

## Performance Issues

### Slow Package Generation

**Symptom:** Package generation takes > 1 second

**Diagnosis:**
```go
func profileGeneration(intent *patchgen.Intent, outputDir string) {
    start := time.Now()
    
    // Measure validation
    validationStart := time.Now()
    validator, _ := patchgen.NewValidator(logger)
    validationTime := time.Since(validationStart)
    
    // Measure generation
    generationStart := time.Now()
    pkg := patchgen.NewPatchPackage(intent, outputDir)
    err := pkg.Generate()
    generationTime := time.Since(generationStart)
    
    fmt.Printf("Total time: %v\n", time.Since(start))
    fmt.Printf("  Validation: %v\n", validationTime)
    fmt.Printf("  Generation: %v\n", generationTime)
    
    if generationTime > time.Second {
        fmt.Println("Possible causes:")
        fmt.Println("- Slow disk I/O")
        fmt.Println("- Network-mounted filesystem")
        fmt.Println("- Antivirus scanning")
    }
}
```

**Solutions:**

1. **Use Local SSD:**
   ```go
   // Use fast local storage
   outputDir := "/tmp/patches" // RAM disk on some systems
   ```

2. **Batch Operations:**
   ```go
   func batchGenerate(intents []*patchgen.Intent, outputDir string) {
       // Process in parallel
       var wg sync.WaitGroup
       semaphore := make(chan struct{}, runtime.NumCPU())
       
       for _, intent := range intents {
           wg.Add(1)
           semaphore <- struct{}{}
           
           go func(i *patchgen.Intent) {
               defer wg.Done()
               defer func() { <-semaphore }()
               
               pkg := patchgen.NewPatchPackage(i, outputDir)
               pkg.Generate()
           }(intent)
       }
       
       wg.Wait()
   }
   ```

### Memory Usage Issues

**Symptom:** High memory consumption

**Diagnosis:**
```go
func monitorMemory() {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    fmt.Printf("Alloc = %v MB\n", m.Alloc / 1024 / 1024)
    fmt.Printf("TotalAlloc = %v MB\n", m.TotalAlloc / 1024 / 1024)
    fmt.Printf("Sys = %v MB\n", m.Sys / 1024 / 1024)
    fmt.Printf("NumGC = %v\n", m.NumGC)
}
```

**Solution:**
```go
// Force garbage collection after batch processing
func processWithGC(intents []*patchgen.Intent) {
    for i, intent := range intents {
        pkg := patchgen.NewPatchPackage(intent, "/tmp/patches")
        pkg.Generate()
        
        // Periodic GC
        if i%100 == 0 {
            runtime.GC()
        }
    }
}
```

## Integration Problems

### Controller Runtime Integration

**Problem:** Integration with controller-runtime fails

**Solution:**
```go
import (
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := log.FromContext(ctx)
    
    // Create validator with controller logger
    validator, err := patchgen.NewValidator(logger)
    if err != nil {
        return ctrl.Result{}, err
    }
    
    // Process intent
    intent, err := r.getIntentFromCR(ctx, req)
    if err != nil {
        return ctrl.Result{}, err
    }
    
    // Generate package
    pkg := patchgen.NewPatchPackage(intent, r.OutputDir)
    if err := pkg.Generate(); err != nil {
        logger.Error(err, "Failed to generate package")
        return ctrl.Result{RequeueAfter: time.Minute}, err
    }
    
    return ctrl.Result{}, nil
}
```

### HTTP API Integration

**Problem:** REST API returns errors

**Solution:**
```go
func handleIntentAPI(w http.ResponseWriter, r *http.Request) {
    // Set proper headers
    w.Header().Set("Content-Type", "application/json")
    
    // Read with size limit
    r.Body = http.MaxBytesReader(w, r.Body, 1048576) // 1MB limit
    
    body, err := io.ReadAll(r.Body)
    if err != nil {
        respondError(w, "Request too large", http.StatusRequestEntityTooLarge)
        return
    }
    
    // Validate
    validator, _ := patchgen.NewValidator(logger)
    intent, err := validator.ValidateIntent(body)
    if err != nil {
        respondError(w, fmt.Sprintf("Validation failed: %v", err), 
            http.StatusBadRequest)
        return
    }
    
    // Generate with timeout
    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()
    
    done := make(chan error, 1)
    go func() {
        pkg := patchgen.NewPatchPackage(intent, "/tmp/patches")
        done <- pkg.Generate()
    }()
    
    select {
    case err := <-done:
        if err != nil {
            respondError(w, "Generation failed", http.StatusInternalServerError)
            return
        }
        respondSuccess(w, "Package generated successfully")
        
    case <-ctx.Done():
        respondError(w, "Request timeout", http.StatusGatewayTimeout)
    }
}
```

## Debugging Techniques

### Enable Debug Logging

```go
import (
    "github.com/go-logr/zapr"
    "go.uber.org/zap"
)

func setupDebugLogger() logr.Logger {
    config := zap.NewDevelopmentConfig()
    config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
    config.EncoderConfig.TimeKey = "timestamp"
    config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    
    zapLog, _ := config.Build()
    return zapr.NewLogger(zapLog)
}
```

### Trace Execution

```go
func traceExecution(intent *patchgen.Intent, outputDir string) {
    trace := func(stage string) func() {
        start := time.Now()
        fmt.Printf("[%s] Starting %s\n", start.Format(time.RFC3339), stage)
        return func() {
            fmt.Printf("[%s] Completed %s (took %v)\n", 
                time.Now().Format(time.RFC3339), stage, time.Since(start))
        }
    }
    
    // Trace validation
    done := trace("validation")
    validator, _ := patchgen.NewValidator(logger)
    validator.ValidateIntent([]byte("..."))
    done()
    
    // Trace generation
    done = trace("package generation")
    pkg := patchgen.NewPatchPackage(intent, outputDir)
    pkg.Generate()
    done()
}
```

### Memory Profiling

```go
import (
    "runtime/pprof"
)

func profileMemory() {
    f, _ := os.Create("mem.prof")
    defer f.Close()
    
    runtime.GC()
    pprof.WriteHeapProfile(f)
    
    // Analyze with: go tool pprof mem.prof
}
```

## Error Reference

### Error Code Table

| Error Message | Cause | Solution |
|--------------|-------|----------|
| `schema validation failed` | Invalid JSON structure | Check schema requirements |
| `invalid JSON` | Malformed JSON | Validate JSON syntax |
| `path traversal detected` | Security violation | Use clean paths |
| `output directory does not exist` | Missing directory | Create directory first |
| `permission denied` | Insufficient privileges | Check file permissions |
| `intent_type must be 'scaling'` | Wrong intent type | Use "scaling" only |
| `replicas out of range` | Invalid replica count | Use 0-100 range |
| `target does not match pattern` | Invalid K8s name | Use lowercase alphanumeric |
| `namespace too long` | Exceeds 63 chars | Shorten namespace name |
| `failed to marshal` | Serialization error | Check data types |

### Error Recovery

```go
func recoverFromError(err error) error {
    switch {
    case strings.Contains(err.Error(), "permission denied"):
        // Try alternative directory
        return useAlternativeDirectory()
        
    case strings.Contains(err.Error(), "does not exist"):
        // Create missing directory
        return createRequiredDirectory()
        
    case strings.Contains(err.Error(), "validation failed"):
        // Log validation details
        return logValidationDetails(err)
        
    default:
        // Generic recovery
        return fmt.Errorf("unrecoverable error: %w", err)
    }
}
```

## Health Checks

### System Health Check

```go
func healthCheck() HealthStatus {
    status := HealthStatus{
        Healthy: true,
        Checks:  make(map[string]bool),
    }
    
    // Check module
    if _, err := patchgen.NewValidator(logger); err != nil {
        status.Checks["validator"] = false
        status.Healthy = false
    } else {
        status.Checks["validator"] = true
    }
    
    // Check file system
    if err := checkFileSystem(); err != nil {
        status.Checks["filesystem"] = false
        status.Healthy = false
    } else {
        status.Checks["filesystem"] = true
    }
    
    // Check memory
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    if m.Alloc > 500*1024*1024 { // 500MB threshold
        status.Checks["memory"] = false
        status.Warning = "High memory usage"
    } else {
        status.Checks["memory"] = true
    }
    
    return status
}
```

### Readiness Probe

```go
func readinessProbe(w http.ResponseWriter, r *http.Request) {
    // Test package generation
    intent := &patchgen.Intent{
        IntentType: "scaling",
        Target:     "readiness-test",
        Namespace:  "default",
        Replicas:   1,
    }
    
    pkg := patchgen.NewPatchPackage(intent, "/tmp")
    if err := pkg.Generate(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "not ready",
            "error":  err.Error(),
        })
        return
    }
    
    // Clean up test package
    os.RemoveAll(pkg.GetPackagePath())
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "ready",
    })
}
```

## Support Escalation

### Level 1: Self-Service

1. Run diagnostic script
2. Check error reference table
3. Review troubleshooting steps
4. Search known issues

### Level 2: Community Support

- GitHub Issues: https://github.com/nephoran/issues
- Community Forum: https://forum.nephoran.io
- Stack Overflow: Tag with `nephoran-patchgen`

### Level 3: Professional Support

For critical production issues:

1. **Collect Diagnostic Information:**
   ```bash
   # Run diagnostic script
   go run diagnose.go > diagnostic-report.txt
   
   # Collect logs
   kubectl logs -n nephoran-system deployment/nephoran-operator > operator.log
   
   # System information
   uname -a > system.txt
   go version >> system.txt
   ```

2. **Create Support Ticket:**
   - Email: support@nephoran.io
   - Include diagnostic report
   - Describe the issue and impact
   - Provide reproduction steps

3. **Priority Levels:**
   - **P1 (Critical)**: Production down, no workaround
   - **P2 (High)**: Production impacted, workaround exists
   - **P3 (Medium)**: Non-production issue
   - **P4 (Low)**: Questions or enhancements

### Security Issues

For security vulnerabilities:
- Email: security@nephoran.io
- Use PGP encryption if possible
- Follow responsible disclosure policy

## Appendix: Quick Reference

### Environment Variables

```bash
# Enable debug logging
export PATCHGEN_DEBUG=true

# Set custom output directory
export PATCHGEN_OUTPUT_DIR=/custom/path

# Set validation timeout
export PATCHGEN_VALIDATION_TIMEOUT=30s

# Enable strict mode
export PATCHGEN_STRICT_MODE=true
```

### Common Commands

```bash
# Test validation
echo '{"intent_type":"scaling","target":"test","namespace":"default","replicas":3}' | \
    go run cmd/validate/main.go

# Generate package
go run cmd/generate/main.go -intent intent.json -output /tmp/patches

# Run tests
go test -v ./internal/patchgen/...

# Benchmark
go test -bench=. -benchmem ./internal/patchgen/...

# Check coverage
go test -cover ./internal/patchgen/...
```

### Useful Regex Patterns

```go
// Kubernetes name validation
var k8sNameRegex = regexp.MustCompile(`^[a-z0-9](?:[-a-z0-9]*[a-z0-9])?$`)

// Correlation ID validation
var correlationIDRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)

// Semantic version validation
var semverRegex = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?$`)
```

## Conclusion

This troubleshooting guide covers the most common issues encountered with the patchgen module. For issues not covered here, please refer to the community support channels or contact professional support for assistance.

Remember to always:
1. Run diagnostics first
2. Check logs for detailed error messages
3. Verify system requirements
4. Test in a non-production environment first

Keep this guide updated as new issues and solutions are discovered.
