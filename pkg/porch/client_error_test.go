package porch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/intent"
)

// TestClient_NetworkErrors tests various network error conditions
func TestClient_NetworkErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) *Client
		testFunc    func(t *testing.T, client *Client)
		expectError string
	}{
		{
			name: "connection refused",
			setupFunc: func(t *testing.T) *Client {
				// Use a port that's not listening
				return NewClient("http://localhost:99999", false)
			},
			testFunc: func(t *testing.T, client *Client) {
				req := &PackageRequest{
					Repository: "test-repo",
					Package:    "test-package",
					Workspace:  "default",
					Namespace:  "default",
					Intent: &intent.NetworkIntent{
						IntentType: "scaling",
						Target:     "test-app",
						Namespace:  "default",
						Replicas:   3,
					},
				}
				_, err := client.CreateOrUpdatePackage(req)
				if err == nil {
					t.Error("Expected connection refused error but got nil")
				}
				if !strings.Contains(err.Error(), "connection refused") &&
					!strings.Contains(err.Error(), "failed to check existing package") {
					t.Errorf("Expected connection error but got: %v", err)
				}
			},
			expectError: "connection refused",
		},
		{
			name: "timeout error",
			setupFunc: func(t *testing.T) *Client {
				// Create a server that responds slowly
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(200 * time.Millisecond) // Longer than client timeout
					w.WriteHeader(http.StatusOK)
				}))
				t.Cleanup(server.Close)
				
				client := NewClient(server.URL, false)
				// Set a very short timeout for the test
				client.httpClient.Timeout = 50 * time.Millisecond
				return client
			},
			testFunc: func(t *testing.T, client *Client) {
				req := &PackageRequest{
					Repository: "test-repo",
					Package:    "test-package",
					Workspace:  "default",
					Namespace:  "default",
					Intent: &intent.NetworkIntent{
						IntentType: "scaling",
						Target:     "test-app",
						Namespace:  "default",
						Replicas:   3,
					},
				}
				_, err := client.CreateOrUpdatePackage(req)
				if err == nil {
					t.Error("Expected timeout error but got nil")
				}
				if !strings.Contains(err.Error(), "timeout") &&
					!strings.Contains(err.Error(), "context deadline exceeded") {
					t.Errorf("Expected timeout error but got: %v", err)
				}
			},
			expectError: "timeout",
		},
		{
			name: "dns resolution failure",
			setupFunc: func(t *testing.T) *Client {
				// Use a non-existent domain
				return NewClient("http://non-existent-domain-12345.invalid", false)
			},
			testFunc: func(t *testing.T, client *Client) {
				req := &PackageRequest{
					Repository: "test-repo",
					Package:    "test-package",
					Workspace:  "default",
					Namespace:  "default",
					Intent: &intent.NetworkIntent{
						IntentType: "scaling",
						Target:     "test-app",
						Namespace:  "default",
						Replicas:   3,
					},
				}
				_, err := client.CreateOrUpdatePackage(req)
				if err == nil {
					t.Error("Expected DNS resolution error but got nil")
				}
				if !strings.Contains(err.Error(), "no such host") &&
					!strings.Contains(err.Error(), "failed to check existing package") {
					t.Errorf("Expected DNS error but got: %v", err)
				}
			},
			expectError: "no such host",
		},
		{
			name: "invalid URL",
			setupFunc: func(t *testing.T) *Client {
				// Use an invalid URL format
				return NewClient("not-a-valid-url", false)
			},
			testFunc: func(t *testing.T, client *Client) {
				req := &PackageRequest{
					Repository: "test-repo",
					Package:    "test-package",
					Workspace:  "default",
					Namespace:  "default",
					Intent: &intent.NetworkIntent{
						IntentType: "scaling",
						Target:     "test-app",
						Namespace:  "default",
						Replicas:   3,
					},
				}
				_, err := client.CreateOrUpdatePackage(req)
				if err == nil {
					t.Error("Expected URL error but got nil")
				}
			},
			expectError: "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupFunc(t)
			tt.testFunc(t, client)
		})
	}
}

