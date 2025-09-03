package providers

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleProvider implements OAuth2/OIDC authentication for Google.

type GoogleProvider struct {
	config *ProviderConfig

	httpClient *http.Client

	oauth2Cfg *oauth2.Config

	oidcConfig *OIDCConfiguration

	jwksCache *JWKSCache

	hostedDomain string // Optional hosted domain restriction
}

// GoogleUserInfo represents Google user information.

type GoogleUserInfo struct {
	ID string `json:"id"`

	Email string `json:"email"`

	VerifiedEmail bool `json:"verified_email"`

	Name string `json:"name"`

	GivenName string `json:"given_name"`

	FamilyName string `json:"family_name"`

	Picture string `json:"picture"`

	Locale string `json:"locale"`

	HostedDomain string `json:"hd,omitempty"`
}

// GoogleIDToken represents Google ID token claims.

type GoogleIDToken struct {
	Issuer string `json:"iss"`

	Subject string `json:"sub"`

	Audience string `json:"aud"`

	ExpiresAt int64 `json:"exp"`

	IssuedAt int64 `json:"iat"`

	AuthTime int64 `json:"auth_time,omitempty"`

	Nonce string `json:"nonce,omitempty"`

	Name string `json:"name"`

	GivenName string `json:"given_name"`

	FamilyName string `json:"family_name"`

	Picture string `json:"picture"`

	Email string `json:"email"`

	EmailVerified bool `json:"email_verified"`

	Locale string `json:"locale"`

	HostedDomain string `json:"hd,omitempty"`

	AtHash string `json:"at_hash,omitempty"`

	jwt.RegisteredClaims
}

// JWKSCache represents a cached JWKS with expiration.

type JWKSCache struct {
	JWKS *JWKS

	ExpiresAt time.Time

	mutex sync.RWMutex
}

// NewGoogleProvider creates a new Google OAuth2/OIDC provider.

func NewGoogleProvider(clientID, clientSecret, redirectURL string, hostedDomain ...string) *GoogleProvider {
	config := &ProviderConfig{
		Name: "google",

		Type: "google",

		ClientID: clientID,

		ClientSecret: clientSecret,

		RedirectURL: redirectURL,

		Scopes: []string{"openid", "profile", "email"},

		Endpoints: ProviderEndpoints{
			AuthURL: google.Endpoint.AuthURL,

			TokenURL: google.Endpoint.TokenURL,

			UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",

			JWKSURL: "https://www.googleapis.com/oauth2/v3/certs",

			DiscoveryURL: "https://accounts.google.com/.well-known/openid_configuration",
		},

		Features: []ProviderFeature{
			FeatureOIDC,

			FeaturePKCE,

			FeatureTokenRefresh,

			FeatureTokenRevocation,

			FeatureUserInfo,

			FeatureJWTTokens,

			FeatureDiscovery,
		},
	}

	oauth2Cfg := &oauth2.Config{
		ClientID: clientID,

		ClientSecret: clientSecret,

		RedirectURL: redirectURL,

		Scopes: config.Scopes,

		Endpoint: google.Endpoint,
	}

	var hd string

	if len(hostedDomain) > 0 {
		hd = hostedDomain[0]
	}

	return &GoogleProvider{
		config: config,

		oauth2Cfg: oauth2Cfg,

		hostedDomain: hd,

		jwksCache: &JWKSCache{},

		httpClient: &http.Client{
			Timeout: 30 * time.Second,

			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		},
	}
}

// NewGoogleProviderWithDomain creates a new Google OAuth2/OIDC provider with hosted domain restriction.

func NewGoogleProviderWithDomain(clientID, clientSecret, redirectURL, hostedDomain string) *GoogleProvider {
	return NewGoogleProvider(clientID, clientSecret, redirectURL, hostedDomain)
}

// GetProviderName returns the provider name.

func (p *GoogleProvider) GetProviderName() string {
	return p.config.Name
}

// GetAuthorizationURL generates OAuth2 authorization URL with PKCE support.

func (p *GoogleProvider) GetAuthorizationURL(state, redirectURI string, options ...AuthOption) (string, *PKCEChallenge, error) {
	opts := ApplyOptions(options...)

	// Update redirect URI if provided.

	config := *p.oauth2Cfg

	if redirectURI != "" {
		config.RedirectURL = redirectURI
	}

	// Pre-allocate with capacity 1 since we know we're adding exactly one option
	authOpts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}

	var challenge *PKCEChallenge

	var err error

	if opts.UsePKCE {

		challenge, err = GeneratePKCEChallenge()
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate PKCE challenge: %w", err)
		}

		authOpts = append(authOpts,

			oauth2.SetAuthURLParam("code_challenge", challenge.CodeChallenge),

			oauth2.SetAuthURLParam("code_challenge_method", challenge.Method),
		)

	}

	// Add hosted domain restriction.

	if p.hostedDomain != "" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("hd", p.hostedDomain))
	}

	// Add custom options.

	if opts.Prompt != "" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("prompt", opts.Prompt))
	}

	if opts.LoginHint != "" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("login_hint", opts.LoginHint))
	}

	if opts.MaxAge > 0 {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("max_age", fmt.Sprintf("%d", opts.MaxAge)))
	}

	for key, value := range opts.CustomParams {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}

	authURL := config.AuthCodeURL(state, authOpts...)

	return authURL, challenge, nil
}

