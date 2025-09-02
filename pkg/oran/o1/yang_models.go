package o1

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// YANGModelRegistry manages YANG model schemas and validation.

type YANGModelRegistry struct {
	models map[string]*YANGModel

	modelsByNS map[string]*YANGModel

	validators map[string]YANGValidator

	mutex sync.RWMutex

	loadedModules map[string]time.Time
}

// YANGModel represents a YANG model definition.

type YANGModel struct {
	Name string `json:"name"`

	Namespace string `json:"namespace"`

	Version string `json:"version"`

	Revision string `json:"revision"`

	Description string `json:"description"`

	Contact string `json:"contact,omitempty"`

	Organization string `json:"organization,omitempty"`

	Schema json.RawMessage `json:"schema"`

	Dependencies []string `json:"dependencies,omitempty"`

	ModulePath string `json:"module_path,omitempty"`

	LoadTime time.Time `json:"load_time"`
}

// YANGValidator provides validation functionality for YANG models.

type YANGValidator interface {
	ValidateData(data interface{}, modelName string) error

	ValidateXPath(xpath, modelName string) error

	GetModelInfo(modelName string) (*YANGModel, error)
}

// StandardYANGValidator implements basic YANG validation.

type StandardYANGValidator struct {
	registry *YANGModelRegistry

	mutex sync.RWMutex
}

// YANGNode represents a YANG schema node.

type YANGNode struct {
	Name string `json:"name"`

	Type string `json:"type"` // container, leaf, leaf-list, list, choice, case

	DataType string `json:"data_type,omitempty"`

	Description string `json:"description,omitempty"`

	Mandatory bool `json:"mandatory,omitempty"`

	Config bool `json:"config"`

	Children map[string]*YANGNode `json:"children,omitempty"`

	Keys []string `json:"keys,omitempty"`

	Constraints json.RawMessage `json:"constraints,omitempty"`
}

// NewYANGModelRegistry creates a new YANG model registry.

func NewYANGModelRegistry() *YANGModelRegistry {
	registry := &YANGModelRegistry{
		models: make(map[string]*YANGModel),

		modelsByNS: make(map[string]*YANGModel),

		validators: make(map[string]YANGValidator),

		loadedModules: make(map[string]time.Time),
	}

	// Register default validator.

	validator := &StandardYANGValidator{registry: registry}

	registry.validators["standard"] = validator

	// Load standard O-RAN YANG models.

	registry.loadORANModels()

	return registry
}

// loadORANModels loads standard O-RAN YANG models.