// TestClient_HTTPErrors tests various HTTP error responses
func TestClient_HTTPErrors(t *testing.T) {
	tests := []struct {
		name         string
		serverSetup  func(t *testing.T) *httptest.Server
		expectError  string
		testOperation string
	}{
		{
			name: "server returns 500 internal server error",
			serverSetup: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal server error"))
				}))
			},
			expectError: "failed to get package: Internal server error",
			testOperation: "CreateOrUpdatePackage",
		},
		{
			name: "server returns 404 not found",
			serverSetup: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "/packages/") {
						w.WriteHeader(http.StatusNotFound)
						w.Write([]byte("Package not found"))
						return
					}
					if strings.Contains(r.URL.Path, "/packages") && r.Method == http.MethodPost {
						w.WriteHeader(http.StatusCreated)
						w.Write([]byte(`{"name": "test-package", "namespace": "default", "revision": "v1", "status": "draft"}`))
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectError: "",
			testOperation: "CreateOrUpdatePackage",
		},
		{
			name: "server returns 401 unauthorized",
			serverSetup: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Unauthorized"))
				}))
			},
			expectError: "failed to get package: Unauthorized",
			testOperation: "CreateOrUpdatePackage",
		},
		{
			name: "server returns 403 forbidden",
			serverSetup: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("Forbidden"))
				}))
			},
			expectError: "failed to get package: Forbidden",
			testOperation: "CreateOrUpdatePackage",
		},
		{
			name: "server returns invalid JSON",
			serverSetup: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "/packages/") {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					if strings.Contains(r.URL.Path, "/packages") && r.Method == http.MethodPost {
						w.WriteHeader(http.StatusCreated)
						w.Write([]byte(`{invalid json`))
						return
					}
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError: "invalid character",
			testOperation: "CreateOrUpdatePackage",
		},
		{
			name: "proposal submission error",
			serverSetup: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "/proposals") {
						w.WriteHeader(http.StatusBadRequest)
						w.Write([]byte("Invalid proposal"))
						return
					}
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError: "failed to submit proposal: Invalid proposal",
			testOperation: "SubmitProposal",
		},
		{
			name: "package approval error",
			serverSetup: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "/approve") {
						w.WriteHeader(http.StatusConflict)
						w.Write([]byte("Package already approved"))
						return
					}
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError: "failed to approve package: Package already approved",
			testOperation: "ApprovePackage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.serverSetup(t)
			defer server.Close()

			client := NewClient(server.URL, false)
			
			req := &PackageRequest{
				Repository: "test-repo",
				Package:    "test-package",
				Workspace:  "default",
				Namespace:  "default",
				Intent: &intent.NetworkIntent{
					IntentType: "scaling",
					Target:     "test-app",
					Namespace:  "default",
					Replicas:   3,
				},
			}

			var err error
			switch tt.testOperation {
			case "CreateOrUpdatePackage":
				_, err = client.CreateOrUpdatePackage(req)
			case "SubmitProposal":
				revision := &PorchPackageRevision{
					Name:      "test-package",
					Namespace: "default",
					Revision:  "v1",
					Status:    "draft",
				}
				_, err = client.SubmitProposal(revision)
			case "ApprovePackage":
				revision := &PorchPackageRevision{
					Name:      "test-package",
					Namespace: "default",
					Revision:  "v1",
					Status:    "draft",
				}
				err = client.ApprovePackage(revision)
			}

			if tt.expectError == "" {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing '%s' but got nil", tt.expectError)
				} else if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s' but got: %v", tt.expectError, err)
				}
			}
		})
	}
}

// TestClient_TLSErrors tests TLS-related error conditions
func TestClient_TLSErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) *Client
		expectError string
	}{
		{
			name: "TLS handshake failure",
			setupFunc: func(t *testing.T) *Client {
				// Create HTTPS server with invalid certificate
				server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				t.Cleanup(server.Close)
				
				// Use the server URL but don't configure the client to skip TLS verification
				client := NewClient(server.URL, false)
				// Force certificate verification failure
				client.httpClient.Transport = &http.Transport{
					TLSClientConfig: nil, // This will use default TLS config which requires valid certs
				}
				return client
			},
			expectError: "certificate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupFunc(t)
			
			req := &PackageRequest{
				Repository: "test-repo",
				Package:    "test-package",
				Workspace:  "default",
				Namespace:  "default",
				Intent: &intent.NetworkIntent{
					IntentType: "scaling",
					Target:     "test-app",
					Namespace:  "default",
					Replicas:   3,
				},
			}

			_, err := client.CreateOrUpdatePackage(req)
			if err == nil {
				t.Error("Expected TLS error but got nil")
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing '%s' but got: %v", tt.expectError, err)
			}
		})
	}
}

// TestClient_RequestErrors tests invalid request data handling
func TestClient_RequestErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test", "namespace": "default", "revision": "v1", "status": "draft"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, false)

	tests := []struct {
		name        string
		req         *PackageRequest
		expectError string
	}{
		{
			name:        "nil request",
			req:         nil,
			expectError: "panic", // This should cause a panic that we'll catch
		},
		{
			name: "empty repository",
			req: &PackageRequest{
				Repository: "",
				Package:    "test-package",
				Workspace:  "default",
				Namespace:  "default",
				Intent: &intent.NetworkIntent{
					IntentType: "scaling",
					Target:     "test-app",
					Namespace:  "default",
					Replicas:   3,
				},
			},
			expectError: "",
		},
		{
			name: "empty package name",
			req: &PackageRequest{
				Repository: "test-repo",
				Package:    "",
				Workspace:  "default",
				Namespace:  "default",
				Intent: &intent.NetworkIntent{
					IntentType: "scaling",
					Target:     "test-app",
					Namespace:  "default",
					Replicas:   3,
				},
			},
			expectError: "",
		},
		{
			name: "nil intent",
			req: &PackageRequest{
				Repository: "test-repo",
				Package:    "test-package",
				Workspace:  "default",
				Namespace:  "default",
				Intent:     nil,
			},
			expectError: "",
		},
		{
			name: "intent with invalid data",
			req: &PackageRequest{
				Repository: "test-repo",
				Package:    "test-package",
				Workspace:  "default",
				Namespace:  "default",
				Intent: &intent.NetworkIntent{
					IntentType: "", // Invalid intent type
					Target:     "",
					Namespace:  "",
					Replicas:   -1,
				},
			},
			expectError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && tt.expectError == "panic" {
					// Expected panic
					return
				} else if r != nil {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			_, err := client.CreateOrUpdatePackage(tt.req)
			
			if tt.expectError == "panic" {
				t.Error("Expected panic but didn't get one")
			} else if tt.expectError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s' but got nil", tt.expectError)
				} else if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s' but got: %v", tt.expectError, err)
				}
			}
		})
	}
}

