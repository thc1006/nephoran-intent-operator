package ingest

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type Intent struct {
	IntentType    string  `json:"intent_type"`
	Target        string  `json:"target"`
	Namespace     string  `json:"namespace"`
	Replicas      int     `json:"replicas"`
	Reason        string  `json:"reason,omitempty"`
	Source        string  `json:"source,omitempty"`
	CorrelationID string  `json:"correlation_id,omitempty"`
}

type Validator struct {
	schema *jsonschema.Schema
}

func NewValidator(schemaPath string) (*Validator, error) {
	c := jsonschema.NewCompiler()
	f, err := os.Open(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("open schema: %w", err)
	}
	defer f.Close()
	if err := c.AddResource("intent.schema.json", f); err != nil {
		return nil, fmt.Errorf("add schema: %w", err)
	}
	// 直接引用本地資源名稱（不再遠端抓）
	s, err := c.Compile("intent.schema.json")
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
