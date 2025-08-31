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

package webhooks

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("NetworkIntent Webhook", func() {
	var (
		validator *NetworkIntentValidator
		scheme    *runtime.Scheme
		decoder   admission.Decoder
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(nephoranv1.AddToScheme(scheme)).To(Succeed())

		decoder = admission.NewDecoder(scheme)

		validator = NewNetworkIntentValidator()
		Expect(validator.InjectDecoder(decoder)).To(Succeed())
	})

	Describe("Basic Validation Tests", func() {
		Context("when validating valid intents", func() {
			DescribeTable("should accept valid telecommunications intents",
				func(intent string) {
					ni := &nephoranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-intent",
							Namespace: "default",
						},
						Spec: nephoranv1.NetworkIntentSpec{
							Intent: intent,
						},
					}

					err := validator.validateNetworkIntent(context.TODO(), ni)
					Expect(err).NotTo(HaveOccurred())
				},
				Entry("AMF deployment", "Deploy a high-availability AMF instance for production with auto-scaling"),
				Entry("Network slice creation", "Create a network slice for URLLC with 1ms latency requirements"),
				Entry("QoS configuration", "Configure QoS policies for enhanced mobile broadband services"),
				Entry("SMF setup", "Setup SMF with redundancy for critical telecommunications workloads"),
				Entry("O-RAN deployment", "Deploy O-RAN Near-RT RIC with xApp orchestration capabilities"),
				Entry("UPF scaling", "Scale UPF instances based on traffic load in the production cluster"),
				Entry("Network function orchestration", "Orchestrate CNF deployment for 5G core network functions"),
			)
		})

		Context("when validating invalid intents", func() {
			DescribeTable("should reject invalid intents",
				func(intent string, expectedErrorContains string) {
					ni := &nephoranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-intent",
							Namespace: "default",
						},
						Spec: nephoranv1.NetworkIntentSpec{
							Intent: intent,
						},
					}

					err := validator.validateNetworkIntent(context.TODO(), ni)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(expectedErrorContains))
				},
				Entry("empty intent", "", "intent cannot be empty"),
				Entry("non-telecom intent", "Make me a sandwich please", "telecommunications-related"),
				Entry("script injection", "<script>alert('xss')</script>Deploy AMF", "malicious pattern"),
				Entry("SQL injection", "Deploy AMF; DROP TABLE users; --", "malicious pattern"),
				Entry("too vague", "do something with the network", "too vague"),
				Entry("contradictory", "enable and disable the AMF service", "contradictory terms"),
				Entry("excessive repetition", "deploy deploy deploy deploy deploy deploy AMF", "consecutive repeated words"),
				Entry("too complex", generateComplexIntent(), "too complex"),
			)
		})
	})

	Describe("Security Validation Tests", func() {
		Context("when checking for malicious patterns", func() {
			It("should reject script injection attempts", func() {
				maliciousIntents := []string{
					"Deploy AMF <script>alert('hack')</script>",
					"Setup network javascript:alert('xss')",
					"Configure AMF onload=malicious()",
				}

				for _, intent := range maliciousIntents {
					err := validator.validateSecurity(intent)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("malicious pattern"))
				}
			})

			It("should reject SQL injection attempts", func() {
				maliciousIntents := []string{
					"Deploy AMF UNION SELECT * FROM secrets",
					"Setup network; DROP TABLE important_data; --",
					"Configure SMF OR 1=1",
				}

				for _, intent := range maliciousIntents {
					err := validator.validateSecurity(intent)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("malicious pattern"))
				}
			})

			It("should reject command injection attempts", func() {
				maliciousIntents := []string{
					"Deploy AMF; rm -rf /",
					"Setup network $(cat /etc/passwd)",
					"Configure UPF | bash",
				}

				for _, intent := range maliciousIntents {
					err := validator.validateSecurity(intent)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("malicious pattern"))
				}
			})

			It("should reject path traversal attempts", func() {
				maliciousIntents := []string{
					"Deploy AMF ../../../etc/passwd",
					"Setup network ../../secrets",
				}

				for _, intent := range maliciousIntents {
					err := validator.validateSecurity(intent)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("malicious pattern"))
				}
			})
		})
	})

	Describe("Telecommunications Relevance Tests", func() {
		Context("when checking telecom keywords", func() {
			It("should accept intents with telecom keywords", func() {
				telecomIntents := []string{
					"Deploy AMF with high availability",
					"Configure O-RAN interface for Near-RT RIC",
					"Setup network slice for URLLC applications",
					"Scale CNF instances based on QoS requirements",
					"Implement 5G core network functions",
				}

				for _, intent := range telecomIntents {
					err := validator.validateTelecomRelevance(intent)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should reject intents without telecom keywords", func() {
				nonTelecomIntents := []string{
					"Make me coffee",
					"Deploy a web application",
					"Create a database backup",
					"Send email notifications",
				}

				for _, intent := range nonTelecomIntents {
					err := validator.validateTelecomRelevance(intent)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("telecommunications-related"))
				}
			})
		})
	})

	Describe("Complexity Validation Tests", func() {
		Context("when checking intent complexity", func() {
			It("should accept reasonably complex intents", func() {
				intent := "Deploy AMF instance with high availability configuration and auto-scaling policies for production environment"
				err := validator.validateComplexity(intent)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should reject overly complex intents", func() {
				intent := generateComplexIntent()
				err := validator.validateComplexity(intent)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("too complex"))
			})

			It("should reject intents with excessive repetition", func() {
				intent := "deploy deploy deploy deploy deploy deploy AMF network function"
				err := validator.validateComplexity(intent)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("consecutive repeated words"))
			})

			It("should reject empty word intents", func() {
				intent := "   \t\n   "
				err := validator.validateComplexity(intent)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no recognizable words"))
			})
		})
	})

	Describe("Business Logic Validation Tests", func() {
		Context("when validating business rules", func() {
			It("should accept intents with action verbs", func() {
				actionIntents := []string{
					"Deploy AMF for production",
					"Configure QoS policies",
					"Setup network slice",
					"Scale UPF instances",
					"Enable monitoring",
				}

				for _, intent := range actionIntents {
					err := validator.validateIntentCoherence(intent)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should reject intents without action verbs", func() {
				nonActionIntents := []string{
					"AMF is good",
					"Network slice seems fine",
					"The system is running",
				}

				for _, intent := range nonActionIntents {
					err := validator.validateIntentCoherence(intent)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("actionable verbs"))
				}
			})

			It("should reject contradictory intents", func() {
				contradictoryIntents := []string{
					"Enable and disable the AMF service",
					"Scale up and scale down the network function",
					"Create and delete the deployment",
				}

				for _, intent := range contradictoryIntents {
					err := validator.validateIntentCoherence(intent)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("contradictory terms"))
				}
			})

			It("should reject vague intents", func() {
				vague_intents := []string{
					"Do something with the AMF",
					"Make it work somehow",
					"Just handle everything",
				}

				for _, intent := range vague_intents {
					err := validator.validateIntentCoherence(intent)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("too vague"))
				}
			})
		})
	})

	Describe("Resource Naming Validation Tests", func() {
		Context("when validating NetworkIntent names", func() {
			It("should accept valid Kubernetes names", func() {
				validNames := []string{
					"amf-production-intent",
					"network-slice-urllc",
					"qos-policy-config",
					"smf-ha-setup",
				}

				for _, name := range validNames {
					ni := &nephoranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      name,
							Namespace: "default",
						},
					}
					err := validator.validateResourceNaming(ni)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should reject invalid names", func() {
				invalidNames := []string{
					"",  // empty
					"a", // too short
					"this-name-is-way-too-long-and-exceeds-the-kubernetes-limit-of-sixty-three-characters",
					"Invalid_Name", // uppercase and underscore
				}

				for _, name := range invalidNames {
					ni := &nephoranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      name,
							Namespace: "default",
						},
					}
					err := validator.validateResourceNaming(ni)
					Expect(err).To(HaveOccurred())
				}
			})
		})
	})

	Describe("Admission Request Handling", func() {
		Context("when handling admission requests", func() {
			It("should allow valid CREATE requests", func() {
				ni := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-intent",
						Namespace: "default",
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: "Deploy AMF with high availability for production cluster",
					},
				}

				raw, err := json.Marshal(ni)
				Expect(err).NotTo(HaveOccurred())

				req := admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						Operation: admissionv1.Create,
						Object: runtime.RawExtension{
							Raw: raw,
						},
					},
				}

				resp := validator.Handle(context.TODO(), req)
				Expect(resp.Allowed).To(BeTrue())
			})

			It("should deny invalid CREATE requests", func() {
				ni := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-intent",
						Namespace: "default",
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: "Make me a sandwich", // non-telecom intent
					},
				}

				raw, err := json.Marshal(ni)
				Expect(err).NotTo(HaveOccurred())

				req := admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						Operation: admissionv1.Create,
						Object: runtime.RawExtension{
							Raw: raw,
						},
					},
				}

				resp := validator.Handle(context.TODO(), req)
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Message).To(ContainSubstring("telecommunications-related"))
			})

			It("should allow DELETE requests", func() {
				req := admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						Operation: admissionv1.Delete,
					},
				}

				resp := validator.Handle(context.TODO(), req)
				Expect(resp.Allowed).To(BeTrue())
				Expect(resp.Result.Message).To(Equal("Delete operations are allowed"))
			})
		})
	})
})

