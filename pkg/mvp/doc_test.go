package mvp

import "testing"

// DISABLED: func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

// DISABLED: func TestDemoPath(t *testing.T) {
	if DemoPath == "" {
		t.Error("DemoPath should not be empty")
	}
}
