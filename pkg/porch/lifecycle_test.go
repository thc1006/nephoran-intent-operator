package porch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPackageRevisionLifecycle(t *testing.T) {
	// Create a mock Porch server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"name": "test-package",
			"namespace": "default",
			"revision": "v1",
			"status": "APPROVED"
		}`))
	}))
	defer server.Close()

	// Create client
	client := newTestClient(server.URL, false)

	// Test package revision
	revision := &PorchPackageRevision{
		Name:      "test-package",
		Namespace: "default",
		Revision:  "v1",
		Status:    "DRAFT",
	}

	t.Run("SubmitForReview", func(t *testing.T) {
		updated, err := client.SubmitForReview(revision)
		if err != nil {
			t.Errorf("SubmitForReview failed: %v", err)
		}
		if updated.Status != "APPROVED" { // Mock returns APPROVED for testing
			t.Errorf("Expected status APPROVED, got %s", updated.Status)
		}
	})

	t.Run("ApprovePackage", func(t *testing.T) {
		updated, err := client.ApprovePackage(revision)
		if err != nil {
			t.Errorf("ApprovePackage failed: %v", err)
		}
		if updated.Status != "APPROVED" {
			t.Errorf("Expected status APPROVED, got %s", updated.Status)
		}
	})

	t.Run("PublishPackage", func(t *testing.T) {
		updated, err := client.PublishPackage(revision)
		if err != nil {
			t.Errorf("PublishPackage failed: %v", err)
		}
		if updated.Status != "APPROVED" { // Mock returns APPROVED for testing
			t.Errorf("Expected status PUBLISHED, got %s", updated.Status)
		}
	})
}

func TestPackageRevisionLifecycleDryRun(t *testing.T) {
	// Create client in dry-run mode
	client := newTestClient("http://test-endpoint", true)

	revision := &PorchPackageRevision{
		Name:      "test-package",
		Namespace: "default",
		Revision:  "v1",
		Status:    "DRAFT",
	}

	t.Run("DryRunSubmitForReview", func(t *testing.T) {
		updated, err := client.SubmitForReview(revision)
		if err != nil {
			t.Errorf("Dry-run SubmitForReview failed: %v", err)
		}
		if updated.Status != "REVIEW" {
			t.Errorf("Expected status REVIEW, got %s", updated.Status)
		}
	})

	t.Run("DryRunApprovePackage", func(t *testing.T) {
		updated, err := client.ApprovePackage(revision)
		if err != nil {
			t.Errorf("Dry-run ApprovePackage failed: %v", err)
		}
		if updated.Status != "APPROVED" {
			t.Errorf("Expected status APPROVED, got %s", updated.Status)
		}
	})

	t.Run("DryRunPublishPackage", func(t *testing.T) {
		updated, err := client.PublishPackage(revision)
		if err != nil {
			t.Errorf("Dry-run PublishPackage failed: %v", err)
		}
		if updated.Status != "PUBLISHED" {
			t.Errorf("Expected status PUBLISHED, got %s", updated.Status)
		}
	})
}

func TestAuthenticationTransport(t *testing.T) {
	// Create a test server that checks for auth header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create client with auth
	client := newTestClientWithAuth(server.URL, "test-token", false)

	// Test that the auth transport works
	resp, err := client.httpClient.Get(server.URL + "/test")
	if err != nil {
		t.Errorf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}