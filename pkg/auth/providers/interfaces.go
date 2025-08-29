
package providers



import (

	"context"

	"crypto/rand"

	"crypto/sha256"

	"encoding/base64"

	"fmt"

	"log/slog"

	"time"

)



// OAuthProvider defines the interface that all OAuth2 providers must implement.

type OAuthProvider interface {

	// GetProviderName returns the unique identifier for this provider.

	GetProviderName() string



	// GetAuthorizationURL generates the OAuth2 authorization URL with PKCE support.

	GetAuthorizationURL(state, redirectURI string, options ...AuthOption) (string, *PKCEChallenge, error)



	// ExchangeCodeForToken exchanges authorization code for access token.

	ExchangeCodeForToken(ctx context.Context, code, redirectURI string, challenge *PKCEChallenge) (*TokenResponse, error)



	// RefreshToken refreshes an access token using refresh token.

	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)



	// GetUserInfo retrieves user information using access token.

	GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)



	// ValidateToken validates an access token.

	ValidateToken(ctx context.Context, accessToken string) (*TokenValidation, error)



	// RevokeToken revokes an access token.

	RevokeToken(ctx context.Context, token string) error



	// SupportsFeature checks if provider supports specific features.

	SupportsFeature(feature ProviderFeature) bool



	// GetConfiguration returns provider configuration.

	GetConfiguration() *ProviderConfig

}



// OIDCProvider extends OAuthProvider with OpenID Connect specific functionality.

type OIDCProvider interface {

	OAuthProvider



	// DiscoverConfiguration discovers OIDC configuration from well-known endpoint.

	DiscoverConfiguration(ctx context.Context) (*OIDCConfiguration, error)



	// ValidateIDToken validates an OpenID Connect ID token.

	ValidateIDToken(ctx context.Context, idToken string) (*IDTokenClaims, error)



	// GetJWKS retrieves JSON Web Key Set for token validation.

	GetJWKS(ctx context.Context) (*JWKS, error)



	// GetUserInfoFromIDToken extracts user info from ID token claims.

	GetUserInfoFromIDToken(idToken string) (*UserInfo, error)

}



// EnterpriseProvider extends OAuthProvider for enterprise-specific features.

type EnterpriseProvider interface {

	OAuthProvider



	// GetGroups retrieves user groups from the provider.

	GetGroups(ctx context.Context, accessToken string) ([]string, error)



	// GetRoles retrieves user roles from the provider.

	GetRoles(ctx context.Context, accessToken string) ([]string, error)



	// CheckGroupMembership checks if user belongs to specific groups.

	CheckGroupMembership(ctx context.Context, accessToken string, groups []string) ([]string, error)



	// GetOrganizations retrieves user's organizations.

	GetOrganizations(ctx context.Context, accessToken string) ([]Organization, error)



	// ValidateUserAccess validates if user has required access level.

	ValidateUserAccess(ctx context.Context, accessToken string, requiredLevel AccessLevel) error

}



// LDAPProvider defines interface for LDAP/AD integration.

type LDAPProvider interface {

	// Connect establishes connection to LDAP server.

	Connect(ctx context.Context) error



	// Authenticate authenticates user with LDAP.

	Authenticate(ctx context.Context, username, password string) (*UserInfo, error)



	// SearchUser searches for user in LDAP directory.

	SearchUser(ctx context.Context, username string) (*UserInfo, error)



	// GetUserGroups retrieves groups for user.

	GetUserGroups(ctx context.Context, username string) ([]string, error)



	// GetUserRoles retrieves roles for user.

	GetUserRoles(ctx context.Context, username string) ([]string, error)



	// ValidateUserAttributes validates user attributes.

	ValidateUserAttributes(ctx context.Context, username string, requiredAttrs map[string]string) error



	// TestConnection tests the LDAP connection.

	TestConnection(ctx context.Context) error



	// Close closes LDAP connection.

	Close() error



	// Testing helpers.

	GetConfig() *LDAPConfig

	GetLogger() *slog.Logger

	MapGroupsToRoles(groups []string) []string

	ExtractGroupNameFromDN(dn string) string

	ContainsString(slice []string, item string) bool

}



