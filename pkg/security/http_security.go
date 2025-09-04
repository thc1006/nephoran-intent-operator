// Package security provides secure HTTP configurations.

package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// SecureHTTPClient creates an HTTP client with security best practices.

func SecureHTTPClient(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,

		DialContext: (&net.Dialer{
			Timeout: 30 * time.Second,

			KeepAlive: 30 * time.Second,
		}).DialContext,

		ForceAttemptHTTP2: true,

		MaxIdleConns: 100,

		MaxIdleConnsPerHost: 10,

		IdleConnTimeout: 90 * time.Second,

		TLSHandshakeTimeout: 10 * time.Second,

		ExpectContinueTimeout: 1 * time.Second,

		DisableCompression: false,

		DisableKeepAlives: false,
	}

	return &http.Client{
		Timeout: timeout,

		Transport: transport,

		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Limit redirects to prevent redirect loops.

			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}

			return nil
		},
	}
}

// ValidateHTTPURL validates and sanitizes URLs to prevent injection attacks.

func ValidateHTTPURL(rawURL string) (*url.URL, error) {
	// Parse the URL.

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Validate scheme (only allow http/https).

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme: %s", u.Scheme)
	}

	// Check for suspicious patterns.

	if u.User != nil {
		return nil, fmt.Errorf("URLs with credentials are not allowed")
	}

	// Validate host.

	if u.Host == "" {
		return nil, fmt.Errorf("URL must have a host")
	}

	// Check for local addresses (prevent SSRF).

	if isLocalAddress(u.Host) {
		return nil, fmt.Errorf("local addresses are not allowed")
	}

	return u, nil
}

// isLocalAddress checks if the address is a local/private address.

func isLocalAddress(host string) bool {
	// Extract hostname from host:port.

	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		hostname = host
	}

	// Check common local hostnames.

	localHosts := []string{
		"localhost",

		"127.0.0.1",

		"::1",

		"0.0.0.0",
	}

	for _, localHost := range localHosts {
		if hostname == localHost {
			return true
		}
	}

	// Parse IP and check if it's private.

	ip := net.ParseIP(hostname)

	if ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
	}

	return false
}

// SecureHTTPServer creates an HTTP server with security best practices.

func SecureHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr: addr,

		Handler: NewSecurityHeadersMiddleware(false).Middleware(handler),

		// Timeouts to prevent DoS attacks.

		ReadTimeout: 15 * time.Second,

		ReadHeaderTimeout: 10 * time.Second,

		WriteTimeout: 15 * time.Second,

		IdleTimeout: 60 * time.Second,

		// Limits.

		MaxHeaderBytes: 1 << 20, // 1 MB

	}
}

// BasicSecurityHeaders adds basic security headers to HTTP responses.

func BasicSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers (OWASP recommendations).

		w.Header().Set("X-Content-Type-Options", "nosniff")

		w.Header().Set("X-Frame-Options", "DENY")

		w.Header().Set("X-XSS-Protection", "1; mode=block")

		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")

		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware implements basic rate limiting.

type RateLimiter struct {
	requests map[string][]time.Time

	limit int

	window time.Duration
}

// NewRateLimiter creates a new rate limiter.

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),

		limit: limit,

		window: window,
	}
}

// RateLimitMiddleware creates a rate limiting middleware.

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP.

		clientIP := r.RemoteAddr

		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = forwarded
		}

		now := time.Now()

		// Clean old requests.

		if requests, exists := rl.requests[clientIP]; exists {

			var valid []time.Time

			for _, t := range requests {
				if now.Sub(t) < rl.window {
					valid = append(valid, t)
				}
			}

			rl.requests[clientIP] = valid

		}

		// Check rate limit.

		if len(rl.requests[clientIP]) >= rl.limit {

			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)

			return

		}

		// Add current request.

		rl.requests[clientIP] = append(rl.requests[clientIP], now)

		next.ServeHTTP(w, r)
	})
}

// SecureRequest creates a secure HTTP request with context.

func SecureRequest(ctx context.Context, method, url string, timeout time.Duration) (*http.Request, error) {
	// Validate URL.

	validURL, err := ValidateURL(url)
	if err != nil {
		return nil, err
	}

	// Create request with context.

	if timeout > 0 {

		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, timeout)

		defer cancel()

	}

	req, err := http.NewRequestWithContext(ctx, method, validURL.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set security headers.

	req.Header.Set("User-Agent", "Nephoran-Security-Client/1.0")

	req.Header.Set("Accept", "application/json")

	return req, nil
}

// SafeCloseBody safely closes an HTTP response body with error handling.

func SafeCloseBody(resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}

	return resp.Body.Close()
}
