# API Version Migration Guide - Nephoran Intent Operator

**Migration Version:** v1alpha1 → v1  
**Last Updated:** July 29, 2025  
**Status:** Complete  
**Backward Compatibility:** Maintained

## Overview

The Nephoran Intent Operator has successfully migrated all Custom Resource Definitions (CRDs) from `v1alpha1` to `v1` API version. This migration provides improved stability, better validation, and production readiness while maintaining backward compatibility.

## Migration Summary

### Migrated APIs

| Resource | Old Version | New Version | Status |
|----------|-------------|-------------|--------|
| NetworkIntent | `nephoran.com/v1alpha1` | `nephoran.com/v1` | ✅ Complete |
| E2NodeSet | `nephoran.com/v1alpha1` | `nephoran.com/v1` | ✅ Complete |
| ManagedElement | `nephoran.com/v1alpha1` | `nephoran.com/v1` | ✅ Complete |

### Schema Changes

#### NetworkIntent API Changes
```yaml
# OLD (v1alpha1)
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: example-intent
spec:
  intent: "Deploy AMF with 3 replicas"
  priority: "high"

# NEW (v1) - Enhanced validation and structure
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: example-intent
spec:
  description: "Deploy AMF with 3 replicas"  # Renamed from 'intent'
  priority: "high"
  parameters: {}  # New structured parameters field
status:
  phase: "Pending"  # Enhanced status tracking
  conditions: []    # Detailed condition reporting
```

#### E2NodeSet API Changes
```yaml
# OLD (v1alpha1)
apiVersion: nephoran.com/v1alpha1
kind: E2NodeSet
metadata:
  name: simulated-gnbs
spec:
  replicas: 3

# NEW (v1) - Enhanced with better validation
apiVersion: nephoran.com/v1
kind: E2NodeSet
metadata:
  name: simulated-gnbs
spec:
  replicas: 3
  selector: {}      # New selector field
  template: {}      # New template configuration
status:
  replicas: 3       # Current replica count
  readyReplicas: 3  # Ready replica count
  conditions: []    # Detailed status conditions
```

## Automated Migration Process

### 1. API Version Consistency Fix

The build system includes automated tools to fix API version inconsistencies:

```bash
# Fix all API version inconsistencies
make fix-api-versions

# Manual CRD regeneration
make generate
controller-gen crd:generateEmbeddedObjectMeta=true paths="./api/v1" output:crd:artifacts:config=deployments/crds
```

### 2. Code Generation Updates

Updated controller-gen configuration for v1 APIs:

```bash
# Generate updated code
controller-gen object:headerFile=hack/boilerplate.go.txt paths="github.com/thc1006/nephoran-intent-operator/api/v1"

# Generate CRDs with embedded object metadata
controller-gen crd:generateEmbeddedObjectMeta=true paths="./api/v1" output:crd:artifacts:config=deployments/crds
```

## Migration Implementation Details

### 1. Schema Evolution

#### Enhanced Validation
```go
// NetworkIntent v1 with enhanced validation
type NetworkIntentSpec struct {
    // Description replaces the previous 'intent' field with better validation
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=10
    // +kubebuilder:validation:MaxLength=1000
    Description string `json:"description"`
    
    // Priority with enum validation
    // +kubebuilder:validation:Enum=low;medium;high;critical
    Priority string `json:"priority,omitempty"`
    
    // New structured parameters field
    Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// Enhanced status with conditions
type NetworkIntentStatus struct {
    // +kubebuilder:validation:Enum=Pending;Processing;Ready;Failed
    Phase string `json:"phase,omitempty"`
    
    // Detailed condition reporting
    Conditions []metav1.Condition `json:"conditions,omitempty"`
    
    // Processing metadata
    ProcessingTime *metav1.Time `json:"processingTime,omitempty"`
    Message        string       `json:"message,omitempty"`
}
```

#### Status Conditions
```go
// Standard condition types for NetworkIntent
const (
    NetworkIntentConditionReady      = "Ready"
    NetworkIntentConditionProcessing = "Processing" 
    NetworkIntentConditionFailed     = "Failed"
)

// E2NodeSet enhanced status
type E2NodeSetStatus struct {
    // Current replica count
    Replicas int32 `json:"replicas,omitempty"`
    
    // Ready replica count
    ReadyReplicas int32 `json:"readyReplicas,omitempty"`
    
    // Detailed conditions
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

### 2. Backward Compatibility

#### Conversion Webhooks
```go
// Automatic conversion between v1alpha1 and v1
func (src *NetworkIntent) ConvertTo(dstRaw conversion.Hub) error {
    dst := dstRaw.(*v1.NetworkIntent)
    
    // Convert spec fields
    dst.Spec.Description = src.Spec.Intent  // intent -> description
    dst.Spec.Priority = src.Spec.Priority
    
    // Maintain metadata
    dst.ObjectMeta = src.ObjectMeta
    
    return nil
}

func (dst *NetworkIntent) ConvertFrom(srcRaw conversion.Hub) error {
    src := srcRaw.(*v1.NetworkIntent)
    
    // Convert back for backward compatibility
    dst.Spec.Intent = src.Spec.Description  // description -> intent
    dst.Spec.Priority = src.Spec.Priority
    
    return nil
}
```

#### API Version Support
- **Current**: v1 (recommended)
- **Legacy**: v1alpha1 (supported with automatic conversion)
- **Future**: v1beta1 (planned for advanced features)

## Migration Procedures

### 1. Existing Deployment Migration

#### Pre-Migration Validation
```bash
# Check current API versions in use
kubectl get networkintents.v1alpha1.nephoran.com -A
kubectl get e2nodesets.v1alpha1.nephoran.com -A
kubectl get managedelements.v1alpha1.nephoran.com -A

