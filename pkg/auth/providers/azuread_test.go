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

func TestNewAzureADProvider(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		redirectURL  string
		tenantID     string
		wantName     string
		wantScopes   []string
		wantTenant   string
	}{
		{
			name:         "Valid Azure AD provider creation with specific tenant",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			redirectURL:  "http://localhost:8080/callback",
			tenantID:     "test-tenant-123",
			wantName:     "azuread",
			wantScopes:   []string{"openid", "email", "profile", "User.Read"},
			wantTenant:   "test-tenant-123",
		},
		{
			name:         "Valid Azure AD provider creation with common tenant",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			redirectURL:  "http://localhost:8080/callback",
			tenantID:     "common",
			wantName:     "azuread",
			wantScopes:   []string{"openid", "email", "profile", "User.Read"},
			wantTenant:   "common",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAzureADProvider(tt.clientID, tt.clientSecret, tt.redirectURL, tt.tenantID)

			assert.NotNil(t, provider)
			assert.Equal(t, tt.wantName, provider.GetProviderName())
			assert.Equal(t, tt.wantTenant, provider.tenantID)
			
			config := provider.GetConfiguration()
			assert.Equal(t, tt.clientID, config.ClientID)
			assert.Equal(t, tt.clientSecret, config.ClientSecret)
			assert.Equal(t, tt.redirectURL, config.RedirectURL)
			assert.Equal(t, tt.wantScopes, config.Scopes)
		})
	}
}

func TestAzureADProvider_GetProviderName(t *testing.T) {
	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "common")
	assert.Equal(t, "azuread", provider.GetProviderName())
}

