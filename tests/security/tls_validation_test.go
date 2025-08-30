// Package security provides comprehensive TLS/mTLS security validation tests
package security

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nephio-project/nephoran-intent-operator/pkg/security"
)

var _ = Describe("TLS Security Validation Suite", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	})

	AfterEach(func() {
		cancel()
	})

	Context("TLS Version Enforcement", func() {
		It("should reject TLS versions below 1.2", func() {
			// Create test server with TLS 1.2 minimum
			tlsConfig := &tls.Config{
				MinVersion: tls.VersionTLS12,
				MaxVersion: tls.VersionTLS13,
			}

			listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			go func() {
				conn, _ := listener.Accept()
				if conn != nil {
					conn.Close()
				}
			}()

			// Try to connect with TLS 1.1
			clientConfig := &tls.Config{
				MaxVersion:         tls.VersionTLS11,
				InsecureSkipVerify: true,
			}

			conn, err := tls.Dial("tcp", listener.Addr().String(), clientConfig)
			if conn != nil {
				conn.Close()
			}

			// Should fail due to version mismatch
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("protocol version"))
		})

		It("should enforce TLS 1.3 when configured as minimum", func() {
			// Create test server with TLS 1.3 minimum
			tlsConfig := &tls.Config{
				MinVersion: tls.VersionTLS13,
				MaxVersion: tls.VersionTLS13,
			}

			listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			serverReady := make(chan bool)
			go func() {
				serverReady <- true
				conn, _ := listener.Accept()
				if conn != nil {
					tlsConn := conn.(*tls.Conn)
					tlsConn.Handshake()
					state := tlsConn.ConnectionState()
					Expect(state.Version).To(Equal(uint16(tls.VersionTLS13)))
					conn.Close()
				}
			}()

			<-serverReady

			// Connect with TLS 1.3
			clientConfig := &tls.Config{
				MinVersion:         tls.VersionTLS13,
				InsecureSkipVerify: true,
			}

			conn, err := tls.Dial("tcp", listener.Addr().String(), clientConfig)
			Expect(err).NotTo(HaveOccurred())
			if conn != nil {
				state := conn.ConnectionState()
				Expect(state.Version).To(Equal(uint16(tls.VersionTLS13)))
				conn.Close()
			}
		})
	})

	Context("Cipher Suite Validation", func() {
		It("should reject weak cipher suites", func() {
			// List of weak ciphers that should be rejected
			weakCiphers := []uint16{
				tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
				tls.TLS_RSA_WITH_RC4_128_SHA,
			}

			// Create server with only strong ciphers
			strongCiphers := []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			}

			serverConfig := &tls.Config{
				MinVersion:   tls.VersionTLS12,
				CipherSuites: strongCiphers,
			}

			listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			go func() {
				conn, _ := listener.Accept()
				if conn != nil {
					conn.Close()
				}
			}()

			// Try to connect with weak ciphers only
			clientConfig := &tls.Config{
				MinVersion:         tls.VersionTLS12,
				CipherSuites:       weakCiphers,
				InsecureSkipVerify: true,
			}

			conn, err := tls.Dial("tcp", listener.Addr().String(), clientConfig)
			if conn != nil {
				conn.Close()
			}

			// Should fail due to no common cipher suites
			Expect(err).To(HaveOccurred())
		})

		It("should prefer ChaCha20-Poly1305 for mobile clients", func() {
			// Server config with ChaCha20 preference
			serverConfig := &tls.Config{
				MinVersion: tls.VersionTLS12,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				},
				PreferServerCipherSuites: true,
			}

			listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			cipherUsed := make(chan uint16, 1)
			go func() {
				conn, _ := listener.Accept()
				if conn != nil {
					tlsConn := conn.(*tls.Conn)
					tlsConn.Handshake()
					state := tlsConn.ConnectionState()
					cipherUsed <- state.CipherSuite
					conn.Close()
				}
			}()

			// Client offers ChaCha20
			clientConfig := &tls.Config{
				MinVersion: tls.VersionTLS12,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
				},
				InsecureSkipVerify: true,
			}

			conn, err := tls.Dial("tcp", listener.Addr().String(), clientConfig)
			Expect(err).NotTo(HaveOccurred())
			if conn != nil {
				conn.Close()
			}

			// Verify ChaCha20 was selected
			select {
			case cipher := <-cipherUsed:
				Expect(cipher).To(Equal(uint16(tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256)))
			case <-time.After(5 * time.Second):
				Fail("Timeout waiting for cipher suite")
			}
		})
	})

	Context("Certificate Validation", func() {
		It("should validate certificate chain properly", func() {
			// Generate test certificate chain
			rootCert, rootKey := generateTestCA("Test Root CA")
			intermediateCert, intermediateKey := generateTestCertificate("Test Intermediate CA", rootCert, rootKey, true)
			leafCert, leafKey := generateTestCertificate("test.example.com", intermediateCert, intermediateKey, false)

			// Create certificate pool
			rootPool := x509.NewCertPool()
			rootPool.AddCert(rootCert)

			intermediatePool := x509.NewCertPool()
			intermediatePool.AddCert(intermediateCert)

			// Verify certificate chain
			opts := x509.VerifyOptions{
				Roots:         rootPool,
				Intermediates: intermediatePool,
				DNSName:       "test.example.com",
				CurrentTime:   time.Now(),
			}

			chains, err := leafCert.Verify(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(chains)).To(BeNumerically(">", 0))
			Expect(len(chains[0])).To(Equal(3)) // Leaf + Intermediate + Root

			// Encode certificates for TLS
			leafPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: leafCert.Raw,
			})
			intermediatePEM := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: intermediateCert.Raw,
			})
			keyPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(leafKey),
			})

			// Create TLS certificate
			tlsCert, err := tls.X509KeyPair(
				append(leafPEM, intermediatePEM...),
				keyPEM,
			)
			Expect(err).NotTo(HaveOccurred())

			// Create server with certificate chain
			serverConfig := &tls.Config{
				Certificates: []tls.Certificate{tlsCert},
				MinVersion:   tls.VersionTLS12,
			}

			listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			go func() {
				conn, _ := listener.Accept()
				if conn != nil {
					conn.Close()
				}
			}()

			// Client validates certificate chain
			clientConfig := &tls.Config{
				RootCAs:    rootPool,
				ServerName: "test.example.com",
				MinVersion: tls.VersionTLS12,
			}

			conn, err := tls.Dial("tcp", listener.Addr().String(), clientConfig)
			Expect(err).NotTo(HaveOccurred())
			if conn != nil {
				state := conn.ConnectionState()
				Expect(len(state.PeerCertificates)).To(Equal(2)) // Leaf + Intermediate
				conn.Close()
			}
		})

		It("should reject expired certificates", func() {
			// Generate expired certificate
			template := &x509.Certificate{
				SerialNumber: big.NewInt(1),
				Subject: pkix.Name{
					CommonName: "expired.example.com",
				},
				NotBefore: time.Now().Add(-48 * time.Hour),
				NotAfter:  time.Now().Add(-24 * time.Hour), // Expired yesterday
				KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
				ExtKeyUsage: []x509.ExtKeyUsage{
					x509.ExtKeyUsageServerAuth,
				},
				DNSNames: []string{"expired.example.com"},
			}

			priv, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).NotTo(HaveOccurred())

			certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
			Expect(err).NotTo(HaveOccurred())

			cert, err := x509.ParseCertificate(certDER)
			Expect(err).NotTo(HaveOccurred())

			// Verify certificate is expired
			opts := x509.VerifyOptions{
				DNSName:     "expired.example.com",
				CurrentTime: time.Now(),
			}

			_, err = cert.Verify(opts)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expired"))
		})

		It("should validate certificate key usage", func() {
			// Generate certificate with specific key usage
			template := &x509.Certificate{
				SerialNumber: big.NewInt(1),
				Subject: pkix.Name{
					CommonName: "keyusage.example.com",
				},
				NotBefore: time.Now(),
				NotAfter:  time.Now().Add(365 * 24 * time.Hour),
				KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
				ExtKeyUsage: []x509.ExtKeyUsage{
					x509.ExtKeyUsageServerAuth,
					x509.ExtKeyUsageClientAuth,
				},
				DNSNames: []string{"keyusage.example.com"},
			}

			priv, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).NotTo(HaveOccurred())

			certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
			Expect(err).NotTo(HaveOccurred())

			cert, err := x509.ParseCertificate(certDER)
			Expect(err).NotTo(HaveOccurred())

			// Validate key usage
			Expect(cert.KeyUsage & x509.KeyUsageDigitalSignature).To(Equal(x509.KeyUsageDigitalSignature))
			Expect(cert.KeyUsage & x509.KeyUsageKeyEncipherment).To(Equal(x509.KeyUsageKeyEncipherment))

			// Validate extended key usage
			hasServerAuth := false
			hasClientAuth := false
			for _, eku := range cert.ExtKeyUsage {
				if eku == x509.ExtKeyUsageServerAuth {
					hasServerAuth = true
				}
				if eku == x509.ExtKeyUsageClientAuth {
					hasClientAuth = true
				}
			}
			Expect(hasServerAuth).To(BeTrue())
			Expect(hasClientAuth).To(BeTrue())
		})
	})

	Context("mTLS Authentication", func() {
		It("should enforce mutual TLS authentication", func() {
			// Generate CA and certificates
			caCert, caKey := generateTestCA("Test mTLS CA")
			serverCert, serverKey := generateTestCertificate("server.example.com", caCert, caKey, false)
			clientCert, clientKey := generateTestCertificate("client.example.com", caCert, caKey, false)

			// Create CA pool
			caPool := x509.NewCertPool()
			caPool.AddCert(caCert)

			// Server certificate
			serverPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: serverCert.Raw,
			})
			serverKeyPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(serverKey),
			})
			serverTLSCert, err := tls.X509KeyPair(serverPEM, serverKeyPEM)
			Expect(err).NotTo(HaveOccurred())

			// Client certificate
			clientPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: clientCert.Raw,
			})
			clientKeyPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(clientKey),
			})
			clientTLSCert, err := tls.X509KeyPair(clientPEM, clientKeyPEM)
			Expect(err).NotTo(HaveOccurred())

			// Server config requiring client cert
			serverConfig := &tls.Config{
				Certificates: []tls.Certificate{serverTLSCert},
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    caPool,
				MinVersion:   tls.VersionTLS12,
			}

			listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			clientVerified := make(chan bool, 1)
			go func() {
				conn, err := listener.Accept()
				if err == nil && conn != nil {
					tlsConn := conn.(*tls.Conn)
					tlsConn.Handshake()
					state := tlsConn.ConnectionState()
					clientVerified <- (len(state.PeerCertificates) > 0)
					conn.Close()
				}
			}()

			// Client config with certificate
			clientConfig := &tls.Config{
				Certificates: []tls.Certificate{clientTLSCert},
				RootCAs:      caPool,
				ServerName:   "server.example.com",
				MinVersion:   tls.VersionTLS12,
			}

			conn, err := tls.Dial("tcp", listener.Addr().String(), clientConfig)
			Expect(err).NotTo(HaveOccurred())
			if conn != nil {
				conn.Close()
			}

			// Verify client was authenticated
			select {
			case verified := <-clientVerified:
				Expect(verified).To(BeTrue())
			case <-time.After(5 * time.Second):
				Fail("Timeout waiting for client verification")
			}
		})

		It("should reject connections without client certificate when mTLS is required", func() {
			// Generate CA and server certificate
			caCert, caKey := generateTestCA("Test mTLS CA")
			serverCert, serverKey := generateTestCertificate("server.example.com", caCert, caKey, false)

			// Create CA pool
			caPool := x509.NewCertPool()
			caPool.AddCert(caCert)

			// Server certificate
			serverPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: serverCert.Raw,
			})
			serverKeyPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(serverKey),
			})
			serverTLSCert, err := tls.X509KeyPair(serverPEM, serverKeyPEM)
			Expect(err).NotTo(HaveOccurred())

			// Server config requiring client cert
			serverConfig := &tls.Config{
				Certificates: []tls.Certificate{serverTLSCert},
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    caPool,
				MinVersion:   tls.VersionTLS12,
			}

			listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			go func() {
				conn, _ := listener.Accept()
				if conn != nil {
					conn.Close()
				}
			}()

			// Client config WITHOUT certificate
			clientConfig := &tls.Config{
				RootCAs:    caPool,
				ServerName: "server.example.com",
				MinVersion: tls.VersionTLS12,
			}

			conn, err := tls.Dial("tcp", listener.Addr().String(), clientConfig)
			if conn != nil {
				conn.Close()
			}

			// Should fail due to missing client certificate
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("certificate required"))
		})
	})

	Context("O-RAN WG11 Compliance", func() {
		It("should validate O-RAN A1 interface TLS requirements", func() {
			// Create O-RAN compliant configuration for A1 interface
			oranConfig, err := security.NewORANCompliantTLS("A1", "enhanced")
			Expect(err).NotTo(HaveOccurred())

			// Validate compliance
			err = oranConfig.ValidateCompliance()
			Expect(err).NotTo(HaveOccurred())

			// Build TLS config
			tlsConfig, err := oranConfig.BuildTLSConfig()
			Expect(err).NotTo(HaveOccurred())

			// Verify minimum TLS version is 1.3 for enhanced profile
			Expect(tlsConfig.MinVersion).To(Equal(uint16(tls.VersionTLS13)))

			// Verify client authentication is required for A1
			Expect(tlsConfig.ClientAuth).To(Equal(tls.RequireAndVerifyClientCert))
		})

		It("should enforce O-RAN cipher suite requirements", func() {
			// Create strict O-RAN configuration
			oranConfig, err := security.NewORANCompliantTLS("E2", "strict")
			Expect(err).NotTo(HaveOccurred())

			tlsConfig, err := oranConfig.BuildTLSConfig()
			Expect(err).NotTo(HaveOccurred())

			// Verify only strongest cipher suite is allowed
			Expect(len(tlsConfig.CipherSuites)).To(Equal(1))
			Expect(tlsConfig.CipherSuites[0]).To(Equal(uint16(tls.TLS_AES_256_GCM_SHA384)))

			// Verify curve preferences
			Expect(len(tlsConfig.CurvePreferences)).To(Equal(1))
			Expect(tlsConfig.CurvePreferences[0]).To(Equal(tls.CurveP384))
		})

		It("should validate O-RAN key strength requirements", func() {
			// Test RSA key strength validation
			weakKey, err := rsa.GenerateKey(rand.Reader, 1024) // Too weak
			Expect(err).NotTo(HaveOccurred())

			strongKey, err := rsa.GenerateKey(rand.Reader, 4096) // Strong enough
			Expect(err).NotTo(HaveOccurred())

			// Create certificates with different key strengths
			weakTemplate := &x509.Certificate{
				SerialNumber: big.NewInt(1),
				Subject:      pkix.Name{CommonName: "weak.example.com"},
				NotBefore:    time.Now(),
				NotAfter:     time.Now().Add(365 * 24 * time.Hour),
			}

			weakCertDER, err := x509.CreateCertificate(rand.Reader, weakTemplate, weakTemplate, &weakKey.PublicKey, weakKey)
			Expect(err).NotTo(HaveOccurred())

			strongTemplate := &x509.Certificate{
				SerialNumber: big.NewInt(2),
				Subject:      pkix.Name{CommonName: "strong.example.com"},
				NotBefore:    time.Now(),
				NotAfter:     time.Now().Add(365 * 24 * time.Hour),
			}

			strongCertDER, err := x509.CreateCertificate(rand.Reader, strongTemplate, strongTemplate, &strongKey.PublicKey, strongKey)
			Expect(err).NotTo(HaveOccurred())

			// Create O-RAN strict compliance checker
			oranConfig, err := security.NewORANCompliantTLS("A1", "strict")
			Expect(err).NotTo(HaveOccurred())

			// Weak certificate should fail validation
			err = oranConfig.CertificateVerifier([][]byte{weakCertDER}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key size"))

			// Strong certificate should pass validation
			err = oranConfig.CertificateVerifier([][]byte{strongCertDER}, nil)
			// Will fail for other reasons (OCSP, etc.) but not key strength
			if err != nil {
				Expect(err.Error()).NotTo(ContainSubstring("key size"))
			}
		})
	})

	Context("Performance and DoS Protection", func() {
		It("should enforce rate limiting for TLS handshakes", func() {
			// Create O-RAN config with rate limiting
			oranConfig, err := security.NewORANCompliantTLS("O1", "baseline")
			Expect(err).NotTo(HaveOccurred())

			// Simulate rapid handshake attempts
			rapidAttempts := 20
			blocked := 0

			for i := 0; i < rapidAttempts; i++ {
				hello := &tls.ClientHelloInfo{
					Conn: &mockConn{remoteAddr: &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 12345}},
				}

				err := oranConfig.PreHandshakeHook(hello)
				if err != nil && err.Error() == "rate limit exceeded" {
					blocked++
				}

				// Small delay to test rate limiting
				time.Sleep(10 * time.Millisecond)
			}

			// Should have blocked some attempts due to rate limiting
			Expect(blocked).To(BeNumerically(">", 0))
		})

		It("should handle session resumption securely", func() {
			// Create server with session tickets enabled
			serverConfig := &tls.Config{
				MinVersion:             tls.VersionTLS13,
				SessionTicketsDisabled: false,
			}

			listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			sessionCache := tls.NewLRUClientSessionCache(10)

			go func() {
				for i := 0; i < 2; i++ {
					conn, _ := listener.Accept()
					if conn != nil {
						conn.Close()
					}
				}
			}()

			// First connection - establish session
			clientConfig := &tls.Config{
				MinVersion:         tls.VersionTLS13,
				ClientSessionCache: sessionCache,
				InsecureSkipVerify: true,
			}

			conn1, err := tls.Dial("tcp", listener.Addr().String(), clientConfig)
			Expect(err).NotTo(HaveOccurred())
			state1 := conn1.ConnectionState()
			conn1.Close()

			// Second connection - should resume session
			conn2, err := tls.Dial("tcp", listener.Addr().String(), clientConfig)
			Expect(err).NotTo(HaveOccurred())
			state2 := conn2.ConnectionState()
			conn2.Close()

			// Verify session was resumed (DidResume field)
			Expect(state1.DidResume).To(BeFalse())
			Expect(state2.DidResume).To(BeTrue())
		})
	})
})