// ExchangeCodeForToken exchanges authorization code for access token.

func (p *GoogleProvider) ExchangeCodeForToken(ctx context.Context, code, redirectURI string, challenge *PKCEChallenge) (*TokenResponse, error) {
	config := *p.oauth2Cfg

	if redirectURI != "" {
		config.RedirectURL = redirectURI
	}

	var opts []oauth2.AuthCodeOption

	if challenge != nil {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", challenge.CodeVerifier))
	}

	token, err := config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, NewProviderError(p.GetProviderName(), "token_exchange_failed",

			"Failed to exchange authorization code for token", err)
	}

	response := &TokenResponse{
		AccessToken: token.AccessToken,

		RefreshToken: token.RefreshToken,

		TokenType: token.TokenType,

		ExpiresIn: int64(time.Until(token.Expiry).Seconds()),

		IssuedAt: time.Now(),
	}

	// Extract ID token if present.

	if idToken, ok := token.Extra("id_token").(string); ok && idToken != "" {
		response.IDToken = idToken
	}

	return response, nil
}

// RefreshToken refreshes an access token using refresh token.

func (p *GoogleProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := p.oauth2Cfg.TokenSource(ctx, token)

	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, NewProviderError(p.GetProviderName(), "token_refresh_failed",

			"Failed to refresh token", err)
	}

	response := &TokenResponse{
		AccessToken: newToken.AccessToken,

		RefreshToken: newToken.RefreshToken,

		TokenType: newToken.TokenType,

		ExpiresIn: int64(time.Until(newToken.Expiry).Seconds()),

		IssuedAt: time.Now(),
	}

	// Extract ID token if present.

	if idToken, ok := newToken.Extra("id_token").(string); ok && idToken != "" {
		response.IDToken = idToken
	}

	return response, nil
}

// GetUserInfo retrieves user information using access token.

func (p *GoogleProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.config.Endpoints.UserInfoURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError(p.GetProviderName(), "userinfo_failed",

			fmt.Sprintf("Google API returned status %d", resp.StatusCode), nil)
	}

	var googleUser GoogleUserInfo

	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Validate hosted domain if required.

	if p.hostedDomain != "" && googleUser.HostedDomain != p.hostedDomain {
		return nil, NewProviderError(p.GetProviderName(), "invalid_hosted_domain",

			fmt.Sprintf("User domain %s does not match required domain %s",

				googleUser.HostedDomain, p.hostedDomain), nil)
	}

	userInfo := &UserInfo{
		Subject: googleUser.ID,

		Email: googleUser.Email,

		EmailVerified: googleUser.VerifiedEmail,

		Name: googleUser.Name,

		GivenName: googleUser.GivenName,

		FamilyName: googleUser.FamilyName,

		Picture: googleUser.Picture,

		Locale: googleUser.Locale,

		Provider: p.GetProviderName(),

		ProviderID: googleUser.ID,

		Attributes: json.RawMessage(`{}`),
	}

	// Add domain-based groups if hosted domain is present.

	if googleUser.HostedDomain != "" {

		userInfo.Groups = []string{fmt.Sprintf("domain:%s", googleUser.HostedDomain)}

		userInfo.Organizations = []Organization{
			{
				ID: googleUser.HostedDomain,

				Name: googleUser.HostedDomain,

				DisplayName: googleUser.HostedDomain,

				Role: "member",
			},
		}

	}

	return userInfo, nil
}

// ValidateToken validates an access token.

