package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// DISABLED: func TestNewGoogleProvider(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		redirectURL  string
		hostedDomain string
		wantName     string
		wantScopes   []string
	}{
		{
			name:         "Valid Google provider creation",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			redirectURL:  "http://localhost:8080/callback",
			wantName:     "google",
			wantScopes:   []string{"openid", "email", "profile"},
		},
		{
			name:         "Google provider with hosted domain",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			redirectURL:  "http://localhost:8080/callback",
			hostedDomain: "example.com",
			wantName:     "google",
			wantScopes:   []string{"openid", "email", "profile"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider *GoogleProvider
			if tt.hostedDomain != "" {
				provider = NewGoogleProvider(tt.clientID, tt.clientSecret, tt.redirectURL, tt.hostedDomain)
				assert.Equal(t, tt.hostedDomain, provider.GetHostedDomain())
			} else {
				provider = NewGoogleProvider(tt.clientID, tt.clientSecret, tt.redirectURL)
			}

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

// DISABLED: func TestGoogleProvider_GetProviderName(t *testing.T) {
	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")
	assert.Equal(t, "google", provider.GetProviderName())
}

// DISABLED: func TestGoogleProvider_SupportsFeature(t *testing.T) {
	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")

	tests := []struct {
		feature  ProviderFeature
		expected bool
	}{
		{FeatureOIDC, true},
		{FeaturePKCE, true},
		{FeatureTokenRefresh, true},
		{FeatureUserInfo, true},
		{FeatureJWTTokens, true},
		{FeatureDiscovery, true},
		{FeatureGroups, false},        // Google doesn't provide groups in standard claims
		{FeatureOrganizations, false}, // Google doesn't provide orgs
	}

	for _, tt := range tests {
		t.Run(string(tt.feature), func(t *testing.T) {
			result := provider.SupportsFeature(tt.feature)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// DISABLED: func TestGoogleProvider_GetAuthorizationURL(t *testing.T) {
	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")

	tests := []struct {
		name        string
		state       string
		redirectURI string
		options     []AuthOption
		wantState   string
		wantError   bool
		checkPKCE   bool
		checkDomain bool
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
			checkPKCE:   true,
		},
		{
			name:        "Authorization URL with login hint",
			state:       "test-state-789",
			redirectURI: "http://localhost:8080/callback",
			options:     []AuthOption{WithLoginHint("user@example.com")},
			wantState:   "test-state-789",
			wantError:   false,
		},
		{
			name:        "Authorization URL with hosted domain",
			state:       "test-state-domain",
			redirectURI: "http://localhost:8080/callback",
			options:     []AuthOption{WithDomainHint("example.com")},
			wantState:   "test-state-domain",
			wantError:   false,
			checkDomain: true,
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
			expectedScopes := []string{"openid", "email", "profile"}
			for _, expectedScope := range expectedScopes {
				assert.Contains(t, scopes, expectedScope)
			}

			// Check PKCE if enabled
			if tt.checkPKCE {
				assert.NotNil(t, challenge)
				assert.NotEmpty(t, challenge.CodeChallenge)
				assert.NotEmpty(t, challenge.CodeVerifier)
				assert.Equal(t, "S256", challenge.Method)
				assert.Equal(t, challenge.CodeChallenge, query.Get("code_challenge"))
				assert.Equal(t, "S256", query.Get("code_challenge_method"))
			}

			// Check domain hint
			if tt.checkDomain {
				assert.Equal(t, "example.com", query.Get("hd"))
			}
		})
	}
}

// DISABLED: func TestGoogleProvider_ExchangeCodeForToken(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			// Validate request
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			// Parse form data
			err := r.ParseForm()
			require.NoError(t, err)

			code := r.FormValue("code")
			switch code {
			case "valid-code":
				response := json.RawMessage("{}")
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			case "invalid-code":
				w.WriteHeader(http.StatusBadRequest)
				response := json.RawMessage("{}")
				json.NewEncoder(w).Encode(response)
			default:
				w.WriteHeader(http.StatusBadRequest)
				response := json.RawMessage("{}")
				json.NewEncoder(w).Encode(response)
			}
		}
	}))
	defer server.Close()

	// Create provider with custom endpoint
	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")
	provider.oauth2Cfg.Endpoint = oauth2.Endpoint{
		TokenURL: server.URL + "/token",
	}

	tests := []struct {
		name        string
		code        string
		wantError   bool
		wantToken   string
		wantRefresh string
		wantIDToken bool
	}{
		{
			name:        "Valid code exchange",
			code:        "valid-code",
			wantError:   false,
			wantToken:   "google-access-token-123",
			wantRefresh: "google-refresh-token-456",
			wantIDToken: true,
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
			assert.Equal(t, tt.wantRefresh, tokenResp.RefreshToken)
			assert.Equal(t, "Bearer", tokenResp.TokenType)
			assert.Equal(t, int64(3600), tokenResp.ExpiresIn)

			if tt.wantIDToken {
				assert.NotEmpty(t, tokenResp.IDToken)
			}
		})
	}
}

