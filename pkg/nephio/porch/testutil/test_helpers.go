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




package testutil



import (

	"context"

	"testing"

	"time"



	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/envtest"



	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"

)



// TestConfig creates a test configuration for porch clients.

func NewTestConfig() *porch.Config {

	return &porch.Config{

		PorchConfig: &porch.PorchServiceConfig{

			Endpoint: "http://localhost:8080",

			Auth: &porch.AuthenticationConfig{

				Type: "none",

			},

			Timeout: 30 * time.Second,

			Retry: &porch.RetryConfig{

				MaxRetries:   3,

				InitialDelay: 1 * time.Second,

			},

		},

		Observability: &porch.ObservabilityConfig{

			Logging: &porch.LoggingConfig{

				Level:  "debug",

				Format: "text",

				Output: []string{"stdout"},

			},

		},

	}

}



// GetTestKubeConfig returns a test Kubernetes configuration.

func GetTestKubeConfig() *rest.Config {

	return &rest.Config{

		Host:    "http://localhost:8080",

		QPS:     100,

		Burst:   150,

		Timeout: 30 * time.Second,

	}

}



// ClientOptions represents options for creating test clients.

type ClientOptions struct {

	Config     *porch.Config

	KubeConfig *rest.Config

	Context    context.Context

	Namespace  string

}



// NewTestEnvironment creates a test environment with envtest.

func NewTestEnvironment(t *testing.T) (*envtest.Environment, *rest.Config) {

	t.Helper()



	testEnv := &envtest.Environment{

		CRDDirectoryPaths: []string{"../../../config/crd/bases"},

	}



	cfg, err := testEnv.Start()

	if err != nil {

		t.Fatalf("Failed to start test environment: %v", err)

	}



	return testEnv, cfg

}



// MockPorchClient creates a mock porch client for testing.

type MockPorchClient struct {

	*porch.Client

	MockResponses map[string]interface{}

	CallLog       []string

}



// NewMockPorchClient creates a new mock client.

func NewMockPorchClient() *MockPorchClient {

	return &MockPorchClient{

		MockResponses: make(map[string]interface{}),

		CallLog:       make([]string, 0),

	}

}



// RecordCall records a method call for verification.

func (m *MockPorchClient) RecordCall(method string) {

	m.CallLog = append(m.CallLog, method)

}



// GetCallLog returns the recorded method calls.

func (m *MockPorchClient) GetCallLog() []string {

	return m.CallLog

}



// SetMockResponse sets a mock response for a given method.

func (m *MockPorchClient) SetMockResponse(method string, response interface{}) {

	m.MockResponses[method] = response

}



// GetMockResponse gets a mock response for a given method.

func (m *MockPorchClient) GetMockResponse(method string) interface{} {

	return m.MockResponses[method]

}



// TestRepository creates a test repository configuration.

func NewTestRepository(name string) *porch.Repository {

	return &porch.Repository{

		ObjectMeta: metav1.ObjectMeta{

			Name:      name,

			Namespace: "default",

		},

		Spec: porch.RepositorySpec{

			Type:         "git",

			URL:          "https://github.com/test/repo.git",

			Branch:       "main",

			Directory:    "/",

			Capabilities: []string{"upstream"},

		},

	}

}



// TestPackageRevision creates a test package revision.

func NewTestPackageRevision(name, repository string) *porch.PackageRevision {

	return &porch.PackageRevision{

		ObjectMeta: metav1.ObjectMeta{

			Name:      name,

			Namespace: "default",

		},

		Spec: porch.PackageRevisionSpec{

			Repository:    repository,

			PackageName:   name,

			Revision:      "v1.0.0",

			Lifecycle:     porch.PackageRevisionLifecyclePublished,

			WorkspaceName: "",

		},

	}

}



// TestFunctionConfig creates a test function configuration.

func NewTestFunctionConfig(name string) porch.FunctionConfig {

	return porch.FunctionConfig{

		Image: "gcr.io/kpt-fn/test-function:v1.0.0",

		ConfigMap: map[string]interface{}{

			"name":        name,

			"description": "Test function for " + name,

		},

	}

}



// AssertNoError is a test helper for asserting no error.

func AssertNoError(t *testing.T, err error) {

	t.Helper()

	if err != nil {

		t.Fatalf("Expected no error, got: %v", err)

	}

}



// AssertError is a test helper for asserting an error occurred.

func AssertError(t *testing.T, err error) {

	t.Helper()

	if err == nil {

		t.Fatal("Expected an error, got nil")

	}

}



// AssertEqual is a test helper for asserting equality.

func AssertEqual(t *testing.T, expected, actual interface{}) {

	t.Helper()

	if expected != actual {

		t.Fatalf("Expected %v, got %v", expected, actual)

	}

}

