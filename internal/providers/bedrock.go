package providers

import "fmt"

// Bedrock provider for Amazon Bedrock
type BedrockProvider struct{}

// NewBedrock creates a new Bedrock provider
func NewBedrock() *BedrockProvider {
	return &BedrockProvider{}
}

// Name returns the provider name
func (p *BedrockProvider) Name() string {
	return "bedrock"
}

// DisplayName returns the human-readable name
func (p *BedrockProvider) DisplayName() string {
	return "Amazon Bedrock"
}

// Description returns the provider description
func (p *BedrockProvider) Description() string {
	return "Use Claude through your AWS account with Amazon Bedrock"
}

// DocumentationURL returns the documentation URL
func (p *BedrockProvider) DocumentationURL() string {
	return "https://docs.aws.amazon.com/bedrock/"
}

// RequiredEnvVars returns required environment variables
func (p *BedrockProvider) RequiredEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
		{
			Name:        "CLAUDE_CODE_USE_BEDROCK",
			Description: "Enable Bedrock mode",
			Sensitive:   false,
		},
		{
			Name:        "AWS_REGION",
			Description: "AWS region (e.g., us-east-1)",
			Sensitive:   false,
		},
	}
}

// OptionalEnvVars returns optional environment variables
func (p *BedrockProvider) OptionalEnvVars() []EnvVarSpec {
	return []EnvVarSpec{
		{
			Name:        "AWS_PROFILE",
			Description: "AWS SSO profile name (for SSO authentication)",
			Sensitive:   false,
		},
		{
			Name:        "AWS_ACCESS_KEY_ID",
			Description: "AWS access key ID (for API key authentication)",
			Sensitive:   true,
		},
		{
			Name:        "AWS_SECRET_ACCESS_KEY",
			Description: "AWS secret access key (for API key authentication)",
			Sensitive:   true,
		},
		{
			Name:        "AWS_BEARER_TOKEN_BEDROCK",
			Description: "AWS bearer token for Bedrock",
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
func (p *BedrockProvider) SupportsModelPinning() bool {
	return true
}

// DefaultModels returns default model mappings
func (p *BedrockProvider) DefaultModels() map[string]string {
	return map[string]string{
		"sonnet": "us.anthropic.claude-sonnet-4-6",
		"haiku":  "us.anthropic.claude-haiku-4-5-20251001-v1:0",
		"opus":   "us.anthropic.claude-opus-4-6-v1",
	}
}

// ModelSuggestions returns model suggestions
func (p *BedrockProvider) ModelSuggestions() map[string][]string {
	return map[string][]string{
		"sonnet": {
			"us.anthropic.claude-sonnet-4-6",
			"us.anthropic.claude-sonnet-4-5",
			"global.anthropic.claude-sonnet-4-6",
		},
		"haiku": {
			"us.anthropic.claude-haiku-4-5-20251001-v1:0",
			"global.anthropic.claude-haiku-4-5-20251001-v1:0",
		},
		"opus": {
			"us.anthropic.claude-opus-4-6-v1",
			"global.anthropic.claude-opus-4-6-v1",
		},
	}
}

// GenerateEnv generates environment variables for this provider
func (p *BedrockProvider) GenerateEnv(config ProviderConfig) (map[string]string, error) {
	env := make(map[string]string)

	// Required: Enable Bedrock
	env["CLAUDE_CODE_USE_BEDROCK"] = "1"

	if config.Credentials == nil {
		return nil, fmt.Errorf("no credentials provided")
	}

	// AWS Region (required)
	region, ok := config.Credentials["AWS_REGION"]
	if !ok || region == "" {
		return nil, fmt.Errorf("AWS_REGION is required")
	}
	env["AWS_REGION"] = region

	// Authentication method
	switch config.AuthMethod {
	case "sso", "profile":
		if profile, ok := config.Credentials["AWS_PROFILE"]; ok && profile != "" {
			env["AWS_PROFILE"] = profile
		}
	case "api_key":
		if accessKey, ok := config.Credentials["AWS_ACCESS_KEY_ID"]; ok && accessKey != "" {
			env["AWS_ACCESS_KEY_ID"] = accessKey
		}
		if secretKey, ok := config.Credentials["AWS_SECRET_ACCESS_KEY"]; ok && secretKey != "" {
			env["AWS_SECRET_ACCESS_KEY"] = secretKey
		}
	case "bearer_token":
		if token, ok := config.Credentials["AWS_BEARER_TOKEN_BEDROCK"]; ok && token != "" {
			env["AWS_BEARER_TOKEN_BEDROCK"] = token
		}
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
func (p *BedrockProvider) Validate(config ProviderConfig) error {
	if config.Credentials == nil {
		return fmt.Errorf("credentials are required")
	}

	if region, ok := config.Credentials["AWS_REGION"]; !ok || region == "" {
		return fmt.Errorf("AWS_REGION is required")
	}

	// Validate at least one auth method is provided
	hasAuth := false
	if profile, ok := config.Credentials["AWS_PROFILE"]; ok && profile != "" {
		hasAuth = true
	}
	if accessKey, ok := config.Credentials["AWS_ACCESS_KEY_ID"]; ok && accessKey != "" {
		hasAuth = true
	}
	if token, ok := config.Credentials["AWS_BEARER_TOKEN_BEDROCK"]; ok && token != "" {
		hasAuth = true
	}

	if !hasAuth {
		return fmt.Errorf("at least one authentication method is required (SSO profile, access keys, or bearer token)")
	}

	return nil
}

// ValidateModel validates a model ID for this provider
func (p *BedrockProvider) ValidateModel(modelType string, modelID string) error {
	validPrefixes := []string{
		"us.anthropic.claude-",
		"global.anthropic.claude-",
		"arn:aws:bedrock:",
	}

	for _, prefix := range validPrefixes {
		if len(modelID) >= len(prefix) && modelID[:len(prefix)] == prefix {
			return nil
		}
	}

	return fmt.Errorf("invalid Bedrock model ID: %s. Expected format: us.anthropic.claude-* or arn:aws:bedrock:*", modelID)
}
