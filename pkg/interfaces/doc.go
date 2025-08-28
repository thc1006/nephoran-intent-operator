// Package interfaces defines common interfaces used across the Nephoran Intent Operator.
// to break import cycles and enable dependency injection.
//
// This package contains:.
// - SecretManager: Interface for secure secret operations.
// - AuditLogger: Interface for security audit logging.
// - ConfigProvider: Interface for configuration access.
// - Common types like APIKeys and RotationResult used across packages.
//
// By defining these interfaces separately, we avoid circular dependencies.
// between the config and security packages while maintaining clean.
// separation of concerns.
package interfaces
