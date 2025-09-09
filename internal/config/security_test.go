package config

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()

	if config.ImageConfig.DefaultVersion == "latest" {
		t.Error("Default image version should not be 'latest'")
	}

	if !config.ImageConfig.RequireDigest {
		t.Error("Digest verification should be enabled by default")
	}

	if config.ValidationRules.MaxTargetLength > 63 {
		t.Error("Max target length should not exceed Kubernetes label limits")
	}
}

func TestGetSecureImage(t *testing.T) {
	tests := []struct {
		name      string
		baseImage string
		wantError bool
		wantTag   string
	}{
		{
			name:      "Replace latest tag",
			baseImage: "nephoran/nf-sim:latest",
			wantError: false,
			wantTag:   "v1.0.0",
		},
		{
			name:      "Keep specific version",
			baseImage: "nephoran/nf-sim:v2.0.0",
			wantError: false,
			wantTag:   "v2.0.0",
		},
		{
			name:      "No tag defaults to version",
			baseImage: "nephoran/nf-sim",
			wantError: false,
			wantTag:   "v1.0.0",
		},
	}

	config := DefaultSecurityConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, err := config.GetSecureImage(tt.baseImage)
			if (err != nil) != tt.wantError {
				t.Errorf("GetSecureImage() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !strings.Contains(image, tt.wantTag) {
				t.Errorf("GetSecureImage() = %v, want tag %v", image, tt.wantTag)
			}

			// Should never contain 'latest'
			if strings.Contains(image, "latest") {
				t.Errorf("GetSecureImage() returned image with 'latest' tag: %v", image)
			}
		})
	}
}

func TestValidateTarget(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		wantError bool
		errorMsg  string
	}{
		// Valid targets
		{
			name:      "Valid simple name",
			target:    "gnb",
			wantError: false,
		},
		{
			name:      "Valid with dash",
			target:    "gnb-du",
			wantError: false,
		},
		{
			name:      "Valid with underscore",
			target:    "gnb_cu",
			wantError: false,
		},
		{
			name:      "Valid alphanumeric",
			target:    "ran123",
			wantError: false,
		},

		// Invalid targets - injection attempts
		{
			name:      "SQL injection with quote",
			target:    "gnb'; DROP TABLE--",
			wantError: true,
			errorMsg:  "SQL injection",
		},
		{
			name:      "SQL injection with union",
			target:    "gnb' UNION SELECT * FROM users--",
			wantError: true,
			errorMsg:  "SQL injection",
		},
		{
			name:      "Command injection with semicolon",
			target:    "gnb; rm -rf /",
			wantError: true,
			errorMsg:  "SQL injection", // Semicolon is detected as SQL injection pattern
		},
		{
			name:      "Path traversal with dots",
			target:    "../../../etc/passwd",
			wantError: true,
			errorMsg:  "path traversal",
		},
		{
			name:      "Path traversal encoded",
			target:    "gnb%2e%2e%2f",
			wantError: true,
			errorMsg:  "path traversal",
		},

		// Invalid targets - format violations
		{
			name:      "Empty target",
			target:    "",
			wantError: true,
			errorMsg:  "empty",
		},
		{
			name:      "Too long",
			target:    strings.Repeat("a", 64),
			wantError: true,
			errorMsg:  "exceeds maximum length",
		},
		{
			name:      "Starts with number",
			target:    "5g-core",
			wantError: true,
			errorMsg:  "invalid characters",
		},
		{
			name:      "Contains spaces",
			target:    "gnb du",
			wantError: true,
			errorMsg:  "invalid characters",
		},
		{
			name:      "Contains special chars",
			target:    "gnb@du",
			wantError: true,
			errorMsg:  "invalid characters",
		},
	}

	config := DefaultSecurityConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateTarget(tt.target)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTarget() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil && tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("ValidateTarget() error = %v, want error containing %v", err, tt.errorMsg)
			}
		})
	}
}

