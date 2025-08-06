# Development Tools

This directory contains development and testing utilities for the Nephoran Intent Operator project.

## Files

### test_import_cycle.go

A development utility that tests for import cycles between packages in the project. This tool is particularly useful during package refactoring to ensure that circular dependencies haven't been introduced.

**Purpose:**
- Verifies that key packages (config, security, interfaces) can be imported together without causing import cycles
- Tests that interface types can be referenced properly
- Validates that package instantiation works correctly

**Packages tested:**
- `pkg/config` - Configuration management
- `pkg/security` - Security utilities and audit logging  
- `pkg/interfaces` - Common interfaces and types

**How to run:**
```bash
go run tools/test_import_cycle.go
```

**Expected output:**
```
Testing import cycle resolution...
Default config created: true
Audit logger created: true
Import cycle test completed successfully!
```

This test should be run after any significant package restructuring or when adding new cross-package dependencies to ensure the project maintains a clean dependency graph.