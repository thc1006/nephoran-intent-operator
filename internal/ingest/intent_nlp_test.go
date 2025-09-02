package ingest

import (
	"testing"
)

func TestRuleBasedIntentParser_ParseIntent(t *testing.T) {
	parser := NewRuleBasedIntentParser()

	tests := []struct {
		name    string
		input   string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:  "Valid scaling intent with namespace",
			input: "scale nf-sim to 4 in ns ran-a",
			want: json.RawMessage(`{}`),
			wantErr: false,
		},
		{
			name:  "Valid scaling intent without namespace",
			input: "scale my-app to 3",
			want: json.RawMessage(`{}`),
			wantErr: false,
		},
		{
			name:  "Valid deployment intent",
			input: "deploy nginx in ns production",
			want: json.RawMessage(`{}`),
			wantErr: false,
		},
		{
			name:  "Valid delete intent",
			input: "delete old-app from ns staging",
			want: json.RawMessage(`{}`),
			wantErr: false,
		},
		{
			name:  "Valid update intent",
			input: "update myapp set replicas=5 in ns prod",
			want: map[string]interface{}{
					"replicas": "5",
				},
				"namespace": "prod",
			},
			wantErr: false,
		},
		{
			name:    "Empty input",
			input:   "",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Invalid command",
			input:   "this is not a valid command",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Invalid replica count",
			input:   "scale app to abc",
			want:    nil,
			wantErr: true,
		},
		{
			name:  "Case insensitive command",
			input: "SCALE APP TO 2 IN NS TEST",
			want: json.RawMessage(`{}`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.ParseIntent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIntent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !compareIntents(got, tt.want) {
				t.Errorf("ParseIntent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateIntent(t *testing.T) {
	tests := []struct {
		name    string
		intent  map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid scaling intent",
			intent: json.RawMessage(`{}`),
			wantErr: false,
		},
		{
			name: "Missing intent_type",
			intent: json.RawMessage(`{}`),
			wantErr: true,
		},
		{
			name: "Missing target",
			intent: json.RawMessage(`{}`),
			wantErr: true,
		},
		{
			name: "Invalid replicas (negative)",
			intent: json.RawMessage(`{}`),
			wantErr: true,
		},
		{
			name: "Valid deployment intent",
			intent: json.RawMessage(`{}`),
			wantErr: false,
		},
		{
			name: "Valid configuration intent",
			intent: map[string]interface{}{
					"replicas": "5",
				},
				"namespace": "prod",
			},
			wantErr: false,
		},
		{
			name: "Invalid configuration (empty config)",
			intent: json.RawMessage(`{}`)"),
				"namespace":   "prod",
			},
			wantErr: true,
		},
		{
			name: "Unknown intent_type",
			intent: json.RawMessage(`{}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIntent(tt.intent)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIntent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// compareIntents compares two intent maps for equality
func compareIntents(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	for key, aVal := range a {
		bVal, ok := b[key]
		if !ok {
			return false
		}

		// Special handling for nested maps (like config)
		if aMap, ok := aVal.(map[string]interface{}); ok {
			if bMap, ok := bVal.(map[string]interface{}); ok {
				if !compareIntents(aMap, bMap) {
					return false
				}
			} else {
				return false
			}
		} else if aVal != bVal {
			return false
		}
	}

	return true
}