// TestClient_ConcurrentRequests tests concurrent request handling
func TestClient_ConcurrentRequests(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		
		if strings.Contains(r.URL.Path, "/packages/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.Contains(r.URL.Path, "/packages") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(fmt.Sprintf(`{"name": "test-package-%d", "namespace": "default", "revision": "v1", "status": "draft"}`, requestCount)))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, false)

	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	// Launch multiple concurrent requests
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			req := &PackageRequest{
				Repository: "test-repo",
				Package:    fmt.Sprintf("test-package-%d", id),
				Workspace:  "default",
				Namespace:  "default",
				Intent: &intent.NetworkIntent{
					IntentType: "scaling",
					Target:     fmt.Sprintf("test-app-%d", id),
					Namespace:  "default",
					Replicas:   3,
				},
			}
			
			_, err := client.CreateOrUpdatePackage(req)
			done <- err
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}

	// Verify all requests were processed
	if requestCount < numGoroutines*2 { // Each CreateOrUpdate makes at least 2 requests
		t.Errorf("Expected at least %d requests but got %d", numGoroutines*2, requestCount)
	}
}

// TestClient_RetryLogic tests network retry behavior
func TestClient_RetryLogic(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Fail the first 2 attempts
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server temporarily unavailable"))
			return
		}
		// Succeed on the 3rd attempt
		if strings.Contains(r.URL.Path, "/packages/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"name": "test-package", "namespace": "default", "revision": "v1", "status": "draft"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, false)

	req := &PackageRequest{
		Repository: "test-repo",
		Package:    "test-package",
		Workspace:  "default",
		Namespace:  "default",
		Intent: &intent.NetworkIntent{
			IntentType: "scaling",
			Target:     "test-app",
			Namespace:  "default",
			Replicas:   3,
		},
	}

	// The current implementation doesn't have retry logic, so this should fail
	_, err := client.CreateOrUpdatePackage(req)
	if err == nil {
		t.Error("Expected error due to server failure but got nil")
	}
	if !strings.Contains(err.Error(), "Server temporarily unavailable") {
		t.Errorf("Expected server error but got: %v", err)
	}
}

// TestClient_ContextCancellation tests context cancellation handling
func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, false)
	
	// Set a very short timeout to trigger cancellation
	client.httpClient.Timeout = 100 * time.Millisecond

	req := &PackageRequest{
		Repository: "test-repo",
		Package:    "test-package",
		Workspace:  "default",
		Namespace:  "default",
		Intent: &intent.NetworkIntent{
			IntentType: "scaling",
			Target:     "test-app",
			Namespace:  "default",
			Replicas:   3,
		},
	}

	_, err := client.CreateOrUpdatePackage(req)
	if err == nil {
		t.Error("Expected timeout error but got nil")
	}
	if !strings.Contains(err.Error(), "timeout") && 
		!strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error but got: %v", err)
	}
}

// TestClient_MalformedResponses tests handling of malformed server responses
func TestClient_MalformedResponses(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
		statusCode   int
		expectError  string
	}{
		{
			name:         "empty response body",
			responseBody: "",
			statusCode:   http.StatusCreated,
			expectError:  "EOF",
		},
		{
			name:         "malformed JSON",
			responseBody: `{invalid json`,
			statusCode:   http.StatusCreated,
			expectError:  "invalid character",
		},
		{
			name:         "wrong JSON structure",
			responseBody: `["array", "instead", "of", "object"]`,
			statusCode:   http.StatusCreated,
			expectError:  "cannot unmarshal array",
		},
		{
			name:         "partial JSON response",
			responseBody: `{"name": "test", "namespace":`,
			statusCode:   http.StatusCreated,
			expectError:  "unexpected EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/packages/") {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, false)
			
			req := &PackageRequest{
				Repository: "test-repo",
				Package:    "test-package",
				Workspace:  "default",
				Namespace:  "default",
				Intent: &intent.NetworkIntent{
					IntentType: "scaling",
					Target:     "test-app",
					Namespace:  "default",
					Replicas:   3,
				},
			}

			_, err := client.CreateOrUpdatePackage(req)
			if err == nil {
				t.Error("Expected JSON parsing error but got nil")
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing '%s' but got: %v", tt.expectError, err)
			}
		})
	}
}