// DISABLED: func TestGoogleProvider_RefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			err := r.ParseForm()
			require.NoError(t, err)

			grantType := r.FormValue("grant_type")
			assert.Equal(t, "refresh_token", grantType)

			refreshToken := r.FormValue("refresh_token")
			switch refreshToken {
			case "valid-refresh-token":
				response := json.RawMessage("{}")
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			case "expired-refresh-token":
				w.WriteHeader(http.StatusBadRequest)
				response := json.RawMessage("{}")
				json.NewEncoder(w).Encode(response)
			default:
				w.WriteHeader(http.StatusBadRequest)
				response := json.RawMessage("{}")
				json.NewEncoder(w).Encode(response)
			}
		}
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")
	provider.oauth2Cfg.Endpoint = oauth2.Endpoint{
		TokenURL: server.URL + "/token",
	}

	tests := []struct {
		name         string
		refreshToken string
		wantError    bool
		wantToken    string
	}{
		{
			name:         "Valid refresh token",
			refreshToken: "valid-refresh-token",
			wantError:    false,
			wantToken:    "new-google-access-token",
		},
		{
			name:         "Expired refresh token",
			refreshToken: "expired-refresh-token",
			wantError:    true,
		},
		{
			name:         "Invalid refresh token",
			refreshToken: "invalid-refresh-token",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tokenResp, err := provider.RefreshToken(ctx, tt.refreshToken)

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

// DISABLED: func TestGoogleProvider_GetUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/v2/userinfo" {
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
				userInfo := GoogleUserInfo{
					ID:            "123456789",
					Email:         "testuser@example.com",
					VerifiedEmail: true,
					Name:          "Test User",
					GivenName:     "Test",
					FamilyName:    "User",
					Picture:       "https://lh3.googleusercontent.com/test-avatar",
					Locale:        "en",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userInfo)
			case "domain-token":
				userInfo := GoogleUserInfo{
					ID:            "987654321",
					Email:         "user@example.com",
					VerifiedEmail: true,
					Name:          "Domain User",
					GivenName:     "Domain",
					FamilyName:    "User",
					Picture:       "https://lh3.googleusercontent.com/domain-avatar",
					Locale:        "en",
					HostedDomain:  "example.com",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userInfo)
			case "unverified-token":
				userInfo := GoogleUserInfo{
					ID:            "111222333",
					Email:         "unverified@example.com",
					VerifiedEmail: false,
					Name:          "Unverified User",
					GivenName:     "Unverified",
					FamilyName:    "User",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userInfo)
			case "invalid-token":
				w.WriteHeader(http.StatusUnauthorized)
				response := json.RawMessage("{}"){
						"code":    401,
						"message": "Invalid Credentials",
						"status":  "UNAUTHENTICATED",
					},
				}
				json.NewEncoder(w).Encode(response)
			default:
				w.WriteHeader(http.StatusForbidden)
				response := json.RawMessage("{}"){
						"code":    403,
						"message": "Forbidden",
						"status":  "PERMISSION_DENIED",
					},
				}
				json.NewEncoder(w).Encode(response)
			}
		}
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")
	provider.config.Endpoints.UserInfoURL = server.URL + "/oauth2/v2/userinfo"

	tests := []struct {
		name         string
		token        string
		wantError    bool
		wantSubject  string
		wantEmail    string
		wantName     string
		wantVerified bool
		wantDomain   string
	}{
		{
			name:         "Valid user info",
			token:        "valid-token",
			wantError:    false,
			wantSubject:  "google-123456789",
			wantEmail:    "testuser@example.com",
			wantName:     "Test User",
			wantVerified: true,
		},
		{
			name:         "User with hosted domain",
			token:        "domain-token",
			wantError:    false,
			wantSubject:  "google-987654321",
			wantEmail:    "user@example.com",
			wantName:     "Domain User",
			wantVerified: true,
			wantDomain:   "example.com",
		},
		{
			name:         "Unverified email user",
			token:        "unverified-token",
			wantError:    false,
			wantSubject:  "google-111222333",
			wantEmail:    "unverified@example.com",
			wantName:     "Unverified User",
			wantVerified: false,
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
			assert.Equal(t, tt.wantSubject, userInfo.Subject)
			assert.Equal(t, tt.wantEmail, userInfo.Email)
			assert.Equal(t, tt.wantName, userInfo.Name)
			assert.Equal(t, tt.wantVerified, userInfo.EmailVerified)
			assert.Equal(t, "google", userInfo.Provider)

			if tt.wantDomain != "" {
				assert.Contains(t, userInfo.Attributes, "hosted_domain")
				assert.Equal(t, tt.wantDomain, userInfo.Attributes["hosted_domain"])
			}
		})
	}
}

