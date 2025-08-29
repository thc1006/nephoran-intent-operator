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

	"fmt"

	"math/rand"

	"time"



	"github.com/google/uuid"



	v1 "github.com/nephio-project/nephoran-intent-operator/api/v1"



	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/rest"



	"sigs.k8s.io/controller-runtime/pkg/client"

	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

)



const (

	// Default test values.

	DefaultTestNamespace = "test-namespace"

	// DefaultTestRepository holds defaulttestrepository value.

	DefaultTestRepository = "test-repo"

	// DefaultTestPackageName holds defaulttestpackagename value.

	DefaultTestPackageName = "test-package"

	// DefaultTestRevision holds defaulttestrevision value.

	DefaultTestRevision = "v1.0.0"

	// DefaultTestBranch holds defaulttestbranch value.

	DefaultTestBranch = "main"

	// DefaultTestURL holds defaulttesturl value.

	DefaultTestURL = "https://github.com/test/test-repo.git"

)



// TestFixture provides a complete test environment (generic version).

type TestFixture struct {

	Client    client.Client

	Context   context.Context

	Namespace string

}



// NewTestFixture creates a new test fixture with default configuration.

func NewTestFixture(ctx context.Context) *TestFixture {

	if ctx == nil {

		ctx = context.Background()

	}



	scheme := runtime.NewScheme()

	_ = v1.AddToScheme(scheme)



	fakeClient := clientfake.NewClientBuilder().

		WithScheme(scheme).

		Build()



	fixture := &TestFixture{

		Client:    fakeClient,

		Context:   ctx,

		Namespace: DefaultTestNamespace,

	}



	return fixture

}



// CreateTestNetworkIntent creates a test NetworkIntent for testing.

func (f *TestFixture) CreateTestNetworkIntent(name string, opts ...NetworkIntentOption) *v1.NetworkIntent {

	if name == "" {

		name = fmt.Sprintf("test-intent-%s", GenerateRandomString(8))

	}



	intent := &v1.NetworkIntent{

		TypeMeta: metav1.TypeMeta{

			APIVersion: "nephoran.com/v1",

			Kind:       "NetworkIntent",

		},

		ObjectMeta: metav1.ObjectMeta{

			Name:      name,

			Namespace: f.Namespace,

			UID:       types.UID(uuid.New().String()),

			Labels: map[string]string{

				"test.nephoran.com/fixture": "true",

			},

		},

		Spec: v1.NetworkIntentSpec{

			Intent:     "Deploy 3 AMF instances in us-east-1 region",

			IntentType: v1.IntentTypeDeployment,

			Priority:   v1.PriorityMedium,

			TargetComponents: []v1.ORANComponent{

				v1.ORANComponentAMF,

			},

		},

		Status: v1.NetworkIntentStatus{

			Phase:          "processing",

			LastMessage:    "Processing deployment intent",

			LastUpdateTime: metav1.Now(),

		},

	}



	// Apply options.

	for _, opt := range opts {

		opt(intent)

	}



	return intent

}



// NetworkIntentOption allows customization of test network intents.

type NetworkIntentOption func(*v1.NetworkIntent)



// WithIntentType sets the intent type.

func WithIntentType(intentType v1.IntentType) NetworkIntentOption {

	return func(intent *v1.NetworkIntent) {

		intent.Spec.IntentType = intentType

	}

}



// WithIntentPriority sets the intent priority.

func WithIntentPriority(priority v1.Priority) NetworkIntentOption {

	return func(intent *v1.NetworkIntent) {

		intent.Spec.Priority = priority

	}

}



// WithTargetComponent adds a target component.

func WithTargetComponent(component v1.ORANComponent) NetworkIntentOption {

	return func(intent *v1.NetworkIntent) {

		intent.Spec.TargetComponents = append(intent.Spec.TargetComponents, component)

	}

}



// WithIntent sets the intent text.

func WithIntent(intentText string) NetworkIntentOption {

	return func(intent *v1.NetworkIntent) {

		intent.Spec.Intent = intentText

	}

}



// WithIntentPhase sets the intent phase.

func WithIntentPhase(phase string) NetworkIntentOption {

	return func(intent *v1.NetworkIntent) {

		intent.Status.Phase = v1.NetworkIntentPhase(phase)

	}

}



// Helper functions for test data generation.



// GenerateRandomString generates a random string of specified length.

func GenerateRandomString(length int) string {

	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, length)

	for i := range b {

		b[i] = charset[rand.Intn(len(charset))]

	}

	return string(b)

}



// Cleanup removes test resources and performs cleanup.

func (f *TestFixture) Cleanup() {

	// Clear any internal state if needed.

}



// AssertCondition checks if a condition exists and has expected values.

func AssertCondition(conditions []metav1.Condition, conditionType string, status metav1.ConditionStatus, reason string) bool {

	for _, condition := range conditions {

		if condition.Type == conditionType &&

			condition.Status == status &&

			condition.Reason == reason {

			return true

		}

	}

	return false

}



// WaitForCondition waits for a condition to be met within a timeout.

func WaitForCondition(ctx context.Context, check func() bool, timeout time.Duration) bool {

	ctx, cancel := context.WithTimeout(ctx, timeout)

	defer cancel()



	ticker := time.NewTicker(100 * time.Millisecond)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return false

		case <-ticker.C:

			if check() {

				return true

			}

		}

	}

}



// GetTestKubeConfig returns a test Kubernetes configuration.

func GetTestKubeConfig() *rest.Config {

	return &rest.Config{

		Host: "https://localhost:6443",

		TLSClientConfig: rest.TLSClientConfig{

			Insecure: true,

		},

	}

}

