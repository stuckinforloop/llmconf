package validator

import (
	"fmt"
	"os"
	"strings"

	"github.com/stuckinforloop/llmconf/internal/config"
	"github.com/stuckinforloop/llmconf/internal/providers"
)

// Validator validates configurations
type Validator struct {
	registry *providers.ProviderRegistry
}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	return &Validator{
		registry: providers.NewRegistry(),
	}
}

// ValidationResult contains the result of a validation
type ValidationResult struct {
	Valid   bool
	Errors  []string
	Warnings []string
}

// ValidateProviderConfig validates a provider configuration
func (v *Validator) ValidateProviderConfig(providerName string, cfg providers.ProviderConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	provider, ok := v.registry.Get(providerName)
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Unknown provider: %s", providerName))
		return result
	}

	// Validate using provider's validation
	if err := provider.Validate(cfg); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
	}

	// Validate model pinning if applicable
	if provider.SupportsModelPinning() && len(cfg.Models) > 0 {
		for modelType, modelID := range cfg.Models {
			if err := provider.ValidateModel(modelType, modelID); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf(
					"Model '%s' for type '%s' may be invalid: %v", modelID, modelType, err))
			}
		}
	}

	return result
}

// ValidateSettings validates Claude Code settings
func (v *Validator) ValidateSettings(settings *config.ClaudeSettings) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if settings.Env == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "Environment variables not initialized")
		return result
	}

	// Detect provider
	env := settings.Env
	var activeProvider string

	if env["CLAUDE_CODE_USE_BEDROCK"] == "1" {
		activeProvider = "bedrock"
	} else if env["CLAUDE_CODE_USE_VERTEX"] == "1" {
		activeProvider = "vertex"
	} else if env["CLAUDE_CODE_USE_FOUNDRY"] == "1" {
		activeProvider = "foundry"
	} else if env["ANTHROPIC_BASE_URL"] == "https://api.fireworks.ai/inference" {
		activeProvider = "fireworks"
	} else if env["ANTHROPIC_BASE_URL"] != "" && env["ANTHROPIC_AUTH_TOKEN"] != "" {
		activeProvider = "litellm"
	} else if env["ANTHROPIC_API_KEY"] != "" {
		activeProvider = "anthropic"
	}

	if activeProvider == "" {
		result.Warnings = append(result.Warnings, "No active provider detected in settings")
		return result
	}

	provider, ok := v.registry.Get(activeProvider)
	if !ok {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Unknown active provider: %s", activeProvider))
		return result
	}

	// Check required env vars
	requiredVars := provider.RequiredEnvVars()
	for _, spec := range requiredVars {
		if spec.Name == "CLAUDE_CODE_USE_BEDROCK" || spec.Name == "CLAUDE_CODE_USE_VERTEX" || spec.Name == "CLAUDE_CODE_USE_FOUNDRY" {
			continue // Skip the enable flags
		}
		if env[spec.Name] == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Missing required variable: %s (%s)", spec.Name, spec.Description))
		}
	}

	// Check model pinning
	if provider.SupportsModelPinning() {
		hasModelPinning := env["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "" || env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != ""
		if !hasModelPinning {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Model pinning is strongly recommended for %s", provider.DisplayName()))
		}
	}

	return result
}

// ValidateEnvVarName validates an environment variable name
func ValidateEnvVarName(name string) error {
	// Basic validation for env var names
	if name == "" {
		return fmt.Errorf("environment variable name cannot be empty")
	}

	// Must start with letter or underscore
	first := name[0]
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
		return fmt.Errorf("environment variable name must start with a letter or underscore")
	}

	// Can contain letters, numbers, and underscores
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("environment variable name contains invalid character: %c", c)
		}
	}

	return nil
}

// IsSensitiveVar checks if an environment variable is sensitive
func IsSensitiveVar(name string) bool {
	sensitivePatterns := []string{
		"API_KEY",
		"SECRET",
		"TOKEN",
		"PASSWORD",
		"CREDENTIAL",
		"PRIVATE_KEY",
	}

	upperName := strings.ToUpper(name)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(upperName, pattern) {
			return true
		}
	}

	return false
}

// CheckFileExists checks if a file exists
func CheckFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// CheckDirExists checks if a directory exists
func CheckDirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