func (yr *YANGModelRegistry) loadORANModels() {
	logger := log.Log.WithName("yang-registry")

	// Define O-RAN standard models.

	oranModels := []*YANGModel{
		{
			Name: "o-ran-hardware",

			Namespace: "urn:o-ran:hardware:1.0",

			Version: "1.0",

			Revision: "2019-03-28",

			Description: "O-RAN Hardware Management YANG module",

			Organization: "O-RAN Alliance",

			Schema: json.RawMessage("{}"),

							Children: map[string]*YANGNode{
								"name": {
									Name: "name",

									Type: "leaf",

									DataType: "string",

									Mandatory: true,
								},

								"class": {
									Name: "class",

									Type: "leaf",

									DataType: "string",
								},

								"state": {
									Name: "state",

									Type: "container",

									Children: map[string]*YANGNode{
										"name": {
											Name: "name",

											Type: "leaf",

											DataType: "string",

											Config: false,
										},
									},
								},
							},
						},
					},
				},
			},
		},

		{
			Name: "o-ran-software-management",

			Namespace: "urn:o-ran:software-management:1.0",

			Version: "1.0",

			Revision: "2019-03-28",

			Description: "O-RAN Software Management YANG module",

			Organization: "O-RAN Alliance",

			Schema: json.RawMessage("{}"),

							Children: map[string]*YANGNode{
								"name": {
									Name: "name",

									Type: "leaf",

									DataType: "string",

									Mandatory: true,
								},

								"status": {
									Name: "status",

									Type: "leaf",

									DataType: "enumeration",

									Constraints: json.RawMessage("{}"),
									},
								},

								"active": {
									Name: "active",

									Type: "leaf",

									DataType: "boolean",
								},

								"running": {
									Name: "running",

									Type: "leaf",

									DataType: "boolean",
								},

								"access": {
									Name: "access",

									Type: "leaf",

									DataType: "enumeration",

									Constraints: json.RawMessage("{}"),
									},
								},

								"build-info": {
									Name: "build-info",

									Type: "container",

									Children: map[string]*YANGNode{
										"build-name": {
											Name: "build-name",

											Type: "leaf",

											DataType: "string",
										},

										"build-version": {
											Name: "build-version",

											Type: "leaf",

											DataType: "string",
										},
									},
								},
							},
						},
					},
				},
			},
		},

		{
			Name: "o-ran-performance-management",

			Namespace: "urn:o-ran:performance-management:1.0",

			Version: "1.0",

			Revision: "2019-03-28",

			Description: "O-RAN Performance Management YANG module",

			Organization: "O-RAN Alliance",

			Schema: json.RawMessage("{}"),

							Children: map[string]*YANGNode{
								"measurement-object-id": {
									Name: "measurement-object-id",

									Type: "leaf",

									DataType: "string",

									Mandatory: true,
								},

								"object-unit": {
									Name: "object-unit",

									Type: "leaf",

									DataType: "string",
								},

								"function": {
									Name: "function",

									Type: "leaf",

									DataType: "string",
								},

								"measurement-type": {
									Name: "measurement-type",

									Type: "list",

									Keys: []string{"measurement-type-id"},

									Children: map[string]*YANGNode{
										"measurement-type-id": {
											Name: "measurement-type-id",

											Type: "leaf",

											DataType: "string",

											Mandatory: true,
										},

										"measurement-description": {
											Name: "measurement-description",

											Type: "leaf",

											DataType: "string",
										},
									},
								},
							},
						},
					},
				},
			},
		},

		{
			Name: "o-ran-fault-management",

			Namespace: "urn:o-ran:fault-management:1.0",

			Version: "1.0",

			Revision: "2019-03-28",

			Description: "O-RAN Fault Management YANG module",

			Organization: "O-RAN Alliance",

			Schema: json.RawMessage("{}"),

							Children: map[string]*YANGNode{
								"fault-id": {
									Name: "fault-id",

									Type: "leaf",

									DataType: "uint16",

									Mandatory: true,
								},

								"fault-source": {
									Name: "fault-source",

									Type: "leaf",

									DataType: "string",

									Mandatory: true,
								},

								"fault-severity": {
									Name: "fault-severity",

									Type: "leaf",

									DataType: "enumeration",

									Constraints: json.RawMessage("{}"),
									},
								},

								"is-cleared": {
									Name: "is-cleared",

									Type: "leaf",

									DataType: "boolean",
								},

								"fault-text": {
									Name: "fault-text",

									Type: "leaf",

									DataType: "string",
								},

								"event-time": {
									Name: "event-time",

									Type: "leaf",

									DataType: "yang:date-and-time",
								},
							},
						},
					},
				},
			},
		},

		{
			Name: "ietf-interfaces",

			Namespace: "urn:ietf:params:xml:ns:yang:ietf-interfaces",

			Version: "1.0",

			Revision: "2018-02-20",

			Description: "IETF Interfaces YANG module",

			Organization: "IETF",

			Schema: json.RawMessage("{}"),

							Children: map[string]*YANGNode{
								"name": {
									Name: "name",

									Type: "leaf",

									DataType: "string",

									Mandatory: true,
								},

								"description": {
									Name: "description",

									Type: "leaf",

									DataType: "string",
								},

								"type": {
									Name: "type",

									Type: "leaf",

									DataType: "identityref",

									Mandatory: true,
								},

								"enabled": {
									Name: "enabled",

									Type: "leaf",

									DataType: "boolean",
								},
							},
						},
					},
				},
			},
		},
	}

	// Register all models.

	for _, model := range oranModels {

		model.LoadTime = time.Now()

		if err := yr.RegisterModel(model); err != nil {
			logger.Error(err, "failed to register O-RAN model", "model", model.Name)
		} else {
			logger.Info("registered O-RAN model", "model", model.Name, "namespace", model.Namespace)
		}

	}
}

