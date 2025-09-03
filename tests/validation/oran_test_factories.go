// Package test_validation provides comprehensive test data factories for O-RAN interface testing.
package test_validation

import (
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nephranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// ORANTestFactory provides factory methods for creating test data.
type ORANTestFactory struct {
	nameCounter int
	timeBase    time.Time
}

// NewORANTestFactory creates a new instance of ORANTestFactory.
func NewORANTestFactory() *ORANTestFactory {
	return &ORANTestFactory{
		nameCounter: 0,
		timeBase:    time.Now(),
	}
}

// GetNextName generates a unique name with a prefix
func (otf *ORANTestFactory) GetNextName(prefix string) string {
	otf.nameCounter++
	return fmt.Sprintf("%s-%d", prefix, otf.nameCounter)
}

// CreateE2Node creates a test E2 node configuration.
func (otf *ORANTestFactory) CreateE2Node(nodeType string) *E2Node {
	return &E2Node{
		NodeID:   fmt.Sprintf("%s-node-001", nodeType),
		NodeType: nodeType,
		Status:   "CONNECTED",
	}
}

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

	policyDataBytes, _ := json.Marshal(policyData)
	
	return &A1Policy{
		PolicyID:      otf.GetNextName("policy"),
		PolicyTypeID:  policyType,
		PolicyData:    json.RawMessage(policyDataBytes),
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

	configDataBytes, _ := json.Marshal(configData)
	
	return &XAppConfig{
		Name:       otf.GetNextName("xapp-" + xappType),
		Version:    "1.0.0",
		ConfigData: json.RawMessage(configDataBytes),
		Status:     "RUNNING",
		DeployedAt: otf.timeBase.Add(time.Duration(otf.nameCounter) * time.Minute),
	}
}

// CreateManagedElement creates a test O1 managed element
func (otf *ORANTestFactory) CreateManagedElement(elementType string) *ManagedElement {
	return &ManagedElement{
		ElementID:     fmt.Sprintf("%s-element-%d", elementType, otf.nameCounter+1),
		ElementType:   elementType,
		Configuration: json.RawMessage(`{}`),
		Status:        "ACTIVE",
		LastSync:      time.Now(),
	}
}

// CreateO1Configuration creates a test O1 configuration
func (otf *ORANTestFactory) CreateO1Configuration(configType, elementID string) *O1Configuration {
	otf.nameCounter++
	return &O1Configuration{
		ConfigID:   fmt.Sprintf("%s-config-%d", configType, otf.nameCounter),
		ElementID:  elementID,
		ConfigType: configType,
		ConfigData: json.RawMessage(`{}`),
		Version:    1,
	}
}

// CreateE2Subscription creates a test E2 subscription
func (otf *ORANTestFactory) CreateE2Subscription(nodeID, serviceModel string) *E2Subscription {
	otf.nameCounter++
	return &E2Subscription{
		SubscriptionID: fmt.Sprintf("sub-%d", otf.nameCounter),
		NodeID:         nodeID,
		ServiceModel:   serviceModel,
		EventTrigger:   json.RawMessage(`{}`),
		Actions:        []E2Action{},
		Status:         "ACTIVE",
		CreatedAt:      time.Now(),
	}
}