// Package security ensures all security-related dependencies are imported
package security

import (
	// SPIFFE/SPIRE dependencies
	_ "github.com/spiffe/go-spiffe/v2/spiffeid"
	_ "github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	_ "github.com/spiffe/go-spiffe/v2/workloadapi"

	// Vault integration
	_ "github.com/hashicorp/vault/api"

	// File system notifications
	_ "github.com/fsnotify/fsnotify"

	// JWT/JWK handling
	_ "github.com/lestrrat-go/jwx/v2/jwk"

	// HTML sanitization
	_ "github.com/microcosm-cc/bluemonday"

	// Crypto algorithms
	_ "golang.org/x/crypto/acme"
	_ "golang.org/x/crypto/acme/autocert"
	_ "golang.org/x/crypto/argon2"
	_ "golang.org/x/crypto/blake2b"
	_ "golang.org/x/crypto/chacha20poly1305"
	_ "golang.org/x/crypto/hkdf"
	_ "golang.org/x/crypto/ocsp"
	_ "golang.org/x/crypto/pbkdf2"
	_ "golang.org/x/crypto/sha3"
	_ "golang.org/x/crypto/ssh"
)

// This file ensures all security dependencies are included in go.mod
// These imports are blank imports to ensure the packages are available
// for use throughout the codebase.
