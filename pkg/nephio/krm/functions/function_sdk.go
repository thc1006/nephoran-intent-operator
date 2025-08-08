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

package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// KRMFunction defines the interface for all KRM functions
// This interface provides a standardized way to implement network function transformations
type KRMFunction interface {
	// Execute transforms the input resource list and returns the transformed resources
	Execute(ctx context.Context, input *ResourceList) (*ResourceList, error)
	
	// Validate checks if the function configuration and input are valid
	Validate(ctx context.Context, config *FunctionConfig) error
	
	// GetMetadata returns metadata about this function
	GetMetadata() *FunctionMetadata
	
	// GetSchema returns the JSON schema for function configuration
	GetSchema() *FunctionSchema
}

// ResourceList represents a list of KRM resources with metadata
type ResourceList struct {
	// Items contains the actual Kubernetes resources
	Items []porch.KRMResource `json:"items"`
	
	// FunctionConfig contains configuration passed to the function
	FunctionConfig map[string]interface{} `json:"functionConfig,omitempty"`
	
	// Results contains any results or messages from processing
	Results []*porch.FunctionResult `json:"results,omitempty"`
	
	// Context provides execution context information
	Context *ExecutionContext `json:"context,omitempty"`
}

// FunctionConfig represents configuration for a KRM function
type FunctionConfig struct {
	// APIVersion of the function config
	APIVersion string `json:"apiVersion"`
	
	// Kind of the function config
	Kind string `json:"kind"`
	
	// Metadata for the function config
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	
	// Data contains the actual configuration
	Data map[string]interface{} `json:"data,omitempty"`
	
	// Spec contains structured configuration
	Spec map[string]interface{} `json:"spec,omitempty"`
}

// ExecutionContext provides context information for function execution
type ExecutionContext struct {
	// Package information
	Package *PackageContext `json:"package,omitempty"`
	
	// Pipeline information
	Pipeline *PipelineContext `json:"pipeline,omitempty"`
	
	// Environment variables
	Environment map[string]string `json:"environment,omitempty"`
	
	// Execution metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	
	// Timing information
	StartTime time.Time `json:"startTime"`
	Timeout   time.Duration `json:"timeout,omitempty"`
}

// PackageContext provides package-specific context
type PackageContext struct {
	Repository  string `json:"repository"`
	PackageName string `json:"packageName"`
	Revision    string `json:"revision"`
	Lifecycle   string `json:"lifecycle"`
}

// PipelineContext provides pipeline-specific context
type PipelineContext struct {
	Name     string `json:"name"`
	Stage    string `json:"stage,omitempty"`
	Function string `json:"function,omitempty"`
}

