//go:build stubs || test

package auth

import (
	"net/http"
)

// NewAuthHandlersStub creates new auth handlers (stub implementation)
func NewAuthHandlersStub(manager interface{}, logger interface{}) AuthHandlersInterface {
	return &StubAuthHandlers{}
}

// StubAuthHandlers provides stub implementations for auth handlers
type StubAuthHandlers struct{}

func (h *StubAuthHandlers) RegisterRoutes(router interface{}) {
	// Stub implementation
}

func (h *StubAuthHandlers) GetProvidersHandler(w http.ResponseWriter, r *http.Request) {
	// Stub implementation
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "stub implementation"}`))
}

func (h *StubAuthHandlers) InitiateLoginHandler(w http.ResponseWriter, r *http.Request) {
	// Stub implementation
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "stub implementation"}`))
}

func (h *StubAuthHandlers) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Stub implementation
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "stub implementation"}`))
}

func (h *StubAuthHandlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Stub implementation
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "stub implementation"}`))
}

func (h *StubAuthHandlers) GetUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	// Stub implementation
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "stub implementation"}`))
}
