# Go Linting Fix Templates - 2025 Best Practices

## Overview
Ready-to-apply templates for common golangci-lint issues based on 2025 best practices and Go 1.24+ requirements.

## Go 1.24 Specific Updates

### Generic Type Aliases (New in 1.24)
```go
// ✅ Good: Generic type alias
type Handler[T any] = func(T) error
type Result[T any] = chan T

// Usage
var stringHandler Handler[string] = func(s string) error { return nil }
```

### Tool Directive in go.mod (New in 1.24)
```go
// In go.mod
tool (
    github.com/golangci/golangci-lint/cmd/golangci-lint v1.64.3
    github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen v2.1.0
)
```

## Package Documentation Templates

### Missing Package Comments (ST1000, package-comments)
```go
// ❌ Bad: No package comment
package clients

// ✅ Good: Package comment explaining purpose
// Package clients provides mTLS-enabled client factories for O-RAN network services.
// It supports secure communication with LLM processors, RAG services, Git repositories,
// and monitoring systems in Kubernetes-based telecommunications environments.
package clients
```

### Package Comment Best Practices
```go
// Package <name> provides <what it does>.
//
// This package is designed for <use case> and includes:
//   - <feature 1>
//   - <feature 2>
//   - <feature 3>
//
// Example usage:
//
//	client := NewClient(config)
//	result, err := client.Process(data)
//	if err != nil {
//		return fmt.Errorf("processing failed: %w", err)
//	}
package packagename
```

## Exported Function/Type Documentation

### Missing Exported Function Documentation
```go
// ❌ Bad: No documentation
func NewMTLSClientFactory(config *ClientFactoryConfig) (*MTLSClientFactory, error) {

// ✅ Good: Complete documentation
// NewMTLSClientFactory creates a new mTLS client factory for secure service communication.
// It initializes the factory with CA management and identity provisioning capabilities.
//
// The factory supports auto-provisioning of service identities and certificate rotation
// for O-RAN network services in Kubernetes environments.
//
// Returns an error if the configuration is invalid or required dependencies are missing.
func NewMTLSClientFactory(config *ClientFactoryConfig) (*MTLSClientFactory, error) {
```

### Exported Type Documentation
```go
// ❌ Bad: No documentation
type MTLSClientFactory struct {

// ✅ Good: Complete type documentation
// MTLSClientFactory creates and manages mTLS-enabled service clients for secure
// communication in O-RAN telecommunications environments.
//
// The factory provides thread-safe client creation with automatic certificate
// management, service identity provisioning, and connection pooling.
//
// Supported service types include LLM processors, RAG services, Git clients,
// and monitoring systems with full mTLS authentication.
type MTLSClientFactory struct {
```

### Exported Constant Documentation
```go
// ❌ Bad: No documentation
const ServiceTypeLLM ServiceType = "llm"

// ✅ Good: Documented constants
const (
	// ServiceTypeLLM represents LLM processor services for natural language processing.
	ServiceTypeLLM ServiceType = "llm"
	
	// ServiceTypeRAG represents Retrieval-Augmented Generation services.
	ServiceTypeRAG ServiceType = "rag"
	
	// ServiceTypeGit represents Git repository clients for GitOps operations.
	ServiceTypeGit ServiceType = "git"
)
```

## Error Handling Patterns (2025 Best Practices)

### Error Wrapping with %w (errorlint)
```go
// ❌ Bad: Error formatting without wrapping
return fmt.Errorf("failed to create client: %s", err.Error())

// ✅ Good: Error wrapping with %w
return fmt.Errorf("failed to create mTLS HTTP client for %s service: %w", serviceType, err)
```

### Context-Aware Error Handling
```go
// ✅ Good: Context with timeout and cancellation
func (f *MTLSClientFactory) CreateClientWithTimeout(ctx context.Context, serviceType ServiceType) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	select {
	case <-ctx.Done():
		return fmt.Errorf("client creation timed out for service %s: %w", serviceType, ctx.Err())
	default:
		// Create client...
	}
}
```

