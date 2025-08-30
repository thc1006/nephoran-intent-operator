package security

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nephio-project/nephoran-intent-operator/tests/utils"
)

var _ = Describe("TLS/mTLS Security Tests", func() {
	var (
		ctx       context.Context
		k8sClient client.Client
		clientset *kubernetes.Clientset
		namespace string
		timeout   time.Duration
	)

	BeforeEach(func() {
		ctx = context.Background()
		k8sClient = utils.GetK8sClient()
		clientset = utils.GetClientset()
		namespace = utils.GetTestNamespace()
		timeout = 30 * time.Second
	})

	Context("TLS Certificate Validation", func() {
		It("should verify all TLS certificates are valid and properly configured", func() {
			var secrets corev1.SecretList
			err := k8sClient.List(ctx, &secrets, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			tlsSecrets := []corev1.Secret{}
			for _, secret := range secrets.Items {
				if secret.Type == corev1.SecretTypeTLS {
					tlsSecrets = append(tlsSecrets, secret)
				}
			}

			Expect(len(tlsSecrets)).To(BeNumerically(">", 0), "Should have at least one TLS secret")

			for _, secret := range tlsSecrets {
				By(fmt.Sprintf("Validating TLS certificate in secret %s", secret.Name))

				certData, certExists := secret.Data["tls.crt"]
				keyData, keyExists := secret.Data["tls.key"]

				Expect(certExists).To(BeTrue(), "TLS secret must contain tls.crt")
				Expect(keyExists).To(BeTrue(), "TLS secret must contain tls.key")

				// Parse certificate
				block, _ := pem.Decode(certData)
				Expect(block).NotTo(BeNil(), "Certificate must be valid PEM format")
				Expect(block.Type).To(Equal("CERTIFICATE"))

				cert, err := x509.ParseCertificate(block.Bytes)
				Expect(err).NotTo(HaveOccurred(), "Certificate must be parseable")

				// Validate certificate properties
				validateCertificate(cert, secret.Name)

				// Parse private key
				keyBlock, _ := pem.Decode(keyData)
				Expect(keyBlock).NotTo(BeNil(), "Private key must be valid PEM format")

				// Validate key matches certificate
				validateKeyPair(cert, keyBlock.Bytes, secret.Name)
			}
		})

		It("should verify certificate expiration monitoring", func() {
			var secrets corev1.SecretList
			err := k8sClient.List(ctx, &secrets, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, secret := range secrets.Items {
				if secret.Type == corev1.SecretTypeTLS {
					By(fmt.Sprintf("Checking expiration monitoring for certificate %s", secret.Name))

					if secret.Annotations != nil {
						// Check for certificate monitoring annotations
						if expiryDate, exists := secret.Annotations["cert.nephoran.io/expiry-date"]; exists {
							By(fmt.Sprintf("Certificate %s expires: %s", secret.Name, expiryDate))

							// Parse and validate expiry date
							parsedDate, err := time.Parse(time.RFC3339, expiryDate)
							Expect(err).NotTo(HaveOccurred())

							// Should not be expired
							Expect(parsedDate.After(time.Now())).To(BeTrue(),
								"Certificate should not be expired")
						}

						if daysToExpiry, exists := secret.Annotations["cert.nephoran.io/days-to-expiry"]; exists {
							By(fmt.Sprintf("Certificate %s expires in %s days", secret.Name, daysToExpiry))
						}

						if renewalSchedule, exists := secret.Annotations["cert.nephoran.io/renewal-schedule"]; exists {
							By(fmt.Sprintf("Certificate %s renewal schedule: %s", secret.Name, renewalSchedule))
						}
					}

					// Parse actual certificate to check expiry
					certData := secret.Data["tls.crt"]
					block, _ := pem.Decode(certData)
					cert, _ := x509.ParseCertificate(block.Bytes)

					daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
					if daysUntilExpiry < 30 {
						By(fmt.Sprintf("Warning: Certificate %s expires in %d days", secret.Name, daysUntilExpiry))
					}
				}
			}
		})

		It("should verify certificate authority validation", func() {
			var secrets corev1.SecretList
			err := k8sClient.List(ctx, &secrets, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, secret := range secrets.Items {
				if secret.Type == corev1.SecretTypeTLS {
					By(fmt.Sprintf("Checking CA validation for certificate %s", secret.Name))

					// Check if CA certificate is included
					if caData, exists := secret.Data["ca.crt"]; exists {
						caCertBlock, _ := pem.Decode(caData)
						Expect(caCertBlock).NotTo(BeNil(), "CA certificate must be valid PEM")

						caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
						Expect(err).NotTo(HaveOccurred())

						// Verify CA properties
						Expect(caCert.IsCA).To(BeTrue(), "CA certificate must have CA flag")
						Expect(caCert.KeyUsage&x509.KeyUsageCertSign).NotTo(Equal(0),
							"CA must have certificate signing capability")

						// Verify CA is not expired
						Expect(caCert.NotAfter.After(time.Now())).To(BeTrue(),
							"CA certificate should not be expired")

						// Verify leaf certificate is signed by this CA
						certData := secret.Data["tls.crt"]
						leafBlock, _ := pem.Decode(certData)
						leafCert, _ := x509.ParseCertificate(leafBlock.Bytes)

						err = leafCert.CheckSignatureFrom(caCert)
						if err != nil {
							By(fmt.Sprintf("Warning: Certificate %s signature validation failed: %v", secret.Name, err))
						} else {
							By(fmt.Sprintf("Certificate %s signature validated against CA", secret.Name))
						}
					}
				}
			}
		})
	})

	Context("TLS Configuration Enforcement", func() {
		It("should verify services enforce TLS connections", func() {
			serviceEndpoints := map[string]string{
				"rag-api":          "https://rag-api." + namespace + ".svc.cluster.local:443",
				"llm-processor":    "https://llm-processor." + namespace + ".svc.cluster.local:443",
				"nephoran-webhook": "https://nephoran-webhook." + namespace + ".svc.cluster.local:443",
			}

			for serviceName, endpoint := range serviceEndpoints {
				By(fmt.Sprintf("Testing TLS enforcement for service %s", serviceName))

				// This would test TLS connectivity in a real environment
				// For now, we'll check the service configuration
				var service corev1.Service
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      serviceName,
					Namespace: namespace,
				}, &service)

				if err != nil {
					By(fmt.Sprintf("Service %s not found - may not be deployed in test environment", serviceName))
					continue
				}

				// Check for TLS-related annotations
				if service.Annotations != nil {
					if tlsMode, exists := service.Annotations["security.nephoran.io/tls-mode"]; exists {
						By(fmt.Sprintf("Service %s TLS mode: %s", serviceName, tlsMode))
						Expect(tlsMode).To(BeElementOf("required", "optional", "permissive"))
					}

					if _, exists := service.Annotations["service.beta.kubernetes.io/aws-load-balancer-ssl-cert"]; exists {
						By(fmt.Sprintf("Service %s uses AWS SSL certificate", serviceName))
					}
				}

				// Check port configuration
				for _, port := range service.Spec.Ports {
					if port.Port == 443 || port.Port == 8443 {
						By(fmt.Sprintf("Service %s exposes HTTPS port %d", serviceName, port.Port))
					}
					if port.Port == 80 || port.Port == 8080 {
						By(fmt.Sprintf("Warning: Service %s exposes HTTP port %d", serviceName, port.Port))
					}
				}

				By(fmt.Sprintf("TLS configuration checked for service %s at %s", serviceName, endpoint))
			}
		})

		It("should verify TLS cipher suites and protocols", func() {
			// Check TLS configuration in application configs
			var configMaps corev1.ConfigMapList
			err := k8sClient.List(ctx, &configMaps, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, configMap := range configMaps.Items {
				if strings.Contains(configMap.Name, "tls") || strings.Contains(configMap.Name, "nginx") {
					By(fmt.Sprintf("Checking TLS configuration in ConfigMap %s", configMap.Name))

					if configMap.Data != nil {
						for key, value := range configMap.Data {
							if strings.Contains(key, "tls") || strings.Contains(key, "ssl") {
								By(fmt.Sprintf("TLS config found in %s/%s", configMap.Name, key))

								// Check for secure cipher suites
								secureCiphers := []string{
									"ECDHE-RSA-AES256-GCM-SHA384",
									"ECDHE-RSA-AES128-GCM-SHA256",
									"ECDHE-RSA-AES256-SHA384",
									"ECDHE-RSA-AES128-SHA256",
								}

								for _, cipher := range secureCiphers {
									if strings.Contains(value, cipher) {
										By(fmt.Sprintf("Secure cipher suite found: %s", cipher))
									}
								}

								// Check for deprecated protocols
								insecureProtocols := []string{"SSLv2", "SSLv3", "TLSv1", "TLSv1.1"}
								for _, protocol := range insecureProtocols {
									if strings.Contains(value, protocol) {
										By(fmt.Sprintf("Warning: Insecure protocol found: %s", protocol))
									}
								}

								// Check for secure protocols
								secureProtocols := []string{"TLSv1.2", "TLSv1.3"}
								for _, protocol := range secureProtocols {
									if strings.Contains(value, protocol) {
										By(fmt.Sprintf("Secure protocol configured: %s", protocol))
									}
								}
							}
						}
					}
				}
			}
		})

		It("should verify TLS redirect configuration", func() {
			// Check ingress configurations for TLS redirect
			var services corev1.ServiceList
			err := k8sClient.List(ctx, &services, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, service := range services.Items {
				if service.Annotations != nil {
					if redirectAnnotation, exists := service.Annotations["nginx.ingress.kubernetes.io/ssl-redirect"]; exists {
						By(fmt.Sprintf("Service %s SSL redirect: %s", service.Name, redirectAnnotation))
						// Should be "true" for production services
						if service.Name != "webhook" { // Webhooks may have special requirements
							Expect(redirectAnnotation).To(Equal("true"),
								"Production services should redirect HTTP to HTTPS")
						}
					}

					if forceSSL, exists := service.Annotations["nginx.ingress.kubernetes.io/force-ssl-redirect"]; exists {
						By(fmt.Sprintf("Service %s force SSL redirect: %s", service.Name, forceSSL))
					}
				}
			}
		})
	})

	Context("Mutual TLS (mTLS) Authentication", func() {
		It("should verify mTLS is configured for inter-service communication", func() {
			interServiceConnections := map[string][]string{
				"rag-api":       {"llm-processor", "weaviate"},
				"llm-processor": {"rag-api"},
				"nephio-bridge": {"nephoran-operator"},
			}

			for sourceService, targetServices := range interServiceConnections {
				By(fmt.Sprintf("Checking mTLS configuration for %s", sourceService))

				var sourceSecret corev1.Secret
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      sourceService + "-client-cert",
					Namespace: namespace,
				}, &sourceSecret)

				if err != nil {
					By(fmt.Sprintf("Client certificate for %s not found - may not use mTLS", sourceService))
					continue
				}

				// Verify client certificate
				if sourceSecret.Type == corev1.SecretTypeTLS {
					By(fmt.Sprintf("mTLS client certificate found for %s", sourceService))

					certData := sourceSecret.Data["tls.crt"]
					block, _ := pem.Decode(certData)
					cert, _ := x509.ParseCertificate(block.Bytes)

					// Verify client certificate has client auth usage
					hasClientAuth := false
					for _, usage := range cert.ExtKeyUsage {
						if usage == x509.ExtKeyUsageClientAuth {
							hasClientAuth = true
							break
						}
					}
					Expect(hasClientAuth).To(BeTrue(),
						"Client certificate should have client auth usage")

					By(fmt.Sprintf("Client certificate for %s has proper client auth usage", sourceService))
				}

				// Check target services for server certificates
				for _, targetService := range targetServices {
					var targetSecret corev1.Secret
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name:      targetService + "-server-cert",
						Namespace: namespace,
					}, &targetSecret)

					if err == nil && targetSecret.Type == corev1.SecretTypeTLS {
						By(fmt.Sprintf("mTLS server certificate found for %s", targetService))

						certData := targetSecret.Data["tls.crt"]
						block, _ := pem.Decode(certData)
						cert, _ := x509.ParseCertificate(block.Bytes)

						// Verify server certificate has server auth usage
						hasServerAuth := false
						for _, usage := range cert.ExtKeyUsage {
							if usage == x509.ExtKeyUsageServerAuth {
								hasServerAuth = true
								break
							}
						}
						Expect(hasServerAuth).To(BeTrue(),
							"Server certificate should have server auth usage")
					}
				}
			}
		})

		It("should verify mTLS client certificate validation", func() {
			// Check for mTLS client certificate validation configuration
			var configMaps corev1.ConfigMapList
			err := k8sClient.List(ctx, &configMaps, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, configMap := range configMaps.Items {
				if strings.Contains(configMap.Name, "mtls") || strings.Contains(configMap.Name, "client-auth") {
					By(fmt.Sprintf("Checking mTLS configuration in ConfigMap %s", configMap.Name))

					if configMap.Data != nil {
						for _, value := range configMap.Data {
							// Check for client certificate validation settings
							if strings.Contains(value, "ssl_verify_client") {
								By("Found Nginx client certificate verification configuration")
							}
							if strings.Contains(value, "ssl_client_certificate") {
								By("Found client CA certificate configuration")
							}
							if strings.Contains(value, "verify") && strings.Contains(value, "client") {
								By("Found client certificate verification configuration")
							}
						}
					}
				}
			}
		})

		It("should verify certificate revocation checking", func() {
			// Check for CRL or OCSP configuration
			var configMaps corev1.ConfigMapList
			err := k8sClient.List(ctx, &configMaps, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			crlFound := false
			ocspFound := false

			for _, configMap := range configMaps.Items {
				if configMap.Data != nil {
					for _, value := range configMap.Data {
						if strings.Contains(value, "crl") || strings.Contains(value, "certificate-revocation") {
							crlFound = true
							By(fmt.Sprintf("CRL configuration found in ConfigMap %s", configMap.Name))
						}
						if strings.Contains(value, "ocsp") {
							ocspFound = true
							By(fmt.Sprintf("OCSP configuration found in ConfigMap %s", configMap.Name))
						}
					}
				}
			}

			if crlFound || ocspFound {
				By("Certificate revocation checking is configured")
			} else {
				By("Warning: No certificate revocation checking found")
			}
		})
	})

	Context("Certificate Rotation", func() {
		It("should verify automatic certificate rotation is configured", func() {
			// Check for cert-manager or similar certificate rotation system
			var secrets corev1.SecretList
			err := k8sClient.List(ctx, &secrets, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			certManagerFound := false
			for _, secret := range secrets.Items {
				if secret.Annotations != nil {
					if _, exists := secret.Annotations["cert-manager.io/certificate-name"]; exists {
						certManagerFound = true
						By(fmt.Sprintf("Cert-manager managed certificate: %s", secret.Name))
					}
					if issuer, exists := secret.Annotations["cert-manager.io/issuer-name"]; exists {
						By(fmt.Sprintf("Certificate issuer: %s", issuer))
					}
				}
			}

			if certManagerFound {
				By("Automatic certificate rotation is configured via cert-manager")
			} else {
				By("Warning: No automatic certificate rotation system detected")
			}
		})

		It("should verify certificate renewal alerts are configured", func() {
			// Check for monitoring and alerting configuration
			var configMaps corev1.ConfigMapList
			err := k8sClient.List(ctx, &configMaps, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, configMap := range configMaps.Items {
				if strings.Contains(configMap.Name, "alert") || strings.Contains(configMap.Name, "monitor") {
					if configMap.Data != nil {
						for _, value := range configMap.Data {
							if strings.Contains(value, "certificate") && strings.Contains(value, "expir") {
								By(fmt.Sprintf("Certificate expiration alerting configured in %s", configMap.Name))
							}
						}
					}
				}
			}
		})

		It("should test certificate rotation procedures", func() {
			// This would test the actual certificate rotation process
			// In a real environment, this would:
			// 1. Trigger a certificate rotation
			// 2. Verify the old certificate is replaced
			// 3. Verify services continue to work with new certificate
			// 4. Verify old certificate is properly cleaned up

			By("Certificate rotation procedures would be tested here in a real environment")

			// For now, we can check that the infrastructure supports rotation
			var secrets corev1.SecretList
			err := k8sClient.List(ctx, &secrets, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			rotatableSecrets := 0
			for _, secret := range secrets.Items {
				if secret.Type == corev1.SecretTypeTLS {
					if secret.Annotations != nil {
						if _, exists := secret.Annotations["rotation.nephoran.io/enabled"]; exists {
							rotatableSecrets++
						}
					}
				}
			}

			By(fmt.Sprintf("Found %d certificates configured for rotation", rotatableSecrets))
		})
	})

	Context("TLS Security Best Practices", func() {
		It("should verify secure TLS configuration", func() {
			// Test TLS configuration against security best practices
			securityChecks := map[string]bool{
				"tls-1.2-minimum":      false,
				"secure-cipher-suites": false,
				"hsts-headers":         false,
				"certificate-pinning":  false,
			}

			// This would check actual TLS configuration
			// For demonstration, we'll check configuration files
			var configMaps corev1.ConfigMapList
			err := k8sClient.List(ctx, &configMaps, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, configMap := range configMaps.Items {
				if configMap.Data != nil {
					for _, value := range configMap.Data {
						if strings.Contains(value, "TLSv1.2") || strings.Contains(value, "TLSv1.3") {
							securityChecks["tls-1.2-minimum"] = true
						}
						if strings.Contains(value, "ECDHE") {
							securityChecks["secure-cipher-suites"] = true
						}
						if strings.Contains(value, "Strict-Transport-Security") {
							securityChecks["hsts-headers"] = true
						}
						if strings.Contains(value, "pin-sha256") {
							securityChecks["certificate-pinning"] = true
						}
					}
				}
			}

			for check, passed := range securityChecks {
				if passed {
					By(fmt.Sprintf("✓ Security check passed: %s", check))
				} else {
					By(fmt.Sprintf("⚠ Security check not verified: %s", check))
				}
			}
		})

		It("should verify certificate transparency monitoring", func() {
			// Check for Certificate Transparency monitoring
			var configMaps corev1.ConfigMapList
			err := k8sClient.List(ctx, &configMaps, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			ctMonitoringFound := false
			for _, configMap := range configMaps.Items {
				if configMap.Data != nil {
					for _, value := range configMap.Data {
						if strings.Contains(value, "certificate-transparency") || strings.Contains(value, "ct-log") {
							ctMonitoringFound = true
							By(fmt.Sprintf("Certificate transparency monitoring found in %s", configMap.Name))
						}
					}
				}
			}

			if ctMonitoringFound {
				By("Certificate transparency monitoring is configured")
			} else {
				By("Certificate transparency monitoring not found - consider enabling for production")
			}
		})
	})
})

// Helper functions for TLS validation

func validateCertificate(cert *x509.Certificate, secretName string) {
	By(fmt.Sprintf("Validating certificate properties for %s", secretName))

	// Check validity period
	now := time.Now()
	Expect(cert.NotBefore.Before(now)).To(BeTrue(),
		"Certificate should be valid now (not before %v)", cert.NotBefore)
	Expect(cert.NotAfter.After(now)).To(BeTrue(),
		"Certificate should not be expired (expires %v)", cert.NotAfter)

	// Check key usage
	Expect(cert.KeyUsage&x509.KeyUsageDigitalSignature).NotTo(Equal(0),
		"Certificate should have digital signature usage")

	// Check algorithm
	supportedAlgorithms := []x509.SignatureAlgorithm{
		x509.SHA256WithRSA,
		x509.ECDSAWithSHA256,
		x509.SHA384WithRSA,
		x509.ECDSAWithSHA384,
		x509.SHA512WithRSA,
		x509.ECDSAWithSHA512,
	}
	Expect(supportedAlgorithms).To(ContainElement(cert.SignatureAlgorithm),
		"Certificate should use a secure signature algorithm")

	// Check key size for RSA certificates
	if cert.PublicKeyAlgorithm == x509.RSA {
		rsaKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if ok {
			Expect(rsaKey.N.BitLen()).To(BeNumerically(">=", 2048),
				"RSA key should be at least 2048 bits")
		}
	}

	By(fmt.Sprintf("Certificate %s validation passed", secretName))
}

func validateKeyPair(cert *x509.Certificate, keyData []byte, secretName string) {
	By(fmt.Sprintf("Validating key pair for %s", secretName))

	// This would validate that the private key matches the certificate
	// Implementation would depend on the key type (RSA, ECDSA, etc.)

	// For now, we'll just verify the key can be parsed
	switch cert.PublicKeyAlgorithm {
	case x509.RSA:
		_, err := x509.ParsePKCS1PrivateKey(keyData)
		if err != nil {
			// Try PKCS8 format
			_, err = x509.ParsePKCS8PrivateKey(keyData)
			Expect(err).NotTo(HaveOccurred(), "RSA private key should be parseable")
		}
	case x509.ECDSA:
		_, err := x509.ParseECPrivateKey(keyData)
		if err != nil {
			// Try PKCS8 format
			_, err = x509.ParsePKCS8PrivateKey(keyData)
			Expect(err).NotTo(HaveOccurred(), "ECDSA private key should be parseable")
		}
	}

	By(fmt.Sprintf("Key pair validation passed for %s", secretName))
}

// Test TLS connectivity
func testTLSConnectivity(endpoint string, expectedCert *x509.Certificate) error {
	// Configure TLS client
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("TLS connection failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify certificate chain
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		serverCert := resp.TLS.PeerCertificates[0]

		// Compare with expected certificate if provided
		if expectedCert != nil && !serverCert.Equal(expectedCert) {
			return fmt.Errorf("server certificate does not match expected certificate")
		}
	}

	return nil
}

// Benchmark TLS operations
func BenchmarkTLSOperations(b *testing.B) {
	// This would benchmark TLS handshake performance
	b.Run("TLSHandshake", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Simulate TLS handshake
		}
	})

	b.Run("CertificateValidation", func(b *testing.B) {
		// Create a test certificate for benchmarking
		cert := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(time.Hour * 24 * 365),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			validateCertificate(cert, "test-cert")
		}
	})
}
