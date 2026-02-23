package oran

import (
	"net"
	"net/http"
	"time"
)

// TLSConfig holds TLS configuration for O-RAN interfaces.

type TLSConfig struct {
	CertFile string

	KeyFile string

	CAFile string

	SkipVerify bool
}

// O1Config defines configuration for O1 interface.

type O1Config struct {
	Endpoint string `json:"endpoint"`

	Timeout time.Duration `json:"timeout"`

	RetryAttempts int `json:"retryAttempts"`

	DefaultPort int `json:"defaultPort"`

	ConnectTimeout time.Duration `json:"connectTimeout"`

	RequestTimeout time.Duration `json:"requestTimeout"`

	MaxRetries int `json:"maxRetries"`

	RetryInterval time.Duration `json:"retryInterval"`

	TLSConfig *TLSConfig `json:"tlsConfig,omitempty"`

	Authentication *AuthConfig `json:"authentication,omitempty"`
}

// SecurityPolicy defines security policy for O1 interface.

type SecurityPolicy struct {
	TLSVersion uint16 `json:"tlsVersion"`

	CipherSuites []string `json:"cipherSuites"`

	RequireClientAuth bool `json:"requireClientAuth"`

	MaxConnections int `json:"maxConnections"`

	RateLimit *RateLimit `json:"rateLimit,omitempty"`
}

// StreamFilter defines filtering configuration for O1 streaming.

type StreamFilter struct {
	MetricNames []string `json:"metricNames,omitempty"`

	TimeRange *TimeRange `json:"timeRange,omitempty"`

	Filters map[string]string `json:"filters,omitempty"`
}

// RateLimit defines rate limiting configuration.

type RateLimit struct {
	RequestsPerSecond float64 `json:"requestsPerSecond"`

	Burst int `json:"burst"`

	TimeWindow time.Duration `json:"timeWindow"`
}

// TimeRange defines a time range filter.

type TimeRange struct {
	Start time.Time `json:"start"`

	End time.Time `json:"end"`
}

// AuthConfig defines authentication configuration.

type AuthConfig struct {
	Type string `json:"type"` // cert, token, basic

	Username string `json:"username,omitempty"`

	Password string `json:"password,omitempty"`

	Token string `json:"token,omitempty"`

	CertFile string `json:"certFile,omitempty"`

	KeyFile string `json:"keyFile,omitempty"`
}

// RateLimiter interface for rate limiting.

type RateLimiter interface {
	Allow() bool

	AllowN(n int) bool

	Wait() error
}

// EmailTemplate defines email template structure.

type EmailTemplate struct {
	Subject string `json:"subject"`

	Body string `json:"body"`

	Headers map[string]string `json:"headers,omitempty"`

	MimeType string `json:"mimeType,omitempty"`
}

// Client provides a generic O-RAN client interface.

type Client struct {
	baseURL string

	httpClient *http.Client

	auth *AuthConfig

	headers map[string]string
}

// NewClient creates a new O-RAN client.

func NewClient(baseURL string, auth *AuthConfig) *Client {
	return &Client{
		baseURL: baseURL,

		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},

		auth: auth,

		headers: make(map[string]string),
	}
}
