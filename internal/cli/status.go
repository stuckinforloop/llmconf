package cli

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/stuckinforloop/llmconf/internal/config"
	"github.com/stuckinforloop/llmconf/internal/providers"
	"github.com/stuckinforloop/llmconf/internal/secrets"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status [tool]",
	Short: "Show current configuration status",
	Long: `Display the current configuration for Claude Code (or other tools).

Shows:
- Active provider and scope
- Environment variables
- Stored credentials status
- Recommendations and warnings`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	tool := "claude-code"
	if len(args) > 0 {
		tool = args[0]
	}

	// Initialize managers
	cfgManager, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	scopeManager, err := config.NewScopeManager()
	if err != nil {
		return fmt.Errorf("failed to initialize scope manager: %w", err)
	}

	// Create secret store
	backend := secrets.NewKeychainStore()
	secretStore := secrets.NewStore(backend)

	// Load current configuration
	cfg, err := cfgManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get active provider
	active, hasActive := cfgManager.GetActiveProvider(cfg, tool)

	// Get current scope
	scope := config.Scope(getScope())

	// Load settings
	settings, err := scopeManager.LoadSettings(scope)
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	fmt.Println()

	// Header
	if tool == "claude-code" {
		color.Cyan("┌─ Claude Code Configuration ")
	} else {
		color.Cyan("┌─ %s Configuration ", tool)
	}
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println("│")

	if !hasActive {
		fmt.Println("│  " + color.YellowString("No provider configured"))
		fmt.Println("│")
		fmt.Println("│  Run 'llmconf init' to get started")
		fmt.Println("│")
		fmt.Println("└" + strings.Repeat("─", 60))
		fmt.Println()
		return nil
	}

	// Active provider info
	fmt.Printf("│  Active Provider: %s\n", color.GreenString(active.Provider))
	fmt.Printf("│  Scope:           %s (%s)\n", active.Scope, scopeManager.GetSettingsPath(config.Scope(active.Scope), tool))
	fmt.Println("│")

	// Environment variables
	envVars := config.GetProviderEnv(settings, active.Provider)
	if len(envVars) > 0 {
		fmt.Println("│  Environment Variables:")
		for key, value := range envVars {
			// Mask sensitive values
			displayValue := value
			if isSensitive(key) && value != "" {
				displayValue = maskValue(value)
			}
			fmt.Printf("│    %s=%s\n", key, displayValue)
		}
		fmt.Println("│")
	}

	// Credentials status
	state, hasState := cfgManager.GetProviderState(cfg, active.Provider)
	if hasState && len(state.Credentials) > 0 {
		fmt.Print("│  Credentials:      ")
		allPresent := true
		for _, credName := range state.Credentials {
			_, err := secretStore.GetCredential(active.Provider, credName)
			if err != nil {
				allPresent = false
				break
			}
		}
		if allPresent {
			fmt.Println(color.GreenString("stored in keychain ✓"))
		} else {
			fmt.Println(color.YellowString("some missing ⚠"))
		}
		fmt.Println("│")
	}

	// Model pinning status
	provider, _ := providers.NewRegistry().Get(active.Provider)
	if provider != nil && provider.SupportsModelPinning() {
		if hasState && len(state.Models) > 0 {
			fmt.Println("│  Model Pinning:    " + color.GreenString("configured ✓"))
			for modelType, modelID := range state.Models {
				fmt.Printf("│    %s: %s\n", modelType, modelID)
			}
		} else {
			fmt.Println("│  Model Pinning:    " + color.YellowString("not configured ⚠"))
			fmt.Println("│    Run 'llmconf set " + active.Provider + "' to add model pinning")
		}
		fmt.Println("│")
	}

	// Detect conflicts
	conflicts, _ := scopeManager.DetectConflicts()
	if len(conflicts) > 0 {
		color.Yellow("├─ Warnings ")
		fmt.Println(strings.Repeat("─", 50))
		fmt.Println("│")
		for i, conflict := range conflicts {
			fmt.Printf("│  %d. %s\n", i+1, conflict.Description)
			if conflict.SuggestedFix != "" {
				fmt.Printf("│     → %s\n", conflict.SuggestedFix)
			}
			fmt.Println("│")
		}
	} else {
		color.Green("├─ Recommendations ")
		fmt.Println(strings.Repeat("─", 50))
		fmt.Println("│")
		fmt.Println("│  ✓ All checks passed!")
		fmt.Println("│")
	}

	fmt.Println("└" + strings.Repeat("─", 60))

	// Show other configured providers
	hasOthers := false
	for name, state := range cfg.Providers {
		if state.Configured && name != active.Provider {
			if !hasOthers {
				fmt.Println()
				fmt.Println("Other Configured Providers:")
				hasOthers = true
			}
			fmt.Printf("  • %s (%s) - run 'llmconf set %s' to switch\n", name, state.Scope, name)
		}
	}

	fmt.Println()
	fmt.Println("Run 'llmconf doctor' for detailed diagnostics")
	fmt.Println()

	return nil
}

func isSensitive(key string) bool {
	sensitiveKeys := []string{
		"API_KEY",
		"SECRET",
		"TOKEN",
		"PASSWORD",
		"CREDENTIAL",
	}

	upperKey := strings.ToUpper(key)
	for _, pattern := range sensitiveKeys {
		if strings.Contains(upperKey, pattern) {
			return true
		}
	}
	return false
}

func maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

// printSettingsTable prints settings in a table format
func printSettingsTable(settings *config.ClaudeSettings) {
	if len(settings.Env) == 0 {
		return
	}

	fmt.Printf("%-30s %s\n", "Variable", "Value")
	fmt.Println(strings.Repeat("-", 70))
	for key, value := range settings.Env {
		displayValue := value
		if isSensitive(key) {
			displayValue = maskValue(value)
		}
		fmt.Printf("%-30s %s\n", key, displayValue)
	}
}