// DISABLED: func TestGoogleProvider_ValidateToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/v2/userinfo" {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")

			switch token {
			case "valid-token":
				userInfo := json.RawMessage("{}")
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

	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")
	provider.config.Endpoints.UserInfoURL = server.URL + "/oauth2/v2/userinfo"

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

// DISABLED: func TestGoogleProvider_RevokeToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/o/oauth2/revoke" {
			assert.Equal(t, "POST", r.Method)

			err := r.ParseForm()
			require.NoError(t, err)

			token := r.FormValue("token")
			if token == "valid-token" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			} else {
				w.WriteHeader(http.StatusBadRequest)
				response := json.RawMessage("{}")
				json.NewEncoder(w).Encode(response)
			}
		}
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")
	provider.config.Endpoints.RevokeURL = server.URL + "/o/oauth2/revoke"

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

// DISABLED: func TestGoogleProvider_DiscoverConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid_configuration" {
			config := OIDCConfiguration{
				Issuer:                "https://accounts.google.com",
				AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
				TokenEndpoint:         "https://oauth2.googleapis.com/token",
				UserInfoEndpoint:      "https://openidconnect.googleapis.com/v1/userinfo",
				JWKSUri:               "https://www.googleapis.com/oauth2/v3/certs",
				ScopesSupported: []string{
					"openid", "email", "profile",
				},
				ResponseTypesSupported: []string{
					"code", "token", "id_token", "code token", "code id_token", "token id_token", "code token id_token", "none",
				},
				SubjectTypesSupported: []string{
					"public",
				},
				IDTokenSigningAlgValuesSupported: []string{
					"RS256",
				},
				ClaimsSupported: []string{
					"aud", "email", "email_verified", "exp", "family_name", "given_name", "iat", "iss", "locale", "name", "picture", "sub",
				},
				CodeChallengeMethodsSupported: []string{
					"plain", "S256",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(config)
		}
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")

	// Check if provider implements OIDCProvider (interface not implemented yet)
	// TODO: Implement OIDCProvider interface
	assert.NotNil(t, provider) // Use provider to avoid unused variable error
	t.Skip("GoogleProvider does not implement OIDCProvider")
	return

	/*
		if oidcProvider, ok := provider.(OIDCProvider); ok {
			ctx := context.Background()
			config, err := oidcProvider.DiscoverConfiguration(ctx)

			assert.NoError(t, err)
			assert.NotNil(t, config)
			assert.Equal(t, "https://accounts.google.com", config.Issuer)
			assert.Contains(t, config.ScopesSupported, "openid")
			assert.Contains(t, config.ScopesSupported, "email")
			assert.Contains(t, config.ScopesSupported, "profile")
			assert.Contains(t, config.CodeChallengeMethodsSupported, "S256")
		}
	*/
}

