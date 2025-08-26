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

// YANGValidator defines the interface for YANG model validation
type YANGValidator interface {
	// ValidatePackageRevision validates a package revision against YANG models
	ValidatePackageRevision(ctx context.Context, pkg *porch.PackageRevision) (*ValidationResult, error)
}

// ValidatorMetrics implements metrics tracking for YANG validation processes
type ValidatorMetrics struct {
	ValidationsTotal   prometheus.Counter
	ValidationDuration prometheus.Histogram
	ValidationErrors   *prometheus.CounterVec
	ModelsLoaded       prometheus.Gauge
	CacheHitRate       prometheus.Gauge
	ActiveValidations  prometheus.Gauge
}

// Create a new ValidatorMetrics instance
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

// Custom type for package revision resource
type PackageResource struct {
	Kind string
	Data interface{}
}

// Structure definitions
type ValidationResult struct {
	Valid             bool
	ModelName         string
	ValidationTime    time.Time
	Errors            []*ValidationError
}

type ValidationError struct {
	Code    string
	Message string
}

// yangValidator implements the YANGValidator interface
type yangValidator struct {
	metrics *ValidatorMetrics
}

// Additional modifications for resource validation
func (v *yangValidator) ValidatePackageRevision(ctx context.Context, pkg *porch.PackageRevision) (*ValidationResult, error) {
	result := &ValidationResult{
		ModelName:      pkg.ObjectMeta.Name,
		ValidationTime: time.Now(),
	}

	configData := make(map[string]interface{})

	for _, resource := range pkg.Spec.Resources {
		// Process different resource types
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

	// Mark result as valid if no errors
	result.Valid = len(result.Errors) == 0

	return result, nil
}