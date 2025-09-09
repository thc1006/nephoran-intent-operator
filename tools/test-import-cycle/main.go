// Package main provides a standalone executable to test import cycles.
package main

import (
	"fmt"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/security"
)

func main() {
	fmt.Println("Testing import cycle resolution...")

	// Test that types from interfaces can be used.
	var _ interfaces.SecretManager
	var _ interfaces.AuditLogger
	var _ interfaces.ConfigProvider
	var _ *interfaces.APIKeys
	var _ *interfaces.RotationResult

	// Test that packages can be instantiated.
	cfg := config.DefaultConfig()
	fmt.Printf("Default config created: %v\n", cfg != nil)

	auditLogger, err := security.NewAuditLogger("", interfaces.AuditLevelInfo)
	if err != nil {
		fmt.Printf("Error creating audit logger: %v\n", err)
	} else {
		fmt.Printf("Audit logger created: %v\n", auditLogger != nil)
		if err := auditLogger.Close(); err != nil {
			fmt.Printf("Warning: Failed to close audit logger: %v\n", err)
		}
	}

	fmt.Println("Import cycle test completed successfully!")
}