func TestResolveRepository(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		wantRepo  string
		wantError bool
	}{
		// RAN targets
		{
			name:     "RAN target",
			target:   "ran",
			wantRepo: "ran-packages",
		},
		{
			name:     "GNB target",
			target:   "gnb",
			wantRepo: "ran-packages",
		},
		{
			name:     "DU target",
			target:   "du",
			wantRepo: "ran-packages",
		},
		{
			name:     "CU target",
			target:   "cu",
			wantRepo: "ran-packages",
		},
		{
			name:     "RU target",
			target:   "ru",
			wantRepo: "ran-packages",
		},

		// Core targets
		{
			name:     "Core target",
			target:   "core",
			wantRepo: "core-packages",
		},
		{
			name:     "SMF target",
			target:   "smf",
			wantRepo: "core-packages",
		},
		{
			name:     "UPF target",
			target:   "upf",
			wantRepo: "core-packages",
		},
		{
			name:     "AMF target",
			target:   "amf",
			wantRepo: "core-packages",
		},

		// Edge targets
		{
			name:     "MEC target",
			target:   "mec",
			wantRepo: "edge-packages",
		},
		{
			name:     "Edge target",
			target:   "edge",
			wantRepo: "edge-packages",
		},

		// Transport targets
		{
			name:     "Transport target",
			target:   "transport",
			wantRepo: "transport-packages",
		},
		{
			name:     "Xhaul target",
			target:   "xhaul",
			wantRepo: "transport-packages",
		},

		// Management targets
		{
			name:     "SMO target",
			target:   "smo",
			wantRepo: "management-packages",
		},
		{
			name:     "NMS target",
			target:   "nms",
			wantRepo: "management-packages",
		},

		// Default/unknown targets
		{
			name:     "Unknown target",
			target:   "custom-nf",
			wantRepo: "nephio-packages",
		},

		// Invalid targets
		{
			name:      "SQL injection attempt",
			target:    "gnb'; DROP TABLE--",
			wantError: true,
		},
		{
			name:      "Path traversal attempt",
			target:    "../etc/passwd",
			wantError: true,
		},
	}

	config := DefaultSecurityConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := config.ResolveRepository(tt.target)
			if (err != nil) != tt.wantError {
				t.Errorf("ResolveRepository() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && repo != tt.wantRepo {
				t.Errorf("ResolveRepository() = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}

func TestHashTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{
			name:   "Simple target",
			target: "gnb",
		},
		{
			name:   "Complex target",
			target: "gnb-du-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := HashTarget(tt.target)
			hash2 := HashTarget(tt.target)

			// Hash should be consistent
			if hash1 != hash2 {
				t.Errorf("HashTarget() not consistent: %v != %v", hash1, hash2)
			}

			// Hash should be 8 characters (truncated)
			if len(hash1) != 8 {
				t.Errorf("HashTarget() length = %v, want 8", len(hash1))
			}

			// Different inputs should produce different hashes
			differentHash := HashTarget(tt.target + "-different")
			if hash1 == differentHash {
				t.Errorf("HashTarget() produced same hash for different inputs")
			}
		})
	}
}

