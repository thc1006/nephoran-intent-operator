package providers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GitHubProvider implements OAuth2 authentication for GitHub
type GitHubProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	oauth2Cfg  *oauth2.Config
}

// GitHubUserInfo represents GitHub user information
type GitHubUserInfo struct {
	ID              int64  `json:"id"`
	Login           string `json:"login"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	AvatarURL       string `json:"avatar_url"`
	HTMLURL         string `json:"html_url"`
	Company         string `json:"company"`
	Location        string `json:"location"`
	Bio             string `json:"bio"`
	TwitterUsername string `json:"twitter_username"`
	PublicRepos     int    `json:"public_repos"`
	PublicGists     int    `json:"public_gists"`
	Followers       int    `json:"followers"`
	Following       int    `json:"following"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// GitHubOrganization represents a GitHub organization
type GitHubOrganization struct {
	ID          int64  `json:"id"`
	Login       string `json:"login"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Name        string `json:"name"`
	Company     string `json:"company"`
	Location    string `json:"location"`
}

// GitHubTeam represents a GitHub team
type GitHubTeam struct {
	ID           int64              `json:"id"`
	Name         string             `json:"name"`
	Slug         string             `json:"slug"`
	Description  string             `json:"description"`
	Privacy      string             `json:"privacy"`
	Permission   string             `json:"permission"`
	Organization GitHubOrganization `json:"organization"`
}

// NewGitHubProvider creates a new GitHub OAuth2 provider
func NewGitHubProvider(clientID, clientSecret, redirectURL string) *GitHubProvider {
	config := &ProviderConfig{
		Name:         "github",
		Type:         "github",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"user:email", "read:org", "read:user"},
		Endpoints: ProviderEndpoints{
			AuthURL:     github.Endpoint.AuthURL,
			TokenURL:    github.Endpoint.TokenURL,
			UserInfoURL: "https://api.github.com/user",
		},
		Features: []ProviderFeature{
			FeaturePKCE,
			FeatureTokenRefresh,
			FeatureUserInfo,
			FeatureGroups,
			FeatureOrganizations,
		},
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       config.Scopes,
		Endpoint:     github.Endpoint,
	}

	return &GitHubProvider{
		config:    config,
		oauth2Cfg: oauth2Cfg,
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

// GetProviderName returns the provider name
func (p *GitHubProvider) GetProviderName() string {
	return p.config.Name
}

// GetAuthorizationURL generates OAuth2 authorization URL with PKCE support
func (p *GitHubProvider) GetAuthorizationURL(state, redirectURI string, options ...AuthOption) (string, *PKCEChallenge, error) {
	opts := ApplyOptions(options...)

	// Update redirect URI if provided
	config := *p.oauth2Cfg
	if redirectURI != "" {
		config.RedirectURL = redirectURI
	}

	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	var challenge *PKCEChallenge
	var err error

	if opts.UsePKCE {
		challenge, err = GeneratePKCEChallenge()
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate PKCE challenge: %w", err)
		}

		// Add PKCE parameters to URL
		parsedURL, err := url.Parse(authURL)
		if err != nil {
			return "", nil, fmt.Errorf("failed to parse auth URL: %w", err)
		}

		query := parsedURL.Query()
		query.Set("code_challenge", challenge.CodeChallenge)
		query.Set("code_challenge_method", challenge.Method)
		parsedURL.RawQuery = query.Encode()
		authURL = parsedURL.String()
	}

	// Add custom parameters
	if opts.LoginHint != "" {
		parsedURL, err := url.Parse(authURL)
		if err != nil {
			return "", challenge, fmt.Errorf("failed to parse auth URL for login hint: %w", err)
		}
		query := parsedURL.Query()
		query.Set("login", opts.LoginHint)
		parsedURL.RawQuery = query.Encode()
		authURL = parsedURL.String()
	}

	for key, value := range opts.CustomParams {
		parsedURL, err := url.Parse(authURL)
		if err != nil {
			return "", challenge, fmt.Errorf("failed to parse auth URL for custom param %s: %w", key, err)
		}
		query := parsedURL.Query()
		query.Set(key, value)
		parsedURL.RawQuery = query.Encode()
		authURL = parsedURL.String()
	}

	return authURL, challenge, nil
}

// ExchangeCodeForToken exchanges authorization code for access token
func (p *GitHubProvider) ExchangeCodeForToken(ctx context.Context, code, redirectURI string, challenge *PKCEChallenge) (*TokenResponse, error) {
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

	return &TokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresIn:    int64(time.Until(token.Expiry).Seconds()),
		IssuedAt:     time.Now(),
	}, nil
}

// RefreshToken refreshes an access token using refresh token
func (p *GitHubProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := p.oauth2Cfg.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, NewProviderError(p.GetProviderName(), "token_refresh_failed",
			"Failed to refresh token", err)
	}

	return &TokenResponse{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		TokenType:    newToken.TokenType,
		ExpiresIn:    int64(time.Until(newToken.Expiry).Seconds()),
		IssuedAt:     time.Now(),
	}, nil
}

// GetUserInfo retrieves user information using access token
func (p *GitHubProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	// Get primary user info
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Nephoran-Intent-Operator/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError(p.GetProviderName(), "userinfo_failed",
			fmt.Sprintf("GitHub API returned status %d", resp.StatusCode), nil)
	}

	var githubUser GitHubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Get user emails if email is null
	emails, err := p.getUserEmails(ctx, accessToken)
	if err == nil && len(emails) > 0 && githubUser.Email == "" {
		// Find primary email
		for _, email := range emails {
			if email.Primary {
				githubUser.Email = email.Email
				break
			}
		}
	}

	// Get organizations
	organizations, err := p.GetOrganizations(ctx, accessToken)
	if err != nil {
		// Log error but don't fail the request
		organizations = []Organization{}
	}

	// Get teams (groups)
	teams, err := p.getUserTeams(ctx, accessToken)
	if err != nil {
		// Log error but don't fail the request
		teams = []string{}
	}

	userInfo := &UserInfo{
		Subject:       fmt.Sprintf("github-%d", githubUser.ID),
		Email:         githubUser.Email,
		EmailVerified: githubUser.Email != "",
		Name:          githubUser.Name,
		PreferredName: githubUser.Login,
		Username:      githubUser.Login,
		Picture:       githubUser.AvatarURL,
		Website:       githubUser.HTMLURL,
		Groups:        teams,
		Organizations: organizations,
		Provider:      p.GetProviderName(),
		ProviderID:    fmt.Sprintf("%d", githubUser.ID),
		Attributes: map[string]interface{}{
			"github_id":        githubUser.ID,
			"github_login":     githubUser.Login,
			"company":          githubUser.Company,
			"location":         githubUser.Location,
			"bio":              githubUser.Bio,
			"twitter_username": githubUser.TwitterUsername,
			"public_repos":     githubUser.PublicRepos,
			"followers":        githubUser.Followers,
			"following":        githubUser.Following,
			"created_at":       githubUser.CreatedAt,
		},
	}

	return userInfo, nil
}

// ValidateToken validates an access token
func (p *GitHubProvider) ValidateToken(ctx context.Context, accessToken string) (*TokenValidation, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Nephoran-Intent-Operator/1.0")

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

	var githubUser GitHubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return &TokenValidation{
			Valid: false,
			Error: "Failed to decode user info",
		}, nil
	}

	return &TokenValidation{
		Valid:    true,
		Username: githubUser.Login,
		// Note: GitHub doesn't provide token expiration info in user endpoint
		ExpiresAt: time.Now().Add(time.Hour), // Assume 1 hour for validation purposes
	}, nil
}

// RevokeToken revokes an access token
func (p *GitHubProvider) RevokeToken(ctx context.Context, token string) error {
	// GitHub doesn't have a standard token revocation endpoint
	// The token becomes invalid when the user revokes access in GitHub settings
	// or when the app is uninstalled
	return NewProviderError(p.GetProviderName(), "not_supported",
		"GitHub does not support programmatic token revocation", nil)
}

// SupportsFeature checks if provider supports specific features
func (p *GitHubProvider) SupportsFeature(feature ProviderFeature) bool {
	for _, f := range p.config.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// GetConfiguration returns provider configuration
func (p *GitHubProvider) GetConfiguration() *ProviderConfig {
	return p.config
}

// GetOrganizations retrieves user's organizations
func (p *GitHubProvider) GetOrganizations(ctx context.Context, accessToken string) ([]Organization, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/orgs", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create organizations request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Nephoran-Intent-Operator/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d for organizations", resp.StatusCode)
	}

	var githubOrgs []GitHubOrganization
	if err := json.NewDecoder(resp.Body).Decode(&githubOrgs); err != nil {
		return nil, fmt.Errorf("failed to decode organizations: %w", err)
	}

	organizations := make([]Organization, len(githubOrgs))
	for i, org := range githubOrgs {
		organizations[i] = Organization{
			ID:          fmt.Sprintf("%d", org.ID),
			Name:        org.Login,
			DisplayName: org.Name,
		}
	}

	return organizations, nil
}

// getUserEmails retrieves user's email addresses
func (p *GitHubProvider) getUserEmails(ctx context.Context, accessToken string) ([]GitHubEmail, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create emails request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Nephoran-Intent-Operator/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d for emails", resp.StatusCode)
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, fmt.Errorf("failed to decode emails: %w", err)
	}

	return emails, nil
}

// getUserTeams retrieves user's team memberships (as groups)
func (p *GitHubProvider) getUserTeams(ctx context.Context, accessToken string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/teams", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create teams request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Nephoran-Intent-Operator/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{}, nil // Teams endpoint might not be accessible
	}

	var teams []GitHubTeam
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		return []string{}, nil // Continue without teams if decoding fails
	}

	teamNames := make([]string, len(teams))
	for i, team := range teams {
		teamNames[i] = fmt.Sprintf("%s/%s", team.Organization.Login, team.Slug)
	}

	return teamNames, nil
}

// GitHubEmail represents a GitHub email address
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// Additional methods to satisfy EnterpriseProvider interface

// GetGroups retrieves user groups (teams) from GitHub
func (p *GitHubProvider) GetGroups(ctx context.Context, accessToken string) ([]string, error) {
	return p.getUserTeams(ctx, accessToken)
}

// GetRoles retrieves user roles from GitHub (organization roles)
func (p *GitHubProvider) GetRoles(ctx context.Context, accessToken string) ([]string, error) {
	organizations, err := p.GetOrganizations(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	roles := make([]string, 0, len(organizations))
	for _, org := range organizations {
		if org.Role != "" {
			roles = append(roles, fmt.Sprintf("%s:%s", org.Name, org.Role))
		} else {
			roles = append(roles, fmt.Sprintf("%s:member", org.Name))
		}
	}

	return roles, nil
}

// CheckGroupMembership checks if user belongs to specific groups (organizations/teams)
func (p *GitHubProvider) CheckGroupMembership(ctx context.Context, accessToken string, groups []string) ([]string, error) {
	userGroups, err := p.GetGroups(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	userOrgs, err := p.GetOrganizations(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	// Create a map of user's groups and organizations
	userGroupMap := make(map[string]bool)
	for _, group := range userGroups {
		userGroupMap[group] = true
	}
	for _, org := range userOrgs {
		userGroupMap[org.Name] = true
		userGroupMap[strings.ToLower(org.Name)] = true
	}

	var memberGroups []string
	for _, group := range groups {
		if userGroupMap[group] || userGroupMap[strings.ToLower(group)] {
			memberGroups = append(memberGroups, group)
		}
	}

	return memberGroups, nil
}

// ValidateUserAccess validates if user has required access level
func (p *GitHubProvider) ValidateUserAccess(ctx context.Context, accessToken string, requiredLevel AccessLevel) error {
	// For GitHub, we can check organization membership or repository access
	// This is a simplified implementation
	userInfo, err := p.GetUserInfo(ctx, accessToken)
	if err != nil {
		return fmt.Errorf("failed to get user info for access validation: %w", err)
	}

	// Basic validation - ensure user has organizations or is verified
	if requiredLevel >= AccessLevelWrite {
		if len(userInfo.Organizations) == 0 && !userInfo.EmailVerified {
			return fmt.Errorf("user does not meet minimum access requirements")
		}
	}

	return nil
}
