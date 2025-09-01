package auth

import "os"

// setMultipleEnvVars sets multiple environment variables and returns a cleanup function
func setMultipleEnvVars(envVars map[string]string) func() {
	originalValues := make(map[string]string)
	
	// Set new values and store originals
	for key, value := range envVars {
		originalValues[key] = os.Getenv(key)
		os.Setenv(key, value)
	}
	
	// Return cleanup function
	return func() {
		for key, originalValue := range originalValues {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}
}