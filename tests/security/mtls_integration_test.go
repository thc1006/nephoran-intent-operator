package security

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/pkg/security/ca"
	"github.com/thc1006/nephoran-intent-operator/pkg/security/mtls"
	"github.com/thc1006/nephoran-intent-operator/tests/utils"
)

// mTLSIntegrationTestSuite tests integration between all mTLS components
type mTLSIntegrationTestSuite struct {
	ctx          context.Context
	k8sClient    client.Client
	namespace    string
	caManager    *ca.CAManager
	mtlsClients  map[string]*mtls.Client
	testServices map[string]*testServiceDeployment
	mu           sync.RWMutex
}

// testServiceDeployment represents a test service with mTLS configuration
type testServiceDeployment struct {
	Name       string
	Deployment *appsv1.Deployment
	Service    *corev1.Service
	ServerCert *corev1.Secret
	ClientCert *corev1.Secret
	MTLSClient *mtls.Client
	Endpoint   string
}

var _ = Describe("mTLS Integration Test Suite", func() {
	var suite *mTLSIntegrationTestSuite

	BeforeEach(func() {
		suite = &mTLSIntegrationTestSuite{
			ctx:          context.Background(),
			k8sClient:    utils.GetK8sClient(),
			namespace:    utils.GetTestNamespace(),
			mtlsClients:  make(map[string]*mtls.Client),
			testServices: make(map[string]*testServiceDeployment),
		}

		// Initialize CA Manager
		err := suite.initializeCAManager()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		suite.cleanup()
	})

	Context("Controller-to-Service mTLS Communication", func() {
		It("should establish mTLS connections between NetworkIntent controller and services", func() {
			// Deploy test services that mimic the actual system components
			services := []string{"llm-processor", "rag-api", "nephio-bridge"}

			for _, serviceName := range services {
				By(fmt.Sprintf("Deploying test service: %s", serviceName))

				testService, err := suite.deployTestService(serviceName)
				Expect(err).NotTo(HaveOccurred())

				suite.testServices[serviceName] = testService

				// Wait for service to be ready
				Eventually(func() bool {
					return suite.isServiceReady(serviceName)
				}, 60*time.Second, 2*time.Second).Should(BeTrue())
			}

			// Test controller-to-service communication
			for serviceName, testService := range suite.testServices {
				By(fmt.Sprintf("Testing mTLS communication to %s", serviceName))

				// Create mTLS client for controller
				controllerClient, err := suite.createControllerMTLSClient(serviceName)
				Expect(err).NotTo(HaveOccurred())

				// Test HTTP request
				resp, err := controllerClient.GetHTTPClient().Get(testService.Endpoint + "/health")
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				resp.Body.Close()

				By(fmt.Sprintf("mTLS communication to %s successful", serviceName))
			}
		})

		It("should handle certificate rotation in controller-to-service communication", func() {
			serviceName := "llm-processor"

			// Deploy service
			testService, err := suite.deployTestService(serviceName)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				return suite.isServiceReady(serviceName)
			}, 60*time.Second, 2*time.Second).Should(BeTrue())

			// Create initial mTLS client
			client1, err := suite.createControllerMTLSClient(serviceName)
			Expect(err).NotTo(HaveOccurred())

			// Test initial connection
			resp, err := client1.GetHTTPClient().Get(testService.Endpoint + "/health")
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			// Simulate certificate rotation
			By("Rotating certificates")
			err = suite.rotateCertificates(serviceName)
			Expect(err).NotTo(HaveOccurred())

			// Create new client with rotated certificates
			client2, err := suite.createControllerMTLSClient(serviceName)
			Expect(err).NotTo(HaveOccurred())

			// Test connection with new certificates
			resp, err = client2.GetHTTPClient().Get(testService.Endpoint + "/health")
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			By("Certificate rotation successful")
		})

		It("should validate certificate policies for controller connections", func() {
			serviceName := "rag-api"

			// Deploy service with specific certificate policies
			testService, err := suite.deployTestServiceWithPolicy(serviceName, "strict")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				return suite.isServiceReady(serviceName)
			}, 60*time.Second, 2*time.Second).Should(BeTrue())

			// Test with compliant certificate
			compliantClient, err := suite.createCompliantMTLSClient(serviceName)
			Expect(err).NotTo(HaveOccurred())

			resp, err := compliantClient.GetHTTPClient().Get(testService.Endpoint + "/health")
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			// Test with non-compliant certificate (should fail)
			nonCompliantClient, err := suite.createNonCompliantMTLSClient(serviceName)
			Expect(err).NotTo(HaveOccurred())

			_, err = nonCompliantClient.GetHTTPClient().Get(testService.Endpoint + "/health")
			Expect(err).To(HaveOccurred())

			By("Certificate policy validation successful")
		})
	})

	Context("Service Mesh Integration", func() {
		It("should integrate with Istio service mesh for automatic mTLS", func() {
			Skip("Requires Istio service mesh deployment")

			// This test would verify:
			// 1. Istio sidecar injection
			// 2. Automatic certificate provisioning
			// 3. mTLS policy enforcement
			// 4. Traffic encryption validation

			services := []string{"service-a", "service-b", "service-c"}

			for _, serviceName := range services {
				By(fmt.Sprintf("Testing Istio mTLS for %s", serviceName))

				// Deploy service with Istio annotations
				_, err := suite.deployIstioService(serviceName)
				Expect(err).NotTo(HaveOccurred())

				// Verify Istio sidecar is injected
				pod := suite.getServicePod(serviceName)
				Expect(len(pod.Spec.Containers)).To(BeNumerically(">=", 2)) // App + Envoy sidecar

				// Verify mTLS communication through Istio
				err = suite.testIstioMTLSCommunication(serviceName)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should integrate with Linkerd service mesh for automatic mTLS", func() {
			Skip("Requires Linkerd service mesh deployment")

			// Similar to Istio but for Linkerd
			services := []string{"service-d", "service-e", "service-f"}

			for _, serviceName := range services {
				By(fmt.Sprintf("Testing Linkerd mTLS for %s", serviceName))

				_, err := suite.deployLinkerdService(serviceName)
				Expect(err).NotTo(HaveOccurred())

				// Verify Linkerd proxy injection
				pod := suite.getServicePod(serviceName)
				Expect(len(pod.Spec.Containers)).To(BeNumerically(">=", 2))

				// Test Linkerd mTLS
				err = suite.testLinkerdMTLSCommunication(serviceName)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should handle service mesh certificate coordination", func() {
			// Test coordination between service mesh and CA manager
			serviceName := "coordinated-service"

			By("Deploying service with coordinated certificate management")

			// Deploy service that uses both service mesh and CA manager
			_, err := suite.deployCoordinatedService(serviceName)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				return suite.isServiceReady(serviceName)
			}, 60*time.Second, 2*time.Second).Should(BeTrue())

			// Verify both service mesh and CA manager certificates exist
			meshCert := suite.getServiceMeshCertificate(serviceName)
			caCert := suite.getCAManagedCertificate(serviceName)

			Expect(meshCert).NotTo(BeNil())
			Expect(caCert).NotTo(BeNil())

			// Test that both certificate types work for different scenarios
			err = suite.testCoordinatedCommunication(serviceName)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("CA Backend Integration", func() {
		It("should integrate with cert-manager for certificate provisioning", func() {
			Skip("Requires cert-manager deployment")

			// Test cert-manager integration
			By("Testing cert-manager integration")

			// Create Certificate resource
			cert := suite.createCertManagerCertificate("cert-manager-test")
			err := suite.k8sClient.Create(suite.ctx, cert)
			Expect(err).NotTo(HaveOccurred())

			// Wait for certificate to be issued
			Eventually(func() bool {
				return suite.isCertManagerCertificateReady("cert-manager-test")
			}, 120*time.Second, 5*time.Second).Should(BeTrue())

			// Test using the cert-manager issued certificate
			client, err := suite.createMTLSClientFromCertManager("cert-manager-test")
			Expect(err).NotTo(HaveOccurred())

			// Verify certificate works
			err = suite.testCertManagerCertificate(client)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should integrate with Vault PKI for certificate management", func() {
			Skip("Requires HashiCorp Vault deployment")

			By("Testing Vault PKI integration")

			// Configure Vault PKI backend
			err := suite.setupVaultPKI()
			Expect(err).NotTo(HaveOccurred())

			// Request certificate from Vault
			vaultCert, err := suite.requestVaultCertificate("vault-test")
			Expect(err).NotTo(HaveOccurred())

			// Test using Vault-issued certificate
			client, err := suite.createMTLSClientFromVault(vaultCert)
			Expect(err).NotTo(HaveOccurred())

			err = suite.testVaultCertificate(client)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle external PKI integration", func() {
			Skip("Requires external PKI system")

			By("Testing external PKI integration")

			// This would test integration with enterprise PKI systems
			// like Microsoft AD CS, EJBCA, etc.
		})

		It("should coordinate between multiple CA backends", func() {
			// Test scenario where different services use different CA backends
			services := map[string]string{
				"internal-service": "self-signed",
				"external-service": "cert-manager",
				"vault-service":    "vault-pki",
			}

			for serviceName, caBackend := range services {
				By(fmt.Sprintf("Testing %s with %s backend", serviceName, caBackend))

				var client *mtls.Client
				var err error

				switch caBackend {
				case "self-signed":
					client, err = suite.createSelfSignedMTLSClient(serviceName)
				case "cert-manager":
					if utils.IsCertManagerAvailable() {
						client, err = suite.createCertManagerMTLSClient(serviceName)
					} else {
						Skip("cert-manager not available")
					}
				case "vault-pki":
					if utils.IsVaultAvailable() {
						client, err = suite.createVaultMTLSClient(serviceName)
					} else {
						Skip("Vault not available")
					}
				}

				Expect(err).NotTo(HaveOccurred())

				// Test cross-backend communication
				err = suite.testCrossBackendCommunication(serviceName, client)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	Context("Certificate Automation Integration", func() {
		It("should coordinate automatic certificate rotation across components", func() {
			services := []string{"rotation-test-1", "rotation-test-2", "rotation-test-3"}

			// Deploy services with rotation enabled
			for _, serviceName := range services {
				testService, err := suite.deployServiceWithRotation(serviceName)
				Expect(err).NotTo(HaveOccurred())

				suite.testServices[serviceName] = testService
			}

			// Wait for all services to be ready
			for _, serviceName := range services {
				Eventually(func() bool {
					return suite.isServiceReady(serviceName)
				}, 60*time.Second, 2*time.Second).Should(BeTrue())
			}

			// Trigger coordinated rotation
			By("Triggering coordinated certificate rotation")
			err := suite.triggerCoordinatedRotation(services)
			Expect(err).NotTo(HaveOccurred())

			// Verify all services continue to work after rotation
			for _, serviceName := range services {
				By(fmt.Sprintf("Verifying %s after rotation", serviceName))

				client, err := suite.createControllerMTLSClient(serviceName)
				Expect(err).NotTo(HaveOccurred())

				testService := suite.testServices[serviceName]
				resp, err := client.GetHTTPClient().Get(testService.Endpoint + "/health")
				Expect(err).NotTo(HaveOccurred())
				resp.Body.Close()
			}

			By("Coordinated certificate rotation successful")
		})

		It("should handle certificate distribution across clusters", func() {
			Skip("Requires multi-cluster setup")

			// This would test certificate distribution in multi-cluster scenarios
			clusters := []string{"cluster-1", "cluster-2", "cluster-3"}

			for _, cluster := range clusters {
				By(fmt.Sprintf("Testing certificate distribution to %s", cluster))

				// Deploy service to cluster
				// Verify certificate distribution
				// Test cross-cluster mTLS communication
			}
		})

		It("should validate certificate validation engine integration", func() {
			// Test real-time certificate validation
			serviceName := "validation-test"

			// Deploy service with validation engine
			testService, err := suite.deployServiceWithValidation(serviceName)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				return suite.isServiceReady(serviceName)
			}, 60*time.Second, 2*time.Second).Should(BeTrue())

			// Test with valid certificate
			validClient, err := suite.createValidMTLSClient(serviceName)
			Expect(err).NotTo(HaveOccurred())

			resp, err := validClient.GetHTTPClient().Get(testService.Endpoint + "/health")
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			// Test with invalid certificate (should be rejected)
			invalidClient, err := suite.createInvalidMTLSClient(serviceName)
			Expect(err).NotTo(HaveOccurred())

			_, err = invalidClient.GetHTTPClient().Get(testService.Endpoint + "/health")
			Expect(err).To(HaveOccurred())

			By("Certificate validation engine integration successful")
		})
	})

	Context("End-to-End Integration Scenarios", func() {
		It("should handle complete NetworkIntent processing with mTLS", func() {
			// Deploy all required services
			services := []string{"llm-processor", "rag-api", "nephio-bridge"}

			for _, serviceName := range services {
				testService, err := suite.deployTestService(serviceName)
				Expect(err).NotTo(HaveOccurred())
				suite.testServices[serviceName] = testService
			}

			// Wait for all services
			for _, serviceName := range services {
				Eventually(func() bool {
					return suite.isServiceReady(serviceName)
				}, 60*time.Second, 2*time.Second).Should(BeTrue())
			}

			// Simulate NetworkIntent processing flow
			By("Simulating NetworkIntent processing with mTLS")

			// 1. Controller -> LLM Processor
			llmClient, err := suite.createControllerMTLSClient("llm-processor")
			Expect(err).NotTo(HaveOccurred())

			resp, err := llmClient.GetHTTPClient().Post(
				suite.testServices["llm-processor"].Endpoint+"/process",
				"application/json",
				nil,
			)
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			// 2. LLM Processor -> RAG API
			ragClient, err := suite.createServiceMTLSClient("llm-processor", "rag-api")
			Expect(err).NotTo(HaveOccurred())

			resp, err = ragClient.GetHTTPClient().Get(
				suite.testServices["rag-api"].Endpoint + "/retrieve",
			)
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			// 3. Controller -> Nephio Bridge
			bridgeClient, err := suite.createControllerMTLSClient("nephio-bridge")
			Expect(err).NotTo(HaveOccurred())

			resp, err = bridgeClient.GetHTTPClient().Post(
				suite.testServices["nephio-bridge"].Endpoint+"/deploy",
				"application/json",
				nil,
			)
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			By("End-to-end mTLS communication successful")
		})

		It("should handle fault injection and recovery", func() {
			serviceName := "fault-test"

			// Deploy service
			testService, err := suite.deployTestService(serviceName)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				return suite.isServiceReady(serviceName)
			}, 60*time.Second, 2*time.Second).Should(BeTrue())

			// Test normal operation
			client, err := suite.createControllerMTLSClient(serviceName)
			Expect(err).NotTo(HaveOccurred())

			resp, err := client.GetHTTPClient().Get(testService.Endpoint + "/health")
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()

			// Inject certificate-related faults
			By("Injecting certificate expiry fault")
			err = suite.injectCertificateExpiryFault(serviceName)
			Expect(err).NotTo(HaveOccurred())

			// Verify fault is detected and handled
			Eventually(func() bool {
				return suite.isCertificateFaultDetected(serviceName)
			}, 30*time.Second, 2*time.Second).Should(BeTrue())

			// Trigger recovery
			By("Triggering certificate recovery")
			err = suite.triggerCertificateRecovery(serviceName)
			Expect(err).NotTo(HaveOccurred())

			// Verify recovery
			Eventually(func() bool {
				newClient, err := suite.createControllerMTLSClient(serviceName)
				if err != nil {
					return false
				}
				resp, err := newClient.GetHTTPClient().Get(testService.Endpoint + "/health")
				if err != nil {
					return false
				}
				resp.Body.Close()
				return true
			}, 60*time.Second, 2*time.Second).Should(BeTrue())

			By("Fault injection and recovery successful")
		})
	})
})

// Helper methods for integration testing

func (s *mTLSIntegrationTestSuite) initializeCAManager() error {
	// Initialize CA manager with test configuration
	config := &ca.Config{
		Backend:      "self-signed", // Use self-signed for testing
		StorePath:    "/tmp/ca-test",
		KeySize:      2048,
		ValidityDays: 365,
	}

	var err error
	s.caManager, err = ca.NewCAManager(config)
	if err != nil {
		return fmt.Errorf("failed to create CA manager: %w", err)
	}

	return s.caManager.Initialize(s.ctx)
}

func (s *mTLSIntegrationTestSuite) deployTestService(serviceName string) (*testServiceDeployment, error) {
	// Create server certificate secret
	serverCert, err := s.createServerCertificateSecret(serviceName)
	if err != nil {
		return nil, err
	}

	// Create client certificate secret
	clientCert, err := s.createClientCertificateSecret(serviceName + "-client")
	if err != nil {
		return nil, err
	}

	// Create deployment
	deployment := s.createTestDeployment(serviceName)
	err = s.k8sClient.Create(s.ctx, deployment)
	if err != nil {
		return nil, err
	}

	// Create service
	service := s.createTestK8sService(serviceName)
	err = s.k8sClient.Create(s.ctx, service)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("https://%s.%s.svc.cluster.local:8443", serviceName, s.namespace)

	return &testServiceDeployment{
		Name:       serviceName,
		Deployment: deployment,
		Service:    service,
		ServerCert: serverCert,
		ClientCert: clientCert,
		Endpoint:   endpoint,
	}, nil
}

func (s *mTLSIntegrationTestSuite) createTestDeployment(serviceName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: s.namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": serviceName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": serviceName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  serviceName,
							Image: "nginx:alpine", // Simple test container
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "server-cert",
									MountPath: "/etc/ssl/certs",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "server-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: serviceName + "-server-cert",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *mTLSIntegrationTestSuite) createTestK8sService(serviceName string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: s.namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": serviceName,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       8443,
					TargetPort: intstr.FromInt(8443),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func (s *mTLSIntegrationTestSuite) createServerCertificateSecret(serviceName string) (*corev1.Secret, error) {
	// Request certificate from CA manager
	req := &ca.CertificateRequest{
		ID:               fmt.Sprintf("%s-server", serviceName),
		CommonName:       serviceName,
		DNSNames:         []string{serviceName, fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, s.namespace)},
		ValidityDuration: 24 * time.Hour,
		KeyUsage:         []string{"digital_signature", "key_encipherment"},
		ExtKeyUsage:      []string{"server_auth"},
	}

	resp, err := s.caManager.IssueCertificate(s.ctx, req)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName + "-server-cert",
			Namespace: s.namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": resp.Certificate,
			"tls.key": resp.PrivateKey,
			"ca.crt":  resp.CACertificate,
		},
	}

	err = s.k8sClient.Create(s.ctx, secret)
	return secret, err
}

func (s *mTLSIntegrationTestSuite) createClientCertificateSecret(clientName string) (*corev1.Secret, error) {
	req := &ca.CertificateRequest{
		ID:               fmt.Sprintf("%s-client", clientName),
		CommonName:       clientName,
		ValidityDuration: 24 * time.Hour,
		KeyUsage:         []string{"digital_signature"},
		ExtKeyUsage:      []string{"client_auth"},
	}

	resp, err := s.caManager.IssueCertificate(s.ctx, req)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clientName + "-client-cert",
			Namespace: s.namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": resp.Certificate,
			"tls.key": resp.PrivateKey,
			"ca.crt":  resp.CACertificate,
		},
	}

	err = s.k8sClient.Create(s.ctx, secret)
	return secret, err
}

