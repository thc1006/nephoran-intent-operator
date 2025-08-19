# DEPRECATED: internal/patch

⚠️ **This module is deprecated and should not be used for new development.**

## Replacement

Use `internal/patchgen` instead, which provides:

- **Collision-resistant naming**: Uses crypto/rand with fallback for package names
- **Enhanced security controls**: Comprehensive validation and security metadata
- **Better timestamp handling**: RFC3339Nano timestamps with nanosecond precision
- **Improved error handling**: More detailed error messages and validation
- **Security compliance**: OWASP and O-RAN WG11 compliant implementation

## Migration

Replace imports of:
```go
import "github.com/thc1006/nephoran-intent-operator/internal/patch"
```

With:
```go
import "github.com/thc1006/nephoran-intent-operator/internal/patchgen"
```

## Key Differences

### Old (internal/patch)
- Basic timestamp using Unix epoch
- Simple package naming without collision resistance
- Minimal validation
- No security metadata

### New (internal/patchgen) 
- RFC3339Nano timestamps with crypto/rand suffix
- Collision-resistant package naming
- Comprehensive input validation
- Security metadata and compliance checks
- Better structured types

## Security Issues in internal/patch

1. **Timestamp Collision**: Uses `time.Now().Unix()` which can cause collisions
2. **No Input Validation**: Missing security checks for path traversal
3. **No Security Metadata**: No tracking of security validation
4. **Weak Package Naming**: Predictable naming scheme vulnerable to conflicts

## Timeline

- **Deprecated**: This version (feat/porch-structured-patch)
- **Removal**: Planned for next major release
- **Support**: Bug fixes only, no new features

## Contact

For migration assistance, see:
- `docs/PATCHGEN_MIGRATION_GUIDE.md`
- `docs/PATCHGEN_SECURITY_DOCUMENTATION.md`