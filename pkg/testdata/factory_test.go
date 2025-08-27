package testdata

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestDataFactoryTestSuite demonstrates comprehensive usage of the test data factory
type TestDataFactoryTestSuite struct {
	suite.Suite
	factory *IntentFactory
	golden  *GoldenFile
	tempDir string
}

func TestTestDataFactoryTestSuite(t *testing.T) {
	suite.Run(t, new(TestDataFactoryTestSuite))
}

func (s *TestDataFactoryTestSuite) SetupTest() {
	s.tempDir = s.T().TempDir()
	s.factory = NewIntentFactory(s.tempDir)
	s.golden = NewGoldenFile(s.tempDir)
}

// Test valid intent creation
func (s *TestDataFactoryTestSuite) TestCreateValidIntent() {
	intent := s.factory.CreateValidIntent("test-app")

	// Verify required fields
	s.Assert().Equal("v1", intent.APIVersion)
	s.Assert().Equal("NetworkIntent", intent.Kind)
	s.Assert().Contains(intent.Metadata.Name, "test-app")
	s.Assert().Equal("scale", intent.Spec.Action)
	s.Assert().Equal("deployment", intent.Spec.Target.Type)
	s.Assert().Equal("test-app", intent.Spec.Target.Name)

	// Verify metadata is populated
	s.Assert().NotEmpty(intent.Metadata.Labels)
	s.Assert().NotEmpty(intent.Metadata.Annotations)
	s.Assert().NotEmpty(intent.Metadata.Timestamp)

	// Verify spec has realistic values
	s.Assert().Greater(intent.Spec.Replicas, 0)
	s.Assert().NotEmpty(intent.Spec.Resources)
}

// Test minimal intent creation
func (s *TestDataFactoryTestSuite) TestCreateMinimalValidIntent() {
	intent := s.factory.CreateMinimalValidIntent("minimal-test")

	// Verify only required fields are set
	s.Assert().Equal("v1", intent.APIVersion)
	s.Assert().Equal("NetworkIntent", intent.Kind)
	s.Assert().Contains(intent.Metadata.Name, "minimal-test")
	s.Assert().Equal("scale", intent.Spec.Action)

	// Verify optional fields are empty
	s.Assert().Empty(intent.Metadata.Namespace)
	s.Assert().Empty(intent.Metadata.Labels)
	s.Assert().Zero(intent.Spec.Replicas)
	s.Assert().Empty(intent.Spec.Resources)
}

// Test malformed intent creation
func (s *TestDataFactoryTestSuite) TestCreateMalformedIntent() {
	malformTypes := []string{
		"missing_comma",
		"missing_closing_brace",
		"invalid_json_syntax",
		"trailing_comma",
		"duplicate_keys",
		"incomplete_string",
		"invalid_escape",
	}

	for _, malformType := range malformTypes {
		s.T().Run(malformType, func(t *testing.T) {
			malformed := s.factory.CreateMalformedIntent(malformType)

			// Verify it's invalid JSON
			var temp interface{}
			err := json.Unmarshal(malformed, &temp)
			assert.Error(t, err, "Malformed intent should not be valid JSON: %s", malformType)
		})
	}
}

// Test oversized intent creation
func (s *TestDataFactoryTestSuite) TestCreateOversizedIntent() {
	targetSize := 1024 * 1024 // 1MB
	oversized := s.factory.CreateOversizedIntent(targetSize)

	s.Assert().Greater(len(oversized), targetSize-300, "Should be close to target size")

	// Should still be valid JSON
	var temp interface{}
	err := json.Unmarshal(oversized, &temp)
	s.Require().NoError(err, "Oversized intent should still be valid JSON")
}

