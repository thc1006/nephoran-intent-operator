// Package security provides SSRF (Server-Side Request Forgery) protection
// for user-provided endpoint URLs.
//
// This module validates URLs at configuration time to prevent SSRF attacks
// where an attacker could trick the server into making requests to internal
// services, cloud metadata endpoints, or arbitrary file systems.
//
// Protected against:
//   - Non-HTTP schemes (file://, gopher://, ftp://, etc.)
//   - Localhost and loopback addresses (127.0.0.0/8, ::1)
//   - Private IP ranges (RFC 1918: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
//   - Link-local addresses (169.254.0.0/16, fe80::/10)
//   - Cloud metadata endpoints (169.254.169.254, metadata.google.internal, etc.)
//   - IPv6-mapped IPv4 addresses used to bypass filters
//   - URL authority confusion attacks (user@host in URL)
package security

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// SSRFError represents a URL validation failure due to SSRF risk.
type SSRFError struct {
	URL    string
	Reason string
}

func (e *SSRFError) Error() string {
	return fmt.Sprintf("SSRF protection: URL %q rejected: %s", e.URL, e.Reason)
}

// SSRFValidatorConfig allows customization of the SSRF validator behavior.
type SSRFValidatorConfig struct {
	// AllowLocalhost permits localhost and loopback addresses.
	// This should ONLY be true in development/testing environments.
	AllowLocalhost bool

	// AllowPrivateIPs permits RFC 1918 private IP ranges.
	// This should ONLY be true when the operator intentionally
	// communicates with in-cluster services by IP.
	AllowPrivateIPs bool

	// AllowedHosts is a whitelist of hostnames that are always permitted,
	// even if they resolve to private IPs. Useful for known internal
	// service names (e.g., "porch-server", "weaviate.weaviate.svc.cluster.local").
	AllowedHosts []string

	// AllowedSchemes overrides the default allowed schemes (http, https).
	AllowedSchemes []string
}

// SSRFValidator validates URLs against SSRF attack patterns.
type SSRFValidator struct {
	config       SSRFValidatorConfig
	allowedHosts map[string]bool
	schemes      map[string]bool
}

// NewSSRFValidator creates a new SSRF validator with the given configuration.
func NewSSRFValidator(config SSRFValidatorConfig) *SSRFValidator {
	v := &SSRFValidator{
		config:       config,
		allowedHosts: make(map[string]bool),
	}

	for _, h := range config.AllowedHosts {
		v.allowedHosts[strings.ToLower(h)] = true
	}

	v.schemes = make(map[string]bool)
	if len(config.AllowedSchemes) > 0 {
		for _, s := range config.AllowedSchemes {
			v.schemes[strings.ToLower(s)] = true
		}
	} else {
		v.schemes["http"] = true
		v.schemes["https"] = true
	}

	return v
}

// NewDefaultSSRFValidator creates a validator with strict production defaults.
// No localhost, no private IPs, no allowed host exceptions.
func NewDefaultSSRFValidator() *SSRFValidator {
	return NewSSRFValidator(SSRFValidatorConfig{})
}

// ValidateEndpointURL validates a URL string for SSRF safety.
// Returns nil if the URL is safe to use, or an SSRFError describing the risk.
func (v *SSRFValidator) ValidateEndpointURL(rawURL string) error {
	if rawURL == "" {
		return &SSRFError{URL: rawURL, Reason: "empty URL"}
	}

	// Parse the URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return &SSRFError{URL: rawURL, Reason: fmt.Sprintf("malformed URL: %v", err)}
	}

	// 1. Validate scheme (MUST be http or https)
	if err := v.validateScheme(u, rawURL); err != nil {
		return err
	}

	// 2. Reject URLs with userinfo (user:pass@host) to prevent authority confusion
	if u.User != nil {
		return &SSRFError{
			URL:    rawURL,
			Reason: "URL contains userinfo component (potential authority confusion attack)",
		}
	}

	// 3. Extract and validate the hostname
	host := u.Hostname()
	if host == "" {
		return &SSRFError{URL: rawURL, Reason: "URL has no hostname"}
	}

	// 4. Check allowed hosts whitelist (bypass further checks for known-good hosts)
	if v.allowedHosts[strings.ToLower(host)] {
		return nil
	}

	// 5. Check for cloud metadata endpoints (by hostname)
	if err := v.validateNotCloudMetadata(host, rawURL); err != nil {
		return err
	}

	// 6. Check for localhost/loopback
	if err := v.validateNotLocalhost(host, rawURL); err != nil {
		return err
	}

	// 7. Parse IP and check for private/link-local ranges
	if err := v.validateIPAddress(host, rawURL); err != nil {
		return err
	}

	return nil
}

