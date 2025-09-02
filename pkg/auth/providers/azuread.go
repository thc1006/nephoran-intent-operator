package providers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// AzureADProvider implements OAuth2/OIDC authentication for Microsoft Azure AD.

type AzureADProvider struct {
	config *ProviderConfig

	httpClient *http.Client

	oauth2Cfg *oauth2.Config

	oidcConfig *OIDCConfiguration

	jwksCache *JWKSCache

	tenantID string

	isMultiTenant bool
}

// AzureADUserInfo represents Azure AD user information.

type AzureADUserInfo struct {
	ID string `json:"id"`

	UserPrincipalName string `json:"userPrincipalName"`

	DisplayName string `json:"displayName"`

	GivenName string `json:"givenName"`

	Surname string `json:"surname"`

	Mail string `json:"mail"`

	MailNickname string `json:"mailNickname"`

	JobTitle string `json:"jobTitle"`

	Department string `json:"department"`

	CompanyName string `json:"companyName"`

	OfficeLocation string `json:"officeLocation"`

	PreferredLanguage string `json:"preferredLanguage"`

	AccountEnabled bool `json:"accountEnabled"`

	UserType string `json:"userType"`

	OnPremisesSecurityIdentifier string `json:"onPremisesSecurityIdentifier"`
}

// AzureADGroup represents an Azure AD group.

type AzureADGroup struct {
	ID string `json:"id"`

	DisplayName string `json:"displayName"`

	Description string `json:"description"`

	GroupTypes []string `json:"groupTypes"`

	SecurityEnabled bool `json:"securityEnabled"`

	MailEnabled bool `json:"mailEnabled"`

	Mail string `json:"mail"`
}

// AzureADDirectoryRole represents an Azure AD directory role.

type AzureADDirectoryRole struct {
	ID string `json:"id"`

	DisplayName string `json:"displayName"`

	Description string `json:"description"`

	RoleTemplateID string `json:"roleTemplateId"`
}

// AzureADApplication represents an Azure AD application.

type AzureADApplication struct {
	ID string `json:"id"`

	DisplayName string `json:"displayName"`

	AppID string `json:"appId"`

	AppRoles []AzureADAppRole `json:"appRoles"`
}

// AzureADAppRole represents an application role.

type AzureADAppRole struct {
	ID string `json:"id"`

	DisplayName string `json:"displayName"`

	Description string `json:"description"`

	Value string `json:"value"`

	AllowedMemberTypes []string `json:"allowedMemberTypes"`

	IsEnabled bool `json:"isEnabled"`
}

// AzureADIDToken represents Azure AD ID token claims.

type AzureADIDToken struct {
	Issuer string `json:"iss"`

	Subject string `json:"sub"`

	Audience string `json:"aud"`

	ExpiresAt int64 `json:"exp"`

	IssuedAt int64 `json:"iat"`

	NotBefore int64 `json:"nbf"`

	AuthTime int64 `json:"auth_time,omitempty"`

	Nonce string `json:"nonce,omitempty"`

	Name string `json:"name"`

	PreferredUsername string `json:"preferred_username"`

	Email string `json:"email"`

	Groups []string `json:"groups,omitempty"`

	Roles []string `json:"roles,omitempty"`

	TenantID string `json:"tid"`

	ObjectID string `json:"oid"`

	Version string `json:"ver"`

	AtHash string `json:"at_hash,omitempty"`
}

// NewAzureADProvider creates a new Azure AD OAuth2/OIDC provider.

