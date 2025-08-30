package auth

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