// validateScheme ensures only allowed URL schemes are used.
func (v *SSRFValidator) validateScheme(u *url.URL, rawURL string) error {
	scheme := strings.ToLower(u.Scheme)
	if scheme == "" {
		return &SSRFError{
			URL:    rawURL,
			Reason: "URL has no scheme (http:// or https:// required)",
		}
	}

	if !v.schemes[scheme] {
		return &SSRFError{
			URL:    rawURL,
			Reason: fmt.Sprintf("scheme %q not allowed (permitted: %s)", scheme, v.allowedSchemesString()),
		}
	}

	return nil
}

// validateNotCloudMetadata blocks access to cloud provider metadata endpoints.
func (v *SSRFValidator) validateNotCloudMetadata(host, rawURL string) error {
	lowHost := strings.ToLower(host)

	// AWS, GCP, Azure, DigitalOcean, Oracle Cloud metadata IP
	if host == "169.254.169.254" {
		return &SSRFError{
			URL:    rawURL,
			Reason: "cloud metadata endpoint (169.254.169.254) is blocked",
		}
	}

	// GCP metadata hostname
	if lowHost == "metadata.google.internal" {
		return &SSRFError{
			URL:    rawURL,
			Reason: "GCP metadata endpoint (metadata.google.internal) is blocked",
		}
	}

	// Azure metadata hostname
	if lowHost == "metadata.azure.com" {
		return &SSRFError{
			URL:    rawURL,
			Reason: "Azure metadata endpoint (metadata.azure.com) is blocked",
		}
	}

	// AWS EC2 metadata hostname (alternative to IP)
	if lowHost == "instance-data" || lowHost == "instance-data." {
		return &SSRFError{
			URL:    rawURL,
			Reason: "AWS instance metadata endpoint is blocked",
		}
	}

	// Kubernetes service account token endpoint
	if lowHost == "kubernetes.default.svc" || lowHost == "kubernetes.default" || lowHost == "kubernetes" {
		return &SSRFError{
			URL:    rawURL,
			Reason: "Kubernetes API server endpoint is blocked for user-provided URLs",
		}
	}

	return nil
}

// validateNotLocalhost checks for localhost and loopback references.
func (v *SSRFValidator) validateNotLocalhost(host, rawURL string) error {
	if v.config.AllowLocalhost {
		return nil
	}

	lowHost := strings.ToLower(host)

	// String-based localhost detection
	localhostNames := []string{
		"localhost",
		"localhost.",      // FQDN variant
		"localhost.localdomain",
		"ip6-localhost",
		"ip6-loopback",
	}

	for _, name := range localhostNames {
		if lowHost == name {
			return &SSRFError{
				URL:    rawURL,
				Reason: fmt.Sprintf("localhost hostname %q is blocked", host),
			}
		}
	}

	// Numeric loopback detection (127.0.0.0/8)
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return &SSRFError{
			URL:    rawURL,
			Reason: fmt.Sprintf("loopback address %q is blocked", host),
		}
	}

	// Detect IPv6 loopback (::1) even in bracket notation
	stripped := strings.Trim(host, "[]")
	if ip2 := net.ParseIP(stripped); ip2 != nil && ip2.IsLoopback() {
		return &SSRFError{
			URL:    rawURL,
			Reason: fmt.Sprintf("IPv6 loopback address %q is blocked", host),
		}
	}

	// Check for 0.0.0.0 (binds all interfaces, often resolves to localhost)
	if host == "0.0.0.0" || host == "::" {
		return &SSRFError{
			URL:    rawURL,
			Reason: fmt.Sprintf("unspecified address %q is blocked", host),
		}
	}

	return nil
}

