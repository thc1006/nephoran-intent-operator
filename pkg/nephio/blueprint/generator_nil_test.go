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

package blueprint

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// TestClientAdapterNilClient verifies proper handling of nil client
func TestClientAdapterNilClient(t *testing.T) {
	adapter := &ClientAdapter{client: nil}

	ctx := context.Background()
	request := &llm.ProcessIntentRequest{
		Intent:  "test intent",
		Context: map[string]string{},
		Metadata: llm.RequestMetadata{
			RequestID: "test-nil",
		},
		Timestamp: time.Now(),
	}

	_, err := adapter.ProcessIntent(ctx, request)
	if err == nil {
		t.Fatal("Expected error for nil client, got nil")
	}

	expectedError := "LLM client is nil - service may not be available"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}

// TestClientAdapterNilRequest verifies proper handling of nil request
func TestClientAdapterNilRequest(t *testing.T) {
	mockClient := llm.NewClient("http://test:8080")
	adapter := &ClientAdapter{client: mockClient}

	ctx := context.Background()

	_, err := adapter.ProcessIntent(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil request, got nil")
	}

	expectedError := "request cannot be nil"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}