package o1

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewO1Adaptor(t *testing.T) {
	_ = testr.New(t) // logger not needed for compat tests
	config := O1AdaptorConfig{
		Timeout:    10 * time.Second,
		RetryCount: 2,
	}

	adapter := NewO1AdaptorCompat(config)

	if adapter == nil {
		t.Fatal("NewO1Adaptor returned nil")
	}

	if adapter.GetTimeout() != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", adapter.GetTimeout())
	}

	if adapter.GetRetryCount() != 2 {
		t.Errorf("Expected retry count 2, got %d", adapter.GetRetryCount())
	}
}

func TestO1Adaptor_ConnectElement(t *testing.T) {
	_ = testr.New(t) // logger not needed for compat tests
	adapter := NewO1AdaptorCompat(O1AdaptorConfig{})

	element := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-element",
			Namespace: "default",
		},
		Spec: nephoranv1.ManagedElementSpec{
			Type: "DU",
			Host: "test-endpoint",
			Port: 8080,
		},
	}

	ctx := context.Background()
	err := adapter.ConnectElement(ctx, element)
	if err != nil {
		t.Errorf("ConnectElement failed: %v", err)
	}
}

func TestO1Adaptor_ConnectElement_NoEndpoint(t *testing.T) {
	_ = testr.New(t) // logger not needed for compat tests
	adapter := NewO1AdaptorCompat(O1AdaptorConfig{})

	element := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-element",
			Namespace: "default",
		},
		Spec: nephoranv1.ManagedElementSpec{
			Type: "DU",
			// No host specified
		},
	}

	ctx := context.Background()
	err := adapter.ConnectElement(ctx, element)

	if err == nil {
		t.Error("Expected error for missing endpoint, got nil")
	}
}

func TestO1Adaptor_GetElementStatus(t *testing.T) {
	_ = testr.New(t) // logger not needed for compat tests
	adapter := NewO1AdaptorCompat(O1AdaptorConfig{})

	element := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-element",
			Namespace: "default",
		},
		Spec: nephoranv1.ManagedElementSpec{
			Type: "DU",
			Host: "test-endpoint",
			Port: 8080,
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

	if !status.Ready {
<<<<<<< HEAD
		t.Errorf("Expected status Ready to be true, got %v", status.Ready)
=======
		t.Errorf("Expected element to be ready, got ready=%t", status.Ready)
>>>>>>> 952ff111560c6d3fb50e044fd58002e2e0b4d871
	}
}
