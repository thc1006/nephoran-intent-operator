// Package o2 implements HTTP utility methods for the O2 IMS API server
package o2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	// Global validator instance
	validate = validator.New()
)

// writeJSONResponse writes a JSON response with proper headers
func (s *O2APIServer) writeJSONResponse(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(statusCode)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			s.logger.Error("failed to encode JSON response",
				"error", err,
				"request_id", r.Context().Value("request_id"),
			)
		}
	}
}

// writeErrorResponse writes an error response following RFC 7807 Problem Details
func (s *O2APIServer) writeErrorResponse(w http.ResponseWriter, r *http.Request, statusCode int, title string, err error) {
	requestID := ""
	if id := r.Context().Value("request_id"); id != nil {
		requestID = id.(string)
	}

	problem := ProblemDetail{
		Type:     "about:blank",
		Title:    title,
		Status:   statusCode,
		Instance: r.URL.Path,
	}

	if err != nil {
		problem.Detail = err.Error()
		s.logger.Error("API error response",
			"error", err,
			"status_code", statusCode,
			"title", title,
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
		)
	} else {
		s.logger.Warn("API error response",
			"status_code", statusCode,
			"title", title,
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
		)
	}

	w.Header().Set("Content-Type", ContentTypeProblemJSON)
	w.WriteHeader(statusCode)

	if encodeErr := json.NewEncoder(w).Encode(problem); encodeErr != nil {
		s.logger.Error("failed to encode error response",
			"error", encodeErr,
			"request_id", requestID,
		)
	}
}

// decodeJSONRequest decodes and validates a JSON request body
func (s *O2APIServer) decodeJSONRequest(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	// Check content type
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, ContentTypeJSON) {
		return fmt.Errorf("invalid content type: expected %s, got %s", ContentTypeJSON, contentType)
	}

	// Decode JSON
	decoder := json.NewDecoder(r.Body)
	if s.config.SecurityConfig != nil && s.config.SecurityConfig.InputValidation != nil &&
		s.config.SecurityConfig.InputValidation.StrictValidation {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate struct if validation is enabled
	if s.config.SecurityConfig != nil && s.config.SecurityConfig.InputValidation != nil &&
		s.config.SecurityConfig.InputValidation.EnableSchemaValidation {
		if err := validate.Struct(target); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	return nil
}

// validateContentLength checks if the request body size is within limits
func (s *O2APIServer) validateContentLength(r *http.Request) error {
	if s.config.SecurityConfig != nil && s.config.SecurityConfig.InputValidation != nil {
		maxSize := int64(s.config.SecurityConfig.InputValidation.MaxRequestSize)
		if maxSize > 0 && r.ContentLength > maxSize {
			return fmt.Errorf("request body too large: %d bytes (max: %d bytes)", r.ContentLength, maxSize)
		}
	}
	return nil
}

// extractRequestContext creates a RequestContext from the HTTP request
func (s *O2APIServer) extractRequestContext(r *http.Request) *RequestContext {
	requestID := ""
	if id := r.Context().Value("request_id"); id != nil {
		requestID = id.(string)
	}

	return &RequestContext{
		RequestID:   requestID,
		Method:      r.Method,
		Path:        r.URL.Path,
		RemoteAddr:  r.RemoteAddr,
		UserAgent:   r.Header.Get("User-Agent"),
		Headers:     r.Header,
		QueryParams: r.URL.Query(),
		StartTime:   r.Context().Value("start_time").(time.Time),
	}
}

// setSecurityHeaders sets security-related HTTP headers
func (s *O2APIServer) setSecurityHeaders(w http.ResponseWriter) {
	// Security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Content Security Policy
	w.Header().Set("Content-Security-Policy", "default-src 'self'")

	// Cache control for sensitive data
	if strings.Contains(w.Header().Get("Content-Type"), "json") {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}
}

// sanitizeInput sanitizes input strings to prevent injection attacks
func (s *O2APIServer) sanitizeInput(input string) string {
	if s.config.SecurityConfig != nil && s.config.SecurityConfig.InputValidation != nil &&
		s.config.SecurityConfig.InputValidation.SanitizeInput {
		// Basic sanitization - remove potentially dangerous characters
		// In production, use a proper sanitization library
		input = strings.ReplaceAll(input, "<", "&lt;")
		input = strings.ReplaceAll(input, ">", "&gt;")
		input = strings.ReplaceAll(input, "\"", "&quot;")
		input = strings.ReplaceAll(input, "'", "&#x27;")
		input = strings.ReplaceAll(input, "&", "&amp;")
	}
	return input
}

// handlePreflight handles CORS preflight requests
func (s *O2APIServer) handlePreflight(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for preflight requests
	if s.corsMiddleware != nil {
		// The CORS middleware will handle this
		return
	}

	// Basic preflight response if no CORS middleware
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.WriteHeader(http.StatusOK)
}

// parseTimeParam parses a time parameter from query string
func (s *O2APIServer) parseTimeParam(r *http.Request, param string) (*time.Time, error) {
	value := r.URL.Query().Get(param)
	if value == "" {
		return nil, nil
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("invalid time format for %s: %s (expected RFC3339)", param, value)
	}

	return &t, nil
}

// buildPaginationResponse builds a paginated response with metadata
func (s *O2APIServer) buildPaginationResponse(data interface{}, total, limit, offset int) map[string]interface{} {
	response := map[string]interface{}{
		"data": data,
		"pagination": map[string]interface{}{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	}

	// Add navigation links if applicable
	if offset > 0 {
		response["pagination"].(map[string]interface{})["hasPrevious"] = true
	}

	if offset+limit < total {
		response["pagination"].(map[string]interface{})["hasNext"] = true
	}

	return response
}

// validateResourceID validates that a resource ID is properly formatted
func (s *O2APIServer) validateResourceID(resourceID string) error {
	if resourceID == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}

	// Basic validation - could be enhanced with more specific rules
	if len(resourceID) < 3 || len(resourceID) > 128 {
		return fmt.Errorf("resource ID must be between 3 and 128 characters")
	}

	// Check for invalid characters
	for _, char := range resourceID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '.') {
			return fmt.Errorf("resource ID contains invalid character: %c", char)
		}
	}

	return nil
}