### Custom Error Types (2025 Pattern)
```go
// ✅ Good: Custom error types with wrapping
type ClientFactoryError struct {
	ServiceType ServiceType
	Operation   string
	Err         error
}

func (e *ClientFactoryError) Error() string {
	return fmt.Sprintf("client factory error for %s during %s: %v", e.ServiceType, e.Operation, e.Err)
}

func (e *ClientFactoryError) Unwrap() error {
	return e.Err
}

// Usage
return &ClientFactoryError{
	ServiceType: serviceType,
	Operation:   "mTLS setup",
	Err:         err,
}
```

## Unused Variable Elimination

### Unused Function Parameters
```go
// ❌ Bad: Unused parameter
func (f *MTLSClientFactory) processRequest(ctx context.Context, data []byte) error {
	// data parameter not used
	return f.doSomething(ctx)
}

// ✅ Good: Use underscore for intentionally unused
func (f *MTLSClientFactory) processRequest(ctx context.Context, _ []byte) error {
	return f.doSomething(ctx)
}

// ✅ Better: Remove unused parameter if possible
func (f *MTLSClientFactory) processRequest(ctx context.Context) error {
	return f.doSomething(ctx)
}
```

### Unused Variables in Loops
```go
// ❌ Bad: Unused loop variable
for i, item := range items {
	fmt.Println(item)
}

// ✅ Good: Use underscore for unused index
for _, item := range items {
	fmt.Println(item)
}
```

### Unused Return Values
```go
// ❌ Bad: Ignoring error return
client.Close()

// ✅ Good: Handle or explicitly ignore
if err := client.Close(); err != nil {
	log.Printf("failed to close client: %v", err)
}

// ✅ Good: Explicit ignore with comment
_ = client.Close() // ignore close error in cleanup
```

## Ineffective Assignment Fixes

### Assignment to Blank Identifier
```go
// ❌ Bad: Ineffective assignment
_ = someFunction()

// ✅ Good: Handle return value or add comment
if err := someFunction(); err != nil {
	return fmt.Errorf("operation failed: %w", err)
}

// ✅ Good: Explicit ignore with reason
_ = someFunction() // result not needed, errors handled internally
```

### Self-Assignment Detection
```go
// ❌ Bad: Self-assignment (ineffective)
config.Timeout = config.Timeout

// ✅ Good: Remove ineffective assignment
// (Simply remove the line if it's truly ineffective)
```

## Modern Go Patterns for 2025 Linters

### Context Usage Patterns (contextcheck)
```go
// ✅ Good: Context-first parameter pattern
func (f *MTLSClientFactory) CreateClient(ctx context.Context, serviceType ServiceType, options ...Option) (*Client, error) {
	// Check context before expensive operations
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context canceled before client creation: %w", err)
	}
	
	// Use context in HTTP requests
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Pass context down the call chain
	return f.createClientInternal(ctx, serviceType, options...)
}
```

### Proper Defer Usage (gocritic)
```go
// ✅ Good: Defer with error handling
func (f *MTLSClientFactory) processWithCleanup() (err error) {
	resource, err := acquireResource()
	if err != nil {
		return fmt.Errorf("failed to acquire resource: %w", err)
	}
	
	defer func() {
		if closeErr := resource.Close(); closeErr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close resource: %w", closeErr)
			} else {
				// Log additional error, don't override original
				log.Printf("additional error closing resource: %v", closeErr)
			}
		}
	}()
	
	// Use resource...
	return nil
}
```

### Interface Compliance Patterns
```go
// ✅ Good: Interface compliance checks
var (
	_ shared.ClientInterface    = (*MTLSLLMClient)(nil)
	_ io.Closer                = (*MTLSClientFactory)(nil)
	_ fmt.Stringer             = (*ClientStats)(nil)
)

// ✅ Good: Interface with context
type ClientInterface interface {
	// ProcessRequest processes a request with context for cancellation.
	ProcessRequest(ctx context.Context, req *Request) (*Response, error)
	
	// Close cleanly shuts down the client and releases resources.
	Close() error
}
```

### Concurrent-Safe Patterns (govet, race detector)
```go
// ✅ Good: Thread-safe access with RWMutex
type ThreadSafeCache struct {
	mu    sync.RWMutex
	items map[string]interface{}
}

func (c *ThreadSafeCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, exists := c.items[key]
	return item, exists
}

func (c *ThreadSafeCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.items == nil {
		c.items = make(map[string]interface{})
	}
	c.items[key] = value
}
```

