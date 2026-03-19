package providers

import "fmt"

// LiteLLM provider for LiteLLM Proxy
type LiteLLMProvider struct{}

// NewLiteLLM creates a new LiteLLM provider
func NewLiteLLM() *LiteLLMProvider {
	return &LiteLLMProvider{}
}

// Name returns the provider name
func (p *LiteLLMProvider) Name() string {
	return "litellm"
}

// DisplayName returns the human-readable name
func (p *LiteLLMProvider) DisplayName() string {
	return "LiteLLM Proxy"
}

// Description returns the provider description
func (p *LiteLLMProvider) Description() string {
	return "Use Claude through a LiteLLM proxy server"
}

// DocumentationURL returns the documentation URL
func (p *LiteLLMProvider) DocumentationURL() string {
	return "https://docs.litellm.ai/"
}

// RequiredEnvVars returns required environment variables
func (p *LiteLLMProvider) RequiredEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
		{
			Name:        "ANTHROPIC_BASE_URL",
			Description: "LiteLLM proxy base URL (e.g., http://localhost:4000)",
			Sensitive:   false,
		},
		{
			Name:        "ANTHROPIC_AUTH_TOKEN",
			Description: "LiteLLM proxy API key",
			Sensitive:   true,
		},
	}
}

// OptionalEnvVars returns optional environment variables
func (p *LiteLLMProvider) OptionalEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
		{
			Name:        "ANTHROPIC_MODEL",
			Description: "Default model",
			Sensitive:   false,
		},
		{
			Name:        "ANTHROPIC_DEFAULT_SONNET_MODEL",
			Description: "Pinned Sonnet model ID",
			Sensitive:   false,
		},
		{
			Name:        "ANTHROPIC_DEFAULT_HAIKU_MODEL",
			Description: "Pinned Haiku model ID",
			Sensitive:   false,
		},
		{
			Name:        "ANTHROPIC_DEFAULT_OPUS_MODEL",
			Description: "Pinned Opus model ID",
			Sensitive:   false,
		},
	}
}

// SupportsModelPinning returns whether model pinning is supported
func (p *LiteLLMProvider) SupportsModelPinning() bool {
	return true
}

// DefaultModels returns default model mappings
func (p *LiteLLMProvider) DefaultModels() map[string]string {
	return map[string]string{
		"sonnet": "claude-sonnet-4-6",
		"haiku":  "claude-haiku-4-5",
		"opus":   "claude-opus-4-6",
	}
}

// ModelSuggestions returns model suggestions
func (p *LiteLLMProvider) ModelSuggestions() map[string][]string {
	return map[string][]string{
		"sonnet": {
			"claude-sonnet-4-6",
			"claude-sonnet-4-5",
		},
		"haiku": {
			"claude-haiku-4-5",
		},
		"opus": {
			"claude-opus-4-6",
		},
	}
}

// GenerateEnv generates environment variables for this provider
func (p *LiteLLMProvider) GenerateEnv(config ProviderConfig) (map[string]string, error) {
	env := make(map[string]string)

	if config.Credentials == nil {
		return nil, fmt.Errorf("no credentials provided")
	}

	// Base URL
	baseURL, ok := config.Credentials["ANTHROPIC_BASE_URL"]
	if !ok || baseURL == "" {
		return nil, fmt.Errorf("ANTHROPIC_BASE_URL is required")
	}
	env["ANTHROPIC_BASE_URL"] = baseURL

	// Auth Token
	authToken, ok := config.Credentials["ANTHROPIC_AUTH_TOKEN"]
	if !ok || authToken == "" {
		return nil, fmt.Errorf("ANTHROPIC_AUTH_TOKEN is required")
	}
	env["ANTHROPIC_AUTH_TOKEN"] = authToken

	// Model pinning
	if config.Models != nil {
		if opus, ok := config.Models["opus"]; ok && opus != "" {
			env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = opus
		}
		if sonnet, ok := config.Models["sonnet"]; ok && sonnet != "" {
			env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = sonnet
		}
		if haiku, ok := config.Models["haiku"]; ok && haiku != "" {
			env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = haiku
		}
		if model, ok := config.Models["default"]; ok && model != "" {
			env["ANTHROPIC_MODEL"] = model
		}
	}

	// Add extra env vars
	for k, v := range config.ExtraEnv {
		env[k] = v
	}

	return env, nil
}

// Validate validates provider configuration
func (p *LiteLLMProvider) Validate(config ProviderConfig) error {
	if config.Credentials == nil {
		return fmt.Errorf("credentials are required")
	}

	if baseURL, ok := config.Credentials["ANTHROPIC_BASE_URL"]; !ok || baseURL == "" {
		return fmt.Errorf("ANTHROPIC_BASE_URL is required")
	}

	if authToken, ok := config.Credentials["ANTHROPIC_AUTH_TOKEN"]; !ok || authToken == "" {
		return fmt.Errorf("ANTHROPIC_AUTH_TOKEN is required")
	}

	return nil
}

// ValidateModel validates a model ID for this provider
func (p *LiteLLMProvider) ValidateModel(modelType string, modelID string) error {
	// LiteLLM is flexible with model IDs
	if modelID == "" {
		return fmt.Errorf("model ID cannot be empty")
	}
	return nil
}
