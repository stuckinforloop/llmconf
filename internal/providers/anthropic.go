package providers

import "fmt"

// Anthropic provider for direct API access
type AnthropicProvider struct{}

// NewAnthropic creates a new Anthropic provider
func NewAnthropic() *AnthropicProvider {
	return &AnthropicProvider{}
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// DisplayName returns the human-readable name
func (p *AnthropicProvider) DisplayName() string {
	return "Anthropic Direct API"
}

// Description returns the provider description
func (p *AnthropicProvider) Description() string {
	return "Use Claude directly through the Anthropic API"
}

// DocumentationURL returns the documentation URL
func (p *AnthropicProvider) DocumentationURL() string {
	return "https://docs.anthropic.com/"
}

// RequiredEnvVars returns required environment variables
func (p *AnthropicProvider) RequiredEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
		{
			Name:        "ANTHROPIC_API_KEY",
			Description: "Your Anthropic API key",
			Sensitive:   true,
			Validate:    nonEmpty,
		},
	}
}

// OptionalEnvVars returns optional environment variables
func (p *AnthropicProvider) OptionalEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
		{
			Name:        "ANTHROPIC_BASE_URL",
			Description: "Custom base URL for Anthropic API (optional)",
			Sensitive:   false,
		},
	}
}

// SupportsModelPinning returns whether model pinning is supported
func (p *AnthropicProvider) SupportsModelPinning() bool {
	return false
}

// DefaultModels returns default model mappings
func (p *AnthropicProvider) DefaultModels() map[string]string {
	return map[string]string{}
}

// ModelSuggestions returns model suggestions
func (p *AnthropicProvider) ModelSuggestions() map[string][]string {
	return map[string][]string{}
}

// GenerateEnv generates environment variables for this provider
func (p *AnthropicProvider) GenerateEnv(config ProviderConfig) (map[string]string, error) {
	env := make(map[string]string)

	if config.Credentials == nil {
		return nil, fmt.Errorf("no credentials provided")
	}

	// Required credentials
	apiKey, ok := config.Credentials["ANTHROPIC_API_KEY"]
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is required")
	}
	env["ANTHROPIC_API_KEY"] = apiKey

	// Optional base URL
	if baseURL, ok := config.Credentials["ANTHROPIC_BASE_URL"]; ok && baseURL != "" {
		env["ANTHROPIC_BASE_URL"] = baseURL
	}

	// Add extra env vars
	for k, v := range config.ExtraEnv {
		env[k] = v
	}

	return env, nil
}

// Validate validates provider configuration
func (p *AnthropicProvider) Validate(config ProviderConfig) error {
	if config.Credentials == nil {
		return fmt.Errorf("credentials are required")
	}

	if apiKey, ok := config.Credentials["ANTHROPIC_API_KEY"]; !ok || apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY is required")
	}

	return nil
}

// ValidateModel validates a model ID for this provider
func (p *AnthropicProvider) ValidateModel(modelType string, modelID string) error {
	// Anthropic doesn't use model pinning
	return fmt.Errorf("model pinning not supported for Anthropic Direct API")
}

// nonEmpty is a basic validation function
func nonEmpty(value string) error {
	if value == "" {
		return fmt.Errorf("value cannot be empty")
	}
	return nil
}
