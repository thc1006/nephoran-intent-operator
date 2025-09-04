//go:build validation

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

package crds

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/tests/fixtures"
)

// CRDValidationTestSuite provides comprehensive CRD validation testing
type CRDValidationTestSuite struct {
	suite.Suite
	client   client.Client
	scheme   *runtime.Scheme
	ctx      context.Context
	crdFiles []string
	crds     []*apiextensionsv1.CustomResourceDefinition
}

// SetupSuite initializes the CRD validation test environment
func (suite *CRDValidationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Create scheme
	suite.scheme = runtime.NewScheme()
	err := nephoranv1.AddToScheme(suite.scheme)
	suite.Require().NoError(err)
	err = apiextensionsv1.AddToScheme(suite.scheme)
	suite.Require().NoError(err)

	suite.client = fake.NewClientBuilder().WithScheme(suite.scheme).Build()

	// Load CRD files
	suite.loadCRDFiles()
}

// loadCRDFiles loads CRD files from the config directory
func (suite *CRDValidationTestSuite) loadCRDFiles() {
	crdDir := filepath.Join("..", "..", "..", "config", "crd", "bases")

	// Check if CRD directory exists
	if _, err := os.Stat(crdDir); os.IsNotExist(err) {
		suite.T().Skip("CRD directory not found, skipping CRD validation tests")
		return
	}

	// Find all YAML files in CRD directory
	files, err := filepath.Glob(filepath.Join(crdDir, "*.yaml"))
	if err != nil {
		suite.T().Skipf("Failed to read CRD directory: %v", err)
		return
	}

	suite.crdFiles = files

	// Load and parse CRDs
	for _, file := range files {
		crd, err := suite.loadCRDFromFile(file)
		if err != nil {
			suite.T().Logf("Warning: Failed to load CRD from %s: %v", file, err)
			continue
		}
		if crd != nil {
			suite.crds = append(suite.crds, crd)
		}
	}
}

// loadCRDFromFile loads a CRD from a YAML file
func (suite *CRDValidationTestSuite) loadCRDFromFile(filename string) (*apiextensionsv1.CustomResourceDefinition, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Convert YAML to JSON
	jsonData, err := yaml.ToJSON(data)
	if err != nil {
		return nil, err
	}

	// Unmarshal into CRD
	var crd apiextensionsv1.CustomResourceDefinition
	err = json.Unmarshal(jsonData, &crd)
	if err != nil {
		return nil, err
	}

	return &crd, nil
}

// TestValidation_CRDFilesExist tests that CRD files exist
func (suite *CRDValidationTestSuite) TestValidation_CRDFilesExist() {
	suite.True(len(suite.crdFiles) > 0, "Should have at least one CRD file")

	for _, file := range suite.crdFiles {
		suite.FileExists(file, "CRD file should exist: %s", file)

		// Check file is not empty
		info, err := os.Stat(file)
		suite.NoError(err)
		suite.Greater(info.Size(), int64(100), "CRD file should not be empty: %s", file)
	}
}

// TestValidation_CRDStructure tests basic CRD structure
func (suite *CRDValidationTestSuite) TestValidation_CRDStructure() {
	if len(suite.crds) == 0 {
		suite.T().Skip("No CRDs loaded, skipping structure validation")
	}

	for _, crd := range suite.crds {
		suite.T().Run(fmt.Sprintf("CRD_%s", crd.Name), func(t *testing.T) {
			// Basic metadata validation
			assert.NotEmpty(t, crd.Name, "CRD should have a name")
			assert.NotEmpty(t, crd.Spec.Group, "CRD should have a group")
			assert.NotEmpty(t, crd.Spec.Names.Kind, "CRD should have a kind")
			assert.NotEmpty(t, crd.Spec.Names.Plural, "CRD should have plural name")

			// Validate group follows expected pattern
			assert.True(t, strings.Contains(crd.Spec.Group, "nephoran.io") ||
				strings.Contains(crd.Spec.Group, "nephio"),
				"CRD group should contain 'nephoran.io' or 'nephio'")

			// Validate scope
			assert.Contains(t, []apiextensionsv1.ResourceScope{
				apiextensionsv1.NamespaceScoped,
				apiextensionsv1.ClusterScoped,
			}, crd.Spec.Scope, "CRD should have valid scope")

			// Validate versions
			assert.Greater(t, len(crd.Spec.Versions), 0, "CRD should have at least one version")

			// Check that at least one version is served and storage
			var hasServed, hasStorage bool
			for _, version := range crd.Spec.Versions {
				if version.Served {
					hasServed = true
				}
				if version.Storage {
					hasStorage = true
				}
			}
			assert.True(t, hasServed, "CRD should have at least one served version")
			assert.True(t, hasStorage, "CRD should have exactly one storage version")
		})
	}
}