func TestAzureADProvider_SupportsFeature(t *testing.T) {
	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "common")

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
		{FeatureGroups, true},
		{FeatureRoles, true},
		{FeatureOrganizations, false}, // Azure AD uses tenants, not organizations
	}

	for _, tt := range tests {
		t.Run(string(tt.feature), func(t *testing.T) {
			result := provider.SupportsFeature(tt.feature)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAzureADProvider_GetAuthorizationURL(t *testing.T) {
	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")

	tests := []struct {
		name        string
		state       string
		redirectURI string
		options     []AuthOption
		wantState   string
		wantError   bool
		checkPKCE   bool
		checkPrompt bool
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
			name:        "Authorization URL with prompt",
			state:       "test-state-789",
			redirectURI: "http://localhost:8080/callback",
			options:     []AuthOption{WithPrompt("consent")},
			wantState:   "test-state-789",
			wantError:   false,
			checkPrompt: true,
		},
		{
			name:        "Authorization URL with login hint",
			state:       "test-state-hint",
			redirectURI: "http://localhost:8080/callback",
			options:     []AuthOption{WithLoginHint("user@example.com")},
			wantState:   "test-state-hint",
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
			expectedScopes := []string{"openid", "email", "profile", "User.Read"}
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

			// Check prompt parameter
			if tt.checkPrompt {
				assert.Equal(t, "consent", query.Get("prompt"))
			}
		})
	}
}

func TestAzureADProvider_ExchangeCodeForToken(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/oauth2/v2.0/token") {
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
					"access_token":  "azure-access-token-123",
					"refresh_token": "azure-refresh-token-456",
					"token_type":    "Bearer",
					"expires_in":    3600,
					"id_token":      "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.test-azure-id-token",
					"scope":         "openid email profile User.Read",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			case "invalid-code":
				w.WriteHeader(http.StatusBadRequest)
				response := map[string]interface{}{
					"error":             "invalid_grant",
					"error_description": "AADSTS70002: Error validating credentials. AADSTS54005: OAuth2 Authorization code was already redeemed",
					"error_codes":       []int{70002, 54005},
					"timestamp":         "2023-01-01 12:00:00Z",
					"trace_id":          "test-trace-id",
					"correlation_id":    "test-correlation-id",
				}
				json.NewEncoder(w).Encode(response)
			default:
				w.WriteHeader(http.StatusBadRequest)
				response := map[string]interface{}{
					"error":             "invalid_request",
					"error_description": "Invalid request",
				}
				json.NewEncoder(w).Encode(response)
			}
		}
	}))
	defer server.Close()

	// Create provider with custom endpoint
	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")
	provider.oauth2Cfg.Endpoint = oauth2.Endpoint{
		TokenURL: server.URL + "/test-tenant/oauth2/v2.0/token",
	}

	tests := []struct {
		name         string
		code         string
		wantError    bool
		wantToken    string
		wantRefresh  string
		wantIDToken  bool
	}{
		{
			name:        "Valid code exchange",
			code:        "valid-code",
			wantError:   false,
			wantToken:   "azure-access-token-123",
			wantRefresh: "azure-refresh-token-456",
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

func TestAzureADProvider_RefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/oauth2/v2.0/token") {
			err := r.ParseForm()
			require.NoError(t, err)

			grantType := r.FormValue("grant_type")
			assert.Equal(t, "refresh_token", grantType)

			refreshToken := r.FormValue("refresh_token")
			switch refreshToken {
			case "valid-refresh-token":
				response := map[string]interface{}{
					"access_token":  "new-azure-access-token",
					"refresh_token": "new-azure-refresh-token",
					"token_type":    "Bearer",
					"expires_in":    3600,
					"scope":         "openid email profile User.Read",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			case "expired-refresh-token":
				w.WriteHeader(http.StatusBadRequest)
				response := map[string]interface{}{
					"error":             "invalid_grant",
					"error_description": "AADSTS70008: The provided authorization code or refresh token has expired due to inactivity",
					"error_codes":       []int{70008},
				}
				json.NewEncoder(w).Encode(response)
			default:
				w.WriteHeader(http.StatusBadRequest)
				response := map[string]interface{}{
					"error":             "invalid_grant",
					"error_description": "Invalid refresh token",
				}
				json.NewEncoder(w).Encode(response)
			}
		}
	}))
	defer server.Close()

	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")
	provider.oauth2Cfg.Endpoint = oauth2.Endpoint{
		TokenURL: server.URL + "/test-tenant/oauth2/v2.0/token",
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
			wantToken:    "new-azure-access-token",
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

func TestAzureADProvider_GetUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1.0/me" {
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
				userInfo := AzureADUserInfo{
					ID:                "12345678-1234-1234-1234-123456789012",
					UserPrincipalName: "testuser@example.com",
					DisplayName:       "Test User",
					GivenName:         "Test",
					Surname:           "User",
					Mail:              "testuser@example.com",
					MailNickname:      "testuser",
					JobTitle:          "Software Engineer",
					Department:        "Engineering",
					CompanyName:       "Example Corp",
					OfficeLocation:    "Building 1",
					PreferredLanguage: "en-US",
					AccountEnabled:    true,
					UserType:          "Member",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userInfo)
			case "external-user-token":
				userInfo := AzureADUserInfo{
					ID:                "87654321-4321-4321-4321-210987654321",
					UserPrincipalName: "external@guest.com",
					DisplayName:       "External User",
					GivenName:         "External",
					Surname:           "User",
					Mail:              "external@guest.com",
					AccountEnabled:    true,
					UserType:          "Guest",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userInfo)
			case "disabled-user-token":
				userInfo := AzureADUserInfo{
					ID:                "disabled123-1234-1234-1234-123456789012",
					UserPrincipalName: "disabled@example.com",
					DisplayName:       "Disabled User",
					AccountEnabled:    false,
					UserType:          "Member",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(userInfo)
			case "invalid-token":
				w.WriteHeader(http.StatusUnauthorized)
				response := map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "InvalidAuthenticationToken",
						"message": "Access token is empty.",
					},
				}
				json.NewEncoder(w).Encode(response)
			default:
				w.WriteHeader(http.StatusForbidden)
				response := map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "Forbidden",
						"message": "Insufficient privileges to complete the operation.",
					},
				}
				json.NewEncoder(w).Encode(response)
			}
		}
	}))
	defer server.Close()

	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")
	provider.config.Endpoints.UserInfoURL = server.URL + "/v1.0/me"

	tests := []struct {
		name         string
		token        string
		wantError    bool
		wantSubject  string
		wantEmail    string
		wantName     string
		wantUserType string
		wantEnabled  *bool
	}{
		{
			name:         "Valid user info",
			token:        "valid-token",
			wantError:    false,
			wantSubject:  "azuread-12345678-1234-1234-1234-123456789012",
			wantEmail:    "testuser@example.com",
			wantName:     "Test User",
			wantUserType: "Member",
			wantEnabled:  &[]bool{true}[0],
		},
		{
			name:         "External user",
			token:        "external-user-token",
			wantError:    false,
			wantSubject:  "azuread-87654321-4321-4321-4321-210987654321",
			wantEmail:    "external@guest.com",
			wantName:     "External User",
			wantUserType: "Guest",
			wantEnabled:  &[]bool{true}[0],
		},
		{
			name:         "Disabled user",
			token:        "disabled-user-token",
			wantError:    false,
			wantSubject:  "azuread-disabled123-1234-1234-1234-123456789012",
			wantName:     "Disabled User",
			wantUserType: "Member",
			wantEnabled:  &[]bool{false}[0],
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
			assert.Equal(t, "azuread", userInfo.Provider)
			
			if tt.wantUserType != "" {
				assert.Contains(t, userInfo.Attributes, "user_type")
				assert.Equal(t, tt.wantUserType, userInfo.Attributes["user_type"])
			}
			
			if tt.wantEnabled != nil {
				assert.Contains(t, userInfo.Attributes, "account_enabled")
				assert.Equal(t, *tt.wantEnabled, userInfo.Attributes["account_enabled"])
			}
		})
	}
}