// FunctionMetadata contains comprehensive metadata about a function
type FunctionMetadata struct {
	// Basic identification
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	
	// Function classification
	Type        FunctionType `json:"type"`
	Categories  []string     `json:"categories,omitempty"`
	Keywords    []string     `json:"keywords,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	
	// Resource type support
	ResourceTypes []ResourceTypeSupport `json:"resourceTypes,omitempty"`
	APIVersions   []string              `json:"apiVersions,omitempty"`
	
	// Performance characteristics
	Performance *PerformanceProfile `json:"performance,omitempty"`
	
	// Security profile
	Security *SecurityProfile `json:"security,omitempty"`
	
	// Telecom-specific metadata
	Telecom *TelecomProfile `json:"telecom,omitempty"`
	
	// Author and licensing
	Author  string `json:"author,omitempty"`
	License string `json:"license,omitempty"`
	
	// Documentation
	Documentation *DocumentationLinks `json:"documentation,omitempty"`
	
	// Examples
	Examples []*FunctionExample `json:"examples,omitempty"`
}

// FunctionType defines the type of KRM function
type FunctionType string

const (
	FunctionTypeMutator   FunctionType = "mutator"
	FunctionTypeValidator FunctionType = "validator"
	FunctionTypeGenerator FunctionType = "generator"
	FunctionTypeComposer  FunctionType = "composer"
)

// ResourceTypeSupport defines support for specific resource types
type ResourceTypeSupport struct {
	Group      string   `json:"group,omitempty"`
	Version    string   `json:"version"`
	Kind       string   `json:"kind"`
	Operations []string `json:"operations"` // create, read, update, delete
	Required   bool     `json:"required,omitempty"`
}

// PerformanceProfile defines performance characteristics
type PerformanceProfile struct {
	TypicalExecutionTime time.Duration      `json:"typicalExecutionTime,omitempty"`
	MaxExecutionTime     time.Duration      `json:"maxExecutionTime,omitempty"`
	MemoryUsage          resource.Quantity  `json:"memoryUsage,omitempty"`
	CPUUsage             resource.Quantity  `json:"cpuUsage,omitempty"`
	ScalabilityLimits    *ScalabilityLimits `json:"scalabilityLimits,omitempty"`
}

// ScalabilityLimits defines scalability constraints
type ScalabilityLimits struct {
	MaxResources    int `json:"maxResources,omitempty"`
	MaxComplexity   int `json:"maxComplexity,omitempty"`
	ParallelSafe    bool `json:"parallelSafe,omitempty"`
	ConcurrencyLimit int `json:"concurrencyLimit,omitempty"`
}

// SecurityProfile defines security characteristics
type SecurityProfile struct {
	RequiredCapabilities []string `json:"requiredCapabilities,omitempty"`
	DroppedCapabilities  []string `json:"droppedCapabilities,omitempty"`
	NetworkAccess        bool     `json:"networkAccess,omitempty"`
	FileSystemAccess     []string `json:"fileSystemAccess,omitempty"`
	PrivilegeEscalation  bool     `json:"privilegeEscalation,omitempty"`
	ReadOnlyRootFS       bool     `json:"readOnlyRootFS,omitempty"`
	RunAsNonRoot         bool     `json:"runAsNonRoot,omitempty"`
}

// TelecomProfile defines telecommunications-specific characteristics
type TelecomProfile struct {
	// Standards compliance
	Standards []StandardCompliance `json:"standards,omitempty"`
	
	// O-RAN interfaces supported
	ORANInterfaces []ORANInterfaceSupport `json:"oranInterfaces,omitempty"`
	
	// Network function types
	NetworkFunctionTypes []string `json:"networkFunctionTypes,omitempty"`
	
	// 5G capabilities
	FiveGCapabilities []string `json:"5gCapabilities,omitempty"`
	
	// Network slice support
	NetworkSliceSupport bool `json:"networkSliceSupport,omitempty"`
	
	// Edge computing support
	EdgeSupport bool `json:"edgeSupport,omitempty"`
}

// StandardCompliance defines compliance with industry standards
type StandardCompliance struct {
	Name     string `json:"name"`     // 3GPP, O-RAN, ETSI, etc.
	Version  string `json:"version"`
	Release  string `json:"release,omitempty"`
	Sections []string `json:"sections,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// ORANInterfaceSupport defines O-RAN interface support
type ORANInterfaceSupport struct {
	Interface string `json:"interface"` // A1, O1, O2, E2, etc.
	Version   string `json:"version"`
	Role      string `json:"role,omitempty"` // consumer, producer, both
	Required  bool   `json:"required,omitempty"`
}

// DocumentationLinks provides links to documentation
type DocumentationLinks struct {
	README   string `json:"readme,omitempty"`
	API      string `json:"api,omitempty"`
	Examples string `json:"examples,omitempty"`
	Guide    string `json:"guide,omitempty"`
}

// FunctionExample provides a usage example
type FunctionExample struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Config      *FunctionConfig        `json:"config"`
	Input       []porch.KRMResource    `json:"input"`
	Output      []porch.KRMResource    `json:"output"`
	Explanation string                 `json:"explanation,omitempty"`
}

// FunctionSchema defines the JSON schema for function configuration
type FunctionSchema struct {
	Type        string                    `json:"type"`
	Properties  map[string]*SchemaProperty `json:"properties,omitempty"`
	Required    []string                   `json:"required,omitempty"`
	Definitions map[string]*FunctionSchema `json:"definitions,omitempty"`
}

