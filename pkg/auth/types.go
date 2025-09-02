package auth

import (
	"net/http"
)

// HandlersConfig represents handlers configuration.
type HandlersConfig struct {
	BaseURL         string `json:"base_url"`
	DefaultRedirect string `json:"default_redirect"`
	LoginPath       string `json:"login_path"`
	CallbackPath    string `json:"callback_path"`
	LogoutPath      string `json:"logout_path"`
	UserInfoPath    string `json:"userinfo_path"`
	EnableAPITokens bool   `json:"enable_api_tokens"`
	TokenPath       string `json:"token_path"`
}

// AuthHandlersConfig is an alias for HandlersConfig to maintain compatibility
type AuthHandlersConfig = HandlersConfig

// Session is an alias for UserSession to maintain compatibility
type Session = UserSession

// AuthMiddlewareConfig represents authentication middleware configuration.
type AuthMiddlewareConfig struct {
	JWTManager     *JWTManager     `json:"-"`
	SessionManager *SessionManager `json:"-"`
	RequireAuth    bool            `json:"require_auth"`
	AllowedPaths   []string        `json:"allowed_paths"`
	HeaderName     string          `json:"header_name"`
	CookieName     string          `json:"cookie_name"`
	ContextKey     string          `json:"context_key"`
}

// UserContext represents user context information passed in requests.
type UserContext struct {
	UserID      string                 `json:"user_id"`
	SessionID   string                 `json:"session_id"`
	Provider    string                 `json:"provider"`
	Roles       []string               `json:"roles"`
	Permissions []string               `json:"permissions"`
	IsAdmin     bool                   `json:"is_admin"`
	Attributes  map[string]interface{} `json:"attributes"`
}

// RBACMiddlewareConfig represents RBAC middleware configuration.
type RBACMiddlewareConfig struct {
	RBACManager       *RBACManager                                 `json:"-"`
	ResourceExtractor func(r *http.Request) string                 `json:"-"`
	ActionExtractor   func(r *http.Request) string                 `json:"-"`
	UserIDExtractor   func(r *http.Request) string                 `json:"-"`
	OnAccessDenied    func(w http.ResponseWriter, r *http.Request) `json:"-"`
}

// CORSConfig represents CORS middleware configuration.
type CORSConfig struct {
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

// RateLimitConfig represents rate limiting middleware configuration.
type RateLimitConfig struct {
	RequestsPerMinute int                                          `json:"requests_per_minute"`
	BurstSize         int                                          `json:"burst_size"`
	KeyGenerator      func(r *http.Request) string                 `json:"-"`
	OnLimitExceeded   func(w http.ResponseWriter, r *http.Request) `json:"-"`
}

// SecurityHeadersConfig represents security headers middleware configuration.
type SecurityHeadersConfig struct {
	ContentSecurityPolicy string            `json:"content_security_policy"`
	XFrameOptions         string            `json:"x_frame_options"`
	XContentTypeOptions   string            `json:"x_content_type_options"`
	ReferrerPolicy        string            `json:"referrer_policy"`
	HSTSMaxAge            int64             `json:"hsts_max_age"`
	HSTSIncludeSubdomains bool              `json:"hsts_include_subdomains"`
	HSTSPreload           bool              `json:"hsts_preload"`
	RemoveServerHeader    bool              `json:"remove_server_header"`
	CustomHeaders         map[string]string `json:"custom_headers"`
}

// RequestLoggingConfig represents request logging middleware configuration.
type RequestLoggingConfig struct {
	Logger           func(entry string) `json:"-"`
	LogHeaders       bool               `json:"log_headers"`
	LogBody          bool               `json:"log_body"`
	MaxBodySize      int                `json:"max_body_size"`
	SkipPaths        []string           `json:"skip_paths"`
	SensitiveHeaders []string           `json:"sensitive_headers"`
}
