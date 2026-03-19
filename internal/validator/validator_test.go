package validator

import (
	"encoding/json"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stuckinforloop/llmconf/internal/config"
	"github.com/stuckinforloop/llmconf/internal/providers"
)

func TestValidateProviderConfig(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name           string
		provider       string
		config         providers.ProviderConfig
		expectValid    bool
		expectErrors   int
		expectWarnings int
	}{
		{
			name:     "valid anthropic",
			provider: "anthropic",
			config: providers.ProviderConfig{
				Credentials: map[string]string{
					"ANTHROPIC_API_KEY": "test-key",
				},
			},
			expectValid: true,
		},
		{
			name:     "invalid anthropic - missing key",
			provider: "anthropic",
			config: providers.ProviderConfig{
				Credentials: map[string]string{},
			},
			expectValid:  false,
			expectErrors: 1,
		},
		{
			name:     "valid bedrock with sso",
			provider: "bedrock",
			config: providers.ProviderConfig{
				Credentials: map[string]string{
					"AWS_REGION":  "us-east-1",
					"AWS_PROFILE": "test-profile",
				},
				AuthMethod: "sso",
			},
			expectValid: true,
		},
		{
			name:     "valid bedrock with api key",
			provider: "bedrock",
			config: providers.ProviderConfig{
				Credentials: map[string]string{
					"AWS_REGION":            "us-west-2",
					"AWS_ACCESS_KEY_ID":     "AKIAIOSFODNN7EXAMPLE",
					"AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
				AuthMethod: "api_key",
			},
			expectValid: true,
		},
		{
			name:     "invalid bedrock - missing region",
			provider: "bedrock",
			config: providers.ProviderConfig{
				Credentials: map[string]string{
					"AWS_PROFILE": "test-profile",
				},
			},
			expectValid:  false,
			expectErrors: 1,
		},
		{
			name:     "bedrock with no auth method",
			provider: "bedrock",
			config: providers.ProviderConfig{
				Credentials: map[string]string{
					"AWS_REGION": "us-east-1",
				},
			},
			expectValid:  false,
			expectErrors: 1, // Bedrock requires an auth method
		},
		{
			name:     "unknown provider",
			provider: "unknown",
			config: providers.ProviderConfig{
				Credentials: map[string]string{},
			},
			expectValid:  false,
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateProviderConfig(tt.provider, tt.config)

			// Snapshot the full validation result
			resultJSON, _ := json.MarshalIndent(result, "", "  ")
			snaps.MatchSnapshot(t, tt.name, string(resultJSON))

			// Also do traditional assertions
			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, result.Valid)
			}
			if len(result.Errors) != tt.expectErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectErrors, len(result.Errors), result.Errors)
			}
			if len(result.Warnings) != tt.expectWarnings {
				t.Errorf("Expected %d warnings, got %d: %v", tt.expectWarnings, len(result.Warnings), result.Warnings)
			}
		})
	}
}

func TestValidateSettings(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name         string
		settings     *config.ClaudeSettings
		expectValid  bool
		expectErrors int
	}{
		{
			name: "valid bedrock settings",
			settings: &config.ClaudeSettings{
				Env: map[string]string{
					"CLAUDE_CODE_USE_BEDROCK":        "1",
					"AWS_REGION":                   "us-east-1",
					"AWS_PROFILE":                  "test-profile",
					"ANTHROPIC_DEFAULT_SONNET_MODEL": "us.anthropic.claude-sonnet-4-6",
				},
				APIKeyHelper: "llmconf credential get bedrock ANTHROPIC_API_KEY",
			},
			expectValid: true,
		},
		{
			name: "valid fireworks settings",
			settings: &config.ClaudeSettings{
				Env: map[string]string{
					"ANTHROPIC_MODEL":                "accounts/fireworks/models/kimi-k2p5",
					"ANTHROPIC_SMALL_FAST_MODEL":     "accounts/fireworks/models/kimi-k2p5",
					"ANTHROPIC_DEFAULT_SONNET_MODEL": "accounts/fireworks/models/kimi-k2p5",
					"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "accounts/fireworks/models/kimi-k2p5",
					"ANTHROPIC_DEFAULT_OPUS_MODEL":   "accounts/fireworks/models/kimi-k2p5",
				},
				APIKeyHelper: "llmconf credential get fireworks ANTHROPIC_API_KEY",
			},
			expectValid: true,
		},
		{
			name: "missing env map",
			settings: &config.ClaudeSettings{
				Env: nil,
			},
			expectValid:  false,
			expectErrors: 1,
		},
		{
			name: "empty settings - no provider",
			settings: &config.ClaudeSettings{
				Env: map[string]string{},
			},
			expectValid: true, // Valid but has warning
		},
		{
			name: "anthropic direct api settings",
			settings: &config.ClaudeSettings{
				Env: map[string]string{
					"ANTHROPIC_API_KEY": "test-key",
				},
			},
			expectValid: true,
		},
		{
			name: "vertex ai settings",
			settings: &config.ClaudeSettings{
				Env: map[string]string{
					"CLOUD_ML_REGION":             "global",
					"ANTHROPIC_VERTEX_PROJECT_ID": "my-project",
				},
			},
			expectValid: true,
		},
		{
			name: "litellm settings",
			settings: &config.ClaudeSettings{
				Env: map[string]string{
					"ANTHROPIC_BASE_URL":   "http://localhost:4000",
					"ANTHROPIC_AUTH_TOKEN": "sk-test",
				},
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateSettings(tt.settings)

			// Snapshot the full validation result
			resultJSON, _ := json.MarshalIndent(result, "", "  ")
			snaps.MatchSnapshot(t, tt.name, string(resultJSON))

			// Traditional assertions
			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, result.Valid)
			}
			if len(result.Errors) != tt.expectErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectErrors, len(result.Errors), result.Errors)
			}
		})
	}
}

