package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stuckinforloop/llmconf/internal/config"
	"github.com/stuckinforloop/llmconf/internal/providers"
	"github.com/stuckinforloop/llmconf/internal/secrets"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove <provider>",
	Short: "Remove a provider configuration",
	Long: `Remove the specified provider configuration.

Warns if the provider is currently active and offers alternatives.
Removes credentials from secret store and clears settings.

Examples:
  llmconf remove bedrock    # Remove Bedrock configuration
  llmconf remove vertex     # Remove Vertex configuration`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func runRemove(cmd *cobra.Command, args []string) error {
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

	// Check if provider is configured
	state, hasState := cfgManager.GetProviderState(cfg, providerName)
	if !hasState || !state.Configured {
		return fmt.Errorf("%s is not configured", provider.DisplayName())
	}

	fmt.Println()

	// Check if provider is currently active
	active, hasActive := cfgManager.GetActiveProvider(cfg, "claude-code")
	if hasActive && active.Provider == providerName {
		printWarning(fmt.Sprintf("%s is currently the active provider!", provider.DisplayName()))
		fmt.Println()

		if nonInteractive {
			return fmt.Errorf("cannot remove active provider, switch to another provider first")
		}

		// Find alternative providers
		alternatives := []string{}
		for name, otherState := range cfg.Providers {
			if name != providerName && otherState.Configured {
				alternatives = append(alternatives, name)
			}
		}

		if len(alternatives) > 0 {
			fmt.Println("Alternative providers available:")
			for _, alt := range alternatives {
				fmt.Printf("  • %s\n", alt)
			}
			fmt.Println()

			var switchNow bool
			fmt.Print("Switch to another provider before removing? (Y/n): ")
			var response string
			fmt.Scanln(&response)
			switchNow = response == "" || response == "y" || response == "Y"

			if switchNow {
				// Switch to first alternative
				if err := runSet(cmd, []string{alternatives[0]}); err != nil {
					return err
				}
			}
		} else {
			fmt.Println("No other providers configured. You'll need to set up a new provider after removal.")
			fmt.Println()
		}
	}

	// Confirm removal
	if !nonInteractive {
		var confirm bool
		fmt.Printf("Are you sure you want to remove %s configuration? (y/N): ", provider.DisplayName())
		var response string
		fmt.Scanln(&response)
		confirm = response == "y" || response == "Y"

		if !confirm {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Remove credentials from secret store
	if err := secretStore.DeleteProvider(providerName); err != nil {
		printWarning(fmt.Sprintf("Failed to delete some credentials: %v", err))
	}

	// Clear settings for this provider
	scope := config.Scope(state.Scope)
	settings, err := scopeManager.LoadSettings(scope)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	loader := config.SettingsLoader{}
	loader.ClearProviderEnv(settings, providerName)

	if err := scopeManager.SaveSettings(scope, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	// Update configuration
	state.Configured = false
	state.Credentials = nil
	state.Models = nil
	cfgManager.SetProviderState(cfg, state)

	// Remove from active if it was active
	if hasActive && active.Provider == providerName {
		delete(cfg.Active, "claude-code")
	}

	if err := cfgManager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println()
	printSuccess(fmt.Sprintf("%s configuration removed", provider.DisplayName()))
	fmt.Println()

	return nil
}