// Test suspicious intent creation for security testing
func (s *TestDataFactoryTestSuite) TestCreateSuspiciousIntent() {
	suspiciousTypes := []string{
		"path_traversal",
		"script_injection",
		"sql_injection",
		"command_injection",
		"null_bytes",
	}

	for _, suspiciousType := range suspiciousTypes {
		s.T().Run(suspiciousType, func(t *testing.T) {
			suspicious := s.factory.CreateSuspiciousIntent(suspiciousType)

			// Should be valid JSON
			var temp interface{}
			err := json.Unmarshal(suspicious, &temp)
			assert.NoError(t, err, "Suspicious intent should be valid JSON")

			// Should contain suspicious patterns
			content := string(suspicious)
			switch suspiciousType {
			case "path_traversal":
				assert.Contains(t, content, "../", "Should contain path traversal")
			case "script_injection":
				assert.Contains(t, content, "<script>", "Should contain script tag")
			case "sql_injection":
				assert.Contains(t, content, "DROP TABLE", "Should contain SQL injection")
			case "command_injection":
				assert.Contains(t, content, "rm -rf", "Should contain dangerous command")
			}
		})
	}
}

// Test deep nested intent creation (JSON bomb)
func (s *TestDataFactoryTestSuite) TestCreateDeepNestedIntent() {
	depths := []int{10, 100, 1000}

	for _, depth := range depths {
		s.T().Run(fmt.Sprintf("depth_%d", depth), func(t *testing.T) {
			nested := s.factory.CreateDeepNestedIntent(depth)

			// Should be valid JSON
			var temp interface{}
			err := json.Unmarshal(nested, &temp)
			assert.NoError(t, err, "Deep nested intent should be valid JSON")

			// Should contain expected nesting
			content := string(nested)
			levelCount := strings.Count(content, "level")
			assert.Greater(t, levelCount, depth/2, "Should contain nested levels")
		})
	}
}

// Test file creation
func (s *TestDataFactoryTestSuite) TestCreateIntentFile() {
	intent := s.factory.CreateValidIntent("file-test")

	filePath, err := s.factory.CreateIntentFile("test-intent.json", intent)
	s.Require().NoError(err)
	s.Assert().FileExists(filePath)

	// Verify file contents
	content, err := os.ReadFile(filePath)
	s.Require().NoError(err)

	var loadedIntent IntentData
	err = json.Unmarshal(content, &loadedIntent)
	s.Require().NoError(err)
	s.Assert().Equal(intent.Metadata.Name, loadedIntent.Metadata.Name)
}

// Test batch intent creation
func (s *TestDataFactoryTestSuite) TestCreateBatchIntents() {
	count := 5
	filePaths, err := s.factory.CreateBatchIntents("batch-test", count)
	s.Require().NoError(err)
	s.Assert().Len(filePaths, count)

	// Verify all files exist and have unique names
	names := make(map[string]bool)
	for _, filePath := range filePaths {
		s.Assert().FileExists(filePath)

		content, err := os.ReadFile(filePath)
		s.Require().NoError(err)

		var intent IntentData
		err = json.Unmarshal(content, &intent)
		s.Require().NoError(err)

		// Verify unique names
		s.Assert().False(names[intent.Metadata.Name], "Names should be unique")
		names[intent.Metadata.Name] = true
	}
}

// Test concurrent test files creation
func (s *TestDataFactoryTestSuite) TestCreateConcurrentTestFiles() {
	count := 10
	staggerMs := 50

	filePaths, err := s.factory.CreateConcurrentTestFiles("concurrent", count, staggerMs)
	s.Require().NoError(err)
	s.Assert().Len(filePaths, count)

	// Verify concurrent test annotations
	for i, filePath := range filePaths {
		content, err := os.ReadFile(filePath)
		s.Require().NoError(err)

		var intent IntentData
		err = json.Unmarshal(content, &intent)
		s.Require().NoError(err)

		// Check concurrent test annotations
		concurrentID := intent.Metadata.Annotations["concurrent-test-id"]
		s.Assert().Equal(string(rune('0'+i)), concurrentID)
		s.Assert().NotEmpty(intent.Metadata.Annotations["created-at"])
	}
}

// Test performance test data creation
func (s *TestDataFactoryTestSuite) TestCreatePerformanceTestData() {
	scenarios := []struct {
		name  string
		count int
	}{
		{"small_files", 10},
		{"large_files", 5},
		{"mixed_sizes", 15},
	}

	for _, scenario := range scenarios {
		s.T().Run(scenario.name, func(t *testing.T) {
			filePaths, err := s.factory.CreatePerformanceTestData(scenario.name, scenario.count)
			require.NoError(t, err)
			assert.Len(t, filePaths, scenario.count)

			// Verify all files exist
			for _, filePath := range filePaths {
				assert.FileExists(t, filePath)
			}
		})
	}
}