func NewAzureADProvider(tenantID, clientID, clientSecret, redirectURL string) *AzureADProvider {
	isMultiTenant := tenantID == "common" || tenantID == "organizations" || tenantID == "consumers"

	authURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID)

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	discoveryURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0/.well-known/openid_configuration", tenantID)

	config := &ProviderConfig{
		Name: "azuread",

		Type: "azuread",

		ClientID: clientID,

		ClientSecret: clientSecret,

		RedirectURL: redirectURL,

		Scopes: []string{"openid", "profile", "email", "User.Read", "Directory.Read.All"},

		Endpoints: ProviderEndpoints{
			AuthURL: authURL,

			TokenURL: tokenURL,

			UserInfoURL: "https://graph.microsoft.com/v1.0/me",

			JWKSURL: fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", tenantID),

			DiscoveryURL: discoveryURL,
		},

		Features: []ProviderFeature{
			FeatureOIDC,

			FeaturePKCE,

			FeatureTokenRefresh,

			FeatureTokenRevocation,

			FeatureUserInfo,

			FeatureGroups,

			FeatureRoles,

			FeatureOrganizations,

			FeatureJWTTokens,

			FeatureDiscovery,
		},
	}

	oauth2Cfg := &oauth2.Config{
		ClientID: clientID,

		ClientSecret: clientSecret,

		RedirectURL: redirectURL,

		Scopes: config.Scopes,

		Endpoint: oauth2.Endpoint{
			AuthURL: authURL,

			TokenURL: tokenURL,
		},
	}

	return &AzureADProvider{
		config: config,

		oauth2Cfg: oauth2Cfg,

		tenantID: tenantID,

		isMultiTenant: isMultiTenant,

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

// GetProviderName returns the provider name.

func (p *AzureADProvider) GetProviderName() string {
	return p.config.Name
}

// GetAuthorizationURL generates OAuth2 authorization URL with PKCE support.

func (p *AzureADProvider) GetAuthorizationURL(state, redirectURI string, options ...AuthOption) (string, *PKCEChallenge, error) {
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

	// Add Azure AD specific options.

	if opts.Prompt != "" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("prompt", opts.Prompt))
	}

	if opts.LoginHint != "" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("login_hint", opts.LoginHint))
	}

	if opts.DomainHint != "" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("domain_hint", opts.DomainHint))
	}

	if opts.MaxAge > 0 {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("max_age", fmt.Sprintf("%d", opts.MaxAge)))
	}

	// Add response_mode=query for better compatibility.

	authOpts = append(authOpts, oauth2.SetAuthURLParam("response_mode", "query"))

	for key, value := range opts.CustomParams {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}

	authURL := config.AuthCodeURL(state, authOpts...)

	return authURL, challenge, nil
}

// ExchangeCodeForToken exchanges authorization code for access token.