func TestAzureADProvider_GetGroups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1.0/me/memberOf" {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")
			
			switch token {
			case "valid-token":
				groups := map[string]interface{}{
					"value": []map[string]interface{}{
						{
							"id":          "group1-1234-5678-9012-123456789012",
							"displayName": "Engineering Team",
							"description": "Engineering team group",
						},
						{
							"id":          "group2-1234-5678-9012-123456789012",
							"displayName": "Admin Users",
							"description": "Administrative users group",
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(groups)
			case "no-groups-token":
				groups := map[string]interface{}{
					"value": []map[string]interface{}{},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(groups)
			default:
				w.WriteHeader(http.StatusUnauthorized)
			}
		}
	}))
	defer server.Close()

	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")

	tests := []struct {
		name           string
		token          string
		wantError      bool
		wantGroupCount int
		wantFirstGroup string
	}{
		{
			name:           "Valid groups",
			token:          "valid-token",
			wantError:      false,
			wantGroupCount: 2,
			wantFirstGroup: "Engineering Team",
		},
		{
			name:           "No groups",
			token:          "no-groups-token",
			wantError:      false,
			wantGroupCount: 0,
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
			
			// Check if provider implements EnterpriseProvider
			if ep, ok := provider.(EnterpriseProvider); ok {
				groups, err := ep.GetGroups(ctx, tt.token)

				if tt.wantError {
					assert.Error(t, err)
					return
				}

				assert.NoError(t, err)
				assert.Len(t, groups, tt.wantGroupCount)
				
				if tt.wantGroupCount > 0 {
					assert.Equal(t, tt.wantFirstGroup, groups[0])
				}
			} else {
				t.Skip("AzureADProvider does not implement EnterpriseProvider")
			}
		})
	}
}

func TestAzureADProvider_ValidateToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1.0/me" {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authHeader, "Bearer ")
			
			switch token {
			case "valid-token":
				userInfo := map[string]interface{}{
					"id":                "123456789",
					"userPrincipalName": "testuser@example.com",
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

	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")
	provider.config.Endpoints.UserInfoURL = server.URL + "/v1.0/me"

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

func TestAzureADProvider_RevokeToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/oauth2/v2.0/logout") {
			assert.Equal(t, "POST", r.Method)
			
			err := r.ParseForm()
			require.NoError(t, err)
			
			token := r.FormValue("token")
			if token == "valid-token" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusBadRequest)
				response := map[string]interface{}{
					"error":             "invalid_request",
					"error_description": "Invalid token",
				}
				json.NewEncoder(w).Encode(response)
			}
		}
	}))
	defer server.Close()

	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")

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

func TestAzureADProvider_DiscoverConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/.well-known/openid_configuration") {
			config := OIDCConfiguration{
				Issuer:                "https://login.microsoftonline.com/test-tenant/v2.0",
				AuthorizationEndpoint: "https://login.microsoftonline.com/test-tenant/oauth2/v2.0/authorize",
				TokenEndpoint:         "https://login.microsoftonline.com/test-tenant/oauth2/v2.0/token",
				JWKSUri:              "https://login.microsoftonline.com/test-tenant/discovery/v2.0/keys",
				ScopesSupported: []string{
					"openid", "email", "profile", "offline_access",
				},
				ResponseTypesSupported: []string{
					"code", "id_token", "code id_token", "id_token token",
				},
				SubjectTypesSupported: []string{
					"pairwise",
				},
				IDTokenSigningAlgValuesSupported: []string{
					"RS256",
				},
				ClaimsSupported: []string{
					"aud", "iss", "iat", "exp", "name", "given_name", "family_name", "email", "sub",
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

	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")

	// Check if provider implements OIDCProvider
	if oidcProvider, ok := provider.(OIDCProvider); ok {
		ctx := context.Background()
		config, err := oidcProvider.DiscoverConfiguration(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "https://login.microsoftonline.com/test-tenant/v2.0", config.Issuer)
		assert.Contains(t, config.ScopesSupported, "openid")
		assert.Contains(t, config.ScopesSupported, "email")
		assert.Contains(t, config.ScopesSupported, "profile")
		assert.Contains(t, config.CodeChallengeMethodsSupported, "S256")
	} else {
		t.Skip("AzureADProvider does not implement OIDCProvider")
	}
}

func TestAzureADProvider_GetConfiguration(t *testing.T) {
	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")
	config := provider.GetConfiguration()

	assert.NotNil(t, config)
	assert.Equal(t, "azuread", config.Name)
	assert.Equal(t, "azuread", config.Type)
	assert.Equal(t, "test-id", config.ClientID)
	assert.Equal(t, "test-secret", config.ClientSecret)
	assert.Equal(t, "http://localhost:8080/callback", config.RedirectURL)
	assert.Contains(t, config.Scopes, "openid")
	assert.Contains(t, config.Scopes, "email")
	assert.Contains(t, config.Scopes, "profile")
	assert.Contains(t, config.Scopes, "User.Read")
	assert.Contains(t, config.Features, FeatureOIDC)
	assert.Contains(t, config.Features, FeaturePKCE)
	assert.Contains(t, config.Features, FeatureTokenRefresh)
	assert.Contains(t, config.Features, FeatureGroups)
}

func TestAzureADProvider_MultiTenant(t *testing.T) {
	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "common")

	assert.True(t, provider.isMultiTenant)
	assert.Equal(t, "common", provider.tenantID)

	authURL, _, err := provider.GetAuthorizationURL("test-state", "http://localhost:8080/callback")
	assert.NoError(t, err)
	assert.Contains(t, authURL, "/common/oauth2")
}