// RegisterModel registers a YANG model in the registry.

func (yr *YANGModelRegistry) RegisterModel(model *YANGModel) error {
	yr.mutex.Lock()

	defer yr.mutex.Unlock()

	if model.Name == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	if model.Namespace == "" {
		return fmt.Errorf("model namespace cannot be empty")
	}

	// Check for conflicts.

	if existing, exists := yr.models[model.Name]; exists {
		if existing.Version != model.Version {
			return fmt.Errorf("model version conflict: %s exists with version %s, attempting to register version %s",

				model.Name, existing.Version, model.Version)
		}
	}

	yr.models[model.Name] = model

	yr.modelsByNS[model.Namespace] = model

	yr.loadedModules[model.Name] = time.Now()

	return nil
}

// GetModel retrieves a YANG model by name.

func (yr *YANGModelRegistry) GetModel(name string) (*YANGModel, error) {
	yr.mutex.RLock()

	defer yr.mutex.RUnlock()

	model, exists := yr.models[name]

	if !exists {
		return nil, fmt.Errorf("model not found: %s", name)
	}

	return model, nil
}

// GetModelByNamespace retrieves a YANG model by namespace.

func (yr *YANGModelRegistry) GetModelByNamespace(namespace string) (*YANGModel, error) {
	yr.mutex.RLock()

	defer yr.mutex.RUnlock()

	model, exists := yr.modelsByNS[namespace]

	if !exists {
		return nil, fmt.Errorf("model not found for namespace: %s", namespace)
	}

	return model, nil
}

// ListModels returns all registered models.

func (yr *YANGModelRegistry) ListModels() []*YANGModel {
	yr.mutex.RLock()

	defer yr.mutex.RUnlock()

	models := make([]*YANGModel, 0, len(yr.models))

	for _, model := range yr.models {
		models = append(models, model)
	}

	return models
}

// ValidateConfig validates configuration data against YANG schemas.

func (yr *YANGModelRegistry) ValidateConfig(ctx context.Context, data interface{}, modelName string) error {
	validator, exists := yr.validators["standard"]

	if !exists {
		return fmt.Errorf("no validator available")
	}

	return validator.ValidateData(data, modelName)
}

// ValidateXPath validates an XPath expression against YANG schemas.

func (yr *YANGModelRegistry) ValidateXPath(xpath, modelName string) error {
	validator, exists := yr.validators["standard"]

	if !exists {
		return fmt.Errorf("no validator available")
	}

	return validator.ValidateXPath(xpath, modelName)
}

// GetSchemaNode retrieves a specific schema node by path.

func (yr *YANGModelRegistry) GetSchemaNode(modelName, path string) (*YANGNode, error) {
	model, err := yr.GetModel(modelName)
	if err != nil {
		return nil, err
	}

	// Parse path and navigate to node.

	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(pathParts) == 0 {
		return nil, fmt.Errorf("invalid path")
	}

	// Start from root schema.

	rootNode, ok := model.Schema[pathParts[0]]

	if !ok {
		return nil, fmt.Errorf("root node not found: %s", pathParts[0])
	}

	currentNode, ok := rootNode.(*YANGNode)

	if !ok {
		return nil, fmt.Errorf("invalid root node type")
	}

	// Navigate through path.

	for i := 1; i < len(pathParts); i++ {

		if currentNode.Children == nil {
			return nil, fmt.Errorf("node has no children: %s", pathParts[i-1])
		}

		nextNode, exists := currentNode.Children[pathParts[i]]

		if !exists {
			return nil, fmt.Errorf("child node not found: %s", pathParts[i])
		}

		currentNode = nextNode

	}

	return currentNode, nil
}

