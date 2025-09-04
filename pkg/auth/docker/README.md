# Docker Registry Authentication

## ORAS v2 Migration Guide

This package provides Docker registry authentication using ORAS (OCI Registry As Storage) v2 library. 

### Migrating from ORAS v1

Previously, you might have used `types.AuthConfig`:

```go
// Old ORAS v1 code
authConfig := &types.AuthConfig{
    Username: "username",
    Password: "password",
}
```

Now with ORAS v2, use the new authentication approach:

```go
// New ORAS v2 code
client := docker.NewClient(nil)
err := client.Login(ctx, &docker.LoginConfig{
    Registry: "example.com",
    Username: "username", 
    Password: "password",
})
```

### Key Changes
- Replaced `types.AuthConfig` with `auth.Credential`
- Authentication is now handled through `auth.Client`
- More flexible credential management
- Better support for different authentication methods

### Features
- Token-based authentication
- Credential store support
- Secure credential handling