// SchemaProperty defines a property in the function schema
type SchemaProperty struct {
	Type         string                 `json:"type"`
	Description  string                 `json:"description,omitempty"`
	Default      interface{}            `json:"default,omitempty"`
	Examples     []interface{}          `json:"examples,omitempty"`
	Enum         []interface{}          `json:"enum,omitempty"`
	Properties   map[string]*SchemaProperty `json:"properties,omitempty"`
	Items        *SchemaProperty        `json:"items,omitempty"`
	Pattern      string                 `json:"pattern,omitempty"`
	MinLength    *int                   `json:"minLength,omitempty"`
	MaxLength    *int                   `json:"maxLength,omitempty"`
	Minimum      *float64               `json:"minimum,omitempty"`
	Maximum      *float64               `json:"maximum,omitempty"`
}

// BaseFunctionImpl provides a base implementation for KRM functions
type BaseFunctionImpl struct {
	metadata *FunctionMetadata
	schema   *FunctionSchema
}

// NewBaseFunctionImpl creates a new base function implementation
func NewBaseFunctionImpl(metadata *FunctionMetadata, schema *FunctionSchema) *BaseFunctionImpl {
	return &BaseFunctionImpl{
		metadata: metadata,
		schema:   schema,
	}
}

// GetMetadata returns the function metadata
func (f *BaseFunctionImpl) GetMetadata() *FunctionMetadata {
	return f.metadata
}

// GetSchema returns the function schema
func (f *BaseFunctionImpl) GetSchema() *FunctionSchema {
	return f.schema
}

// Validate provides basic configuration validation
func (f *BaseFunctionImpl) Validate(ctx context.Context, config *FunctionConfig) error {
	if config == nil {
		return fmt.Errorf("function configuration is required")
	}
	
	// Validate against schema if available
	if f.schema != nil {
		return f.validateAgainstSchema(config, f.schema)
	}
	
	return nil
}

// Helper methods for common operations

// FindResourcesByGVK finds resources by GroupVersionKind
func FindResourcesByGVK(resources []porch.KRMResource, gvk schema.GroupVersionKind) []porch.KRMResource {
	var matches []porch.KRMResource
	
	for _, resource := range resources {
		if resource.Kind == gvk.Kind {
			// Check API version
			if gvk.Group == "" {
				if resource.APIVersion == gvk.Version {
					matches = append(matches, resource)
				}
			} else {
				expectedAPIVersion := gvk.Group + "/" + gvk.Version
				if resource.APIVersion == expectedAPIVersion {
					matches = append(matches, resource)
				}
			}
		}
	}
	
	return matches
}

// FindResourceByName finds a resource by name and kind
func FindResourceByName(resources []porch.KRMResource, kind, name string) *porch.KRMResource {
	for _, resource := range resources {
		if resource.Kind == kind {
			if resourceName, ok := resource.Metadata["name"].(string); ok && resourceName == name {
				return &resource
			}
		}
	}
	return nil
}

// GetResourceName safely extracts the name from resource metadata
func GetResourceName(resource *porch.KRMResource) (string, error) {
	if resource.Metadata == nil {
		return "", fmt.Errorf("resource metadata is nil")
	}
	
	name, ok := resource.Metadata["name"].(string)
	if !ok {
		return "", fmt.Errorf("resource name not found or not a string")
	}
	
	return name, nil
}

// GetResourceNamespace safely extracts the namespace from resource metadata
func GetResourceNamespace(resource *porch.KRMResource) string {
	if resource.Metadata == nil {
		return ""
	}
	
	namespace, ok := resource.Metadata["namespace"].(string)
	if !ok {
		return ""
	}
	
	return namespace
}

// SetResourceAnnotation sets an annotation on a resource
func SetResourceAnnotation(resource *porch.KRMResource, key, value string) {
	if resource.Metadata == nil {
		resource.Metadata = make(map[string]interface{})
	}
	
	annotations, ok := resource.Metadata["annotations"].(map[string]interface{})
	if !ok {
		annotations = make(map[string]interface{})
		resource.Metadata["annotations"] = annotations
	}
	
	annotations[key] = value
}

// GetResourceAnnotation gets an annotation from a resource
func GetResourceAnnotation(resource *porch.KRMResource, key string) (string, bool) {
	if resource.Metadata == nil {
		return "", false
	}
	
	annotations, ok := resource.Metadata["annotations"].(map[string]interface{})
	if !ok {
		return "", false
	}
	
	value, ok := annotations[key].(string)
	return value, ok
}

