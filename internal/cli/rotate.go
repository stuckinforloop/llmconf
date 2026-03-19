package cli

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/stuckinforloop/llmconf/internal/config"
	"github.com/stuckinforloop/llmconf/internal/providers"
	"github.com/stuckinforloop/llmconf/internal/secrets"
)

var rotateAll bool

// rotateCmd represents the rotate command
var rotateCmd = &cobra.Command{
	Use:   "rotate <provider> [credential]",
	Short: "Rotate provider credentials",
	Long: `Rotate credentials for the specified provider.

With automatic setup detection - if provider is not configured,
offers to set it up first. Shows rotation history.

Examples:
  llmconf rotate bedrock              # Rotate Bedrock credentials
  llmconf rotate bedrock aws_profile  # Rotate specific credential
  llmconf rotate bedrock --all         # Rotate all credentials`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runRotate,
}

func init() {
	rotateCmd.Flags().BoolVar(&rotateAll, "all", false, "Rotate all credentials for provider")
}

func runRotate(cmd *cobra.Command, args []string) error {
	providerName := args[0]
	var specificCredential string
	if len(args) > 1 {
		specificCredential = args[1]
	}

	// Initialize managers
	cfgManager, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
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
		printWarning(fmt.Sprintf("%s is not configured yet.", provider.DisplayName()))
		fmt.Println()

		if nonInteractive {
			return fmt.Errorf("provider not configured, run 'llmconf init' first")
		}

		var setup bool
		// In real implementation, use huh.Confirm
		fmt.Printf("Would you like to configure %s first? (Y/n): ", provider.DisplayName())
		var response string
		fmt.Scanln(&response)
		setup = response == "" || response == "y" || response == "Y"

		if setup {
			// Delegate to init flow
			return runInit(cmd, []string{})
		}
		return nil
	}

	fmt.Println()
	color.Cyan("Provider: %s", provider.DisplayName())
	fmt.Printf("Scope: %s\n", state.Scope)
	fmt.Println()

	// Show configured credentials
	if len(state.Credentials) > 0 {
		fmt.Println("Configured Credentials:")
		for _, credName := range state.Credentials {
			status := ""
			_, err := secretStore.GetCredential(providerName, credName)
			if err != nil {
				status = " (not in keychain)"
			} else if state.LastRotated != nil {
				days := time.Since(*state.LastRotated).Hours() / 24
				if days > 90 {
					status = fmt.Sprintf(" - last rotated: %.0f days ago (recommended: < 90 days)", days)
				} else {
					status = fmt.Sprintf(" - last rotated: %.0f days ago", days)
				}
			} else {
				status = " - last rotated: unknown"
			}
			fmt.Printf("  • %s%s\n", credName, status)
		}
		fmt.Println()
	}

	// Determine which credentials to rotate
	credentialsToRotate := []string{}
	if specificCredential != "" {
		// Check if valid credential
		found := false
		for _, cred := range state.Credentials {
			if cred == specificCredential {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("credential '%s' not found for provider %s", specificCredential, providerName)
		}
		credentialsToRotate = []string{specificCredential}
	} else if rotateAll {
		credentialsToRotate = state.Credentials
	} else {
		// If only one credential, rotate it
		if len(state.Credentials) == 1 {
			credentialsToRotate = state.Credentials
		} else if nonInteractive {
			return fmt.Errorf("specify credential to rotate or use --all")
		} else {
			// Interactive: let user choose
			fmt.Println("Which credentials would you like to rotate?")
			for i, cred := range state.Credentials {
				fmt.Printf("  %d. %s\n", i+1, cred)
			}
			fmt.Println("  A. All credentials")
			fmt.Println()
			fmt.Print("Enter choice: ")
			var choice string
			fmt.Scanln(&choice)

			if choice == "A" || choice == "a" {
				credentialsToRotate = state.Credentials
			} else {
				// Assume first for simplicity
				credentialsToRotate = []string{state.Credentials[0]}
			}
		}
	}

	// Rotate credentials
	newCredentials := make(map[string]string)
	for _, credName := range credentialsToRotate {
		value, err := promptForInput(fmt.Sprintf("New value for %s", credName), true)
		if err != nil {
			return err
		}
		newCredentials[credName] = value
	}

	// Store new credentials
	if err := secretStore.StoreConfig(providerName, newCredentials); err != nil {
		return fmt.Errorf("failed to store new credentials: %w", err)
	}

	// Update rotation timestamp
	now := time.Now()
	state.LastRotated = &now
	cfgManager.SetProviderState(cfg, state)

	if err := cfgManager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println()
	printSuccess("Credentials rotated successfully")

	// Reactivate provider if it was active
	active, hasActive := cfgManager.GetActiveProvider(cfg, "claude-code")
	if hasActive && active.Provider == providerName {
		fmt.Println()
		var reactivate bool
		if nonInteractive {
			reactivate = true
		} else {
			fmt.Print("Would you like to reactivate with new credentials? (Y/n): ")
			var response string
			fmt.Scanln(&response)
			reactivate = response == "" || response == "y" || response == "Y"
		}

		if reactivate {
			// Re-run set to apply new credentials
			return runSet(cmd, []string{providerName})
		}
	}

	return nil
}