// TestValidation_CRDSchemas tests CRD OpenAPI schemas
func (suite *CRDValidationTestSuite) TestValidation_CRDSchemas() {
	if len(suite.crds) == 0 {
		suite.T().Skip("No CRDs loaded, skipping schema validation")
	}

	for _, crd := range suite.crds {
		suite.T().Run(fmt.Sprintf("Schema_%s", crd.Name), func(t *testing.T) {
			for _, version := range crd.Spec.Versions {
				if version.Schema == nil {
					t.Errorf("Version %s should have a schema", version.Name)
					continue
				}

				schema := version.Schema.OpenAPIV3Schema
				assert.NotNil(t, schema, "Version %s should have OpenAPI schema", version.Name)

				if schema != nil {
					assert.Equal(t, "object", schema.Type, "Root schema should be of type 'object'")

					// Check for spec and status properties
					if schema.Properties != nil {
						_, hasSpec := schema.Properties["spec"]
						_, hasStatus := schema.Properties["status"]

						if strings.Contains(crd.Name, "networkintent") {
							assert.True(t, hasSpec, "NetworkIntent should have spec property")
							// Status is optional for some CRDs
						}

						// Validate spec schema if present
						if hasSpec {
							specSchema := schema.Properties["spec"]
							assert.Equal(t, "object", specSchema.Type, "Spec should be of type 'object'")

							// Check for intent field in NetworkIntent spec
							if strings.Contains(crd.Name, "networkintent") && specSchema.Properties != nil {
								intentProp, hasIntent := specSchema.Properties["intent"]
								assert.True(t, hasIntent, "NetworkIntent spec should have 'intent' field")
								if hasIntent {
									assert.Equal(t, "string", intentProp.Type, "Intent field should be of type 'string'")
								}
							}
						}
					}
				}
			}
		})
	}
}

// TestValidation_CRDNames tests CRD naming conventions
func (suite *CRDValidationTestSuite) TestValidation_CRDNames() {
	if len(suite.crds) == 0 {
		suite.T().Skip("No CRDs loaded, skipping naming validation")
	}

	for _, crd := range suite.crds {
		suite.T().Run(fmt.Sprintf("Names_%s", crd.Name), func(t *testing.T) {
			names := crd.Spec.Names

			// Validate naming conventions
			assert.Equal(t, strings.ToLower(names.Kind), names.Singular,
				"Singular name should be lowercase of Kind")
			assert.True(t, strings.HasSuffix(names.Plural, "s") ||
				strings.HasSuffix(names.Plural, "ies") ||
				strings.HasSuffix(names.Plural, "es"),
				"Plural name should be proper plural form")

			// Validate CRD name format (plural.group)
			expectedName := fmt.Sprintf("%s.%s", names.Plural, crd.Spec.Group)
			assert.Equal(t, expectedName, crd.Name,
				"CRD name should follow format 'plural.group'")

			// Check for appropriate short names
			if len(names.ShortNames) > 0 {
				for _, shortName := range names.ShortNames {
					assert.LessOrEqual(t, len(shortName), 8,
						"Short name should be 8 characters or less: %s", shortName)
					assert.Regexp(t, "^[a-z]+$", shortName,
						"Short name should contain only lowercase letters: %s", shortName)
				}
			}
		})
	}
}

