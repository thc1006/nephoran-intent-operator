package loop

import (
	"testing"
)

func TestIsIntentFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "valid intent file",
			filename: "intent-20250812T145115Z.json",
			want:     true,
		},
		{
			name:     "valid intent file with simple name",
			filename: "intent-test.json",
			want:     true,
		},
		{
			name:     "valid intent file with complex name",
			filename: "intent-scale-up-ran-cells.json",
			want:     true,
		},
		{
			name:     "missing intent prefix",
			filename: "scale-20250812T145115Z.json",
			want:     false,
		},
		{
			name:     "missing json extension",
			filename: "intent-20250812T145115Z.txt",
			want:     false,
		},
		{
			name:     "missing both prefix and extension",
			filename: "config.yaml",
			want:     false,
		},
		{
			name:     "intent prefix but no extension",
			filename: "intent-test",
			want:     false,
		},
		{
			name:     "json extension but no intent prefix",
			filename: "config.json",
			want:     false,
		},
		{
			name:     "empty filename",
			filename: "",
			want:     false,
		},
		{
			name:     "just intent prefix",
			filename: "intent-",
			want:     false,
		},
		{
			name:     "just json extension",
			filename: ".json",
			want:     false,
		},
		{
			name:     "intent in middle of filename",
			filename: "test-intent-file.json",
			want:     false,
		},
		{
			name:     "case sensitive check",
			filename: "Intent-test.json",
			want:     false,
		},
		{
			name:     "windows path with valid filename",
			filename: "intent-windows.json",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsIntentFile(tt.filename); got != tt.want {
				t.Errorf("IsIntentFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}
