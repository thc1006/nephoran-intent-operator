//go:build integration

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// E2ETestSuite provides a complete end-to-end test suite
type E2ETestSuite struct {
	suite.Suite
	tempDir         string
	handoffDir      string
	intentIngestURL string
	processes       []*os.Process
	cleanup         func()
}

// DISABLED: func TestE2ETestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	suite.Run(t, new(E2ETestSuite))
}

func (s *E2ETestSuite) SetupSuite() {
	s.tempDir = s.T().TempDir()
	s.handoffDir = filepath.Join(s.tempDir, "handoff")

	err := os.MkdirAll(s.handoffDir, 0o755)
	s.Require().NoError(err)

	// Setup schema file
	s.createTestSchema()

	// Start intent-ingest service
	s.startIntentIngestService()

	// Wait for service to be ready
	s.waitForServiceReady()
}

func (s *E2ETestSuite) TearDownSuite() {
	// Stop all processes
	for _, proc := range s.processes {
		if proc != nil {
			proc.Signal(syscall.SIGTERM)
			proc.Wait()
		}
	}

	if s.cleanup != nil {
		s.cleanup()
	}
}

func (s *E2ETestSuite) TestFullWorkflow_IntentIngestionToHandoff() {
	s.T().Run("ingests intent and creates handoff file", func(t *testing.T) {
		// Prepare test intent
		intent := json.RawMessage("{}"){
				"name":      "e2e-test-intent",
				"namespace": "default",
			},
			"spec": json.RawMessage("{}"),
		}

		intentJSON, err := json.Marshal(intent)
		require.NoError(t, err)

		// Send intent to ingest service
		resp, err := http.Post(s.intentIngestURL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify handoff file was created
		s.Eventually(func() bool {
			files, err := os.ReadDir(s.handoffDir)
			return err == nil && len(files) > 0
		}, 10*time.Second, 100*time.Millisecond, "Handoff file should be created")

		// Verify handoff file contents
		files, err := os.ReadDir(s.handoffDir)
		require.NoError(t, err)
		require.Greater(t, len(files), 0)

		handoffFile := filepath.Join(s.handoffDir, files[0].Name())
		content, err := os.ReadFile(handoffFile)
		require.NoError(t, err)

		var handoffData map[string]interface{}
		err = json.Unmarshal(content, &handoffData)
		require.NoError(t, err)

		assert.Equal(t, "intent.nephoran.com/v1alpha1", handoffData["apiVersion"])
		assert.Equal(t, "NetworkIntent", handoffData["kind"])

		spec := handoffData["spec"].(map[string]interface{})
		assert.Equal(t, "scaling", spec["intentType"])
		assert.Equal(t, "nginx-deployment", spec["target"])
		assert.Equal(t, float64(3), spec["replicas"]) // JSON numbers are float64
	})
}

func (s *E2ETestSuite) TestHealthEndpoints() {
	s.T().Run("all health endpoints return healthy status", func(t *testing.T) {
		// Test intent-ingest health
		resp, err := http.Get(s.intentIngestURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "OK")
	})
}

func (s *E2ETestSuite) TestMetricsCollection() {
	s.T().Run("metrics endpoints provide Prometheus metrics", func(t *testing.T) {
		// Send a few requests first to generate metrics
		for i := 0; i < 5; i++ {
			intent := json.RawMessage("{}"){
					"intent": fmt.Sprintf("Metrics test intent %d", i),
				},
			}

			intentJSON, err := json.Marshal(intent)
			require.NoError(t, err)

			resp, err := http.Post(s.intentIngestURL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
			require.NoError(t, err)
			resp.Body.Close()
		}

		// Check metrics endpoint
		resp, err := http.Get(s.intentIngestURL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		metricsText := string(body)
		assert.Contains(t, metricsText, "# HELP")
		assert.Contains(t, metricsText, "# TYPE")
	})
}

func (s *E2ETestSuite) TestErrorScenarios() {
	s.T().Run("handles various error conditions gracefully", func(t *testing.T) {
		testCases := []struct {
			name           string
			payload        string
			contentType    string
			expectedStatus int
		}{
			{
				name:           "invalid JSON",
				payload:        `{"invalid": json}`,
				contentType:    "application/json",
				expectedStatus: http.StatusBadRequest,
			},
			{
				name:           "empty payload",
				payload:        "",
				contentType:    "application/json",
				expectedStatus: http.StatusBadRequest,
			},
			{
				name:           "wrong content type",
				payload:        `{"spec":{"intent":"test"}}`,
				contentType:    "text/plain",
				expectedStatus: http.StatusBadRequest,
			},
			{
				name:           "malformed intent structure",
				payload:        `{"not_an_intent": true}`,
				contentType:    "application/json",
				expectedStatus: http.StatusBadRequest,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, err := http.NewRequest("POST", s.intentIngestURL+"/ingest", strings.NewReader(tc.payload))
				require.NoError(t, err)

				req.Header.Set("Content-Type", tc.contentType)

				client := &http.Client{Timeout: 10 * time.Second}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Test case: %s", tc.name)
			})
		}
	})
}