// generateComplexIntent creates an overly complex intent for testing
func generateComplexIntent() string {
	words := make([]string, DefaultComplexityRules.MaxWords+10)
	for i := 0; i < len(words); i++ {
		if i%10 == 0 {
			words[i] = "AMF" // Add telecom keyword every 10 words
		} else {
			words[i] = "word" + string(rune(i%26+97)) // Generate unique words
		}
	}
	return "Deploy " + strings.Join(words, " ") + " network function"
}

func TestNetworkIntentWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NetworkIntent Webhook Suite")
}

// Test individual validation functions
func TestValidateIntentContent(t *testing.T) {
	validator := NewNetworkIntentValidator()

	tests := []struct {
		name        string
		intent      string
		expectError bool
	}{
		{
			name:        "valid intent",
			intent:      "Deploy AMF with high availability",
			expectError: false,
		},
		{
			name:        "empty intent",
			intent:      "",
			expectError: true,
		},
		{
			name:        "whitespace only",
			intent:      "   \t\n   ",
			expectError: true,
		},
		{
			name:        "excessive line breaks",
			intent:      "Deploy AMF\n" + strings.Repeat("\n", 60) + "with HA",
			expectError: true,
		},
		{
			name:        "excessive tabs",
			intent:      "Deploy\t" + strings.Repeat("\t", 25) + "AMF",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateIntentContent(tt.intent)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Note: TestValidateSecurity and TestValidateTelecomRelevance are implemented 
// in networkintent_webhook_comprehensive_test.go to avoid duplication
