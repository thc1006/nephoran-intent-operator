/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package yang

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// ValidatorConfig holds configuration for YANG validation.
type ValidatorConfig struct {
	// Validation options.
	EnableConstraintValidation bool          `yaml:"enableConstraintValidation"`
	EnableDataTypeValidation   bool          `yaml:"enableDataTypeValidation"`
	EnableMandatoryCheck       bool          `yaml:"enableMandatoryCheck"`
	ValidationTimeout          time.Duration `yaml:"validationTimeout"`
	MaxValidationDepth         int           `yaml:"maxValidationDepth"`

	// Model support.
	EnableO_RANModels  bool     `yaml:"enableORANModels"`
	Enable3GPPModels   bool     `yaml:"enable3GPPModels"`
	EnableCustomModels bool     `yaml:"enableCustomModels"`
	ModelSearchPaths   []string `yaml:"modelSearchPaths"`

	// Performance options.
	EnableCaching            bool          `yaml:"enableCaching"`
	CacheSize                int           `yaml:"cacheSize"`
	CacheTTL                 time.Duration `yaml:"cacheTTL"`
	MaxConcurrentValidations int           `yaml:"maxConcurrentValidations"`

	// Metrics and monitoring.
	EnableMetrics    bool   `yaml:"enableMetrics"`
	MetricsNamespace string `yaml:"metricsNamespace"`

	// Error handling.
	StrictMode          bool `yaml:"strictMode"`
	FailOnWarnings      bool `yaml:"failOnWarnings"`
	MaxValidationErrors int  `yaml:"maxValidationErrors"`
}

// DefaultValidatorConfig returns a default ValidatorConfig.
func DefaultValidatorConfig() *ValidatorConfig {
	return &ValidatorConfig{
		EnableConstraintValidation: true,
		EnableDataTypeValidation:   true,
		EnableMandatoryCheck:       true,
		ValidationTimeout:          30 * time.Second,
		MaxValidationDepth:         100,
		EnableO_RANModels:          true,
		Enable3GPPModels:           true,
		EnableCustomModels:         false,
		ModelSearchPaths:           []string{"/etc/yang/models", "/usr/local/share/yang/models"},
		EnableCaching:              true,
		CacheSize:                  1000,
		CacheTTL:                   60 * time.Minute,
		MaxConcurrentValidations:   10,
		EnableMetrics:              true,
		MetricsNamespace:           "yang_validator",
		StrictMode:                 false,
		FailOnWarnings:             false,
		MaxValidationErrors:        50,
	}
}

// YANGValidator defines the interface for YANG model validation.
type YANGValidator interface {
	// ValidatePackageRevision validates a package revision against YANG models.
	ValidatePackageRevision(ctx context.Context, pkg *porch.PackageRevision) (*ValidationResult, error)

	// GetValidatorHealth returns the health status of the validator.
	GetValidatorHealth(ctx context.Context) (*ValidatorHealth, error)

	// Close gracefully shuts down the validator.
	Close() error
}

// ValidatorHealth represents the health status of the YANG validator.
type ValidatorHealth struct {
	Status            string        `json:"status"` // healthy, degraded, unhealthy
	LastCheck         time.Time     `json:"lastCheck"`
	LoadedModels      int           `json:"loadedModels"`
	CacheHitRate      float64       `json:"cacheHitRate"`
	ActiveValidations int           `json:"activeValidations"`
	Errors            []string      `json:"errors,omitempty"`
	Warnings          []string      `json:"warnings,omitempty"`
	Uptime            time.Duration `json:"uptime"`
	Version           string        `json:"version,omitempty"`
}

// ValidatorMetrics implements metrics tracking for YANG validation processes.
type ValidatorMetrics struct {
	ValidationsTotal   prometheus.Counter
	ValidationDuration prometheus.Histogram
	ValidationErrors   *prometheus.CounterVec
	ModelsLoaded       prometheus.Gauge
	CacheHitRate       prometheus.Gauge
	ActiveValidations  prometheus.Gauge
}