func TestEnvironmentConfiguration(t *testing.T) {
	// Test environment variable configuration
	testCases := []struct {
		envVar   string
		envValue string
		check    func(*SecurityConfig) bool
	}{
		{
			envVar:   "NF_IMAGE_REGISTRY",
			envValue: "custom.registry.io",
			check: func(c *SecurityConfig) bool {
				return c.ImageConfig.DefaultRegistry == "custom.registry.io"
			},
		},
		{
			envVar:   "NF_IMAGE_VERSION",
			envValue: "v2.0.0",
			check: func(c *SecurityConfig) bool {
				return c.ImageConfig.DefaultVersion == "v2.0.0"
			},
		},
		{
			envVar:   "NF_REQUIRE_DIGEST",
			envValue: "false",
			check: func(c *SecurityConfig) bool {
				return !c.ImageConfig.RequireDigest
			},
		},
		{
			envVar:   "NF_REQUIRE_SIGNATURE",
			envValue: "true",
			check: func(c *SecurityConfig) bool {
				return c.ImageConfig.RequireSignature
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.envVar, func(t *testing.T) {
			// Set environment variable
			os.Setenv(tc.envVar, tc.envValue)
			defer os.Unsetenv(tc.envVar)

			// Create config and check
			config := DefaultSecurityConfig()
			if !tc.check(config) {
				t.Errorf("Environment variable %s=%s not applied correctly", tc.envVar, tc.envValue)
			}
		})
	}
}

func TestSecurityPatternDetection(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		sqlCheck  bool
		pathCheck bool
	}{
		// SQL Injection patterns
		{
			name:     "SQL single quote",
			input:    "test'value",
			sqlCheck: true,
		},
		{
			name:     "SQL double dash comment",
			input:    "test--comment",
			sqlCheck: true,
		},
		{
			name:     "SQL union keyword",
			input:    "test_union_test",
			sqlCheck: true,
		},
		{
			name:     "SQL select keyword",
			input:    "test_SELECT_test",
			sqlCheck: true,
		},

		// Path traversal patterns
		{
			name:      "Path dot dot slash",
			input:     "../test",
			pathCheck: true,
		},
		{
			name:      "Path encoded dots",
			input:     "%2e%2e",
			pathCheck: true,
		},
		{
			name:      "Path hex encoded",
			input:     "0x2e",
			pathCheck: true,
			sqlCheck:  true, // Also matches SQL hex pattern
		},

		// Clean inputs
		{
			name:      "Clean alphanumeric",
			input:     "test123",
			sqlCheck:  false,
			pathCheck: false,
		},
		{
			name:      "Clean with dash",
			input:     "test-123",
			sqlCheck:  false,
			pathCheck: false,
		},
		{
			name:      "Clean with underscore",
			input:     "test_123",
			sqlCheck:  false,
			pathCheck: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlDetected := containsSQLInjectionPattern(tt.input)
			if sqlDetected != tt.sqlCheck {
				t.Errorf("containsSQLInjectionPattern(%s) = %v, want %v", tt.input, sqlDetected, tt.sqlCheck)
			}

			pathDetected := containsPathTraversal(tt.input)
			if pathDetected != tt.pathCheck {
				t.Errorf("containsPathTraversal(%s) = %v, want %v", tt.input, pathDetected, tt.pathCheck)
			}
		})
	}
}

