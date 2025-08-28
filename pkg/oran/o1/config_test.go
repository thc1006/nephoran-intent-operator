package o1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestO1Config(t *testing.T) {
	testCases := []struct {
		name           string
		config         *O1Config
		expectedError  bool
		errorMsg       string
	}{
		{
			name: "Valid TLS Config with CA File",
			config: &O1Config{
				Address:       "localhost",
				Port:          8080,
				Username:      "admin",
				Password:      "password",
				RetryInterval: 5 * time.Second,
				TLSConfig: &TLSConfig{
					Enabled:    true,
					CAFile:     "/path/to/ca.crt",
					SkipVerify: false,
				},
			},
			expectedError: false,
		},
		{
			name: "Valid TLS Config with Skip Verify",
			config: &O1Config{
				Address:       "localhost",
				Port:          8080,
				Username:      "admin",
				Password:      "password",
				RetryInterval: 5 * time.Second,
				TLSConfig: &TLSConfig{
					Enabled:     true,
					SkipVerify:  true,
					MinVersion:  "1.3",
				},
			},
			expectedError: false,
		},
		{
			name: "Missing CA File without Skip Verify",
			config: &O1Config{
				Address:       "localhost",
				Port:          8080,
				Username:      "admin",
				Password:      "password",
				RetryInterval: 5 * time.Second,
				TLSConfig: &TLSConfig{
					Enabled:    true,
					SkipVerify: false,
				},
			},
			expectedError: true,
			errorMsg:      "CA file is required when skip verify is false",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			securityManager, err := NewO1SecurityManager(tc.config)
			
			if tc.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, securityManager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, securityManager)
				
				// Validate specific TLS config
				if tc.config.TLSConfig != nil && tc.config.TLSConfig.Enabled {
					assert.NotNil(t, securityManager.TLSConfig)
					
					if tc.config.TLSConfig.MinVersion == "1.3" {
						assert.Equal(t, securityManager.TLSConfig.MinVersion, uint16(0x0304))
					}
				}
			}
		})
	}
}