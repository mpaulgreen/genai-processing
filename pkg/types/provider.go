package types

// ProviderConfig holds configuration for creating LLM providers
type ProviderConfig struct {
	APIKey     string                 `json:"api_key"`
	Endpoint   string                 `json:"endpoint,omitempty"`
	ModelName  string                 `json:"model_name,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}