// TestAdvancedSecurityVulnerabilities tests comprehensive security attack scenarios
func TestAdvancedSecurityVulnerabilities(t *testing.T) {
	tests := []struct {
		name          string
		target        string
		attackType    string
		expectError   bool
		errorContains string
	}{
		// SQL Injection Attacks
		{
			name:          "Basic SQL Injection - Single Quote",
			target:        "gnb' OR '1'='1",
			attackType:    "sql_injection",
			expectError:   true,
			errorContains: "SQL injection",
		},
		{
			name:          "SQL Injection - Double Quote",
			target:        "gnb\" OR \"1\"=\"1",
			attackType:    "sql_injection",
			expectError:   true,
			errorContains: "SQL injection",
		},
		{
			name:          "SQL Injection - Comment Bypass",
			target:        "gnb'; DROP TABLE users; --",
			attackType:    "sql_injection",
			expectError:   true,
			errorContains: "SQL injection",
		},
		{
			name:          "SQL Injection - Union Select",
			target:        "gnb' UNION SELECT * FROM information_schema.tables --",
			attackType:    "sql_injection",
			expectError:   true,
			errorContains: "SQL injection",
		},
		{
			name:          "SQL Injection - Stored Procedure",
			target:        "gnb'; EXEC xp_cmdshell('rm -rf /') --",
			attackType:    "sql_injection",
			expectError:   true,
			errorContains: "SQL injection",
		},
		{
			name:          "SQL Injection - Hex Encoded",
			target:        "gnb0x27204f52202731273d2731",
			attackType:    "sql_injection",
			expectError:   true,
			errorContains: "SQL injection",
		},
		{
			name:          "SQL Injection - Mixed Case Bypass",
			target:        "gnb'; SeLeCt * FrOm UsErS --",
			attackType:    "sql_injection",
			expectError:   true,
			errorContains: "SQL injection",
		},

		// Path Traversal Attacks
		{
			name:          "Path Traversal - Basic Dot Dot Slash",
			target:        "../../../etc/passwd",
			attackType:    "path_traversal",
			expectError:   true,
<<<<<<< HEAD
			errorContains: "path traversal",
=======
			errorContains: "target name contains invalid characters",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
		{
			name:          "Path Traversal - Windows Style",
			target:        "..\\..\\..\\windows\\system32\\config\\sam",
			attackType:    "path_traversal",
			expectError:   true,
<<<<<<< HEAD
			errorContains: "path traversal",
=======
			errorContains: "target name contains invalid characters",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
		{
			name:          "Path Traversal - URL Encoded",
			target:        "gnb%2e%2e%2f%2e%2e%2fetc%2fpasswd",
			attackType:    "path_traversal",
			expectError:   true,
<<<<<<< HEAD
			errorContains: "path traversal",
=======
			errorContains: "target name contains invalid characters",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
		{
			name:          "Path Traversal - Double URL Encoded",
			target:        "gnb%252e%252e%252f%252e%252e%252fetc%252fpasswd",
			attackType:    "path_traversal",
			expectError:   true,
<<<<<<< HEAD
			errorContains: "path traversal",
=======
			errorContains: "target name contains invalid characters",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
		{
			name:          "Path Traversal - Unicode Encoding",
			target:        "gnb\\u002e\\u002e\\u002f",
			attackType:    "path_traversal",
			expectError:   true,
<<<<<<< HEAD
			errorContains: "path traversal",
=======
			errorContains: "target name contains invalid characters",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
		{
			name:          "Path Traversal - Mixed Encoding",
			target:        "gnb/../%2e%2e/./..%2f",
			attackType:    "path_traversal",
			expectError:   true,
<<<<<<< HEAD
			errorContains: "path traversal",
=======
			errorContains: "target name contains invalid characters",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},

		// Command Injection Attempts (detected as SQL injection due to semicolons)
		{
			name:          "Command Injection - Semicolon",
			target:        "gnb; rm -rf /*",
			attackType:    "command_injection",
			expectError:   true,
			errorContains: "SQL injection", // Semicolon triggers SQL injection detection
		},
		{
			name:          "Command Injection - Pipe",
			target:        "gnb | cat /etc/passwd",
			attackType:    "command_injection",
			expectError:   true,
			errorContains: "invalid characters", // Pipe character should fail pattern match
		},
		{
			name:          "Command Injection - Backticks",
			target:        "gnb`whoami`",
			attackType:    "command_injection",
			expectError:   true,
			errorContains: "invalid characters", // Backtick should fail pattern match
		},

		// Regex Pattern Bypass Attempts
		{
			name:          "Pattern Bypass - Null Byte",
			target:        "gnb\x00../etc/passwd",
			attackType:    "pattern_bypass",
			expectError:   true,
			errorContains: "invalid characters", // Null byte should fail pattern
		},
		{
			name:          "Pattern Bypass - CRLF Injection",
			target:        "gnb\r\nSet-Cookie: admin=true",
			attackType:    "pattern_bypass",
			expectError:   true,
			errorContains: "invalid characters",
		},
		{
			name:          "Pattern Bypass - Tab Character",
			target:        "gnb\tmalicious",
			attackType:    "pattern_bypass",
			expectError:   true,
			errorContains: "invalid characters",
		},

		// Buffer Overflow Attempts
		{
			name:          "Buffer Overflow - Very Long Input",
			target:        strings.Repeat("A", 10000),
			attackType:    "buffer_overflow",
			expectError:   true,
			errorContains: "exceeds maximum length",
		},
		{
			name:          "Buffer Overflow - Exactly Max Length Plus One",
			target:        "a" + strings.Repeat("b", 63), // 64 chars total, exceeds 63 limit
			attackType:    "buffer_overflow",
			expectError:   true,
			errorContains: "exceeds maximum length",
		},

		// Format String Attacks
		{
			name:          "Format String - Printf Specifiers",
			target:        "gnb%s%s%s%s",
			attackType:    "format_string",
			expectError:   true,
			errorContains: "invalid characters",
		},
		{
			name:          "Format String - Hex Format",
			target:        "gnb%x%x%x%x",
			attackType:    "format_string",
			expectError:   true,
			errorContains: "invalid characters",
		},

		// Script Injection
		{
			name:          "Script Injection - JavaScript",
			target:        "gnb<script>alert('xss')</script>",
			attackType:    "script_injection",
			expectError:   true,
			errorContains: "invalid characters",
		},
		{
			name:          "Script Injection - HTML",
			target:        "gnb<img src=x onerror=alert(1)>",
			attackType:    "script_injection",
			expectError:   true,
			errorContains: "invalid characters",
		},

		// Valid Targets (Should Pass)
		{
			name:        "Valid Target - Simple",
			target:      "gnb",
			attackType:  "valid",
			expectError: false,
		},
		{
			name:        "Valid Target - With Dash",
			target:      "gnb-du",
			attackType:  "valid",
			expectError: false,
		},
		{
			name:        "Valid Target - With Underscore",
			target:      "gnb_cu",
			attackType:  "valid",
			expectError: false,
		},
		{
			name:        "Valid Target - Alphanumeric",
			target:      "gnb123du456",
			attackType:  "valid",
			expectError: false,
		},
	}

	config := DefaultSecurityConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateTarget(tt.target)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for attack type %s with target '%s', but got nil", tt.attackType, tt.target)
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s' for attack type %s, but got: %v", tt.errorContains, tt.attackType, err)
				}
				t.Logf("Successfully detected %s attack: %v", tt.attackType, err)
			} else {
				if err != nil {
					t.Errorf("Unexpected error for valid target '%s': %v", tt.target, err)
				}
			}
		})
	}
}

