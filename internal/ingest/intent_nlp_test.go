package ingest

import (
	"testing"
<<<<<<< HEAD
=======

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
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
<<<<<<< HEAD
			want: map[string]interface{}{},
=======
			want: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				"replicas":    4,
				"namespace":   "ran-a",
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			wantErr: false,
		},
		{
			name:  "Valid scaling intent without namespace",
			input: "scale my-app to 3",
<<<<<<< HEAD
			want: map[string]interface{}{},
=======
			want: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "my-app",
				"replicas":    3,
				"namespace":   "default",
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			wantErr: false,
		},
		{
			name:  "Valid deployment intent",
			input: "deploy nginx in ns production",
<<<<<<< HEAD
			want: map[string]interface{}{},
=======
			want: map[string]interface{}{
				"intent_type": "deployment",
				"target":      "nginx",
				"namespace":   "production",
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			wantErr: false,
		},
		{
			name:  "Valid delete intent",
			input: "delete old-app from ns staging",
<<<<<<< HEAD
			want: map[string]interface{}{},
=======
			want: map[string]interface{}{
				"intent_type": "deletion",
				"target":      "old-app",
				"namespace":   "staging",
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			wantErr: false,
		},
		{
			name:  "Valid update intent",
			input: "update myapp set replicas=5 in ns prod",
			want: map[string]interface{}{
<<<<<<< HEAD
				"replicas":  "5",
				"namespace": "prod",
=======
				"intent_type": "configuration",
				"target":      "myapp",
				"namespace":   "prod",
				"config": map[string]interface{}{
					"replicas": "5",
				},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
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
<<<<<<< HEAD
			want: map[string]interface{}{},
=======
			want: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "APP",
				"replicas":    2,
				"namespace":   "TEST",
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
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
<<<<<<< HEAD
			if !tt.wantErr && !compareIntents(got, tt.want) {
				t.Errorf("ParseIntent() = %v, want %v", got, tt.want)
=======
			if !tt.wantErr {
				require.NotNil(t, got, "ParseIntent() should not return nil for valid input")
				
				// Use robust assertions for required fields
				if tt.want != nil {
					assert.Equal(t, tt.want["intent_type"], got["intent_type"], "intent_type should match")
					assert.Equal(t, tt.want["target"], got["target"], "target should match")
					assert.Equal(t, tt.want["namespace"], got["namespace"], "namespace should match")
					
					// Check replicas if present in expected result
					if expectedReplicas, exists := tt.want["replicas"]; exists {
						assert.Equal(t, expectedReplicas, got["replicas"], "replicas should match")
					}
					
					// Check config if present in expected result
					if expectedConfig, exists := tt.want["config"]; exists {
						assert.Equal(t, expectedConfig, got["config"], "config should match")
					}
				}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
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
<<<<<<< HEAD
			intent: map[string]interface{}{},
=======
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-app",
				"namespace":   "default",
				"replicas":    1,
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			wantErr: false,
		},
		{
			name: "Missing intent_type",
<<<<<<< HEAD
			intent: map[string]interface{}{},
=======
			intent: map[string]interface{}{
				"target":    "test-app",
				"namespace": "default",
				"replicas":  1,
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			wantErr: true,
		},
		{
			name: "Missing target",
<<<<<<< HEAD
			intent: map[string]interface{}{},
=======
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"namespace":   "default",
				"replicas":    1,
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			wantErr: true,
		},
		{
			name: "Invalid replicas (negative)",
<<<<<<< HEAD
			intent: map[string]interface{}{},
=======
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-app",
				"namespace":   "default",
				"replicas":    -1,
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			wantErr: true,
		},
		{
			name: "Valid deployment intent",
<<<<<<< HEAD
			intent: map[string]interface{}{},
=======
			intent: map[string]interface{}{
				"intent_type": "deployment",
				"target":      "deploy-app",
				"namespace":   "default",
				"replicas":    2,
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			wantErr: false,
		},
		{
			name: "Valid configuration intent",
			intent: map[string]interface{}{
<<<<<<< HEAD
				"replicas":  "5",
				"namespace": "prod",
=======
				"intent_type": "configuration",
				"target":      "config-app",
				"namespace":   "prod",
				"config": map[string]interface{}{
					"replicas": "5",
				},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			},
			wantErr: false,
		},
		{
			name:    "Invalid configuration (empty config)",
			intent:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "Unknown intent_type",
			intent: map[string]interface{}{},
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

