package validation

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// EndpointValidationError represents an error validating an endpoint
type EndpointValidationError struct {
	Endpoint string
	Service  string
	Err      error
}

func (e *EndpointValidationError) Error() string {
	return fmt.Sprintf("%s endpoint validation failed: %v", e.Service, e.Err)
}

// ValidationConfig controls endpoint validation behavior
type ValidationConfig struct {
	// ValidateDNS enables DNS resolution checking
	ValidateDNS bool
	// ValidateReachability enables HTTP connectivity checking
	ValidateReachability bool
	// Timeout for reachability checks
	Timeout time.Duration
}

// DefaultValidationConfig returns validation config with sensible defaults
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		ValidateDNS:          false, // DNS check can be slow, disabled by default
		ValidateReachability: false, // HTTP check requires service running, disabled by default
		Timeout:              5 * time.Second,
	}
}

// ValidateEndpoint validates an endpoint URL with comprehensive checks
func ValidateEndpoint(endpoint, serviceName string, config *ValidationConfig) error {
	if config == nil {
		config = DefaultValidationConfig()
	}

	// Allow empty for optional endpoints
	if endpoint == "" {
		return nil
	}

	// Parse URL
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return &EndpointValidationError{
			Endpoint: endpoint,
			Service:  serviceName,
			Err:      fmt.Errorf("invalid URL format: %w", err),
		}
	}

	// Validate scheme
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return &EndpointValidationError{
			Endpoint: endpoint,
			Service:  serviceName,
			Err:      fmt.Errorf("unsupported URL scheme '%s' (only http/https allowed)", parsed.Scheme),
		}
	}

	// Validate hostname is present
	if parsed.Host == "" {
		return &EndpointValidationError{
			Endpoint: endpoint,
			Service:  serviceName,
			Err:      fmt.Errorf("missing hostname in URL"),
		}
	}

	// Extract hostname without port for DNS check
	hostname := parsed.Hostname()
	if hostname == "" {
		return &EndpointValidationError{
			Endpoint: endpoint,
			Service:  serviceName,
			Err:      fmt.Errorf("unable to extract hostname from URL"),
		}
	}

	// DNS resolution check (optional)
	if config.ValidateDNS {
		if err := validateDNSResolution(hostname, serviceName); err != nil {
			return err
		}
	}

	// Reachability check (optional)
	if config.ValidateReachability {
		if err := validateReachability(endpoint, serviceName, config.Timeout); err != nil {
			return err
		}
	}

	return nil
}

// validateDNSResolution checks if the hostname can be resolved
func validateDNSResolution(hostname, serviceName string) error {
	// Skip localhost and IPs
	if hostname == "localhost" || net.ParseIP(hostname) != nil {
		return nil
	}

	// Attempt DNS resolution with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resolver := &net.Resolver{}
	addrs, err := resolver.LookupHost(ctx, hostname)
	if err != nil {
		// Provide helpful error messages for common cases
		if strings.Contains(err.Error(), "no such host") {
			return &EndpointValidationError{
				Endpoint: hostname,
				Service:  serviceName,
				Err:      fmt.Errorf("cannot resolve hostname (DNS lookup failed): %w. Check if service is deployed and DNS is configured", err),
			}
		}
		return &EndpointValidationError{
			Endpoint: hostname,
			Service:  serviceName,
			Err:      fmt.Errorf("DNS resolution failed: %w", err),
		}
	}

	if len(addrs) == 0 {
		return &EndpointValidationError{
			Endpoint: hostname,
			Service:  serviceName,
			Err:      fmt.Errorf("DNS resolution returned no addresses"),
		}
	}

	return nil
}

// validateReachability checks if the endpoint is reachable via HTTP
func validateReachability(endpoint, serviceName string, timeout time.Duration) error {
	client := &http.Client{
		Timeout: timeout,
		// Don't follow redirects for validation
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Head(endpoint)
	if err != nil {
		// Provide helpful error messages
		if strings.Contains(err.Error(), "connection refused") {
			return &EndpointValidationError{
				Endpoint: endpoint,
				Service:  serviceName,
				Err:      fmt.Errorf("endpoint unreachable (connection refused): %w. Check if service is running", err),
			}
		}
		if strings.Contains(err.Error(), "timeout") {
			return &EndpointValidationError{
				Endpoint: endpoint,
				Service:  serviceName,
				Err:      fmt.Errorf("endpoint unreachable (timeout): %w. Check network connectivity", err),
			}
		}
		return &EndpointValidationError{
			Endpoint: endpoint,
			Service:  serviceName,
			Err:      fmt.Errorf("endpoint unreachable: %w", err),
		}
	}
	defer resp.Body.Close()

	// Accept any HTTP response (even errors) as "reachable"
	// We just want to know the service is listening
	return nil
}

// ValidateEndpoints validates multiple endpoints and returns all errors
func ValidateEndpoints(endpoints map[string]string, config *ValidationConfig) []error {
	var errors []error
	for serviceName, endpoint := range endpoints {
		if err := ValidateEndpoint(endpoint, serviceName, config); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// GetCommonErrorSuggestions returns helpful suggestions for common endpoint errors
func GetCommonErrorSuggestions(serviceName string) string {
	suggestions := map[string]string{
		"A1 Mediator": "Set A1_MEDIATOR_URL environment variable or --a1-endpoint flag. Example: http://service-ricplt-a1mediator-http.ricplt:8080",
		"Porch Server": "Set PORCH_SERVER_URL environment variable or --porch-server flag. Example: http://porch-server:7007",
		"LLM Service":  "Set LLM_PROCESSOR_URL environment variable or --llm-endpoint flag. Example: http://ollama-service:11434",
		"RAG Service":  "Set RAG_API_URL environment variable. Example: http://rag-service:8000",
	}

	if suggestion, ok := suggestions[serviceName]; ok {
		return suggestion
	}
	return fmt.Sprintf("Configure %s endpoint via environment variable or command-line flag", serviceName)
}