// StandardYANGValidator implementation.

// ValidateData validates data against a YANG model.

func (sv *StandardYANGValidator) ValidateData(data interface{}, modelName string) error {
	model, err := sv.registry.GetModel(modelName)
	if err != nil {
		return fmt.Errorf("model validation failed: %w", err)
	}

	// Convert data to map for validation.

	var dataMap map[string]interface{}

	switch v := data.(type) {

	case map[string]interface{}:

		dataMap = v

	case string:

		// Try to parse as JSON.

		if err := json.Unmarshal([]byte(v), &dataMap); err != nil {
			return fmt.Errorf("failed to parse data as JSON: %w", err)
		}

	default:

		return fmt.Errorf("unsupported data type for validation")

	}

	// Validate against schema.

	return sv.validateNode(dataMap, model.Schema)
}

// validateNode validates a data node against schema node.

func (sv *StandardYANGValidator) validateNode(data, schema map[string]interface{}) error {
	for schemaKey, schemaValue := range schema {

		schemaNode, ok := schemaValue.(*YANGNode)

		if !ok {
			continue
		}

		dataValue, exists := data[schemaKey]

		// Check mandatory fields.

		if schemaNode.Mandatory && !exists {
			return fmt.Errorf("mandatory field missing: %s", schemaKey)
		}

		if !exists {
			continue
		}

		// Validate based on node type.

		switch schemaNode.Type {

		case "leaf":

			if err := sv.validateLeaf(dataValue, schemaNode); err != nil {
				return fmt.Errorf("validation failed for leaf %s: %w", schemaKey, err)
			}

		case "container":

			containerData, ok := dataValue.(map[string]interface{})

			if !ok {
				return fmt.Errorf("container %s must be an object", schemaKey)
			}

			if err := sv.validateNode(containerData, convertYANGChildren(schemaNode.Children)); err != nil {
				return fmt.Errorf("container validation failed for %s: %w", schemaKey, err)
			}

		case "list":

			listData, ok := dataValue.([]interface{})

			if !ok {
				return fmt.Errorf("list %s must be an array", schemaKey)
			}

			for i, item := range listData {

				itemMap, ok := item.(map[string]interface{})

				if !ok {
					return fmt.Errorf("list item %d in %s must be an object", i, schemaKey)
				}

				if err := sv.validateNode(itemMap, convertYANGChildren(schemaNode.Children)); err != nil {
					return fmt.Errorf("list item validation failed for %s[%d]: %w", schemaKey, i, err)
				}

			}

		}

	}

	return nil
}

// validateLeaf validates a leaf node value.

func (sv *StandardYANGValidator) validateLeaf(value interface{}, node *YANGNode) error {
	// Basic type validation.

	switch node.DataType {

	case "string":

		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}

	case "boolean":

		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}

	case "uint16", "uint32", "uint64", "int16", "int32", "int64":

		switch value.(type) {

		case int, int16, int32, int64, uint, uint16, uint32, uint64, float64:

			// OK.

		default:

			return fmt.Errorf("expected numeric type, got %T", value)

		}

	case "enumeration":

		strValue, ok := value.(string)

		if !ok {
			return fmt.Errorf("enumeration value must be string, got %T", value)
		}

		if node.Constraints != nil {
			if enumValues, exists := node.Constraints["enum"]; exists {

				validEnums, ok := enumValues.([]string)

				if ok {

					for _, validValue := range validEnums {
						if strValue == validValue {
							return nil
						}
					}

					return fmt.Errorf("invalid enumeration value: %s, valid values: %v", strValue, validEnums)

				}

			}
		}

	}

	return nil
}

// ValidateXPath validates an XPath expression.