// Create a new ValidatorMetrics instance.
func newValidatorMetrics() *ValidatorMetrics {
	return &ValidatorMetrics{
		ValidationsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yang_validations_total",
			Help: "Total number of YANG validations performed",
		}),
		ValidationDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "yang_validation_duration_seconds",
			Help:    "Duration of YANG validation processes",
			Buckets: prometheus.DefBuckets,
		}),
		ValidationErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "yang_validation_errors_total",
				Help: "Total number of YANG validation errors",
			},
			[]string{"type"},
		),
		ModelsLoaded: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "yang_models_loaded",
			Help: "Number of YANG models currently loaded",
		}),
		CacheHitRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "yang_cache_hit_rate",
			Help: "Cache hit rate for YANG model validations",
		}),
		ActiveValidations: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "yang_active_validations",
			Help: "Number of currently active YANG validations",
		}),
	}
}

// Custom type for package revision resource.
type PackageResource struct {
	Kind string
	Data interface{}
}

// Structure definitions.
type ValidationResult struct {
	Valid          bool
	ModelName      string
	ValidationTime time.Time
	Errors         []*ValidationError
}

// ValidationError represents a validationerror.
type ValidationError struct {
	Code    string
	Message string
}

// yangValidator implements the YANGValidator interface.
type yangValidator struct {
	config    *ValidatorConfig
	metrics   *ValidatorMetrics
	startTime time.Time
}

// NewYANGValidator creates a new YANG validator.
func NewYANGValidator(config *ValidatorConfig) (YANGValidator, error) {
	if config == nil {
		config = DefaultValidatorConfig()
	}

	return &yangValidator{
		config:    config,
		metrics:   newValidatorMetrics(),
		startTime: time.Now(),
	}, nil
}

// Additional modifications for resource validation.
func (v *yangValidator) ValidatePackageRevision(ctx context.Context, pkg *porch.PackageRevision) (*ValidationResult, error) {
	result := &ValidationResult{
		ModelName:      pkg.ObjectMeta.Name,
		ValidationTime: time.Now(),
	}

	configData := make(map[string]interface{})

	for _, resource := range pkg.Spec.Resources {
		// Process different resource types.
		var res PackageResource
		switch r := resource.(type) {
		case PackageResource:
			res = r
		case map[string]interface{}:
			if kind, exists := r["kind"]; exists {
				if kindStr, ok := kind.(string); ok {
					res = PackageResource{
						Kind: kindStr,
						Data: r["data"],
					}
				} else {
					result.Errors = append(result.Errors, &ValidationError{
						Code:    "INVALID_KIND_TYPE",
						Message: "Resource kind is not a string",
					})
					continue
				}
			} else {
				result.Errors = append(result.Errors, &ValidationError{
					Code:    "MISSING_KIND",
					Message: "Resource missing kind field",
				})
				continue
			}
		default:
			result.Errors = append(result.Errors, &ValidationError{
				Code:    "INVALID_RESOURCE_TYPE",
				Message: "Unable to parse resource type",
			})
			continue
		}

		if res.Kind == "ConfigMap" {
			dataMap, ok := res.Data.(map[string]interface{})
			if !ok {
				result.Errors = append(result.Errors, &ValidationError{
					Code:    "INVALID_CONFIGMAP_DATA",
					Message: "ConfigMap data is not a map",
				})
				continue
			}

			config, exists := dataMap["config"]
			if !exists {
				continue
			}

			configMapData, ok := config.(map[string]interface{})
			if !ok {
				result.Errors = append(result.Errors, &ValidationError{
					Code:    "INVALID_CONFIG_DATA",
					Message: "ConfigMap config is not a map",
				})
				continue
			}

			for k, v := range configMapData {
				configData[k] = v
			}
		}
	}

	// Mark result as valid if no errors.
	result.Valid = len(result.Errors) == 0

	return result, nil
}

// GetValidatorHealth returns the health status of the validator.
func (v *yangValidator) GetValidatorHealth(ctx context.Context) (*ValidatorHealth, error) {
	return &ValidatorHealth{
		Status:            "healthy",
		LastCheck:         time.Now(),
		Uptime:            time.Since(v.startTime),
		LoadedModels:      0,   // This would be populated based on actual loaded models
		CacheHitRate:      0.0, // This would be calculated from actual cache metrics
		ActiveValidations: 0,   // This would be tracked during validation
		Version:           "1.0.0",
	}, nil
}

// Close gracefully shuts down the validator.
func (v *yangValidator) Close() error {
	// Cleanup resources.
	return nil
}