// TokenResponse represents OAuth2/OIDC token response.

type TokenResponse struct {

	AccessToken  string    `json:"access_token"`

	RefreshToken string    `json:"refresh_token,omitempty"`

	IDToken      string    `json:"id_token,omitempty"`

	TokenType    string    `json:"token_type"`

	ExpiresIn    int64     `json:"expires_in"`

	Scope        string    `json:"scope,omitempty"`

	IssuedAt     time.Time `json:"issued_at"`



	// Provider-specific fields.

	Extra map[string]interface{} `json:"extra,omitempty"`

}



// UserInfo represents user information from identity provider.

type UserInfo struct {

	// Standard OIDC claims.

	Subject       string `json:"sub"`

	Email         string `json:"email"`

	EmailVerified bool   `json:"email_verified"`

	Name          string `json:"name"`

	GivenName     string `json:"given_name"`

	FamilyName    string `json:"family_name"`

	MiddleName    string `json:"middle_name"`

	Nickname      string `json:"nickname"`

	PreferredName string `json:"preferred_username"`

	Picture       string `json:"picture"`

	Website       string `json:"website"`

	Gender        string `json:"gender"`

	Birthdate     string `json:"birthdate"`

	Zoneinfo      string `json:"zoneinfo"`

	Locale        string `json:"locale"`

	UpdatedAt     int64  `json:"updated_at"`



	// Provider-specific fields.

	Username      string         `json:"username"`

	Groups        []string       `json:"groups"`

	Roles         []string       `json:"roles"`

	Permissions   []string       `json:"permissions"`

	Organizations []Organization `json:"organizations"`



	// Metadata.

	Provider   string                 `json:"provider"`

	ProviderID string                 `json:"provider_id"`

	Attributes map[string]interface{} `json:"attributes,omitempty"`

}



// Organization represents user's organization.

type Organization struct {

	ID          string   `json:"id"`

	Name        string   `json:"name"`

	DisplayName string   `json:"display_name"`

	Role        string   `json:"role"`

	Permissions []string `json:"permissions"`

}



// TokenValidation represents token validation result.

type TokenValidation struct {

	Valid     bool      `json:"valid"`

	ExpiresAt time.Time `json:"expires_at"`

	Scopes    []string  `json:"scopes"`

	ClientID  string    `json:"client_id"`

	Username  string    `json:"username"`

	Error     string    `json:"error,omitempty"`

}



// PKCEChallenge represents PKCE challenge for OAuth2.

type PKCEChallenge struct {

	CodeVerifier  string `json:"code_verifier"`

	CodeChallenge string `json:"code_challenge"`

	Method        string `json:"code_challenge_method"`

}



// GeneratePKCEChallenge generates a PKCE challenge.

func GeneratePKCEChallenge() (*PKCEChallenge, error) {

	// Generate code verifier (43-128 characters).

	verifier := make([]byte, 32)

	if _, err := rand.Read(verifier); err != nil {

		return nil, fmt.Errorf("failed to generate code verifier: %w", err)

	}



	codeVerifier := base64.RawURLEncoding.EncodeToString(verifier)



	// Generate code challenge using S256 method.

	hash := sha256.Sum256([]byte(codeVerifier))

	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])



	return &PKCEChallenge{

		CodeVerifier:  codeVerifier,

		CodeChallenge: codeChallenge,

		Method:        "S256",

	}, nil

}



// OIDCConfiguration represents OpenID Connect discovery configuration.

type OIDCConfiguration struct {

	Issuer                            string   `json:"issuer"`

	AuthorizationEndpoint             string   `json:"authorization_endpoint"`

	TokenEndpoint                     string   `json:"token_endpoint"`

	UserInfoEndpoint                  string   `json:"userinfo_endpoint"`

	JWKSUri                           string   `json:"jwks_uri"`

	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`

	ScopesSupported                   []string `json:"scopes_supported"`

	ResponseTypesSupported            []string `json:"response_types_supported"`

	ResponseModesSupported            []string `json:"response_modes_supported,omitempty"`

	GrantTypesSupported               []string `json:"grant_types_supported"`

	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`

	SubjectTypesSupported             []string `json:"subject_types_supported"`

	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`

	ClaimsSupported                   []string `json:"claims_supported"`

	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported,omitempty"`

}



