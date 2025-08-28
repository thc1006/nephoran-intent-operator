package generator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SanitizePath prevents directory traversal in path generation
func SanitizePath(path string) string {
	// Remove directory traversal attempts
	path = filepath.Clean(path)
	
	// Prevent absolute paths
	if filepath.IsAbs(path) {
		path = filepath.Base(path)
	}
	
	// Remove any remaining ".." or leading/trailing slashes
	path = strings.TrimPrefix(path, "../")
	path = strings.TrimPrefix(path, "/")
	path = strings.Replace(path, "../", "", -1)
	
	return path
}

// SanitizeCommand prevents command injection
func SanitizeCommand(cmd string) string {
	// Regex for dangerous shell characters
	dangerousChars := regexp.MustCompile(`[;&|<>()$` + "`" + `]`)
	
	// Remove dangerous characters
	sanitizedCmd := dangerousChars.ReplaceAllString(cmd, "")
	
	// Additional layer of sanitization
	sanitizedCmd = strings.ReplaceAll(sanitizedCmd, ";", "")
	sanitizedCmd = strings.ReplaceAll(sanitizedCmd, "|", "")
	sanitizedCmd = strings.ReplaceAll(sanitizedCmd, "&", "")
	
	return sanitizedCmd
}

// ValidateBinaryContent checks content for security risks
func ValidateBinaryContent(content []byte) bool {
	// Check content size (max 5 MB)
	if len(content) > 5*1024*1024 {
		return false
	}
	
	// Check for binary/non-printable characters
	for _, b := range content {
		if b < 32 && b != 9 && b != 10 && b != 13 {
			return false
		}
	}
	
	// Basic YAML validation
	contentStr := string(content)
	yamlValidationRegex := regexp.MustCompile(`^(\s*[a-zA-Z0-9_-]+\s*:\s*[^\n]+\n)*$`)
	
	return yamlValidationRegex.MatchString(contentStr)
}

// GenerateUniqueName generates a cryptographically secure unique package name
func GenerateUniqueName(baseName string) string {
	// Sanitize the base name
	baseName = SanitizePath(baseName)
	
	// Generate 8 bytes of random data
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp if crypto/rand fails
		timestamp := time.Now().UnixNano()
		return fmt.Sprintf("%s-%d", baseName, timestamp)
	}
	
	// Convert to hex string
	randomHex := hex.EncodeToString(randomBytes)
	
	return fmt.Sprintf("%s-%s", baseName, randomHex)
}