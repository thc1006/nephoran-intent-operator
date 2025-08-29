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

// YANGModelRegistry manages YANG model schemas and validation
type YANGModelRegistry struct {
	models        map[string]*YANGModel
	modelsByNS    map[string]*YANGModel
	validators    map[string]YANGValidator
	mutex         sync.RWMutex
	loadedModules map[string]time.Time
}

// YANGModel represents a YANG model definition
// YANGModel defined in netconf_server.go

// YANGValidator provides validation functionality for YANG models
type YANGValidator interface {
	ValidateData(data interface{}, modelName string) error
	ValidateXPath(xpath string, modelName string) error
	GetModelInfo(modelName string) (*YANGModel, error)
}

// StandardYANGValidator implements basic YANG validation
type StandardYANGValidator struct {
	registry *YANGModelRegistry
	mutex    sync.RWMutex
}

// YANGNode represents a YANG schema node
type YANGNode struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // container, leaf, leaf-list, list, choice, case
	DataType    string                 `json:"data_type,omitempty"`
	Description string                 `json:"description,omitempty"`
	Mandatory   bool                   `json:"mandatory,omitempty"`
	Config      bool                   `json:"config"`
	Children    map[string]*YANGNode   `json:"children,omitempty"`
	Keys        []string               `json:"keys,omitempty"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
}

// NewYANGModelRegistry creates a new YANG model registry
func NewYANGModelRegistry() *YANGModelRegistry {
	registry := &YANGModelRegistry{
		models:        make(map[string]*YANGModel),
		modelsByNS:    make(map[string]*YANGModel),
		validators:    make(map[string]YANGValidator),
		loadedModules: make(map[string]time.Time),
	}

	// Register default validator
	validator := &StandardYANGValidator{registry: registry}
	registry.validators["standard"] = validator

	// Load standard O-RAN YANG models
	registry.loadORANModels()

	return registry
}

// loadORANModels loads standard O-RAN YANG models
func (yr *YANGModelRegistry) loadORANModels() {
	logger := log.Log.WithName("yang-registry")

	// Define O-RAN standard models
	oranModels := []*YANGModel{
		{
			Name:      "o-ran-hardware",
			Namespace: "urn:o-ran:hardware:1.0",
			Version:   "1.0",
			Content:   `<!-- O-RAN Hardware YANG model content would go here -->`,
			Features:  []string{"hardware-management"},
		},
		{
			Name:      "o-ran-software-management",
			Namespace: "urn:o-ran:software-management:1.0",
			Version:   "1.0",
			Content:   `<!-- O-RAN Software Management YANG model content would go here -->`,
			Features:  []string{"software-management"},
		},
		{
			Name:      "o-ran-performance-management",
			Namespace: "urn:o-ran:performance-management:1.0",
			Version:   "1.0",
			Content:   `<!-- O-RAN Performance Management YANG model content would go here -->`,
			Features:  []string{"performance-management"},
		},
		{
			Name:      "o-ran-fault-management",
			Namespace: "urn:o-ran:fault-management:1.0",
			Version:   "1.0",
			Content:   `<!-- O-RAN Fault Management YANG model content would go here -->`,
			Features:  []string{"fault-management"},
		},
		{
			Name:      "ietf-interfaces",
			Namespace: "urn:ietf:params:xml:ns:yang:ietf-interfaces",
			Version:   "1.0",
			Content:   `<!-- IETF Interfaces YANG model content would go here -->`,
			Features:  []string{"interfaces"},
		},
	}

	// Register all models
	for _, model := range oranModels {
		// TODO: Add LoadTime field to YANGModel if needed
		// model.LoadTime = time.Now()
		if err := yr.RegisterModel(model); err != nil {
			logger.Error(err, "failed to register O-RAN model", "model", model.Name)
		} else {
			logger.Info("registered O-RAN model", "model", model.Name, "namespace", model.Namespace)
		}
	}
}

// RegisterModel registers a YANG model in the registry
func (yr *YANGModelRegistry) RegisterModel(model *YANGModel) error {
	yr.mutex.Lock()
	defer yr.mutex.Unlock()

	if model.Name == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	if model.Namespace == "" {
		return fmt.Errorf("model namespace cannot be empty")
	}

	// Check for conflicts
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

// GetModel retrieves a YANG model by name
func (yr *YANGModelRegistry) GetModel(name string) (*YANGModel, error) {
	yr.mutex.RLock()
	defer yr.mutex.RUnlock()

	model, exists := yr.models[name]
	if !exists {
		return nil, fmt.Errorf("model not found: %s", name)
	}

	return model, nil
}

// GetModelByNamespace retrieves a YANG model by namespace
func (yr *YANGModelRegistry) GetModelByNamespace(namespace string) (*YANGModel, error) {
	yr.mutex.RLock()
	defer yr.mutex.RUnlock()

	model, exists := yr.modelsByNS[namespace]
	if !exists {
		return nil, fmt.Errorf("model not found for namespace: %s", namespace)
	}

	return model, nil
}

// ListModels returns all registered models
func (yr *YANGModelRegistry) ListModels() []*YANGModel {
	yr.mutex.RLock()
	defer yr.mutex.RUnlock()

	models := make([]*YANGModel, 0, len(yr.models))
	for _, model := range yr.models {
		models = append(models, model)
	}

	return models
}

// ValidateConfig validates configuration data against YANG schemas
func (yr *YANGModelRegistry) ValidateConfig(ctx context.Context, data interface{}, modelName string) error {
	validator, exists := yr.validators["standard"]
	if !exists {
		return fmt.Errorf("no validator available")
	}

	return validator.ValidateData(data, modelName)
}

// ValidateXPath validates an XPath expression against YANG schemas
func (yr *YANGModelRegistry) ValidateXPath(xpath string, modelName string) error {
	validator, exists := yr.validators["standard"]
	if !exists {
		return fmt.Errorf("no validator available")
	}

	return validator.ValidateXPath(xpath, modelName)
}

// GetSchemaNode retrieves a specific schema node by path
func (yr *YANGModelRegistry) GetSchemaNode(modelName string, path string) (*YANGNode, error) {
	_, err := yr.GetModel(modelName)
	if err != nil {
		return nil, err
	}

	// Parse path and navigate to node
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathParts) == 0 {
		return nil, fmt.Errorf("invalid path")
	}

	// TODO: Add Schema field to YANGModel to support schema navigation
	// rootNode, ok := model.Schema[pathParts[0]]
	// if !ok {
	//     return nil, fmt.Errorf("root node not found: %s", pathParts[0])
	// }
	return nil, fmt.Errorf("schema navigation not supported - Schema field missing from YANGModel")

	// currentNode, ok := rootNode.(*YANGNode)
	// if !ok {
	//     return nil, fmt.Errorf("invalid root node type")
	// }

	// // Navigate through path
	// for i := 1; i < len(pathParts); i++ {
	//     if currentNode.Children == nil {
	//         return nil, fmt.Errorf("node has no children: %s", pathParts[i-1])
	//     }

	//     nextNode, exists := currentNode.Children[pathParts[i]]
	//     if !exists {
	//         return nil, fmt.Errorf("child node not found: %s", pathParts[i])
	//     }

	//     currentNode = nextNode
	// }

	// return currentNode, nil
}

// StandardYANGValidator implementation

// ValidateData validates data against a YANG model
func (sv *StandardYANGValidator) ValidateData(data interface{}, modelName string) error {
	_, err := sv.registry.GetModel(modelName)
	if err != nil {
		return fmt.Errorf("model validation failed: %w", err)
	}

	// Convert data to map for validation
	var dataMap map[string]interface{}
	switch v := data.(type) {
	case map[string]interface{}:
		dataMap = v
	case string:
		// Try to parse as JSON
		if err := json.Unmarshal([]byte(v), &dataMap); err != nil {
			return fmt.Errorf("failed to parse data as JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported data type for validation")
	}

	// TODO: Add Schema field to YANGModel to support validation
	// return sv.validateNode(dataMap, model.Schema)
	return fmt.Errorf("validation not supported - Schema field missing from YANGModel")
}

// validateNode validates a data node against schema node
func (sv *StandardYANGValidator) validateNode(data map[string]interface{}, schema map[string]interface{}) error {
	for schemaKey, schemaValue := range schema {
		schemaNode, ok := schemaValue.(*YANGNode)
		if !ok {
			continue
		}

		dataValue, exists := data[schemaKey]

		// Check mandatory fields
		if schemaNode.Mandatory && !exists {
			return fmt.Errorf("mandatory field missing: %s", schemaKey)
		}

		if !exists {
			continue
		}

		// Validate based on node type
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

// validateLeaf validates a leaf node value
func (sv *StandardYANGValidator) validateLeaf(value interface{}, node *YANGNode) error {
	// Basic type validation
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
			// OK
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

// ValidateXPath validates an XPath expression
func (sv *StandardYANGValidator) ValidateXPath(xpath string, modelName string) error {
	// Basic XPath syntax validation
	if xpath == "" {
		return fmt.Errorf("XPath cannot be empty")
	}

	// Check for basic XPath syntax
	xpathRegex := regexp.MustCompile(`^(/[a-zA-Z0-9_-]+(\[[^\]]+\])*)+$`)
	if !xpathRegex.MatchString(xpath) {
		return fmt.Errorf("invalid XPath syntax: %s", xpath)
	}

	// Implement comprehensive XPath validation against schema
	sv.mutex.RLock()
	_, exists := sv.registry.models[modelName]
	sv.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("model %s not found for XPath validation", modelName)
	}

	// Parse XPath into components for validation
	parts := strings.Split(strings.TrimPrefix(xpath, "/"), "/")
	if len(parts) == 0 {
		return fmt.Errorf("invalid XPath structure: %s", xpath)
	}

	// TODO: Add Schema field to YANGModel to support XPath validation
	// var currentSchema interface{} = model.Schema
	return fmt.Errorf("XPath validation not supported - Schema field missing from YANGModel")

	// Commented out entire XPath validation function due to missing Schema field
	// for i, part := range parts {
	//     ... [entire function body commented out]
	// }
	// return nil
}

// validateXPathCondition validates XPath condition syntax against schema.
// It supports the following XPath condition patterns:
//
// 1. Numeric indices: [1], [2], [3], etc.
//   - Used for selecting the nth occurrence of an element
//   - Example: /network-functions/function[1]
//
// 2. Attribute conditions: [@attr='value']
//   - Tests if an attribute equals a specific value
//   - Validates that the attribute exists in the schema
//   - Example: /interface[@name='eth0']
//
// 3. Text content conditions: [text()='value']
//   - Tests if the text content of a node equals a value
//   - Example: /status[text()='active']
//
// 4. Position conditions: [position()=N], [position()>N], [position()<N]
//   - Tests the position of a node in its sibling set
//   - Supports operators: =, <, >, <=, >=
//   - Example: /item[position()<=5]
//
// 5. Last position: [last()]
//   - Selects the last element in a node set
//   - Example: /logs/entry[last()]
//
// 6. Complex conditions: condition1 and condition2, condition1 or condition2
//   - Combines multiple conditions with logical operators
//   - Recursively validates each sub-condition
//   - Example: [@type='interface' and @status='up']
//
// The function performs schema validation to ensure that referenced attributes
// exist in the YANG model schema when available.
//
// Returns an error if:
// - The condition is empty
// - The syntax doesn't match any supported pattern
// - An attribute referenced in the condition doesn't exist in the schema
// - A sub-condition in a complex expression is invalid
func (sv *StandardYANGValidator) validateXPathCondition(condition string, nodeSchema interface{}) error {
	// Basic condition validation - empty conditions are not allowed
	if condition == "" {
		return fmt.Errorf("empty condition not allowed")
	}

	// Handle numeric indices like [1], [2], etc.
	if numericRegex := regexp.MustCompile(`^\d+$`); numericRegex.MatchString(condition) {
		return nil // Valid numeric index
	}

	// Handle attribute conditions like [@attr='value']
	if attrRegex := regexp.MustCompile(`^@\w+\s*=\s*['"][^'"]*['"]$`); attrRegex.MatchString(condition) {
		// Extract attribute name and validate against schema
		attrMatch := regexp.MustCompile(`^@(\w+)`).FindStringSubmatch(condition)
		if len(attrMatch) > 1 {
			attrName := attrMatch[1]
			// Check if attribute exists in schema
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

	// Handle node value conditions like [text()='value']
	if textRegex := regexp.MustCompile(`^text\(\)\s*=\s*['"][^'"]*['"]$`); textRegex.MatchString(condition) {
		return nil // Valid text condition
	}

	// Handle position conditions like [position()=1]
	if positionRegex := regexp.MustCompile(`^position\(\)\s*[=<>]=?\s*\d+$`); positionRegex.MatchString(condition) {
		return nil // Valid position condition
	}

	// Handle last() condition
	if condition == "last()" {
		return nil
	}

	// Handle complex conditions with logical operators
	if complexRegex := regexp.MustCompile(`^.+\s+(and|or)\s+.+$`); complexRegex.MatchString(condition) {
		// Split by logical operators and validate each part
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

// GetModelInfo returns model information
func (sv *StandardYANGValidator) GetModelInfo(modelName string) (*YANGModel, error) {
	return sv.registry.GetModel(modelName)
}

// Helper function to convert YANG children to schema format
func convertYANGChildren(children map[string]*YANGNode) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range children {
		result[key] = value
	}
	return result
}

// GetStatistics returns registry statistics
func (yr *YANGModelRegistry) GetStatistics() map[string]interface{} {
	yr.mutex.RLock()
	defer yr.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_models"] = len(yr.models)
	stats["loaded_modules"] = len(yr.loadedModules)
	stats["validators"] = len(yr.validators)

	modelStats := make(map[string]interface{})
	for name, model := range yr.models {
		modelStats[name] = map[string]interface{}{
			"version": model.Version,
			// "revision":   model.Revision, // Field not available in YANGModel
			"namespace": model.Namespace,
			// "load_time":  model.LoadTime, // Field not available in YANGModel
			// "has_schema": len(model.Schema) > 0, // Field not available in YANGModel
			"features": len(model.Features),
		}
	}
	stats["models"] = modelStats

	return stats
}

// LoadStandardO1Models loads standard O1 models for K8s 1.31+ compatibility
func (yr *YANGModelRegistry) LoadStandardO1Models() error {
	logger := log.Log.WithName("yang-registry")
	logger.Info("Loading standard O1 YANG models for K8s 1.31+")

	// Load additional O1-specific models
	o1Models := []*YANGModel{
		{
			Name:      "o1-security-management",
			Namespace: "urn:o-ran:o1:security-management:1.0",
			Version:   "1.0",
			Content:   `<!-- O1 Security Management YANG model for K8s 1.31+ -->`,
			Features:  []string{"security-management", "tls-config", "authentication"},
		},
		{
			Name:      "o1-streaming-management",
			Namespace: "urn:o-ran:o1:streaming-management:1.0",
			Version:   "1.0",
			Content:   `<!-- O1 Streaming Management YANG model for K8s 1.31+ -->`,
			Features:  []string{"streaming-management", "websocket", "compression"},
		},
		{
			Name:      "o1-kubernetes-integration",
			Namespace: "urn:o-ran:o1:kubernetes:1.0",
			Version:   "1.0",
			Content:   `<!-- O1 Kubernetes Integration YANG model for K8s 1.31+ -->`,
			Features:  []string{"kubernetes-integration", "cel-validation", "custom-resources"},
		},
		{
			Name:      "o1-observability",
			Namespace: "urn:o-ran:o1:observability:1.0",
			Version:   "1.0",
			Content:   `<!-- O1 Observability YANG model for K8s 1.31+ -->`,
			Features:  []string{"observability", "metrics", "tracing", "logging"},
		},
	}

	// Register all O1 models
	for _, model := range o1Models {
		if err := yr.RegisterModel(model); err != nil {
			logger.Error(err, "Failed to register O1 model", "model", model.Name)
			return fmt.Errorf("failed to register O1 model %s: %w", model.Name, err)
		}
		logger.Info("Successfully loaded O1 model", "name", model.Name, "version", model.Version)
	}

	// Load existing O-RAN models too
	yr.loadORANModels()

	logger.Info("Successfully loaded all standard O1 models for K8s 1.31+", "total_models", len(o1Models))
	return nil
}
