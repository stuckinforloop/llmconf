package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// CurrentVersion is the current configuration format version
	CurrentVersion = "1.0.0"
	// ConfigFileName is the name of the llmconf config file
	ConfigFileName = "config.json"
)

// LLMConfConfig is the internal llmconf configuration
type LLMConfConfig struct {
	Version   string                    `json:"version"`
	Providers map[string]ProviderState  `json:"providers"`
	Active    map[string]ActiveProvider `json:"active"`
}

// ProviderState tracks the state of a configured provider
type ProviderState struct {
	Name         string            `json:"name"`
	Configured   bool              `json:"configured"`
	Scope        string            `json:"scope"`
	Credentials  []string          `json:"credentials"`
	Models       map[string]string `json:"models,omitempty"`
	AuthMethod   string            `json:"auth_method,omitempty"`
	LastRotated  *time.Time        `json:"last_rotated,omitempty"`
}

// ActiveProvider tracks which provider is active for a tool
type ActiveProvider struct {
	Provider string `json:"provider"`
	Scope    string `json:"scope"`
}

// ClaudeSettings represents Claude Code's settings.json structure
type ClaudeSettings struct {
	Schema               string            `json:"$schema,omitempty"`
	Env                  map[string]string `json:"env,omitempty"`
	APIKeyHelper         string            `json:"apiKeyHelper,omitempty"`
	AWSAuthRefresh       string            `json:"awsAuthRefresh,omitempty"`
	AWSCredentialExport  string            `json:"awsCredentialExport,omitempty"`
	ModelOverrides       map[string]string `json:"modelOverrides,omitempty"`
	AvailableModels      []string          `json:"availableModels,omitempty"`
}

// ConfigManager handles loading and saving configuration
type ConfigManager struct {
	configDir string
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager() (*ConfigManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "llmconf")
	return &ConfigManager{
		configDir: configDir,
	}, nil
}

// Load loads the llmconf configuration
func (cm *ConfigManager) Load() (*LLMConfConfig, error) {
	configPath := cm.configPath()

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config
		return cm.defaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Parse JSON
	var config LLMConfConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Ensure maps are initialized
	if config.Providers == nil {
		config.Providers = make(map[string]ProviderState)
	}
	if config.Active == nil {
		config.Active = make(map[string]ActiveProvider)
	}

	return &config, nil
}

// Save saves the llmconf configuration
func (cm *ConfigManager) Save(config *LLMConfConfig) error {
	// Ensure directory exists
	if err := os.MkdirAll(cm.configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Set version if not set
	if config.Version == "" {
		config.Version = CurrentVersion
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	configPath := cm.configPath()
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// GetProviderState retrieves the state of a provider
func (cm *ConfigManager) GetProviderState(config *LLMConfConfig, provider string) (ProviderState, bool) {
	state, ok := config.Providers[provider]
	return state, ok
}

// SetProviderState updates the state of a provider
func (cm *ConfigManager) SetProviderState(config *LLMConfConfig, state ProviderState) {
	if config.Providers == nil {
		config.Providers = make(map[string]ProviderState)
	}
	config.Providers[state.Name] = state
}

// GetActiveProvider retrieves the active provider for a tool
func (cm *ConfigManager) GetActiveProvider(config *LLMConfConfig, tool string) (ActiveProvider, bool) {
	active, ok := config.Active[tool]
	return active, ok
}

// SetActiveProvider sets the active provider for a tool
func (cm *ConfigManager) SetActiveProvider(config *LLMConfConfig, tool string, provider string, scope string) {
	if config.Active == nil {
		config.Active = make(map[string]ActiveProvider)
	}
	config.Active[tool] = ActiveProvider{
		Provider: provider,
		Scope:    scope,
	}
}

// IsConfigured checks if a provider is configured
func (cm *ConfigManager) IsConfigured(config *LLMConfConfig, provider string) bool {
	state, ok := config.Providers[provider]
	return ok && state.Configured
}

// configPath returns the path to the config file
func (cm *ConfigManager) configPath() string {
	return filepath.Join(cm.configDir, ConfigFileName)
}

// defaultConfig returns a default empty configuration
func (cm *ConfigManager) defaultConfig() *LLMConfConfig {
	return &LLMConfConfig{
		Version:   CurrentVersion,
		Providers: make(map[string]ProviderState),
		Active:    make(map[string]ActiveProvider),
	}
}

// ProviderNames returns a list of known provider names
func ProviderNames() []string {
	return []string{
		"anthropic",
		"bedrock",
		"vertex",
		"foundry",
		"fireworks",
		"litellm",
	}
}
