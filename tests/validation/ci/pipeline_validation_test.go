//go:build ci

/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

// GitHubWorkflow represents a GitHub Actions workflow
type GitHubWorkflow struct {
	Name string `yaml:"name"`
	On   struct {
		Push struct {
			Branches     []string `yaml:"branches,omitempty"`
			PathsIgnore  []string `yaml:"paths-ignore,omitempty"`
		} `yaml:"push,omitempty"`
		PullRequest struct {
			Branches []string `yaml:"branches,omitempty"`
		} `yaml:"pull_request,omitempty"`
		WorkflowDispatch interface{} `yaml:"workflow_dispatch,omitempty"`
	} `yaml:"on"`
	Concurrency struct {
		Group            string `yaml:"group"`
		CancelInProgress bool   `yaml:"cancel-in-progress"`
	} `yaml:"concurrency,omitempty"`
	Jobs map[string]GitHubJob `yaml:"jobs"`
}

// GitHubJob represents a job in a GitHub Actions workflow
type GitHubJob struct {
	RunsOn   interface{}       `yaml:"runs-on"`
	Strategy *GitHubStrategy   `yaml:"strategy,omitempty"`
	Steps    []GitHubStep      `yaml:"steps"`
	Needs    interface{}       `yaml:"needs,omitempty"`
	If       string            `yaml:"if,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
}

// GitHubStrategy represents a job strategy
type GitHubStrategy struct {
	Matrix map[string]interface{} `yaml:"matrix,omitempty"`
}

// GitHubStep represents a step in a GitHub Actions job
type GitHubStep struct {
	Name string            `yaml:"name,omitempty"`
	Uses string            `yaml:"uses,omitempty"`
	Run  string            `yaml:"run,omitempty"`
	With map[string]string `yaml:"with,omitempty"`
	Env  map[string]string `yaml:"env,omitempty"`
}

// Makefile represents basic Makefile structure for validation
type Makefile struct {
	Content string
	Targets []string
}

// CIPipelineValidationTestSuite provides comprehensive CI/CD pipeline validation
type CIPipelineValidationTestSuite struct {
	suite.Suite
	workflowDir     string
	workflows       map[string]*GitHubWorkflow
	makefile        *Makefile
	projectRoot     string
}

// SetupSuite initializes the CI pipeline validation test environment
func (suite *CIPipelineValidationTestSuite) SetupSuite() {
	suite.projectRoot = filepath.Join("..", "..", "..")
	suite.workflowDir = filepath.Join(suite.projectRoot, ".github", "workflows")
	suite.workflows = make(map[string]*GitHubWorkflow)

	// Load GitHub Actions workflows
	suite.loadGitHubWorkflows()

	// Load Makefile
	suite.loadMakefile()
}

// loadGitHubWorkflows loads all GitHub Actions workflow files
func (suite *CIPipelineValidationTestSuite) loadGitHubWorkflows() {
	if _, err := os.Stat(suite.workflowDir); os.IsNotExist(err) {
		suite.T().Skip("GitHub workflows directory not found")
		return
	}

	files, err := filepath.Glob(filepath.Join(suite.workflowDir, "*.yml"))
	if err != nil {
		suite.T().Skipf("Failed to read workflow directory: %v", err)
		return
	}

	yamlFiles, err := filepath.Glob(filepath.Join(suite.workflowDir, "*.yaml"))
	if err == nil {
		files = append(files, yamlFiles...)
	}

	for _, file := range files {
		workflow, err := suite.loadWorkflowFromFile(file)
		if err != nil {
			suite.T().Logf("Warning: Failed to load workflow from %s: %v", file, err)
			continue
		}
		
		filename := filepath.Base(file)
		suite.workflows[filename] = workflow
	}
}

// loadWorkflowFromFile loads a GitHub Actions workflow from a YAML file
func (suite *CIPipelineValidationTestSuite) loadWorkflowFromFile(filename string) (*GitHubWorkflow, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var workflow GitHubWorkflow
	err = yaml.Unmarshal(data, &workflow)
	if err != nil {
		return nil, err
	}

	return &workflow, nil
}

// loadMakefile loads the Makefile for validation
func (suite *CIPipelineValidationTestSuite) loadMakefile() {
	makefilePath := filepath.Join(suite.projectRoot, "Makefile")
	
	data, err := os.ReadFile(makefilePath)
	if err != nil {
		suite.T().Logf("Warning: Could not read Makefile: %v", err)
		return
	}

	suite.makefile = &Makefile{
		Content: string(data),
		Targets: suite.extractMakeTargets(string(data)),
	}
}

// extractMakeTargets extracts target names from Makefile content
func (suite *CIPipelineValidationTestSuite) extractMakeTargets(content string) []string {
	var targets []string
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "@") {
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				target := strings.TrimSpace(parts[0])
				if !strings.Contains(target, "=") && !strings.Contains(target, " ") {
					targets = append(targets, target)
				}
			}
		}
	}
	
	return targets
}

// TestCIPipeline_WorkflowFilesExist tests that essential workflow files exist
func (suite *CIPipelineValidationTestSuite) TestCIPipeline_WorkflowFilesExist() {
	requiredWorkflows := []string{"ci.yml", "ci.yaml"}
	
	hasMainWorkflow := false
	for _, required := range requiredWorkflows {
		if _, exists := suite.workflows[required]; exists {
			hasMainWorkflow = true
			break
		}
	}
	
	suite.True(hasMainWorkflow, "Should have a main CI workflow (ci.yml or ci.yaml)")
	suite.Greater(len(suite.workflows), 0, "Should have at least one workflow")
}

// TestCIPipeline_WorkflowStructure tests basic workflow structure
func (suite *CIPipelineValidationTestSuite) TestCIPipeline_WorkflowStructure() {
	if len(suite.workflows) == 0 {
		suite.T().Skip("No workflows loaded, skipping structure validation")
	}

	for filename, workflow := range suite.workflows {
		suite.T().Run(fmt.Sprintf("Workflow_%s", filename), func(t *testing.T) {
			// Basic structure validation
			assert.NotEmpty(t, workflow.Name, "Workflow should have a name")
			assert.NotEmpty(t, workflow.Jobs, "Workflow should have at least one job")
			
			// Validate trigger configuration
			hasValidTrigger := workflow.On.Push.Branches != nil ||
				workflow.On.PullRequest.Branches != nil ||
				workflow.On.WorkflowDispatch != nil
			assert.True(t, hasValidTrigger, "Workflow should have valid triggers")
			
			// Validate concurrency configuration
			if workflow.Concurrency.Group != "" {
				assert.NotEmpty(t, workflow.Concurrency.Group, 
					"Concurrency group should not be empty if specified")
			}
		})
	}
}

// TestCIPipeline_UbuntuOnlyCompliance tests Ubuntu-only compliance
func (suite *CIPipelineValidationTestSuite) TestCIPipeline_UbuntuOnlyCompliance() {
	for filename, workflow := range suite.workflows {
		suite.T().Run(fmt.Sprintf("UbuntuCompliance_%s", filename), func(t *testing.T) {
			for jobName, job := range workflow.Jobs {
				// Check runs-on configuration
				switch runsOn := job.RunsOn.(type) {
				case string:
					assert.Contains(t, runsOn, "ubuntu", 
						"Job %s should run on Ubuntu (found: %s)", jobName, runsOn)
					assert.NotContains(t, runsOn, "windows", 
						"Job %s should not run on Windows", jobName)
					assert.NotContains(t, runsOn, "macos", 
						"Job %s should not run on macOS", jobName)
				case []interface{}:
					for _, os := range runsOn {
						osStr := fmt.Sprintf("%v", os)
						assert.Contains(t, osStr, "ubuntu", 
							"Job %s should only run on Ubuntu (found: %s)", jobName, osStr)
					}
				}
				
				// Check strategy matrix for cross-platform elements
				if job.Strategy != nil && job.Strategy.Matrix != nil {
					if osMatrix, exists := job.Strategy.Matrix["os"]; exists {
						switch osList := osMatrix.(type) {
						case []interface{}:
							for _, os := range osList {
								osStr := fmt.Sprintf("%v", os)
								assert.Contains(t, osStr, "ubuntu", 
									"Matrix OS should only include Ubuntu variants: %s", osStr)
								assert.NotContains(t, osStr, "windows", 
									"Matrix OS should not include Windows: %s", osStr)
								assert.NotContains(t, osStr, "macos", 
									"Matrix OS should not include macOS: %s", osStr)
							}
						}
					}
				}
			}
		})
	}
}

// TestCIPipeline_EssentialJobs tests that essential CI jobs are present
func (suite *CIPipelineValidationTestSuite) TestCIPipeline_EssentialJobs() {
	essentialJobs := []string{"test", "build", "lint", "security"}
	
	for filename, workflow := range suite.workflows {
		suite.T().Run(fmt.Sprintf("EssentialJobs_%s", filename), func(t *testing.T) {
			jobNames := make([]string, 0, len(workflow.Jobs))
			for jobName := range workflow.Jobs {
				jobNames = append(jobNames, strings.ToLower(jobName))
			}
			
			for _, essential := range essentialJobs {
				hasJob := false
				for _, jobName := range jobNames {
					if strings.Contains(jobName, essential) {
						hasJob = true
						break
					}
				}
				
				if !hasJob {
					t.Logf("Warning: Essential job type '%s' not found in workflow %s. "+
						"Available jobs: %v", essential, filename, jobNames)
				}
			}
			
			// At least should have test or build job
			hasTestOrBuild := false
			for _, jobName := range jobNames {
				if strings.Contains(jobName, "test") || strings.Contains(jobName, "build") {
					hasTestOrBuild = true
					break
				}
			}
			assert.True(t, hasTestOrBuild, 
				"Workflow should have at least a test or build job")
		})
	}
}

// TestCIPipeline_SecurityScanning tests security scanning configuration
func (suite *CIPipelineValidationTestSuite) TestCIPipeline_SecurityScanning() {
	for filename, workflow := range suite.workflows {
		suite.T().Run(fmt.Sprintf("Security_%s", filename), func(t *testing.T) {
			hasSecurityJob := false
			
			for jobName, job := range workflow.Jobs {
				// Check if job is security-related
				if strings.Contains(strings.ToLower(jobName), "security") ||
				   strings.Contains(strings.ToLower(jobName), "scan") {
					hasSecurityJob = true
					
					// Validate security steps
					hasGosec := false
					hasVulnCheck := false
					
					for _, step := range job.Steps {
						stepContent := strings.ToLower(step.Run + step.Uses + step.Name)
						if strings.Contains(stepContent, "gosec") {
							hasGosec = true
						}
						if strings.Contains(stepContent, "govulncheck") ||
						   strings.Contains(stepContent, "vulnerability") {
							hasVulnCheck = true
						}
					}
					
					if hasGosec || hasVulnCheck {
						t.Logf("Security scanning found in job %s", jobName)
					}
				}
			}
			
			if !hasSecurityJob {
				t.Logf("Info: No dedicated security job found in workflow %s", filename)
			}
		})
	}
}

// TestCIPipeline_TestIntegration tests test integration in CI
func (suite *CIPipelineValidationTestSuite) TestCIPipeline_TestIntegration() {
	for filename, workflow := range suite.workflows {
		suite.T().Run(fmt.Sprintf("TestIntegration_%s", filename), func(t *testing.T) {
			hasTestJob := false
			
			for jobName, job := range workflow.Jobs {
				if strings.Contains(strings.ToLower(jobName), "test") {
					hasTestJob = true
					
					// Check for Go testing steps
					hasGoTest := false
					hasCoverage := false
					
					for _, step := range job.Steps {
						stepContent := strings.ToLower(step.Run + step.Uses)
						if strings.Contains(stepContent, "go test") {
							hasGoTest = true
						}
						if strings.Contains(stepContent, "coverage") {
							hasCoverage = true
						}
					}
					
					assert.True(t, hasGoTest, 
						"Test job %s should include Go testing", jobName)
					
					if hasCoverage {
						t.Logf("Coverage reporting found in test job %s", jobName)
					}
				}
			}
			
			if !hasTestJob {
				t.Logf("Warning: No test job found in workflow %s", filename)
			}
		})
	}
}

// TestCIPipeline_MakefileIntegration tests Makefile integration with CI
func (suite *CIPipelineValidationTestSuite) TestCIPipeline_MakefileIntegration() {
	if suite.makefile == nil {
		suite.T().Skip("No Makefile found, skipping integration validation")
	}

	// Test that essential make targets exist
	essentialTargets := []string{"test", "build", "lint", "clean"}
	
	for _, target := range essentialTargets {
		hasTarget := false
		for _, makeTarget := range suite.makefile.Targets {
			if strings.Contains(makeTarget, target) {
				hasTarget = true
				break
			}
		}
		
		if !hasTarget {
			suite.T().Logf("Warning: Essential make target '%s' not found", target)
		}
	}

	// Test that CI workflows reference make targets
	for filename, workflow := range suite.workflows {
		suite.T().Run(fmt.Sprintf("MakeIntegration_%s", filename), func(t *testing.T) {
			makeReferences := 0
			
			for _, job := range workflow.Jobs {
				for _, step := range job.Steps {
					if strings.Contains(step.Run, "make ") {
						makeReferences++
					}
				}
			}
			
			if makeReferences > 0 {
				t.Logf("Found %d make command references in workflow %s", 
					makeReferences, filename)
			}
		})
	}
}

// TestCIPipeline_PerformanceOptimization tests CI performance optimizations
func (suite *CIPipelineValidationTestSuite) TestCIPipeline_PerformanceOptimization() {
	for filename, workflow := range suite.workflows {
		suite.T().Run(fmt.Sprintf("Performance_%s", filename), func(t *testing.T) {
			// Check for concurrency configuration
			hasConcurrency := workflow.Concurrency.Group != ""
			if hasConcurrency {
				assert.True(t, workflow.Concurrency.CancelInProgress, 
					"Concurrency should use cancel-in-progress for performance")
			}
			
			// Check for caching strategies
			for jobName, job := range workflow.Jobs {
				hasCaching := false
				
				for _, step := range job.Steps {
					if strings.Contains(step.Uses, "cache") ||
					   strings.Contains(strings.ToLower(step.Name), "cache") {
						hasCaching = true
						break
					}
				}
				
				if hasCaching {
					t.Logf("Caching found in job %s", jobName)
				}
			}
			
			// Check for parallel execution
			hasParallelJobs := len(workflow.Jobs) > 1
			if hasParallelJobs {
				t.Logf("Workflow %s has %d jobs for parallel execution", 
					filename, len(workflow.Jobs))
			}
		})
	}
}

// TestCIPipeline_ArtifactManagement tests artifact and report management
func (suite *CIPipelineValidationTestSuite) TestCIPipeline_ArtifactManagement() {
	for filename, workflow := range suite.workflows {
		suite.T().Run(fmt.Sprintf("Artifacts_%s", filename), func(t *testing.T) {
			hasArtifactUpload := false
			hasTestResults := false
			
			for _, job := range workflow.Jobs {
				for _, step := range job.Steps {
					if strings.Contains(step.Uses, "upload-artifact") {
						hasArtifactUpload = true
					}
					if strings.Contains(strings.ToLower(step.Name), "test") &&
					   strings.Contains(strings.ToLower(step.Name), "result") {
						hasTestResults = true
					}
				}
			}
			
			if hasArtifactUpload {
				t.Logf("Artifact upload found in workflow %s", filename)
			}
			if hasTestResults {
				t.Logf("Test result handling found in workflow %s", filename)
			}
		})
	}
}

// TestCIPipeline_EnvironmentVariables tests environment variable management
func (suite *CIPipelineValidationTestSuite) TestCIPipeline_EnvironmentVariables() {
	for filename, workflow := range suite.workflows {
		suite.T().Run(fmt.Sprintf("EnvVars_%s", filename), func(t *testing.T) {
			for jobName, job := range workflow.Jobs {
				// Check job-level environment variables
				if len(job.Env) > 0 {
					t.Logf("Job %s has %d environment variables", jobName, len(job.Env))
				}
				
				// Check step-level environment variables
				for i, step := range job.Steps {
					if len(step.Env) > 0 {
						t.Logf("Job %s step %d has %d environment variables", 
							jobName, i, len(step.Env))
					}
					
					// Check for secrets usage (should use GitHub secrets)
					for key, value := range step.Env {
						if strings.Contains(strings.ToUpper(key), "TOKEN") ||
						   strings.Contains(strings.ToUpper(key), "SECRET") ||
						   strings.Contains(strings.ToUpper(key), "KEY") {
							assert.Contains(t, value, "${{", 
								"Secret environment variable %s should use GitHub secrets syntax", key)
						}
					}
				}
			}
		})
	}
}

// TestCIPipeline_BranchProtection tests branch protection alignment
func (suite *CIPipelineValidationTestSuite) TestCIPipeline_BranchProtection() {
	protectedBranches := []string{"main", "integrate/mvp"}
	
	for filename, workflow := range suite.workflows {
		suite.T().Run(fmt.Sprintf("BranchProtection_%s", filename), func(t *testing.T) {
			// Check push triggers
			if len(workflow.On.Push.Branches) > 0 {
				for _, branch := range workflow.On.Push.Branches {
					if contains(protectedBranches, branch) {
						t.Logf("Workflow %s triggers on protected branch: %s", filename, branch)
					}
				}
			}
			
			// Check PR triggers
			if len(workflow.On.PullRequest.Branches) > 0 {
				for _, branch := range workflow.On.PullRequest.Branches {
					if contains(protectedBranches, branch) {
						t.Logf("Workflow %s triggers on PRs to protected branch: %s", filename, branch)
					}
				}
			}
		})
	}
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestSuite runner function
func TestCIPipelineValidationTestSuite(t *testing.T) {
	suite.Run(t, new(CIPipelineValidationTestSuite))
}

// Additional standalone CI validation tests
func TestCIPipeline_BasicWorkflowSyntax(t *testing.T) {
	// Test basic workflow syntax validation
	workflowDir := filepath.Join("..", "..", "..", ".github", "workflows")
	
	if _, err := os.Stat(workflowDir); os.IsNotExist(err) {
		t.Skip("GitHub workflows directory not found")
	}
	
	files, err := filepath.Glob(filepath.Join(workflowDir, "*.yml"))
	require.NoError(t, err)
	
	yamlFiles, err := filepath.Glob(filepath.Join(workflowDir, "*.yaml"))
	if err == nil {
		files = append(files, yamlFiles...)
	}
	
	assert.Greater(t, len(files), 0, "Should have at least one workflow file")
	
	for _, file := range files {
		data, err := os.ReadFile(file)
		assert.NoError(t, err, "Should be able to read workflow file: %s", file)
		
		var workflow map[string]interface{}
		err = yaml.Unmarshal(data, &workflow)
		assert.NoError(t, err, "Workflow should be valid YAML: %s", file)
		
		// Basic structure checks
		assert.Contains(t, workflow, "name", "Workflow should have a name: %s", file)
		assert.Contains(t, workflow, "jobs", "Workflow should have jobs: %s", file)
	}
}

func TestCIPipeline_MakefileTargets(t *testing.T) {
	// Test that Makefile has essential targets for CI
	makefilePath := filepath.Join("..", "..", "..", "Makefile")
	
	data, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Skipf("Could not read Makefile: %v", err)
	}
	
	content := string(data)
	
	// Essential targets for CI
	essentialTargets := []string{"test", "build", "lint", "clean"}
	
	for _, target := range essentialTargets {
		assert.Contains(t, content, target+":", 
			"Makefile should have %s target", target)
	}
	
	// Test-specific targets
	testTargets := []string{"test-unit", "test-integration", "test-e2e"}
	hasTestTarget := false
	
	for _, target := range testTargets {
		if strings.Contains(content, target+":") {
			hasTestTarget = true
			break
		}
	}
	
	assert.True(t, hasTestTarget, 
		"Makefile should have at least one specific test target")
}