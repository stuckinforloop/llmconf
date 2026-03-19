package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stuckinforloop/llmconf/internal/config"
	"github.com/stuckinforloop/llmconf/internal/providers"
	"github.com/stuckinforloop/llmconf/internal/secrets"
)

var (
	skipValidation bool
	skipModelPinningSet bool
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set <provider>",
	Short: "Switch to a different LLM provider",
	Long: `Switch to the specified provider with automatic configuration detection.

This command intelligently handles:
- Provider not configured → Auto-configures
- Provider configured but incomplete → Completes setup
- Scope mismatch → Offers to copy or move config
- Conflict detection → Guided resolution

Examples:
  llmconf set bedrock           # Switch to Bedrock (auto-detect scope)
  llmconf set bedrock --project # Use project scope
  llmconf set vertex --global   # Use global scope
  llmconf set fireworks         # Configure Fireworks if needed`,
	Args: cobra.ExactArgs(1),
	RunE: runSet,
}

func init() {
	setCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "Skip credential validation")
	setCmd.Flags().BoolVar(&skipModelPinningSet, "skip-model-pinning", false, "Skip model pinning")
}

func runSet(cmd *cobra.Command, args []string) error {
	providerName := args[0]

	// Initialize managers
	cfgManager, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	scopeManager, err := config.NewScopeManager()
	if err != nil {
		return fmt.Errorf("failed to initialize scope manager: %w", err)
	}

	backend := secrets.NewKeychainStore()
	secretStore := secrets.NewStore(backend)

	// Load current configuration
	cfg, err := cfgManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get provider
	registry := providers.NewRegistry()
	provider, ok := registry.Get(providerName)
	if !ok {
		return fmt.Errorf("unknown provider: %s", providerName)
	}

	// Get target scope
	targetScope := getScope()
	scope := config.Scope(targetScope)

	// Check if provider is already configured
	state, hasState := cfgManager.GetProviderState(cfg, providerName)
	needsConfiguration := !hasState || !state.Configured

	// Check for scope mismatch
	scopeMismatch := hasState && state.Scope != "" && state.Scope != targetScope

	// Check for conflicts
	conflicts, _ := scopeManager.DetectConflicts()

	// Check for incomplete config (missing model pinning)
	incompleteConfig := hasState && provider.SupportsModelPinning() && len(state.Models) == 0 && !skipModelPinningSet

	fmt.Println()

	// Scenario 1: Provider not configured → Auto-configure
	if needsConfiguration {
		printWarning(fmt.Sprintf("%s is not configured yet.", provider.DisplayName()))
		fmt.Println()
		fmt.Printf("Let's configure it now:\n\n")

		// Configure the provider
		providerConfig, err := configureProviderInteractive(provider)
		if err != nil {
			return err
		}

		// Store credentials
		if err := secretStore.StoreConfig(provider.Name(), providerConfig.Credentials); err != nil {
			return fmt.Errorf("failed to store credentials: %w", err)
		}

		// Handle model pinning
		models := make(map[string]string)
		if provider.SupportsModelPinning() && !skipModelPinningSet {
			printWarning("Model pinning is strongly recommended for " + provider.DisplayName())
			fmt.Println()

			if !nonInteractive {
				models = configureModelPinning(provider)
			}
		}
		providerConfig.Models = models

		// Generate and save env vars
		if err := applyProviderConfig(scopeManager, scope, provider, providerConfig, providerName); err != nil {
			return err
		}

		// Update state
		state = config.ProviderState{
			Name:        provider.Name(),
			Configured:  true,
			Scope:       targetScope,
			Credentials: getCredentialNames(provider),
			Models:      models,
			AuthMethod:  providerConfig.AuthMethod,
		}
		cfgManager.SetProviderState(cfg, state)

		fmt.Println()
		printSuccess(fmt.Sprintf("%s configured successfully!", provider.DisplayName()))

	} else if scopeMismatch {
		// Scenario 2: Scope mismatch
		printWarning(fmt.Sprintf("%s is already configured in %s scope.", provider.DisplayName(), state.Scope))
		fmt.Println()

		if nonInteractive {
			// In non-interactive mode, copy config to target scope
			if err := copyProviderConfig(secretStore, cfgManager, cfg, provider, state.Scope, targetScope); err != nil {
				return err
			}
		} else {
			// Ask user what to do
			fmt.Println("Would you like to:")
			fmt.Println("  1. Copy config to target scope (recommended)")
			fmt.Println("  2. Move config from current scope to target scope")
			fmt.Println("  3. Cancel")
			fmt.Println()
			// For now, default to copy
			if err := copyProviderConfig(secretStore, cfgManager, cfg, provider, state.Scope, targetScope); err != nil {
				return err
			}
		}

	} else if incompleteConfig {
		// Scenario 3: Incomplete config (missing model pinning)
		printWarning(fmt.Sprintf("%s is configured but missing model pinning.", provider.DisplayName()))
		fmt.Println()
		fmt.Println("Model pinning is strongly recommended to prevent breakage")
		fmt.Println("when Anthropic releases new models.")
		fmt.Println()

		if !nonInteractive {
			var addPinning bool
			// In a real implementation, this would use huh.Confirm
			addPinning = true // Default to yes

			if addPinning {
				models := configureModelPinning(provider)
				state.Models = models
				cfgManager.SetProviderState(cfg, state)

				// Update settings with models
				providerConfig := &providers.ProviderConfig{
					Credentials: make(map[string]string),
					Models:      models,
				}
				if err := applyProviderConfig(scopeManager, scope, provider, providerConfig, providerName); err != nil {
					return err
				}

				printSuccess("Model pinning added")
			}
		}

	} else {
		// Scenario 4: Clean switch
		// Just need to activate this provider
	}

	// Handle conflicts
	for _, conflict := range conflicts {
		if conflict.Type == "api_key_conflict" && provider.Name() != "anthropic" {
			printWarning(fmt.Sprintf("Conflict detected: %s", conflict.Description))
			fmt.Println()
			// In a real implementation, offer to fix
		}
	}

	// Activate the provider
	cfgManager.SetActiveProvider(cfg, "claude-code", providerName, targetScope)

	// Save configuration
	if err := cfgManager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Update settings - merge provider config with existing settings
	settings, _ := scopeManager.LoadSettings(scope)

	// Clear only provider-specific env vars to avoid contamination
	// Preserve user-defined env vars that aren't managed by llmconf
	clearProviderEnvVars(settings.Env)

	// Generate non-sensitive env vars only (credentials are fetched via apiKeyHelper)
	providerConfig := &providers.ProviderConfig{
		Credentials: make(map[string]string), // Empty - credentials come from keychain
		Models:      state.Models,
	}

	env, err := provider.GenerateEnv(*providerConfig)
	if err != nil {
		return fmt.Errorf("failed to generate environment: %w", err)
	}

	// Set non-sensitive env vars
	for key, value := range env {
		settings.Env[key] = value
	}

	// Set apiKeyHelper for providers that need dynamic credential fetching
	// Fireworks and other providers that use ANTHROPIC_API_KEY need this
	if providerName == "fireworks" || providerName == "anthropic" || providerName == "litellm" {
		settings.APIKeyHelper = fmt.Sprintf("llmconf credential get %s ANTHROPIC_API_KEY", providerName)
	}

	// Bedrock and Vertex use different auth mechanisms
	if providerName == "bedrock" && state.AuthMethod == "sso" {
		// For SSO, we might need awsAuthRefresh
		// This would be set during configuration
	}

	if err := scopeManager.SaveSettings(scope, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	// Success message
	fmt.Println()
	printSuccess(fmt.Sprintf("Switched to %s (%s scope)", provider.DisplayName(), targetScope))
	fmt.Println()

	// Show previous provider if different
	if hasState {
		prevActive, _ := cfgManager.GetActiveProvider(cfg, "claude-code")
		if prevActive.Provider != "" && prevActive.Provider != providerName {
			fmt.Printf("  Previous: %s\n", prevActive.Provider)
		}
	}
	fmt.Printf("  Current:  %s\n", provider.DisplayName())
	fmt.Printf("  Scope:    %s\n", targetScope)

	if len(state.Models) > 0 {
		fmt.Printf("  Models:   ")
		first := true
		for modelType, modelID := range state.Models {
			if !first {
				fmt.Printf(", ")
			}
			fmt.Printf("%s=%s", modelType, modelID)
			first = false
		}
		fmt.Println()
	}

	fmt.Println()

	return nil
}

// getProviderEnvVars returns a list of env vars managed by llmconf
func getProviderEnvVars() []string {
	return []string{
		// Anthropic/Fireworks/LiteLLM
		"ANTHROPIC_API_KEY",
		"ANTHROPIC_BASE_URL",
		"ANTHROPIC_MODEL",
		"ANTHROPIC_SMALL_FAST_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"CLAUDE_CODE_USE_BEDROCK",
		"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS",
		// AWS/Bedrock
		"AWS_REGION",
		"AWS_PROFILE",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		// Vertex
		"CLOUD_ML_REGION",
		"ANTHROPIC_VERTEX_PROJECT_ID",
		// Foundry
		"AZURE_FOUNDRY_ENDPOINT",
		"AZURE_FOUNDRY_DEPLOYMENT_NAME",
		"AZURE_FOUNDRY_API_VERSION",
		// LiteLLM
		"LITELLM_PROXY_API_KEY",
		"LITELLM_PROXY_BASE_URL",
	}
}

// clearProviderEnvVars removes only provider-managed env vars
// preserving user-defined custom env vars
func clearProviderEnvVars(env map[string]string) {
	providerVars := getProviderEnvVars()
	for _, key := range providerVars {
		delete(env, key)
	}
}

// applyProviderConfig generates env vars and saves settings
func applyProviderConfig(scopeManager *config.ScopeManager, scope config.Scope, provider providers.Provider, cfg *providers.ProviderConfig, providerName string) error {
	settings, err := scopeManager.LoadSettings(scope)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Clear only provider-specific env vars, preserve user-defined ones
	clearProviderEnvVars(settings.Env)

	// Generate non-sensitive env vars only
	// Remove credentials from config before generating env
	secureConfig := &providers.ProviderConfig{
		Credentials: make(map[string]string), // Empty - credentials come via apiKeyHelper
		Models:      cfg.Models,
		ExtraEnv:    cfg.ExtraEnv,
	}

	env, err := provider.GenerateEnv(*secureConfig)
	if err != nil {
		return fmt.Errorf("failed to generate environment: %w", err)
	}

	// Set env vars (non-sensitive only)
	for key, value := range env {
		settings.Env[key] = value
	}

	// Set apiKeyHelper for providers that use API keys
	if providerName == "fireworks" || providerName == "anthropic" || providerName == "litellm" {
		settings.APIKeyHelper = fmt.Sprintf("llmconf credential get %s ANTHROPIC_API_KEY", providerName)
	}

	if err := scopeManager.SaveSettings(scope, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	return nil
}

// copyProviderConfig copies provider configuration from one scope to another
func copyProviderConfig(secretStore *secrets.Store, cfgManager *config.ConfigManager, cfg *config.LLMConfConfig, provider providers.Provider, fromScope, toScope string) error {
	// Load credentials from secret store (this validates they exist)
	_, err := secretStore.LoadConfig(provider.Name(), getCredentialNames(provider))
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	// Copy to target scope in secret store (they're stored globally, so this is a no-op)
	// But we mark the state as available in the new scope
	state, _ := cfgManager.GetProviderState(cfg, provider.Name())
	state.Scope = toScope
	cfgManager.SetProviderState(cfg, state)

	return nil
}
