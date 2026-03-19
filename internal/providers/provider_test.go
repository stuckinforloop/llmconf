package providers

import (
	"encoding/json"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

// providerDataForSnapshot returns provider data without function fields (which can't be JSON marshaled)
type providerDataForSnapshot struct {
	Name                 string              `json:"name"`
	DisplayName          string              `json:"displayName"`
	Description          string              `json:"description"`
	DocumentationURL     string              `json:"documentationURL"`
	SupportsModelPinning bool                `json:"supportsModelPinning"`
	RequiredEnvVars      []EnvVarSpecLite    `json:"requiredEnvVars"`
	OptionalEnvVars      []EnvVarSpecLite    `json:"optionalEnvVars"`
	DefaultModels        map[string]string   `json:"defaultModels"`
	ModelSuggestions     map[string][]string `json:"modelSuggestions"`
}

// EnvVarSpecLite is a version of EnvVarSpec without the Validate function
type EnvVarSpecLite struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Sensitive   bool   `json:"sensitive"`
}

func convertEnvVarSpecs(specs []EnvVarSpec) []EnvVarSpecLite {
	result := make([]EnvVarSpecLite, len(specs))
	for i, spec := range specs {
		result[i] = EnvVarSpecLite{
			Name:        spec.Name,
			Description: spec.Description,
			Sensitive:   spec.Sensitive,
		}
	}
	return result
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	// Get all providers and sort by name for deterministic snapshots
	providers := registry.List()
	// Sort providers by name for deterministic ordering
	for i := 0; i < len(providers)-1; i++ {
		for j := i + 1; j < len(providers); j++ {
			if providers[i].Name() > providers[j].Name() {
				providers[i], providers[j] = providers[j], providers[i]
			}
		}
	}
	providerData := make([]providerDataForSnapshot, 0, len(providers))

	for _, p := range providers {
		data := providerDataForSnapshot{
			Name:                 p.Name(),
			DisplayName:          p.DisplayName(),
			Description:          p.Description(),
			DocumentationURL:     p.DocumentationURL(),
			SupportsModelPinning: p.SupportsModelPinning(),
			RequiredEnvVars:      convertEnvVarSpecs(p.RequiredEnvVars()),
			OptionalEnvVars:      convertEnvVarSpecs(p.OptionalEnvVars()),
			DefaultModels:        p.DefaultModels(),
			ModelSuggestions:     p.ModelSuggestions(),
		}
		providerData = append(providerData, data)
	}

	// Serialize to JSON for consistent snapshot
	jsonBytes, err := json.MarshalIndent(providerData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal provider data: %v", err)
	}

	snaps.MatchSnapshot(t, string(jsonBytes))
}

func TestAnthropicProvider(t *testing.T) {
	provider := NewAnthropic()

	// Snapshot provider metadata
	metadata := providerDataForSnapshot{
		Name:                 provider.Name(),
		DisplayName:          provider.DisplayName(),
		Description:          provider.Description(),
		DocumentationURL:     provider.DocumentationURL(),
		SupportsModelPinning: provider.SupportsModelPinning(),
		RequiredEnvVars:      convertEnvVarSpecs(provider.RequiredEnvVars()),
		OptionalEnvVars:      convertEnvVarSpecs(provider.OptionalEnvVars()),
	}

	jsonBytes, _ := json.MarshalIndent(metadata, "", "  ")
	snaps.MatchSnapshot(t, "metadata", string(jsonBytes))

	// Test GenerateEnv with valid config
	config := ProviderConfig{
		Credentials: map[string]string{
			"ANTHROPIC_API_KEY": "test-key",
		},
	}

	env, err := provider.GenerateEnv(config)
	if err != nil {
		t.Fatalf("Expected GenerateEnv to succeed, got error: %v", err)
	}

	envJSON, _ := json.MarshalIndent(env, "", "  ")
	snaps.MatchSnapshot(t, "generateEnv_valid", string(envJSON))

	// Test GenerateEnv with custom base URL
	configWithBaseURL := ProviderConfig{
		Credentials: map[string]string{
			"ANTHROPIC_API_KEY":  "test-key",
			"ANTHROPIC_BASE_URL": "https://custom.api.com",
		},
	}

	envWithBaseURL, _ := provider.GenerateEnv(configWithBaseURL)
	envWithBaseURLJSON, _ := json.MarshalIndent(envWithBaseURL, "", "  ")
	snaps.MatchSnapshot(t, "generateEnv_with_baseurl", string(envWithBaseURLJSON))

	// Test validation error
	err = provider.Validate(ProviderConfig{Credentials: map[string]string{}})
	snaps.MatchSnapshot(t, "validation_error", err.Error())
}

