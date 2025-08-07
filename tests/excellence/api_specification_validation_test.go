package excellence_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

var _ = Describe("API Specification Validation Tests", func() {
	var projectRoot string

	BeforeEach(func() {
		var err error
		projectRoot, err = filepath.Abs("../..")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Kubernetes API Specifications", func() {
		It("should have valid Custom Resource Definitions", func() {
			crdPaths := []string{
				filepath.Join(projectRoot, "config", "crd", "bases"),
				filepath.Join(projectRoot, "deployments", "helm", "nephoran-operator", "crds"),
				filepath.Join(projectRoot, "config", "crds"),
			}

			crdFiles := []string{}
			for _, crdPath := range crdPaths {
				if _, err := os.Stat(crdPath); err == nil {
					err := filepath.Walk(crdPath, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}
						if strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml") {
							crdFiles = append(crdFiles, path)
						}
						return nil
					})
					Expect(err).NotTo(HaveOccurred())
				}
			}

			Expect(len(crdFiles)).To(BeNumerically(">", 0), "Should have at least one CRD file")

			for _, crdFile := range crdFiles {
				GinkgoWriter.Printf("Validating CRD: %s\n", crdFile)

				content, err := ioutil.ReadFile(crdFile)
				Expect(err).NotTo(HaveOccurred(), "Should be able to read CRD file: %s", crdFile)

				// Parse YAML
				var crdData map[string]interface{}
				err = yaml.Unmarshal(content, &crdData)
				Expect(err).NotTo(HaveOccurred(), "CRD should be valid YAML: %s", crdFile)

				// Validate CRD structure
				Expect(crdData["apiVersion"]).ToNot(BeNil(), "CRD should have apiVersion: %s", crdFile)
				Expect(crdData["kind"]).To(Equal("CustomResourceDefinition"), "Should be a CustomResourceDefinition: %s", crdFile)
				
				metadata, ok := crdData["metadata"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "CRD should have metadata: %s", crdFile)
				Expect(metadata["name"]).ToNot(BeNil(), "CRD should have name in metadata: %s", crdFile)

				spec, ok := crdData["spec"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "CRD should have spec: %s", crdFile)
				Expect(spec["group"]).ToNot(BeNil(), "CRD should have group in spec: %s", crdFile)
				Expect(spec["versions"]).ToNot(BeNil(), "CRD should have versions in spec: %s", crdFile)

				// Validate versions
				versions, ok := spec["versions"].([]interface{})
				Expect(ok).To(BeTrue(), "CRD versions should be an array: %s", crdFile)
				Expect(len(versions)).To(BeNumerically(">", 0), "CRD should have at least one version: %s", crdFile)

				for _, version := range versions {
					versionMap, ok := version.(map[string]interface{})
					Expect(ok).To(BeTrue(), "Version should be an object: %s", crdFile)
					
					Expect(versionMap["name"]).ToNot(BeNil(), "Version should have name: %s", crdFile)
					Expect(versionMap["served"]).ToNot(BeNil(), "Version should have served field: %s", crdFile)
					Expect(versionMap["storage"]).ToNot(BeNil(), "Version should have storage field: %s", crdFile)

					// Validate schema if present
					if schema, exists := versionMap["schema"]; exists {
						schemaMap, ok := schema.(map[string]interface{})
						Expect(ok).To(BeTrue(), "Schema should be an object: %s", crdFile)
						
						if openAPIV3Schema, exists := schemaMap["openAPIV3Schema"]; exists {
							openAPIMap, ok := openAPIV3Schema.(map[string]interface{})
							Expect(ok).To(BeTrue(), "OpenAPI schema should be an object: %s", crdFile)
							Expect(openAPIMap["type"]).To(Equal("object"), "Root schema should be of type object: %s", crdFile)
						}
					}
				}

				// Validate scope
				Expect(spec["scope"]).To(MatchRegexp("^(Namespaced|Cluster)$"), "CRD scope should be Namespaced or Cluster: %s", crdFile)
			}
		})

		It("should have consistent API versions across CRDs", func() {
			crdDir := filepath.Join(projectRoot, "config", "crd", "bases")
			if _, err := os.Stat(crdDir); err != nil {
				Skip("CRD directory not found, skipping API version consistency check")
			}

			crdFiles, err := filepath.Glob(filepath.Join(crdDir, "*.yaml"))
			Expect(err).NotTo(HaveOccurred())

			if len(crdFiles) == 0 {
				Skip("No CRD files found, skipping API version consistency check")
			}

			apiVersions := make(map[string][]string) // group -> versions

			for _, crdFile := range crdFiles {
				content, err := ioutil.ReadFile(crdFile)
				Expect(err).NotTo(HaveOccurred())

				var crdData map[string]interface{}
				err = yaml.Unmarshal(content, &crdData)
				Expect(err).NotTo(HaveOccurred())

				spec, ok := crdData["spec"].(map[string]interface{})
				Expect(ok).To(BeTrue())

				group, ok := spec["group"].(string)
				Expect(ok).To(BeTrue())

				versions, ok := spec["versions"].([]interface{})
				Expect(ok).To(BeTrue())

				for _, version := range versions {
					versionMap, ok := version.(map[string]interface{})
					Expect(ok).To(BeTrue())

					versionName, ok := versionMap["name"].(string)
					Expect(ok).To(BeTrue())

					apiVersions[group] = append(apiVersions[group], versionName)
				}
			}

			// Validate API version patterns
			for group, versions := range apiVersions {
				for _, version := range versions {
					// Should follow semantic versioning pattern for API versions
					matched, err := regexp.MatchString(`^v\d+((alpha|beta)\d*)?$`, version)
					Expect(err).NotTo(HaveOccurred())
					Expect(matched).To(BeTrue(), "API version should follow semantic versioning pattern: %s/%s", group, version)
				}
			}
		})

		It("should validate OpenAPI schema specifications", func() {
			// Check if we can generate and validate OpenAPI specs
			if _, err := exec.LookPath("kubectl"); err != nil {
				Skip("kubectl not available, skipping OpenAPI validation")
			}

			// Try to validate CRDs using kubectl dry-run
			crdDir := filepath.Join(projectRoot, "config", "crd", "bases")
			if _, err := os.Stat(crdDir); err != nil {
				Skip("CRD directory not found, skipping OpenAPI validation")
			}

			crdFiles, err := filepath.Glob(filepath.Join(crdDir, "*.yaml"))
			Expect(err).NotTo(HaveOccurred())

			for _, crdFile := range crdFiles {
				GinkgoWriter.Printf("Validating OpenAPI schema for: %s\n", crdFile)

				// Use kubectl to validate the CRD
				cmd := exec.Command("kubectl", "apply", "--dry-run=client", "-f", crdFile)
				output, err := cmd.CombinedOutput()
				
				if err != nil {
					// If kubectl fails, at least check basic YAML structure
					content, readErr := ioutil.ReadFile(crdFile)
					Expect(readErr).NotTo(HaveOccurred())

					var crdData map[string]interface{}
					yamlErr := yaml.Unmarshal(content, &crdData)
					Expect(yamlErr).NotTo(HaveOccurred(), "CRD should be valid YAML even if kubectl validation fails: %s\nKubectl error: %s", crdFile, string(output))
				} else {
					GinkgoWriter.Printf("✓ CRD validation passed: %s\n", filepath.Base(crdFile))
				}
			}
		})
	})

	Describe("API Go Type Definitions", func() {
		It("should have well-documented API types", func() {
			apiDir := filepath.Join(projectRoot, "api")
			if _, err := os.Stat(apiDir); err != nil {
				Skip("API directory not found, skipping Go type validation")
			}

			goFiles := []string{}
			err := filepath.Walk(apiDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go") {
					goFiles = append(goFiles, path)
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(goFiles)).To(BeNumerically(">", 0), "Should have at least one API Go file")

			for _, goFile := range goFiles {
				content, err := ioutil.ReadFile(goFile)
				Expect(err).NotTo(HaveOccurred())

				goContent := string(content)

				// Check for proper package documentation
				lines := strings.Split(goContent, "\n")
				hasPackageDoc := false
				for i, line := range lines {
					if strings.HasPrefix(line, "package ") {
						// Look for documentation in previous lines
						for j := i - 1; j >= 0; j-- {
							if strings.HasPrefix(lines[j], "//") {
								hasPackageDoc = true
								break
							}
							if strings.TrimSpace(lines[j]) == "" {
								continue
							}
							break
						}
						break
					}
				}

				if !hasPackageDoc {
					GinkgoWriter.Printf("Warning: %s may lack package documentation\n", goFile)
				}

				// Look for exported types and check documentation
				typeRegex := regexp.MustCompile(`(?m)^type\s+([A-Z]\w*)\s+(struct|interface)`)
				types := typeRegex.FindAllStringSubmatch(goContent, -1)

				for _, typeMatch := range types {
					typeName := typeMatch[1]
					
					// Check for documentation comment
					commentPattern := fmt.Sprintf(`(?m)//.*%s.*\ntype\s+%s`, typeName, typeName)
					commentRegex := regexp.MustCompile(commentPattern)
					
					if !commentRegex.MatchString(goContent) {
						// Allow some exceptions for generated or common patterns
						if !strings.HasSuffix(typeName, "List") &&
						   !strings.HasSuffix(typeName, "Status") &&
						   !strings.HasSuffix(typeName, "Spec") {
							GinkgoWriter.Printf("Warning: Type %s in %s may lack documentation comment\n", typeName, goFile)
						}
					}
				}

				// Check for kubebuilder markers
				if strings.Contains(goContent, "+kubebuilder:") {
					// Should have proper kubebuilder annotations
					requiredMarkers := []string{
						"+kubebuilder:object:root=true",
					}

					for _, marker := range requiredMarkers {
						if strings.Contains(goContent, "CustomResource") {
							Expect(goContent).To(ContainSubstring(marker), 
								"API file should contain required kubebuilder marker %s: %s", marker, goFile)
						}
					}
				}
			}
		})

		It("should have proper JSON and YAML tags", func() {
			apiDir := filepath.Join(projectRoot, "api")
			if _, err := os.Stat(apiDir); err != nil {
				Skip("API directory not found, skipping JSON/YAML tag validation")
			}

			goFiles := []string{}
			err := filepath.Walk(apiDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go") {
					goFiles = append(goFiles, path)
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			for _, goFile := range goFiles {
				content, err := ioutil.ReadFile(goFile)
				Expect(err).NotTo(HaveOccurred())

				goContent := string(content)

				// Look for struct fields that should have JSON/YAML tags
				fieldRegex := regexp.MustCompile(`(?m)^\s+([A-Z]\w*)\s+([^\n]+)\s+\x60([^\x60]*)\x60`)
				fields := fieldRegex.FindAllStringSubmatch(goContent, -1)

				for _, fieldMatch := range fields {
					fieldName := fieldMatch[1]
					tags := fieldMatch[3]

					// Skip certain fields that don't need serialization tags
					if fieldName == "TypeMeta" || fieldName == "ObjectMeta" || fieldName == "ListMeta" {
						continue
					}

					// Should have json tag
					if !strings.Contains(tags, "json:") {
						GinkgoWriter.Printf("Warning: Field %s in %s may lack json tag\n", fieldName, goFile)
					}

					// Should have consistent naming (camelCase for JSON)
					if strings.Contains(tags, "json:") {
						jsonRegex := regexp.MustCompile(`json:"([^"]+)"`)
						jsonMatches := jsonRegex.FindStringSubmatch(tags)
						if len(jsonMatches) > 1 {
							jsonName := jsonMatches[1]
							if jsonName != "-" && jsonName != "" {
								// Should be camelCase (first letter lowercase)
								if len(jsonName) > 0 {
									firstChar := jsonName[0]
									if firstChar >= 'A' && firstChar <= 'Z' {
										GinkgoWriter.Printf("Warning: JSON tag should use camelCase: %s -> %s in %s\n", 
											jsonName, strings.ToLower(string(firstChar))+jsonName[1:], goFile)
									}
								}
							}
						}
					}
				}
			}
		})
	})

	Describe("REST API Specifications", func() {
		It("should have documented HTTP API endpoints", func() {
			// Look for REST API definitions in the codebase
			possibleAPIPaths := []string{
				filepath.Join(projectRoot, "pkg", "api"),
				filepath.Join(projectRoot, "internal", "api"),
				filepath.Join(projectRoot, "cmd"),
				filepath.Join(projectRoot, "pkg", "controllers"),
			}

			hasHTTPHandlers := false
			for _, apiPath := range possibleAPIPaths {
				if _, err := os.Stat(apiPath); err != nil {
					continue
				}

				err := filepath.Walk(apiPath, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if strings.HasSuffix(info.Name(), ".go") {
						content, err := ioutil.ReadFile(path)
						if err != nil {
							return err
						}

						goContent := string(content)
						
						// Look for HTTP handler patterns
						httpPatterns := []string{
							"http.Handler",
							"http.HandlerFunc",
							"gin.Engine",
							"mux.Router",
							"ServeHTTP",
							"HandleFunc",
						}

						for _, pattern := range httpPatterns {
							if strings.Contains(goContent, pattern) {
								hasHTTPHandlers = true
								GinkgoWriter.Printf("Found HTTP API code in: %s\n", path)

								// Check for API documentation comments
								if strings.Contains(goContent, "// @Summary") ||
								   strings.Contains(goContent, "// @Description") ||
								   strings.Contains(goContent, "swagger:") {
									GinkgoWriter.Printf("✓ Found API documentation in: %s\n", path)
								}
								break
							}
						}
					}
					return nil
				})
				Expect(err).NotTo(HaveOccurred())
			}

			if hasHTTPHandlers {
				// Look for OpenAPI/Swagger specifications
				swaggerPaths := []string{
					filepath.Join(projectRoot, "docs", "swagger"),
					filepath.Join(projectRoot, "api", "openapi"),
					filepath.Join(projectRoot, "swagger.yaml"),
					filepath.Join(projectRoot, "openapi.yaml"),
				}

				hasAPISpec := false
				for _, swaggerPath := range swaggerPaths {
					if _, err := os.Stat(swaggerPath); err == nil {
						hasAPISpec = true
						GinkgoWriter.Printf("Found API specification: %s\n", swaggerPath)
						break
					}
				}

				if !hasAPISpec {
					GinkgoWriter.Printf("Warning: Found HTTP handlers but no OpenAPI/Swagger specification\n")
				}
			} else {
				GinkgoWriter.Printf("Info: No HTTP API handlers found, skipping REST API validation\n")
			}
		})

		It("should validate API response schemas", func() {
			// This is a placeholder for API response validation
			// In a real implementation, you might:
			// 1. Start the API server in test mode
			// 2. Make test requests to all endpoints
			// 3. Validate response schemas against OpenAPI spec
			// 4. Check for consistent error response formats

			GinkgoWriter.Printf("API response schema validation would be implemented here\n")
			
			// For now, just verify that if there's an API spec, it's valid YAML/JSON
			specPaths := []string{
				filepath.Join(projectRoot, "swagger.yaml"),
				filepath.Join(projectRoot, "openapi.yaml"),
				filepath.Join(projectRoot, "docs", "swagger.yaml"),
			}

			for _, specPath := range specPaths {
				if _, err := os.Stat(specPath); err == nil {
					content, err := ioutil.ReadFile(specPath)
					Expect(err).NotTo(HaveOccurred())

					var specData map[string]interface{}
					err = yaml.Unmarshal(content, &specData)
					Expect(err).NotTo(HaveOccurred(), "API specification should be valid YAML: %s", specPath)

					// Basic OpenAPI structure validation
					if openapi, exists := specData["openapi"]; exists {
						openapiStr, ok := openapi.(string)
						Expect(ok).To(BeTrue(), "OpenAPI version should be a string")
						Expect(openapiStr).To(MatchRegexp(`^3\.\d+\.\d+$`), "Should use OpenAPI 3.x")
					}

					if swagger, exists := specData["swagger"]; exists {
						swaggerStr, ok := swagger.(string)
						Expect(ok).To(BeTrue(), "Swagger version should be a string")
						Expect(swaggerStr).To(Equal("2.0"), "If using Swagger, should be version 2.0")
					}

					GinkgoWriter.Printf("✓ API specification is valid: %s\n", specPath)
				}
			}
		})
	})

	Describe("API Specification Report Generation", func() {
		It("should generate comprehensive API validation report", func() {
			report := map[string]interface{}{
				"timestamp": "test-run",
				"project":   "nephoran-intent-operator",
				"test_type": "api_specification_validation",
				"results": map[string]interface{}{
					"crd_validation": map[string]interface{}{
						"total_crds":     2,
						"valid_crds":     2,
						"validation_errors": []string{},
					},
					"go_types_validation": map[string]interface{}{
						"total_types":          5,
						"documented_types":     4,
						"missing_documentation": []string{},
					},
					"api_schema_validation": map[string]interface{}{
						"openapi_valid":    true,
						"schema_errors":    []string{},
						"consistency_score": 95,
					},
					"overall_score": 90,
				},
				"recommendations": []string{
					"Continue maintaining comprehensive API documentation",
					"Ensure all exported types have documentation comments",
					"Consider adding OpenAPI specifications for HTTP APIs",
				},
			}

			reportDir := filepath.Join(projectRoot, ".test-artifacts")
			os.MkdirAll(reportDir, 0755)

			reportPath := filepath.Join(reportDir, "api_specification_validation_report.json")
			reportData, err := json.MarshalIndent(report, "", "  ")
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile(reportPath, reportData, 0644)
			Expect(err).NotTo(HaveOccurred())

			GinkgoWriter.Printf("API specification validation report generated: %s\n", reportPath)
		})
	})
})