func TestValidateEnvVarName(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		valid   bool
	}{
		{"valid uppercase", "API_KEY", true},
		{"valid lowercase", "api_key", true},
		{"valid with numbers", "API_KEY_123", true},
		{"valid starts underscore", "_API_KEY", true},
		{"invalid empty", "", false},
		{"invalid starts number", "1API_KEY", false},
		{"invalid contains hyphen", "API-KEY", false},
		{"invalid contains space", "API KEY", false},
		{"invalid contains special", "API$KEY", false},
	}

	results := make(map[string]map[string]interface{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvVarName(tt.varName)

			result := map[string]interface{}{
				"varName": tt.varName,
				"valid":   err == nil,
			}
			if err != nil {
				result["error"] = err.Error()
			}

			results[tt.name] = result

			if tt.valid && err != nil {
				t.Errorf("Expected %s to be valid, got error: %v", tt.varName, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Expected %s to be invalid, but got no error", tt.varName)
			}
		})
	}

	// Snapshot all validation results
	jsonBytes, _ := json.MarshalIndent(results, "", "  ")
	snaps.MatchSnapshot(t, "env_var_validation", string(jsonBytes))
}

func TestIsSensitiveVar(t *testing.T) {
	tests := []struct {
		name      string
		varName   string
		sensitive bool
	}{
		{"api key", "ANTHROPIC_API_KEY", true},
		{"secret", "AWS_SECRET_ACCESS_KEY", true},
		{"token", "AWS_BEARER_TOKEN", true},
		{"password", "DB_PASSWORD", true},
		{"credential", "CREDENTIAL", true},
		{"private key", "PRIVATE_KEY", true},
		{"auth", "AUTH_TOKEN", true},
		{"region", "AWS_REGION", false},
		{"profile", "AWS_PROFILE", false},
		{"endpoint", "API_ENDPOINT", false},
		{"base url", "ANTHROPIC_BASE_URL", false},
	}

	results := make(map[string]map[string]interface{})

	for _, tt := range tests {
		result := map[string]interface{}{
			"varName":   tt.varName,
			"sensitive": IsSensitiveVar(tt.varName),
		}
		results[tt.name] = result
	}

	// Snapshot all results
	jsonBytes, _ := json.MarshalIndent(results, "", "  ")
	snaps.MatchSnapshot(t, "sensitive_var_detection", string(jsonBytes))
}

func TestValidationResultStruct(t *testing.T) {
	// Test various validation result combinations for snapshotting
	results := []ValidationResult{
		{
			Valid:    true,
			Errors:   []string{},
			Warnings: []string{},
		},
		{
			Valid:  false,
			Errors: []string{"API key is required"},
			Warnings: []string{
				"Model pinning recommended",
			},
		},
		{
			Valid: false,
			Errors: []string{
				"AWS_REGION is required",
				"No authentication method provided",
			},
			Warnings: []string{
				"SSO profile recommended over API keys",
			},
		},
	}

	jsonBytes, _ := json.MarshalIndent(results, "", "  ")
	snaps.MatchSnapshot(t, "validation_result_examples", string(jsonBytes))
}
