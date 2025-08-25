package security

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"math/big"
)

// SecureHTTPClient creates an HTTP client with secure defaults
func SecureHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:               tls.VersionTLS13,
				PreferServerCipherSuites: true,
				InsecureSkipVerify:       false,
				SessionTicketsDisabled:   true,
				Renegotiation:           tls.RenegotiateNever,
			},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true,
		},
	}
}

// SecureFileOpen opens a file with path traversal protection
func SecureFileOpen(basePath, userPath string) (*os.File, error) {
	// Clean and resolve the path
	cleanPath := filepath.Clean(userPath)
	
	// Ensure it's not an absolute path
	if filepath.IsAbs(cleanPath) {
		return nil, fmt.Errorf("absolute paths are not allowed")
	}
	
	// Join with base path and clean again
	fullPath := filepath.Join(basePath, cleanPath)
	finalPath := filepath.Clean(fullPath)
	
	// Ensure the final path is within the base path
	if !strings.HasPrefix(finalPath, filepath.Clean(basePath)) {
		return nil, fmt.Errorf("path traversal detected")
	}
	
	// Check for symbolic links
	info, err := os.Lstat(finalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("symbolic links are not allowed")
	}
	
	return os.Open(finalPath)
}

// GenerateSecureToken generates a cryptographically secure token
func GenerateSecureToken(length int) (string, error) {
	if length < 32 {
		length = 32 // Minimum secure length
	}
	
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// SecretManager provides secure secret management
type SecretManager struct {
	namespace string
}

// NewSecretManager creates a new secret manager
func NewSecretManager(namespace string) *SecretManager {
	return &SecretManager{
		namespace: namespace,
	}
}

// GetSecret retrieves a secret from Kubernetes secrets or environment variables
func (sm *SecretManager) GetSecret(ctx context.Context, key string) (string, error) {
	// First try environment variable
	envKey := fmt.Sprintf("%s_%s", strings.ToUpper(sm.namespace), strings.ToUpper(key))
	if value := os.Getenv(envKey); value != "" {
		return value, nil
	}
	
	// Try without namespace prefix
	if value := os.Getenv(strings.ToUpper(key)); value != "" {
		return value, nil
	}
	
	// In production, this would fetch from Kubernetes secrets
	// For now, return an error to force proper configuration
	return "", fmt.Errorf("secret %s not found in environment", key)
}

// SecurityHeadersMiddleware adds security headers to HTTP responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		next.ServeHTTP(w, r)
	})
}

// SanitizeInput sanitizes user input to prevent injection attacks
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	// Limit length to prevent DoS
	const maxLength = 10000
	if len(input) > maxLength {
		input = input[:maxLength]
	}
	
	return input
}

// ValidatePath validates a file path for security issues
func ValidatePath(path string) error {
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal detected")
	}
	
	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("null byte detected in path")
	}
	
	// Check for absolute paths (depending on use case)
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed")
	}
	
	return nil
}

// SecureRandomString generates a cryptographically secure random string
func SecureRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}