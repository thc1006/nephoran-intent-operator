package test_validation

import (
	"encoding/json"
	"fmt"
	"time"
	
	vegeta "github.com/tsenart/vegeta/lib"
)

// AdvancedLoadGenerator provides advanced load generation capabilities.
type AdvancedLoadGenerator struct {
	// Add load generator configuration fields
}

// ConstantRateScenario represents a constant rate load scenario.
type ConstantRateScenario struct {
	Rate     int
	Duration time.Duration
}

// RampUpScenario represents a ramp-up load scenario.
type RampUpScenario struct {
	StartRate    int
	EndRate      int
	Duration     time.Duration
	StepDuration time.Duration
}

// SpikeScenario represents a spike load scenario.
type SpikeScenario struct {
	BaseRate   int
	SpikeRate  int
	SpikeDuration time.Duration
	Name       string
}

// LoadRequest represents a load test request.
type LoadRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
}

// Rest of the file remains the same, but these specific methods are corrected

func (alg *AdvancedLoadGenerator) generateVegetaTargets() []vegeta.Target {
	targets := []vegeta.Target{}

	intents := []string{
		"Deploy AMF with high availability",
		"Configure SMF with QoS policies",
		"Setup UPF for edge deployment",
		"Create network slice for IoT",
		"Deploy Near-RT RIC",
	}

	for i, _ := range intents {
		// Corrected JSON marshaling
		body := map[string]interface{}{
			"metadata": map[string]string{
				"name": fmt.Sprintf("vegeta-test-%d", i),
				"namespace": "default",
			},
			"spec": map[string]interface{}{},
		}

		bodyBytes, _ := json.Marshal(body)

		targets = append(targets, vegeta.Target{
			Method: "POST",
			URL: fmt.Sprintf("http://localhost:8080/api/v1/intents"),
			Body: bodyBytes,
			Header: map[string][]string{
				"Content-Type": {"application/json"},
			},
		})
	}

	return targets
}

func (crs *ConstantRateScenario) generateIntentBody(id int) []byte {
	// Corrected JSON marshaling
	body := map[string]interface{}{
		"metadata": map[string]string{
			"name": fmt.Sprintf("constant-rate-%d", id),
			"namespace": "default",
		},
		"spec": map[string]interface{}{},
	}

	bodyBytes, _ := json.Marshal(body)
	return bodyBytes
}

func (rus *RampUpScenario) generateIntentBody(id int) []byte {
	// Corrected JSON marshaling
	body := map[string]interface{}{
		"metadata": map[string]string{
			"name": fmt.Sprintf("ramp-up-%d", id),
			"namespace": "default",
		},
		"spec": map[string]interface{}{},
	}

	bodyBytes, _ := json.Marshal(body)
	return bodyBytes
}

func (ss *SpikeScenario) generateRequest(id, rate int) *LoadRequest {
	// Corrected JSON marshaling
	body := map[string]interface{}{
		"metadata": map[string]string{
			"name": fmt.Sprintf("spike-%d-rate-%d", id, rate),
			"namespace": "default",
		},
		"spec": map[string]interface{}{},
	}

	bodyBytes, _ := json.Marshal(body)

	return &LoadRequest{
		Method: "POST",
		URL: "http://localhost:8080/api/v1/intents",
		Body: bodyBytes,
		Headers: map[string]string{"Content-Type": "application/json"},
	}
}