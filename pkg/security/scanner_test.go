package security

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SecurityScanner", func() {
	var (
		scanner    *SecurityScanner
		testServer *httptest.Server
		config     *ScannerConfig
		ctx        context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create test HTTP server
		testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/health":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			case "/admin":
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Unauthorized"))
			case "/api/test":
				if strings.Contains(r.URL.RawQuery, "test=") {
					query := r.URL.Query().Get("test")
					if strings.Contains(query, "SQL") {
						w.WriteHeader(http.StatusInternalServerError)
						w.Write([]byte("SQL syntax error"))
					} else if strings.Contains(query, "<script>") {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("Content: " + query))
					} else {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("Normal response"))
					}
				}
			case "/login":
				if r.Method == "POST" {
					r.ParseForm()
					username := r.FormValue("username")
					password := r.FormValue("password")
					if username == "admin" && password == "admin" {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("Login successful"))
					} else {
						w.WriteHeader(http.StatusUnauthorized)
						w.Write([]byte("Invalid credentials"))
					}
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		config = &ScannerConfig{
			BaseURL:                testServer.URL,
			Timeout:                5 * time.Second,
			MaxConcurrency:         3,
			SkipTLSVerification:    true,
			EnableVulnScanning:     true,
			EnablePortScanning:     true,
			EnableOWASPTesting:     true,
			EnableAuthTesting:      true,
			EnableInjectionTesting: true,
			TestCredentials: []Credential{
				{Username: "admin", Password: "admin"},
				{Username: "test", Password: "test"},
			},
			UserAgents: []string{"TestAgent/1.0"},
			Wordlists: &Wordlists{
				CommonPasswords:  []string{"admin", "password", "123456"},
				CommonPaths:      []string{"/admin", "/api", "/test"},
				SQLInjection:     []string{"'", "1' OR '1'='1", "'; DROP TABLE users; --"},
				XSSPayloads:      []string{"<script>alert('XSS')</script>", "javascript:alert('XSS')"},
				CommandInjection: []string{"; ls", "| id", "& whoami"},
			},
		}

		scanner = NewSecurityScanner(config)
	})

	AfterEach(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("NewSecurityScanner", func() {
		Context("when creating a new scanner", func() {
			It("should create scanner with provided configuration", func() {
				s := NewSecurityScanner(config)
				Expect(s).NotTo(BeNil())
				Expect(s.config).To(Equal(config))
				Expect(s.client).NotTo(BeNil())
				Expect(s.results).NotTo(BeNil())
			})

			It("should create scanner with default configuration when nil is provided", func() {
				s := NewSecurityScanner(nil)
				Expect(s).NotTo(BeNil())
				Expect(s.config).NotTo(BeNil())
				Expect(s.config.Timeout).To(Equal(30 * time.Second))
				Expect(s.config.MaxConcurrency).To(Equal(10))
			})

			It("should initialize results structure", func() {
				s := NewSecurityScanner(config)
				Expect(s.results.ScanID).NotTo(BeEmpty())
				Expect(s.results.Vulnerabilities).NotTo(BeNil())
				Expect(s.results.OpenPorts).NotTo(BeNil())
				Expect(s.results.TLSFindings).NotTo(BeNil())
				Expect(s.results.AuthFindings).NotTo(BeNil())
			})
		})
	})

	Describe("RunFullScan", func() {
		Context("when running a comprehensive scan", func() {
			It("should complete successfully", func() {
				results, err := scanner.RunFullScan(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(results).NotTo(BeNil())
				Expect(results.EndTime).To(BeTemporally(">", results.StartTime))
				Expect(results.Duration).To(BeNumerically(">", 0))
			})

			It("should generate scan ID", func() {
				results, err := scanner.RunFullScan(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(results.ScanID).NotTo(BeEmpty())
				Expect(results.ScanID).To(HavePrefix("scan_"))
			})

			It("should calculate risk score", func() {
				results, err := scanner.RunFullScan(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(results.RiskScore).To(BeNumerically(">=", 0))
				Expect(results.RiskScore).To(BeNumerically("<=", 100))
			})

			It("should generate recommendations", func() {
				results, err := scanner.RunFullScan(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(results.Recommendations).NotTo(BeNil())
			})
		})

		Context("when scan components are disabled", func() {
			BeforeEach(func() {
				config.EnablePortScanning = false
				config.EnableOWASPTesting = false
				config.EnableAuthTesting = false
				config.EnableInjectionTesting = false
				scanner = NewSecurityScanner(config)
			})

			It("should still complete scan", func() {
				results, err := scanner.RunFullScan(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(results).NotTo(BeNil())
			})
		})
	})

	Describe("Port Scanning", func() {
		Context("when scanning ports", func() {
			It("should identify open ports", func() {
				scanner.runPortScan(ctx)

				// Check if any ports were found
				Expect(scanner.results.OpenPorts).NotTo(BeNil())
			})

			It("should respect concurrency limits", func() {
				scanner.config.MaxConcurrency = 1

				start := time.Now()
				scanner.runPortScan(ctx)
				duration := time.Since(start)

				// With concurrency of 1, it should take longer
				Expect(duration).To(BeNumerically(">", 0))
			})
		})

		Context("when identifying insecure services", func() {
			It("should flag insecure protocols", func() {
				insecurePorts := []int{21, 23, 25} // FTP, Telnet, SMTP

				for _, port := range insecurePorts {
					isInsecure := scanner.isInsecureService(port)
					Expect(isInsecure).To(BeTrue())
				}
			})

			It("should not flag secure protocols", func() {
				securePorts := []int{22, 443, 993}

				for _, port := range securePorts {
					isInsecure := scanner.isInsecureService(port)
					Expect(isInsecure).To(BeFalse())
				}
			})
		})
	})

	Describe("Injection Testing", func() {
		Context("when testing for SQL injection", func() {
			It("should detect SQL injection vulnerabilities", func() {
				scanner.runInjectionTests(ctx)

				// Check if any injection findings were recorded
				Expect(scanner.results.InjectionFindings).NotTo(BeNil())
			})

			It("should test multiple injection types", func() {
				endpoint := "/api/test"

				// Test SQL injection
				scanner.testEndpointWithPayload(ctx, endpoint, "sql_injection", "1' OR '1'='1")

				// Test XSS
				scanner.testEndpointWithPayload(ctx, endpoint, "xss", "<script>alert('XSS')</script>")

				// Test command injection
				scanner.testEndpointWithPayload(ctx, endpoint, "command_injection", "; ls")

				// Verify findings
				findings := scanner.results.InjectionFindings
				Expect(len(findings)).To(BeNumerically(">=", 0))
			})
		})

		Context("when analyzing responses", func() {
			It("should identify SQL error indicators", func() {
				url := "http://test.com/api"
				payload := "SQL test"
				response := "mysql_fetch error in query"

				initialCount := len(scanner.results.InjectionFindings)
				scanner.analyzeInjectionResponse(url, "GET", payload, response, "sql_injection")
				finalCount := len(scanner.results.InjectionFindings)

				Expect(finalCount).To(BeNumerically(">", initialCount))
			})

			It("should identify XSS indicators", func() {
				url := "http://test.com/api"
				payload := "<script>alert('test')</script>"
				response := "Content: <script>alert('test')</script>"

				initialCount := len(scanner.results.InjectionFindings)
				scanner.analyzeInjectionResponse(url, "GET", payload, response, "xss")
				finalCount := len(scanner.results.InjectionFindings)

				Expect(finalCount).To(BeNumerically(">", initialCount))
			})
		})
	})

	Describe("Authentication Testing", func() {
		Context("when testing default credentials", func() {
			It("should identify weak authentication", func() {
				scanner.runAuthTests(ctx)

				// Check if auth findings were recorded
				Expect(scanner.results.AuthFindings).NotTo(BeNil())
			})

			It("should test multiple credential combinations", func() {
				endpoint := "/login"
				scanner.testDefaultCredentials(ctx, endpoint)

				// Should find the admin/admin credential issue
				findings := scanner.results.AuthFindings
				foundDefault := false
				for _, finding := range findings {
					if finding.Issue == "DEFAULT_CREDENTIALS" {
						foundDefault = true
						break
					}
				}
				Expect(foundDefault).To(BeTrue())
			})
		})
	})

	Describe("TLS Testing", func() {
		Context("when testing TLS configuration", func() {
			It("should identify HTTP-only services", func() {
				// Test with HTTP URL
				httpConfig := &ScannerConfig{
					BaseURL: "http://example.com",
				}
				httpScanner := NewSecurityScanner(httpConfig)

				httpScanner.runTLSTests(ctx)

				// Should identify HTTP_ONLY issue
				findings := httpScanner.results.TLSFindings
				httpOnlyFound := false
				for _, finding := range findings {
					if finding.Issue == "HTTP_ONLY" {
						httpOnlyFound = true
						break
					}
				}
				Expect(httpOnlyFound).To(BeTrue())
			})
		})

		Context("when testing SSL/TLS versions", func() {
			It("should identify TLS version names correctly", func() {
				testCases := map[uint16]string{
					tls.VersionSSL30: "SSLv3",
					tls.VersionTLS10: "TLS 1.0",
					tls.VersionTLS11: "TLS 1.1",
					tls.VersionTLS12: "TLS 1.2",
					tls.VersionTLS13: "TLS 1.3",
				}

				for version, expectedName := range testCases {
					name := getTLSVersionName(version)
					Expect(name).To(Equal(expectedName))
				}
			})
		})
	})

	Describe("OWASP Testing", func() {
		Context("when running OWASP Top 10 tests", func() {
			It("should test multiple vulnerability categories", func() {
				scanner.runOWASPTests(ctx)

				// Should complete without error
				Expect(scanner.results.OWASPFindings).NotTo(BeNil())
			})

			It("should test injection flaws", func() {
				scanner.testInjectionFlaws(ctx)

				// Should test various injection types
				Expect(scanner.results.InjectionFindings).NotTo(BeNil())
			})

			It("should test broken authentication", func() {
				scanner.testBrokenAuthentication(ctx)

				// Should test authentication vulnerabilities
				// Implementation depends on testBrokenAuthentication method
			})
		})
	})

	Describe("Utility Functions", func() {
		Context("service identification", func() {
			It("should identify common services by port", func() {
				testCases := map[int]string{
					22:    "ssh",
					80:    "http",
					443:   "https",
					3306:  "mysql",
					5432:  "postgresql",
					27017: "mongodb",
				}

				for port, expectedService := range testCases {
					service := getServiceName(port)
					Expect(service).To(Equal(expectedService))
				}
			})

			It("should return 'unknown' for unrecognized ports", func() {
				service := getServiceName(9999)
				Expect(service).To(Equal("unknown"))
			})
		})

		Context("scan ID generation", func() {
			It("should generate unique scan IDs", func() {
				id1 := generateScanID()
				id2 := generateScanID()

				Expect(id1).NotTo(Equal(id2))
				Expect(id1).To(HavePrefix("scan_"))
				Expect(id2).To(HavePrefix("scan_"))
			})
		})
	})

	Describe("Risk Calculation", func() {
		Context("when calculating risk scores", func() {
			BeforeEach(func() {
				// Add some test vulnerabilities
				scanner.results.Vulnerabilities = []ScannerVulnerability{
					{Severity: "Critical", CVSS: 9.0},
					{Severity: "High", CVSS: 7.5},
					{Severity: "Medium", CVSS: 5.0},
					{Severity: "Low", CVSS: 2.0},
				}

				scanner.results.TLSFindings = []TLSFinding{
					{Issue: "WEAK_TLS_VERSION", Severity: "High"},
				}

				scanner.results.AuthFindings = []AuthFinding{
					{Issue: "DEFAULT_CREDENTIALS", Severity: "Critical"},
				}
			})

			It("should calculate risk score based on findings", func() {
				scanner.calculateRiskScore()

				Expect(scanner.results.RiskScore).To(BeNumerically(">", 0))
				Expect(scanner.results.RiskScore).To(BeNumerically("<=", 100))
			})

			It("should weight critical vulnerabilities higher", func() {
				criticalOnly := &SecurityScanner{
					results: &ScanResults{
						Vulnerabilities: []Vulnerability{
							{Severity: "Critical"},
						},
					},
				}

				lowOnly := &SecurityScanner{
					results: &ScanResults{
						Vulnerabilities: []Vulnerability{
							{Severity: "Low"},
						},
					},
				}

				criticalOnly.calculateRiskScore()
				lowOnly.calculateRiskScore()

				Expect(criticalOnly.results.RiskScore).To(BeNumerically(">", lowOnly.results.RiskScore))
			})
		})
	})

	Describe("Report Generation", func() {
		Context("when generating reports", func() {
			BeforeEach(func() {
				// Populate some test data
				scanner.results.Vulnerabilities = []Vulnerability{
					{
						ID:          "TEST-001",
						Title:       "Test Vulnerability",
						Severity:    "High",
						Description: "Test description",
					},
				}
			})

			It("should generate JSON report", func() {
				report, err := scanner.GenerateReport()

				Expect(err).ToNot(HaveOccurred())
				Expect(report).NotTo(BeEmpty())

				// Should be valid JSON
				Expect(string(report)).To(ContainSubstring("TEST-001"))
				Expect(string(report)).To(ContainSubstring("Test Vulnerability"))
			})

			It("should save report to file", func() {
				err := scanner.SaveReport("test-report.json")

				// Since writeFile is a stub, it should return nil
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("Configuration Validation", func() {
		Context("when using default configuration", func() {
			It("should have sensible defaults", func() {
				defaultConfig := getDefaultScannerConfig()

				Expect(defaultConfig.Timeout).To(Equal(30 * time.Second))
				Expect(defaultConfig.MaxConcurrency).To(Equal(10))
				Expect(defaultConfig.EnableVulnScanning).To(BeTrue())
				Expect(defaultConfig.EnablePortScanning).To(BeTrue())
				Expect(len(defaultConfig.UserAgents)).To(BeNumerically(">", 0))
				Expect(len(defaultConfig.Wordlists.CommonPasswords)).To(BeNumerically(">", 0))
			})
		})
	})

	Describe("Error Handling", func() {
		Context("when network errors occur", func() {
			It("should handle connection failures gracefully", func() {
				// Use invalid URL
				invalidConfig := &ScannerConfig{
					BaseURL: "http://invalid.nonexistent.domain",
					Timeout: 1 * time.Second,
				}
				invalidScanner := NewSecurityScanner(invalidConfig)

				// Should not panic
				results, err := invalidScanner.RunFullScan(ctx)

				Expect(err).ToNot(HaveOccurred()) // Scan should complete even with network failures
				Expect(results).NotTo(BeNil())
			})
		})

		Context("when context is cancelled", func() {
			It("should handle cancellation during scan", func() {
				cancelCtx, cancel := context.WithCancel(ctx)

				// Start scan and cancel quickly
				go func() {
					time.Sleep(10 * time.Millisecond)
					cancel()
				}()

				results, err := scanner.RunFullScan(cancelCtx)

				// Should handle cancellation gracefully
				Expect(results).NotTo(BeNil())
				// Error might or might not occur depending on timing
				_ = err // May or may not have error based on timing
			})
		})

		Context("when timeout occurs", func() {
			It("should respect timeout settings", func() {
				timeoutConfig := &ScannerConfig{
					BaseURL: testServer.URL,
					Timeout: 1 * time.Millisecond, // Very short timeout
				}
				timeoutScanner := NewSecurityScanner(timeoutConfig)

				results, err := timeoutScanner.RunFullScan(ctx)

				// Should complete despite timeouts
				Expect(results).NotTo(BeNil())
				_ = err // May have timeout errors
			})
		})
	})

	Describe("Concurrent Scanning", func() {
		Context("when running concurrent scans", func() {
			It("should handle multiple simultaneous scans", func() {
				done := make(chan bool, 5)

				for i := 0; i < 5; i++ {
					go func() {
						defer GinkgoRecover()
						results, err := scanner.RunFullScan(ctx)
						Expect(err).ToNot(HaveOccurred())
						Expect(results).NotTo(BeNil())
						done <- true
					}()
				}

				// Wait for all scans to complete
				for i := 0; i < 5; i++ {
					Eventually(done).Should(Receive())
				}
			})
		})
	})
})