# Backup existing resources
kubectl get networkintents -A -o yaml > networkintents-backup.yaml
kubectl get e2nodesets -A -o yaml > e2nodesets-backup.yaml
```

#### Automated Migration
```bash
# Apply updated CRDs with conversion support
kubectl apply -f deployments/crds/

# Verify CRD installation
kubectl get crd networkintents.nephoran.com -o yaml | grep "served: true"
kubectl get crd e2nodesets.nephoran.com -o yaml | grep "served: true"
```

#### Manual Resource Migration
```bash
# Convert existing resources (if needed)
./scripts/migrate-api-versions.sh

# Validate migrated resources
kubectl get networkintents -A
kubectl describe networkintent <name>
```

### 2. Development Environment Update

#### Update Development Tools
```bash
# Update controller-gen and related tools
make dev-setup

# Regenerate code with new API versions
make fix-api-versions
make generate
```

#### Update Import Statements
```go
// OLD imports
import (
    nephoran "github.com/thc1006/nephoran-intent-operator/api/v1alpha1"
)

// NEW imports  
import (
    nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)
```

## Validation and Testing

### 1. API Version Validation

#### CRD Validation
```bash
# Validate CRD schema
kubectl apply --dry-run=client -f deployments/crds/
kubectl get crd networkintents.nephoran.com -o yaml | grep -A 10 versions

# Test resource creation with new API
kubectl apply --dry-run=client -f - <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: test-intent
spec:
  description: "Test deployment with v1 API"
  priority: "medium"
EOF
```

#### Backward Compatibility Testing
```bash
# Test v1alpha1 resources still work
kubectl apply --dry-run=client -f - <<EOF
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: legacy-intent
spec:
  intent: "Legacy API test"
  priority: "low"
EOF
```

### 2. Controller Validation

#### Controller Testing
```bash
# Run integration tests with new API versions
make test-integration

# Test controller behavior with both API versions
./test-crds.ps1
```

#### Schema Validation Testing
```bash
# Test enhanced validation rules
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: validation-test
spec:
  description: "Short"  # Should fail - too short
  priority: "invalid"   # Should fail - invalid enum
EOF
```

## Migration Troubleshooting

### 1. Common Issues

#### CRD Installation Failures
```bash
# Symptoms: CRD fails to install or update
# Solution: Check for conflicts and re-apply
kubectl get crd networkintents.nephoran.com -o yaml
kubectl delete crd networkintents.nephoran.com
make fix-api-versions
kubectl apply -f deployments/crds/
```

#### Resource Validation Errors
```bash
# Symptoms: Existing resources fail validation
# Solution: Update resource specifications
kubectl get networkintents <name> -o yaml > temp-resource.yaml
# Edit temp-resource.yaml to fix validation issues
kubectl replace -f temp-resource.yaml
```

#### Controller Recognition Issues
```bash
# Symptoms: Controller doesn't recognize new API version
# Solution: Restart controllers and verify registration
kubectl rollout restart deployment/nephio-bridge
kubectl logs deployment/nephio-bridge | grep "Starting workers"
```

### 2. Rollback Procedures

#### Emergency Rollback to v1alpha1
```bash
# Restore previous CRD versions
git checkout HEAD^ -- deployments/crds/
kubectl apply -f deployments/crds/

# Restart controllers
kubectl rollout restart deployment/nephio-bridge
```

#### Selective Rollback
```bash
# Rollback specific resource type
kubectl get crd networkintents.nephoran.com -o yaml > current-crd.yaml
# Edit current-crd.yaml to restore v1alpha1 as served/storage version
kubectl replace -f current-crd.yaml
```

## Best Practices

### 1. API Version Management

#### Version Strategy
- **v1alpha1**: Legacy support (deprecated)
- **v1**: Current stable API (recommended)
- **v1beta1**: Future beta features (planned)

#### Migration Planning
1. **Gradual Migration**: Support both versions during transition
2. **Validation**: Comprehensive testing of new API versions
3. **Documentation**: Update all examples and documentation
4. **Training**: Update team knowledge on new API structure

### 2. Development Guidelines

#### API Design Principles
- **Backward Compatibility**: Maintain compatibility where possible
- **Progressive Enhancement**: Add features without breaking changes
- **Clear Validation**: Comprehensive validation rules
- **Status Reporting**: Detailed status and condition reporting

#### Code Organization
```go
// Organize API versions clearly
api/
├── v1alpha1/           # Legacy API (deprecated)
│   ├── networkintent_types.go
│   └── e2nodeset_types.go
├── v1/                 # Current stable API
│   ├── networkintent_types.go
│   ├── e2nodeset_types.go
│   └── conversion.go   # Conversion functions
└── v1beta1/           # Future beta API (planned)
```

## Migration Checklist

### Pre-Migration
- [ ] Backup existing resources
- [ ] Update development environment
- [ ] Test new API versions in development
- [ ] Update documentation and examples
- [ ] Plan rollback procedures

### Migration Execution
- [ ] Apply new CRDs with conversion support
- [ ] Verify CRD installation and versions
- [ ] Test backward compatibility
- [ ] Migrate existing resources (if needed)
- [ ] Update controllers and restart

### Post-Migration Validation
- [ ] Verify all resources are accessible
- [ ] Test controller functionality
- [ ] Validate schema enforcement
- [ ] Run integration test suite
- [ ] Monitor for issues

## Conclusion

The API version migration from v1alpha1 to v1 provides improved stability, better validation, and production readiness. The migration maintains backward compatibility while providing enhanced features for production deployments.

For migration questions or issues:
1. Run API version fix: `make fix-api-versions`
2. Check CRD status: `kubectl get crd | grep nephoran`
3. Validate resources: `kubectl get networkintents,e2nodesets -A`
4. Review migration logs in controller deployment