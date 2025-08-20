package testdata

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IntentData represents the structure of a network intent
// This uses the flat structure expected by validateIntentFields
type IntentData struct {
	IntentType string                 `json:"intent_type"`
	Target     string                 `json:"target"`
	Namespace  string                 `json:"namespace"`
	Replicas   int                    `json:"replicas,omitempty"`
	Source     string                 `json:"source,omitempty"`
	Resources  map[string]interface{} `json:"resources,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	Timestamp  string                 `json:"timestamp,omitempty"`

	// Keep legacy fields for invalid test cases that expect object targets
	APIVersion *string    `json:"apiVersion,omitempty"`
	Kind       *string    `json:"kind,omitempty"`
	Metadata   *Metadata  `json:"metadata,omitempty"`
	Spec       *Spec      `json:"spec,omitempty"`
}

type Metadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Timestamp   string            `json:"timestamp,omitempty"`
}

type Spec struct {
	Action    string                 `json:"action"`
	Target    Target                 `json:"target"`
	Replicas  int                    `json:"replicas,omitempty"`
	Resources map[string]interface{} `json:"resources,omitempty"`
	Selector  map[string]string      `json:"selector,omitempty"`
}

type Target struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// IntentFactory provides methods to create test intent data
type IntentFactory struct {
	BaseDir string
	Counter int
	rand    *rand.Rand
}

// NewIntentFactory creates a new intent factory
func NewIntentFactory(baseDir string) *IntentFactory {
	source := rand.NewSource(time.Now().UnixNano())
	return &IntentFactory{
		BaseDir: baseDir,
		rand:    rand.New(source),
	}
}

// CreateValidIntent creates a valid network intent with realistic data
// Uses string-based target format as expected by validateIntentFields
func (f *IntentFactory) CreateValidIntent(name string) IntentData {
	f.Counter++
	
	intentTypes := []string{"scaling", "deployment", "service"}
	namespaces := []string{"default", "production", "staging", "development"}
	resourceTypes := []string{"deployment", "statefulset", "daemonset"}
	
	ns := namespaces[f.rand.Intn(len(namespaces))]
	resourceType := resourceTypes[f.rand.Intn(len(resourceTypes))]
	
	return IntentData{
		IntentType: intentTypes[f.rand.Intn(len(intentTypes))],
		Target:     fmt.Sprintf("%s/%s", resourceType, name),
		Namespace:  ns,
		Replicas:   f.rand.Intn(10) + 1,
		Source:     "test-factory",
		Resources: map[string]interface{}{
			"cpu":    fmt.Sprintf("%dm", f.rand.Intn(1000)+100),
			"memory": fmt.Sprintf("%dMi", f.rand.Intn(1024)+128),
		},
		Labels: map[string]string{
			"app":         name,
			"version":     fmt.Sprintf("v1.%d", f.rand.Intn(10)),
			"environment": "test",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// CreateMinimalValidIntent creates an intent with only required fields
func (f *IntentFactory) CreateMinimalValidIntent(name string) IntentData {
	f.Counter++
	return IntentData{
		IntentType: "scaling",
		Target:     fmt.Sprintf("deployment/%s", name),
		Namespace:  "default",
	}
}

// CreateIntentWithTarget creates an intent with a specific target configuration
func (f *IntentFactory) CreateIntentWithTarget(name string, targetType, targetName, targetNamespace string) IntentData {
	f.Counter++
	intent := f.CreateValidIntent(name)
	
	intent.Target = fmt.Sprintf("%s/%s", targetType, targetName)
	intent.Namespace = targetNamespace
	
	return intent
}

// CreateValidIntentWithReplicas creates a valid intent with specific replica count
func (f *IntentFactory) CreateValidIntentWithReplicas(name string, replicas int) IntentData {
	f.Counter++
	intent := f.CreateValidIntent(name)
	intent.Replicas = replicas
	return intent
}

// CreateValidScaleUpIntent creates a valid scale-up intent
func (f *IntentFactory) CreateValidScaleUpIntent(name string) IntentData {
	return f.CreateIntentWithTarget(name, "deployment", fmt.Sprintf("%s-app", name), "production")
}

// CreateValidScaleDownIntent creates a valid scale-down intent
func (f *IntentFactory) CreateValidScaleDownIntent(name string) IntentData {
	intent := f.CreateIntentWithTarget(name, "deployment", fmt.Sprintf("%s-app", name), "production")
	intent.IntentType = "scaling"
	intent.Replicas = 1 // Scale down to minimum
	return intent
}

// CreateValidStatefulSetIntent creates a valid intent targeting a StatefulSet
func (f *IntentFactory) CreateValidStatefulSetIntent(name string) IntentData {
	return f.CreateIntentWithTarget(name, "statefulset", fmt.Sprintf("%s-db", name), "database")
}

// CreateValidDaemonSetIntent creates a valid intent targeting a DaemonSet  
func (f *IntentFactory) CreateValidDaemonSetIntent(name string) IntentData {
	return f.CreateIntentWithTarget(name, "daemonset", fmt.Sprintf("%s-agent", name), "monitoring")
}

// CreateIntentWithCustomFields creates an intent with custom fields
func (f *IntentFactory) CreateIntentWithCustomFields(name string, customFields map[string]interface{}) IntentData {
	f.Counter++
	intent := f.CreateValidIntent(name)
	
	// Merge custom fields into intent
	intentMap := make(map[string]interface{})
	intentBytes, _ := json.Marshal(intent)
	json.Unmarshal(intentBytes, &intentMap)
	
	for key, value := range customFields {
		intentMap[key] = value
	}
	
	// Convert back to IntentData (with potential custom fields)
	intentBytes, _ = json.Marshal(intentMap)
	json.Unmarshal(intentBytes, &intent)
	
	return intent
}

// CreateInvalidObjectTargetIntent creates intents with object-based targets (should be rejected)
func (f *IntentFactory) CreateInvalidObjectTargetIntent(name string) []byte {
	f.Counter++
	invalid := map[string]interface{}{
		"intent_type": "scaling",
		"target": map[string]interface{}{
			"type": "deployment",
			"name": name,
		},
		"namespace": "default",
		"replicas":  3,
	}
	
	bytes, _ := json.Marshal(invalid)
	return bytes
}

// CreateKubernetesStyleIntent creates intents in Kubernetes CRD style (should be handled differently)
func (f *IntentFactory) CreateKubernetesStyleIntent(name string) IntentData {
	f.Counter++
	apiVersion := "v1"
	kind := "NetworkIntent"
	
	return IntentData{
		APIVersion: &apiVersion,
		Kind:       &kind,
		Metadata: &Metadata{
			Name:      fmt.Sprintf("%s-%d", name, f.Counter),
			Namespace: "default",
			Labels: map[string]string{
				"app": name,
			},
			Timestamp: time.Now().Format(time.RFC3339),
		},
		Spec: &Spec{
			Action: "scale",
			Target: Target{
				Type: "deployment",
				Name: name,
			},
			Replicas: 3,
		},
	}
}

// CreateMalformedIntent creates various types of malformed JSON for testing
func (f *IntentFactory) CreateMalformedIntent(malformationType string) []byte {
	switch malformationType {
	case "missing_comma":
		return []byte(`{"intent_type": "scaling", "target": "deployment/test" "namespace": "default"}`)
	case "missing_closing_brace":
		return []byte(`{"intent_type": "scaling", "target": "deployment/test", "namespace": "default"`)
	case "invalid_json_syntax":
		return []byte(`{intent_type: "scaling", target: "deployment/test", namespace: "default"}`)
	case "trailing_comma":
		return []byte(`{"intent_type": "scaling", "target": "deployment/test", "namespace": "default",}`)
	case "duplicate_keys":
		return []byte(`{"intent_type": "scaling", "intent_type": "deployment", "target": "deployment/test"}`)
	case "incomplete_string":
		return []byte(`{"intent_type": "scaling", "target": "deployment/test", "namespace": "default`)
	case "invalid_escape":
		return []byte(`{"intent_type": "scaling", "target": "deployment/test", "data": "invalid\xescape"}`)
	default:
		// Default malformed - missing closing quote
		return []byte(`{"intent_type": "scaling, "target": "deployment/test", "namespace": "default"}`)
	}
}

// CreateOversizedIntent creates an intent that exceeds size limits
func (f *IntentFactory) CreateOversizedIntent(targetSize int) []byte {
	padding := strings.Repeat("x", targetSize-200) // Leave room for JSON structure
	oversized := map[string]interface{}{
		"intent_type": "scaling",
		"target":      "deployment/test",
		"namespace":   "default",
		"replicas":    3,
		"data":        padding,
	}
	
	bytes, _ := json.Marshal(oversized)
	return bytes
}

// CreateSuspiciousIntent creates intents with suspicious patterns for security testing
func (f *IntentFactory) CreateSuspiciousIntent(suspiciousType string) []byte {
	switch suspiciousType {
	case "path_traversal":
		return []byte(`{"intent_type": "scaling", "target": "../../../etc/passwd", "namespace": "default"}`)
	case "script_injection":
		return []byte(`{"intent_type": "scaling", "target": "deployment/test", "namespace": "<script>alert('xss')</script>", "command": "<script>alert('xss')</script>"}`)
	case "sql_injection":
		return []byte(`{"intent_type": "scaling", "target": "deployment/test", "namespace": "'; DROP TABLE users; --", "query": "'; DROP TABLE users; --"}`)
	case "command_injection":
		return []byte(`{"intent_type": "scaling", "target": "deployment/test", "namespace": "default", "cmd": "ls && rm -rf /"}`)
	case "null_bytes":
		return []byte("{\x00\"intent_type\": \"scaling\", \"target\": \"deployment/test\", \"namespace\": \"default\"}")
	default:
		return []byte(`{"intent_type": "scaling", "target": "deployment/test", "namespace": "default", "suspicious": "payload"}`)
	}
}

// CreateDeepNestedIntent creates an intent with deeply nested structures (JSON bomb)
func (f *IntentFactory) CreateDeepNestedIntent(depth int) []byte {
	nested := "\"end\""
	for i := 0; i < depth; i++ {
		nested = fmt.Sprintf(`{"level%d": %s}`, i, nested)
	}
	
	content := fmt.Sprintf(`{"intent_type": "scaling", "target": "deployment/test", "namespace": "default", "nested": %s}`, nested)
	return []byte(content)
}

// CreateIntentFile creates an intent file on disk
func (f *IntentFactory) CreateIntentFile(filename string, intent IntentData) (string, error) {
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return "", err
	}
	
	filePath := filepath.Join(f.BaseDir, filename)
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", err
	}
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}
	
	return filePath, nil
}