// SetResourceLabel sets a label on a resource
func SetResourceLabel(resource *porch.KRMResource, key, value string) {
	if resource.Metadata == nil {
		resource.Metadata = make(map[string]interface{})
	}
	
	labels, ok := resource.Metadata["labels"].(map[string]interface{})
	if !ok {
		labels = make(map[string]interface{})
		resource.Metadata["labels"] = labels
	}
	
	labels[key] = value
}

// GetResourceLabel gets a label from a resource
func GetResourceLabel(resource *porch.KRMResource, key string) (string, bool) {
	if resource.Metadata == nil {
		return "", false
	}
	
	labels, ok := resource.Metadata["labels"].(map[string]interface{})
	if !ok {
		return "", false
	}
	
	value, ok := labels[key].(string)
	return value, ok
}

// HasLabel checks if resource has a specific label
func HasLabel(resource *porch.KRMResource, key string) bool {
	_, exists := GetResourceLabel(resource, key)
	return exists
}

// HasAnnotation checks if resource has a specific annotation
func HasAnnotation(resource *porch.KRMResource, key string) bool {
	_, exists := GetResourceAnnotation(resource, key)
	return exists
}

// MatchesLabelSelector checks if resource matches label selector
func MatchesLabelSelector(resource *porch.KRMResource, selector map[string]string) bool {
	if len(selector) == 0 {
		return true
	}
	
	for key, expectedValue := range selector {
		actualValue, exists := GetResourceLabel(resource, key)
		if !exists || actualValue != expectedValue {
			return false
		}
	}
	
	return true
}

// CreateResult creates a function result message
func CreateResult(severity, message string, tags map[string]string) *porch.FunctionResult {
	result := &porch.FunctionResult{
		Message:  message,
		Severity: severity,
	}
	
	if tags != nil {
		result.Tags = tags
	}
	
	return result
}

// CreateInfo creates an info-level result
func CreateInfo(message string, tags ...map[string]string) *porch.FunctionResult {
	var tagMap map[string]string
	if len(tags) > 0 {
		tagMap = tags[0]
	}
	return CreateResult("info", message, tagMap)
}

// CreateWarning creates a warning-level result
func CreateWarning(message string, tags ...map[string]string) *porch.FunctionResult {
	var tagMap map[string]string
	if len(tags) > 0 {
		tagMap = tags[0]
	}
	return CreateResult("warning", message, tagMap)
}

// CreateError creates an error-level result
func CreateError(message string, tags ...map[string]string) *porch.FunctionResult {
	var tagMap map[string]string
	if len(tags) > 0 {
		tagMap = tags[0]
	}
	return CreateResult("error", message, tagMap)
}

// ValidateResourceList validates a resource list
func ValidateResourceList(rl *ResourceList) error {
	if rl == nil {
		return fmt.Errorf("resource list cannot be nil")
	}
	
	if len(rl.Items) == 0 {
		return fmt.Errorf("resource list cannot be empty")
	}
	
	// Validate each resource
	for i, resource := range rl.Items {
		if err := ValidateKRMResource(&resource); err != nil {
			return fmt.Errorf("resource %d is invalid: %w", i, err)
		}
	}
	
	return nil
}

// ValidateKRMResource validates a single KRM resource
func ValidateKRMResource(resource *porch.KRMResource) error {
	if resource.APIVersion == "" {
		return fmt.Errorf("apiVersion is required")
	}
	
	if resource.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	
	if resource.Metadata == nil {
		return fmt.Errorf("metadata is required")
	}
	
	name, ok := resource.Metadata["name"].(string)
	if !ok || name == "" {
		return fmt.Errorf("metadata.name is required and must be a non-empty string")
	}
	
	return nil
}

// DeepCopyResource creates a deep copy of a KRM resource
func DeepCopyResource(resource *porch.KRMResource) (*porch.KRMResource, error) {
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource: %w", err)
	}
	
	var copy porch.KRMResource
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
	}
	
	return &copy, nil
}

// DeepCopyResourceList creates a deep copy of a resource list
func DeepCopyResourceList(rl *ResourceList) (*ResourceList, error) {
	data, err := json.Marshal(rl)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource list: %w", err)
	}
	
	var copy ResourceList
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource list: %w", err)
	}
	
	return &copy, nil
}

