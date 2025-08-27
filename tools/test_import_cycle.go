// FIXME: Adding package comment per revive linter
// Package main provides import cycle detection utilities for the codebase
package main

import (
	"fmt"

	// Test that both packages can be imported without causing import cycle
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/security"
)

func main() {
	fmt.Println("Testing import cycle resolution...")

	// Test that types from interfaces can be used
	var _ interfaces.SecretManager
	var _ interfaces.AuditLogger
	var _ interfaces.ConfigProvider
	var _ *interfaces.APIKeys
	var _ *interfaces.RotationResult

	// Test that packages can be instantiated
	cfg := config.DefaultConfig()
	fmt.Printf("Default config created: %v\n", cfg != nil)

	auditLogger, err := security.NewAuditLogger("", interfaces.AuditLevelInfo)
	if err != nil {
		fmt.Printf("Error creating audit logger: %v\n", err)
	} else {
		fmt.Printf("Audit logger created: %v\n", auditLogger != nil)
		auditLogger.Close()
	}

	fmt.Println("Import cycle test completed successfully!")
}
