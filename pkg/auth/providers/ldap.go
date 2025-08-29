
package providers



import (

	"context"

	"crypto/tls"

	"fmt"

	"log/slog"

	"strings"

	"time"



	ldap "github.com/go-ldap/ldap/v3"

)



// LDAPClient implements LDAP/AD authentication.

type LDAPClient struct {

	config *LDAPConfig

	conn   *ldap.Conn

	logger *slog.Logger

}



// LDAPConfig represents LDAP configuration.

type LDAPConfig struct {

	// Connection settings.

	Host         string `json:"host"`

	Port         int    `json:"port"`

	UseSSL       bool   `json:"use_ssl"`

	UseTLS       bool   `json:"use_tls"`

	SkipVerify   bool   `json:"skip_verify"`

	BindDN       string `json:"bind_dn"`

	BindPassword string `json:"bind_password"`



	// Search settings.

	BaseDN          string `json:"base_dn"`

	UserFilter      string `json:"user_filter"`

	GroupFilter     string `json:"group_filter"`

	UserSearchBase  string `json:"user_search_base"`

	GroupSearchBase string `json:"group_search_base"`



	// Attribute mappings.

	UserAttributes  LDAPAttributeMap `json:"user_attributes"`

	GroupAttributes LDAPAttributeMap `json:"group_attributes"`



	// Authentication settings.

	AuthMethod        string        `json:"auth_method"` // "simple", "sasl"

	Timeout           time.Duration `json:"timeout"`

	ConnectionTimeout time.Duration `json:"connection_timeout"`

	MaxConnections    int           `json:"max_connections"`



	// Active Directory specific.

	IsActiveDirectory bool   `json:"is_active_directory"`

	Domain            string `json:"domain"`



	// Group membership.

	GroupMemberAttribute string `json:"group_member_attribute"`

	UserGroupAttribute   string `json:"user_group_attribute"`



	// Role mapping.

	RoleMappings map[string][]string `json:"role_mappings"`

	DefaultRoles []string            `json:"default_roles"`

}



// LDAPAttributeMap maps LDAP attributes to standard fields.

type LDAPAttributeMap struct {

	Username    string `json:"username"`

	Email       string `json:"email"`

	FirstName   string `json:"first_name"`

	LastName    string `json:"last_name"`

	DisplayName string `json:"display_name"`

	Groups      string `json:"groups"`

	Title       string `json:"title"`

	Department  string `json:"department"`

	Phone       string `json:"phone"`

	Manager     string `json:"manager"`

}



// LDAPUserInfo represents LDAP user information.

type LDAPUserInfo struct {

	DN          string            `json:"dn"`

	Username    string            `json:"username"`

	Email       string            `json:"email"`

	FirstName   string            `json:"first_name"`

	LastName    string            `json:"last_name"`

	DisplayName string            `json:"display_name"`

	Title       string            `json:"title"`

	Department  string            `json:"department"`

	Phone       string            `json:"phone"`

	Manager     string            `json:"manager"`

	Groups      []string          `json:"groups"`

	Attributes  map[string]string `json:"attributes"`

	Active      bool              `json:"active"`

}



// LDAPGroupInfo represents LDAP group information.

type LDAPGroupInfo struct {

	DN          string   `json:"dn"`

	Name        string   `json:"name"`

	DisplayName string   `json:"display_name"`

	Description string   `json:"description"`

	Members     []string `json:"members"`

	Type        string   `json:"type"`

}



// NewLDAPProvider creates a new LDAP provider (alias for NewLDAPClient).

func NewLDAPProvider(config *LDAPConfig, logger *slog.Logger) LDAPProvider {

	return NewLDAPClient(config, logger)

}



// NewLDAPClient creates a new LDAP client.

