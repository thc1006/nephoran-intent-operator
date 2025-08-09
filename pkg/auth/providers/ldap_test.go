package providers

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLDAPProvider(t *testing.T) {
	tests := []struct {
		name           string
		config         *LDAPConfig
		expectDefaults bool
	}{
		{
			name: "basic_openldap_config",
			config: &LDAPConfig{
				Host:   "ldap.example.com",
				BaseDN: "dc=example,dc=com",
			},
			expectDefaults: true,
		},
		{
			name: "active_directory_config",
			config: &LDAPConfig{
				Host:              "ad.example.com",
				BaseDN:            "dc=example,dc=com",
				IsActiveDirectory: true,
				Domain:            "example.com",
			},
			expectDefaults: true,
		},
		{
			name: "custom_config_with_ssl",
			config: &LDAPConfig{
				Host:              "secure-ldap.example.com",
				Port:              636,
				UseSSL:            true,
				BaseDN:            "ou=users,dc=example,dc=com",
				Timeout:           45 * time.Second,
				MaxConnections:    20,
				IsActiveDirectory: false,
			},
			expectDefaults: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
			provider := NewLDAPProvider(tt.config, logger)

			assert.NotNil(t, provider)
			assert.Equal(t, tt.config, provider.config)
			assert.Equal(t, logger, provider.logger)

			if tt.expectDefaults {
				// Check that defaults were applied
				if tt.config.IsActiveDirectory {
					assert.Equal(t, 389, tt.config.Port) // Default LDAP port
					assert.Equal(t, "sAMAccountName", tt.config.UserAttributes.Username)
					assert.Contains(t, tt.config.UserFilter, "objectClass=user")
				} else {
					assert.Equal(t, 389, tt.config.Port)
					assert.Equal(t, "uid", tt.config.UserAttributes.Username)
					assert.Contains(t, tt.config.UserFilter, "objectClass=inetOrgPerson")
				}

				assert.Equal(t, 30*time.Second, tt.config.Timeout)
				assert.Equal(t, 10*time.Second, tt.config.ConnectionTimeout)
				assert.Equal(t, 10, tt.config.MaxConnections)
			}

			// SSL port default
			if tt.config.UseSSL && tt.config.Port == 0 {
				assert.Equal(t, 636, tt.config.Port)
			}
		})
	}
}

func TestLDAPConfig_AttributeMapping(t *testing.T) {
	tests := []struct {
		name          string
		isAD          bool
		expectedAttrs LDAPAttributeMap
	}{
		{
			name: "openldap_defaults",
			isAD: false,
			expectedAttrs: LDAPAttributeMap{
				Username:    "uid",
				Email:       "mail",
				FirstName:   "givenName",
				LastName:    "sn",
				DisplayName: "displayName",
				Title:       "title",
				Department:  "ou",
				Phone:       "telephoneNumber",
			},
		},
		{
			name: "active_directory_defaults",
			isAD: true,
			expectedAttrs: LDAPAttributeMap{
				Username:    "sAMAccountName",
				Email:       "mail",
				FirstName:   "givenName",
				LastName:    "sn",
				DisplayName: "displayName",
				Title:       "title",
				Department:  "department",
				Phone:       "telephoneNumber",
				Manager:     "manager",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LDAPConfig{
				Host:              "test.example.com",
				BaseDN:            "dc=test,dc=com",
				IsActiveDirectory: tt.isAD,
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
			_ = NewLDAPProvider(config, logger)

			assert.Equal(t, tt.expectedAttrs.Username, config.UserAttributes.Username)
			assert.Equal(t, tt.expectedAttrs.Email, config.UserAttributes.Email)
			assert.Equal(t, tt.expectedAttrs.FirstName, config.UserAttributes.FirstName)
			assert.Equal(t, tt.expectedAttrs.LastName, config.UserAttributes.LastName)
			assert.Equal(t, tt.expectedAttrs.DisplayName, config.UserAttributes.DisplayName)
			assert.Equal(t, tt.expectedAttrs.Title, config.UserAttributes.Title)
			assert.Equal(t, tt.expectedAttrs.Department, config.UserAttributes.Department)
			assert.Equal(t, tt.expectedAttrs.Phone, config.UserAttributes.Phone)

			if tt.isAD {
				assert.Equal(t, tt.expectedAttrs.Manager, config.UserAttributes.Manager)
			}
		})
	}
}

func TestLDAPConfig_FilterDefaults(t *testing.T) {
	tests := []struct {
		name          string
		isAD          bool
		expectedUser  string
		expectedGroup string
	}{
		{
			name:          "openldap_filters",
			isAD:          false,
			expectedUser:  "(objectClass=inetOrgPerson)",
			expectedGroup: "(objectClass=groupOfNames)",
		},
		{
			name:          "active_directory_filters",
			isAD:          true,
			expectedUser:  "(&(objectClass=user)(objectCategory=person)(!userAccountControl:1.2.840.113556.1.4.803:=2))",
			expectedGroup: "(objectClass=group)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LDAPConfig{
				Host:              "test.example.com",
				BaseDN:            "dc=test,dc=com",
				IsActiveDirectory: tt.isAD,
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
			_ = NewLDAPProvider(config, logger)

			assert.Equal(t, tt.expectedUser, config.UserFilter)
			assert.Equal(t, tt.expectedGroup, config.GroupFilter)
		})
	}
}