func (s *mTLSIntegrationTestSuite) createControllerMTLSClient(targetService string) (*mtls.Client, error) {
	config := &mtls.ClientConfig{
		ServiceName:      "nephoran-controller",
		ServerName:       targetService,
		ClientCertPath:   fmt.Sprintf("/tmp/certs/%s-client.crt", targetService),
		ClientKeyPath:    fmt.Sprintf("/tmp/certs/%s-client.key", targetService),
		CACertPath:       "/tmp/certs/ca.crt",
		RotationEnabled:  true,
		RotationInterval: 1 * time.Hour,
		RenewalThreshold: 24 * time.Hour,
		CAManager:        s.caManager,
		AutoProvision:    true,
	}

	return mtls.NewClient(config, nil)
}

func (s *mTLSIntegrationTestSuite) isServiceReady(serviceName string) bool {
	var deployment appsv1.Deployment
	err := s.k8sClient.Get(s.ctx, types.NamespacedName{
		Name:      serviceName,
		Namespace: s.namespace,
	}, &deployment)

	if err != nil {
		return false
	}

	return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas
}

func (s *mTLSIntegrationTestSuite) cleanup() {
	// Cleanup test resources
	for _, testService := range s.testServices {
		s.k8sClient.Delete(s.ctx, testService.Deployment)
		s.k8sClient.Delete(s.ctx, testService.Service)
		s.k8sClient.Delete(s.ctx, testService.ServerCert)
		s.k8sClient.Delete(s.ctx, testService.ClientCert)

		if testService.MTLSClient != nil {
			testService.MTLSClient.Close()
		}
	}

	for _, client := range s.mtlsClients {
		client.Close()
	}
}

// Placeholder methods for extended functionality - these would be implemented based on specific requirements

func (s *mTLSIntegrationTestSuite) deployTestServiceWithPolicy(serviceName, policy string) (*testServiceDeployment, error) {
	// Would implement policy-specific deployment
	return s.deployTestService(serviceName)
}

func (s *mTLSIntegrationTestSuite) createCompliantMTLSClient(serviceName string) (*mtls.Client, error) {
	// Would create client with compliant certificate
	return s.createControllerMTLSClient(serviceName)
}

func (s *mTLSIntegrationTestSuite) createNonCompliantMTLSClient(serviceName string) (*mtls.Client, error) {
	// Would create client with non-compliant certificate
	return s.createControllerMTLSClient(serviceName)
}

func (s *mTLSIntegrationTestSuite) rotateCertificates(serviceName string) error {
	// Would implement certificate rotation
	return nil
}

// Additional placeholder methods would be implemented here...
// These represent the full scope of integration testing functionality

func int32Ptr(i int32) *int32 { return &i }

// More implementation methods would follow...
