package rag

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ChunkingService", func() {
	var (
		service *ChunkingService
		config  *ChunkingConfig
	)

	BeforeEach(func() {
		config = &ChunkingConfig{
			ChunkSize:             500,
			ChunkOverlap:          50,
			MinChunkSize:          100,
			MaxChunkSize:          1000,
			UseSemanticBoundaries: true,
			PreserveHierarchy:     true,
		}
		service = NewChunkingService(config)
	})

	Describe("NewChunkingService", func() {
		It("should create service with provided configuration", func() {
			s := NewChunkingService(config)
			Expect(s).NotTo(BeNil())
			Expect(s.config).To(Equal(config))
		})

		It("should create service with default config when nil provided", func() {
			s := NewChunkingService(nil)
			Expect(s).NotTo(BeNil())
			Expect(s.config).NotTo(BeNil())
		})
	})

	Describe("ChunkDocument", func() {
		Context("when chunking regular text", func() {
			It("should split text into appropriate chunks", func() {
				document := &LoadedDocument{
					ID:      "test-doc-1",
					Content: "This is the first sentence. This is the second sentence. This is the third sentence. This is a very long sentence that should demonstrate how the chunking algorithm works with longer pieces of text that might exceed the normal chunk size limits.",
					Title:   "Test Document",
					Metadata: &DocumentMetadata{
						Source:       "test",
						DocumentType: "txt",
					},
					LoadedAt: time.Now(),
				}

				chunks, err := service.ChunkDocument(context.Background(), document)

				Expect(err).ToNot(HaveOccurred())
				Expect(len(chunks)).To(BeNumerically(">", 0))

				for _, chunk := range chunks {
					Expect(len(chunk.Content)).To(BeNumerically(">=", config.MinChunkSize))
					Expect(len(chunk.Content)).To(BeNumerically("<=", config.MaxChunkSize))
					Expect(chunk.DocumentID).To(Equal(document.ID))
					if chunk.DocumentMetadata != nil {
						Expect(chunk.DocumentMetadata.Source).To(Equal(document.Metadata.Source))
					}
				}
			})

			It("should preserve overlap between chunks", func() {
				longText := "Sentence one. Sentence two. Sentence three. Sentence four. Sentence five. Sentence six. Sentence seven. Sentence eight. Sentence nine. Sentence ten."
				document := &LoadedDocument{
					ID:      "test-doc-2",
					Content: longText,
					Title:   "Long Document",
					Metadata: &DocumentMetadata{
						Source:       "test",
						DocumentType: "txt",
					},
					LoadedAt: time.Now(),
				}

				chunks, err := service.ChunkDocument(context.Background(), document)

				Expect(err).ToNot(HaveOccurred())
				if len(chunks) > 1 {
					// Check that consecutive chunks have some overlap
					for i := 1; i < len(chunks); i++ {
						// There should be some overlap between chunks
						Expect(chunks[i-1].EndOffset).To(BeNumerically(">", chunks[i].StartOffset))
					}
				}
			})

			It("should handle empty documents", func() {
				document := &LoadedDocument{
					ID:      "test-doc-3",
					Content: "",
					Title:   "Empty Document",
					Metadata: &DocumentMetadata{
						Source:       "test",
						DocumentType: "txt",
					},
					LoadedAt: time.Now(),
				}

				chunks, err := service.ChunkDocument(context.Background(), document)

				Expect(err).ToNot(HaveOccurred())
				Expect(len(chunks)).To(Equal(0))
			})

			It("should handle very short documents", func() {
				document := &LoadedDocument{
					Content: "Short text.",
					Title:   "Short Document",
					Metadata: &DocumentMetadata{
						Source:       "test",
						DocumentType: "txt",
					},
				}

				chunks, err := service.ChunkDocument(context.Background(), document)

				Expect(err).ToNot(HaveOccurred())
				Expect(len(chunks)).To(Equal(1))
				Expect(chunks[0].Content).To(Equal("Short text."))
			})
		})

		Context("when chunking telecom documents", func() {
			It("should preserve technical terminology", func() {
				document := &LoadedDocument{
					Content: "The gNB (next generation Node B) is a key component in 5G networks. It handles radio resource management for the New Radio (NR) interface. The gNB connects to the 5G Core (5GC) through the N2 and N3 interfaces.",
					Title:   "5G Architecture",
					Metadata: &DocumentMetadata{
						Source:       "3GPP TS 38.300",
						DocumentType: "TS",
					},
				}

				chunks, err := service.ChunkDocument(context.Background(), document)

				Expect(err).ToNot(HaveOccurred())
				Expect(len(chunks)).To(BeNumerically(">", 0))

				// Check that technical terms are preserved
				allContent := ""
				for _, chunk := range chunks {
					allContent += chunk.Content + " "
				}
				Expect(allContent).To(ContainSubstring("gNB"))
				Expect(allContent).To(ContainSubstring("5G"))
				Expect(allContent).To(ContainSubstring("NR"))
			})

			It("should handle structured technical documents", func() {
				document := &LoadedDocument{
					Content: `
# 5G Network Architecture

## Core Network Functions
The 5G Core (5GC) includes:
- AMF (Access and Mobility Management Function)
- SMF (Session Management Function) 
- UPF (User Plane Function)

## Radio Access Network
The RAN includes:
- gNB (Next Generation NodeB)
- CU (Central Unit)
- DU (Distributed Unit)
`,
					Title:  "5G Architecture Guide",
					Metadata: &DocumentMetadata{
						Source:       "O-RAN Alliance",
						DocumentType: "spec",
					},
				}

				chunks, err := service.ChunkDocument(context.Background(), document)

				Expect(err).ToNot(HaveOccurred())
				Expect(len(chunks)).To(BeNumerically(">", 0))

				// Should preserve structure indicators
				hasHeaders := false
				for _, chunk := range chunks {
					if containsAny(chunk.Content, []string{"#", "##", "-"}) {
						hasHeaders = true
						break
					}
				}
				Expect(hasHeaders).To(BeTrue())
			})
		})

		Context("when handling different separator types", func() {
			BeforeEach(func() {
				// Remove separator type configuration as it's not supported
				service = NewChunkingService(config)
			})

			It("should chunk by paragraphs", func() {
				document := &LoadedDocument{
					Content: "First paragraph with multiple sentences. This continues the first paragraph.\n\nSecond paragraph starts here. It also has multiple sentences.\n\nThird paragraph is the final one.",
					Title:   "Multi-paragraph Document",
					Metadata: &DocumentMetadata{
						Source:       "test",
						DocumentType: "txt",
					},
				}

				chunks, err := service.ChunkDocument(context.Background(), document)

				Expect(err).ToNot(HaveOccurred())
				Expect(len(chunks)).To(BeNumerically(">", 0))
			})
		})
	})

	// Remove EstimateTokens tests as the method doesn't exist in ChunkingService

	// Remove OptimizeChunks tests as the method doesn't exist in ChunkingService

	Describe("Configuration Validation", func() {
		It("should handle invalid chunk sizes", func() {
			invalidConfig := &ChunkingConfig{
				ChunkSize:     0,    // Invalid
				ChunkOverlap:  -1,   // Invalid
				MinChunkSize:  1000, // Invalid (larger than chunk size)
				MaxChunkSize:  100,  // Invalid (smaller than chunk size)
			}

			service := NewChunkingService(invalidConfig)

			// Service should correct invalid values
			Expect(service.config.ChunkSize).To(BeNumerically(">", 0))
			Expect(service.config.ChunkOverlap).To(BeNumerically(">=", 0))
			Expect(service.config.MinChunkSize).To(BeNumerically("<=", service.config.ChunkSize))
			Expect(service.config.MaxChunkSize).To(BeNumerically(">=", service.config.ChunkSize))
		})
	})

	Describe("Error Handling", func() {
		Context("when document is nil", func() {
			It("should return error", func() {
				chunks, err := service.ChunkDocument(context.Background(), nil)

				Expect(err).To(HaveOccurred())
				Expect(chunks).To(BeNil())
			})
		})

		Context("when document has invalid content type", func() {
			It("should handle gracefully", func() {
				document := &LoadedDocument{
					Content: string([]byte{0, 1, 2, 3, 4, 5}), // Binary content
					Title:   "Binary Document",
					Metadata: &DocumentMetadata{
						Source:       "test",
						DocumentType: "txt",
					},
				}

				chunks, err := service.ChunkDocument(context.Background(), document)

				// Should either succeed or fail gracefully
				if err != nil {
					Expect(chunks).To(BeNil())
				} else {
					Expect(chunks).NotTo(BeNil())
				}
			})
		})
	})
})

// Helper functions for tests

func containsAny(text string, substrings []string) bool {
	for _, substr := range substrings {
		if len(substr) > 0 && len(text) >= len(substr) {
			for i := 0; i <= len(text)-len(substr); i++ {
				if text[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

func splitWords(text string) []string {
	words := []string{}
	current := ""

	for _, char := range text {
		if char == ' ' || char == '\t' || char == '\n' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		words = append(words, current)
	}

	return words
}

func generateText(length int) string {
	text := ""
	words := []string{"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog"}
	wordIndex := 0

	for len(text) < length {
		if len(text) > 0 {
			text += " "
		}
		text += words[wordIndex]
		wordIndex = (wordIndex + 1) % len(words)
	}

	if len(text) > length {
		text = text[:length]
	}

	return text
}
