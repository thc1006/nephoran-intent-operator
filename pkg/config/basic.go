//go:build fast_build
// +build fast_build

package config

import (
	"os"
	"strconv"
	"time"
)

// BasicConfig provides minimal configuration for fast builds
type BasicConfig struct {
	Port      int           `json:"port"`
	LogLevel  string        `json:"log_level"`
	Timeout   time.Duration `json:"timeout"`
	EnableTLS bool          `json:"enable_tls"`
}

// LoadBasicConfig loads minimal configuration for fast builds
func LoadBasicConfig() (*BasicConfig, error) {
	cfg := &BasicConfig{
		Port:      8080,
		LogLevel:  "info",
		Timeout:   30 * time.Second,
		EnableTLS: false,
	}

	// Override with environment variables if present
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	if timeout := os.Getenv("TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			cfg.Timeout = t
		}
	}

	if tls := os.Getenv("ENABLE_TLS"); tls != "" {
		if t, err := strconv.ParseBool(tls); err == nil {
			cfg.EnableTLS = t
		}
	}

	return cfg, nil
}
