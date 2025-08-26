package o1

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

func TestNewO1Adaptor(t *testing.T) {
	logger := testr.New(t)
	config := O1AdaptorConfig{
		Timeout:    10 * time.Second,
		RetryCount: 2,
	}
	
	adapter := NewO1Adaptor(logger, config)
	
	if adapter == nil {
		t.Fatal("NewO1Adaptor returned nil")
	}
	
	if adapter.timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", adapter.timeout)
	}
	
	if adapter.retryCount != 2 {
		t.Errorf("Expected retry count 2, got %d", adapter.retryCount)
	}
}

func TestO1Adaptor_ConnectElement(t *testing.T) {
	logger := testr.New(t)
	adapter := NewO1Adaptor(logger, O1AdaptorConfig{})
	
	element := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-element",
			Namespace: "default",
		},
		Spec: nephoranv1.ManagedElementSpec{
			NetworkElementType: "DU",
			Endpoint:          "http://test-endpoint:8080",
		},
	}
	
	ctx := context.Background()
	err := adapter.ConnectElement(ctx, element)
	
	if err != nil {
		t.Errorf("ConnectElement failed: %v", err)
	}
}

func TestO1Adaptor_ConnectElement_NoEndpoint(t *testing.T) {
	logger := testr.New(t)
	adapter := NewO1Adaptor(logger, O1AdaptorConfig{})
	
	element := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-element",
			Namespace: "default",
		},
		Spec: nephoranv1.ManagedElementSpec{
			NetworkElementType: "DU",
			// No endpoint specified
		},
	}
	
	ctx := context.Background()
	err := adapter.ConnectElement(ctx, element)
	
	if err == nil {
		t.Error("Expected error for missing endpoint, got nil")
	}
}

func TestO1Adaptor_GetElementStatus(t *testing.T) {
	logger := testr.New(t)
	adapter := NewO1Adaptor(logger, O1AdaptorConfig{})
	
	element := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-element",
			Namespace: "default",
		},
		Spec: nephoranv1.ManagedElementSpec{
			NetworkElementType: "DU",
			Endpoint:          "http://test-endpoint:8080",
		},
	}
	
	ctx := context.Background()
	status, err := adapter.GetElementStatus(ctx, element)
	
	if err != nil {
		t.Errorf("GetElementStatus failed: %v", err)
	}
	
	if status == nil {
		t.Fatal("GetElementStatus returned nil status")
	}
	
	if status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", status.Phase)
	}
	
	if status.ConnectionStatus != "Connected" {
		t.Errorf("Expected connection status 'Connected', got '%s'", status.ConnectionStatus)
	}
}