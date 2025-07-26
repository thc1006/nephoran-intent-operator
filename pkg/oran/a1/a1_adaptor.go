package a1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	nephoranv1alpha1 "nephoran-intent-operator/pkg/apis/nephoran/v1alpha1"
)

// A1Adaptor is responsible for handling A1 interface communications.
type A1Adaptor struct {
	Client *http.Client
}

// NewA1Adaptor creates a new A1Adaptor.
func NewA1Adaptor() *A1Adaptor {
	return &A1Adaptor{
		Client: &http.Client{},
	}
}

// Policy represents a simple A1 policy.
type Policy struct {
	PolicyID   string      `json:"policy_id"`
	PolicyData interface{} `json:"policy_data"`
}

// ApplyPolicy sends a policy to the Near-RT RIC.
func (a *A1Adaptor) ApplyPolicy(ctx context.Context, me *nephoranv1alpha1.ManagedElement, policyType string, policyData interface{}) error {
	if me.Spec.A1.PolicyType == "" {
		return fmt.Errorf("A1 configuration for ManagedElement %s is incomplete", me.Name)
	}

	// In a real implementation, the policy ID and endpoint would be more dynamic.
	policyID := fmt.Sprintf("policy-%s-%s", me.Name, policyType)
	ricURL := "http://near-rt-ric.oran.svc.cluster.local:8080/a1/policy"

	policy := Policy{
		PolicyID:   policyID,
		PolicyData: policyData,
	}

	body, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal A1 policy for %s: %w", me.Name, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, ricURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create A1 request for %s: %w", me.Name, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send A1 policy for %s: %w", me.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to apply A1 policy for %s: received status code %d", me.Name, resp.StatusCode)
	}

	return nil
}
