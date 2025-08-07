# Comprehensive API Request Validation Policy
# Open Policy Agent (OPA) policies for Nephoran Intent Operator
# Validates incoming API requests for security, format, and compliance

package nephoran.api.validation

import rego.v1

# ============================================================================
# REQUEST SIZE VALIDATION
# ============================================================================

# Maximum request sizes (in bytes)
max_request_sizes := {
	"default": 1048576,         # 1MB default
	"/process": 2097152,        # 2MB for intent processing
	"/stream": 1048576,         # 1MB for streaming
	"/admin/status": 4096,      # 4KB for status endpoints
	"/healthz": 1024,           # 1KB for health checks
	"/readyz": 1024,            # 1KB for readiness checks
	"/metrics": 1024,           # 1KB for metrics
}

# Deny requests exceeding size limits
deny contains msg if {
	input.method in {"POST", "PUT", "PATCH"}
	content_length := to_number(input.headers["content-length"])
	path := input.path
	max_size := object.get(max_request_sizes, path, max_request_sizes.default)
	content_length > max_size
	msg := sprintf("Request size %d exceeds limit %d for path %s", [content_length, max_size, path])
}

# ============================================================================
# RATE LIMITING VALIDATION
# ============================================================================

# Rate limits per user (requests per second)
rate_limits := {
	"anonymous": 5,
	"user": 10,
	"operator": 25,
	"admin": 50,
}

# Rate limit enforcement (requires external rate limiting data)
deny contains msg if {
	user_type := get_user_type(input)
	limit := object.get(rate_limits, user_type, rate_limits.anonymous)
	current_rate := get_current_rate(input.user_id, input.timestamp)
	current_rate > limit
	msg := sprintf("Rate limit exceeded: %d req/s > %d req/s for user type %s", [current_rate, limit, user_type])
}

# ============================================================================
# JWT TOKEN VALIDATION
# ============================================================================

# JWT validation rules
deny contains msg if {
	input.method != "GET"
	input.path not in {"/healthz", "/readyz", "/metrics"}
	not input.headers.authorization
	msg := "Authorization header required for non-GET requests to protected endpoints"
}

deny contains msg if {
	input.headers.authorization
	not startswith(input.headers.authorization, "Bearer ")
	msg := "Authorization header must use Bearer token format"
}

# Token expiration validation
deny contains msg if {
	token := extract_token(input.headers.authorization)
	payload := decode_jwt_payload(token)
	current_time := time.now_ns() / 1000000000  # Convert to seconds
	payload.exp < current_time
	msg := sprintf("JWT token expired at %d, current time %d", [payload.exp, current_time])
}

# Token issuer validation
deny contains msg if {
	token := extract_token(input.headers.authorization)
	payload := decode_jwt_payload(token)
	not payload.iss in {"nephoran-auth-service", "trusted-issuer"}
	msg := sprintf("Invalid JWT issuer: %s", [payload.iss])
}

# ============================================================================
# INPUT SANITIZATION AND VALIDATION
# ============================================================================

# SQL injection patterns
sql_injection_patterns := [
	"(?i)(union\\s+select)",
	"(?i)(drop\\s+table)",
	"(?i)(insert\\s+into)",
	"(?i)(delete\\s+from)",
	"(?i)(update\\s+.+\\s+set)",
	"(?i)(exec\\s*\\()",
	"(?i)(script\\s*>)",
	"'\\s*;\\s*--",
	"'\\s*or\\s+'?\\d",
	"'\\s*and\\s+'?\\d",
]

# XSS patterns
xss_patterns := [
	"(?i)<script[^>]*>",
	"(?i)</script>",
	"(?i)javascript:",
	"(?i)vbscript:",
	"(?i)onload\\s*=",
	"(?i)onerror\\s*=",
	"(?i)onclick\\s*=",
	"(?i)<iframe[^>]*>",
	"(?i)<object[^>]*>",
	"(?i)<embed[^>]*>",
]

# Check for SQL injection attempts
deny contains msg if {
	request_body := input.body
	some pattern in sql_injection_patterns
	regex.match(pattern, request_body)
	msg := sprintf("Potential SQL injection detected in request body: pattern %s", [pattern])
}

# Check for XSS attempts
deny contains msg if {
	request_body := input.body
	some pattern in xss_patterns
	regex.match(pattern, request_body)
	msg := sprintf("Potential XSS attack detected in request body: pattern %s", [pattern])
}

# Validate JSON structure for POST requests
deny contains msg if {
	input.method == "POST"
	input.path in {"/process", "/stream"}
	input.headers["content-type"] != "application/json"
	msg := "Content-Type must be application/json for POST requests"
}

# Validate request body is valid JSON
deny contains msg if {
	input.method in {"POST", "PUT", "PATCH"}
	input.headers["content-type"] == "application/json"
	not is_valid_json(input.body)
	msg := "Request body must be valid JSON"
}

# ============================================================================
# INTENT FORMAT VALIDATION
# ============================================================================

# Valid intent structure for /process endpoint
deny contains msg if {
	input.path == "/process"
	input.method == "POST"
	request_data := json.unmarshal(input.body)
	not request_data.intent
	msg := "Request must contain 'intent' field"
}

