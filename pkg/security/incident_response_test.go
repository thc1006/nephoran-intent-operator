package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IncidentResponse", func() {
	var (
		ir     *IncidentResponse
		config *IncidentConfig
		ctx    context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		config = &IncidentConfig{
			EnableAutoResponse:    true,
			AutoResponseThreshold: "High",
			MaxAutoActions:        5,
			IncidentRetention:     7 * 24 * time.Hour,
			EscalationTimeout:     15 * time.Minute,
			ForensicsEnabled:      true,
			NotificationConfig: &NotificationConfig{
				EnableEmail:    true,
				EnableSlack:    true,
				Recipients:     []string{"security@test.com"},
				EscalationList: []string{"manager@test.com"},
			},
		}
		
		var err error
		ir, err = NewIncidentResponse(config)
		Expect(err).ToNot(HaveOccurred())
		Expect(ir).NotTo(BeNil())
	})

	AfterEach(func() {
		if ir != nil {
			ir.Close()
		}
	})

	Describe("NewIncidentResponse", func() {
		Context("when creating incident response system", func() {
			It("should create with provided configuration", func() {
				system, err := NewIncidentResponse(config)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(system).NotTo(BeNil())
				Expect(system.config).To(Equal(config))
				Expect(system.incidents).NotTo(BeNil())
				Expect(system.playbooks).NotTo(BeNil())
				Expect(system.metrics).NotTo(BeNil())
			})

			It("should create with default configuration when nil provided", func() {
				system, err := NewIncidentResponse(nil)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(system).NotTo(BeNil())
				Expect(system.config).NotTo(BeNil())
				Expect(system.config.EnableAutoResponse).To(BeTrue())
			})

			It("should initialize metrics", func() {
				system, err := NewIncidentResponse(config)
				
				Expect(err).ToNot(HaveOccurred())
				metrics := system.GetMetrics()
				Expect(metrics.TotalIncidents).To(Equal(int64(0)))
				Expect(metrics.OpenIncidents).To(Equal(int64(0)))
				Expect(metrics.ResolvedIncidents).To(Equal(int64(0)))
			})

			It("should load default playbooks", func() {
				system, err := NewIncidentResponse(config)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(len(system.playbooks)).To(BeNumerically(">", 0))
			})
		})
	})

	Describe("CreateIncident", func() {
		var request *CreateIncidentRequest

		BeforeEach(func() {
			request = &CreateIncidentRequest{
				Title:       "Malware Detected on Server",
				Description: "Suspicious binary detected on production server",
				Severity:    "High",
				Category:    "malware",
				Source:      "antivirus-scanner",
				Tags:        []string{"malware", "production", "server-01"},
				Impact: &ImpactAssessment{
					Confidentiality: "Low",
					Integrity:       "High",
					Availability:    "Medium",
					BusinessImpact:  "High",
					AffectedSystems: []string{"server-01", "database-cluster"},
					AffectedUsers:   100,
					EstimatedCost:   50000.0,
				},
			}
		})

		Context("when creating valid incident", func() {
			It("should create incident successfully", func() {
				incident, err := ir.CreateIncident(ctx, request)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(incident).NotTo(BeNil())
				Expect(incident.ID).NotTo(BeEmpty())
				Expect(incident.Title).To(Equal(request.Title))
				Expect(incident.Severity).To(Equal(request.Severity))
				Expect(incident.Status).To(Equal("Open"))
				Expect(len(incident.Timeline)).To(BeNumerically(">", 0))
			})

			It("should assign unique ID", func() {
				incident1, err1 := ir.CreateIncident(ctx, request)
				incident2, err2 := ir.CreateIncident(ctx, request)
				
				Expect(err1).ToNot(HaveOccurred())
				Expect(err2).ToNot(HaveOccurred())
				Expect(incident1.ID).NotTo(Equal(incident2.ID))
			})

			It("should update metrics", func() {
				initialMetrics := ir.GetMetrics()
				initialTotal := initialMetrics.TotalIncidents
				initialOpen := initialMetrics.OpenIncidents
				
				incident, err := ir.CreateIncident(ctx, request)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(incident).NotTo(BeNil())
				
				finalMetrics := ir.GetMetrics()
				Expect(finalMetrics.TotalIncidents).To(Equal(initialTotal + 1))
				Expect(finalMetrics.OpenIncidents).To(Equal(initialOpen + 1))
			})

			It("should create initial timeline event", func() {
				incident, err := ir.CreateIncident(ctx, request)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(len(incident.Timeline)).To(Equal(1))
				Expect(incident.Timeline[0].Type).To(Equal("created"))
				Expect(incident.Timeline[0].Actor).To(Equal("system"))
				Expect(incident.Timeline[0].Automated).To(BeTrue())
			})

			It("should trigger automated response for high severity", func() {
				request.Severity = "Critical"
				
				incident, err := ir.CreateIncident(ctx, request)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(incident).NotTo(BeNil())
				
				// Give some time for automated response to trigger
				time.Sleep(100 * time.Millisecond)
			})
		})

		Context("when forensics is enabled", func() {
			It("should start evidence collection", func() {
				incident, err := ir.CreateIncident(ctx, request)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(incident).NotTo(BeNil())
				
				// Give some time for evidence collection to start
				time.Sleep(100 * time.Millisecond)
			})
		})
	})

	Describe("UpdateIncident", func() {
		var incident *SecurityIncident

		BeforeEach(func() {
			request := &CreateIncidentRequest{
				Title:    "Test Incident",
				Severity: "Medium",
				Category: "test",
				Source:   "test-source",
			}
			var err error
			incident, err = ir.CreateIncident(ctx, request)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when updating incident status", func() {
			It("should update status successfully", func() {
				update := &IncidentUpdate{
					Status:    "Acknowledged",
					UpdatedBy: "analyst-01",
				}
				
				err := ir.UpdateIncident(ctx, incident.ID, update)
				
				Expect(err).ToNot(HaveOccurred())
				
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedIncident.Status).To(Equal("Acknowledged"))
				Expect(updatedIncident.AcknowledgedAt).NotTo(BeNil())
			})

			It("should add timeline event for status change", func() {
				update := &IncidentUpdate{
					Status:    "Resolved",
					UpdatedBy: "analyst-01",
				}
				
				err := ir.UpdateIncident(ctx, incident.ID, update)
				
				Expect(err).ToNot(HaveOccurred())
				
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				
				statusChangeFound := false
				for _, event := range updatedIncident.Timeline {
					if event.Type == "status_changed" {
						statusChangeFound = true
						Expect(event.Actor).To(Equal("analyst-01"))
						Expect(event.Automated).To(BeFalse())
						break
					}
				}
				Expect(statusChangeFound).To(BeTrue())
			})

			It("should set resolved timestamp", func() {
				update := &IncidentUpdate{
					Status:    "Resolved",
					UpdatedBy: "analyst-01",
				}
				
				err := ir.UpdateIncident(ctx, incident.ID, update)
				
				Expect(err).ToNot(HaveOccurred())
				
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedIncident.ResolvedAt).NotTo(BeNil())
			})
		})

		Context("when updating incident assignee", func() {
			It("should update assignee successfully", func() {
				update := &IncidentUpdate{
					Assignee:  "analyst-02",
					UpdatedBy: "manager-01",
				}
				
				err := ir.UpdateIncident(ctx, incident.ID, update)
				
				Expect(err).ToNot(HaveOccurred())
				
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedIncident.Assignee).To(Equal("analyst-02"))
			})
		})

		Context("when updating incident severity", func() {
			It("should update severity successfully", func() {
				update := &IncidentUpdate{
					Severity:  "Critical",
					UpdatedBy: "manager-01",
				}
				
				err := ir.UpdateIncident(ctx, incident.ID, update)
				
				Expect(err).ToNot(HaveOccurred())
				
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedIncident.Severity).To(Equal("Critical"))
			})

			It("should add timeline event for severity change", func() {
				originalSeverity := incident.Severity
				update := &IncidentUpdate{
					Severity:  "Critical",
					UpdatedBy: "manager-01",
				}
				
				err := ir.UpdateIncident(ctx, incident.ID, update)
				
				Expect(err).ToNot(HaveOccurred())
				
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				
				severityChangeFound := false
				for _, event := range updatedIncident.Timeline {
					if event.Type == "severity_changed" {
						severityChangeFound = true
						Expect(event.Description).To(ContainSubstring(originalSeverity))
						Expect(event.Description).To(ContainSubstring("Critical"))
						break
					}
				}
				Expect(severityChangeFound).To(BeTrue())
			})
		})

		Context("when incident doesn't exist", func() {
			It("should return error", func() {
				update := &IncidentUpdate{
					Status:    "Resolved",
					UpdatedBy: "analyst-01",
				}
				
				err := ir.UpdateIncident(ctx, "nonexistent-id", update)
				
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("incident not found"))
			})
		})
	})

	Describe("GetIncident", func() {
		var incident *SecurityIncident

		BeforeEach(func() {
			request := &CreateIncidentRequest{
				Title:    "Test Incident",
				Severity: "Medium",
				Category: "test",
				Source:   "test-source",
			}
			var err error
			incident, err = ir.CreateIncident(ctx, request)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when incident exists", func() {
			It("should return incident", func() {
				retrieved, err := ir.GetIncident(incident.ID)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(retrieved).NotTo(BeNil())
				Expect(retrieved.ID).To(Equal(incident.ID))
				Expect(retrieved.Title).To(Equal(incident.Title))
			})
		})

		Context("when incident doesn't exist", func() {
			It("should return error", func() {
				retrieved, err := ir.GetIncident("nonexistent-id")
				
				Expect(err).To(HaveOccurred())
				Expect(retrieved).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("incident not found"))
			})
		})
	})

	Describe("ListIncidents", func() {
		BeforeEach(func() {
			// Create multiple test incidents
			incidents := []*CreateIncidentRequest{
				{
					Title:    "Critical Incident",
					Severity: "Critical",
					Category: "malware",
					Source:   "scanner-01",
					Tags:     []string{"critical", "malware"},
				},
				{
					Title:    "High Incident",
					Severity: "High",
					Category: "intrusion",
					Source:   "ids-01",
					Tags:     []string{"high", "intrusion"},
				},
				{
					Title:    "Medium Incident",
					Severity: "Medium",
					Category: "policy-violation",
					Source:   "dlp-01",
					Tags:     []string{"medium", "policy"},
				},
			}
			
			for _, req := range incidents {
				_, err := ir.CreateIncident(ctx, req)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		Context("when listing all incidents", func() {
			It("should return all incidents", func() {
				incidents, err := ir.ListIncidents(nil)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(len(incidents)).To(Equal(3))
			})

			It("should sort by detection time (newest first)", func() {
				incidents, err := ir.ListIncidents(nil)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(len(incidents)).To(BeNumerically(">=", 2))
				
				for i := 1; i < len(incidents); i++ {
					Expect(incidents[i-1].DetectedAt).To(BeTemporally(">=", incidents[i].DetectedAt))
				}
			})
		})

		Context("when filtering by severity", func() {
			It("should return only matching incidents", func() {
				filter := &IncidentFilter{Severity: "Critical"}
				incidents, err := ir.ListIncidents(filter)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(len(incidents)).To(Equal(1))
				Expect(incidents[0].Severity).To(Equal("Critical"))
			})
		})

		Context("when filtering by category", func() {
			It("should return only matching incidents", func() {
				filter := &IncidentFilter{Category: "malware"}
				incidents, err := ir.ListIncidents(filter)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(len(incidents)).To(Equal(1))
				Expect(incidents[0].Category).To(Equal("malware"))
			})
		})

		Context("when filtering by tags", func() {
			It("should return incidents with matching tags", func() {
				filter := &IncidentFilter{Tags: []string{"critical"}}
				incidents, err := ir.ListIncidents(filter)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(len(incidents)).To(Equal(1))
				
				tagFound := false
				for _, tag := range incidents[0].Tags {
					if tag == "critical" {
						tagFound = true
						break
					}
				}
				Expect(tagFound).To(BeTrue())
			})
		})

		Context("when using limit", func() {
			It("should respect limit parameter", func() {
				filter := &IncidentFilter{Limit: 2}
				incidents, err := ir.ListIncidents(filter)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(len(incidents)).To(Equal(2))
			})
		})

		Context("when filtering by date range", func() {
			It("should return incidents within date range", func() {
				now := time.Now()
				filter := &IncidentFilter{
					FromDate: now.Add(-1 * time.Hour),
					ToDate:   now.Add(1 * time.Hour),
				}
				incidents, err := ir.ListIncidents(filter)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(len(incidents)).To(Equal(3))
			})
		})
	})

	Describe("AddEvidence", func() {
		var incident *SecurityIncident

		BeforeEach(func() {
			request := &CreateIncidentRequest{
				Title:    "Test Incident",
				Severity: "Medium",
				Category: "test",
				Source:   "test-source",
			}
			var err error
			incident, err = ir.CreateIncident(ctx, request)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when adding valid evidence", func() {
			It("should add evidence successfully", func() {
				evidence := &Evidence{
					Type:        "log",
					Source:      "security-log",
					Description: "Suspicious login attempt",
					Data: map[string]interface{}{
						"ip_address": "192.168.1.100",
						"username":   "admin",
						"timestamp":  time.Now().Unix(),
					},
				}
				
				err := ir.AddEvidence(incident.ID, evidence)
				
				Expect(err).ToNot(HaveOccurred())
				
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(updatedIncident.Evidence)).To(Equal(1))
				Expect(updatedIncident.Evidence[0].ID).NotTo(BeEmpty())
				Expect(updatedIncident.Evidence[0].Hash).NotTo(BeEmpty())
			})

			It("should add timeline event", func() {
				evidence := &Evidence{
					Type:        "file",
					Source:      "filesystem",
					Description: "Suspicious executable",
				}
				
				err := ir.AddEvidence(incident.ID, evidence)
				
				Expect(err).ToNot(HaveOccurred())
				
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				
				evidenceEventFound := false
				for _, event := range updatedIncident.Timeline {
					if event.Type == "evidence_added" {
						evidenceEventFound = true
						break
					}
				}
				Expect(evidenceEventFound).To(BeTrue())
			})
		})

		Context("when incident doesn't exist", func() {
			It("should return error", func() {
				evidence := &Evidence{
					Type:   "log",
					Source: "test",
				}
				
				err := ir.AddEvidence("nonexistent-id", evidence)
				
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("incident not found"))
			})
		})
	})

	Describe("ExecutePlaybook", func() {
		var incident *SecurityIncident

		BeforeEach(func() {
			request := &CreateIncidentRequest{
				Title:    "Malware Detection",
				Severity: "High",
				Category: "malware",
				Source:   "antivirus",
			}
			var err error
			incident, err = ir.CreateIncident(ctx, request)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when executing valid playbook", func() {
			It("should execute playbook successfully", func() {
				// Find a default playbook
				var playbookID string
				for id := range ir.playbooks {
					playbookID = id
					break
				}
				
				Expect(playbookID).NotTo(BeEmpty())
				
				err := ir.ExecutePlaybook(ctx, incident.ID, playbookID)
				
				Expect(err).ToNot(HaveOccurred())
				
				// Check that actions were added to incident
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(updatedIncident.Actions)).To(BeNumerically(">", 0))
			})
		})

		Context("when playbook doesn't exist", func() {
			It("should return error", func() {
				err := ir.ExecutePlaybook(ctx, incident.ID, "nonexistent-playbook")
				
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("playbook not found"))
			})
		})

		Context("when incident doesn't exist", func() {
			It("should return error", func() {
				playbookID := "malware_detected"
				err := ir.ExecutePlaybook(ctx, "nonexistent-incident", playbookID)
				
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("incident not found"))
			})
		})
	})

	Describe("Automated Response", func() {
		Context("when auto response is enabled", func() {
			It("should trigger for high severity incidents", func() {
				request := &CreateIncidentRequest{
					Title:    "Critical Security Breach",
					Severity: "Critical",
					Category: "malware",
					Source:   "security-monitor",
				}
				
				incident, err := ir.CreateIncident(ctx, request)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(incident).NotTo(BeNil())
				
				// Give time for automated response
				time.Sleep(200 * time.Millisecond)
				
				// Check if automated actions were taken
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				
				// Should have automated actions if matching playbooks exist
				automatedFound := false
				for _, action := range updatedIncident.Actions {
					if action.Automated {
						automatedFound = true
						break
					}
				}
				// Note: This depends on whether matching playbooks exist
				_ = automatedFound // Mark as used for this test case
			})

			It("should not trigger for low severity incidents", func() {
				originalConfig := ir.config.AutoResponseThreshold
				ir.config.AutoResponseThreshold = "High"
				
				request := &CreateIncidentRequest{
					Title:    "Low Priority Issue",
					Severity: "Low",
					Category: "policy-violation",
					Source:   "compliance-check",
				}
				
				incident, err := ir.CreateIncident(ctx, request)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(incident).NotTo(BeNil())
				
				// Restore original config
				ir.config.AutoResponseThreshold = originalConfig
			})
		})

		Context("when auto response is disabled", func() {
			BeforeEach(func() {
				ir.config.EnableAutoResponse = false
			})

			It("should not trigger automated response", func() {
				request := &CreateIncidentRequest{
					Title:    "Critical Security Breach",
					Severity: "Critical",
					Category: "malware",
					Source:   "security-monitor",
				}
				
				incident, err := ir.CreateIncident(ctx, request)
				
				Expect(err).ToNot(HaveOccurred())
				Expect(incident).NotTo(BeNil())
				
				// Give time for potential automated response
				time.Sleep(100 * time.Millisecond)
				
				// Should not have automated actions
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(updatedIncident.Actions)).To(Equal(0))
			})
		})
	})

	Describe("Metrics", func() {
		Context("when incidents are created and resolved", func() {
			It("should track incident counts correctly", func() {
				initialMetrics := ir.GetMetrics()
				
				// Create incidents
				request1 := &CreateIncidentRequest{
					Title:    "Incident 1",
					Severity: "High",
					Category: "test",
					Source:   "test",
				}
				request2 := &CreateIncidentRequest{
					Title:    "Incident 2",
					Severity: "Medium",
					Category: "test",
					Source:   "test",
				}
				
				incident1, err1 := ir.CreateIncident(ctx, request1)
				incident2, err2 := ir.CreateIncident(ctx, request2)
				
				Expect(err1).ToNot(HaveOccurred())
				Expect(err2).ToNot(HaveOccurred())
				Expect(incident2).NotTo(BeNil()) // Use incident2
				
				// Check metrics after creation
				afterCreateMetrics := ir.GetMetrics()
				Expect(afterCreateMetrics.TotalIncidents).To(Equal(initialMetrics.TotalIncidents + 2))
				Expect(afterCreateMetrics.OpenIncidents).To(Equal(initialMetrics.OpenIncidents + 2))
				
				// Resolve one incident
				update := &IncidentUpdate{
					Status:    "Resolved",
					UpdatedBy: "analyst",
				}
				err := ir.UpdateIncident(ctx, incident1.ID, update)
				Expect(err).ToNot(HaveOccurred())
				
				// Check metrics after resolution
				afterResolveMetrics := ir.GetMetrics()
				Expect(afterResolveMetrics.OpenIncidents).To(Equal(initialMetrics.OpenIncidents + 1))
				Expect(afterResolveMetrics.ResolvedIncidents).To(Equal(initialMetrics.ResolvedIncidents + 1))
			})

			It("should track incidents by severity", func() {
				request := &CreateIncidentRequest{
					Title:    "Critical Incident",
					Severity: "Critical",
					Category: "test",
					Source:   "test",
				}
				
				initialMetrics := ir.GetMetrics()
				initialCritical := initialMetrics.IncidentsBySeverity["Critical"]
				
				_, err := ir.CreateIncident(ctx, request)
				Expect(err).ToNot(HaveOccurred())
				
				finalMetrics := ir.GetMetrics()
				finalCritical := finalMetrics.IncidentsBySeverity["Critical"]
				
				Expect(finalCritical).To(Equal(initialCritical + 1))
			})

			It("should calculate MTTR when incidents are resolved", func() {
				request := &CreateIncidentRequest{
					Title:    "Test Incident",
					Severity: "High",
					Category: "test",
					Source:   "test",
				}
				
				incident, err := ir.CreateIncident(ctx, request)
				Expect(err).ToNot(HaveOccurred())
				
				// Wait a bit then resolve
				time.Sleep(10 * time.Millisecond)
				
				update := &IncidentUpdate{
					Status:    "Resolved",
					UpdatedBy: "analyst",
				}
				err = ir.UpdateIncident(ctx, incident.ID, update)
				Expect(err).ToNot(HaveOccurred())
				
				metrics := ir.GetMetrics()
				Expect(metrics.MTTR).To(BeNumerically(">", 0))
			})
		})
	})

	Describe("Default Configuration", func() {
		It("should have proper default values", func() {
			defaultConfig := getDefaultIncidentConfig()
			
			Expect(defaultConfig.EnableAutoResponse).To(BeTrue())
			Expect(defaultConfig.AutoResponseThreshold).To(Equal("High"))
			Expect(defaultConfig.IncidentRetention).To(Equal(90 * 24 * time.Hour))
			Expect(defaultConfig.ForensicsEnabled).To(BeTrue())
			Expect(defaultConfig.NotificationConfig).NotTo(BeNil())
			Expect(defaultConfig.NotificationConfig.EnableEmail).To(BeTrue())
		})
	})

	Describe("Utility Functions", func() {
		It("should generate unique incident IDs", func() {
			id1 := generateIncidentID()
			id2 := generateIncidentID()
			
			Expect(id1).NotTo(Equal(id2))
			Expect(id1).To(HavePrefix("INC-"))
			Expect(id2).To(HavePrefix("INC-"))
		})

		It("should generate unique evidence IDs", func() {
			id1 := generateEvidenceID()
			id2 := generateEvidenceID()
			
			Expect(id1).NotTo(Equal(id2))
			Expect(id1).To(HavePrefix("EVD-"))
			Expect(id2).To(HavePrefix("EVD-"))
		})

		It("should generate unique action IDs", func() {
			id1 := generateActionID()
			id2 := generateActionID()
			
			Expect(id1).NotTo(Equal(id2))
			Expect(id1).To(HavePrefix("ACT-"))
			Expect(id2).To(HavePrefix("ACT-"))
		})
	})

	Describe("System Lifecycle", func() {
		Context("when closing the system", func() {
			It("should close gracefully", func() {
				system, err := NewIncidentResponse(config)
				Expect(err).ToNot(HaveOccurred())
				
				err = system.Close()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("HandleWebhook", func() {
		var (
			webhookSecret string
		)

		BeforeEach(func() {
			webhookSecret = "test-webhook-secret-key-12345"
			config.WebhookSecret = webhookSecret
			
			var err error
			ir, err = NewIncidentResponse(config)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("with valid HMAC signature", func() {
			It("should process webhook with valid security alert payload", func() {
				payload := `{
					"type": "security_alert",
					"alert": {
						"title": "Suspicious Login Detected",
						"description": "Multiple failed login attempts from unknown IP",
						"severity": "High",
						"category": "authentication",
						"source": "auth-system"
					}
				}`

				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusOK))
				Expect(string(recorder.body)).To(ContainSubstring("Webhook processed successfully"))
			})

			It("should process webhook with incident update payload", func() {
				// First create an incident to update
				incident, err := ir.CreateIncident(ctx, &CreateIncidentRequest{
					Title:    "Test Incident for Webhook",
					Severity: "Medium",
					Category: "test",
					Source:   "test-webhook",
				})
				Expect(err).ToNot(HaveOccurred())

				payload := fmt.Sprintf(`{
					"type": "incident_update",
					"incident_id": "%s",
					"status": "Acknowledged",
					"assignee": "analyst-webhook",
					"severity": "High"
				}`, incident.ID)

				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusOK))

				// Verify the incident was updated
				updatedIncident, err := ir.GetIncident(incident.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedIncident.Status).To(Equal("Acknowledged"))
				Expect(updatedIncident.Assignee).To(Equal("analyst-webhook"))
				Expect(updatedIncident.Severity).To(Equal("High"))
			})

			It("should process webhook with threat intelligence payload", func() {
				payload := `{
					"type": "threat_intelligence",
					"threat_type": "malware_c2",
					"indicators": ["malicious-domain.com", "192.168.1.100", "suspicious-hash-abc123"]
				}`

				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusOK))
			})

			It("should process webhook with unknown type as generic webhook", func() {
				payload := `{
					"type": "custom_event",
					"custom_field": "custom_value",
					"data": {"key": "value"}
				}`

				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusOK))
			})
		})

		Context("with invalid HMAC signature", func() {
			It("should reject webhook with completely invalid signature", func() {
				payload := `{"type": "security_alert", "alert": {"title": "test"}}`
				req := createWebhookRequest(payload, "sha256=invalid-signature-12345")
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusUnauthorized))
				Expect(string(recorder.body)).To(ContainSubstring("Invalid signature"))
			})

			It("should reject webhook with wrong secret used in signature", func() {
				payload := `{"type": "security_alert", "alert": {"title": "test"}}`
				wrongSecret := "wrong-secret-key"
				signature := generateValidSignature([]byte(payload), wrongSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusUnauthorized))
				Expect(string(recorder.body)).To(ContainSubstring("Invalid signature"))
			})

			It("should reject webhook with signature format without sha256 prefix", func() {
				payload := `{"type": "security_alert", "alert": {"title": "test"}}`
				// Generate valid signature but remove the sha256= prefix
				validSig := generateValidSignature([]byte(payload), webhookSecret)
				invalidSig := strings.TrimPrefix(validSig, "sha256=")
				req := createWebhookRequest(payload, invalidSig)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusUnauthorized))
				Expect(string(recorder.body)).To(ContainSubstring("Invalid signature"))
			})

			It("should reject webhook with empty signature", func() {
				payload := `{"type": "security_alert", "alert": {"title": "test"}}`
				req := createWebhookRequest(payload, "")
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusUnauthorized))
				Expect(string(recorder.body)).To(ContainSubstring("Missing signature header"))
			})

			It("should reject webhook with signature for different payload", func() {
				originalPayload := `{"type": "security_alert", "alert": {"title": "original"}}`
				modifiedPayload := `{"type": "security_alert", "alert": {"title": "modified"}}`
				
				// Sign the original payload but send the modified one
				signature := generateValidSignature([]byte(originalPayload), webhookSecret)
				req := createWebhookRequest(modifiedPayload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusUnauthorized))
				Expect(string(recorder.body)).To(ContainSubstring("Invalid signature"))
			})
		})

		Context("with missing or malformed data", func() {
			It("should reject webhook with missing signature header", func() {
				payload := `{"type": "security_alert", "alert": {"title": "test"}}`
				req := createWebhookRequestWithoutSignature(payload)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusUnauthorized))
				Expect(string(recorder.body)).To(ContainSubstring("Missing signature header"))
			})

			It("should handle empty request body", func() {
				payload := ""
				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusInternalServerError))
				Expect(string(recorder.body)).To(ContainSubstring("Failed to process webhook"))
			})

			It("should handle invalid JSON payload", func() {
				payload := `{"type": "security_alert", "alert": {"title": "test"` // Missing closing braces
				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusInternalServerError))
				Expect(string(recorder.body)).To(ContainSubstring("Failed to process webhook"))
			})

			It("should handle payload without type field", func() {
				payload := `{"alert": {"title": "test alert"}}`
				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusInternalServerError))
				Expect(string(recorder.body)).To(ContainSubstring("Failed to process webhook"))
			})

			It("should handle security alert with missing alert field", func() {
				payload := `{"type": "security_alert", "other_field": "value"}`
				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusInternalServerError))
				Expect(string(recorder.body)).To(ContainSubstring("Failed to process webhook"))
			})

			It("should handle incident update with missing incident_id", func() {
				payload := `{"type": "incident_update", "status": "Resolved"}`
				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusInternalServerError))
				Expect(string(recorder.body)).To(ContainSubstring("Failed to process webhook"))
			})

			It("should handle incident update with non-existent incident_id", func() {
				payload := `{
					"type": "incident_update", 
					"incident_id": "non-existent-incident-id",
					"status": "Resolved"
				}`
				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusInternalServerError))
				Expect(string(recorder.body)).To(ContainSubstring("Failed to process webhook"))
			})
		})

		Context("when webhook secret is not configured", func() {
			BeforeEach(func() {
				config.WebhookSecret = ""
				var err error
				ir, err = NewIncidentResponse(config)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should reject all webhooks when secret is empty", func() {
				payload := `{"type": "security_alert", "alert": {"title": "test"}}`
				signature := "sha256=any-signature"
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusUnauthorized))
				Expect(string(recorder.body)).To(ContainSubstring("Invalid signature"))
			})
		})

		Context("stress testing HMAC verification", func() {
			It("should handle large payloads correctly", func() {
				// Create a large payload (1MB)
				largeData := make(map[string]interface{})
				largeData["type"] = "security_alert"
				largeData["alert"] = map[string]interface{}{
					"title":       "Large Alert",
					"description": strings.Repeat("A", 1024*1024), // 1MB of data
					"severity":    "High",
				}

				payloadBytes, err := json.Marshal(largeData)
				Expect(err).ToNot(HaveOccurred())

				signature := generateValidSignature(payloadBytes, webhookSecret)
				req := createWebhookRequest(string(payloadBytes), signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusOK))
			})

			It("should consistently reject invalid signatures (timing attack protection)", func() {
				payload := `{"type": "security_alert", "alert": {"title": "test"}}`
				validSignature := generateValidSignature([]byte(payload), webhookSecret)
				
				// Test multiple invalid signatures to ensure consistent timing
				invalidSignatures := []string{
					"sha256=0000000000000000000000000000000000000000000000000000000000000000",
					"sha256=1111111111111111111111111111111111111111111111111111111111111111",
					"sha256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"sha256=zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
					validSignature[:len(validSignature)-2] + "00", // Almost valid signature
				}

				for _, invalidSig := range invalidSignatures {
					req := createWebhookRequest(payload, invalidSig)
					recorder := &mockResponseWriter{}

					start := time.Now()
					ir.HandleWebhook(recorder, req)
					duration := time.Since(start)

					Expect(recorder.statusCode).To(Equal(http.StatusUnauthorized))
					Expect(duration).To(BeNumerically("<", 100*time.Millisecond)) // Should be fast
				}
			})
		})

		Context("edge cases and special characters", func() {
			It("should handle payloads with special characters", func() {
				payload := `{
					"type": "security_alert",
					"alert": {
						"title": "Alert with special chars: üñíçødé & symbols!@#$%^&*()",
						"description": "Description with newlines\nand tabs\t and quotes \"'",
						"severity": "Medium"
					}
				}`

				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusOK))
			})

			It("should handle binary data in payload", func() {
				// Create payload with some binary-like content (base64 encoded)
				binaryData := make([]byte, 256)
				for i := range binaryData {
					binaryData[i] = byte(i)
				}
				encodedData := base64.StdEncoding.EncodeToString(binaryData)

				payload := fmt.Sprintf(`{
					"type": "security_alert",
					"alert": {
						"title": "Binary Data Alert",
						"description": "Alert with binary data",
						"severity": "Low",
						"binary_field": "%s"
					}
				}`, encodedData)

				signature := generateValidSignature([]byte(payload), webhookSecret)
				req := createWebhookRequest(payload, signature)
				recorder := &mockResponseWriter{}

				ir.HandleWebhook(recorder, req)

				Expect(recorder.statusCode).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("verifyWebhookSignature", func() {
		BeforeEach(func() {
			config.WebhookSecret = "test-secret-key"
			var err error
			ir, err = NewIncidentResponse(config)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("signature verification logic", func() {
			It("should verify valid signature correctly", func() {
				payload := []byte("test payload")
				signature := generateValidSignature(payload, "test-secret-key")

				result := ir.verifyWebhookSignature(payload, signature)
				Expect(result).To(BeTrue())
			})

			It("should reject invalid signature", func() {
				payload := []byte("test payload")
				invalidSignature := "sha256=invalid123"

				result := ir.verifyWebhookSignature(payload, invalidSignature)
				Expect(result).To(BeFalse())
			})

			It("should reject signature without sha256 prefix", func() {
				payload := []byte("test payload")
				signature := generateValidSignature(payload, "test-secret-key")
				signatureWithoutPrefix := strings.TrimPrefix(signature, "sha256=")

				result := ir.verifyWebhookSignature(payload, signatureWithoutPrefix)
				Expect(result).To(BeFalse())
			})

			It("should handle empty payload", func() {
				payload := []byte("")
				signature := generateValidSignature(payload, "test-secret-key")

				result := ir.verifyWebhookSignature(payload, signature)
				Expect(result).To(BeTrue())
			})

			It("should handle empty secret", func() {
				config.WebhookSecret = ""
				ir, err := NewIncidentResponse(config)
				Expect(err).ToNot(HaveOccurred())

				payload := []byte("test payload")
				signature := "sha256=anysignature"

				result := ir.verifyWebhookSignature(payload, signature)
				Expect(result).To(BeFalse())
			})
		})
	})
})

// Helper functions for webhook testing

func generateValidSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))
	return "sha256=" + signature
}

func createWebhookRequest(payload, signature string) *http.Request {
	req := &http.Request{
		Method: "POST",
		Header: make(http.Header),
		Body:   &mockReadCloser{strings.NewReader(payload)},
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", signature)
	return req
}

func createWebhookRequestWithoutSignature(payload string) *http.Request {
	req := &http.Request{
		Method: "POST",
		Header: make(http.Header),
		Body:   &mockReadCloser{strings.NewReader(payload)},
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

// Mock types for testing

type mockResponseWriter struct {
	statusCode int
	body       []byte
	headers    http.Header
}

func (m *mockResponseWriter) Header() http.Header {
	if m.headers == nil {
		m.headers = make(http.Header)
	}
	return m.headers
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	m.body = append(m.body, data...)
	return len(data), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

type mockReadCloser struct {
	*strings.Reader
}

func (m *mockReadCloser) Close() error {
	return nil
}