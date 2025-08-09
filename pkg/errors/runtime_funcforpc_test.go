package errors

import (
	"runtime"
	"testing"
)

// getValidTestFunc returns a valid runtime.Func for testing
func getValidTestFunc() *runtime.Func {
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		return nil
	}
	return runtime.FuncForPC(pc)
}

// TestGetSafeFunctionName tests the safe function name extraction
func TestGetSafeFunctionName(t *testing.T) {
	tests := []struct {
		name     string
		fn       *runtime.Func
		expected string
	}{
		{
			name:     "nil function",
			fn:       nil,
			expected: "",
		},
		{
			name:     "valid function",
			fn:       getValidTestFunc(),
			expected: "", // This will get the actual function name, not empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSafeFunctionName(tt.fn)

			// For the nil case, we expect empty string
			if tt.name == "nil function" && result != tt.expected {
				t.Errorf("getSafeFunctionName() = %v, want %v", result, tt.expected)
			}

			// For the valid function case, we expect some function name (not empty)
			if tt.name == "valid function" && result == "" {
				t.Errorf("getSafeFunctionName() should return non-empty string for valid function, got empty")
			}
		})
	}
}

// TestGetSafeFunctionNamePanicRecovery tests panic recovery in getSafeFunctionName
func TestGetSafeFunctionNamePanicRecovery(t *testing.T) {
	// Create an invalid runtime.Func by using an invalid PC
	// This is a bit tricky to test since runtime.FuncForPC usually returns nil for invalid PCs
	// But we can still test our nil check and recovery mechanism

	// Test with nil
	result := getSafeFunctionName(nil)
	if result != "" {
		t.Errorf("getSafeFunctionName(nil) = %v, want empty string", result)
	}

	// Test that the function doesn't panic even with edge cases
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("getSafeFunctionName should not panic, but it did: %v", r)
		}
	}()

	// Get a valid PC and function
	pc, _, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("Could not get caller info for test")
	}

	fn := runtime.FuncForPC(pc)
	result = getSafeFunctionName(fn)

	// Result should not be empty for a valid function
	if fn != nil && result == "" {
		t.Errorf("getSafeFunctionName should return non-empty result for valid function")
	}
}

// TestGetCallerInfoSafety tests that GetCallerInfo handles edge cases safely
func TestGetCallerInfoSafety(t *testing.T) {
	// Test normal case
	file, line, function, ok := GetCallerInfo(0)
	if !ok {
		t.Error("GetCallerInfo should succeed for skip=0")
	}

	if file == "" || line == 0 || function == "" {
		t.Errorf("GetCallerInfo should return valid values, got file=%s, line=%d, function=%s", file, line, function)
	}

	// Test with high skip value (should fail gracefully)
	file, line, function, ok = GetCallerInfo(1000)
	if ok {
		t.Error("GetCallerInfo should fail for very high skip values")
	}

	if file != "" || line != 0 || function != "" {
		t.Errorf("GetCallerInfo should return empty values when failed, got file=%s, line=%d, function=%s", file, line, function)
	}
}

// TestIsInPackageSafety tests that IsInPackage handles edge cases safely
func TestIsInPackageSafety(t *testing.T) {
	// Test normal case - should find testing package
	found := IsInPackage("testing", 10)
	if !found {
		t.Error("IsInPackage should find 'testing' package in current stack")
	}

	// Test with non-existent package
	found = IsInPackage("nonexistent-package-name", 10)
	if found {
		t.Error("IsInPackage should not find non-existent package")
	}

	// Test with zero maxDepth
	found = IsInPackage("testing", 0)
	if found {
		t.Error("IsInPackage should not find any package with maxDepth=0")
	}
}

// TestStackTraceSafety tests that stack trace functions handle edge cases safely
func TestStackTraceSafety(t *testing.T) {
	// Test normal case
	stack := getStackTrace(1)
	if len(stack) == 0 {
		t.Error("getStackTrace should return non-empty stack trace")
	}

	// Ensure all stack entries are properly formatted and don't contain panic indicators
	for i, entry := range stack {
		if entry == "" {
			t.Errorf("Stack entry %d should not be empty", i)
		}

		// Should contain file:line format
		if len(entry) < 3 || !containsColon(entry) {
			t.Errorf("Stack entry %d should be properly formatted: %s", i, entry)
		}
	}
}

// Helper function to check if string contains colon
func containsColon(s string) bool {
	for _, c := range s {
		if c == ':' {
			return true
		}
	}
	return false
}
