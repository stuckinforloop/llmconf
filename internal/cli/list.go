package cli

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/stuckinforloop/llmconf/internal/config"
	"github.com/stuckinforloop/llmconf/internal/providers"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured providers",
	Long: `Display all providers and their current status.

Shows:
- Configured providers with their scope
- Active provider for Claude Code
- Missing configuration that needs attention
- Quick actions to complete setup`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	// Initialize managers
	cfgManager, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	// Load current configuration
	cfg, err := cfgManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get active provider
	active, hasActive := cfgManager.GetActiveProvider(cfg, "claude-code")

	// Create provider registry
	registry := providers.NewRegistry()

	fmt.Println()
	color.Cyan("Configured Providers:")
	fmt.Println()

	// Create simple table format
	headerFmt := "%-15s %-12s %-12s %s\n"
	fmt.Printf(headerFmt, "Provider", "Status", "Scope", "Details")
	fmt.Println(strings.Repeat("-", 70))

	// Build table rows
	for _, provider := range registry.List() {
		name := provider.Name()
		state, hasState := cfgManager.GetProviderState(cfg, name)

		var status, scope, details string

		if hasState && state.Configured {
			if hasActive && active.Provider == name {
				status = color.GreenString("✓ active")
			} else {
				status = color.GreenString("✓ ready")
			}
			scope = state.Scope

			// Build details
			if state.AuthMethod != "" {
				details = fmt.Sprintf("%s auth", state.AuthMethod)
			}
			if provider.SupportsModelPinning() {
				if len(state.Models) > 0 {
					if details != "" {
						details += ", "
					}
					details += "models pinned"
				} else {
					if details != "" {
						details += ", "
					}
					details += color.YellowString("missing model pinning ⚠")
				}
			}
		} else {
			status = color.RedString("✗ not configured")
			scope = "-"
			details = "Run 'llmconf set " + name + "' to configure"
		}

		fmt.Printf("%-15s %-12s %-12s %s\n", provider.DisplayName(), status, scope, details)
	}

	// Show active provider info
	fmt.Println()
	if hasActive {
		fmt.Printf("Active for Claude Code: %s (%s scope)\n",
			color.GreenString(active.Provider),
			active.Scope)
	} else {
		fmt.Println("Active for Claude Code: " + color.YellowString("none configured"))
	}

	// Show quick actions
	fmt.Println()
	color.Cyan("Quick actions:")
	for _, provider := range registry.List() {
		name := provider.Name()
		state, hasState := cfgManager.GetProviderState(cfg, name)

		if hasState && state.Configured {
			if !hasActive || active.Provider != name {
				fmt.Printf("  llmconf set %-12s - Switch to %s\n", name, provider.DisplayName())
			}
			if provider.SupportsModelPinning() && len(state.Models) == 0 {
				fmt.Printf("  llmconf set %-12s - Complete configuration (add model pinning)\n", name+" --fix")
			}
		} else {
			fmt.Printf("  llmconf set %-12s - Configure %s\n", name, provider.DisplayName())
		}
	}
	fmt.Println()

	return nil
}