// Test custom fields intent creation
func (s *TestDataFactoryTestSuite) TestCreateIntentWithCustomFields() {
	customFields := map[string]interface{}{
		"customAction": "deploy",
		"priority":     "high",
		"metadata":     map[string]string{"custom": "value"},
	}

	intent := s.factory.CreateIntentWithCustomFields("custom-test", customFields)

	// Verify base fields are still present
	s.Assert().Equal("v1", intent.APIVersion)
	s.Assert().Equal("NetworkIntent", intent.Kind)

	// Marshal to check custom fields were added
	data, err := json.Marshal(intent.Spec)
	s.Require().NoError(err)
	content := string(data)

	s.Assert().Contains(content, "customAction")
	s.Assert().Contains(content, "priority")
}

// Test golden file functionality
func (s *TestDataFactoryTestSuite) TestGoldenFileIntegration() {
	intent := s.factory.CreateValidIntent("golden-test")

	// Test saving golden file (only when UPDATE_GOLDEN=true)
	s.golden.SaveGoldenJSON(s.T(), "test-intent.json", intent)

	// Test loading golden file
	if os.Getenv("UPDATE_GOLDEN") != "true" {
		// Only test loading if we're not updating
		var loadedIntent IntentData
		s.golden.LoadGoldenJSON(s.T(), "test-intent.json", &loadedIntent)

		// Compare with golden
		s.golden.CompareJSONWithGolden(s.T(), "test-intent.json", intent)
	}
}

// Test factory reset and statistics
func (s *TestDataFactoryTestSuite) TestFactoryManagement() {
	// Create some intents to increment counter
	s.factory.CreateValidIntent("test-1")
	s.factory.CreateValidIntent("test-2")
	s.factory.CreateValidIntent("test-3")

	initialCount := s.factory.GetCreatedFileCount()
	s.Assert().Equal(3, initialCount)

	// Reset factory
	s.factory.Reset()
	resetCount := s.factory.GetCreatedFileCount()
	s.Assert().Equal(0, resetCount)

	// Create more intents after reset
	s.factory.CreateValidIntent("test-after-reset")
	afterResetCount := s.factory.GetCreatedFileCount()
	s.Assert().Equal(1, afterResetCount)
}

// Benchmark tests for performance measurement
func BenchmarkCreateValidIntent(b *testing.B) {
	factory := NewIntentFactory(b.TempDir())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.CreateValidIntent("benchmark-test")
	}
}

func BenchmarkCreateIntentFile(b *testing.B) {
	factory := NewIntentFactory(b.TempDir())
	intent := factory.CreateValidIntent("benchmark-file-test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.CreateIntentFile(fmt.Sprintf("bench-file-%d.json", i), intent)
	}
}

func BenchmarkCreateBatchIntents(b *testing.B) {
	factory := NewIntentFactory(b.TempDir())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.CreateBatchIntents(fmt.Sprintf("bench-batch-%d", i), 10)
	}
}

// Example usage demonstrating realistic test scenarios
func ExampleIntentFactory_usage() {
	tempDir := "/tmp/test-data"
	factory := NewIntentFactory(tempDir)

	// Create a valid intent for positive testing
	validIntent := factory.CreateValidIntent("example-app")
	fmt.Printf("Created valid intent: %s\n", validIntent.Metadata.Name)

	// Create malformed intent for error handling testing
	malformed := factory.CreateMalformedIntent("missing_comma")
	fmt.Printf("Created malformed intent: %d bytes\n", len(malformed))

	// Create batch of intents for load testing
	filePaths, _ := factory.CreateBatchIntents("load-test", 100)
	fmt.Printf("Created %d test files for load testing\n", len(filePaths))

	// Create performance test data
	perfFiles, _ := factory.CreatePerformanceTestData("mixed_sizes", 50)
	fmt.Printf("Created %d performance test files\n", len(perfFiles))

	// Clean up
	factory.CleanupFiles()
	fmt.Println("Cleaned up test files")
}