func NewLDAPClient(config *LDAPConfig, logger *slog.Logger) *LDAPClient {

	// Set defaults.

	if config.Port == 0 {

		if config.UseSSL {

			config.Port = 636

		} else {

			config.Port = 389

		}

	}



	if config.Timeout == 0 {

		config.Timeout = 30 * time.Second

	}



	if config.ConnectionTimeout == 0 {

		config.ConnectionTimeout = 10 * time.Second

	}



	if config.MaxConnections == 0 {

		config.MaxConnections = 10

	}



	// Set default attribute mappings.

	if config.UserAttributes.Username == "" {

		if config.IsActiveDirectory {

			config.UserAttributes.Username = "sAMAccountName"

			config.UserAttributes.Email = "mail"

			config.UserAttributes.FirstName = "givenName"

			config.UserAttributes.LastName = "sn"

			config.UserAttributes.DisplayName = "displayName"

			config.UserAttributes.Title = "title"

			config.UserAttributes.Department = "department"

			config.UserAttributes.Phone = "telephoneNumber"

			config.UserAttributes.Manager = "manager"

		} else {

			config.UserAttributes.Username = "uid"

			config.UserAttributes.Email = "mail"

			config.UserAttributes.FirstName = "givenName"

			config.UserAttributes.LastName = "sn"

			config.UserAttributes.DisplayName = "displayName"

			config.UserAttributes.Title = "title"

			config.UserAttributes.Department = "ou"

			config.UserAttributes.Phone = "telephoneNumber"

		}

	}



	// Set default filters.

	if config.UserFilter == "" {

		if config.IsActiveDirectory {

			config.UserFilter = "(&(objectClass=user)(objectCategory=person)(!userAccountControl:1.2.840.113556.1.4.803:=2))"

		} else {

			config.UserFilter = "(objectClass=inetOrgPerson)"

		}

	}



	if config.GroupFilter == "" {

		if config.IsActiveDirectory {

			config.GroupFilter = "(objectClass=group)"

		} else {

			config.GroupFilter = "(objectClass=groupOfNames)"

		}

	}



	if config.GroupMemberAttribute == "" {

		if config.IsActiveDirectory {

			config.GroupMemberAttribute = "member"

		} else {

			config.GroupMemberAttribute = "member"

		}

	}



	if config.UserGroupAttribute == "" {

		if config.IsActiveDirectory {

			config.UserGroupAttribute = "memberOf"

		} else {

			config.UserGroupAttribute = "memberOf"

		}

	}



	return &LDAPClient{

		config: config,

		logger: logger,

	}

}



// Connect establishes connection to LDAP server.

func (p *LDAPClient) Connect(ctx context.Context) error {

	address := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)



	var conn *ldap.Conn

	var err error



	if p.config.UseSSL {

		// Direct SSL connection.

		tlsConfig := &tls.Config{

			ServerName:         p.config.Host,

			InsecureSkipVerify: p.config.SkipVerify,

		}

		conn, err = ldap.DialURL(fmt.Sprintf("ldaps://%s", address), ldap.DialWithTLSConfig(tlsConfig))

	} else {

		// Plain connection.

		conn, err = ldap.DialURL(fmt.Sprintf("ldap://%s", address))

		if err == nil && p.config.UseTLS {

			// Upgrade to TLS.

			tlsConfig := &tls.Config{

				ServerName:         p.config.Host,

				InsecureSkipVerify: p.config.SkipVerify,

			}

			err = conn.StartTLS(tlsConfig)

		}

	}



	if err != nil {

		return fmt.Errorf("failed to connect to LDAP server: %w", err)

	}



	// Set connection timeout.

	conn.SetTimeout(p.config.ConnectionTimeout)



	p.conn = conn



	// Bind with service account.

	if p.config.BindDN != "" && p.config.BindPassword != "" {

		err = p.conn.Bind(p.config.BindDN, p.config.BindPassword)

		if err != nil {

			p.conn.Close()

			return fmt.Errorf("failed to bind to LDAP server: %w", err)

		}

	}



	p.logger.Info("Connected to LDAP server",

		"host", p.config.Host,

		"port", p.config.Port,

		"ssl", p.config.UseSSL,

		"tls", p.config.UseTLS)



	return nil

}



// Authenticate authenticates user with LDAP.