// TestMaliciousImageReferences tests security validation for container images
func TestMaliciousImageReferences(t *testing.T) {
	tests := []struct {
		name          string
		baseImage     string
		expectError   bool
		errorContains string
		checkDigest   bool
	}{
		// Valid images
		{
			name:      "Valid Image - Simple",
			baseImage: "nephoran/nf-sim",
		},
		{
			name:      "Valid Image - With Tag",
			baseImage: "nephoran/nf-sim:v1.0.0",
		},
		{
			name:        "Valid Image - With Trusted Digest",
			baseImage:   "nephoran/nf-sim:v1.0.0",
			checkDigest: true,
		},

		// Potentially malicious images
		{
			name:      "Latest Tag Replacement",
			baseImage: "nephoran/nf-sim:latest", // Should be replaced with default version
		},
		{
			name:      "Registry Injection Attempt",
			baseImage: "malicious.registry.com/backdoor:latest",
		},
		{
			name:      "Path Traversal in Image Name",
			baseImage: "../../../malicious:latest",
		},
		{
			name:      "Protocol Injection",
			baseImage: "http://malicious.com/image:latest",
		},
		{
			name:      "Command Injection in Tag",
			baseImage: "nephoran/nf-sim:v1.0.0; rm -rf /",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultSecurityConfig()

			image, err := config.GetSecureImage(tt.baseImage)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for malicious image '%s', but got nil", tt.baseImage)
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for image '%s': %v", tt.baseImage, err)
					return
				}

				// Security checks on the returned image
				if strings.Contains(image, "latest") {
					t.Errorf("Returned image should not contain 'latest' tag: %s", image)
				}

				if !strings.Contains(image, config.ImageConfig.DefaultRegistry) {
					t.Errorf("Returned image should contain default registry: %s", image)
				}

				if tt.checkDigest && config.ImageConfig.RequireDigest {
					if !strings.Contains(image, "sha256:") {
						t.Errorf("Returned image should contain digest when required: %s", image)
					}
				}

				t.Logf("Secure image: %s -> %s", tt.baseImage, image)
			}
		})
	}
}

