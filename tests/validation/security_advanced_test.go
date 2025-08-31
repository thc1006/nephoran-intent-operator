// Package validation provides advanced security testing scenarios
package test_validation

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nephio-project/nephoran-intent-operator/tests/framework"
)

var _ = ginkgo.Describe("Advanced Security Testing Suite", ginkgo.Ordered, func() {
	var (
		testSuite    *framework.TestSuite
		ctx          context.Context
		cancel       context.CancelFunc
		k8sClient    client.Client
		secValidator *SecurityValidator
	)

	ginkgo.BeforeAll(func() {
		ginkgo.By("Initializing Advanced Security Test Suite")
		testConfig := framework.DefaultTestConfig()
		testConfig.MockExternalAPIs = true
		testSuite = framework.NewTestSuite(testConfig)
		testSuite.SetupSuite()

		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Minute)
		k8sClient = testSuite.GetK8sClient()

		config := DefaultValidationConfig()
		config.EnableSecurityTesting = true
		secValidator = NewSecurityValidator(config)
		secValidator.SetK8sClient(k8sClient)
	})

	ginkgo.AfterAll(func() {
		cancel()
		testSuite.TearDownSuite()
	})

	ginkgo.Context("Advanced Authentication Mechanisms", func() {
		ginkgo.It("should implement JWT token validation with RSA signatures", func() {
			ginkgo.By("Testing JWT token validation and signature verification")

			// Generate RSA key pair for JWT signing
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create JWT signing keys secret
			jwtSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jwt-signing-keys",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/jwt": "signing",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"private.key": x509.MarshalPKCS1PrivateKey(privateKey),
					"public.key":  publicKeyBytes,
					"algorithm":   []byte("RS256"),
					"issuer":      []byte("nephoran-intent-operator"),
					"audience":    []byte("nephoran-api"),
				},
			}

			err = k8sClient.Create(ctx, jwtSecret)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create JWT validation configuration
			jwtConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jwt-validation-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"validate_signature": "true",
					"validate_expiry":    "true",
					"validate_nbf":       "true",
					"validate_issuer":    "true",
					"validate_audience":  "true",
					"clock_skew":         "60s",
					"max_token_age":      "24h",
				},
			}

			err = k8sClient.Create(ctx, jwtConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})

// Helper function to generate API keys
func generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// Helper function to generate test certificates
func generateTestCertificate(cn string) (certPEM, keyPEM []byte, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return certPEM, keyPEM, nil
}