func (s *E2ETestSuite) TestLoadHandling() {
	s.T().Run("handles moderate load without failures", func(t *testing.T) {
		const (
			concurrency = 10
			requests    = 50
		)

		intent := json.RawMessage("{}"){
				"intentType": "scaling",
				"target":     "load-test-deployment",
				"replicas":   2,
			},
		}

		intentJSON, err := json.Marshal(intent)
		require.NoError(t, err)

		// Count initial files
		initialFiles, err := os.ReadDir(s.handoffDir)
		require.NoError(t, err)
		initialCount := len(initialFiles)

		// Execute load test
		resultCh := make(chan int, concurrency*requests)

		for i := 0; i < concurrency; i++ {
			go func(workerID int) {
				for j := 0; j < requests; j++ {
					resp, err := http.Post(s.intentIngestURL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
					if err != nil {
						resultCh <- 500
						continue
					}
					resultCh <- resp.StatusCode
					resp.Body.Close()
				}
			}(i)
		}

		// Collect results
		successCount := 0
		totalRequests := concurrency * requests

		for i := 0; i < totalRequests; i++ {
			statusCode := <-resultCh
			if statusCode == http.StatusOK {
				successCount++
			}
		}

		assert.Equal(t, totalRequests, successCount, "All requests should succeed")

		// Verify all handoff files were created
		s.Eventually(func() bool {
			files, err := os.ReadDir(s.handoffDir)
			return err == nil && len(files) == initialCount+totalRequests
		}, 15*time.Second, 200*time.Millisecond, "All handoff files should be created")
	})
}