## Go 1.24+ Specific Patterns

### Range Over Integers (intrange linter)
```go
// ✅ Good: Use new range syntax for integers (Go 1.22+)
for i := range 10 {
	fmt.Printf("iteration %d\n", i)
}

// ✅ Good: Range over channels
for value := range valueChannel {
	process(value)
}
```

### Copy Loop Variable (copyloopvar - automatic in Go 1.22+)
```go
// ✅ Good: No longer need to copy loop variables in Go 1.22+
var handlers []func()
for _, item := range items {
	handlers = append(handlers, func() {
		process(item) // item is automatically copied per iteration
	})
}
```

## Security Patterns (gosec)

### Secure Random Number Generation
```go
// ❌ Bad: Weak random number generator
import "math/rand"
token := rand.Int63()

// ✅ Good: Cryptographically secure random
import "crypto/rand"
import "math/big"

func generateSecureToken() (int64, error) {
	max := big.NewInt(1 << 62)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, fmt.Errorf("failed to generate secure random number: %w", err)
	}
	return n.Int64(), nil
}
```

### Secure File Permissions
```go
// ❌ Bad: Overly permissive file creation
file, err := os.Create("/tmp/config.yaml")

// ✅ Good: Secure file permissions
file, err := os.OpenFile("/tmp/config.yaml", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
if err != nil {
	return fmt.Errorf("failed to create config file: %w", err)
}
defer file.Close()
```

## Performance Patterns (prealloc, gocritic)

### Slice Preallocation
```go
// ❌ Bad: Growing slice without preallocation
var results []Result
for _, item := range items {
	if item.IsValid() {
		results = append(results, process(item))
	}
}

// ✅ Good: Preallocate slice
results := make([]Result, 0, len(items)) // preallocate capacity
for _, item := range items {
	if item.IsValid() {
		results = append(results, process(item))
	}
}
```

### String Builder for Concatenation
```go
// ❌ Bad: String concatenation in loop
var result string
for _, item := range items {
	result += item + ","
}

// ✅ Good: Use strings.Builder
var builder strings.Builder
builder.Grow(len(items) * 10) // estimate capacity
for i, item := range items {
	if i > 0 {
		builder.WriteString(",")
	}
	builder.WriteString(item)
}
result := builder.String()
```

## Quick Fix Commands

### Common linter fixes
```bash
# Fix imports and formatting
goimports -w .
gofumpt -w .

# Fix specific linter issues
golangci-lint run --fix

# Run specific linters
golangci-lint run --enable-only=errcheck,gosec,govet,staticcheck

# Check for unused variables
golangci-lint run --enable-only=unused,ineffassign

# Verify documentation
golangci-lint run --enable-only=revive,stylecheck,godot
```

## IDE Integration Snippets

### VS Code snippets for quick fixes
```json
{
  "Package Comment": {
    "prefix": "pkgdoc",
    "body": [
      "// Package $1 provides $2.",
      "//",
      "// This package is designed for $3 and includes:",
      "//   - $4",
      "//   - $5",
      "//",
      "// Example usage:",
      "//",
      "//\t$6",
      "package $1"
    ]
  },
  
  "Function Documentation": {
    "prefix": "funcdoc",
    "body": [
      "// $1 $2.",
      "//",
      "// $3",
      "//",
      "// Returns an error if $4."
    ]
  },
  
  "Error Wrapping": {
    "prefix": "errwrap",
    "body": [
      "return fmt.Errorf(\"$1: %w\", $2)"
    ]
  }
}
```

## Summary

These templates cover the most common linting issues in 2025 with Go 1.24+ compatibility:

1. **Package documentation** - Required by stylecheck and revive
2. **Exported type/function docs** - Required by revive and godot
3. **Error handling** - Enhanced with %w wrapping and context awareness
4. **Unused variables** - Clean elimination patterns
5. **Ineffective assignments** - Detection and removal
6. **Modern Go patterns** - Context-first, interfaces, concurrency
7. **Security patterns** - Secure random, file permissions
8. **Performance patterns** - Preallocation, string building

Apply these patterns consistently to maintain high code quality and pass modern linter requirements.