func (p *AzureADProvider) ExchangeCodeForToken(ctx context.Context, code, redirectURI string, challenge *PKCEChallenge) (*TokenResponse, error) {
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

func (p *AzureADProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
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

func (p *AzureADProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
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

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError(p.GetProviderName(), "userinfo_failed",

			fmt.Sprintf("Microsoft Graph API returned status %d", resp.StatusCode), nil)
	}

	var azureUser AzureADUserInfo

	if err := json.NewDecoder(resp.Body).Decode(&azureUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Get user groups.

	groups, err := p.GetGroups(ctx, accessToken)
	if err != nil {
		// Log error but don't fail the request.

		groups = []string{}
	}

	// Get user roles.

	roles, err := p.GetRoles(ctx, accessToken)
	if err != nil {
		// Log error but don't fail the request.

		roles = []string{}
	}

	userInfo := &UserInfo{
		Subject: azureUser.ID,

		Email: azureUser.Mail,

		EmailVerified: azureUser.Mail != "",

		Name: azureUser.DisplayName,

		GivenName: azureUser.GivenName,

		FamilyName: azureUser.Surname,

		PreferredName: azureUser.UserPrincipalName,

		Username: azureUser.MailNickname,

		Groups: groups,

		Roles: roles,

		Provider: p.GetProviderName(),

		ProviderID: azureUser.ID,

		Attributes: json.RawMessage("{}"),
	}

	// Add tenant as organization if not multi-tenant.

	if !p.isMultiTenant {
		userInfo.Organizations = []Organization{
			{
				ID: p.tenantID,

				Name: p.tenantID,

				DisplayName: azureUser.CompanyName,

				Role: "member",
			},
		}
	}

	return userInfo, nil
}

// ValidateToken validates an access token.

func (p *AzureADProvider) ValidateToken(ctx context.Context, accessToken string) (*TokenValidation, error) {
	// Use Microsoft Graph /me endpoint for validation.

	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/me", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create token validation request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &TokenValidation{
			Valid: false,

			Error: err.Error(),
		}, nil
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
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

	var azureUser AzureADUserInfo

	if err := json.NewDecoder(resp.Body).Decode(&azureUser); err != nil {
		return &TokenValidation{
			Valid: false,

			Error: "Failed to decode user info",
		}, nil
	}

	return &TokenValidation{
		Valid: true,

		Username: azureUser.UserPrincipalName,

		ClientID: p.config.ClientID,

		// Azure AD access tokens typically have 1 hour expiry.

		ExpiresAt: time.Now().Add(time.Hour),
	}, nil
}

// RevokeToken revokes an access token.

func (p *AzureADProvider) RevokeToken(ctx context.Context, token string) error {
	// Azure AD doesn't have a standard token revocation endpoint.

	// Tokens are revoked when the user signs out or the session expires.

	return NewProviderError(p.GetProviderName(), "not_supported",

		"Azure AD does not support programmatic token revocation", nil)
}

// SupportsFeature checks if provider supports specific features.

func (p *AzureADProvider) SupportsFeature(feature ProviderFeature) bool {
	for _, f := range p.config.Features {
		if f == feature {
			return true
		}
	}

	return false
}

// GetConfiguration returns provider configuration.

func (p *AzureADProvider) GetConfiguration() *ProviderConfig {
	return p.config
}

// Enterprise Provider Implementation.

// GetGroups retrieves user groups from Azure AD.

func (p *AzureADProvider) GetGroups(ctx context.Context, accessToken string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/me/memberOf", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create groups request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("groups request failed with status %d", resp.StatusCode)
	}

	var response struct {
		Value []AzureADGroup `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode groups: %w", err)
	}

	groups := make([]string, len(response.Value))

	for i, group := range response.Value {
		groups[i] = group.DisplayName
	}

	return groups, nil
}

// GetRoles retrieves user roles from Azure AD.

func (p *AzureADProvider) GetRoles(ctx context.Context, accessToken string) ([]string, error) {
	// Get directory roles.

	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/me/memberOf/microsoft.graph.directoryRole", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create roles request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{}, nil // Directory roles might not be accessible
	}

	var response struct {
		Value []AzureADDirectoryRole `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return []string{}, nil // Continue without roles if decoding fails
	}

	roles := make([]string, len(response.Value))

	for i, role := range response.Value {
		roles[i] = role.DisplayName
	}

	return roles, nil
}

// CheckGroupMembership checks if user belongs to specific groups.