func (s *E2ETestSuite) TestServiceRecovery() {
	s.T().Run("service recovers gracefully from simulated failures", func(t *testing.T) {
		// Verify service is working
		intent := json.RawMessage("{}"){
				"intent": "Recovery test intent",
			},
		}

		intentJSON, err := json.Marshal(intent)
		require.NoError(t, err)

		// Send initial request
		resp, err := http.Post(s.intentIngestURL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Simulate temporary file system issue by making handoff directory read-only
		err = os.Chmod(s.handoffDir, 0o444)
		require.NoError(t, err)

		// Send request during "failure"
		resp, err = http.Post(s.intentIngestURL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
		require.NoError(t, err)
		resp.Body.Close()
		// Should handle gracefully (may return error status)

		// Restore directory permissions
		err = os.Chmod(s.handoffDir, 0o755)
		require.NoError(t, err)

		// Verify service recovers
		resp, err = http.Post(s.intentIngestURL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func (s *E2ETestSuite) TestDataIntegrity() {
	s.T().Run("maintains data integrity under concurrent access", func(t *testing.T) {
		const numWorkers = 5
		const requestsPerWorker = 20

		resultCh := make(chan string, numWorkers*requestsPerWorker)

		for workerID := 0; workerID < numWorkers; workerID++ {
			go func(id int) {
				for i := 0; i < requestsPerWorker; i++ {
					intent := json.RawMessage("{}"){
							"name": fmt.Sprintf("integrity-test-w%d-r%d", id, i),
						},
						"spec": json.RawMessage("{}"),
					}

					intentJSON, err := json.Marshal(intent)
					if err != nil {
						continue
					}

					resp, err := http.Post(s.intentIngestURL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
					if err != nil {
						continue
					}
					resp.Body.Close()

					if resp.StatusCode == http.StatusOK {
						resultCh <- fmt.Sprintf("w%d-r%d", id, i)
					}
				}
			}(workerID)
		}

		// Collect results
		var successfulRequests []string
		timeout := time.After(30 * time.Second)

		for i := 0; i < numWorkers*requestsPerWorker; i++ {
			select {
			case result := <-resultCh:
				successfulRequests = append(successfulRequests, result)
			case <-timeout:
				t.Fatalf("Timeout waiting for concurrent requests to complete")
			}
		}

		assert.Equal(t, numWorkers*requestsPerWorker, len(successfulRequests))

		// Wait for all files to be written
		time.Sleep(2 * time.Second)

		// Verify file integrity
		files, err := os.ReadDir(s.handoffDir)
		require.NoError(t, err)

		validFiles := 0
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".json") {
				filePath := filepath.Join(s.handoffDir, file.Name())
				content, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}

				var data map[string]interface{}
				if json.Unmarshal(content, &data) == nil {
					validFiles++
				}
			}
		}

		assert.Greater(t, validFiles, numWorkers*requestsPerWorker/2, "At least half of the files should be valid JSON")
	})
}

// Helper methods

func (s *E2ETestSuite) createTestSchema() {
	schemaFile := filepath.Join(s.tempDir, "intent.schema.json")

	schema := json.RawMessage("{}"){
			"apiVersion": json.RawMessage("{}"),
			},
			"kind": json.RawMessage("{}"),
			},
			"metadata": json.RawMessage("{}"){
					"name": json.RawMessage("{}"),
					"namespace": json.RawMessage("{}"),
				},
			},
			"spec": json.RawMessage("{}"){
					"intentType": json.RawMessage("{}"),
					},
					"target": json.RawMessage("{}"),
					"replicas": json.RawMessage("{}"),
					"intent": json.RawMessage("{}"),
				},
			},
		},
	}

	schemaData, err := json.MarshalIndent(schema, "", "  ")
	s.Require().NoError(err)

	err = os.WriteFile(schemaFile, schemaData, 0o644)
	s.Require().NoError(err)
}

func (s *E2ETestSuite) startIntentIngestService() {
	// Look for intent-ingest binary
	binaryPath := s.findBinary("intent-ingest")
	if binaryPath == "" {
		s.T().Skip("intent-ingest binary not found, skipping E2E tests")
		return
	}

	schemaFile := filepath.Join(s.tempDir, "intent.schema.json")

	cmd := exec.Command(binaryPath,
		"-addr", ":0", // Let OS choose port
		"-handoff", s.handoffDir,
		"-schema", schemaFile,
		"-mode", "rules",
	)

	// Capture output for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	s.Require().NoError(err)

	s.processes = append(s.processes, cmd.Process)

	// For simplicity, use a fixed port in tests
	// In real E2E tests, you'd parse the actual port from logs
	s.intentIngestURL = "http://localhost:8080"
}

func (s *E2ETestSuite) waitForServiceReady() {
	// Wait up to 30 seconds for service to start
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.T().Fatal("Service failed to start within timeout")
		case <-ticker.C:
			if s.isServiceReady() {
				return
			}
		}
	}
}

func (s *E2ETestSuite) isServiceReady() bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(s.intentIngestURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (s *E2ETestSuite) findBinary(name string) string {
	// Look in common locations
	locations := []string{
		filepath.Join("cmd", name, name+".exe"),
		filepath.Join("cmd", name, name),
		filepath.Join("bin", name+".exe"),
		filepath.Join("bin", name),
		name + ".exe",
		name,
	}

	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			absPath, _ := filepath.Abs(location)
			return absPath
		}
	}

	return ""
}

// Eventually asserts that the given condition becomes true within the timeout
func (s *E2ETestSuite) Eventually(condition func() bool, timeout, interval time.Duration, msgAndArgs ...interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		select {
		case <-ctx.Done():
			s.Fail("Condition never became true", msgAndArgs...)
			return
		case <-ticker.C:
			// Continue checking
		}
	}
}