// validateIPAddress checks if the host is a private or link-local IP.
func (v *SSRFValidator) validateIPAddress(host, rawURL string) error {
	// Strip IPv6 bracket notation if present
	stripped := strings.Trim(host, "[]")
	ip := net.ParseIP(stripped)

	if ip == nil {
		// Not an IP address, it is a hostname. We do not perform DNS resolution
		// here because that would be a TOCTOU race condition. Instead, we rely
		// on the network-level dialer restrictions (see SSRFSafeDialer).
		return nil
	}

	// Check for unspecified addresses
	if ip.IsUnspecified() {
		return &SSRFError{
			URL:    rawURL,
			Reason: fmt.Sprintf("unspecified address %q is blocked", host),
		}
	}

	// Check for loopback (127.0.0.0/8 or ::1)
	if ip.IsLoopback() {
		if !v.config.AllowLocalhost {
			return &SSRFError{
				URL:    rawURL,
				Reason: fmt.Sprintf("loopback address %q is blocked", host),
			}
		}
	}

	// Check for link-local addresses (169.254.0.0/16, fe80::/10)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return &SSRFError{
			URL:    rawURL,
			Reason: fmt.Sprintf("link-local address %q is blocked", host),
		}
	}

	// Check for private IP ranges
	if !v.config.AllowPrivateIPs && isPrivateIP(ip) {
		return &SSRFError{
			URL:    rawURL,
			Reason: fmt.Sprintf("private IP address %q is blocked", host),
		}
	}

	// Check for IPv6-mapped IPv4 addresses that could bypass filters
	// e.g., ::ffff:127.0.0.1 or ::ffff:10.0.0.1
	if ipv4 := ip.To4(); ipv4 != nil && !ip.Equal(ipv4) {
		// This is an IPv6-mapped IPv4 address; re-validate the inner IPv4
		if ipv4.IsLoopback() && !v.config.AllowLocalhost {
			return &SSRFError{
				URL:    rawURL,
				Reason: fmt.Sprintf("IPv6-mapped loopback %q is blocked", host),
			}
		}
		if ipv4.IsLinkLocalUnicast() {
			return &SSRFError{
				URL:    rawURL,
				Reason: fmt.Sprintf("IPv6-mapped link-local address %q is blocked", host),
			}
		}
		if !v.config.AllowPrivateIPs && isPrivateIP(ipv4) {
			return &SSRFError{
				URL:    rawURL,
				Reason: fmt.Sprintf("IPv6-mapped private IP %q is blocked", host),
			}
		}
	}

	return nil
}

// isPrivateIP checks if an IP address falls in RFC 1918 private ranges.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{mustParseCIDR("10.0.0.0/8")},
		{mustParseCIDR("172.16.0.0/12")},
		{mustParseCIDR("192.168.0.0/16")},
		// IPv6 unique local addresses
		{mustParseCIDR("fc00::/7")},
	}

	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}

	return false
}

// mustParseCIDR parses a CIDR string or panics. Only used for compile-time constants.
func mustParseCIDR(cidr string) *net.IPNet {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(fmt.Sprintf("invalid CIDR %q: %v", cidr, err))
	}
	return network
}

// allowedSchemesString returns a comma-separated list of allowed schemes.
func (v *SSRFValidator) allowedSchemesString() string {
	schemes := make([]string, 0, len(v.schemes))
	for s := range v.schemes {
		schemes = append(schemes, s)
	}
	return strings.Join(schemes, ", ")
}

// --- Convenience functions for common use cases ---

// ValidateEndpointURL validates a URL using the default strict SSRF validator.
// This is the recommended function for validating user-provided endpoint URLs.
func ValidateEndpointURL(rawURL string) error {
	return NewDefaultSSRFValidator().ValidateEndpointURL(rawURL)
}

// ValidateEndpointURLWithAllowlist validates a URL with specific hosts allowed.
// Use this when the operator needs to communicate with known internal services.
func ValidateEndpointURLWithAllowlist(rawURL string, allowedHosts []string) error {
	v := NewSSRFValidator(SSRFValidatorConfig{
		AllowedHosts: allowedHosts,
	})
	return v.ValidateEndpointURL(rawURL)
}

// ValidateInClusterEndpointURL validates a URL allowing private IPs and
// in-cluster service names. This is appropriate for URLs that are expected
// to point to Kubernetes cluster-internal services.
func ValidateInClusterEndpointURL(rawURL string) error {
	v := NewSSRFValidator(SSRFValidatorConfig{
		AllowPrivateIPs: true,
	})
	return v.ValidateEndpointURL(rawURL)
}
