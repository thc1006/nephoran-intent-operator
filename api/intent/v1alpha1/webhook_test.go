package v1alpha1

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWebhookValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NetworkIntent Webhook Suite")
}

var _ = Describe("NetworkIntent Webhook", func() {
	var (
		ctx       context.Context
		ni        *NetworkIntent
		validator *NetworkIntent
	)

	BeforeEach(func() {
		ctx = context.Background()
		ni = &NetworkIntent{}
		validator = &NetworkIntent{}
	})

	Describe("Defaulting webhook", func() {
		It("should set default source to 'user' when not specified", func() {
			ni := &NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "default",
				},
				Spec: NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "my-deployment",
					Namespace:  "default",
					Replicas:   3,
					// Source not specified
				},
			}

			err := ni.Default(ctx, ni)
			Expect(err).NotTo(HaveOccurred())
			Expect(ni.Spec.Source).To(Equal("user"))
		})

		It("should not override existing source value", func() {
			ni := &NetworkIntent{
				Spec: NetworkIntentSpec{
					Source: "planner",
				},
			}

			err := ni.Default(ctx, ni)
			Expect(err).NotTo(HaveOccurred())
			Expect(ni.Spec.Source).To(Equal("planner"))
		})
	})

	Describe("Validating webhook", func() {
		Context("with valid NetworkIntent", func() {
			It("should accept valid scaling intent", func() {
				ni := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "scaling",
						Target:     "my-deployment",
						Namespace:  "default",
						Replicas:   5,
						Source:     "user",
					},
				}

				warnings, err := ni.ValidateCreate(ctx, ni)
				Expect(warnings).To(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})

			It("should accept replicas at minimum boundary (0)", func() {
				ni := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "scaling",
						Target:     "my-deployment",
						Namespace:  "default",
						Replicas:   0,
						Source:     "user",
					},
				}

				warnings, err := ni.ValidateCreate(ctx, ni)
				Expect(warnings).To(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})

<<<<<<< HEAD
			It("should accept large replicas values", func() {
=======
			It("should accept large replicas values with warning", func() {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				ni := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "scaling",
						Target:     "my-deployment",
						Namespace:  "default",
						Replicas:   1000,
						Source:     "user",
					},
				}

				warnings, err := ni.ValidateCreate(ctx, ni)
<<<<<<< HEAD
				Expect(warnings).To(BeNil())
				Expect(err).NotTo(HaveOccurred())
=======
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).NotTo(BeNil()) // Expect warning for high replicas
				Expect(len(warnings)).To(Equal(1))
				Expect(warnings[0]).To(ContainSubstring("very high value"))
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			})

			It("should accept all valid source values", func() {
				validSources := []string{"user", "planner", "test"}

				for _, source := range validSources {
					ni := &NetworkIntent{
						Spec: NetworkIntentSpec{
							IntentType: "scaling",
							Target:     "my-deployment",
							Namespace:  "default",
							Replicas:   5,
							Source:     source,
						},
					}

					warnings, err := ni.ValidateCreate(ctx, ni)
					Expect(warnings).To(BeNil())
					Expect(err).NotTo(HaveOccurred(), "source=%s should be valid", source)
				}
			})
		})

		Context("with invalid NetworkIntent", func() {
			It("should reject non-scaling intent type", func() {
				ni := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "provisioning", // Invalid
						Target:     "my-deployment",
						Namespace:  "default",
						Replicas:   5,
						Source:     "user",
					},
				}

				warnings, err := ni.ValidateCreate(ctx, ni)
				Expect(warnings).To(BeNil())
				Expect(err).To(HaveOccurred())
<<<<<<< HEAD
				Expect(err.Error()).To(ContainSubstring("only 'scaling' supported"))
=======
				Expect(err.Error()).To(ContainSubstring("spec.intentType must be 'scaling', got: provisioning"))
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			})

			It("should reject negative replicas", func() {
				ni := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "scaling",
						Target:     "my-deployment",
						Namespace:  "default",
						Replicas:   -1, // Negative
						Source:     "user",
					},
				}

				warnings, err := ni.ValidateCreate(ctx, ni)
				Expect(warnings).To(BeNil())
				Expect(err).To(HaveOccurred())
<<<<<<< HEAD
				Expect(err.Error()).To(ContainSubstring("must be >= 0"))
