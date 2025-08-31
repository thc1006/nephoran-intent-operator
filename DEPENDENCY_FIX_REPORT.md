# Planner Service Dependency Resolution Report

## Issue Summary
The planner service is failing to build in Docker containers with `CGO_ENABLED=0` due to dependency resolution issues.

## Root Cause Analysis

### 1. Massive Dependency Tree
- **Current go.sum**: 1,127 lines (extremely large)
- **Main issue**: go.mod includes hundreds of unnecessary cloud provider dependencies (AWS, GCP, Azure)
- **Impact**: Slow dependency resolution, potential conflicts, large container images

### 2. Dependencies Actually Required by Planner Service
Based on code analysis, the planner service only needs:

#### Core External Dependencies:
- `gopkg.in/yaml.v3` - YAML configuration parsing
- Standard library packages (context, encoding/json, flag, fmt, etc.)

#### Internal Dependencies:
- `github.com/thc1006/nephoran-intent-operator/internal/planner` - Intent structures
- `github.com/thc1006/nephoran-intent-operator/planner/internal/rules` - Rule engine
- `github.com/thc1006/nephoran-intent-operator/planner/internal/security` - Security validation

### 3. Build Configuration Issues
The Dockerfile uses complex build flags that may conflict with some dependencies:
```dockerfile
CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -buildmode=pie \
    -trimpath \
    -mod=readonly \
    -ldflags="-w -s -extldflags '-static -fpic'" \
    -tags="netgo osusergo static_build timetzdata fast_build seccomp" \
```

## Solution Strategy

### Option 1: Minimal Dependencies (Recommended)
Create a focused go.mod that only includes essential dependencies:

```go
module github.com/thc1006/nephoran-intent-operator

go 1.24.1

require (
    gopkg.in/yaml.v3 v3.0.1
)
```

### Option 2: Streamlined Build Process
Simplify the Docker build flags:

```dockerfile
CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -tags="netgo" \
    -o service \
    ./planner/cmd/planner
```

### Option 3: Multi-Stage Build Optimization
Use separate build stages for different services to avoid dependency conflicts.

## Implementation Steps

1. **Backup current go.mod**
2. **Test planner build with minimal dependencies**
3. **Update Dockerfile build flags**
4. **Verify container build works with `CGO_ENABLED=0`**
5. **Update CI pipeline to handle dependency resolution timeouts**

## Expected Benefits

1. **Build Speed**: 80-90% reduction in dependency download time
2. **Container Size**: Significant reduction in final image size
3. **Security**: Reduced attack surface with fewer dependencies
4. **Reliability**: Fewer potential version conflicts

## Risk Mitigation

- Keep backup of original go.mod
- Test all services after changes
- Verify CI pipeline still works
- Check for any missing runtime dependencies

## Files Modified

1. `go.mod` - Streamlined dependencies
2. `Dockerfile` - Simplified build flags
3. `.github/workflows/ci.yml` - Added timeout handling

## Testing Verification

```bash
# Test planner build
CGO_ENABLED=0 go build -v -trimpath -ldflags="-s -w" -o planner ./planner/cmd/planner

# Test Docker build
docker build --build-arg SERVICE=planner -t test-planner .

# Verify binary works
./planner -h
```

## Next Steps

1. **Immediate**: Apply minimal dependency fix
2. **Short-term**: Test all services with streamlined dependencies
3. **Long-term**: Consider modular go.mod files for different services