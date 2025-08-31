// Package security provides test runner for TLS security validation
package security

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/security"
)

// RunTLSAuditTests runs comprehensive TLS audit tests
func RunTLSAuditTests() error {
	fmt.Println("Starting TLS Security Audit Tests...")

	// Test 1: O-RAN Compliance Configuration
	fmt.Println("Test 1: O-RAN Compliance Configuration")
	oranConfig, err := security.NewORANCompliantTLS("A1", "enhanced")
	if err != nil {
		return fmt.Errorf("failed to create O-RAN config: %w", err)
	}

	// Validate compliance
	if err := oranConfig.ValidateCompliance(); err != nil {
		return fmt.Errorf("O-RAN compliance validation failed: %w", err)
	}
	fmt.Println("✓ O-RAN A1 Enhanced profile configuration created successfully")

	// Test 2: TLS Config Building
	fmt.Println("Test 2: TLS Config Building")
	tlsConfig, err := oranConfig.BuildTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to build TLS config: %w", err)
	}

	if tlsConfig.MinVersion != 0x0304 { // TLS 1.3
		return fmt.Errorf("expected TLS 1.3, got version %x", tlsConfig.MinVersion)
	}
	fmt.Println("✓ TLS configuration built with correct version enforcement")

	// Test 3: TLS Auditor
	fmt.Println("Test 3: TLS Auditor Configuration")
	auditorConfig := &security.AuditorConfig{
		Endpoints:           []string{"example.com:443"},
		Timeout:             10 * time.Second,
		DeepScan:            false,
		CheckOCSP:           false,
		CheckCRL:            false,
		TestWeakCiphers:     true,
		TestRenegotiation:   false,
		ComplianceStandards: []string{"ORAN-WG11", "NIST-SP800-52"},
		OutputFormat:        "json",
	}

	auditor := security.NewTLSAuditor(auditorConfig)
	if auditor == nil {
		return fmt.Errorf("failed to create TLS auditor")
	}
	fmt.Println("✓ TLS auditor created successfully")

	// Test 4: Generate Report
	fmt.Println("Test 4: Report Generation")
	report := auditor.GenerateReport()
	if report == nil {
		return fmt.Errorf("failed to generate audit report")
	}

	if report.ReportID == "" {
		return fmt.Errorf("report ID should not be empty")
	}
	fmt.Println("✓ Audit report generated successfully")

	// Test 5: Export Report
	fmt.Println("Test 5: Report Export")
	jsonData, err := auditor.ExportReport("json")
	if err != nil {
		return fmt.Errorf("failed to export JSON report: %w", err)
	}

	if len(jsonData) == 0 {
		return fmt.Errorf("exported JSON data should not be empty")
	}
	fmt.Println("✓ Report exported to JSON successfully")

	fmt.Println("All TLS security tests passed!")
	return nil
}

// TestORANComplianceProfiles tests different O-RAN compliance profiles
func TestORANComplianceProfiles(t *testing.T) {
	profiles := []string{"baseline", "enhanced", "strict"}
	interfaces := []string{"A1", "E2", "O1", "O2"}

	for _, profile := range profiles {
		for _, iface := range interfaces {
			t.Run(fmt.Sprintf("%s-%s", iface, profile), func(t *testing.T) {
				config, err := security.NewORANCompliantTLS(iface, profile)
				if err != nil {
					// Some combinations might not be allowed
					t.Logf("Expected error for %s-%s: %v", iface, profile, err)
					return
				}

				if err := config.ValidateCompliance(); err != nil {
					t.Errorf("Compliance validation failed for %s-%s: %v", iface, profile, err)
				}

				tlsConfig, err := config.BuildTLSConfig()
				if err != nil {
					t.Errorf("Failed to build TLS config for %s-%s: %v", iface, profile, err)
				}

				// Verify minimum security requirements
				if tlsConfig.MinVersion < 0x0303 { // TLS 1.2 minimum
					t.Errorf("TLS version too low for %s-%s: %x", iface, profile, tlsConfig.MinVersion)
				}
			})
		}
	}
}

// TestTLSVersionEnforcement tests TLS version enforcement
func TestTLSVersionEnforcement(t *testing.T) {
	// Test strict profile only allows TLS 1.3
	config, err := security.NewORANCompliantTLS("A1", "strict")
	if err != nil {
		t.Fatalf("Failed to create strict config: %v", err)
	}

	tlsConfig, err := config.BuildTLSConfig()
	if err != nil {
		t.Fatalf("Failed to build TLS config: %v", err)
	}

	if tlsConfig.MinVersion != 0x0304 { // TLS 1.3
		t.Errorf("Strict profile should require TLS 1.3, got %x", tlsConfig.MinVersion)
	}

	if tlsConfig.MaxVersion != 0x0304 { // TLS 1.3
		t.Errorf("Strict profile should max at TLS 1.3, got %x", tlsConfig.MaxVersion)
	}
}

