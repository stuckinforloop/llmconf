package providers

import "fmt"

// Fireworks provider for Fireworks AI
type FireworksProvider struct{}

// NewFireworks creates a new Fireworks provider
func NewFireworks() *FireworksProvider {
	return &FireworksProvider{}
}

// Name returns the provider name
func (p *FireworksProvider) Name() string {
	return "fireworks"
}

// DisplayName returns the human-readable name
func (p *FireworksProvider) DisplayName() string {
	return "Fireworks AI"
}

// Description returns the provider description
func (p *FireworksProvider) Description() string {
	return "Use Claude through Fireworks AI API"
}

// DocumentationURL returns the documentation URL
func (p *FireworksProvider) DocumentationURL() string {
	return "https://docs.fireworks.ai/"
}

// RequiredEnvVars returns required environment variables
func (p *FireworksProvider) RequiredEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
		{
			Name:        "ANTHROPIC_BASE_URL",
			Description: "Fireworks API base URL",
			Sensitive:   false,
		},
		{
			Name:        "ANTHROPIC_API_KEY",
			Description: "Your Fireworks API key",
			Sensitive:   true,
		},
		{
			Name:        "ANTHROPIC_MODEL",
			Description: "Default model (all models must be the same for Fireworks)",
			Sensitive:   false,
		},
		{
			Name:        "ANTHROPIC_SMALL_FAST_MODEL",
			Description: "Small/fast model (same as ANTHROPIC_MODEL for Fireworks)",
			Sensitive:   false,
		},
	}
}

// OptionalEnvVars returns optional environment variables
func (p *FireworksProvider) OptionalEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
		{
			Name:        "ANTHROPIC_DEFAULT_SONNET_MODEL",
			Description: "Pinned Sonnet model ID (same as main model)",
			Sensitive:   false,
		},
		{
			Name:        "ANTHROPIC_DEFAULT_HAIKU_MODEL",
			Description: "Pinned Haiku model ID (same as main model)",
			Sensitive:   false,
		},
		{
			Name:        "ANTHROPIC_DEFAULT_OPUS_MODEL",
			Description: "Pinned Opus model ID (same as main model)",
			Sensitive:   false,
		},
		{
			Name:        "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS",
			Description: "Disable experimental betas",
			Sensitive:   false,
		},
	}
}

// SupportsModelPinning returns whether model pinning is supported
func (p *FireworksProvider) SupportsModelPinning() bool {
	return true
}

// DefaultModels returns default model mappings
func (p *FireworksProvider) DefaultModels() map[string]string {
	return map[string]string{
		"default": "accounts/fireworks/models/kimi-k2p5",
		"sonnet":  "accounts/fireworks/models/kimi-k2p5",
		"haiku":   "accounts/fireworks/models/kimi-k2p5",
		"opus":    "accounts/fireworks/models/kimi-k2p5",
	}
}

// ModelSuggestions returns model suggestions
func (p *FireworksProvider) ModelSuggestions() map[string][]string {
	return map[string][]string{
		"default": {
			"accounts/fireworks/models/kimi-k2p5",
			"accounts/fireworks/models/glm-5",
		},
		"sonnet": {
			"accounts/fireworks/models/kimi-k2p5",
		},
		"haiku": {
			"accounts/fireworks/models/kimi-k2p5",
		},
		"opus": {
			"accounts/fireworks/models/kimi-k2p5",
		},
	}
}

// GenerateEnv generates environment variables for this provider
func (p *FireworksProvider) GenerateEnv(config ProviderConfig) (map[string]string, error) {
	env := make(map[string]string)

	// Base URL
	env["ANTHROPIC_BASE_URL"] = "https://api.fireworks.ai/inference"

	// Model - Fireworks requires all models to be the same
	model := "accounts/fireworks/models/kimi-k2p5" // default
	if config.Models != nil {
		if m, ok := config.Models["default"]; ok && m != "" {
			model = m
		}
	}

	// Set all model vars to the same value (Fireworks requirement)
	env["ANTHROPIC_MODEL"] = model
	env["ANTHROPIC_SMALL_FAST_MODEL"] = model
	env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = model
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = model
	env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = model

	// Disable experimental betas
	env["CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"] = "1"

	// Add extra env vars
	for k, v := range config.ExtraEnv {
		env[k] = v
	}

	return env, nil
}

// GetAPIKeyHelper returns the apiKeyHelper command for secure credential retrieval
func (p *FireworksProvider) GetAPIKeyHelper() string {
	return "llmconf credential get fireworks ANTHROPIC_API_KEY"
}

// Validate validates provider configuration
func (p *FireworksProvider) Validate(config ProviderConfig) error {
	if config.Credentials == nil {
		return fmt.Errorf("credentials are required")
	}

	if apiKey, ok := config.Credentials["ANTHROPIC_API_KEY"]; !ok || apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY is required")
	}

	return nil
}

// ValidateModel validates a model ID for this provider
func (p *FireworksProvider) ValidateModel(modelType string, modelID string) error {
	// Fireworks models typically start with accounts/fireworks/models/
	if len(modelID) >= 28 && modelID[:28] == "accounts/fireworks/models/" {
		return nil
	}

	return fmt.Errorf("invalid Fireworks model ID: %s. Expected format: accounts/fireworks/models/*", modelID)
}
