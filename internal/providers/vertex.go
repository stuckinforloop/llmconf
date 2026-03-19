package providers

import "fmt"

// Vertex provider for Google Vertex AI
type VertexProvider struct{}

// NewVertex creates a new Vertex provider
func NewVertex() *VertexProvider {
	return &VertexProvider{}
}

// Name returns the provider name
func (p *VertexProvider) Name() string {
	return "vertex"
}

// DisplayName returns the human-readable name
func (p *VertexProvider) DisplayName() string {
	return "Google Vertex AI"
}

// Description returns the provider description
func (p *VertexProvider) Description() string {
	return "Use Claude through Google Cloud Platform with Vertex AI"
}

// DocumentationURL returns the documentation URL
func (p *VertexProvider) DocumentationURL() string {
	return "https://cloud.google.com/vertex-ai"
}

// RequiredEnvVars returns required environment variables
func (p *VertexProvider) RequiredEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
		{
			Name:        "CLAUDE_CODE_USE_VERTEX",
			Description: "Enable Vertex AI mode",
			Sensitive:   false,
		},
		{
			Name:        "CLOUD_ML_REGION",
			Description: "GCP region (e.g., global, us-central1)",
			Sensitive:   false,
		},
		{
			Name:        "ANTHROPIC_VERTEX_PROJECT_ID",
			Description: "GCP project ID",
			Sensitive:   false,
		},
	}
}

// OptionalEnvVars returns optional environment variables
func (p *VertexProvider) OptionalEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
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
func (p *VertexProvider) SupportsModelPinning() bool {
	return true
}

// DefaultModels returns default model mappings
func (p *VertexProvider) DefaultModels() map[string]string {
	return map[string]string{
		"sonnet": "claude-sonnet-4-6",
		"haiku":  "claude-haiku-4-5@20251001",
		"opus":   "claude-opus-4-6",
	}
}

// ModelSuggestions returns model suggestions
func (p *VertexProvider) ModelSuggestions() map[string][]string {
	return map[string][]string{
		"sonnet": {
			"claude-sonnet-4-6",
			"claude-sonnet-4-5",
		},
		"haiku": {
			"claude-haiku-4-5@20251001",
		},
		"opus": {
			"claude-opus-4-6",
		},
	}
}

// GenerateEnv generates environment variables for this provider
func (p *VertexProvider) GenerateEnv(config ProviderConfig) (map[string]string, error) {
	env := make(map[string]string)

	// Required: Enable Vertex
	env["CLAUDE_CODE_USE_VERTEX"] = "1"

	if config.Credentials == nil {
		return nil, fmt.Errorf("no credentials provided")
	}

	// GCP Region
	region, ok := config.Credentials["CLOUD_ML_REGION"]
	if !ok || region == "" {
		return nil, fmt.Errorf("CLOUD_ML_REGION is required")
	}
	env["CLOUD_ML_REGION"] = region

	// GCP Project ID
	projectID, ok := config.Credentials["ANTHROPIC_VERTEX_PROJECT_ID"]
	if !ok || projectID == "" {
		return nil, fmt.Errorf("ANTHROPIC_VERTEX_PROJECT_ID is required")
	}
	env["ANTHROPIC_VERTEX_PROJECT_ID"] = projectID

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
func (p *VertexProvider) Validate(config ProviderConfig) error {
	if config.Credentials == nil {
		return fmt.Errorf("credentials are required")
	}

	if region, ok := config.Credentials["CLOUD_ML_REGION"]; !ok || region == "" {
		return fmt.Errorf("CLOUD_ML_REGION is required")
	}

	if projectID, ok := config.Credentials["ANTHROPIC_VERTEX_PROJECT_ID"]; !ok || projectID == "" {
		return fmt.Errorf("ANTHROPIC_VERTEX_PROJECT_ID is required")
	}

	return nil
}

// ValidateModel validates a model ID for this provider
func (p *VertexProvider) ValidateModel(modelType string, modelID string) error {
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

	return fmt.Errorf("invalid Vertex model ID: %s. Expected format: claude-*", modelID)
}