deny contains msg if {
	input.path == "/process"
	input.method == "POST"
	request_data := json.unmarshal(input.body)
	intent_text := request_data.intent
	count(intent_text) < 10
	msg := "Intent text must be at least 10 characters long"
}

deny contains msg if {
	input.path == "/process"
	input.method == "POST"
	request_data := json.unmarshal(input.body)
	intent_text := request_data.intent
	count(intent_text) > 5000
	msg := "Intent text must not exceed 5000 characters"
}

# Allowed characters in intent (telecommunications context)
allowed_intent_pattern := "^[a-zA-Z0-9\\s\\-_.,!?()/:@#\\[\\]{}=\"']+$"

deny contains msg if {
	input.path == "/process"
	input.method == "POST"
	request_data := json.unmarshal(input.body)
	intent_text := request_data.intent
	not regex.match(allowed_intent_pattern, intent_text)
	msg := "Intent contains invalid characters. Only alphanumeric, spaces, and common punctuation allowed"
}

# ============================================================================
# O-RAN COMPLIANCE VALIDATION
# ============================================================================

# Valid O-RAN interface types
valid_oran_interfaces := {
	"A1", "O1", "O2", "E2",
	"F1", "X2", "Xn", "N1", "N2", "N3", "N4", "N6",
}

# O-RAN specific validation for interface references
deny contains msg if {
	input.path == "/process"
	input.method == "POST"
	request_data := json.unmarshal(input.body)
	oran_config := request_data.oran_config
	oran_config.interface_type
	not oran_config.interface_type in valid_oran_interfaces
	msg := sprintf("Invalid O-RAN interface type: %s", [oran_config.interface_type])
}

# Validate network function names follow O-RAN naming conventions
oran_nf_pattern := "^(AMF|SMF|UPF|NSSF|PCF|UDM|UDR|AUSF|NRF|CHF|O-DU|O-CU|O-RU|Near-RT-RIC|Non-RT-RIC|SMO)$"

deny contains msg if {
	input.path == "/process"
	input.method == "POST"
	request_data := json.unmarshal(input.body)
	nf_config := request_data.network_function
	nf_config.type
	not regex.match(oran_nf_pattern, nf_config.type)
	msg := sprintf("Invalid O-RAN network function type: %s", [nf_config.type])
}

# ============================================================================
# AUTHENTICATION AND AUTHORIZATION
# ============================================================================

# Required roles for different endpoints
endpoint_roles := {
	"/process": {"operator", "admin"},
	"/stream": {"operator", "admin"},
	"/admin/status": {"admin"},
	"/admin/circuit-breaker/status": {"admin"},
}

# Role-based access control
deny contains msg if {
	required_roles := object.get(endpoint_roles, input.path, {})
	count(required_roles) > 0
	user_roles := get_user_roles(input)
	not any_role_matches(user_roles, required_roles)
	msg := sprintf("Insufficient privileges. Required roles: %v, user roles: %v", [required_roles, user_roles])
}

# ============================================================================
# TELECOMMUNICATIONS-SPECIFIC VALIDATION
# ============================================================================

# Valid 5G network slice types
valid_slice_types := {"eMBB", "URLLC", "mMTC"}

deny contains msg if {
	input.path == "/process"
	input.method == "POST"
	request_data := json.unmarshal(input.body)
	slice_config := request_data.slice_config
	slice_config.type
	not slice_config.type in valid_slice_types
	msg := sprintf("Invalid 5G slice type: %s. Must be one of: %v", [slice_config.type, valid_slice_types])
}

# Valid QoS class identifiers (5QI)
valid_5qi_values := [1, 2, 3, 4, 5, 6, 7, 8, 9, 65, 66, 67, 75]

deny contains msg if {
	input.path == "/process"
	input.method == "POST"
	request_data := json.unmarshal(input.body)
	qos_config := request_data.qos_config
	qos_config.five_qi
	not qos_config.five_qi in valid_5qi_values
	msg := sprintf("Invalid 5QI value: %d. Must be one of: %v", [qos_config.five_qi, valid_5qi_values])
}

# ============================================================================
# HELPER FUNCTIONS
# ============================================================================

# Extract JWT token from Authorization header
extract_token(auth_header) := token if {
	startswith(auth_header, "Bearer ")
	token := substring(auth_header, 7, -1)
}

# Decode JWT payload (simplified - in reality would need proper JWT validation)
decode_jwt_payload(token) := payload if {
	parts := split(token, ".")
	count(parts) == 3
	payload := json.unmarshal(base64.decode(parts[1]))
}

# Get user type from request context
get_user_type(input) := user_type if {
	token := extract_token(input.headers.authorization)
	payload := decode_jwt_payload(token)
	user_type := payload.user_type
} else := "anonymous"

# Get user roles from JWT token
get_user_roles(input) := roles if {
	token := extract_token(input.headers.authorization)
	payload := decode_jwt_payload(token)
	roles := payload.roles
} else := []

# Check if user has any required role
any_role_matches(user_roles, required_roles) if {
	some role in user_roles
	role in required_roles
}

# Get current rate for user (would be implemented with external data)
get_current_rate(user_id, timestamp) := rate if {
	# This would typically query external rate limiting store
	# For demo purposes, return 0
	rate := 0
}

# Validate JSON format
is_valid_json(body) if {
	json.unmarshal(body)
}