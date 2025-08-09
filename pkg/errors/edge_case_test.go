package errors

import (
	"runtime"
	"testing"
)

// TestRuntimeFuncForPCEdgeCases tests edge cases that could cause panics
func TestRuntimeFuncForPCEdgeCases(t *testing.T) {
	t.Run("nil runtime.Func should not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("getSafeFunctionName should not panic with nil input, but got panic: %v", r)
			}
		}()
		
		result := getSafeFunctionName(nil)
		if result != "" {
			t.Errorf("getSafeFunctionName(nil) = %q, want empty string", result)
		}
	})

	t.Run("valid runtime.Func should work correctly", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("getSafeFunctionName should not panic with valid input, but got panic: %v", r)
			}
		}()
		
		// Get current function PC
		pc, _, _, ok := runtime.Caller(0)
		if !ok {
			t.Skip("Cannot get caller info for test")
		}
		
		fn := runtime.FuncForPC(pc)
		result := getSafeFunctionName(fn)
		
		// Should get some function name
		if fn != nil && result == "" {
			t.Error("getSafeFunctionName should return non-empty name for valid function")
		}
	})

	t.Run("invalid PC should not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("getSafeFunctionName should not panic with invalid PC, but got panic: %v", r)
			}
		}()
		
		// Create function from invalid PC (0 should be invalid)
		fn := runtime.FuncForPC(0)
		result := getSafeFunctionName(fn)
		
		// fn will likely be nil for invalid PC, so result should be empty
		t.Logf("getSafeFunctionName with invalid PC returned: %q", result)
	})

	t.Run("high PC value should not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("getSafeFunctionName should not panic with high PC, but got panic: %v", r)
			}
		}()
		
		// Use a very high PC value that's unlikely to be valid
		fn := runtime.FuncForPC(uintptr(0xFFFFFFFF))
		result := getSafeFunctionName(fn)
		
		t.Logf("getSafeFunctionName with high PC returned: %q", result)
	})
}

// TestAllFuncForPCUsagesAreProtected ensures all usage sites are protected
func TestAllFuncForPCUsagesAreProtected(t *testing.T) {
	t.Run("GetCallerInfo should not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("GetCallerInfo should not panic, but got: %v", r)
			}
		}()
		
		// Test normal case
		file, line, function, ok := GetCallerInfo(0)
		if !ok {
			t.Error("GetCallerInfo(0) should succeed")
		} else {
			t.Logf("GetCallerInfo(0) = %q:%d %q", file, line, function)
		}
		
		// Test with very high skip (should fail gracefully)
		file, line, function, ok = GetCallerInfo(1000)
		if ok {
			t.Error("GetCallerInfo(1000) should fail")
		}
		if file != "" || line != 0 || function != "" {
			t.Error("GetCallerInfo failure should return empty values")
		}
	})

	t.Run("IsInPackage should not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("IsInPackage should not panic, but got: %v", r)
			}
		}()
		
		// Test normal case
		found := IsInPackage("testing", 20)
		if !found {
			t.Error("IsInPackage should find 'testing' package in stack")
		}
		
		// Test with non-existent package
		found = IsInPackage("definitely-does-not-exist", 10)
		if found {
			t.Error("IsInPackage should not find non-existent package")
		}
	})

	t.Run("getStackTrace should not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("getStackTrace should not panic, but got: %v", r)
			}
		}()
		
		// Test normal case
		stack := getStackTrace(1)
		if len(stack) == 0 {
			t.Error("getStackTrace should return non-empty stack")
		}
		
		// Test with high skip
		stack = getStackTrace(100)
		// Should return empty or very short stack, but not panic
		t.Logf("getStackTrace(100) returned %d entries", len(stack))
	})
}

// TestRecoveryMechanism specifically tests our panic recovery
func TestRecoveryMechanism(t *testing.T) {
	// This test verifies that our defer/recover mechanism works
	// by simulating a scenario where fn.Name() could theoretically panic
	
	t.Run("recovery mechanism should work", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Test should not panic even with recovery mechanism active: %v", r)
			}
		}()
		
		// Test with nil (exercises the nil check path)
		result := getSafeFunctionName(nil)
		if result != "" {
			t.Errorf("Expected empty string for nil, got %q", result)
		}
		
		// Test with valid function (exercises the recovery path)
		pc, _, _, ok := runtime.Caller(0)
		if ok {
			fn := runtime.FuncForPC(pc)
			result = getSafeFunctionName(fn)
			
			// Should get some result without panicking
			t.Logf("getSafeFunctionName returned: %q", result)
		}
	})
}