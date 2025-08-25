package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
)

// AuthenticatedLLMProcessorHandler wraps with authentication
type AuthenticatedLLMProcessorHandler struct {
	*LLMProcessorHandler
	authIntegration *auth.NephoranAuthIntegration
	config          *config.LLMProcessorConfig
	logger          *slog.Logger
}

// NewAuthenticatedLLMProcessorHandler creates authenticated handler
func NewAuthenticatedLLMProcessorHandler(
	originalHandler *LLMProcessorHandler,
	authIntegration *auth.NephoranAuthIntegration,
	config *config.LLMProcessorConfig,
	logger *slog.Logger,
) *AuthenticatedLLMProcessorHandler {
	return &AuthenticatedLLMProcessorHandler{
		LLMProcessorHandler: originalHandler,
		authIntegration:     authIntegration,
		config:              config,
		logger:              logger,
	}
}

// ProcessIntentHandler wraps original with authentication
func (ah *AuthenticatedLLMProcessorHandler) ProcessIntentHandler(w http.ResponseWriter, r *http.Request) {
	if ah.config.AuthEnabled && ah.config.RequireAuth {
		authContext := auth.GetAuthContext(r.Context())
		if authContext == nil {
			ah.writeAuthError(w, "Authentication required")
			return
		}

		if !ah.hasProcessPermission(authContext) {
			ah.writeAuthError(w, "Insufficient permissions")
			return
		}

		ah.logAuthenticatedRequest(r, authContext, "process_intent")
	}

	ah.LLMProcessorHandler.ProcessIntentHandler(w, r)
}

// hasProcessPermission checks if user can process intents
func (ah *AuthenticatedLLMProcessorHandler) hasProcessPermission(authContext *auth.AuthContext) bool {
	for _, role := range authContext.Roles {
		if role == "nephoran-operator" || role == "nephoran-admin" {
			return true
		}
	}
	return false
}

// logAuthenticatedRequest logs authenticated requests
func (ah *AuthenticatedLLMProcessorHandler) logAuthenticatedRequest(r *http.Request, authContext *auth.AuthContext, operation string) {
	ah.logger.Info("Authenticated request",
		slog.String("operation", operation),
		slog.String("user_id", authContext.UserID),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path))
}

// writeAuthError writes authentication error
func (ah *AuthenticatedLLMProcessorHandler) writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	response := map[string]interface{}{
		"error":     "authentication_required",
		"message":   message,
		"timestamp": time.Now().Format(time.RFC3339),
		"status":    "error",
	}

	json.NewEncoder(w).Encode(response)
}