func TestBedrockProvider(t *testing.T) {
	provider := NewBedrock()

	// Snapshot provider metadata
	metadata := providerDataForSnapshot{
		Name:                 provider.Name(),
		DisplayName:          provider.DisplayName(),
		Description:          provider.Description(),
		DocumentationURL:     provider.DocumentationURL(),
		SupportsModelPinning: provider.SupportsModelPinning(),
		RequiredEnvVars:      convertEnvVarSpecs(provider.RequiredEnvVars()),
		OptionalEnvVars:      convertEnvVarSpecs(provider.OptionalEnvVars()),
		DefaultModels:        provider.DefaultModels(),
		ModelSuggestions:     provider.ModelSuggestions(),
	}

	jsonBytes, _ := json.MarshalIndent(metadata, "", "  ")
	snaps.MatchSnapshot(t, "metadata", string(jsonBytes))

	// Test GenerateEnv with SSO auth
	config := ProviderConfig{
		Credentials: map[string]string{
			"AWS_REGION":  "us-east-1",
			"AWS_PROFILE": "my-sso-profile",
		},
		AuthMethod: "sso",
		Models: map[string]string{
			"sonnet": "us.anthropic.claude-sonnet-4-6",
			"haiku":  "us.anthropic.claude-haiku-4-5",
		},
	}

	env, err := provider.GenerateEnv(config)
	if err != nil {
		t.Fatalf("Expected GenerateEnv to succeed, got error: %v", err)
	}

	envJSON, _ := json.MarshalIndent(env, "", "  ")
	snaps.MatchSnapshot(t, "generateEnv_sso", string(envJSON))

	// Test GenerateEnv with API key auth
	configAPIKey := ProviderConfig{
		Credentials: map[string]string{
			"AWS_REGION":            "us-west-2",
			"AWS_ACCESS_KEY_ID":     "AKIAIOSFODNN7EXAMPLE",
			"AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
		AuthMethod: "api_key",
	}

	envAPIKey, _ := provider.GenerateEnv(configAPIKey)
	envAPIKeyJSON, _ := json.MarshalIndent(envAPIKey, "", "  ")
	snaps.MatchSnapshot(t, "generateEnv_api_key", string(envAPIKeyJSON))

	// Test GenerateEnv with bearer token
	configBearer := ProviderConfig{
		Credentials: map[string]string{
			"AWS_REGION":               "eu-west-1",
			"AWS_BEARER_TOKEN_BEDROCK": "test-bearer-token",
		},
		AuthMethod: "bearer_token",
	}

	envBearer, _ := provider.GenerateEnv(configBearer)
	envBearerJSON, _ := json.MarshalIndent(envBearer, "", "  ")
	snaps.MatchSnapshot(t, "generateEnv_bearer", string(envBearerJSON))

	// Test validation error
	err = provider.Validate(ProviderConfig{Credentials: map[string]string{}})
	snaps.MatchSnapshot(t, "validation_error", err.Error())
}

func TestVertexProvider(t *testing.T) {
	provider := NewVertex()

	// Snapshot provider metadata
	metadata := providerDataForSnapshot{
		Name:                 provider.Name(),
		DisplayName:          provider.DisplayName(),
		Description:          provider.Description(),
		DocumentationURL:     provider.DocumentationURL(),
		SupportsModelPinning: provider.SupportsModelPinning(),
		RequiredEnvVars:      convertEnvVarSpecs(provider.RequiredEnvVars()),
		OptionalEnvVars:      convertEnvVarSpecs(provider.OptionalEnvVars()),
		DefaultModels:        provider.DefaultModels(),
		ModelSuggestions:     provider.ModelSuggestions(),
	}

	jsonBytes, _ := json.MarshalIndent(metadata, "", "  ")
	snaps.MatchSnapshot(t, "metadata", string(jsonBytes))

	// Test GenerateEnv
	config := ProviderConfig{
		Credentials: map[string]string{
			"CLOUD_ML_REGION":             "global",
			"ANTHROPIC_VERTEX_PROJECT_ID": "my-gcp-project",
		},
		Models: map[string]string{
			"sonnet": "claude-3-5-sonnet-v2",
		},
	}

	env, err := provider.GenerateEnv(config)
	if err != nil {
		t.Fatalf("Expected GenerateEnv to succeed, got error: %v", err)
	}

	envJSON, _ := json.MarshalIndent(env, "", "  ")
	snaps.MatchSnapshot(t, "generateEnv", string(envJSON))
}

func TestFoundryProvider(t *testing.T) {
	provider := NewFoundry()

	// Snapshot provider metadata
	metadata := providerDataForSnapshot{
		Name:                 provider.Name(),
		DisplayName:          provider.DisplayName(),
		Description:          provider.Description(),
		DocumentationURL:     provider.DocumentationURL(),
		SupportsModelPinning: provider.SupportsModelPinning(),
		RequiredEnvVars:      convertEnvVarSpecs(provider.RequiredEnvVars()),
		OptionalEnvVars:      convertEnvVarSpecs(provider.OptionalEnvVars()),
		DefaultModels:        provider.DefaultModels(),
		ModelSuggestions:     provider.ModelSuggestions(),
	}

	jsonBytes, _ := json.MarshalIndent(metadata, "", "  ")
	snaps.MatchSnapshot(t, "metadata", string(jsonBytes))

	// Test GenerateEnv
	config := ProviderConfig{
		Credentials: map[string]string{
			"ANTHROPIC_FOUNDRY_RESOURCE": "my-foundry-resource",
			"ANTHROPIC_FOUNDRY_API_KEY":  "my-api-key",
		},
		Models: map[string]string{
			"sonnet": "claude-sonnet-4-6",
		},
	}

	env, err := provider.GenerateEnv(config)
	if err != nil {
		t.Fatalf("Expected GenerateEnv to succeed, got error: %v", err)
	}

	envJSON, _ := json.MarshalIndent(env, "", "  ")
	snaps.MatchSnapshot(t, "generateEnv", string(envJSON))
}

func TestFireworksProvider(t *testing.T) {
	provider := NewFireworks()

	// Snapshot provider metadata
	metadata := providerDataForSnapshot{
		Name:                 provider.Name(),
		DisplayName:          provider.DisplayName(),
		Description:          provider.Description(),
		DocumentationURL:     provider.DocumentationURL(),
		SupportsModelPinning: provider.SupportsModelPinning(),
		RequiredEnvVars:      convertEnvVarSpecs(provider.RequiredEnvVars()),
		OptionalEnvVars:      convertEnvVarSpecs(provider.OptionalEnvVars()),
		DefaultModels:        provider.DefaultModels(),
		ModelSuggestions:     provider.ModelSuggestions(),
	}

	jsonBytes, _ := json.MarshalIndent(metadata, "", "  ")
	snaps.MatchSnapshot(t, "metadata", string(jsonBytes))

	// Test GenerateEnv with models (credentials come via apiKeyHelper)
	config := ProviderConfig{
		Credentials: make(map[string]string), // Empty - credentials fetched via apiKeyHelper
		Models: map[string]string{
			"default": "accounts/fireworks/models/kimi-k2p5",
		},
	}

	env, err := provider.GenerateEnv(config)
	if err != nil {
		t.Fatalf("Expected GenerateEnv to succeed, got error: %v", err)
	}

	envJSON, _ := json.MarshalIndent(env, "", "  ")
	snaps.MatchSnapshot(t, "generateEnv", string(envJSON))
}

func TestLiteLLMProvider(t *testing.T) {
	provider := NewLiteLLM()

	// Snapshot provider metadata
	metadata := providerDataForSnapshot{
		Name:                 provider.Name(),
		DisplayName:          provider.DisplayName(),
		Description:          provider.Description(),
		DocumentationURL:     provider.DocumentationURL(),
		SupportsModelPinning: provider.SupportsModelPinning(),
		RequiredEnvVars:      convertEnvVarSpecs(provider.RequiredEnvVars()),
		OptionalEnvVars:      convertEnvVarSpecs(provider.OptionalEnvVars()),
		DefaultModels:        provider.DefaultModels(),
		ModelSuggestions:     provider.ModelSuggestions(),
	}

	jsonBytes, _ := json.MarshalIndent(metadata, "", "  ")
	snaps.MatchSnapshot(t, "metadata", string(jsonBytes))

	// Test GenerateEnv
	config := ProviderConfig{
		Credentials: map[string]string{
			"ANTHROPIC_BASE_URL":   "http://localhost:4000",
			"ANTHROPIC_AUTH_TOKEN": "sk-test-token",
		},
		Models: map[string]string{
			"default": "claude-sonnet-4-6",
		},
	}

	env, err := provider.GenerateEnv(config)
	if err != nil {
		t.Fatalf("Expected GenerateEnv to succeed, got error: %v", err)
	}

	envJSON, _ := json.MarshalIndent(env, "", "  ")
	snaps.MatchSnapshot(t, "generateEnv", string(envJSON))
}

func TestProviderModelSuggestions(t *testing.T) {
	registry := NewRegistry()

	allSuggestions := make(map[string]map[string][]string)

	for _, name := range registry.Names() {
		provider, _ := registry.Get(name)
		suggestions := provider.ModelSuggestions()
		if len(suggestions) > 0 {
			allSuggestions[name] = suggestions
		}
	}

	jsonBytes, _ := json.MarshalIndent(allSuggestions, "", "  ")
	snaps.MatchSnapshot(t, string(jsonBytes))
}
