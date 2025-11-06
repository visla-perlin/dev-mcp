package config

// LLMConfig represents the configuration for large language models
type LLMConfig struct {
	Providers []ProviderConfig `yaml:"providers"`
}

// ProviderConfig represents the configuration for a specific provider
type ProviderConfig struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	APIKey   string `yaml:"api_key"`
	Endpoint string `yaml:"endpoint"`
	Enabled  bool   `yaml:"enabled"`
	Model    string `yaml:"model"`
}

// OpenAIConfig represents the configuration for OpenAI
type OpenAIConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

// AnthropicConfig represents the configuration for Anthropic
type AnthropicConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

// LocalConfig represents the configuration for local models
type LocalConfig struct {
	Endpoint string `yaml:"endpoint"`
	Model    string `yaml:"model"`
}