// FilterResourcesByNamespace filters resources by namespace
func FilterResourcesByNamespace(resources []porch.KRMResource, namespace string) []porch.KRMResource {
	var filtered []porch.KRMResource
	
	for _, resource := range resources {
		resourceNamespace := GetResourceNamespace(&resource)
		if resourceNamespace == namespace {
			filtered = append(filtered, resource)
		}
	}
	
	return filtered
}

// FilterResourcesByLabel filters resources by label
func FilterResourcesByLabel(resources []porch.KRMResource, key, value string) []porch.KRMResource {
	var filtered []porch.KRMResource
	
	for _, resource := range resources {
		if labelValue, exists := GetResourceLabel(&resource, key); exists && labelValue == value {
			filtered = append(filtered, resource)
		}
	}
	
	return filtered
}

// GetSpecField safely gets a field from resource spec
func GetSpecField(resource *porch.KRMResource, fieldPath string) (interface{}, error) {
	if resource.Spec == nil {
		return nil, fmt.Errorf("resource spec is nil")
	}
	
	return getNestedField(resource.Spec, fieldPath)
}

// SetSpecField safely sets a field in resource spec
func SetSpecField(resource *porch.KRMResource, fieldPath string, value interface{}) error {
	if resource.Spec == nil {
		resource.Spec = make(map[string]interface{})
	}
	
	return setNestedField(resource.Spec, fieldPath, value)
}

// GetStatusField safely gets a field from resource status
func GetStatusField(resource *porch.KRMResource, fieldPath string) (interface{}, error) {
	if resource.Status == nil {
		return nil, fmt.Errorf("resource status is nil")
	}
	
	return getNestedField(resource.Status, fieldPath)
}

// SetStatusField safely sets a field in resource status
func SetStatusField(resource *porch.KRMResource, fieldPath string, value interface{}) error {
	if resource.Status == nil {
		resource.Status = make(map[string]interface{})
	}
	
	return setNestedField(resource.Status, fieldPath, value)
}

// Private helper methods

func (f *BaseFunctionImpl) validateAgainstSchema(config *FunctionConfig, schema *FunctionSchema) error {
	// Basic schema validation - in production, use a proper JSON schema validator
	if schema.Required != nil {
		for _, requiredField := range schema.Required {
			if _, exists := config.Data[requiredField]; !exists {
				return fmt.Errorf("required field '%s' is missing", requiredField)
			}
		}
	}
	
	return nil
}

func getNestedField(data map[string]interface{}, fieldPath string) (interface{}, error) {
	parts := strings.Split(fieldPath, ".")
	current := data
	
	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return nil, fmt.Errorf("field '%s' not found", fieldPath)
		}
		
		if i == len(parts)-1 {
			return value, nil
		}
		
		next, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field '%s' is not a map", strings.Join(parts[:i+1], "."))
		}
		
		current = next
	}
	
	return nil, fmt.Errorf("empty field path")
}

func setNestedField(data map[string]interface{}, fieldPath string, value interface{}) error {
	parts := strings.Split(fieldPath, ".")
	current := data
	
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return nil
		}
		
		if existing, exists := current[part]; exists {
			if next, ok := existing.(map[string]interface{}); ok {
				current = next
			} else {
				return fmt.Errorf("field '%s' exists but is not a map", strings.Join(parts[:i+1], "."))
			}
		} else {
			next := make(map[string]interface{})
			current[part] = next
			current = next
		}
	}
	
	return fmt.Errorf("empty field path")
}

// Logging helpers for functions
func LogInfo(ctx context.Context, msg string, keysAndValues ...interface{}) {
	logger := log.FromContext(ctx).WithName("krm-function")
	logger.Info(msg, keysAndValues...)
}

func LogError(ctx context.Context, err error, msg string, keysAndValues ...interface{}) {
	logger := log.FromContext(ctx).WithName("krm-function")
	logger.Error(err, msg, keysAndValues...)
}

func LogWarning(ctx context.Context, msg string, keysAndValues ...interface{}) {
	logger := log.FromContext(ctx).WithName("krm-function")
	logger.V(1).Info("WARNING: "+msg, keysAndValues...)
}