package testdata

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// IntentData represents the structure of a network intent
type IntentData struct {
	APIVersion string    `json:"apiVersion"`
	Kind       string    `json:"kind"`
	Metadata   Metadata  `json:"metadata"`
	Spec       Spec      `json:"spec"`
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
func (f *IntentFactory) CreateValidIntent(name string) IntentData {
	f.Counter++
	
	actions := []string{"scale", "deploy", "update", "delete"}
	namespaces := []string{"default", "production", "staging", "development"}
	
	return IntentData{
		APIVersion: "v1",
		Kind:       "NetworkIntent",
		Metadata: Metadata{
			Name:      fmt.Sprintf("%s-%d", name, f.Counter),
			Namespace: namespaces[f.rand.Intn(len(namespaces))],
			Labels: map[string]string{
				"app":         name,
				"version":     fmt.Sprintf("v1.%d", f.rand.Intn(10)),
				"environment": "test",
			},
			Annotations: map[string]string{
				"conductor-loop/created-by": "test-factory",
				"conductor-loop/test-id":    uuid.New().String(),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		},
		Spec: Spec{
			Action: actions[f.rand.Intn(len(actions))],
			Target: Target{
				Type:      "deployment",
				Name:      name,
				Namespace: namespaces[f.rand.Intn(len(namespaces))],
			},
			Replicas: f.rand.Intn(10) + 1,
			Resources: map[string]interface{}{
				"cpu":    fmt.Sprintf("%dm", f.rand.Intn(1000)+100),
				"memory": fmt.Sprintf("%dMi", f.rand.Intn(1024)+128),
			},
			Selector: map[string]string{
				"app": name,
			},
		},
	}
}

// CreateMinimalValidIntent creates an intent with only required fields
func (f *IntentFactory) CreateMinimalValidIntent(name string) IntentData {
	f.Counter++
	return IntentData{
		APIVersion: "v1",
		Kind:       "NetworkIntent",
		Metadata: Metadata{
			Name: fmt.Sprintf("%s-minimal-%d", name, f.Counter),
		},
		Spec: Spec{
			Action: "scale",
			Target: Target{
				Type: "deployment",
				Name: name,
			},
		},
	}
}

// CreateIntentWithTarget creates an intent with a specific target configuration
func (f *IntentFactory) CreateIntentWithTarget(name string, targetType, targetName, targetNamespace string) IntentData {
	f.Counter++
	intent := f.CreateValidIntent(name)
	
	intent.Spec.Target = Target{
		Type:      targetType,
		Name:      targetName,
		Namespace: targetNamespace,
	}
	
	return intent
}

// CreateValidIntentWithReplicas creates a valid intent with specific replica count
func (f *IntentFactory) CreateValidIntentWithReplicas(name string, replicas int) IntentData {
	f.Counter++
	intent := f.CreateValidIntent(name)
	intent.Spec.Replicas = replicas
	return intent
}

// CreateValidScaleUpIntent creates a valid scale-up intent
func (f *IntentFactory) CreateValidScaleUpIntent(name string) IntentData {
	return f.CreateIntentWithTarget(name, "deployment", fmt.Sprintf("%s-app", name), "production")
}

// CreateValidScaleDownIntent creates a valid scale-down intent
func (f *IntentFactory) CreateValidScaleDownIntent(name string) IntentData {
	intent := f.CreateIntentWithTarget(name, "deployment", fmt.Sprintf("%s-app", name), "production")
	intent.Spec.Action = "scale"
	intent.Spec.Replicas = 1 // Scale down to minimum
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

// CreateIntentWithCustomFields creates an intent with custom specification
func (f *IntentFactory) CreateIntentWithCustomFields(name string, customSpec map[string]interface{}) IntentData {
	f.Counter++
	intent := f.CreateValidIntent(name)
	
	// Merge custom fields into spec
	specMap := make(map[string]interface{})
	specBytes, _ := json.Marshal(intent.Spec)
	json.Unmarshal(specBytes, &specMap)
	
	for key, value := range customSpec {
		specMap[key] = value
	}
	
	// Convert back to Spec (with potential custom fields)
	specBytes, _ = json.Marshal(specMap)
	json.Unmarshal(specBytes, &intent.Spec)
	
	return intent
}

// CreateMalformedIntent creates various types of malformed JSON for testing
func (f *IntentFactory) CreateMalformedIntent(malformationType string) []byte {
	switch malformationType {
	case "missing_comma":
		return []byte(`{"apiVersion": "v1", "kind": "NetworkIntent" "action": "scale"}`)
	case "missing_closing_brace":
		return []byte(`{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale"`)
	case "invalid_json_syntax":
		return []byte(`{apiVersion: "v1", kind: "NetworkIntent", action: "scale"}`)
	case "trailing_comma":
		return []byte(`{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale",}`)
	case "duplicate_keys":
		return []byte(`{"apiVersion": "v1", "apiVersion": "v2", "kind": "NetworkIntent"}`)
	case "incomplete_string":
		return []byte(`{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale`)
	case "invalid_escape":
		return []byte(`{"apiVersion": "v1", "kind": "NetworkIntent", "data": "invalid\xescape"}`)
	default:
		// Default malformed - missing closing quote
		return []byte(`{"apiVersion": "v1, "kind": "NetworkIntent", "action": "scale"}`)
	}
}

// CreateOversizedIntent creates an intent that exceeds size limits
func (f *IntentFactory) CreateOversizedIntent(targetSize int) []byte {
	padding := strings.Repeat("x", targetSize-200) // Leave room for JSON structure
	oversized := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name": "oversized-intent",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": "deployment/test",
			"data":   padding,
		},
	}
	
	bytes, _ := json.Marshal(oversized)
	return bytes
}

// CreateSuspiciousIntent creates intents with suspicious patterns for security testing
func (f *IntentFactory) CreateSuspiciousIntent(suspiciousType string) []byte {
	switch suspiciousType {
	case "path_traversal":
		return []byte(`{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"target": "../../../etc/passwd"}}`)
	case "script_injection":
		return []byte(`{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"command": "<script>alert('xss')</script>"}}`)
	case "sql_injection":
		return []byte(`{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"query": "'; DROP TABLE users; --"}}`)
	case "command_injection":
		return []byte(`{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"cmd": "ls && rm -rf /"}}`)
	case "null_bytes":
		return []byte("{\x00\"apiVersion\": \"v1\", \"kind\": \"NetworkIntent\"}")
	default:
		return []byte(`{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"suspicious": "payload"}}`)
	}
}

// CreateDeepNestedIntent creates an intent with deeply nested structures (JSON bomb)
func (f *IntentFactory) CreateDeepNestedIntent(depth int) []byte {
	nested := "\"end\""
	for i := 0; i < depth; i++ {
		nested = fmt.Sprintf(`{"level%d": %s}`, i, nested)
	}
	
	content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"nested": %s}}`, nested)
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
		intent.Metadata.Annotations["concurrent-test-id"] = fmt.Sprintf("%d", i)
		intent.Metadata.Annotations["created-at"] = time.Now().Format(time.RFC3339Nano)
		
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