func (p *GoogleProvider) ValidateToken(ctx context.Context, accessToken string) (*TokenValidation, error) {
	// Use Google's tokeninfo endpoint for validation.

	tokenInfoURL := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?access_token=%s", accessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", tokenInfoURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create token validation request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &TokenValidation{
			Valid: false,

			Error: err.Error(),
		}, nil
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode == http.StatusBadRequest {
		return &TokenValidation{
			Valid: false,

			Error: "Token is invalid or expired",
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &TokenValidation{
			Valid: false,

			Error: fmt.Sprintf("Unexpected status code: %d", resp.StatusCode),
		}, nil
	}

	var tokenInfo map[string]interface{}

	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return &TokenValidation{
			Valid: false,

			Error: "Failed to decode token info",
		}, nil
	}

	// Check if token is valid.

	if errMsg, exists := tokenInfo["error"]; exists {
		return &TokenValidation{
			Valid: false,

			Error: fmt.Sprintf("Token validation error: %v", errMsg),
		}, nil
	}

	validation := &TokenValidation{
		Valid: true,
	}

	// Extract token information.

	if clientID, ok := tokenInfo["aud"].(string); ok {
		validation.ClientID = clientID
	}

	if scope, ok := tokenInfo["scope"].(string); ok {
		validation.Scopes = strings.Split(scope, " ")
	}

	if expiresIn, ok := tokenInfo["expires_in"].(float64); ok {
		validation.ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	}

	return validation, nil
}

// RevokeToken revokes an access token.

func (p *GoogleProvider) RevokeToken(ctx context.Context, token string) error {
	revokeURL := "https://oauth2.googleapis.com/revoke"

	data := url.Values{}

	data.Set("token", token)

	req, err := http.NewRequestWithContext(ctx, "POST", revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode != http.StatusOK {
		return NewProviderError(p.GetProviderName(), "token_revocation_failed",

			fmt.Sprintf("Token revocation failed with status %d", resp.StatusCode), nil)
	}

	return nil
}

// SupportsFeature checks if provider supports specific features.

func (p *GoogleProvider) SupportsFeature(feature ProviderFeature) bool {
	for _, f := range p.config.Features {
		if f == feature {
			return true
		}
	}

	return false
}

// GetConfiguration returns provider configuration.

func (p *GoogleProvider) GetConfiguration() *ProviderConfig {
	return p.config
}

// OIDC Provider Implementation.

// DiscoverConfiguration discovers OIDC configuration from well-known endpoint.

func (p *GoogleProvider) DiscoverConfiguration(ctx context.Context) (*OIDCConfiguration, error) {
	if p.oidcConfig != nil {
		return p.oidcConfig, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", p.config.Endpoints.DiscoveryURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC configuration: %w", err)
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery failed with status %d", resp.StatusCode)
	}

	var config OIDCConfiguration

	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode OIDC configuration: %w", err)
	}

	p.oidcConfig = &config

	return &config, nil
}

// ValidateIDToken validates an OpenID Connect ID token.

