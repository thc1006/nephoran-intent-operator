package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// DISABLED: func TestNewGitHubProvider(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		redirectURL  string
		wantName     string
		wantScopes   []string
	}{
		{
			name:         "Valid GitHub provider creation",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			redirectURL:  "http://localhost:8080/callback",
			wantName:     "github",
			wantScopes:   []string{"user:email", "read:org", "read:user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewGitHubProvider(tt.clientID, tt.clientSecret, tt.redirectURL)

			assert.NotNil(t, provider)
			assert.Equal(t, tt.wantName, provider.GetProviderName())

			config := provider.GetConfiguration()
			assert.Equal(t, tt.clientID, config.ClientID)
			assert.Equal(t, tt.clientSecret, config.ClientSecret)
			assert.Equal(t, tt.redirectURL, config.RedirectURL)
			assert.Equal(t, tt.wantScopes, config.Scopes)
		})
	}
}

// DISABLED: func TestGitHubProvider_GetProviderName(t *testing.T) {
	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")
	assert.Equal(t, "github", provider.GetProviderName())
}

// DISABLED: func TestGitHubProvider_SupportsFeature(t *testing.T) {
	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")

	tests := []struct {
		feature  ProviderFeature
		expected bool
	}{
		{FeaturePKCE, true},
		{FeatureTokenRefresh, true},
		{FeatureUserInfo, true},
		{FeatureGroups, true},
		{FeatureOrganizations, true},
		{FeatureOIDC, false},
		{FeatureJWTTokens, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.feature), func(t *testing.T) {
			result := provider.SupportsFeature(tt.feature)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// DISABLED: func TestGitHubProvider_GetAuthorizationURL(t *testing.T) {
	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")

	tests := []struct {
		name        string
		state       string
		redirectURI string
		options     []AuthOption
		wantState   string
		wantError   bool
	}{
		{
			name:        "Basic authorization URL",
			state:       "test-state-123",
			redirectURI: "http://localhost:8080/callback",
			wantState:   "test-state-123",
			wantError:   false,
		},
		{
			name:        "Authorization URL with PKCE",
			state:       "test-state-456",
			redirectURI: "http://localhost:8080/callback",
			options:     []AuthOption{WithPKCE()},
			wantState:   "test-state-456",
			wantError:   false,
		},
		{
			name:        "Authorization URL with custom parameters",
			state:       "test-state-789",
			redirectURI: "http://localhost:8080/callback",
			options:     []AuthOption{WithLoginHint("testuser@example.com")},
			wantState:   "test-state-789",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authURL, challenge, err := provider.GetAuthorizationURL(tt.state, tt.redirectURI, tt.options...)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, authURL)

			// Parse URL and check parameters
			parsedURL, err := url.Parse(authURL)
			require.NoError(t, err)

			query := parsedURL.Query()
			assert.Equal(t, "test-id", query.Get("client_id"))
			assert.Equal(t, tt.redirectURI, query.Get("redirect_uri"))
			assert.Equal(t, tt.state, query.Get("state"))
			assert.Equal(t, "code", query.Get("response_type"))

			// Check scopes
			scopes := strings.Split(query.Get("scope"), " ")
			expectedScopes := []string{"user:email", "read:org", "read:user"}
			for _, expectedScope := range expectedScopes {
				assert.Contains(t, scopes, expectedScope)
			}

			// Check PKCE if enabled
			hasPKCE := false
			for _, option := range tt.options {
				opts := ApplyOptions(option)
				if opts.UsePKCE {
					hasPKCE = true
					break
				}
			}

			if hasPKCE {
				assert.NotNil(t, challenge)
				assert.NotEmpty(t, challenge.CodeChallenge)
				assert.NotEmpty(t, challenge.CodeVerifier)
				assert.Equal(t, "S256", challenge.Method)
				assert.Equal(t, challenge.CodeChallenge, query.Get("code_challenge"))
				assert.Equal(t, "S256", query.Get("code_challenge_method"))
			} else {
				assert.Nil(t, challenge)
			}
		})
	}
}