// TestCipherSuitesValidation tests cipher suite validation
func TestCipherSuitesValidation(t *testing.T) {
	// Test that strict profile has minimal cipher suites
	config, err := security.NewORANCompliantTLS("E2", "strict")
	if err != nil {
		t.Fatalf("Failed to create strict E2 config: %v", err)
	}

	if len(config.CipherSuites) != 1 {
		t.Errorf("Strict profile should have exactly 1 cipher suite, got %d", len(config.CipherSuites))
	}

	// Verify it's the strongest cipher
	expectedCipher := uint16(0x1302) // TLS_AES_256_GCM_SHA384
	if config.CipherSuites[0] != expectedCipher {
		t.Errorf("Expected strongest cipher %x, got %x", expectedCipher, config.CipherSuites[0])
	}
}

// TestAuditReportGeneration tests audit report generation
func TestAuditReportGeneration(t *testing.T) {
	config := &security.AuditorConfig{
		Endpoints:    []string{"test.example.com:443"},
		Timeout:      5 * time.Second,
		OutputFormat: "json",
	}

	auditor := security.NewTLSAuditor(config)
	report := auditor.GenerateReport()

	if report.ReportID == "" {
		t.Error("Report ID should not be empty")
	}

	if report.Timestamp.IsZero() {
		t.Error("Report timestamp should be set")
	}

	// Test JSON export
	jsonData, err := auditor.ExportReport("json")
	if err != nil {
		t.Errorf("Failed to export JSON: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("JSON export should not be empty")
	}

	// Test HTML export
	htmlData, err := auditor.ExportReport("html")
	if err != nil {
		t.Errorf("Failed to export HTML: %v", err)
	}

	if len(htmlData) == 0 {
		t.Error("HTML export should not be empty")
	}
}

// RunInteractiveDemo runs an interactive demo of TLS security features
func RunInteractiveDemo() {
	fmt.Println("=== Nephoran TLS Security Demo ===\n")

	fmt.Println("1. Creating O-RAN Compliant TLS Configurations...")

	// Demo different profiles
	profiles := []struct {
		iface   string
		profile string
		desc    string
	}{
		{"A1", "enhanced", "A1 Interface (Non-RT RIC ↔ Near-RT RIC)"},
		{"E2", "strict", "E2 Interface (Near-RT RIC ↔ E2 Nodes)"},
		{"O1", "baseline", "O1 Interface (Management)"},
	}

	for _, p := range profiles {
		fmt.Printf("\n  Creating %s configuration:\n", p.desc)
		config, err := security.NewORANCompliantTLS(p.iface, p.profile)
		if err != nil {
			log.Printf("    ❌ Error: %v", err)
			continue
		}

		fmt.Printf("    ✓ Profile: %s\n", config.SecurityProfile)
		fmt.Printf("    ✓ Compliance Level: %s\n", config.ComplianceLevel)
		fmt.Printf("    ✓ Min TLS Version: %s\n", getTLSVersionName(config.MinTLSVersion))
		fmt.Printf("    ✓ OCSP Required: %v\n", config.OCSPStaplingRequired)

		if err := config.ValidateCompliance(); err != nil {
			log.Printf("    ❌ Compliance validation failed: %v", err)
		} else {
			fmt.Printf("    ✓ O-RAN WG11 Compliant\n")
		}
	}

	fmt.Println("\n2. Running TLS Security Audit...")

	auditorConfig := &security.AuditorConfig{
		Endpoints:           []string{"google.com:443", "github.com:443"},
		Timeout:             10 * time.Second,
		DeepScan:            true,
		TestWeakCiphers:     true,
		ComplianceStandards: []string{"ORAN-WG11", "NIST-SP800-52", "OWASP-TLS"},
		OutputFormat:        "json",
	}

	auditor := security.NewTLSAuditor(auditorConfig)
	report := auditor.GenerateReport()

	fmt.Printf("    ✓ Report ID: %s\n", report.ReportID)
	fmt.Printf("    ✓ Security Posture: %s\n", report.Summary.SecurityPosture)
	fmt.Printf("    ✓ Risk Score: %d/100\n", report.RiskScore)
	fmt.Printf("    ✓ O-RAN Compliant: %v\n", report.Summary.ORANCompliant)

	if len(report.Recommendations) > 0 {
		fmt.Printf("\n    Top Recommendations:\n")
		for i, rec := range report.Recommendations {
			if i >= 3 {
				break
			}
			fmt.Printf("      %d. %s (Priority: %d)\n", i+1, rec.Title, rec.Priority)
		}
	}

	fmt.Println("\n3. Exporting Security Report...")

	jsonData, err := auditor.ExportReport("json")
	if err != nil {
		log.Printf("    ❌ JSON export failed: %v", err)
	} else {
		fmt.Printf("    ✓ JSON report: %d bytes\n", len(jsonData))
	}

	htmlData, err := auditor.ExportReport("html")
	if err != nil {
		log.Printf("    ❌ HTML export failed: %v", err)
	} else {
		fmt.Printf("    ✓ HTML report: %d bytes\n", len(htmlData))
	}

	fmt.Println("\n=== Demo Complete ===")
}

// Helper function to get TLS version name
func getTLSVersionName(version uint16) string {
	switch version {
	case 0x0300:
		return "SSL 3.0"
	case 0x0301:
		return "TLS 1.0"
	case 0x0302:
		return "TLS 1.1"
	case 0x0303:
		return "TLS 1.2"
	case 0x0304:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}