func (p *GoogleProvider) ValidateIDToken(ctx context.Context, idToken string) (*IDTokenClaims, error) {
	// Parse token without verification first to get header.

	token, err := jwt.Parse(idToken, nil)
	if err != nil {
		// In jwt v5, we just check for parsing errors without detailed error types.

		return nil, fmt.Errorf("error parsing token: %w", err)
	}

	// Get key ID from token header.

	keyID, ok := token.Header["kid"].(string)

	if !ok {
		return nil, fmt.Errorf("token missing key ID")
	}

	// Get JWKS and find the key.

	jwks, err := p.GetJWKS(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	var signingKey *rsa.PublicKey

	for _, key := range jwks.Keys {
		if key.KeyID == keyID && key.KeyType == "RSA" {

			signingKey, err = p.parseRSAKey(&key)
			if err != nil {
				continue
			}

			break

		}
	}

	if signingKey == nil {
		return nil, fmt.Errorf("signing key not found")
	}

	// Parse and validate token.

	claims := &GoogleIDToken{}

	parsedToken, err := jwt.ParseWithClaims(idToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return signingKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate ID token: %w", err)
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("invalid ID token")
	}

	// Validate issuer.

	if claims.Issuer != "https://accounts.google.com" {
		return nil, fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}

	// Validate audience.

	if claims.Audience != p.config.ClientID {
		return nil, fmt.Errorf("invalid audience: %s", claims.Audience)
	}

	// Convert to standard IDTokenClaims.

	standardClaims := &IDTokenClaims{
		Issuer: claims.Issuer,

		Subject: claims.Subject,

		Audience: claims.Audience,

		ExpiresAt: claims.ExpiresAt,

		IssuedAt: claims.IssuedAt,

		AuthTime: claims.AuthTime,

		Nonce: claims.Nonce,

		Name: claims.Name,

		GivenName: claims.GivenName,

		FamilyName: claims.FamilyName,

		Picture: claims.Picture,

		Email: claims.Email,

		EmailVerified: claims.EmailVerified,

		Extra: json.RawMessage(`{}`),
	}

	return standardClaims, nil
}

// GetJWKS retrieves JSON Web Key Set for token validation.

func (p *GoogleProvider) GetJWKS(ctx context.Context) (*JWKS, error) {
	p.jwksCache.mutex.RLock()

	if p.jwksCache.JWKS != nil && time.Now().Before(p.jwksCache.ExpiresAt) {

		jwks := p.jwksCache.JWKS

		p.jwksCache.mutex.RUnlock()

		return jwks, nil

	}

	p.jwksCache.mutex.RUnlock()

	// Fetch JWKS.

	req, err := http.NewRequestWithContext(ctx, "GET", p.config.Endpoints.JWKSURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS request failed with status %d", resp.StatusCode)
	}

	var jwks JWKS

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Cache JWKS for 24 hours.

	p.jwksCache.mutex.Lock()

	p.jwksCache.JWKS = &jwks

	p.jwksCache.ExpiresAt = time.Now().Add(24 * time.Hour)

	p.jwksCache.mutex.Unlock()

	return &jwks, nil
}

// GetUserInfoFromIDToken extracts user info from ID token claims.

func (p *GoogleProvider) GetUserInfoFromIDToken(idToken string) (*UserInfo, error) {
	claims, err := p.ValidateIDToken(context.Background(), idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ID token: %w", err)
	}

	userInfo := &UserInfo{
		Subject: claims.Subject,

		Email: claims.Email,

		EmailVerified: claims.EmailVerified,

		Name: claims.Name,

		GivenName: claims.GivenName,

		FamilyName: claims.FamilyName,

		Picture: claims.Picture,

		Provider: p.GetProviderName(),

		ProviderID: claims.Subject,

		Attributes: claims.Extra,
	}

	// Add domain-based groups if hosted domain is present.
	var extra map[string]interface{}
	if len(claims.Extra) > 0 {
		if err := json.Unmarshal(claims.Extra, &extra); err == nil {
			if hostedDomain, ok := extra["hosted_domain"].(string); ok && hostedDomain != "" {

				userInfo.Groups = []string{fmt.Sprintf("domain:%s", hostedDomain)}

				userInfo.Organizations = []Organization{
					{
						ID: hostedDomain,

						Name: hostedDomain,

						DisplayName: hostedDomain,

						Role: "member",
					},
				}
			}
		}
	}

	return userInfo, nil
}

// parseRSAKey parses an RSA public key from JWK.

func (p *GoogleProvider) parseRSAKey(key *JWK) (*rsa.PublicKey, error) {
	// This is a simplified implementation.

	// In production, you'd want to use a proper JWK parsing library.

	return nil, fmt.Errorf("RSA key parsing not implemented in this example")
}

// Additional helper methods.

// SetHostedDomain sets or updates the hosted domain restriction.

func (p *GoogleProvider) SetHostedDomain(domain string) {
	p.hostedDomain = domain
}

// GetHostedDomain returns the current hosted domain restriction.

func (p *GoogleProvider) GetHostedDomain() string {
	return p.hostedDomain
}

// Enterprise provider methods (limited for Google).

// GetGroups retrieves user groups (limited for Google).

func (p *GoogleProvider) GetGroups(ctx context.Context, accessToken string) ([]string, error) {
	userInfo, err := p.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	return userInfo.Groups, nil
}

// GetRoles retrieves user roles (not supported by Google).

func (p *GoogleProvider) GetRoles(ctx context.Context, accessToken string) ([]string, error) {
	return []string{}, nil
}

// CheckGroupMembership checks if user belongs to specific groups.

func (p *GoogleProvider) CheckGroupMembership(ctx context.Context, accessToken string, groups []string) ([]string, error) {
	userGroups, err := p.GetGroups(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	userGroupMap := make(map[string]bool)

	for _, group := range userGroups {
		userGroupMap[group] = true
	}

	var memberGroups []string

	for _, group := range groups {
		if userGroupMap[group] {
			memberGroups = append(memberGroups, group)
		}
	}

	return memberGroups, nil
}

// ValidateUserAccess validates if user has required access level.

func (p *GoogleProvider) ValidateUserAccess(ctx context.Context, accessToken string, requiredLevel AccessLevel) error {
	userInfo, err := p.GetUserInfo(ctx, accessToken)
	if err != nil {
		return fmt.Errorf("failed to get user info for access validation: %w", err)
	}

	// Basic validation - ensure user has verified email.

	if !userInfo.EmailVerified {
		return fmt.Errorf("user email is not verified")
	}

	// Check hosted domain if required.

	if p.hostedDomain != "" {
		var attributes map[string]interface{}
		if len(userInfo.Attributes) > 0 {
			if err := json.Unmarshal(userInfo.Attributes, &attributes); err == nil {
				if hostedDomain, ok := attributes["hosted_domain"].(string); !ok || hostedDomain != p.hostedDomain {
					return fmt.Errorf("user is not from required hosted domain")
				}
			} else {
				return fmt.Errorf("user is not from required hosted domain")
			}
		} else {
			return fmt.Errorf("user is not from required hosted domain")
		}
	}

	return nil
}
