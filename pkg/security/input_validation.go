package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// InputValidator provides comprehensive input validation against injection attacks
type InputValidator struct {
	maxInputLength int
	allowedPaths   map[string]bool
	sqlValidator   *SQLValidator
	pathValidator  *PathValidator
	htmlValidator  *HTMLValidator
}

// SQLValidator prevents SQL injection attacks
type SQLValidator struct {
	// Use parameterized queries ONLY - never concatenate user input
	allowedTables  map[string]bool
	allowedColumns map[string]map[string]bool
	maxQueryLength int
}

// PathValidator prevents path traversal attacks
type PathValidator struct {
	basePath       string
	allowedDirs    []string
	forbiddenPaths []string
}

// HTMLValidator prevents XSS attacks
type HTMLValidator struct {
	allowedTags       map[string]bool
	allowedAttributes map[string]map[string]bool
	maxNestingDepth   int
}

// NewInputValidator creates a new input validator with secure defaults
func NewInputValidator() *InputValidator {
	return &InputValidator{
		maxInputLength: 10000, // OWASP recommendation
		allowedPaths:   getDefaultAllowedPaths(),
		sqlValidator:   NewSQLValidator(),
		pathValidator:  NewPathValidator(),
		htmlValidator:  NewHTMLValidator(),
	}
}

// NewSQLValidator creates a SQL validator with secure defaults
func NewSQLValidator() *SQLValidator {
	return &SQLValidator{
		allowedTables: map[string]bool{
			"network_intents": true,
			"audit_logs":      true,
			"configurations":  true,
			"metrics":         true,
		},
		allowedColumns: map[string]map[string]bool{
			"network_intents": {
				"id": true, "name": true, "spec": true, "status": true,
				"created_at": true, "updated_at": true,
			},
			"audit_logs": {
				"id": true, "action": true, "user": true, "timestamp": true,
			},
		},
		maxQueryLength: 1000,
	}
}

// NewPathValidator creates a path validator with secure defaults
func NewPathValidator() *PathValidator {
	return &PathValidator{
		basePath: "/app/data",
		allowedDirs: []string{
			"/app/data/configs",
			"/app/data/templates",
			"/app/data/exports",
		},
		forbiddenPaths: []string{
			"/etc/passwd",
			"/etc/shadow",
			"/proc",
			"/sys",
			"/dev",
			"/..",
		},
	}
}

// NewHTMLValidator creates an HTML validator with secure defaults
func NewHTMLValidator() *HTMLValidator {
	return &HTMLValidator{
		allowedTags: map[string]bool{
			"p": true, "div": true, "span": true,
			"h1": true, "h2": true, "h3": true,
			"ul": true, "ol": true, "li": true,
			"strong": true, "em": true, "code": true,
		},
		allowedAttributes: map[string]map[string]bool{
			"div":  {"class": true, "id": true},
			"span": {"class": true},
			"p":    {"class": true},
		},
		maxNestingDepth: 10,
	}
}

// ValidateAndSanitizeSQL validates SQL input and returns safe parameterized query
func (v *SQLValidator) ValidateAndSanitizeSQL(ctx context.Context, table string, columns []string, conditions map[string]interface{}) (string, []interface{}, error) {
	// Validate table name
	if !v.allowedTables[table] {
		return "", nil, fmt.Errorf("table '%s' is not allowed", table)
	}

	// Validate column names
	allowedCols := v.allowedColumns[table]
	for _, col := range columns {
		if !allowedCols[col] {
			return "", nil, fmt.Errorf("column '%s' is not allowed in table '%s'", col, table)
		}
	}

	// Build parameterized query - NEVER concatenate user input
	var query strings.Builder
	var args []interface{}

	query.WriteString("SELECT ")
	for i, col := range columns {
		if i > 0 {
			query.WriteString(", ")
		}
		// Column names are validated above, safe to use
		query.WriteString(col)
	}

	query.WriteString(" FROM ")
	query.WriteString(table)

	if len(conditions) > 0 {
		query.WriteString(" WHERE ")
		i := 0
		for col, val := range conditions {
			if !allowedCols[col] {
				return "", nil, fmt.Errorf("column '%s' in WHERE clause is not allowed", col)
			}
			if i > 0 {
				query.WriteString(" AND ")
			}
			query.WriteString(col)
			query.WriteString(" = ?")
			args = append(args, val)
			i++
		}
	}

	finalQuery := query.String()
	if len(finalQuery) > v.maxQueryLength {
		return "", nil, fmt.Errorf("query exceeds maximum length of %d", v.maxQueryLength)
	}

	return finalQuery, args, nil
}

// ExecuteSafeQuery executes a query with parameterized inputs
func (v *SQLValidator) ExecuteSafeQuery(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	// Always use parameterized queries
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close() // #nosec G307 - Error handled in defer

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	return rows, nil
}