func (sv *StandardYANGValidator) ValidateXPath(xpath, modelName string) error {
	// Basic XPath syntax validation.

	if xpath == "" {
		return fmt.Errorf("XPath cannot be empty")
	}

	// Check for basic XPath syntax.

	xpathRegex := regexp.MustCompile(`^(/[a-zA-Z0-9_-]+(\[[^\]]+\])*)+$`)

	if !xpathRegex.MatchString(xpath) {
		return fmt.Errorf("invalid XPath syntax: %s", xpath)
	}

	// Implement comprehensive XPath validation against schema.

	sv.mutex.RLock()

	model, exists := sv.registry.models[modelName]

	sv.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("model %s not found for XPath validation", modelName)
	}

	// Parse XPath into components for validation.

	parts := strings.Split(strings.TrimPrefix(xpath, "/"), "/")

	if len(parts) == 0 {
		return fmt.Errorf("invalid XPath structure: %s", xpath)
	}

	// Validate each XPath component against the model schema.

	var currentSchema interface{} = model.Schema

	for i, part := range parts {

		// Handle array indices [condition].

		nodeName := part

		var condition string

		if bracketStart := strings.Index(part, "["); bracketStart != -1 {

			bracketEnd := strings.Index(part, "]")

			if bracketEnd == -1 || bracketEnd <= bracketStart {
				return fmt.Errorf("invalid XPath bracket syntax in component: %s", part)
			}

			nodeName = part[:bracketStart]

			condition = part[bracketStart+1 : bracketEnd]

		}

		// Check if the node exists in the current schema level.

		if currentSchemaMap, ok := currentSchema.(map[string]interface{}); ok {
			if nodeSchema, exists := currentSchemaMap[nodeName]; exists {

				// Move to the next schema level.

				if nodeMap, ok := nodeSchema.(map[string]interface{}); ok {
					if childrenSchema, exists := nodeMap["children"]; exists {
						currentSchema = childrenSchema
					} else {
						// Leaf node - validate this is the last component.

						if i != len(parts)-1 {
							return fmt.Errorf("XPath attempts to traverse beyond leaf node: %s at %s", nodeName, xpath)
						}
					}
				}

				// Validate condition syntax if present.

				if condition != "" {
					if err := sv.validateXPathCondition(condition, nodeSchema); err != nil {
						return fmt.Errorf("invalid XPath condition '%s' in %s: %w", condition, part, err)
					}
				}

			} else {
				return fmt.Errorf("XPath component '%s' not found in model %s schema", nodeName, modelName)
			}
		} else if currentSchemaSlice, ok := currentSchema.([]interface{}); ok {

			// Handle array/list schemas.

			found := false

			for _, item := range currentSchemaSlice {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if name, exists := itemMap["name"]; exists && name == nodeName {
						if childrenSchema, exists := itemMap["children"]; exists {

							currentSchema = childrenSchema

							found = true

							break

						}
					}
				}
			}

			if !found {
				return fmt.Errorf("XPath component '%s' not found in list schema for model %s", nodeName, modelName)
			}

		} else {
			return fmt.Errorf("invalid schema structure at XPath component '%s' in model %s", nodeName, modelName)
		}

	}

	return nil
}

// validateXPathCondition validates XPath condition syntax against schema.

// It supports the following XPath condition patterns:.

//

// 1. Numeric indices: [1], [2], [3], etc.

//   - Used for selecting the nth occurrence of an element.

//   - Example: /network-functions/function[1].

//

// 2. Attribute conditions: [@attr='value'].

//   - Tests if an attribute equals a specific value.

//   - Validates that the attribute exists in the schema.

//   - Example: /interface[@name='eth0'].

//

// 3. Text content conditions: [text()='value'].

//   - Tests if the text content of a node equals a value.

//   - Example: /status[text()='active'].

//

// 4. Position conditions: [position()=N], [position()>N], [position()<N].

//   - Tests the position of a node in its sibling set.

//   - Supports operators: =, <, >, <=, >=.

//   - Example: /item[position()<=5].

//

// 5. Last position: [last()].

//   - Selects the last element in a node set.