func TestAzureADProvider_TenantSpecific(t *testing.T) {
	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "specific-tenant-123")

	assert.False(t, provider.isMultiTenant)
	assert.Equal(t, "specific-tenant-123", provider.tenantID)

	authURL, _, err := provider.GetAuthorizationURL("test-state", "http://localhost:8080/callback")
	assert.NoError(t, err)
	assert.Contains(t, authURL, "/specific-tenant-123/oauth2")
}

// Benchmark tests
func BenchmarkAzureADProvider_GetAuthorizationURL(b *testing.B) {
	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = provider.GetAuthorizationURL("test-state", "http://localhost:8080/callback")
	}
}

func BenchmarkAzureADProvider_GetAuthorizationURL_WithPKCE(b *testing.B) {
	provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = provider.GetAuthorizationURL("test-state", "http://localhost:8080/callback", WithPKCE())
	}
}

// Helper functions for testing
func createTestAzureADProvider() *AzureADProvider {
	return NewAzureADProvider("test-client-id", "test-client-secret", "http://localhost:8080/callback", "test-tenant")
}

func createMockAzureADServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test-tenant/oauth2/v2.0/token":
			response := map[string]interface{}{
				"access_token":  "test-access-token",
				"refresh_token": "test-refresh-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"id_token":      "test-id-token",
				"scope":         "openid email profile User.Read",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		case "/v1.0/me":
			userInfo := AzureADUserInfo{
				ID:                "12345678-1234-1234-1234-123456789012",
				UserPrincipalName: "testuser@example.com",
				DisplayName:       "Test User",
				GivenName:         "Test",
				Surname:           "User",
				Mail:              "testuser@example.com",
				AccountEnabled:    true,
				UserType:          "Member",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userInfo)
		case "/test-tenant/v2.0/.well-known/openid_configuration":
			config := OIDCConfiguration{
				Issuer:                "https://login.microsoftonline.com/test-tenant/v2.0",
				AuthorizationEndpoint: "https://login.microsoftonline.com/test-tenant/oauth2/v2.0/authorize",
				TokenEndpoint:         "https://login.microsoftonline.com/test-tenant/oauth2/v2.0/token",
				JWKSUri:              "https://login.microsoftonline.com/test-tenant/discovery/v2.0/keys",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(config)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// Test edge cases and error conditions
func TestAzureADProvider_EdgeCases(t *testing.T) {
	t.Run("Empty tenant ID defaults to common", func(t *testing.T) {
		provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "")
		assert.Equal(t, "common", provider.tenantID)
		assert.True(t, provider.isMultiTenant)
	})

	t.Run("Invalid client credentials", func(t *testing.T) {
		provider := NewAzureADProvider("", "", "http://localhost:8080/callback", "test-tenant")
		assert.NotNil(t, provider)
		
		config := provider.GetConfiguration()
		assert.Empty(t, config.ClientID)
		assert.Empty(t, config.ClientSecret)
	})

	t.Run("Network timeout handling", func(t *testing.T) {
		// Create a server that never responds
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second) // Longer than typical timeout
		}))
		defer server.Close()

		provider := NewAzureADProvider("test-id", "test-secret", "http://localhost:8080/callback", "test-tenant")
		provider.oauth2Cfg.Endpoint = oauth2.Endpoint{
			TokenURL: server.URL + "/test-tenant/oauth2/v2.0/token",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := provider.ExchangeCodeForToken(ctx, "test-code", "http://localhost:8080/callback", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}