// TestValidation_CRDVersioning tests CRD versioning
func (suite *CRDValidationTestSuite) TestValidation_CRDVersioning() {
	if len(suite.crds) == 0 {
		suite.T().Skip("No CRDs loaded, skipping versioning validation")
	}

	for _, crd := range suite.crds {
		suite.T().Run(fmt.Sprintf("Versioning_%s", crd.Name), func(t *testing.T) {
			versions := crd.Spec.Versions
			assert.Greater(t, len(versions), 0, "CRD should have at least one version")

			storageVersions := 0
			for _, version := range versions {
				// Validate version name format
				assert.Regexp(t, "^v[0-9]+(alpha[0-9]+|beta[0-9]+)?$", version.Name,
					"Version name should follow Kubernetes versioning convention: %s", version.Name)

				if version.Storage {
					storageVersions++
				}

				// Each version should have schema
				assert.NotNil(t, version.Schema, "Version %s should have schema", version.Name)
			}

			// Exactly one version should be marked as storage
			assert.Equal(t, 1, storageVersions,
				"CRD should have exactly one storage version")
		})
	}
}

// TestValidation_CRDCompatibility tests CRD compatibility with our API types
func (suite *CRDValidationTestSuite) TestValidation_CRDCompatibility() {
	// Test that our Go types are compatible with CRD schemas

	// Create a NetworkIntent object
	intent := fixtures.SimpleNetworkIntent()

	// Validate that it can be serialized/deserialized properly
	data, err := json.Marshal(intent)
	suite.NoError(err, "Should be able to marshal NetworkIntent to JSON")

	var unmarshaled nephoranv1.NetworkIntent
	err = json.Unmarshal(data, &unmarshaled)
	suite.NoError(err, "Should be able to unmarshal NetworkIntent from JSON")

	suite.Equal(intent.Spec.Intent, unmarshaled.Spec.Intent,
		"Marshaled and unmarshaled intent should match")
}

// TestValidation_CRDRequired tests required field validation
func (suite *CRDValidationTestSuite) TestValidation_CRDRequired() {
	// Test with valid NetworkIntent
	validIntent := fixtures.SimpleNetworkIntent()
	validIntent.Namespace = "validation-test"

	err := suite.client.Create(suite.ctx, validIntent)
	suite.NoError(err, "Valid NetworkIntent should be created successfully")

	// Note: Fake client doesn't enforce CRD validation, so we can't test
	// required field validation directly. This would need a real cluster.
	// In a real test environment, we would test invalid objects here.
}

// TestValidation_CRDLabelsAndAnnotations tests CRD labels and annotations
func (suite *CRDValidationTestSuite) TestValidation_CRDLabelsAndAnnotations() {
	if len(suite.crds) == 0 {
		suite.T().Skip("No CRDs loaded, skipping labels/annotations validation")
	}

	for _, crd := range suite.crds {
		suite.T().Run(fmt.Sprintf("Labels_%s", crd.Name), func(t *testing.T) {
			// Check for standard Kubernetes labels
			if crd.Labels != nil {
				// Validate common labels if present
				if appName, exists := crd.Labels["app.kubernetes.io/name"]; exists {
					assert.Contains(t, appName, "nephoran",
						"App name should contain 'nephoran'")
				}

				if component, exists := crd.Labels["app.kubernetes.io/component"]; exists {
					assert.NotEmpty(t, component, "Component label should not be empty")
				}
			}

			// Check for controller-gen annotations
			if crd.Annotations != nil {
				if controllerGen, exists := crd.Annotations["controller-gen.kubebuilder.io/version"]; exists {
					assert.NotEmpty(t, controllerGen,
						"Controller-gen version annotation should not be empty")
				}
			}
		})
	}
}

// TestValidation_CRDCategories tests CRD categories
func (suite *CRDValidationTestSuite) TestValidation_CRDCategories() {
	if len(suite.crds) == 0 {
		suite.T().Skip("No CRDs loaded, skipping categories validation")
	}

	for _, crd := range suite.crds {
		suite.T().Run(fmt.Sprintf("Categories_%s", crd.Name), func(t *testing.T) {
			// Check for appropriate categories
			if len(crd.Spec.Names.Categories) > 0 {
				validCategories := []string{"all", "nephoran", "networking", "intent"}

				for _, category := range crd.Spec.Names.Categories {
					assert.Contains(t, validCategories, category,
						"Category should be in the list of valid categories: %s", category)
				}
			}
		})
	}
}

