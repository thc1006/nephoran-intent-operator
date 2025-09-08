package security

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateDigest(t *testing.T) {
	tests := []struct {
		name    string
		digest  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid sha256 digest lowercase",
			digest:  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr: false,
		},
		{
			name:    "valid sha256 digest uppercase",
			digest:  "sha256:E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
			wantErr: false,
		},
		{
			name:    "valid sha256 digest mixed case",
			digest:  "sha256:E3b0C44298fc1c149afbf4c8996FB92427ae41e4649b934CA495991B7852b855",
			wantErr: false,
		},
		{
			name:    "invalid algorithm sha512",
			digest:  "sha512:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr: true,
			errMsg:  "unsupported digest algorithm",
		},
		{
			name:    "invalid algorithm md5",
			digest:  "md5:5d41402abc4b2a76b9719d911017c592",
			wantErr: true,
			errMsg:  "unsupported digest algorithm",
		},
		{
			name:    "missing algorithm prefix",
			digest:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr: true,
			errMsg:  "invalid image digest",
		},
		{
			name:    "hex too short",
			digest:  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b85",
			wantErr: true,
			errMsg:  "invalid sha256 hex format",
		},
		{
			name:    "hex too long",
			digest:  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b8550",
			wantErr: true,
			errMsg:  "invalid sha256 hex format",
		},
		{
			name:    "invalid hex characters",
			digest:  "sha256:g3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr: true,
			errMsg:  "invalid image digest",
		},
		{
			name:    "contains double quotes",
			digest:  `sha256:"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`,
			wantErr: true,
			errMsg:  "contains forbidden characters",
		},
		{
			name:    "contains single quotes",
			digest:  "sha256:'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'",
			wantErr: true,
			errMsg:  "contains forbidden characters",
		},
		{
			name:    "contains semicolon",
			digest:  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855;",
			wantErr: true,
			errMsg:  "contains forbidden characters",
		},
		{
			name:    "contains space",
			digest:  "sha256: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr: true,
			errMsg:  "contains forbidden characters",
		},
		{
			name:    "contains tab",
			digest:  "sha256:\te3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr: true,
			errMsg:  "contains forbidden characters",
		},
		{
			name:    "contains newline",
			digest:  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\n",
			wantErr: true,
			errMsg:  "contains forbidden characters",
		},
		{
			name:    "contains carriage return",
			digest:  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\r",
			wantErr: true,
			errMsg:  "contains forbidden characters",
		},
		{
			name:    "contains null byte",
			digest:  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\x00",
			wantErr: true,
			errMsg:  "contains forbidden characters",
		},
		{
			name:    "empty string",
			digest:  "",
			wantErr: true,
			errMsg:  "invalid image digest",
		},
		{
			name:    "only algorithm",
			digest:  "sha256:",
			wantErr: true,
			errMsg:  "invalid image digest",
		},
		{
			name:    "malformed separator",
			digest:  "sha256::e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr: true,
			errMsg:  "invalid image digest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDigest(tt.digest)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDigest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateDigest() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateDigest_InjectionAttempts(t *testing.T) {
	// Test various injection attempts
	injectionTests := []struct {
		name   string
		digest string
	}{
		{
			name:   "command injection with semicolon",
			digest: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855; rm -rf /`,
		},
		{
			name:   "SQL injection attempt",
			digest: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855' OR '1'='1`,
		},
		{
			name:   "command substitution dollar",
			digest: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855$(whoami)`,
		},
		{
			name:   "command substitution backtick",
			digest: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`id`",
		},
		{
			name:   "null byte injection",
			digest: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\x00evil",
		},
		{
			name:   "CRLF injection",
			digest: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\r\nX-Injected: true",
		},
		{
			name:   "path traversal attempt",
			digest: "sha256:../../../etc/passwd",
		},
		{
			name:   "unicode control character",
			digest: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\u0000",
		},
		{
			name:   "escape sequence",
			digest: `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\x20`,
		},
	}

	for _, tt := range injectionTests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDigest(tt.digest)
			if err == nil {
				t.Errorf("ValidateDigest() should reject injection attempt: %q", tt.digest)
			}
			if !errors.Is(err, ErrInvalidDigest) && !errors.Is(err, ErrUnsupportedDigest) {
				t.Errorf("ValidateDigest() unexpected error type for injection: %v", err)
			}
		})
	}
}

func TestNormalizeDigest(t *testing.T) {
	tests := []struct {
		name     string
		digest   string
		want     string
		wantErr  bool
	}{
		{
			name:    "normalize uppercase",
			digest:  "sha256:E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
			want:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr: false,
		},
		{
			name:    "already lowercase",
			digest:  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			want:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr: false,
		},
		{
			name:    "mixed case",
			digest:  "sha256:E3b0C44298FC1c149aFbF4c8996fB92427aE41e4649B934cA495991B7852b855",
			want:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr: false,
		},
		{
			name:    "invalid digest",
			digest:  "sha256:invalid",
			want:    "",
			wantErr: true,
		},
		{
			name:    "injection attempt",
			digest:  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855; echo evil",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeDigest(tt.digest)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeDigest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeDigest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkValidateDigest(b *testing.B) {
	validDigest := "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateDigest(validDigest)
	}
}

func BenchmarkValidateDigestInvalid(b *testing.B) {
	invalidDigest := "sha256:invalidhexcharacters"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateDigest(invalidDigest)
	}
}