=======
				Expect(err.Error()).To(ContainSubstring("spec.replicas must be non-negative, got: -1"))
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			})

			It("should reject empty target", func() {
				ni := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "scaling",
						Target:     "", // Empty
						Namespace:  "default",
						Replicas:   5,
						Source:     "user",
					},
				}

				warnings, err := ni.ValidateCreate(ctx, ni)
				Expect(warnings).To(BeNil())
				Expect(err).To(HaveOccurred())
<<<<<<< HEAD
				Expect(err.Error()).To(ContainSubstring("must be non-empty"))
=======
				Expect(err.Error()).To(ContainSubstring("spec.target cannot be empty"))
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			})

			It("should reject empty namespace", func() {
				ni := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "scaling",
						Target:     "my-deployment",
						Namespace:  "", // Empty
						Replicas:   5,
						Source:     "user",
					},
				}

				warnings, err := ni.ValidateCreate(ctx, ni)
				Expect(warnings).To(BeNil())
				Expect(err).To(HaveOccurred())
<<<<<<< HEAD
				Expect(err.Error()).To(ContainSubstring("must be non-empty"))
=======
				Expect(err.Error()).To(ContainSubstring("spec.namespace cannot be empty"))
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			})

			It("should reject invalid source value", func() {
				ni := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "scaling",
						Target:     "my-deployment",
						Namespace:  "default",
						Replicas:   5,
						Source:     "invalid", // Invalid source
					},
				}

				warnings, err := ni.ValidateCreate(ctx, ni)
				Expect(warnings).To(BeNil())
<<<<<<< HEAD
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must be 'user', 'planner', or 'test'"))
=======
				// Current implementation does validate source values
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.source must be 'user', 'planner', or 'test', got: invalid"))
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			})

			It("should report multiple validation errors", func() {
				ni := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "invalid", // Invalid
						Target:     "",        // Empty
						Namespace:  "",        // Empty
						Replicas:   -5,        // Negative
						Source:     "invalid", // Invalid
					},
				}

				warnings, err := ni.ValidateCreate(ctx, ni)
				Expect(warnings).To(BeNil())
				Expect(err).To(HaveOccurred())

				// Check for all expected error messages
				errorMsg := err.Error()
<<<<<<< HEAD
				Expect(errorMsg).To(ContainSubstring("only 'scaling' supported"))
				Expect(errorMsg).To(ContainSubstring("must be >= 0"))
				Expect(errorMsg).To(ContainSubstring("target"))
				Expect(errorMsg).To(ContainSubstring("namespace"))
				Expect(errorMsg).To(ContainSubstring("must be 'user', 'planner', or 'test'"))
=======
				Expect(errorMsg).To(ContainSubstring("spec.intentType must be 'scaling', got: invalid"))
				Expect(errorMsg).To(ContainSubstring("spec.replicas must be non-negative, got: -5"))
				Expect(errorMsg).To(ContainSubstring("spec.target cannot be empty"))
				Expect(errorMsg).To(ContainSubstring("spec.namespace cannot be empty"))
				Expect(errorMsg).To(ContainSubstring("spec.source must be 'user', 'planner', or 'test', got: invalid"))
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			})
		})

		Context("ValidateUpdate", func() {
			It("should validate updates with same rules as create", func() {
				oldNI := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "scaling",
						Target:     "my-deployment",
						Namespace:  "default",
						Replicas:   5,
						Source:     "user",
					},
				}

				newNI := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "scaling",
						Target:     "my-deployment",
						Namespace:  "default",
						Replicas:   -2, // Invalid - negative
						Source:     "user",
					},
				}

				warnings, err := ni.ValidateUpdate(ctx, oldNI, newNI)
				Expect(warnings).To(BeNil())
				Expect(err).To(HaveOccurred())
<<<<<<< HEAD
				Expect(err.Error()).To(ContainSubstring("must be >= 0"))
=======
				Expect(err.Error()).To(ContainSubstring("spec.replicas must be non-negative, got: -2"))
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			})
		})

		Context("ValidateDelete", func() {
			It("should allow deletion without validation", func() {
				ni := &NetworkIntent{
					Spec: NetworkIntentSpec{
						IntentType: "scaling",
						Target:     "my-deployment",
						Namespace:  "default",
						Replicas:   5,
						Source:     "user",
					},
				}

				warnings, err := validator.ValidateDelete(ctx, ni)
				Expect(warnings).To(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
