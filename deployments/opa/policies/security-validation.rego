# Security-focused validation policies for Nephoran Intent Operator
# Focuses on security threats, attack patterns, and compliance

package nephoran.security.validation

import rego.v1

# ============================================================================
# SECURITY THREAT DETECTION
# ============================================================================

# Command injection patterns
command_injection_patterns := [
	"(?i);\\s*(rm|del|format|shutdown|reboot)",
	"(?i)\\|\\s*(cat|type|more|less)",
	"(?i)&&\\s*(whoami|id|pwd)",
	"(?i)`[^`]*`",  # Backticks
	"(?i)\\$\\([^)]*\\)",  # Command substitution
	"(?i)nc\\s+-",  # Netcat
	"(?i)curl\\s+",
	"(?i)wget\\s+",
	"(?i)/bin/(sh|bash|dash)",
	"(?i)powershell",
]

# Path traversal patterns
path_traversal_patterns := [
	"\\.\\./",
	"\\.\\.\\\\",
	"%2e%2e%2f",
	"%2e%2e%5c",
	"..%2f",
	"..\\5c",
]

# LDAP injection patterns
ldap_injection_patterns := [
	"\\*\\)\\(",
	"\\)\\(&",
	"\\*\\)\\|",
	"\\|\\(\\*",
]

# NoSQL injection patterns
nosql_injection_patterns := [
	"(?i)\\$ne:",
	"(?i)\\$gt:",
	"(?i)\\$lt:",
	"(?i)\\$where:",
	"(?i)\\$regex:",
	"(?i)\\$exists:",
]

# Detect command injection attempts
deny contains msg if {
	some pattern in command_injection_patterns
	regex.match(pattern, input.body)
	msg := sprintf("Command injection attempt detected: %s", [pattern])
}

# Detect path traversal attempts
deny contains msg if {
	some pattern in path_traversal_patterns
	contains(input.body, pattern)
	msg := sprintf("Path traversal attempt detected: %s", [pattern])
}

# Detect LDAP injection attempts
deny contains msg if {
	some pattern in ldap_injection_patterns
	regex.match(pattern, input.body)
	msg := sprintf("LDAP injection attempt detected: %s", [pattern])
}

# Detect NoSQL injection attempts
deny contains msg if {
	some pattern in nosql_injection_patterns
	regex.match(pattern, input.body)
	msg := sprintf("NoSQL injection attempt detected: %s", [pattern])
}

# ============================================================================
# HEADER SECURITY VALIDATION
# ============================================================================

# Required security headers for responses (would be enforced by middleware)
required_response_headers := {
	"X-Content-Type-Options",
	"X-Frame-Options",
	"X-XSS-Protection",
	"Strict-Transport-Security",
}

# Dangerous headers that should not be present in requests
dangerous_headers := {
	"X-Forwarded-Proto",
	"X-Forwarded-Host", 
	"X-Original-URL",
	"X-Rewrite-URL",
}

# Block requests with dangerous headers
deny contains msg if {
	some header in dangerous_headers
	input.headers[lower(header)]
	msg := sprintf("Dangerous header detected: %s", [header])
}

# Validate User-Agent header format
deny contains msg if {
	user_agent := input.headers["user-agent"]
	user_agent
	count(user_agent) > 512
	msg := "User-Agent header exceeds maximum length of 512 characters"
}

# Block suspicious User-Agent patterns
suspicious_user_agents := [
	"(?i)(sqlmap|nmap|nikto|burp|zap)",
	"(?i)(bot|crawler|spider)",
	"(?i)(scan|attack|exploit)",
]

deny contains msg if {
	user_agent := input.headers["user-agent"]
	some pattern in suspicious_user_agents
	regex.match(pattern, user_agent)
	msg := sprintf("Suspicious User-Agent detected: %s", [user_agent])
}

# ============================================================================
# IP AND NETWORK VALIDATION
# ============================================================================

# Blocked IP ranges (RFC 1918 private networks should not access public APIs)
blocked_ip_ranges := [
	"127.0.0.0/8",    # Localhost
	"169.254.0.0/16", # Link-local
	"224.0.0.0/4",    # Multicast
]

# Allowed IP ranges for admin endpoints
admin_ip_ranges := [
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
]