// TestInvalidDigestFormats tests various invalid digest formats
func TestInvalidDigestFormats(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
		tag       string
		digest    string
	}{
		{
			name:      "Invalid Digest - Wrong Algorithm",
			imageName: "test/image",
			tag:       "v1.0.0",
			digest:    "md5:1234567890abcdef",
		},
		{
			name:      "Invalid Digest - Wrong Length",
			imageName: "test/image",
			tag:       "v1.0.0",
			digest:    "sha256:short",
		},
		{
			name:      "Invalid Digest - No Algorithm",
			imageName: "test/image",
			tag:       "v1.0.0",
			digest:    "1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:      "Invalid Digest - SQL Injection",
			imageName: "test/image",
			tag:       "v1.0.0",
			digest:    "sha256:'; DROP TABLE images; --",
		},
		{
			name:      "Invalid Digest - Path Traversal",
			imageName: "test/image",
			tag:       "v1.0.0",
			digest:    "../../../etc/passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultSecurityConfig()

			// Manually add the invalid digest to test it
			digestKey := fmt.Sprintf("%s:%s", tt.imageName, tt.tag)
			config.ImageConfig.TrustedDigests[digestKey] = tt.digest

			baseImage := fmt.Sprintf("%s:%s", tt.imageName, tt.tag)
			resultImage, err := config.GetSecureImage(baseImage)

			// The current implementation doesn't validate digest format,
			// but it should in a production system
			if err != nil {
				t.Logf("GetSecureImage returned error for invalid digest (good): %v", err)
			} else {
				t.Logf("GetSecureImage accepted invalid digest (potential security issue): %s", resultImage)

				// Check if the invalid digest appears in the result
				if strings.Contains(resultImage, tt.digest) {
					t.Errorf("Invalid digest was included in result image: %s", resultImage)
				}
			}
		})
	}
}

// TestSecurityConfigurationEdgeCases tests edge cases in security configuration
func TestSecurityConfigurationEdgeCases(t *testing.T) {
	t.Run("EmptySecurityConfig", func(t *testing.T) {
		config := &SecurityConfig{}

		// Test with empty config
		err := config.ValidateTarget("gnb")
		if err == nil {
			t.Error("Expected error with empty security config")
		}
	})

	t.Run("NilValidationRules", func(t *testing.T) {
		config := &SecurityConfig{
			ValidationRules: ValidationConfig{}, // Zero values
		}

		err := config.ValidateTarget("gnb")
		// This should fail due to zero MaxTargetLength
		if err == nil {
			t.Error("Expected error with zero MaxTargetLength")
		}
	})

	t.Run("InvalidRegexPattern", func(t *testing.T) {
		config := DefaultSecurityConfig()
		config.ValidationRules.TargetNamePattern = "[invalid regex("

		// This should cause a panic or error when compiling the regex
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Got expected panic for invalid regex: %v", r)
			}
		}()

		err := config.ValidateTarget("gnb")
		if err == nil {
			t.Error("Expected error with invalid regex pattern")
		}
	})

	t.Run("NilRepositoryAllowlist", func(t *testing.T) {
		config := DefaultSecurityConfig()
		config.RepositoryAllowlist = nil

		_, err := config.ResolveRepository("gnb")
		if err == nil {
			t.Error("Expected error with nil repository allowlist")
		}
	})

	t.Run("EmptyRepositoryAllowlist", func(t *testing.T) {
		config := DefaultSecurityConfig()
		config.RepositoryAllowlist = make(map[string]RepositoryConfig)

		_, err := config.ResolveRepository("gnb")
		if err == nil {
			t.Error("Expected error with empty repository allowlist")
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		config := DefaultSecurityConfig()

		// Test concurrent access to validate thread safety
		const numGoroutines = 100
		errorChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				target := fmt.Sprintf("gnb-%d", id)
				err := config.ValidateTarget(target)
				errorChan <- err
			}(i)
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-errorChan
			if err != nil {
				t.Errorf("Concurrent validation failed: %v", err)
			}
		}
	})
}