func (p *LDAPClient) Authenticate(ctx context.Context, username, password string) (*UserInfo, error) {

	if p.conn == nil {

		if err := p.Connect(ctx); err != nil {

			return nil, err

		}

	}



	// Search for user.

	userDN, ldapUser, err := p.findUser(username)

	if err != nil {

		return nil, fmt.Errorf("user not found: %w", err)

	}



	// Try to bind with user credentials.

	err = p.conn.Bind(userDN, password)

	if err != nil {

		p.logger.Warn("Authentication failed",

			"username", username,

			"error", err)

		return nil, fmt.Errorf("authentication failed: invalid credentials")

	}



	// Re-bind with service account for subsequent operations.

	if p.config.BindDN != "" {

		err = p.conn.Bind(p.config.BindDN, p.config.BindPassword)

		if err != nil {

			return nil, fmt.Errorf("failed to re-bind service account: %w", err)

		}

	}



	// Get user groups.

	groups, err := p.GetUserGroups(ctx, username)

	if err != nil {

		p.logger.Warn("Failed to get user groups", "username", username, "error", err)

		groups = []string{} // Continue without groups

	}



	// Get user roles from group mappings.

	roles := p.mapGroupsToRoles(groups)



	// Convert to standard UserInfo.

	userInfo := &UserInfo{

		Subject:       ldapUser.Username,

		Email:         ldapUser.Email,

		EmailVerified: ldapUser.Email != "",

		Name:          ldapUser.DisplayName,

		GivenName:     ldapUser.FirstName,

		FamilyName:    ldapUser.LastName,

		Username:      ldapUser.Username,

		Groups:        groups,

		Roles:         roles,

		Provider:      "ldap",

		ProviderID:    userDN,

		Attributes: map[string]interface{}{

			"ldap_dn":    ldapUser.DN,

			"title":      ldapUser.Title,

			"department": ldapUser.Department,

			"phone":      ldapUser.Phone,

			"manager":    ldapUser.Manager,

			"active":     ldapUser.Active,

		},

	}



	p.logger.Info("User authenticated successfully",

		"username", username,

		"groups_count", len(groups),

		"roles_count", len(roles))



	return userInfo, nil

}



// SearchUser searches for user in LDAP directory.

func (p *LDAPClient) SearchUser(ctx context.Context, username string) (*UserInfo, error) {

	if p.conn == nil {

		if err := p.Connect(ctx); err != nil {

			return nil, err

		}

	}



	userDN, ldapUser, err := p.findUser(username)

	if err != nil {

		return nil, err

	}



	// Get user groups.

	groups, err := p.GetUserGroups(ctx, username)

	if err != nil {

		groups = []string{}

	}



	// Map groups to roles.

	roles := p.mapGroupsToRoles(groups)



	userInfo := &UserInfo{

		Subject:       ldapUser.Username,

		Email:         ldapUser.Email,

		EmailVerified: ldapUser.Email != "",

		Name:          ldapUser.DisplayName,

		GivenName:     ldapUser.FirstName,

		FamilyName:    ldapUser.LastName,

		Username:      ldapUser.Username,

		Groups:        groups,

		Roles:         roles,

		Provider:      "ldap",

		ProviderID:    userDN,

		Attributes: map[string]interface{}{

			"ldap_dn":    ldapUser.DN,

			"title":      ldapUser.Title,

			"department": ldapUser.Department,

			"phone":      ldapUser.Phone,

			"manager":    ldapUser.Manager,

			"active":     ldapUser.Active,

		},

	}



	return userInfo, nil

}



// GetUserGroups retrieves groups for user.

