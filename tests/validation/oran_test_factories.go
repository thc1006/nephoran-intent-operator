// Package test_validation provides comprehensive test data factories for O-RAN interface testing.
package test_validation

import (
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nephranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// CreateA1Policy creates a test A1 policy with strongly typed handling.
func (otf *ORANTestFactory) CreateA1Policy(policyType string) *A1Policy {
	var policyData map[string]any

	switch policyType {
	case "traffic-steering":
		policyData = map[string]any{
			"primaryPathWeight":   0.7,
			"secondaryPathWeight": 0.3,
		}

	case "qos-optimization":
		policyData = map[string]any{
			"targetLatency":     10,
			"targetThroughput":  1000, // Mbps
			"priorityClass":     "high",
		}

	case "admission-control":
		policyData = map[string]any{
			"maxUEPerCell":     1000,
			"resourceReserved": 0.2,
			"qosProfiles":      []string{"URLLC", "eMBB"},
		}

	case "energy-saving":
		policyData = map[string]any{
			"timeWindow": map[string]any{
				"start": "22:00",
				"end":   "05:00",
			},
			"powerReductionTarget": 0.2,
		}

	default:
		policyData = map[string]any{
			"defaultPolicy": true,
		}
	}

	return &A1Policy{
		PolicyID:      otf.GetNextName("policy"),
		PolicyTypeID:  policyType,
		PolicyData:    policyData,
		Status:        "ACTIVE",
		CreatedAt:     otf.timeBase.Add(time.Duration(otf.nameCounter) * time.Second),
		UpdatedAt:     otf.timeBase.Add(time.Duration(otf.nameCounter) * time.Second),
	}
}

// CreateXAppConfig creates a strongly typed xApp configuration.
func (otf *ORANTestFactory) CreateXAppConfig(xappType string) *XAppConfig {
	var configData map[string]any

	switch xappType {
	case "qoe-optimizer":
		configData = map[string]any{
			"adaptationThreshold": 0.1,
			"metrics": []string{"latency", "jitter", "throughput"},
		}

	case "load-balancer":
		configData = map[string]any{
			"loadBalancingAlgorithm": "weighted-round-robin",
			"weightingFactor":        map[string]float64{
				"cell-1": 1.2,
				"cell-2": 0.8,
			},
		}

	case "anomaly-detector":
		configData = map[string]any{
			"alertThreshold":   0.95,
			"sensitivityLevel": "high",
			"detectionMode":    "adaptive",
		}

	case "slice-optimizer":
		configData = map[string]any{
			"optimizationGoal":    "resource-efficiency",
			"rebalanceInterval":   "30s",
			"minSliceResources": map[string]string{
				"cpu":    "100m",
				"memory": "256Mi",
			},
			"sliceProfiles": []string{"eMBB", "URLLC", "MIoT"},
		}

	default:
		configData = map[string]any{
			"defaultConfig": true,
		}
	}

	return &XAppConfig{
		Name:       otf.GetNextName("xapp-" + xappType),
		Version:    "1.0.0",
		ConfigData: configData,
		Status:     "RUNNING",
		DeployedAt: otf.timeBase.Add(time.Duration(otf.nameCounter) * time.Minute),
	}
}