// Helper functions for test certificate generation

func generateTestCA(commonName string) (*x509.Certificate, *rsa.PrivateKey) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
	}

	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	certDER, _ := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	cert, _ := x509.ParseCertificate(certDER)

	return cert, priv
}

func generateTestCertificate(commonName string, parent *x509.Certificate, parentKey *rsa.PrivateKey, isCA bool) (*x509.Certificate, *rsa.PrivateKey) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}

	if !isCA {
		template.DNSNames = []string{commonName, "localhost"}
		template.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1)}
	}

	if isCA {
		template.KeyUsage |= x509.KeyUsageCertSign
		template.MaxPathLen = 1
	}

	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	certDER, _ := x509.CreateCertificate(rand.Reader, template, parent, &priv.PublicKey, parentKey)
	cert, _ := x509.ParseCertificate(certDER)

	return cert, priv
}

// Mock connection for testing
type mockConn struct {
	net.Conn
	remoteAddr net.Addr
}

func (m *mockConn) RemoteAddr() net.Addr {
	return m.remoteAddr
}

// Test certificate data (would be generated in real tests)
const (
	testTLSCertPEM = `-----BEGIN CERTIFICATE-----
[Test certificate data]
-----END CERTIFICATE-----`

	testTLSKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
[Test key data]
-----END RSA PRIVATE KEY-----`

	testCACertPEM = `-----BEGIN CERTIFICATE-----
[Test CA certificate data]
-----END CERTIFICATE-----`

	testClientCertPEM = `-----BEGIN CERTIFICATE-----
[Test client certificate data]
-----END CERTIFICATE-----`

	testClientKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
[Test client key data]
-----END RSA PRIVATE KEY-----`
)