func (p *LDAPClient) GetUserGroups(ctx context.Context, username string) ([]string, error) {

	if p.conn == nil {

		if err := p.Connect(ctx); err != nil {

			return nil, err

		}

	}



	// First find the user to get their DN.

	userDN, _, err := p.findUser(username)

	if err != nil {

		return nil, err

	}



	var groups []string



	// Method 1: Query groups that have this user as a member.

	searchBase := p.config.GroupSearchBase

	if searchBase == "" {

		searchBase = p.config.BaseDN

	}



	filter := fmt.Sprintf("(&%s(%s=%s))", p.config.GroupFilter, p.config.GroupMemberAttribute, userDN)



	searchRequest := ldap.NewSearchRequest(

		searchBase,

		ldap.ScopeWholeSubtree,

		ldap.NeverDerefAliases,

		0, 0, false,

		filter,

		[]string{"cn", "displayName", "description"},

		nil,

	)



	result, err := p.conn.Search(searchRequest)

	if err == nil {

		for _, entry := range result.Entries {

			groupName := entry.GetAttributeValue("cn")

			if groupName != "" {

				groups = append(groups, groupName)

			}

		}

	}



	// Method 2: Check user's memberOf attribute (if available).

	userSearchRequest := ldap.NewSearchRequest(

		userDN,

		ldap.ScopeBaseObject,

		ldap.NeverDerefAliases,

		0, 0, false,

		"(objectClass=*)",

		[]string{p.config.UserGroupAttribute},

		nil,

	)



	userResult, err := p.conn.Search(userSearchRequest)

	if err == nil && len(userResult.Entries) > 0 {

		memberOfValues := userResult.Entries[0].GetAttributeValues(p.config.UserGroupAttribute)

		for _, memberOf := range memberOfValues {

			// Extract group name from DN.

			groupName := p.extractGroupNameFromDN(memberOf)

			if groupName != "" && !p.containsString(groups, groupName) {

				groups = append(groups, groupName)

			}

		}

	}



	return groups, nil

}



// GetUserRoles retrieves roles for user based on group mappings.

func (p *LDAPClient) GetUserRoles(ctx context.Context, username string) ([]string, error) {

	groups, err := p.GetUserGroups(ctx, username)

	if err != nil {

		return nil, err

	}



	roles := p.mapGroupsToRoles(groups)

	return roles, nil

}



// ValidateUserAttributes validates user attributes.

func (p *LDAPClient) ValidateUserAttributes(ctx context.Context, username string, requiredAttrs map[string]string) error {

	userInfo, err := p.SearchUser(ctx, username)

	if err != nil {

		return err

	}



	for attrName, expectedValue := range requiredAttrs {

		if actualValue, exists := userInfo.Attributes[attrName]; !exists {

			return fmt.Errorf("required attribute %s not found", attrName)

		} else if actualValue != expectedValue {

			return fmt.Errorf("attribute %s has value %v, expected %s", attrName, actualValue, expectedValue)

		}

	}



	return nil

}



// Close closes LDAP connection.

func (p *LDAPClient) Close() error {

	if p.conn != nil {

		p.conn.Close()

		p.conn = nil

		p.logger.Info("LDAP connection closed")

	}

	return nil

}



// Private helper methods.