func (p *AzureADProvider) CheckGroupMembership(ctx context.Context, accessToken string, groups []string) ([]string, error) {
	userGroups, err := p.GetGroups(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	userGroupMap := make(map[string]bool)

	for _, group := range userGroups {

		userGroupMap[group] = true

		userGroupMap[strings.ToLower(group)] = true

	}

	var memberGroups []string

	for _, group := range groups {
		if userGroupMap[group] || userGroupMap[strings.ToLower(group)] {
			memberGroups = append(memberGroups, group)
		}
	}

	return memberGroups, nil
}

// GetOrganizations retrieves user's organizations (tenant information).

func (p *AzureADProvider) GetOrganizations(ctx context.Context, accessToken string) ([]Organization, error) {
	if p.isMultiTenant {
		// For multi-tenant apps, we'd need to get tenant info from the token.

		return []Organization{}, nil
	}

	// Get organization info.

	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/organization", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []Organization{}, nil // Organization info might not be accessible
	}

	var response struct {
		Value []struct {
			ID string `json:"id"`

			DisplayName string `json:"displayName"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return []Organization{}, nil
	}

	organizations := make([]Organization, len(response.Value))

	for i, org := range response.Value {
		organizations[i] = Organization{
			ID: org.ID,

			Name: org.DisplayName,

			DisplayName: org.DisplayName,

			Role: "member",
		}
	}

	return organizations, nil
}

// ValidateUserAccess validates if user has required access level.

func (p *AzureADProvider) ValidateUserAccess(ctx context.Context, accessToken string, requiredLevel AccessLevel) error {
	userInfo, err := p.GetUserInfo(ctx, accessToken)
	if err != nil {
		return fmt.Errorf("failed to get user info for access validation: %w", err)
	}

	// Check if account is enabled.
	var attributes map[string]interface{}
	if len(userInfo.Attributes) > 0 {
		if err := json.Unmarshal(userInfo.Attributes, &attributes); err == nil {
			if accountEnabled, ok := attributes["account_enabled"].(bool); ok && !accountEnabled {
				return fmt.Errorf("user account is disabled")
			}
		}
	}

	// For admin levels, check if user has any admin roles.

	if requiredLevel >= AccessLevelAdmin {

		roles, err := p.GetRoles(ctx, accessToken)
		if err != nil {
			return fmt.Errorf("failed to get user roles for access validation: %w", err)
		}

		hasAdminRole := false

		adminRoles := []string{"Global Administrator", "User Administrator", "Application Administrator", "Cloud Application Administrator"}

		for _, role := range roles {

			for _, adminRole := range adminRoles {
				if strings.EqualFold(role, adminRole) {

					hasAdminRole = true

					break

				}
			}

			if hasAdminRole {
				break
			}

		}

		if !hasAdminRole {
			return fmt.Errorf("user does not have required administrative privileges")
		}

	}

	return nil
}

// OIDC Provider Implementation.

// DiscoverConfiguration discovers OIDC configuration from well-known endpoint.

func (p *AzureADProvider) DiscoverConfiguration(ctx context.Context) (*OIDCConfiguration, error) {
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

	defer resp.Body.Close()

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

func (p *AzureADProvider) ValidateIDToken(ctx context.Context, idToken string) (*IDTokenClaims, error) {
	// Similar to Google implementation but with Azure AD specific validation.

	// This is a simplified version - in production you'd want full JWT validation.

	return nil, fmt.Errorf("ID token validation not fully implemented for Azure AD")
}

// GetJWKS retrieves JSON Web Key Set for token validation.

func (p *AzureADProvider) GetJWKS(ctx context.Context) (*JWKS, error) {
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

	defer resp.Body.Close()

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

func (p *AzureADProvider) GetUserInfoFromIDToken(idToken string) (*UserInfo, error) {
	claims, err := p.ValidateIDToken(context.Background(), idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ID token: %w", err)
	}

	userInfo := &UserInfo{
		Subject: claims.Subject,

		Email: claims.Email,

		EmailVerified: claims.EmailVerified,

		Name: claims.Name,

		Groups: claims.Groups,

		Roles: claims.Roles,

		Provider: p.GetProviderName(),

		ProviderID: claims.Subject,

		Attributes: claims.Extra,
	}

	return userInfo, nil
}

// Additional Azure AD specific methods.

// GetTenantID returns the tenant ID.

func (p *AzureADProvider) GetTenantID() string {
	return p.tenantID
}

// IsMultiTenant returns whether this is a multi-tenant application.

func (p *AzureADProvider) IsMultiTenant() bool {
	return p.isMultiTenant
}

// SetTenantID updates the tenant ID (useful for multi-tenant apps).

func (p *AzureADProvider) SetTenantID(tenantID string) {
	p.tenantID = tenantID

	p.isMultiTenant = tenantID == "common" || tenantID == "organizations" || tenantID == "consumers"

	// Update endpoints.

	authURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID)

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	p.config.Endpoints.AuthURL = authURL

	p.config.Endpoints.TokenURL = tokenURL

	p.config.Endpoints.JWKSURL = fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", tenantID)

	p.config.Endpoints.DiscoveryURL = fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0/.well-known/openid_configuration", tenantID)

	p.oauth2Cfg.Endpoint = oauth2.Endpoint{
		AuthURL: authURL,

		TokenURL: tokenURL,
	}

	// Clear cached OIDC config.

	p.oidcConfig = nil
}
