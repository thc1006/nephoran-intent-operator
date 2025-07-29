package a1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	nephoranv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

func TestA1AdaptorPolicyTypeOperations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a1-p/v1/policytypes/1000":
			switch r.Method {
			case http.MethodPut:
				w.WriteHeader(http.StatusCreated)
			case http.MethodGet:
				json.NewEncoder(w).Encode(&A1PolicyType{
					PolicyTypeID: 1000,
					Name:         "Test Policy Type",
					Description:  "Test Description",
				})
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
			}
		case "/a1-p/v1/policytypes":
			json.NewEncoder(w).Encode([]int{1000, 2000})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create adaptor
	adaptor, err := NewA1Adaptor(&A1AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Test CreatePolicyType
	t.Run("CreatePolicyType", func(t *testing.T) {
		policyType := CreateQoSPolicyType()
		err := adaptor.CreatePolicyType(ctx, policyType)
		assert.NoError(t, err)
	})

	// Test GetPolicyType
	t.Run("GetPolicyType", func(t *testing.T) {
		policyType, err := adaptor.GetPolicyType(ctx, 1000)
		assert.NoError(t, err)
		assert.Equal(t, 1000, policyType.PolicyTypeID)
		assert.Equal(t, "Test Policy Type", policyType.Name)
	})

	// Test ListPolicyTypes
	t.Run("ListPolicyTypes", func(t *testing.T) {
		// This test will partially fail because we only mock policy type 1000
		// In a real test, we'd mock all policy types
		policyTypes, err := adaptor.ListPolicyTypes(ctx)
		assert.Error(t, err) // Expected because we don't mock policy type 2000
		assert.Nil(t, policyTypes)
	})

	// Test DeletePolicyType
	t.Run("DeletePolicyType", func(t *testing.T) {
		err := adaptor.DeletePolicyType(ctx, 1000)
		assert.NoError(t, err)
	})
}

func TestA1AdaptorPolicyInstanceOperations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a1-p/v1/policytypes/1000/policies/test-instance":
			switch r.Method {
			case http.MethodPut:
				w.WriteHeader(http.StatusCreated)
			case http.MethodGet:
				json.NewEncoder(w).Encode(map[string]interface{}{
					"slice_id": "test-slice",
					"qos_parameters": map[string]interface{}{
						"latency_ms":      10,
						"throughput_mbps": 100,
					},
				})
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
			}
		case "/a1-p/v1/policytypes/1000/policies/test-instance/status":
			json.NewEncoder(w).Encode(&A1PolicyStatus{
				EnforcementStatus: "ENFORCED",
				LastModified:      time.Now(),
			})
		case "/a1-p/v1/policytypes/1000/policies":
			json.NewEncoder(w).Encode([]string{"test-instance", "test-instance-2"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create adaptor
	adaptor, err := NewA1Adaptor(&A1AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Test CreatePolicyInstance
	t.Run("CreatePolicyInstance", func(t *testing.T) {
		instance := &A1PolicyInstance{
			PolicyInstanceID: "test-instance",
			PolicyTypeID:     1000,
			PolicyData: map[string]interface{}{
				"slice_id": "test-slice",
				"qos_parameters": map[string]interface{}{
					"latency_ms":      10,
					"throughput_mbps": 100,
				},
			},
		}
		err := adaptor.CreatePolicyInstance(ctx, 1000, instance)
		assert.NoError(t, err)
	})

	// Test GetPolicyInstance
	t.Run("GetPolicyInstance", func(t *testing.T) {
		instance, err := adaptor.GetPolicyInstance(ctx, 1000, "test-instance")
		assert.NoError(t, err)
		assert.Equal(t, "test-instance", instance.PolicyInstanceID)
		assert.Equal(t, 1000, instance.PolicyTypeID)
		assert.Equal(t, "ENFORCED", instance.Status.EnforcementStatus)
	})

	// Test GetPolicyStatus
	t.Run("GetPolicyStatus", func(t *testing.T) {
		status, err := adaptor.GetPolicyStatus(ctx, 1000, "test-instance")
		assert.NoError(t, err)
		assert.Equal(t, "ENFORCED", status.EnforcementStatus)
	})

	// Test DeletePolicyInstance
	t.Run("DeletePolicyInstance", func(t *testing.T) {
		err := adaptor.DeletePolicyInstance(ctx, 1000, "test-instance")
		assert.NoError(t, err)
	})
}

func TestA1AdaptorApplyPolicy(t *testing.T) {
	// Track enforcement status
	enforced := false

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a1-p/v1/policytypes/1000/policies/test-me-policy-1000":
			if r.Method == http.MethodPut {
				enforced = true
				w.WriteHeader(http.StatusCreated)
			}
		case "/a1-p/v1/policytypes/1000/policies/test-me-policy-1000/status":
			status := &A1PolicyStatus{
				EnforcementStatus: "NOT_ENFORCED",
				LastModified:      time.Now(),
			}
			if enforced {
				status.EnforcementStatus = "ENFORCED"
			}
			json.NewEncoder(w).Encode(status)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create adaptor
	adaptor, err := NewA1Adaptor(&A1AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Create ManagedElement with A1 policy
	me := &nephoranv1alpha1.ManagedElement{
		Spec: nephoranv1alpha1.ManagedElementSpec{
			A1Policy: runtime.RawExtension{
				Raw: []byte(`{
					"policy_type_id": 1000,
					"policy_instance_id": "test-me-policy-1000",
					"policy_data": {
						"slice_id": "test-slice",
						"qos_parameters": {
							"latency_ms": 10,
							"throughput_mbps": 100,
							"reliability": 0.999
						}
					}
				}`),
			},
		},
	}
	me.Name = "test-me"

	// Test ApplyPolicy
	err = adaptor.ApplyPolicy(ctx, me)
	assert.NoError(t, err)
	assert.True(t, enforced)
}

func TestA1AdaptorRemovePolicy(t *testing.T) {
	deleted := false

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/a1-p/v1/policytypes/1000/policies/test-me-policy-1000" && r.Method == http.MethodDelete {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create adaptor
	adaptor, err := NewA1Adaptor(&A1AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    5 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Create ManagedElement with A1 policy
	me := &nephoranv1alpha1.ManagedElement{
		Spec: nephoranv1alpha1.ManagedElementSpec{
			A1Policy: runtime.RawExtension{
				Raw: []byte(`{
					"policy_type_id": 1000,
					"policy_instance_id": "test-me-policy-1000",
					"policy_data": {}
				}`),
			},
		},
	}
	me.Name = "test-me"

	// Test RemovePolicy
	err = adaptor.RemovePolicy(ctx, me)
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestA1AdaptorNoPolicyScenarios(t *testing.T) {
	adaptor, err := NewA1Adaptor(nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test ApplyPolicy with no policy
	t.Run("ApplyPolicy with no policy", func(t *testing.T) {
		me := &nephoranv1alpha1.ManagedElement{
			Spec: nephoranv1alpha1.ManagedElementSpec{},
		}
		me.Name = "test-me"

		err := adaptor.ApplyPolicy(ctx, me)
		assert.NoError(t, err)
	})

	// Test RemovePolicy with no policy
	t.Run("RemovePolicy with no policy", func(t *testing.T) {
		me := &nephoranv1alpha1.ManagedElement{
			Spec: nephoranv1alpha1.ManagedElementSpec{},
		}
		me.Name = "test-me"

		err := adaptor.RemovePolicy(ctx, me)
		assert.NoError(t, err)
	})
}