func (p *LDAPClient) findUser(username string) (string, *LDAPUserInfo, error) {

	searchBase := p.config.UserSearchBase

	if searchBase == "" {

		searchBase = p.config.BaseDN

	}



	// Build search filter.

	var filter string

	if p.config.IsActiveDirectory && p.config.Domain != "" {

		// For AD, try both sAMAccountName and userPrincipalName.

		filter = fmt.Sprintf("(&%s(|(%s=%s)(%s=%s@%s)))",

			p.config.UserFilter,

			p.config.UserAttributes.Username, username,

			"userPrincipalName", username, p.config.Domain)

	} else {

		filter = fmt.Sprintf("(&%s(%s=%s))",

			p.config.UserFilter,

			p.config.UserAttributes.Username, username)

	}



	// Define attributes to retrieve.

	attributes := []string{

		p.config.UserAttributes.Username,

		p.config.UserAttributes.Email,

		p.config.UserAttributes.FirstName,

		p.config.UserAttributes.LastName,

		p.config.UserAttributes.DisplayName,

		p.config.UserAttributes.Title,

		p.config.UserAttributes.Department,

		p.config.UserAttributes.Phone,

		p.config.UserAttributes.Manager,

		p.config.UserGroupAttribute,

	}



	// Add AD-specific attributes.

	if p.config.IsActiveDirectory {

		attributes = append(attributes, "userAccountControl", "objectSid")

	}



	searchRequest := ldap.NewSearchRequest(

		searchBase,

		ldap.ScopeWholeSubtree,

		ldap.NeverDerefAliases,

		1, // Size limit - we only want one result

		int(p.config.Timeout.Seconds()),

		false,

		filter,

		attributes,

		nil,

	)



	result, err := p.conn.Search(searchRequest)

	if err != nil {

		return "", nil, fmt.Errorf("search failed: %w", err)

	}



	if len(result.Entries) == 0 {

		return "", nil, fmt.Errorf("user not found")

	}



	entry := result.Entries[0]



	// Check if user is active (for Active Directory).

	active := true

	if p.config.IsActiveDirectory {

		uac := entry.GetAttributeValue("userAccountControl")

		if uac != "" {

			// Check if account is disabled (bit 2).

			// This is a simplified check - full implementation would parse the UAC properly.

			active = !strings.Contains(uac, "2")

		}

	}



	ldapUser := &LDAPUserInfo{

		DN:          entry.DN,

		Username:    entry.GetAttributeValue(p.config.UserAttributes.Username),

		Email:       entry.GetAttributeValue(p.config.UserAttributes.Email),

		FirstName:   entry.GetAttributeValue(p.config.UserAttributes.FirstName),

		LastName:    entry.GetAttributeValue(p.config.UserAttributes.LastName),

		DisplayName: entry.GetAttributeValue(p.config.UserAttributes.DisplayName),

		Title:       entry.GetAttributeValue(p.config.UserAttributes.Title),

		Department:  entry.GetAttributeValue(p.config.UserAttributes.Department),

		Phone:       entry.GetAttributeValue(p.config.UserAttributes.Phone),

		Manager:     entry.GetAttributeValue(p.config.UserAttributes.Manager),

		Active:      active,

		Attributes:  make(map[string]string),

	}



	// Store all attributes.

	for _, attr := range entry.Attributes {

		if len(attr.Values) > 0 {

			ldapUser.Attributes[attr.Name] = attr.Values[0]

		}

	}



	return entry.DN, ldapUser, nil

}



func (p *LDAPClient) mapGroupsToRoles(groups []string) []string {

	var roles []string

	roleSet := make(map[string]bool)



	// Add default roles.

	for _, role := range p.config.DefaultRoles {

		if !roleSet[role] {

			roles = append(roles, role)

			roleSet[role] = true

		}

	}



	// Map groups to roles.

	for _, group := range groups {

		if mappedRoles, exists := p.config.RoleMappings[group]; exists {

			for _, role := range mappedRoles {

				if !roleSet[role] {

					roles = append(roles, role)

					roleSet[role] = true

				}

			}

		}

	}



	return roles

}



func (p *LDAPClient) extractGroupNameFromDN(dn string) string {

	// Extract CN from DN.

	parts := strings.Split(dn, ",")

	for _, part := range parts {

		part = strings.TrimSpace(part)

		if strings.HasPrefix(strings.ToLower(part), "cn=") {

			return part[3:] // Remove "CN=" prefix

		}

	}

	return ""

}



func (p *LDAPClient) containsString(slice []string, item string) bool {

	for _, s := range slice {

		if s == item {

			return true

		}

	}

	return false

}



// GetGroupInfo retrieves detailed group information.

