package integration

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
	"github.com/thc1006/nephoran-intent-operator/pkg/security"
)

var _ = Describe("RAG Security Integration", func() {
	var (
		ragService      *rag.RAGService
		securityScanner *security.SecurityScanner
		incidentResp    *security.IncidentResponse
		ctx             context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Setup RAG service with mock clients
		mockWeaviate := &MockWeaviateClient{
			searchResponse: &rag.SearchResponse{
				Results: []*rag.SearchResult{
					{
						Document: &rag.TelecomDocument{
							ID:       "security-doc-1",
							Content:  "Security best practices for 5G networks include proper authentication, encryption, and network slicing isolation.",
							Source:   "3GPP TS 33.501",
							Title:    "5G Security Architecture",
							Category: "Security",
						},
						Score: 0.95,
					},
				},
				Took: 50 * time.Millisecond,
			},
		}

		mockLLM := &MockLLMClient{
			processResponse: "Based on the security documentation, here are the recommended security practices for 5G networks...",
		}

		ragConfig := &rag.RAGConfig{
			DefaultSearchLimit: 10,
			MaxSearchLimit:     50,
			MinConfidenceScore: 0.5,
			EnableCaching:      true,
			EnableReranking:    true,
		}

		ragService = rag.NewRAGService(mockWeaviate, mockLLM, ragConfig)

		// Setup security scanner
		scannerConfig := &security.ScannerConfig{
			BaseURL:            "https://test-target.local",
			Timeout:            10 * time.Second,
			EnableVulnScanning: true,
			EnableOWASPTesting: true,
		}

		securityScanner = security.NewSecurityScanner(scannerConfig)

		// Setup incident response
		irConfig := &security.IncidentConfig{
			EnableAutoResponse:    true,
			AutoResponseThreshold: "High",
			ForensicsEnabled:      true,
			NotificationConfig: &security.NotificationConfig{
				EnableEmail: true,
				Recipients:  []string{"security@test.com"},
			},
		}

		var err error
		incidentResp, err = security.NewIncidentResponse(irConfig)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if incidentResp != nil {
			incidentResp.Close()
		}
	})

	Describe("Security Knowledge Retrieval", func() {
		Context("when querying for security best practices", func() {
			It("should retrieve relevant security documentation", func() {
				request := &rag.RAGRequest{
					Query:             "What are the security best practices for 5G network deployment?",
					IntentType:        "security",
					MaxResults:        5,
					UseHybridSearch:   true,
					EnableReranking:   true,
					IncludeSourceRefs: true,
				}

				response, err := ragService.ProcessQuery(ctx, request)

				Expect(err).ToNot(HaveOccurred())
				Expect(response).NotTo(BeNil())
				Expect(response.Answer).To(ContainSubstring("security"))
				Expect(response.Confidence).To(BeNumerically(">", 0.5))
				Expect(len(response.SourceDocuments)).To(BeNumerically(">", 0))

				// Check that source is security-related
				securityDoc := response.SourceDocuments[0]
				Expect(securityDoc.Document.Category).To(Equal("Security"))
				Expect(securityDoc.Document.Source).To(ContainSubstring("33.501")) // Security spec
			})

			It("should handle security-specific terminology", func() {
				securityQueries := []string{
					"How to implement network slicing security?",
					"What are the authentication mechanisms in 5G?",
					"How to secure the O-RAN interfaces?",
					"What encryption methods are recommended for 5G Core?",
				}

				for _, query := range securityQueries {
					request := &rag.RAGRequest{
						Query:      query,
						IntentType: "security",
						MaxResults: 3,
					}

					response, err := ragService.ProcessQuery(ctx, request)

					Expect(err).ToNot(HaveOccurred())
					Expect(response).NotTo(BeNil())
					Expect(response.Answer).NotTo(BeEmpty())
				}
			})
		})
	})

	Describe("Security Incident Creation from Scan Results", func() {
		Context("when security scan finds vulnerabilities", func() {
			It("should create security incidents for critical findings", func() {
				// Run security scan
				scanResults, err := securityScanner.RunFullScan(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(scanResults).NotTo(BeNil())

				// Create incidents for high-risk findings
				if scanResults.RiskScore > 70 {
					incidentRequest := &security.CreateIncidentRequest{
						Title:       "High Risk Security Scan Results",
						Description: "Security scan identified multiple high-risk vulnerabilities",
						Severity:    "High",
						Category:    "vulnerability",
						Source:      "security-scanner",
						Tags:        []string{"scan", "vulnerability", "automated"},
						Impact: &security.ImpactAssessment{
							Confidentiality: "Medium",
							Integrity:       "High",
							Availability:    "Medium",
							BusinessImpact:  "High",
							AffectedSystems: []string{"web-services", "api-gateway"},
							EstimatedCost:   25000.0,
						},
					}

					incident, err := incidentResp.CreateIncident(ctx, incidentRequest)

					Expect(err).ToNot(HaveOccurred())
					Expect(incident).NotTo(BeNil())
					Expect(incident.Severity).To(Equal("High"))
					Expect(incident.Category).To(Equal("vulnerability"))
				}
			})

			It("should add scan results as evidence to incidents", func() {
				// Create a test incident
				incidentRequest := &security.CreateIncidentRequest{
					Title:    "Test Security Incident",
					Severity: "Medium",
					Category: "test",
					Source:   "integration-test",
				}

				incident, err := incidentResp.CreateIncident(ctx, incidentRequest)
				Expect(err).ToNot(HaveOccurred())

				// Add scan results as evidence
				evidence := &security.Evidence{
					Type:        "scan_results",
					Source:      "security-scanner",
					Description: "Comprehensive security scan results",
					Data: map[string]interface{}{
						"scan_id":             securityScanner.GenerateReport(),
						"vulnerability_count": 5,
						"risk_score":          75.5,
						"scan_duration":       "5m30s",
					},
				}

				err = incidentResp.AddEvidence(incident.ID, evidence)

				Expect(err).ToNot(HaveOccurred())

				// Verify evidence was added
				updatedIncident, err := incidentResp.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(updatedIncident.Evidence)).To(Equal(1))
				Expect(updatedIncident.Evidence[0].Type).To(Equal("scan_results"))
			})
		})
	})

	Describe("AI-Assisted Incident Response", func() {
		Context("when security incident requires remediation guidance", func() {
			It("should provide AI-powered remediation recommendations", func() {
				// Create security incident
				incidentRequest := &security.CreateIncidentRequest{
					Title:       "SQL Injection Vulnerability Detected",
					Description: "Application vulnerable to SQL injection attacks on login endpoint",
					Severity:    "Critical",
					Category:    "injection",
					Source:      "web-scanner",
					Tags:        []string{"sql-injection", "web-app", "critical"},
				}

				incident, err := incidentResp.CreateIncident(ctx, incidentRequest)
				Expect(err).ToNot(HaveOccurred())

				// Query RAG for remediation guidance
				ragRequest := &rag.RAGRequest{
					Query:             "How to remediate SQL injection vulnerabilities in web applications?",
					IntentType:        "troubleshooting",
					Context:           incident.Description,
					MaxResults:        5,
					IncludeSourceRefs: true,
				}

				ragResponse, err := ragService.ProcessQuery(ctx, ragRequest)

				Expect(err).ToNot(HaveOccurred())
				Expect(ragResponse).NotTo(BeNil())
				Expect(ragResponse.Answer).To(ContainSubstring("SQL injection"))
				Expect(ragResponse.Confidence).To(BeNumerically(">", 0.6))

				// Add RAG response as remediation guidance
				evidence := &security.Evidence{
					Type:        "remediation_guidance",
					Source:      "ai-assistant",
					Description: "AI-generated remediation recommendations",
					Data: map[string]interface{}{
						"query":      ragRequest.Query,
						"response":   ragResponse.Answer,
						"confidence": ragResponse.Confidence,
						"sources":    len(ragResponse.SourceDocuments),
					},
				}

				err = incidentResp.AddEvidence(incident.ID, evidence)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should correlate similar incidents using AI", func() {
				// Create multiple related incidents
				baseRequest := &security.CreateIncidentRequest{
					Severity: "High",
					Category: "injection",
					Source:   "scanner",
				}

				incidents := make([]*security.SecurityIncident, 3)
				incidentTitles := []string{
					"SQL Injection on Login Page",
					"SQL Injection on Search Function",
					"SQL Injection on Contact Form",
				}

				for i, title := range incidentTitles {
					request := *baseRequest
					request.Title = title
					request.Description = "SQL injection vulnerability detected in " + title

					var err error
					incidents[i], err = incidentResp.CreateIncident(ctx, &request)
					Expect(err).ToNot(HaveOccurred())
				}

				// Query for pattern analysis
				ragRequest := &rag.RAGRequest{
					Query:      "What are common patterns in SQL injection attacks and how to prevent them systematically?",
					IntentType: "analysis",
					MaxResults: 3,
				}

				ragResponse, err := ragService.ProcessQuery(ctx, ragRequest)

				Expect(err).ToNot(HaveOccurred())
				Expect(ragResponse).NotTo(BeNil())
				Expect(ragResponse.Answer).NotTo(BeEmpty())

				// All incidents should be SQL injection related
				for _, incident := range incidents {
					Expect(incident.Category).To(Equal("injection"))
					Expect(incident.Title).To(ContainSubstring("SQL Injection"))
				}
			})
		})
	})

	Describe("Automated Security Response Workflow", func() {
		Context("when critical vulnerability is detected", func() {
			It("should trigger end-to-end automated response", func() {
				// Simulate critical security finding
				incidentRequest := &security.CreateIncidentRequest{
					Title:       "Critical RCE Vulnerability in Production",
					Description: "Remote code execution vulnerability found in production web server",
					Severity:    "Critical",
					Category:    "rce",
					Source:      "vulnerability-scanner",
					Tags:        []string{"rce", "production", "web-server", "critical"},
					Impact: &security.ImpactAssessment{
						Confidentiality: "High",
						Integrity:       "High",
						Availability:    "High",
						BusinessImpact:  "Critical",
						AffectedSystems: []string{"web-server-01", "database", "api-gateway"},
						AffectedUsers:   10000,
						EstimatedCost:   500000.0,
					},
				}

				// Create incident (should trigger automated response)
				incident, err := incidentResp.CreateIncident(ctx, incidentRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(incident.Severity).To(Equal("Critical"))

				// Allow time for automated response
				time.Sleep(500 * time.Millisecond)

				// Query AI for emergency response procedures
				ragRequest := &rag.RAGRequest{
					Query:             "What are the immediate steps for responding to a critical RCE vulnerability in production?",
					IntentType:        "troubleshooting",
					Context:           incident.Description,
					MaxResults:        3,
					IncludeSourceRefs: true,
				}

				ragResponse, err := ragService.ProcessQuery(ctx, ragRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(ragResponse.Confidence).To(BeNumerically(">", 0.7))

				// Verify incident was processed
				updatedIncident, err := incidentResp.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(updatedIncident.Timeline)).To(BeNumerically(">", 1))

				// Check metrics were updated
				metrics := incidentResp.GetMetrics()
				Expect(metrics.TotalIncidents).To(BeNumerically(">", 0))
			})
		})
	})

	Describe("Security Knowledge Base Updates", func() {
		Context("when new security incidents are resolved", func() {
			It("should contribute to knowledge base for future incidents", func() {
				// Create and resolve an incident
				incidentRequest := &security.CreateIncidentRequest{
					Title:    "XSS Vulnerability Remediated",
					Severity: "Medium",
					Category: "xss",
					Source:   "code-review",
				}

				incident, err := incidentResp.CreateIncident(ctx, incidentRequest)
				Expect(err).ToNot(HaveOccurred())

				// Add resolution details
				update := &security.IncidentUpdate{
					Status:    "Resolved",
					UpdatedBy: "security-analyst",
				}

				err = incidentResp.UpdateIncident(ctx, incident.ID, update)
				Expect(err).ToNot(HaveOccurred())

				// Query for similar remediation guidance
				ragRequest := &rag.RAGRequest{
					Query:      "How was XSS vulnerability remediated in previous incidents?",
					IntentType: "troubleshooting",
					MaxResults: 5,
				}

				ragResponse, err := ragService.ProcessQuery(ctx, ragRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(ragResponse).NotTo(BeNil())

				// Should provide relevant guidance
				Expect(ragResponse.Answer).NotTo(BeEmpty())
			})
		})
	})

	Describe("Performance and Scalability", func() {
		Context("when handling multiple concurrent operations", func() {
			It("should maintain performance under load", func() {
				// Simulate concurrent security operations
				done := make(chan bool, 10)

				for i := 0; i < 10; i++ {
					go func(index int) {
						defer GinkgoRecover()

						// Create incident
						incidentRequest := &security.CreateIncidentRequest{
							Title:    "Concurrent Test Incident",
							Severity: "Medium",
							Category: "test",
							Source:   "load-test",
						}

						_, err := incidentResp.CreateIncident(ctx, incidentRequest)
						Expect(err).ToNot(HaveOccurred())

						// Query RAG
						ragRequest := &rag.RAGRequest{
							Query:      "Security best practices",
							MaxResults: 3,
						}

						ragResponse, err := ragService.ProcessQuery(ctx, ragRequest)
						Expect(err).ToNot(HaveOccurred())
						Expect(ragResponse).NotTo(BeNil())

						done <- true
					}(i)
				}

				// Wait for all operations to complete
				for i := 0; i < 10; i++ {
					Eventually(done).Should(Receive())
				}

				// Verify system stability
				metrics := incidentResp.GetMetrics()
				Expect(metrics.TotalIncidents).To(BeNumerically(">=", 10))

				ragMetrics := ragService.GetMetrics()
				Expect(ragMetrics.TotalQueries).To(BeNumerically(">=", 10))
			})
		})
	})
})

// Mock implementations for integration tests

type MockWeaviateClient struct {
	searchResponse *rag.SearchResponse
	searchError    error
}

func (m *MockWeaviateClient) Search(ctx context.Context, query *rag.SearchQuery) (*rag.SearchResponse, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResponse, nil
}

func (m *MockWeaviateClient) GetHealthStatus() *rag.HealthStatus {
	return &rag.HealthStatus{
		IsHealthy: true,
		LastCheck: time.Now(),
		Version:   "1.0.0",
	}
}

type MockLLMClient struct {
	processResponse string
	processError    error
}

func (m *MockLLMClient) ProcessIntent(ctx context.Context, prompt string) (string, error) {
	if m.processError != nil {
		return "", m.processError
	}
	return m.processResponse, nil
}
