# Cloud Provider SDK Dependencies

## Overview
This project uses the latest stable cloud provider SDKs for multi-cloud support.

## Supported Providers
- AWS (aws-sdk-go-v2)
- Azure (azure-sdk-for-go)
- Google Cloud Platform
- OpenStack
- VMware

## Dependency Resolution

### Recommended Steps
```bash
# Use default Go module proxy
go mod download

# If network issues persist, use direct download
GOPROXY=direct go mod download

# Verify dependencies
go mod tidy
go mod verify
```

### Troubleshooting
- Network timeout? Try manual downloads
- Incompatible versions? Check Go module requirements
- Proxy issues? Use `GOPROXY=direct`

## Version Compatibility
- Go Version: 1.24.6
- Kubernetes: 1.29+
- Cloud SDK Versions: Latest stable (2025 release)

## Networking Configuration
If you encounter network-related issues:
1. Check firewall settings
2. Verify Go module proxy access
3. Configure `GOPRIVATE` for internal modules

## Support
Contact cloud provider integration team for specific SDK support.