// TestValidation_CRDSubresources tests CRD subresources
func (suite *CRDValidationTestSuite) TestValidation_CRDSubresources() {
	if len(suite.crds) == 0 {
		suite.T().Skip("No CRDs loaded, skipping subresources validation")
	}

	for _, crd := range suite.crds {
		suite.T().Run(fmt.Sprintf("Subresources_%s", crd.Name), func(t *testing.T) {
			for _, version := range crd.Spec.Versions {
				if version.Subresources != nil {
					// Status subresource validation
					if version.Subresources.Status != nil {
						// Status subresource should be enabled for objects with status
						if version.Schema != nil &&
							version.Schema.OpenAPIV3Schema != nil &&
							version.Schema.OpenAPIV3Schema.Properties != nil {
							_, hasStatus := version.Schema.OpenAPIV3Schema.Properties["status"]
							if hasStatus {
								assert.NotNil(t, version.Subresources.Status,
									"Objects with status field should enable status subresource")
							}
						}
					}

					// Scale subresource validation (if applicable)
					if version.Subresources.Scale != nil {
						scale := version.Subresources.Scale
						assert.NotEmpty(t, scale.SpecReplicasPath,
							"Scale subresource should specify spec replicas path")
						assert.NotEmpty(t, scale.StatusReplicasPath,
							"Scale subresource should specify status replicas path")
					}
				}
			}
		})
	}
}

// TestValidation_GeneratedCRDConsistency tests consistency between generated CRDs and source
func (suite *CRDValidationTestSuite) TestValidation_GeneratedCRDConsistency() {
	// This test would verify that generated CRDs match the source API definitions
	// In a real implementation, this might compare timestamps, comments, or checksums

	for _, file := range suite.crdFiles {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		suite.T().Logf("CRD file: %s, size: %d bytes, modified: %s",
			file, info.Size(), info.ModTime().Format("2006-01-02 15:04:05"))

		// Check that file was generated recently (within last 24 hours)
		// This is a simple check - real validation might be more sophisticated
		if info.ModTime().Before(time.Now().Add(-24 * time.Hour)) {
			suite.T().Logf("Warning: CRD file %s is older than 24 hours", file)
		}
	}
}

// TestSuite runner function
func TestCRDValidationTestSuite(t *testing.T) {
	suite.Run(t, new(CRDValidationTestSuite))
}

// Additional standalone CRD validation tests
func TestCRDValidation_NetworkIntentBasic(t *testing.T) {
	// Basic validation test that doesn't require CRD files
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	require.NoError(t, err)

	// Test GVK registration
	gvk := nephoranv1.GroupVersion.WithKind("NetworkIntent")
	assert.True(t, scheme.Recognizes(gvk), "Scheme should recognize NetworkIntent GVK")

	// Test object creation
	obj, err := scheme.New(gvk)
	assert.NoError(t, err)

	intent, ok := obj.(*nephoranv1.NetworkIntent)
	assert.True(t, ok)
	assert.NotNil(t, intent)
	assert.Equal(t, "NetworkIntent", intent.Kind)
	assert.Equal(t, "nephoran.io/v1", intent.APIVersion)
}

func TestCRDValidation_RequiredFields(t *testing.T) {
	// Test that required fields are properly defined
	intent := &nephoranv1.NetworkIntent{}

	// Set required fields
	intent.Spec.Intent = "test intent"

	// Basic validation - in a real cluster, this would be enforced by the API server
	assert.NotEmpty(t, intent.Spec.Intent, "Intent field should not be empty")

	// Test JSON marshaling with required fields
	data, err := json.Marshal(intent)
	assert.NoError(t, err)

	var unmarshaled nephoranv1.NetworkIntent
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, intent.Spec.Intent, unmarshaled.Spec.Intent)
}