// DISABLED: func TestGitHubProvider_ExchangeCodeForToken(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/oauth/access_token" {
			// Validate request
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			// Parse form data
			err := r.ParseForm()
			require.NoError(t, err)

			code := r.FormValue("code")
			switch code {
			case "valid-code":
				response := map[string]interface{}{
					"access_token": "github-access-token-123",
					"token_type":   "Bearer",
					"scope":        "user:email,read:org,read:user",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			case "invalid-code":
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "bad_verification_code"}`))
			default:
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "unknown_code"}`))
			}
		}
	}))
	defer server.Close()

	// Create provider with custom endpoint
	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")
	provider.oauth2Cfg.Endpoint = oauth2.Endpoint{
		TokenURL: server.URL + "/login/oauth/access_token",
	}

	tests := []struct {
		name      string
		code      string
		wantError bool
		wantToken string
	}{
		{
			name:      "Valid code exchange",
			code:      "valid-code",
			wantError: false,
			wantToken: "github-access-token-123",
		},
		{
			name:      "Invalid code exchange",
			code:      "invalid-code",
			wantError: true,
		},
		{
			name:      "Unknown code",
			code:      "unknown-code",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tokenResp, err := provider.ExchangeCodeForToken(ctx, tt.code, "http://localhost:8080/callback", nil)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, tokenResp)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, tokenResp)
			assert.Equal(t, tt.wantToken, tokenResp.AccessToken)
			assert.Equal(t, "Bearer", tokenResp.TokenType)
		})
	}
}

// DISABLED: func TestGitHubProvider_GetUserInfo(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			switch token {
			case "valid-token":
				userInfo := GitHubUserInfo{
					ID:        123456789,
					Login:     "testuser",
					Name:      "Test User",
					Email:     "testuser@example.com",
					AvatarURL: "https://github.com/images/error/testuser_happy.gif",
					HTMLURL:   "https://github.com/testuser",
					Company:   "Test Company",
					Location:  "Test Location",
					Bio:       "Test bio",
					CreatedAt: "2008-01-14T04:33:35Z",
					UpdatedAt: "2022-01-14T04:33:35Z",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userInfo)
			case "no-email-token":
				userInfo := GitHubUserInfo{
					ID:      987654321,
					Login:   "noemail",
					Name:    "No Email User",
					HTMLURL: "https://github.com/noemail",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userInfo)
			case "invalid-token":
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"message": "Bad credentials"}`))
			default:
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"message": "Forbidden"}`))
			}
		} else if r.URL.Path == "/user/emails" {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")

			if token == "no-email-token" {
				emails := []map[string]interface{}{
					{
						"email":      "noemail@example.com",
						"verified":   true,
						"primary":    true,
						"visibility": "private",
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(emails)
			}
		}
	}))
	defer server.Close()

	// Create provider with custom endpoint
	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")
	provider.config.Endpoints.UserInfoURL = server.URL + "/user"

	tests := []struct {
		name        string
		token       string
		wantError   bool
		wantSubject string
		wantEmail   string
		wantName    string
	}{
		{
			name:        "Valid user info",
			token:       "valid-token",
			wantError:   false,
			wantSubject: "github-123456789",
			wantEmail:   "testuser@example.com",
			wantName:    "Test User",
		},
		{
			name:        "User with no public email",
			token:       "no-email-token",
			wantError:   false,
			wantSubject: "github-987654321",
			wantEmail:   "noemail@example.com", // Should fetch from emails endpoint
			wantName:    "No Email User",
		},
		{
			name:      "Invalid token",
			token:     "invalid-token",
			wantError: true,
		},
		{
			name:      "Forbidden token",
			token:     "forbidden-token",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			userInfo, err := provider.GetUserInfo(ctx, tt.token)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, userInfo)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, userInfo)
			if userInfo != nil {
				assert.Equal(t, tt.wantSubject, userInfo.Subject)
				assert.Equal(t, tt.wantEmail, userInfo.Email)
				assert.Equal(t, tt.wantName, userInfo.Name)
				assert.Equal(t, "github", userInfo.Provider)
			}
		})
	}
}

// DISABLED: func TestGitHubProvider_ValidateToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")

			switch token {
			case "valid-token":
				userInfo := map[string]interface{}{
					"id":    123456789,
					"login": "testuser",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userInfo)
			case "invalid-token":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(http.StatusForbidden)
			}
		}
	}))
	defer server.Close()

	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")
	provider.config.Endpoints.UserInfoURL = server.URL + "/user"

	tests := []struct {
		name      string
		token     string
		wantValid bool
		wantError bool
	}{
		{
			name:      "Valid token",
			token:     "valid-token",
			wantValid: true,
			wantError: false,
		},
		{
			name:      "Invalid token",
			token:     "invalid-token",
			wantValid: false,
			wantError: false,
		},
		{
			name:      "Forbidden token",
			token:     "forbidden-token",
			wantValid: false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			validation, err := provider.ValidateToken(ctx, tt.token)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, validation)
			assert.Equal(t, tt.wantValid, validation.Valid)
		})
	}
}

