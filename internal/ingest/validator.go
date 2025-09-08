// internal/ingest/validator.go.

package ingest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// Intent represents a network scaling intent with target deployment and replica configuration.

type Intent struct {
	IntentType      string    `json:"intent_type"`
	Target          string    `json:"target"`
	Namespace       string    `json:"namespace"`
	Replicas        int       `json:"replicas"`
	Reason          string    `json:"reason,omitempty"`
	Source          string    `json:"source,omitempty"`
	CorrelationID   string    `json:"correlation_id,omitempty"`
	Priority        int       `json:"priority,omitempty"`
	CreatedAt       string    `json:"created_at,omitempty"`
	UpdatedAt       string    `json:"updated_at,omitempty"`
	Status          string    `json:"status,omitempty"`
	TargetResources []string  `json:"target_resources,omitempty"`
	Constraints     *Constraints `json:"constraints,omitempty"`
	NephioContext   *NephioContext `json:"nephio_context,omitempty"`
}

// Constraints represents optional constraints for intent execution
type Constraints struct {
	MaxReplicas        int      `json:"max_replicas,omitempty"`
	MinReplicas        int      `json:"min_replicas,omitempty"`
	AllowedNamespaces  []string `json:"allowed_namespaces,omitempty"`
	KptPackageName     string   `json:"kpt_package_name,omitempty"`
	PorchRepository    string   `json:"porch_repository,omitempty"`
}

// NephioContext represents Nephio/Porch integration context
type NephioContext struct {
	PackageRevision  string `json:"package_revision,omitempty"`
	WorkloadCluster  string `json:"workload_cluster,omitempty"`
	BlueprintName    string `json:"blueprint_name,omitempty"`
}

// Validator represents a validator.

type Validator struct {
	schema *jsonschema.Schema
}

// NewValidator performs newvalidator operation.

func NewValidator(schemaPath string) (*Validator, error) {
	// 若給的是資料夾，補成標準位置.

	if info, err := os.Stat(schemaPath); err == nil && info.IsDir() {
		schemaPath = filepath.Join(schemaPath, "docs", "contracts", "intent.schema.json")
	}

	if !filepath.IsAbs(schemaPath) {
		if cwd, err := os.Getwd(); err == nil {
			schemaPath = filepath.Join(cwd, schemaPath)
		}
	}

	// 讀入 schema -> 轉成 JSON 值（v6 的 AddResource 需要 JSON 值）.

	b, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("open schema: %w", err)
	}

	var doc any

	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse schema json: %w", err)
	}

	c := jsonschema.NewCompiler()

	// （可選）如果你的環境離線、避免去抓 metaschema，.

	// 你可以事先把 2020-12 metaschema 下載到本地，然後：.

	//   var meta any; _ = json.Unmarshal(metaBytes, &meta).

	//   _ = c.AddResource("https://json-schema.org/draft/2020-12/schema", meta).

	// 不加也行：編譯器會依 $schema 自動載入。.

	const resName = "intent.schema.json"

	if err := c.AddResource(resName, doc); err != nil {
		return nil, fmt.Errorf("add resource: %w", err)
	}

	s, err := c.Compile(resName)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}

	return &Validator{schema: s}, nil
}

// ValidateBytes performs validatebytes operation.

func (v *Validator) ValidateBytes(b []byte) (*Intent, error) {
	var tmp map[string]interface{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}

	// Validate the raw payload first - no modifications
	if err := v.schema.Validate(tmp); err != nil {
		return nil, fmt.Errorf("jsonschema validation failed: %w", err)
	}

	// After validation passes, unmarshal into the Intent struct
	var in Intent
	if err := json.Unmarshal(b, &in); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &in, nil
}