func (p *LDAPClient) GetGroupInfo(ctx context.Context, groupName string) (*LDAPGroupInfo, error) {

	if p.conn == nil {

		if err := p.Connect(ctx); err != nil {

			return nil, err

		}

	}



	searchBase := p.config.GroupSearchBase

	if searchBase == "" {

		searchBase = p.config.BaseDN

	}



	filter := fmt.Sprintf("(&%s(cn=%s))", p.config.GroupFilter, groupName)



	searchRequest := ldap.NewSearchRequest(

		searchBase,

		ldap.ScopeWholeSubtree,

		ldap.NeverDerefAliases,

		1,

		int(p.config.Timeout.Seconds()),

		false,

		filter,

		[]string{"cn", "displayName", "description", p.config.GroupMemberAttribute, "groupType"},

		nil,

	)



	result, err := p.conn.Search(searchRequest)

	if err != nil {

		return nil, fmt.Errorf("group search failed: %w", err)

	}



	if len(result.Entries) == 0 {

		return nil, fmt.Errorf("group not found")

	}



	entry := result.Entries[0]



	groupInfo := &LDAPGroupInfo{

		DN:          entry.DN,

		Name:        entry.GetAttributeValue("cn"),

		DisplayName: entry.GetAttributeValue("displayName"),

		Description: entry.GetAttributeValue("description"),

		Members:     entry.GetAttributeValues(p.config.GroupMemberAttribute),

		Type:        entry.GetAttributeValue("groupType"),

	}



	return groupInfo, nil

}



// ListGroups lists all groups.

func (p *LDAPClient) ListGroups(ctx context.Context) ([]*LDAPGroupInfo, error) {

	if p.conn == nil {

		if err := p.Connect(ctx); err != nil {

			return nil, err

		}

	}



	searchBase := p.config.GroupSearchBase

	if searchBase == "" {

		searchBase = p.config.BaseDN

	}



	searchRequest := ldap.NewSearchRequest(

		searchBase,

		ldap.ScopeWholeSubtree,

		ldap.NeverDerefAliases,

		0,

		int(p.config.Timeout.Seconds()),

		false,

		p.config.GroupFilter,

		[]string{"cn", "displayName", "description", "groupType"},

		nil,

	)



	result, err := p.conn.Search(searchRequest)

	if err != nil {

		return nil, fmt.Errorf("group listing failed: %w", err)

	}



	var groups []*LDAPGroupInfo

	for _, entry := range result.Entries {

		groupInfo := &LDAPGroupInfo{

			DN:          entry.DN,

			Name:        entry.GetAttributeValue("cn"),

			DisplayName: entry.GetAttributeValue("displayName"),

			Description: entry.GetAttributeValue("description"),

			Type:        entry.GetAttributeValue("groupType"),

		}

		groups = append(groups, groupInfo)

	}



	return groups, nil

}



// TestConnection tests LDAP connection and authentication.

func (p *LDAPClient) TestConnection(ctx context.Context) error {

	if err := p.Connect(ctx); err != nil {

		return fmt.Errorf("connection failed: %w", err)

	}



	// Test search capability.

	searchRequest := ldap.NewSearchRequest(

		p.config.BaseDN,

		ldap.ScopeBaseObject,

		ldap.NeverDerefAliases,

		1,

		int(p.config.Timeout.Seconds()),

		false,

		"(objectClass=*)",

		[]string{"dn"},

		nil,

	)



	_, err := p.conn.Search(searchRequest)

	if err != nil {

		return fmt.Errorf("search test failed: %w", err)

	}



	p.logger.Info("LDAP connection test successful")

	return nil

}



// GetConfig returns the LDAP configuration (for testing).

func (p *LDAPClient) GetConfig() *LDAPConfig {

	return p.config

}



// GetLogger returns the logger (for testing).

func (p *LDAPClient) GetLogger() *slog.Logger {

	return p.logger

}



// MapGroupsToRoles maps LDAP groups to roles (exported for testing).

func (p *LDAPClient) MapGroupsToRoles(groups []string) []string {

	return p.mapGroupsToRoles(groups)

}



// ExtractGroupNameFromDN extracts group name from DN (exported for testing).

func (p *LDAPClient) ExtractGroupNameFromDN(dn string) string {

	return p.extractGroupNameFromDN(dn)

}



// ContainsString checks if slice contains string (exported for testing).

func (p *LDAPClient) ContainsString(slice []string, item string) bool {

	return p.containsString(slice, item)

}

