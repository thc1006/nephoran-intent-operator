package auth

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// OAuth2Provider represents an enterprise OAuth2 identity provider
type OAuth2Provider struct {
	Name         string
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
	TLSConfig    *tls.Config
	httpClient   *http.Client
}

// OAuth2Config holds OAuth2 configuration for multiple providers
type OAuth2Config struct {
	Providers     map[string]*OAuth2Provider `json:"providers"`
	DefaultScopes []string                   `json:"default_scopes"`
	TokenTTL      time.Duration              `json:"token_ttl"`
	RefreshTTL    time.Duration              `json:"refresh_ttl"`
}

// TokenResponse represents an OAuth2 token response
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	Scope        string    `json:"scope,omitempty"`
	IssuedAt     time.Time `json:"issued_at"`
}

// UserInfo represents user information from the identity provider
type UserInfo struct {
	Subject       string   `json:"sub"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Name          string   `json:"name"`
	PreferredName string   `json:"preferred_username"`
	Groups        []string `json:"groups"`
	Roles         []string `json:"roles"`
}

// AuthenticationResult contains the result of OAuth2 authentication
type AuthenticationResult struct {
	Token    *TokenResponse `json:"token"`
	UserInfo *UserInfo      `json:"user_info"`
	Provider string         `json:"provider"`
}

// NewOAuth2Provider creates a new OAuth2 provider instance
func NewOAuth2Provider(name, clientID, clientSecret, authURL, tokenURL, userInfoURL string, scopes []string) *OAuth2Provider {
	return &OAuth2Provider{
		Name:         name,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      authURL,
		TokenURL:     tokenURL,
		UserInfoURL:  userInfoURL,
		Scopes:       scopes,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
				},
			},
		},
	}
}

// GetAuthorizationURL generates the OAuth2 authorization URL
func (p *OAuth2Provider) GetAuthorizationURL(state, redirectURI string) string {
	config := p.getOAuth2Config(redirectURI)
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCodeForToken exchanges authorization code for access token
func (p *OAuth2Provider) ExchangeCodeForToken(ctx context.Context, code, redirectURI string) (*TokenResponse, error) {
	config := p.getOAuth2Config(redirectURI)
	
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresIn:    int64(token.Expiry.Sub(time.Now()).Seconds()),
		IssuedAt:     time.Now(),
	}, nil
}

// ValidateToken validates an OAuth2 access token
func (p *OAuth2Provider) ValidateToken(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status: %d", resp.StatusCode)
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// RefreshToken refreshes an OAuth2 access token using refresh token
func (p *OAuth2Provider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", p.ClientID)
	data.Set("client_secret", p.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", p.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status: %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	tokenResp.IssuedAt = time.Now()
	return &tokenResp, nil
}

// GetClientCredentialsToken gets a token using client credentials flow
func (p *OAuth2Provider) GetClientCredentialsToken(ctx context.Context) (*TokenResponse, error) {
	config := clientcredentials.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		TokenURL:     p.TokenURL,
		Scopes:       p.Scopes,
	}

	token, err := config.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client credentials token: %w", err)
	}

	return &TokenResponse{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		ExpiresIn:   int64(token.Expiry.Sub(time.Now()).Seconds()),
		IssuedAt:    time.Now(),
	}, nil
}

// getOAuth2Config creates OAuth2 config for this provider
func (p *OAuth2Provider) getOAuth2Config(redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		RedirectURL:  redirectURI,
		Scopes:       p.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  p.AuthURL,
			TokenURL: p.TokenURL,
		},
	}
}

// Enterprise Identity Provider Configurations

// NewAzureADProvider creates an Azure AD OAuth2 provider
func NewAzureADProvider(tenantID, clientID, clientSecret string) *OAuth2Provider {
	return NewOAuth2Provider(
		"azure-ad",
		clientID,
		clientSecret,
		fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID),
		fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID),
		"https://graph.microsoft.com/v1.0/me",
		[]string{"openid", "profile", "email", "User.Read"},
	)
}

// NewOktaProvider creates an Okta OAuth2 provider
func NewOktaProvider(domain, clientID, clientSecret string) *OAuth2Provider {
	return NewOAuth2Provider(
		"okta",
		clientID,
		clientSecret,
		fmt.Sprintf("https://%s/oauth2/default/v1/authorize", domain),
		fmt.Sprintf("https://%s/oauth2/default/v1/token", domain),
		fmt.Sprintf("https://%s/oauth2/default/v1/userinfo", domain),
		[]string{"openid", "profile", "email", "groups"},
	)
}

// NewKeycloakProvider creates a Keycloak OAuth2 provider
func NewKeycloakProvider(baseURL, realm, clientID, clientSecret string) *OAuth2Provider {
	return NewOAuth2Provider(
		"keycloak",
		clientID,
		clientSecret,
		fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/auth", baseURL, realm),
		fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/token", baseURL, realm),
		fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/userinfo", baseURL, realm),
		[]string{"openid", "profile", "email", "roles"},
	)
}

// NewGoogleProvider creates a Google OAuth2 provider
func NewGoogleProvider(clientID, clientSecret string) *OAuth2Provider {
	return NewOAuth2Provider(
		"google",
		clientID,
		clientSecret,
		"https://accounts.google.com/o/oauth2/auth",
		"https://oauth2.googleapis.com/token",
		"https://www.googleapis.com/oauth2/v2/userinfo",
		[]string{"openid", "profile", "email"},
	)
}

// JWT Token Management

// JWTClaims represents custom JWT claims
type JWTClaims struct {
	jwt.RegisteredClaims
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	Groups   []string `json:"groups"`
	Roles    []string `json:"roles"`
	Provider string   `json:"provider"`
}

// GenerateJWT generates a JWT token for authenticated user
func GenerateJWT(userInfo *UserInfo, provider string, secretKey []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userInfo.Subject,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "nephoran-intent-operator",
		},
		Email:    userInfo.Email,
		Name:     userInfo.Name,
		Groups:   userInfo.Groups,
		Roles:    userInfo.Roles,
		Provider: provider,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ValidateJWT validates and parses a JWT token
func ValidateJWT(tokenString string, secretKey []byte) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid JWT token")
}