# Block requests from suspicious IP ranges for public endpoints
deny contains msg if {
	client_ip := input.client_ip
	input.path not in {"/healthz", "/readyz", "/metrics"}
	some blocked_range in blocked_ip_ranges
	net.cidr_contains(blocked_range, client_ip)
	msg := sprintf("Request blocked from IP range %s (client: %s)", [blocked_range, client_ip])
}

# Restrict admin endpoints to internal networks
deny contains msg if {
	client_ip := input.client_ip
	startswith(input.path, "/admin/")
	not any_cidr_match(admin_ip_ranges, client_ip)
	msg := sprintf("Admin endpoint access denied from IP: %s", [client_ip])
}

# ============================================================================
# CONTENT SECURITY VALIDATION
# ============================================================================

# Maximum nesting depth for JSON objects
max_json_depth := 10

# Validate JSON structure complexity
deny contains msg if {
	input.method in {"POST", "PUT", "PATCH"}
	input.headers["content-type"] == "application/json"
	request_data := json.unmarshal(input.body)
	depth := calculate_json_depth(request_data)
	depth > max_json_depth
	msg := sprintf("JSON nesting depth %d exceeds maximum %d", [depth, max_json_depth])
}

# Block binary content in JSON requests
deny contains msg if {
	input.method in {"POST", "PUT", "PATCH"}
	input.headers["content-type"] == "application/json"
	contains(input.body, "\\u0000")  # Null byte
	msg := "Binary content detected in JSON request"
}

# Validate file upload content (if applicable)
allowed_mime_types := {
	"application/json",
	"text/plain",
	"text/yaml",
	"application/yaml",
}

deny contains msg if {
	content_type := input.headers["content-type"]
	content_type
	not content_type in allowed_mime_types
	msg := sprintf("Content type not allowed: %s", [content_type])
}

# ============================================================================
# TELECOMMUNICATIONS SECURITY
# ============================================================================

# Valid encryption algorithms for telecom context
allowed_encryption_algorithms := {
	"AES-256-GCM",
	"ChaCha20-Poly1305",
	"ECDHE-RSA-AES256-GCM-SHA384",
	"ECDHE-ECDSA-AES256-GCM-SHA384",
}

# Validate encryption configuration in requests
deny contains msg if {
	input.path == "/process"
	request_data := json.unmarshal(input.body)
	security_config := request_data.security_config
	encryption := security_config.encryption
	encryption.algorithm
	not encryption.algorithm in allowed_encryption_algorithms
	msg := sprintf("Unsupported encryption algorithm: %s", [encryption.algorithm])
}

# Valid key lengths for different algorithms
min_key_lengths := {
	"RSA": 2048,
	"ECDSA": 256,
	"AES": 256,
}

deny contains msg if {
	input.path == "/process"
	request_data := json.unmarshal(input.body)
	security_config := request_data.security_config
	key_config := security_config.key_config
	algorithm := key_config.type
	key_length := key_config.length
	min_length := object.get(min_key_lengths, algorithm, 0)
	key_length < min_length
	msg := sprintf("Key length %d too short for %s, minimum %d", [key_length, algorithm, min_length])
}

# ============================================================================
# COMPLIANCE VALIDATION
# ============================================================================

# GDPR compliance checks
deny contains msg if {
	contains(input.body, "personal_data")
	request_data := json.unmarshal(input.body)
	personal_data := request_data.personal_data
	not personal_data.consent_given
	msg := "Personal data processing requires explicit consent (GDPR compliance)"
}

# NIST cybersecurity framework validation
nist_security_levels := {"low", "moderate", "high"}

deny contains msg if {
	input.path == "/process"
	request_data := json.unmarshal(input.body)
	security_config := request_data.security_config
	nist_level := security_config.nist_level
	nist_level
	not nist_level in nist_security_levels
	msg := sprintf("Invalid NIST security level: %s", [nist_level])
}

# ============================================================================
# HELPER FUNCTIONS
# ============================================================================

# Check if IP matches any CIDR range
any_cidr_match(ranges, ip) if {
	some range in ranges
	net.cidr_contains(range, ip)
}

# Calculate maximum depth of JSON object
calculate_json_depth(obj) := depth if {
	is_object(obj)
	depths := [calculate_json_depth(obj[key]) | some key]
	depth := max(depths) + 1
} else := depth if {
	is_array(obj)
	depths := [calculate_json_depth(obj[i]) | some i]
	depth := max(depths) + 1
} else := 1