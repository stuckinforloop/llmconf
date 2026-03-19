package providers

import "fmt"

// Foundry provider for Microsoft Foundry
type FoundryProvider struct{}

// NewFoundry creates a new Foundry provider
func NewFoundry() *FoundryProvider {
	return &FoundryProvider{}
}

// Name returns the provider name
func (p *FoundryProvider) Name() string {
	return "foundry"
}

// DisplayName returns the human-readable name
func (p *FoundryProvider) DisplayName() string {
	return "Microsoft Foundry"
}

// Description returns the provider description
func (p *FoundryProvider) Description() string {
	return "Use Claude through Microsoft Azure AI Foundry"
}

// DocumentationURL returns the documentation URL
func (p *FoundryProvider) DocumentationURL() string {
	return "https://learn.microsoft.com/en-us/azure/ai-foundry/"
}

// RequiredEnvVars returns required environment variables
func (p *FoundryProvider) RequiredEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
		{
			Name:        "CLAUDE_CODE_USE_FOUNDRY",
			Description: "Enable Foundry mode",
			Sensitive:   false,
		},
		{
			Name:        "ANTHROPIC_FOUNDRY_RESOURCE",
			Description: "Azure AI Foundry resource name",
			Sensitive:   false,
		},
	}
}

// OptionalEnvVars returns optional environment variables
func (p *FoundryProvider) OptionalEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
		{
			Name:        "ANTHROPIC_FOUNDRY_API_KEY",
			Description: "Azure AI Foundry API key",
			Sensitive:   true,
		},
		{
			Name:        "ANTHROPIC_DEFAULT_OPUS_MODEL",
			Description: "Pinned Opus model ID",
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
	}
}

// SupportsModelPinning returns whether model pinning is supported
func (p *FoundryProvider) SupportsModelPinning() bool {
	return true
}

// DefaultModels returns default model mappings
func (p *FoundryProvider) DefaultModels() map[string]string {
	return map[string]string{
		"sonnet": "claude-sonnet-4-6",
		"haiku":  "claude-haiku-4-5",
		"opus":   "claude-opus-4-6",
	}
}

// ModelSuggestions returns model suggestions
func (p *FoundryProvider) ModelSuggestions() map[string][]string {
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
func (p *FoundryProvider) GenerateEnv(config ProviderConfig) (map[string]string, error) {
	env := make(map[string]string)

	// Required: Enable Foundry
	env["CLAUDE_CODE_USE_FOUNDRY"] = "1"

	if config.Credentials == nil {
		return nil, fmt.Errorf("no credentials provided")
	}

	// Resource name
	resource, ok := config.Credentials["ANTHROPIC_FOUNDRY_RESOURCE"]
	if !ok || resource == "" {
		return nil, fmt.Errorf("ANTHROPIC_FOUNDRY_RESOURCE is required")
	}
	env["ANTHROPIC_FOUNDRY_RESOURCE"] = resource

	// API Key (optional, can use Azure CLI)
	if apiKey, ok := config.Credentials["ANTHROPIC_FOUNDRY_API_KEY"]; ok && apiKey != "" {
		env["ANTHROPIC_FOUNDRY_API_KEY"] = apiKey
	}

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
	}

	// Add extra env vars
	for k, v := range config.ExtraEnv {
		env[k] = v
	}

	return env, nil
}

// Validate validates provider configuration
func (p *FoundryProvider) Validate(config ProviderConfig) error {
	if config.Credentials == nil {
		return fmt.Errorf("credentials are required")
	}

	if resource, ok := config.Credentials["ANTHROPIC_FOUNDRY_RESOURCE"]; !ok || resource == "" {
		return fmt.Errorf("ANTHROPIC_FOUNDRY_RESOURCE is required")
	}

	return nil
}

// ValidateModel validates a model ID for this provider
func (p *FoundryProvider) ValidateModel(modelType string, modelID string) error {
	validPrefixes := []string{
		"claude-sonnet-",
		"claude-haiku-",
		"claude-opus-",
	}

	for _, prefix := range validPrefixes {
		if len(modelID) >= len(prefix) && modelID[:len(prefix)] == prefix {
			return nil
		}
	}

	return fmt.Errorf("invalid Foundry model ID: %s. Expected format: claude-*", modelID)
}
