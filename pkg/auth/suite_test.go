package auth

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestAuth bootstraps Ginkgo v2 test suite for the auth package.
// This package implements authentication, authorization, and security features
// including JWT, OAuth2, LDAP, and MFA capabilities for O-RAN deployments.
// RegisterFailHandler(Fail) wires Gomega assertions to Ginkgo's failure handler.
func TestAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authentication Suite")
}