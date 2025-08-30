// internal/ingest/validator.go
package ingest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type Intent struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Priority        int                    `json:"priority,omitempty"`
	Description     string                 `json:"description"`
	Parameters      map[string]interface{} `json:"parameters"`
	TargetResources []string               `json:"target_resources"`
	Constraints     map[string]interface{} `json:"constraints,omitempty"`
	CreatedAt       string                 `json:"created_at,omitempty"`
	UpdatedAt       string                 `json:"updated_at,omitempty"`
	Status          string                 `json:"status"`
}

type Validator struct {
	schema *jsonschema.Schema
}

func NewValidator(schemaPath string) (*Validator, error) {
	// 若給的是資料夾，補成標準位置
	if info, err := os.Stat(schemaPath); err == nil && info.IsDir() {
		schemaPath = filepath.Join(schemaPath, "docs", "contracts", "intent.schema.json")
	}
	if !filepath.IsAbs(schemaPath) {
		if cwd, err := os.Getwd(); err == nil {
			schemaPath = filepath.Join(cwd, schemaPath)
		}
	}

	// 讀入 schema -> 轉成 JSON 值（v6 的 AddResource 需要 JSON 值）
	b, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("open schema: %w", err)
	}
	var doc any
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse schema json: %w", err)
	}

	c := jsonschema.NewCompiler()

	// （可選）如果你的環境離線、避免去抓 metaschema，
	// 你可以事先把 2020-12 metaschema 下載到本地，然後：
	//   var meta any; _ = json.Unmarshal(metaBytes, &meta)
	//   _ = c.AddResource("https://json-schema.org/draft/2020-12/schema", meta)
	// 不加也行：編譯器會依 $schema 自動載入。

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

func (v *Validator) ValidateBytes(b []byte) (*Intent, error) {
	var tmp any
	if err := json.Unmarshal(b, &tmp); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}
	if err := v.schema.Validate(tmp); err != nil {
		return nil, err
	}
	var in Intent
	if err := json.Unmarshal(b, &in); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &in, nil
}
