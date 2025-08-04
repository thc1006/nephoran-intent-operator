package security

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSecurity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Security Suite")
}

var _ = BeforeSuite(func() {
	// Any global setup for security tests
})

var _ = AfterSuite(func() {
	// Any global cleanup for security tests
})