// CreateIntentFileWithContent creates a file with raw content (for malformed intents)
func (f *IntentFactory) CreateIntentFileWithContent(filename string, content []byte) (string, error) {
	filePath := filepath.Join(f.BaseDir, filename)
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", err
	}
	
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return "", err
	}
	
	return filePath, nil
}

// CreateBatchIntents creates multiple intent files for batch testing
func (f *IntentFactory) CreateBatchIntents(baseName string, count int) ([]string, error) {
	var filePaths []string
	
	for i := 0; i < count; i++ {
		intent := f.CreateValidIntent(fmt.Sprintf("%s-%d", baseName, i))
		filename := fmt.Sprintf("%s-%d.json", baseName, i)
		
		filePath, err := f.CreateIntentFile(filename, intent)
		if err != nil {
			return nil, fmt.Errorf("failed to create intent file %s: %w", filename, err)
		}
		
		filePaths = append(filePaths, filePath)
	}
	
	return filePaths, nil
}

// CreateConcurrentTestFiles creates files designed for concurrent processing tests
func (f *IntentFactory) CreateConcurrentTestFiles(prefix string, count int, staggerDelayMs int) ([]string, error) {
	var filePaths []string
	
	for i := 0; i < count; i++ {
		intent := f.CreateValidIntent(fmt.Sprintf("%s-concurrent-%d", prefix, i))
		
		// Add unique identifiers for tracking concurrent processing
		if intent.Labels == nil {
			intent.Labels = make(map[string]string)
		}
		intent.Labels["concurrent-test-id"] = fmt.Sprintf("%d", i)
		intent.Labels["created-at"] = time.Now().Format(time.RFC3339Nano)
		
		filename := fmt.Sprintf("%s-concurrent-%d.json", prefix, i)
		filePath, err := f.CreateIntentFile(filename, intent)
		if err != nil {
			return nil, fmt.Errorf("failed to create concurrent test file %s: %w", filename, err)
		}
		
		filePaths = append(filePaths, filePath)
		
		// Add stagger delay if specified
		if staggerDelayMs > 0 {
			time.Sleep(time.Duration(staggerDelayMs) * time.Millisecond)
		}
	}
	
	return filePaths, nil
}