// extractUserContext extracts user information from request context
func (s *O2APIServer) extractUserContext(r *http.Request) (*UserContext, error) {
	// This would typically extract user information from JWT token or session
	userCtx := &UserContext{
		UserID:   r.Header.Get("X-User-ID"),
		TenantID: r.Header.Get("X-Tenant-ID"),
		Roles:    strings.Split(r.Header.Get("X-User-Roles"), ","),
	}

	// If authentication is enabled, validate user context
	if s.config.AuthenticationConfig != nil && s.config.AuthenticationConfig.Enabled {
		if userCtx.UserID == "" {
			return nil, fmt.Errorf("user ID is required")
		}
	}

	return userCtx, nil
}

// checkPermissions checks if the user has permission for the requested operation
func (s *O2APIServer) checkPermissions(userCtx *UserContext, operation string, resource string) error {
	// Basic permission checking - would be enhanced with proper RBAC in production
	if userCtx == nil {
		return fmt.Errorf("user context is required")
	}

	// Check if user has required role for the operation
	requiredRoles := s.getRequiredRoles(operation, resource)
	if len(requiredRoles) == 0 {
		return nil // No special permissions required
	}

	for _, userRole := range userCtx.Roles {
		for _, requiredRole := range requiredRoles {
			if userRole == requiredRole {
				return nil // User has required permission
			}
		}
	}

	return fmt.Errorf("insufficient permissions for operation %s on resource %s", operation, resource)
}

// getRequiredRoles returns the roles required for a specific operation and resource
func (s *O2APIServer) getRequiredRoles(operation string, resource string) []string {
	// Simple role mapping - would be configurable in production
	roleMap := map[string]map[string][]string{
		"create": {
			"resource_pool":       {"admin", "operator"},
			"resource":            {"admin", "operator"},
			"deployment":          {"admin", "operator"},
			"deployment_template": {"admin"},
		},
		"update": {
			"resource_pool":       {"admin", "operator"},
			"resource":            {"admin", "operator"},
			"deployment":          {"admin", "operator"},
			"deployment_template": {"admin"},
		},
		"delete": {
			"resource_pool":       {"admin"},
			"resource":            {"admin", "operator"},
			"deployment":          {"admin", "operator"},
			"deployment_template": {"admin"},
		},
		"read": {
			"*": {"admin", "operator", "viewer"},
		},
	}

	if resourceRoles, exists := roleMap[operation][resource]; exists {
		return resourceRoles
	}

	// Check for wildcard permissions
	if wildcardRoles, exists := roleMap[operation]["*"]; exists {
		return wildcardRoles
	}

	return []string{} // No special permissions required
}

// UserContext represents user context information
type UserContext struct {
	UserID   string   `json:"user_id"`
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
}

// Supporting types for API utilities