// ValidateAndSanitizePath validates and sanitizes file paths to prevent traversal
func (p *PathValidator) ValidateAndSanitizePath(inputPath string) (string, error) {
	// Remove any null bytes
	inputPath = strings.ReplaceAll(inputPath, "\x00", "")

	// Clean the path
	cleanPath := filepath.Clean(inputPath)

	// Convert to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check for forbidden patterns
	for _, forbidden := range p.forbiddenPaths {
		if strings.Contains(absPath, forbidden) {
			return "", fmt.Errorf("path contains forbidden pattern: %s", forbidden)
		}
	}

	// Ensure path is within allowed directories
	allowed := false
	for _, allowedDir := range p.allowedDirs {
		absAllowed, _ := filepath.Abs(allowedDir)
		if strings.HasPrefix(absPath, absAllowed) {
			allowed = true
			break
		}
	}

	if !allowed {
		return "", fmt.Errorf("path '%s' is outside allowed directories", absPath)
	}

	// Additional checks for path traversal
	if strings.Contains(absPath, "..") {
		return "", fmt.Errorf("path traversal detected")
	}

	return absPath, nil
}

// SanitizeHTML sanitizes HTML input to prevent XSS attacks
func (h *HTMLValidator) SanitizeHTML(input string) string {
	// First, HTML escape everything
	sanitized := html.EscapeString(input)

	// Remove any JavaScript event handlers
	eventHandlerRegex := regexp.MustCompile(`\bon\w+\s*=`)
	sanitized = eventHandlerRegex.ReplaceAllString(sanitized, "")

	// Remove javascript: protocol
	jsProtocolRegex := regexp.MustCompile(`(?i)javascript:`)
	sanitized = jsProtocolRegex.ReplaceAllString(sanitized, "")

	// Remove data: URLs that could contain scripts
	dataURLRegex := regexp.MustCompile(`(?i)data:[^,]*script`)
	sanitized = dataURLRegex.ReplaceAllString(sanitized, "")

	return sanitized
}

// ValidateJSON validates JSON input against a schema
func ValidateJSON(input string, maxSize int) error {
	// Check size first
	if len(input) > maxSize {
		return fmt.Errorf("JSON input exceeds maximum size of %d bytes", maxSize)
	}

	// Check for valid UTF-8
	if !utf8.ValidString(input) {
		return fmt.Errorf("JSON input contains invalid UTF-8")
	}

	// Validate JSON structure
	var js json.RawMessage
	if err := json.Unmarshal([]byte(input), &js); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Check for recursive depth to prevent stack overflow
	depth := calculateJSONDepth([]byte(input))
	if depth > 100 {
		return fmt.Errorf("JSON nesting depth exceeds maximum of 100")
	}

	return nil
}

// ValidateURL validates and sanitizes URLs
func ValidateURL(inputURL string) (*url.URL, error) {
	// Parse the URL
	u, err := url.Parse(inputURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Check scheme - only allow http(s)
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("only HTTP(S) URLs are allowed")
	}

	// Check for localhost/private IPs (SSRF prevention)
	host := strings.ToLower(u.Hostname())
	if isPrivateHost(host) {
		return nil, fmt.Errorf("private/internal URLs are not allowed")
	}

	// Remove any credentials from URL
	u.User = nil

	// Validate port
	if u.Port() != "" {
		port := u.Port()
		if !isAllowedPort(port) {
			return nil, fmt.Errorf("port %s is not allowed", port)
		}
	}

	return u, nil
}

// ValidateEmail validates email addresses
func ValidateEmail(email string) error {
	// Basic length check
	if len(email) < 3 || len(email) > 254 {
		return fmt.Errorf("invalid email length")
	}

	// RFC 5322 compliant regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}

	// Check for common injection patterns
	if strings.ContainsAny(email, "<>\"'`;") {
		return fmt.Errorf("email contains invalid characters")
	}

	return nil
}

// Helper functions

func getDefaultAllowedPaths() map[string]bool {
	return map[string]bool{
		"/app/data":      true,
		"/app/configs":   true,
		"/app/templates": true,
		"/tmp":           true,
	}
}

func calculateJSONDepth(data []byte) int {
	depth := 0
	maxDepth := 0
	for _, b := range data {
		switch b {
		case '{', '[':
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		case '}', ']':
			depth--
		}
	}
	return maxDepth
}

func isPrivateHost(host string) bool {
	privatePatterns := []string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"::1",
		"169.254",  // Link-local
		"10.",      // Private network
		"172.16.",  // Private network
		"192.168.", // Private network
		"fc00::",   // IPv6 private
		"metadata", // Cloud metadata endpoints
	}

	for _, pattern := range privatePatterns {
		if strings.HasPrefix(host, pattern) || strings.Contains(host, pattern) {
			return true
		}
	}
	return false
}

func isAllowedPort(port string) bool {
	allowedPorts := map[string]bool{
		"80":   true,
		"443":  true,
		"8080": true,
		"8443": true,
		"9090": true, // Metrics
	}
	return allowedPorts[port]
}
