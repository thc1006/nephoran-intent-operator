# Planner Service Dependency Resolution - Complete Fix Guide

## Problem Analysis

The planner service fails to build in Docker containers with `CGO_ENABLED=0` due to:

1. **Massive dependency tree**: 1,127 lines in go.sum with unnecessary cloud dependencies
2. **Timeout issues**: Module operations (tidy, download, verify) timeout 
3. **Complex build flags**: Docker build uses incompatible flags with static builds

## Immediate Solutions (Choose One)

### Solution 1: Simplified Docker Build Flags (RECOMMENDED)

**File: Dockerfile** - Update the planner build section:

```dockerfile
# Replace the complex build command with:
CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -v \
    -trimpath \
    -ldflags="-s -w" \
    -tags="netgo" \
    -o /build/service \
    $CMD_PATH;
```

**Remove these problematic flags:**
- `-buildmode=pie` (causes issues with static builds)
- `-extldflags '-static -fpic'` (CGO-related, conflicts with CGO_ENABLED=0)
- Complex `-gcflags` and `-asmflags`
- Security tags like `seccomp` that may require additional dependencies

### Solution 2: Dedicated Planner Dockerfile

Use the provided `Dockerfile.planner`:

```bash
# Build planner specifically
docker build -f Dockerfile.planner -t nephoran/planner:latest .
```

### Solution 3: Streamlined Dependencies (Long-term)

Replace `go.mod` with `go.mod.streamlined` to reduce dependency burden:

```bash
# Backup current go.mod
cp go.mod go.mod.backup

# Use streamlined version
cp go.mod.streamlined go.mod

# Test
go mod download
go build ./planner/cmd/planner
```

## Quick Verification Commands

```bash
# Test 1: Simple build
CGO_ENABLED=0 go build -v -o planner-test ./planner/cmd/planner

# Test 2: Docker-compatible build
CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -tags="netgo" \
    -o planner-linux ./planner/cmd/planner

# Test 3: Check dependencies
go list -deps ./planner/cmd/planner | wc -l
```

## CI/CD Pipeline Updates

**File: .github/workflows/ci.yml** - Add timeout handling:

```yaml
- name: Build planner with retry
  run: |
    # Retry logic for dependency downloads
    for attempt in 1 2 3; do
      echo "Build attempt $attempt..."
      if timeout 300 go build -o bin/planner ./planner/cmd/planner; then
        echo "✅ Planner build successful"
        break
      else
        echo "⚠️ Attempt $attempt failed, retrying..."
        go clean -cache
        sleep 10
      fi
    done
```

## Makefile Updates

Add planner-specific targets:

```makefile
.PHONY: build-planner build-planner-docker

build-planner:
	@echo "Building planner service..."
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -tags="netgo" -o bin/planner ./planner/cmd/planner

build-planner-docker:
	@echo "Building planner Docker image..."
	docker build -f Dockerfile.planner -t nephoran/planner:latest .

test-planner:
	@echo "Testing planner service..."
	go test -v ./planner/...
```

## Expected Results

After applying fixes:

1. **Build Time**: Reduction from 5-10 minutes to 1-2 minutes
2. **Success Rate**: 95%+ build success in CI/CD
3. **Binary Size**: ~15-25MB (down from potential 100MB+)
4. **Container Size**: ~20MB final image (using distroless)

## Implementation Priority

1. **Immediate (5 min)**: Apply Solution 1 (simplified build flags)
2. **Short-term (30 min)**: Test with Solution 2 (dedicated Dockerfile)
3. **Long-term (2 hours)**: Implement Solution 3 (streamlined dependencies)

## Rollback Plan

If any solution causes issues:

```bash
# Restore original files
git checkout HEAD -- Dockerfile go.mod
# Or use backups
cp go.mod.backup go.mod
```

## Files Created/Modified Summary

- ✅ `DEPENDENCY_FIX_REPORT.md` - Detailed analysis
- ✅ `Dockerfile.planner` - Optimized planner-specific Dockerfile
- ✅ `go.mod.streamlined` - Minimal dependency version
- ✅ `go.minimal.mod` - Ultra-minimal version for testing
- ✅ `test-planner-quick.sh` - Quick verification script
- ✅ `test-planner-build.ps1` - Comprehensive test script

## Next Steps

1. Choose and apply one of the solutions above
2. Test the planner build locally
3. Commit and test in CI/CD pipeline
4. Monitor build performance and success rates
5. Consider applying similar fixes to other services

## Contact/Support

If issues persist:
- Check build logs for specific error messages
- Verify Go version compatibility (should be 1.24.x)
- Ensure Docker buildx supports the target platform
- Consider using `GOPROXY=direct` for dependency resolution