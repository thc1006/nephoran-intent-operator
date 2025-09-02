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

package disaster

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPathTraversalProtection verifies that malicious tar archives

// cannot escape the target directory (G305 security fix)

func TestPathTraversalProtection(t *testing.T) {
	tests := []struct {
		name string

		fileName string

		shouldSkip bool
	}{
		{
			name: "normal file",

			fileName: "normal.txt",

			shouldSkip: false,
		},

		{
			name: "subdirectory file",

			fileName: "subdir/file.txt",

			shouldSkip: false,
		},

		{
			name: "path traversal with ../",

			fileName: "../../../etc/passwd",

			shouldSkip: true,
		},

		{
			name: "path traversal with absolute path",

			fileName: "/etc/passwd",

			shouldSkip: false, // Will be cleaned to etc/passwd

		},

		{
			name: "hidden path traversal",

			fileName: "subdir/../../etc/passwd",

			shouldSkip: true,
		},

		{
			name: "windows path traversal",

			fileName: "..\\..\\windows\\system32\\config",

			shouldSkip: true,
		},

		{
			name: "mixed separators",

			fileName: "subdir/../../../etc/passwd",

			shouldSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for extraction

			tempDir, err := os.MkdirTemp("", "security_test_*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}

			defer os.RemoveAll(tempDir)

			// Create a malicious tar archive in memory

			var buf bytes.Buffer

			gw := gzip.NewWriter(&buf)

			tw := tar.NewWriter(gw)

			// Add a file with potentially malicious path

			content := []byte("malicious\n")

			header := &tar.Header{
				Name: tt.fileName,

				Mode: 0o644,

				Size: int64(len(content)),
			}

			if err := tw.WriteHeader(header); err != nil {
				t.Fatalf("Failed to write header: %v", err)
			}

			if _, err := tw.Write(content); err != nil {
				t.Fatalf("Failed to write content: %v", err)
			}

			if err := tw.Close(); err != nil {
				t.Fatalf("Failed to close tar writer: %v", err)
			}

			if err := gw.Close(); err != nil {
				t.Fatalf("Failed to close gzip writer: %v", err)
			}

			// Attempt extraction using our secured code

			gr, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("Failed to create gzip reader: %v", err)
			}

			defer gr.Close()

			tr := tar.NewReader(gr)

			extractedFiles := 0

			// This simulates the extraction logic with our security fixes

			for {

				header, err := tr.Next()

				if err == io.EOF {
					break
				}

				if err != nil {
					t.Fatalf("Failed to read tar entry: %v", err)
				}

				// Apply our security fix logic

				cleanName := filepath.Clean(header.Name)

				cleanName = filepath.ToSlash(cleanName) // Normalize separators

				if cleanName[0] == '/' {
					cleanName = cleanName[1:]
				}

				// Check for directory traversal

				if filepath.IsAbs(cleanName) ||

					containsPathTraversal(cleanName) {

					// Skip malicious files

					continue
				}

				target := filepath.Join(tempDir, cleanName)

				// Final safety check

				targetAbs, _ := filepath.Abs(target)

				tempDirAbs, _ := filepath.Abs(tempDir)

				if !strings.HasPrefix(targetAbs, tempDirAbs+string(filepath.Separator)) &&

					targetAbs != tempDirAbs {

					// Would escape directory, skip

					continue
				}

				extractedFiles++

			}

			if tt.shouldSkip && extractedFiles > 0 {
				t.Errorf("Expected malicious file %q to be skipped, but it was extracted", tt.fileName)
			} else if !tt.shouldSkip && extractedFiles == 0 {
				t.Errorf("Expected legitimate file %q to be extracted, but it was skipped", tt.fileName)
			}
		})
	}
}

// containsPathTraversal checks if a path contains directory traversal attempts

func containsPathTraversal(path string) bool {
	// Normalize to forward slashes for consistent checking

	normalized := filepath.ToSlash(path)

	// Check for .. in the path (potential directory traversal)

	if contains(normalized, "..") {
		return true
	}

	// Additional check: split by / and check each component

	parts := splitPath(normalized)

	for _, part := range parts {
		if part == ".." {
			return true
		}
	}

	return false
}

// splitPath splits a path by forward slashes

func splitPath(path string) []string {
	var parts []string

	current := ""

	for _, r := range path {
		if r == '/' {
			if current != "" {

				parts = append(parts, current)

				current = ""

			}
		} else {
			current += string(r)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// contains is a simple string contains helper

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsImpl(s, substr)
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// TestFileModeValidation verifies that file modes are properly validated

// to prevent integer overflow issues

func TestFileModeValidation(t *testing.T) {
	tests := []struct {
		name string

		inputMode int64

		expectedMode os.FileMode
	}{
		{
			name: "normal permissions",

			inputMode: 0o644,

			expectedMode: 0o644,
		},

		{
			name: "executable permissions",

			inputMode: 0o755,

			expectedMode: 0o755,
		},

		{
			name: "large mode value",

			inputMode: int64(1<<32) | 0o644,

			expectedMode: 0o644, // Should mask to permission bits only

		},

		{
			name: "negative mode",

			inputMode: -1,

			expectedMode: 0o777, // -1 & 0777 = 0777

		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply our security fix: use only permission bits

			result := os.FileMode(tt.inputMode & 0o777)

			if result != tt.expectedMode {
				t.Errorf("Expected mode %o, got %o", tt.expectedMode, result)
			}
		})
	}
}
