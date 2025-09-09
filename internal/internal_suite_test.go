package internal_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInternal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Internal Suite")
}

var _ = Describe("Internal Package", func() {
	Context("when bootstrapping", func() {
		It("should initialize successfully", func() {
			Expect(true).To(BeTrue())
		})
	})
})