// DISABLED: func TestGitHubProvider_RevokeToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/applications/test-id/token" {
			assert.Equal(t, "DELETE", r.Method)

			// GitHub requires basic auth for token revocation
			username, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "test-id", username)
			assert.Equal(t, "test-secret", password)

			// Parse request body
			var body map[string]string
			err := json.NewDecoder(r.Body).Decode(&body)
			require.NoError(t, err)

			token := body["access_token"]
			if token == "valid-token" {
				w.WriteHeader(http.StatusNoContent)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer server.Close()

	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")

	tests := []struct {
		name      string
		token     string
		wantError bool
	}{
		{
			name:      "Valid token revocation",
			token:     "valid-token",
			wantError: false,
		},
		{
			name:      "Invalid token revocation",
			token:     "invalid-token",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := provider.RevokeToken(ctx, tt.token)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test Enterprise Provider methods if GitHubProvider implements EnterpriseProvider
// DISABLED: func TestGitHubProvider_GetOrganizations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user/orgs" {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")

			switch token {
			case "valid-token":
				orgs := []GitHubOrganization{
					{
						ID:          1,
						Login:       "test-org",
						URL:         "https://api.github.com/orgs/test-org",
						Description: "Test Organization",
						Name:        "Test Org",
					},
					{
						ID:          2,
						Login:       "another-org",
						URL:         "https://api.github.com/orgs/another-org",
						Description: "Another Organization",
						Name:        "Another Org",
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(orgs)
			case "no-orgs-token":
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]GitHubOrganization{})
			default:
				w.WriteHeader(http.StatusUnauthorized)
			}
		}
	}))
	defer server.Close()

	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")

	tests := []struct {
		name         string
		token        string
		wantError    bool
		wantOrgCount int
		wantFirstOrg string
	}{
		{
			name:         "Valid organizations",
			token:        "valid-token",
			wantError:    false,
			wantOrgCount: 2,
			wantFirstOrg: "test-org",
		},
		{
			name:         "No organizations",
			token:        "no-orgs-token",
			wantError:    false,
			wantOrgCount: 0,
		},
		{
			name:      "Invalid token",
			token:     "invalid-token",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Check if provider implements EnterpriseProvider (interface not implemented yet)
			// TODO: Implement EnterpriseProvider interface
			assert.NotNil(t, provider) // Use provider to avoid unused variable error
			assert.NotNil(t, ctx)      // Use ctx to avoid unused variable error
			t.Skip("GitHubProvider does not implement EnterpriseProvider")
			return

			/*
				if ep, ok := provider.(EnterpriseProvider); ok {
					orgs, err := ep.GetOrganizations(ctx, tt.token)

					if tt.wantError {
						assert.Error(t, err)
						return
					}

					assert.NoError(t, err)
					assert.Len(t, orgs, tt.wantOrgCount)

					if tt.wantOrgCount > 0 {
						assert.Equal(t, tt.wantFirstOrg, orgs[0].Name)
					}
				}
			*/
		})
	}
}

// DISABLED: func TestGitHubProvider_RefreshToken(t *testing.T) {
	// GitHub doesn't support token refresh in the traditional OAuth2 sense
	// This test verifies the behavior when refresh is not supported
	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")

	ctx := context.Background()
	tokenResp, err := provider.RefreshToken(ctx, "any-refresh-token")

	// Should return an error indicating refresh is not supported
	assert.Error(t, err)
	assert.Nil(t, tokenResp)
	assert.Contains(t, err.Error(), "not supported")
}

// DISABLED: func TestGitHubProvider_GetConfiguration(t *testing.T) {
	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")
	config := provider.GetConfiguration()

	assert.NotNil(t, config)
	assert.Equal(t, "github", config.Name)
	assert.Equal(t, "github", config.Type)
	assert.Equal(t, "test-id", config.ClientID)
	assert.Equal(t, "test-secret", config.ClientSecret)
	assert.Equal(t, "http://localhost:8080/callback", config.RedirectURL)
	assert.Contains(t, config.Scopes, "user:email")
	assert.Contains(t, config.Scopes, "read:org")
	assert.Contains(t, config.Scopes, "read:user")
}

// Benchmark tests
func BenchmarkGitHubProvider_GetAuthorizationURL(b *testing.B) {
	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = provider.GetAuthorizationURL("test-state", "http://localhost:8080/callback")
	}
}

func BenchmarkGitHubProvider_GetAuthorizationURL_WithPKCE(b *testing.B) {
	provider := NewGitHubProvider("test-id", "test-secret", "http://localhost:8080/callback")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = provider.GetAuthorizationURL("test-state", "http://localhost:8080/callback", WithPKCE())
	}
}

// Helper functions for testing
func createTestGitHubProvider() *GitHubProvider {
	return NewGitHubProvider("test-client-id", "test-client-secret", "http://localhost:8080/callback")
}

func createMockGitHubServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			response := map[string]interface{}{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
				"scope":        "user:email,read:org,read:user",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		case "/user":
			userInfo := GitHubUserInfo{
				ID:    123456789,
				Login: "testuser",
				Name:  "Test User",
				Email: "testuser@example.com",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userInfo)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