// CreatePerformanceTestData creates data specifically for performance testing
func (f *IntentFactory) CreatePerformanceTestData(scenario string, count int) ([]string, error) {
	var filePaths []string
	
	switch scenario {
	case "small_files":
		// Small files for high-throughput testing
		for i := 0; i < count; i++ {
			intent := f.CreateMinimalValidIntent(fmt.Sprintf("perf-small-%d", i))
			filename := fmt.Sprintf("perf-small-%d.json", i)
			filePath, err := f.CreateIntentFile(filename, intent)
			if err != nil {
				return nil, err
			}
			filePaths = append(filePaths, filePath)
		}
		
	case "large_files":
		// Large files for memory/processing testing
		for i := 0; i < count; i++ {
			intent := f.CreateIntentWithCustomFields(fmt.Sprintf("perf-large-%d", i), map[string]interface{}{
				"large_data": strings.Repeat("data", 1000),
				"metadata":   strings.Repeat("meta", 500),
			})
			filename := fmt.Sprintf("perf-large-%d.json", i)
			filePath, err := f.CreateIntentFile(filename, intent)
			if err != nil {
				return nil, err
			}
			filePaths = append(filePaths, filePath)
		}
		
	case "mixed_sizes":
		// Mix of small and large files
		for i := 0; i < count; i++ {
			var intent IntentData
			if i%3 == 0 {
				// Large file
				intent = f.CreateIntentWithCustomFields(fmt.Sprintf("perf-mixed-%d", i), map[string]interface{}{
					"large_payload": strings.Repeat("x", 5000),
				})
			} else {
				// Small file
				intent = f.CreateMinimalValidIntent(fmt.Sprintf("perf-mixed-%d", i))
			}
			
			filename := fmt.Sprintf("perf-mixed-%d.json", i)
			filePath, err := f.CreateIntentFile(filename, intent)
			if err != nil {
				return nil, err
			}
			filePaths = append(filePaths, filePath)
		}
	}
	
	return filePaths, nil
}

// CleanupFiles removes all files created by this factory
func (f *IntentFactory) CleanupFiles() error {
	if f.BaseDir == "" || f.BaseDir == "/" {
		return fmt.Errorf("refusing to cleanup root or empty directory")
	}
	
	return os.RemoveAll(f.BaseDir)
}

// GetCreatedFileCount returns the number of files created by this factory
func (f *IntentFactory) GetCreatedFileCount() int {
	return f.Counter
}

// Reset resets the factory counter and random seed
func (f *IntentFactory) Reset() {
	f.Counter = 0
	source := rand.NewSource(time.Now().UnixNano())
	f.rand = rand.New(source)
}