func TestLDAPProvider_MapGroupsToRoles(t *testing.T) {
	config := &LDAPConfig{
		Host:   "test.example.com",
		BaseDN: "dc=test,dc=com",
		RoleMappings: map[string][]string{
			"cn=Administrators,ou=Groups,dc=test,dc=com": {"admin", "operator"},
			"cn=Operators,ou=Groups,dc=test,dc=com":      {"operator"},
			"cn=Users,ou=Groups,dc=test,dc=com":          {"user"},
		},
		DefaultRoles: []string{"user", "viewer"},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	provider := NewLDAPProvider(config, logger)

	tests := []struct {
		name          string
		groups        []string
		expectedRoles []string
	}{
		{
			name:          "no_groups",
			groups:        []string{},
			expectedRoles: []string{"user", "viewer"},
		},
		{
			name:          "admin_group",
			groups:        []string{"cn=Administrators,ou=Groups,dc=test,dc=com"},
			expectedRoles: []string{"user", "viewer", "admin", "operator"},
		},
		{
			name:          "operator_group",
			groups:        []string{"cn=Operators,ou=Groups,dc=test,dc=com"},
			expectedRoles: []string{"user", "viewer", "operator"},
		},
		{
			name:          "multiple_groups",
			groups:        []string{"cn=Operators,ou=Groups,dc=test,dc=com", "cn=Users,ou=Groups,dc=test,dc=com"},
			expectedRoles: []string{"user", "viewer", "operator"},
		},
		{
			name:          "unmapped_groups",
			groups:        []string{"cn=Guests,ou=Groups,dc=test,dc=com"},
			expectedRoles: []string{"user", "viewer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles := provider.mapGroupsToRoles(tt.groups)

			// Sort both slices for comparison
			assert.ElementsMatch(t, tt.expectedRoles, roles)
		})
	}
}

func TestLDAPProvider_ExtractGroupNameFromDN(t *testing.T) {
	config := &LDAPConfig{
		Host:   "test.example.com",
		BaseDN: "dc=test,dc=com",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	provider := NewLDAPProvider(config, logger)

	tests := []struct {
		name         string
		dn           string
		expectedName string
	}{
		{
			name:         "simple_group_dn",
			dn:           "cn=Administrators,ou=Groups,dc=test,dc=com",
			expectedName: "Administrators",
		},
		{
			name:         "complex_group_dn",
			dn:           "CN=Domain Admins,CN=Users,DC=example,DC=com",
			expectedName: "Domain Admins",
		},
		{
			name:         "no_cn_component",
			dn:           "ou=Groups,dc=test,dc=com",
			expectedName: "",
		},
		{
			name:         "empty_dn",
			dn:           "",
			expectedName: "",
		},
		{
			name:         "malformed_dn",
			dn:           "invalid-dn-format",
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.extractGroupNameFromDN(tt.dn)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}

func TestLDAPProvider_ContainsString(t *testing.T) {
	config := &LDAPConfig{
		Host:   "test.example.com",
		BaseDN: "dc=test,dc=com",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	provider := NewLDAPProvider(config, logger)

	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item_exists",
			slice:    []string{"admin", "operator", "user"},
			item:     "operator",
			expected: true,
		},
		{
			name:     "item_not_exists",
			slice:    []string{"admin", "operator", "user"},
			item:     "guest",
			expected: false,
		},
		{
			name:     "empty_slice",
			slice:    []string{},
			item:     "admin",
			expected: false,
		},
		{
			name:     "empty_item",
			slice:    []string{"admin", "operator", "user"},
			item:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.containsString(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock LDAP server tests would require an actual LDAP server or mock
// For integration tests, consider using testcontainers with OpenLDAP

func TestLDAPProvider_ConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *LDAPConfig
		expectError bool
	}{
		{
			name: "valid_minimal_config",
			config: &LDAPConfig{
				Host:   "ldap.example.com",
				BaseDN: "dc=example,dc=com",
			},
			expectError: false,
		},
		{
			name: "valid_full_config",
			config: &LDAPConfig{
				Host:                 "ldaps.example.com",
				Port:                 636,
				UseSSL:               true,
				BindDN:               "cn=admin,dc=example,dc=com",
				BindPassword:         "password",
				BaseDN:               "dc=example,dc=com",
				UserFilter:           "(objectClass=person)",
				GroupFilter:          "(objectClass=group)",
				UserSearchBase:       "ou=users,dc=example,dc=com",
				GroupSearchBase:      "ou=groups,dc=example,dc=com",
				AuthMethod:           "simple",
				Timeout:              30 * time.Second,
				ConnectionTimeout:    10 * time.Second,
				MaxConnections:       5,
				GroupMemberAttribute: "member",
				UserGroupAttribute:   "memberOf",
				RoleMappings: map[string][]string{
					"admin": {"system-admin"},
				},
				DefaultRoles: []string{"user"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
			provider := NewLDAPProvider(tt.config, logger)

			// Basic validation - provider should be created
			assert.NotNil(t, provider)

			// Configuration should be preserved
			assert.Equal(t, tt.config.Host, provider.config.Host)
			assert.Equal(t, tt.config.BaseDN, provider.config.BaseDN)
		})
	}
}

func TestLDAPProvider_ConnectionConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		config       *LDAPConfig
		expectedPort int
		expectedSSL  bool
		expectedTLS  bool
	}{
		{
			name: "default_ldap_port",
			config: &LDAPConfig{
				Host:   "ldap.example.com",
				BaseDN: "dc=example,dc=com",
			},
			expectedPort: 389,
			expectedSSL:  false,
			expectedTLS:  false,
		},
		{
			name: "ssl_ldap_port",
			config: &LDAPConfig{
				Host:   "ldaps.example.com",
				UseSSL: true,
				BaseDN: "dc=example,dc=com",
			},
			expectedPort: 636,
			expectedSSL:  true,
			expectedTLS:  false,
		},
		{
			name: "tls_upgrade",
			config: &LDAPConfig{
				Host:   "ldap.example.com",
				UseTLS: true,
				BaseDN: "dc=example,dc=com",
			},
			expectedPort: 389,
			expectedSSL:  false,
			expectedTLS:  true,
		},
		{
			name: "custom_port",
			config: &LDAPConfig{
				Host:   "ldap.example.com",
				Port:   10389,
				BaseDN: "dc=example,dc=com",
			},
			expectedPort: 10389,
			expectedSSL:  false,
			expectedTLS:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
			provider := NewLDAPProvider(tt.config, logger)

			assert.Equal(t, tt.expectedPort, provider.config.Port)
			assert.Equal(t, tt.expectedSSL, provider.config.UseSSL)
			assert.Equal(t, tt.expectedTLS, provider.config.UseTLS)
		})
	}
}

// Benchmark tests for performance validation

func BenchmarkLDAPProvider_MapGroupsToRoles(b *testing.B) {
	config := &LDAPConfig{
		Host:   "test.example.com",
		BaseDN: "dc=test,dc=com",
		RoleMappings: map[string][]string{
			"group1": {"role1", "role2"},
			"group2": {"role2", "role3"},
			"group3": {"role3", "role4"},
			"group4": {"role4", "role5"},
			"group5": {"role5", "role1"},
		},
		DefaultRoles: []string{"user", "viewer"},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	provider := NewLDAPProvider(config, logger)

	groups := []string{"group1", "group2", "group3", "group4", "group5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.mapGroupsToRoles(groups)
	}
}

func BenchmarkLDAPProvider_ExtractGroupNameFromDN(b *testing.B) {
	config := &LDAPConfig{
		Host:   "test.example.com",
		BaseDN: "dc=test,dc=com",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	provider := NewLDAPProvider(config, logger)

	dn := "CN=Domain Admins,CN=Users,DC=example,DC=com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.extractGroupNameFromDN(dn)
	}
}

// Table-driven test for comprehensive configuration scenarios
func TestLDAPProvider_ComprehensiveConfiguration(t *testing.T) {
	testCases := []struct {
		name       string
		config     *LDAPConfig
		assertions func(t *testing.T, provider *LDAPProvider)
	}{
		{
			name: "enterprise_active_directory",
			config: &LDAPConfig{
				Host:              "ad.enterprise.com",
				Port:              636,
				UseSSL:            true,
				BindDN:            "CN=Service Account,OU=Service Accounts,DC=enterprise,DC=com",
				BindPassword:      "service-password",
				BaseDN:            "DC=enterprise,DC=com",
				UserSearchBase:    "OU=Users,DC=enterprise,DC=com",
				GroupSearchBase:   "OU=Groups,DC=enterprise,DC=com",
				IsActiveDirectory: true,
				Domain:            "enterprise.com",
				RoleMappings: map[string][]string{
					"Enterprise Admins": {"system-admin", "admin"},
					"Network Operators": {"network-operator", "operator"},
					"Telecom Engineers": {"telecom-engineer", "operator"},
					"Read Only Users":   {"viewer", "readonly"},
				},
				DefaultRoles:   []string{"user"},
				Timeout:        60 * time.Second,
				MaxConnections: 20,
			},
			assertions: func(t *testing.T, provider *LDAPProvider) {
				// Validate Active Directory specific settings
				assert.True(t, provider.config.IsActiveDirectory)
				assert.Equal(t, "enterprise.com", provider.config.Domain)
				assert.Equal(t, "sAMAccountName", provider.config.UserAttributes.Username)
				assert.Contains(t, provider.config.UserFilter, "objectClass=user")
				assert.Contains(t, provider.config.GroupFilter, "objectClass=group")

				// Validate role mappings work correctly
				roles := provider.mapGroupsToRoles([]string{"Enterprise Admins"})
				assert.Contains(t, roles, "system-admin")
				assert.Contains(t, roles, "admin")
				assert.Contains(t, roles, "user") // default role
			},
		},
		{
			name: "openldap_development",
			config: &LDAPConfig{
				Host:            "openldap.dev.local",
				Port:            389,
				UseTLS:          true,
				BindDN:          "cn=admin,dc=dev,dc=local",
				BindPassword:    "admin-password",
				BaseDN:          "dc=dev,dc=local",
				UserSearchBase:  "ou=people,dc=dev,dc=local",
				GroupSearchBase: "ou=groups,dc=dev,dc=local",
				RoleMappings: map[string][]string{
					"developers": {"developer", "operator"},
					"admins":     {"admin", "developer", "operator"},
					"ops":        {"operator"},
				},
				DefaultRoles:   []string{"user", "developer"},
				Timeout:        30 * time.Second,
				MaxConnections: 10,
			},
			assertions: func(t *testing.T, provider *LDAPProvider) {
				// Validate OpenLDAP specific settings
				assert.False(t, provider.config.IsActiveDirectory)
				assert.Equal(t, "uid", provider.config.UserAttributes.Username)
				assert.Contains(t, provider.config.UserFilter, "objectClass=inetOrgPerson")
				assert.Contains(t, provider.config.GroupFilter, "objectClass=groupOfNames")

				// Validate role mappings
				roles := provider.mapGroupsToRoles([]string{"developers", "ops"})
				assert.Contains(t, roles, "developer")
				assert.Contains(t, roles, "operator")
				assert.Contains(t, roles, "user") // default role
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
			provider := NewLDAPProvider(tc.config, logger)

			require.NotNil(t, provider)
			tc.assertions(t, provider)
		})
	}
}

// Error handling tests
func TestLDAPProvider_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name              string
		config            *LDAPConfig
		expectPanic       bool
		expectNilProvider bool
	}{
		{
			name:        "nil_config",
			config:      nil,
			expectPanic: true,
		},
		{
			name: "empty_host",
			config: &LDAPConfig{
				Host:   "",
				BaseDN: "dc=example,dc=com",
			},
			expectNilProvider: false, // Provider should still be created but won't connect
		},
		{
			name: "empty_base_dn",
			config: &LDAPConfig{
				Host:   "ldap.example.com",
				BaseDN: "",
			},
			expectNilProvider: false, // Provider should still be created but searches will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

			if tt.expectPanic {
				assert.Panics(t, func() {
					NewLDAPProvider(tt.config, logger)
				})
				return
			}

			provider := NewLDAPProvider(tt.config, logger)

			if tt.expectNilProvider {
				assert.Nil(t, provider)
			} else {
				assert.NotNil(t, provider)
			}
		})
	}
}
