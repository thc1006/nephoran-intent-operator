//go:build stubs || test

package auth

import (
	"net/http"
)

// NewAuthHandlersStub creates new auth handlers (stub implementation).
func NewAuthHandlersStub(manager, logger interface{}) AuthHandlersInterface {
	return &StubAuthHandlers{}
}

// StubAuthHandlers provides stub implementations for auth handlers.
type StubAuthHandlers struct{}

// RegisterRoutes performs registerroutes operation.
func (h *StubAuthHandlers) RegisterRoutes(router interface{}) {
	// Stub implementation.
}

// GetProvidersHandler performs getprovidershandler operation.
func (h *StubAuthHandlers) GetProvidersHandler(w http.ResponseWriter, r *http.Request) {
	// Stub implementation.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "stub implementation"}`))
}

// InitiateLoginHandler performs initiateloginhandler operation.
func (h *StubAuthHandlers) InitiateLoginHandler(w http.ResponseWriter, r *http.Request) {
	// Stub implementation.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "stub implementation"}`))
}

// CallbackHandler performs callbackhandler operation.
func (h *StubAuthHandlers) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Stub implementation.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "stub implementation"}`))
}

// LogoutHandler performs logouthandler operation.
func (h *StubAuthHandlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Stub implementation.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "stub implementation"}`))
}

// GetUserInfoHandler performs getuserinfohandler operation.
func (h *StubAuthHandlers) GetUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	// Stub implementation.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "stub implementation"}`))
}
