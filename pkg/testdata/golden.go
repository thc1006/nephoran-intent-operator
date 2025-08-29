
package testdata



import (

	"encoding/json"

	"os"

	"path/filepath"

	"testing"



	"github.com/stretchr/testify/require"

)



// GoldenFile manages golden file testing for consistent test outcomes.

type GoldenFile struct {

	baseDir string

}



// NewGoldenFile creates a new golden file manager.

func NewGoldenFile(baseDir string) *GoldenFile {

	return &GoldenFile{

		baseDir: baseDir,

	}

}



// SaveGoldenFile saves test data as a golden file for future comparison.

func (g *GoldenFile) SaveGoldenFile(t *testing.T, filename string, data []byte) {

	goldenPath := filepath.Join(g.baseDir, "testdata", "golden", filename)



	// Only update golden files when explicitly requested.

	if os.Getenv("UPDATE_GOLDEN") == "true" {

		// Ensure directory exists.

		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))

		require.NoError(t, os.WriteFile(goldenPath, data, 0640))

		t.Logf("Updated golden file: %s", goldenPath)

	}

}



// LoadGoldenFile loads the expected golden file content.

func (g *GoldenFile) LoadGoldenFile(t *testing.T, filename string) []byte {

	goldenPath := filepath.Join(g.baseDir, "testdata", "golden", filename)

	data, err := os.ReadFile(goldenPath)

	require.NoError(t, err, "Golden file not found: %s\nRun tests with UPDATE_GOLDEN=true to create", goldenPath)

	return data

}



// SaveGoldenJSON saves JSON data with proper formatting.

func (g *GoldenFile) SaveGoldenJSON(t *testing.T, filename string, data interface{}) {

	jsonData, err := json.MarshalIndent(data, "", "  ")

	require.NoError(t, err)

	g.SaveGoldenFile(t, filename, jsonData)

}



// LoadGoldenJSON loads and unmarshals JSON golden file.

func (g *GoldenFile) LoadGoldenJSON(t *testing.T, filename string, target interface{}) {

	data := g.LoadGoldenFile(t, filename)

	require.NoError(t, json.Unmarshal(data, target))

}



// CompareWithGolden compares actual data with golden file.

func (g *GoldenFile) CompareWithGolden(t *testing.T, filename string, actual []byte) {

	if os.Getenv("UPDATE_GOLDEN") == "true" {

		g.SaveGoldenFile(t, filename, actual)

		return

	}



	expected := g.LoadGoldenFile(t, filename)

	require.Equal(t, string(expected), string(actual),

		"Output differs from golden file %s\nRun with UPDATE_GOLDEN=true to update", filename)

}



// CompareJSONWithGolden compares JSON data with golden file.

func (g *GoldenFile) CompareJSONWithGolden(t *testing.T, filename string, actual interface{}) {

	actualJSON, err := json.MarshalIndent(actual, "", "  ")

	require.NoError(t, err)



	if os.Getenv("UPDATE_GOLDEN") == "true" {

		g.SaveGoldenJSON(t, filename, actual)

		return

	}



	var expected interface{}

	g.LoadGoldenJSON(t, filename, &expected)



	expectedJSON, err := json.MarshalIndent(expected, "", "  ")

	require.NoError(t, err)



	require.Equal(t, string(expectedJSON), string(actualJSON),

		"JSON output differs from golden file %s\nRun with UPDATE_GOLDEN=true to update", filename)

}



// GoldenTestSuite provides common golden file testing methods.

type GoldenTestSuite struct {

	Golden *GoldenFile

}



// SetupGoldenTestSuite initializes the golden test suite.

func SetupGoldenTestSuite(baseDir string) *GoldenTestSuite {

	return &GoldenTestSuite{

		Golden: NewGoldenFile(baseDir),

	}

}



// AssertGoldenFile is a helper for common golden file assertions.

func (s *GoldenTestSuite) AssertGoldenFile(t *testing.T, filename string, actual []byte) {

	s.Golden.CompareWithGolden(t, filename, actual)

}



// AssertGoldenJSON is a helper for JSON golden file assertions.

func (s *GoldenTestSuite) AssertGoldenJSON(t *testing.T, filename string, actual interface{}) {

	s.Golden.CompareJSONWithGolden(t, filename, actual)

}