// IDTokenClaims represents OpenID Connect ID token claims.

type IDTokenClaims struct {

	// Standard claims.

	Issuer    string `json:"iss"`

	Subject   string `json:"sub"`

	Audience  string `json:"aud"`

	ExpiresAt int64  `json:"exp"`

	IssuedAt  int64  `json:"iat"`

	AuthTime  int64  `json:"auth_time,omitempty"`

	Nonce     string `json:"nonce,omitempty"`



	// Profile claims.

	Name              string `json:"name,omitempty"`

	GivenName         string `json:"given_name,omitempty"`

	FamilyName        string `json:"family_name,omitempty"`

	MiddleName        string `json:"middle_name,omitempty"`

	Nickname          string `json:"nickname,omitempty"`

	PreferredUsername string `json:"preferred_username,omitempty"`

	Picture           string `json:"picture,omitempty"`

	Website           string `json:"website,omitempty"`



	// Email claims.

	Email         string `json:"email,omitempty"`

	EmailVerified bool   `json:"email_verified,omitempty"`



	// Additional claims.

	Groups []string               `json:"groups,omitempty"`

	Roles  []string               `json:"roles,omitempty"`

	Extra  map[string]interface{} `json:"-"`

}



// JWKS represents JSON Web Key Set.

type JWKS struct {

	Keys []JWK `json:"keys"`

}



// JWK represents a JSON Web Key.

type JWK struct {

	KeyType   string   `json:"kty"`

	KeyID     string   `json:"kid"`

	Use       string   `json:"use"`

	Algorithm string   `json:"alg"`

	Modulus   string   `json:"n,omitempty"`

	Exponent  string   `json:"e,omitempty"`

	X         string   `json:"x,omitempty"`

	Y         string   `json:"y,omitempty"`

	Curve     string   `json:"crv,omitempty"`

	D         string   `json:"d,omitempty"`

	K         string   `json:"k,omitempty"`

	KeyOps    []string `json:"key_ops,omitempty"`

	X5c       []string `json:"x5c,omitempty"`

	X5t       string   `json:"x5t,omitempty"`

	X5tS256   string   `json:"x5t#S256,omitempty"`

}



// ProviderConfig represents provider configuration.

type ProviderConfig struct {

	Name         string                 `json:"name"`

	Type         string                 `json:"type"`

	ClientID     string                 `json:"client_id"`

	ClientSecret string                 `json:"client_secret"`

	RedirectURL  string                 `json:"redirect_url"`

	Scopes       []string               `json:"scopes"`

	Endpoints    ProviderEndpoints      `json:"endpoints"`

	Features     []ProviderFeature      `json:"features"`

	Metadata     map[string]interface{} `json:"metadata,omitempty"`

}



// ProviderEndpoints represents OAuth2/OIDC endpoints.

type ProviderEndpoints struct {

	AuthURL       string `json:"auth_url"`

	TokenURL      string `json:"token_url"`

	UserInfoURL   string `json:"userinfo_url"`

	JWKSURL       string `json:"jwks_url,omitempty"`

	RevokeURL     string `json:"revoke_url,omitempty"`

	IntrospectURL string `json:"introspect_url,omitempty"`

	DiscoveryURL  string `json:"discovery_url,omitempty"`

}



// ProviderFeature represents supported provider features.

type ProviderFeature string