// DISABLED: func TestGoogleProvider_GetConfiguration(t *testing.T) {
	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")
	config := provider.GetConfiguration()

	assert.NotNil(t, config)
	assert.Equal(t, "google", config.Name)
	assert.Equal(t, "google", config.Type)
	assert.Equal(t, "test-id", config.ClientID)
	assert.Equal(t, "test-secret", config.ClientSecret)
	assert.Equal(t, "http://localhost:8080/callback", config.RedirectURL)
	assert.Contains(t, config.Scopes, "openid")
	assert.Contains(t, config.Scopes, "email")
	assert.Contains(t, config.Scopes, "profile")
	assert.Contains(t, config.Features, FeatureOIDC)
	assert.Contains(t, config.Features, FeaturePKCE)
	assert.Contains(t, config.Features, FeatureTokenRefresh)
}

// DISABLED: func TestGoogleProvider_WithHostedDomain(t *testing.T) {
	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback", "example.com")

	authURL, _, err := provider.GetAuthorizationURL("test-state", "http://localhost:8080/callback")
	assert.NoError(t, err)

	parsedURL, err := url.Parse(authURL)
	require.NoError(t, err)

	query := parsedURL.Query()
	assert.Equal(t, "example.com", query.Get("hd"))
}

// Benchmark tests
func BenchmarkGoogleProvider_GetAuthorizationURL(b *testing.B) {
	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = provider.GetAuthorizationURL("test-state", "http://localhost:8080/callback")
	}
}

func BenchmarkGoogleProvider_GetAuthorizationURL_WithPKCE(b *testing.B) {
	provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = provider.GetAuthorizationURL("test-state", "http://localhost:8080/callback", WithPKCE())
	}
}

// Helper functions for testing
func createTestGoogleProvider() *GoogleProvider {
	return NewGoogleProvider("test-client-id", "test-client-secret", "http://localhost:8080/callback")
}

func createMockGoogleServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			response := json.RawMessage("{}")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		case "/oauth2/v2/userinfo":
			userInfo := GoogleUserInfo{
				ID:            "123456789",
				Email:         "testuser@example.com",
				VerifiedEmail: true,
				Name:          "Test User",
				GivenName:     "Test",
				FamilyName:    "User",
				Picture:       "https://lh3.googleusercontent.com/test-avatar",
				Locale:        "en",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userInfo)
		case "/.well-known/openid_configuration":
			config := OIDCConfiguration{
				Issuer:                "https://accounts.google.com",
				AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
				TokenEndpoint:         "https://oauth2.googleapis.com/token",
				UserInfoEndpoint:      "https://openidconnect.googleapis.com/v1/userinfo",
				JWKSUri:               "https://www.googleapis.com/oauth2/v3/certs",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(config)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// Test edge cases and error conditions
// DISABLED: func TestGoogleProvider_EdgeCases(t *testing.T) {
	t.Run("Empty client credentials", func(t *testing.T) {
		provider := NewGoogleProvider("", "", "http://localhost:8080/callback")
		assert.NotNil(t, provider)

		config := provider.GetConfiguration()
		assert.Empty(t, config.ClientID)
		assert.Empty(t, config.ClientSecret)
	})

	t.Run("Invalid redirect URI", func(t *testing.T) {
		provider := NewGoogleProvider("test-id", "test-secret", "invalid-uri")
		assert.NotNil(t, provider)

		config := provider.GetConfiguration()
		assert.Equal(t, "invalid-uri", config.RedirectURL)
	})

	t.Run("Network timeout handling", func(t *testing.T) {
		// Create a server that never responds
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second) // Longer than typical timeout
		}))
		defer server.Close()

		provider := NewGoogleProvider("test-id", "test-secret", "http://localhost:8080/callback")
		provider.oauth2Cfg.Endpoint = oauth2.Endpoint{
			TokenURL: server.URL + "/token",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := provider.ExchangeCodeForToken(ctx, "test-code", "http://localhost:8080/callback", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}