//   - Example: /logs/entry[last()].

//

// 6. Complex conditions: condition1 and condition2, condition1 or condition2.

//   - Combines multiple conditions with logical operators.

//   - Recursively validates each sub-condition.

//   - Example: [@type='interface' and @status='up'].

//

// The function performs schema validation to ensure that referenced attributes.

// exist in the YANG model schema when available.

//

// Returns an error if:.

// - The condition is empty.

// - The syntax doesn't match any supported pattern.

// - An attribute referenced in the condition doesn't exist in the schema.

// - A sub-condition in a complex expression is invalid.

func (sv *StandardYANGValidator) validateXPathCondition(condition string, nodeSchema interface{}) error {
	// Basic condition validation - empty conditions are not allowed.

	if condition == "" {
		return fmt.Errorf("empty condition not allowed")
	}

	// Handle numeric indices like [1], [2], etc.

	if numericRegex := regexp.MustCompile(`^\d+$`); numericRegex.MatchString(condition) {
		return nil // Valid numeric index
	}

	// Handle attribute conditions like [@attr='value'].

	if attrRegex := regexp.MustCompile(`^@\w+\s*=\s*['"][^'"]*['"]$`); attrRegex.MatchString(condition) {

		// Extract attribute name and validate against schema.

		attrMatch := regexp.MustCompile(`^@(\w+)`).FindStringSubmatch(condition)

		if len(attrMatch) > 1 {

			attrName := attrMatch[1]

			// Check if attribute exists in schema.

			if nodeMap, ok := nodeSchema.(map[string]interface{}); ok {
				if attributes, exists := nodeMap["attributes"]; exists {
					if attrMap, ok := attributes.(map[string]interface{}); ok {
						if _, exists := attrMap[attrName]; !exists {
							return fmt.Errorf("attribute '%s' not found in schema", attrName)
						}
					}
				}
			}

		}

		return nil

	}

	// Handle node value conditions like [text()='value'].

	if textRegex := regexp.MustCompile(`^text\(\)\s*=\s*['"][^'"]*['"]$`); textRegex.MatchString(condition) {
		return nil // Valid text condition
	}

	// Handle position conditions like [position()=1].

	if positionRegex := regexp.MustCompile(`^position\(\)\s*[=<>]=?\s*\d+$`); positionRegex.MatchString(condition) {
		return nil // Valid position condition
	}

	// Handle last() condition.

	if condition == "last()" {
		return nil
	}

	// Handle complex conditions with logical operators.

	if complexRegex := regexp.MustCompile(`^.+\s+(and|or)\s+.+$`); complexRegex.MatchString(condition) {

		// Split by logical operators and validate each part.

		parts := regexp.MustCompile(`\s+(and|or)\s+`).Split(condition, -1)

		for _, part := range parts {
			if err := sv.validateXPathCondition(strings.TrimSpace(part), nodeSchema); err != nil {
				return fmt.Errorf("invalid condition part '%s': %w", part, err)
			}
		}

		return nil

	}

	return fmt.Errorf("unsupported condition syntax: %s", condition)
}

// GetModelInfo returns model information.

func (sv *StandardYANGValidator) GetModelInfo(modelName string) (*YANGModel, error) {
	return sv.registry.GetModel(modelName)
}

// Helper function to convert YANG children to schema format.

func convertYANGChildren(children map[string]*YANGNode) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range children {
		result[key] = value
	}

	return result
}

// GetStatistics returns registry statistics.

func (yr *YANGModelRegistry) GetStatistics() map[string]interface{} {
	yr.mutex.RLock()

	defer yr.mutex.RUnlock()

	stats := make(map[string]interface{})

	stats["total_models"] = len(yr.models)

	stats["loaded_modules"] = len(yr.loadedModules)

	stats["validators"] = len(yr.validators)

	modelStats := make(map[string]interface{})

	for name, model := range yr.models {
		modelStats[name] = json.RawMessage("{}")
	}

	stats["models"] = modelStats

	return stats
}