const (

	// FeatureOIDC holds featureoidc value.

	FeatureOIDC ProviderFeature = "oidc"

	// FeaturePKCE holds featurepkce value.

	FeaturePKCE ProviderFeature = "pkce"

	// FeatureTokenRefresh holds featuretokenrefresh value.

	FeatureTokenRefresh ProviderFeature = "token_refresh"

	// FeatureTokenRevocation holds featuretokenrevocation value.

	FeatureTokenRevocation ProviderFeature = "token_revocation"

	// FeatureTokenIntrospection holds featuretokenintrospection value.

	FeatureTokenIntrospection ProviderFeature = "token_introspection"

	// FeatureUserInfo holds featureuserinfo value.

	FeatureUserInfo ProviderFeature = "userinfo"

	// FeatureGroups holds featuregroups value.

	FeatureGroups ProviderFeature = "groups"

	// FeatureRoles holds featureroles value.

	FeatureRoles ProviderFeature = "roles"

	// FeatureOrganizations holds featureorganizations value.

	FeatureOrganizations ProviderFeature = "organizations"

	// FeatureJWTTokens holds featurejwttokens value.

	FeatureJWTTokens ProviderFeature = "jwt_tokens"

	// FeatureDiscovery holds featurediscovery value.

	FeatureDiscovery ProviderFeature = "discovery"

)



// AccessLevel represents user access levels.

type AccessLevel int



const (

	// AccessLevelRead holds accesslevelread value.

	AccessLevelRead AccessLevel = iota

	// AccessLevelWrite holds accesslevelwrite value.

	AccessLevelWrite

	// AccessLevelAdmin holds accessleveladmin value.

	AccessLevelAdmin

	// AccessLevelSuperAdmin holds accesslevelsuperadmin value.

	AccessLevelSuperAdmin

)



// AuthOption represents authorization options.

type AuthOption func(*AuthOptions)



// AuthOptions represents options for authorization.

type AuthOptions struct {

	// PKCE options.

	UsePKCE bool



	// Additional parameters.

	Prompt     string

	LoginHint  string

	DomainHint string

	MaxAge     int

	UILocales  []string



	// Custom parameters.

	CustomParams map[string]string

}



// WithPKCE enables PKCE for the authorization request.

func WithPKCE() AuthOption {

	return func(opts *AuthOptions) {

		opts.UsePKCE = true

	}

}



// WithPrompt sets the prompt parameter.

func WithPrompt(prompt string) AuthOption {

	return func(opts *AuthOptions) {

		opts.Prompt = prompt

	}

}



// WithLoginHint sets the login_hint parameter.

func WithLoginHint(hint string) AuthOption {

	return func(opts *AuthOptions) {

		opts.LoginHint = hint

	}

}



// WithDomainHint sets the domain_hint parameter (Microsoft specific).

func WithDomainHint(hint string) AuthOption {

	return func(opts *AuthOptions) {

		opts.DomainHint = hint

	}

}



// WithMaxAge sets the max_age parameter.

func WithMaxAge(maxAge int) AuthOption {

	return func(opts *AuthOptions) {

		opts.MaxAge = maxAge

	}

}



// WithCustomParam adds a custom parameter.

func WithCustomParam(key, value string) AuthOption {

	return func(opts *AuthOptions) {

		if opts.CustomParams == nil {

			opts.CustomParams = make(map[string]string)

		}

		opts.CustomParams[key] = value

	}

}



// ApplyOptions applies auth options.

func ApplyOptions(options ...AuthOption) *AuthOptions {

	opts := &AuthOptions{}

	for _, option := range options {

		option(opts)

	}

	return opts

}



// ProviderError represents provider-specific errors.

type ProviderError struct {

	Provider    string `json:"provider"`

	Code        string `json:"error"`

	Description string `json:"error_description"`

	URI         string `json:"error_uri,omitempty"`

	Cause       error  `json:"-"`

}



// Error performs error operation.

func (e *ProviderError) Error() string {

	if e.Description != "" {

		return fmt.Sprintf("%s: %s - %s", e.Provider, e.Code, e.Description)

	}

	return fmt.Sprintf("%s: %s", e.Provider, e.Code)

}



// Unwrap performs unwrap operation.

func (e *ProviderError) Unwrap() error {

	return e.Cause

}



// NewProviderError creates a new provider error.

func NewProviderError(provider, code, description string, cause error) *ProviderError {

	return &ProviderError{

		Provider:    provider,

		Code:        code,

		Description: description,

		Cause:       cause,

	}

}

