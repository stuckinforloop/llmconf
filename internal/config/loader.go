package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SettingsLoader handles loading and saving Claude Code settings.json files
type SettingsLoader struct{}

// NewSettingsLoader creates a new SettingsLoader
func NewSettingsLoader() *SettingsLoader {
	return &SettingsLoader{}
}

// Load reads a settings.json file
func (sl *SettingsLoader) Load(path string) (*ClaudeSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ClaudeSettings{
				Env: make(map[string]string),
			}, nil
		}
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}

	if settings.Env == nil {
		settings.Env = make(map[string]string)
	}

	return &settings, nil
}

// Save writes a settings.json file
func (sl *SettingsLoader) Save(path string, settings *ClaudeSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

// Merge merges settings from multiple sources
func (sl *SettingsLoader) Merge(settings ...*ClaudeSettings) *ClaudeSettings {
	result := &ClaudeSettings{
		Env: make(map[string]string),
	}

	for _, s := range settings {
		if s == nil {
			continue
		}

		// Merge env vars
		for k, v := range s.Env {
			result.Env[k] = v
		}

		// Take last non-empty values for other fields
		if s.APIKeyHelper != "" {
			result.APIKeyHelper = s.APIKeyHelper
		}
		if s.AWSAuthRefresh != "" {
			result.AWSAuthRefresh = s.AWSAuthRefresh
		}
		if s.AWSCredentialExport != "" {
			result.AWSCredentialExport = s.AWSCredentialExport
		}
		if len(s.ModelOverrides) > 0 {
			result.ModelOverrides = s.ModelOverrides
		}
		if len(s.AvailableModels) > 0 {
			result.AvailableModels = s.AvailableModels
		}
	}

	return result
}

// ToJSON converts settings to JSON string
func (sl *SettingsLoader) ToJSON(settings *ClaudeSettings) (string, error) {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON parses settings from JSON string
func (sl *SettingsLoader) FromJSON(jsonStr string) (*ClaudeSettings, error) {
	var settings ClaudeSettings
	if err := json.Unmarshal([]byte(jsonStr), &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// AddEnvVar adds an environment variable to settings
func (sl *SettingsLoader) AddEnvVar(settings *ClaudeSettings, key, value string) {
	if settings.Env == nil {
		settings.Env = make(map[string]string)
	}
	settings.Env[key] = value
}

// RemoveEnvVar removes an environment variable from settings
func (sl *SettingsLoader) RemoveEnvVar(settings *ClaudeSettings, key string) {
	delete(settings.Env, key)
}

// GetEnvVar retrieves an environment variable from settings
func (sl *SettingsLoader) GetEnvVar(settings *ClaudeSettings, key string) (string, bool) {
	val, ok := settings.Env[key]
	return val, ok
}

// HasEnvVar checks if an environment variable exists in settings
func (sl *SettingsLoader) HasEnvVar(settings *ClaudeSettings, key string) bool {
	_, ok := settings.Env[key]
	return ok
}

// SetModelOverride sets a model override
func (sl *SettingsLoader) SetModelOverride(settings *ClaudeSettings, modelType, modelID string) {
	if settings.ModelOverrides == nil {
		settings.ModelOverrides = make(map[string]string)
	}
	settings.ModelOverrides[modelType] = modelID
}

// DetectProvider detects the active provider from settings
func (sl *SettingsLoader) DetectProvider(settings *ClaudeSettings) string {
	env := settings.Env

	if env["CLAUDE_CODE_USE_BEDROCK"] == "1" {
		return "bedrock"
	}
	if env["CLAUDE_CODE_USE_VERTEX"] == "1" {
		return "vertex"
	}
	if env["CLAUDE_CODE_USE_FOUNDRY"] == "1" {
		return "foundry"
	}
	if env["ANTHROPIC_BASE_URL"] == "https://api.fireworks.ai/inference" {
		return "fireworks"
	}
	if env["ANTHROPIC_BASE_URL"] != "" && env["ANTHROPIC_AUTH_TOKEN"] != "" {
		return "litellm"
	}
	if env["ANTHROPIC_API_KEY"] != "" && env["CLAUDE_CODE_USE_BEDROCK"] != "1" {
		return "anthropic"
	}

	return ""
}

// GetProviderEnv extracts provider-specific env vars
func GetProviderEnv(settings *ClaudeSettings, provider string) map[string]string {
	result := make(map[string]string)

	// Define env var prefixes/patterns for each provider
	patterns := map[string][]string{
		"bedrock": {
			"CLAUDE_CODE_USE_BEDROCK",
			"AWS_REGION",
			"AWS_PROFILE",
			"AWS_ACCESS_KEY_ID",
			"AWS_SECRET_ACCESS_KEY",
			"AWS_BEARER_TOKEN_BEDROCK",
			"ANTHROPIC_DEFAULT_",
		},
		"vertex": {
			"CLAUDE_CODE_USE_VERTEX",
			"CLOUD_ML_REGION",
			"ANTHROPIC_VERTEX_PROJECT_ID",
			"ANTHROPIC_DEFAULT_",
		},
		"foundry": {
			"CLAUDE_CODE_USE_FOUNDRY",
			"ANTHROPIC_FOUNDRY_",
			"ANTHROPIC_DEFAULT_",
		},
		"fireworks": {
			"ANTHROPIC_BASE_URL",
			"ANTHROPIC_API_KEY",
			"ANTHROPIC_MODEL",
			"ANTHROPIC_SMALL_FAST_MODEL",
			"ANTHROPIC_DEFAULT_",
			"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS",
		},
		"litellm": {
			"ANTHROPIC_BASE_URL",
			"ANTHROPIC_AUTH_TOKEN",
			"ANTHROPIC_DEFAULT_",
		},
		"anthropic": {
			"ANTHROPIC_API_KEY",
			"ANTHROPIC_BASE_URL",
		},
	}

	providerPatterns, ok := patterns[provider]
	if !ok {
		return result
	}

	for key, value := range settings.Env {
		for _, pattern := range providerPatterns {
			if key == pattern || (len(pattern) < len(key) && key[:len(pattern)] == pattern) {
				result[key] = value
				break
			}
		}
	}

	return result
}

// ClearProviderEnv removes all env vars for a specific provider
func (sl *SettingsLoader) ClearProviderEnv(settings *ClaudeSettings, provider string) {
	providerEnv := GetProviderEnv(settings, provider)
	for key := range providerEnv {
		delete(settings.Env, key)
	}
}
