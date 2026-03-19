package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/stuckinforloop/llmconf/internal/config"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage llmconf configuration",
	Long: `View and manage llmconf internal configuration.

Subcommands:
  config view    - View internal configuration
  config path    - Show config file paths
  config reset   - Reset configuration (dangerous)`,
}

// configViewCmd shows internal configuration
var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View internal configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgManager, err := config.NewConfigManager()
		if err != nil {
			return fmt.Errorf("failed to initialize config manager: %w", err)
		}

		cfg, err := cfgManager.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		fmt.Println()
		fmt.Println("llmconf Internal Configuration:")
		fmt.Println()
		fmt.Printf("Version: %s\n", cfg.Version)
		fmt.Println()

		if len(cfg.Providers) > 0 {
			fmt.Println("Configured Providers:")
			for name, state := range cfg.Providers {
				fmt.Printf("  %s:\n", name)
				fmt.Printf("    Configured: %v\n", state.Configured)
				fmt.Printf("    Scope: %s\n", state.Scope)
				if state.AuthMethod != "" {
					fmt.Printf("    Auth Method: %s\n", state.AuthMethod)
				}
				if len(state.Credentials) > 0 {
					fmt.Printf("    Credentials: %v\n", state.Credentials)
				}
				if len(state.Models) > 0 {
					fmt.Printf("    Models: %v\n", state.Models)
				}
			}
			fmt.Println()
		}

		if len(cfg.Active) > 0 {
			fmt.Println("Active Providers:")
			for tool, active := range cfg.Active {
				fmt.Printf("  %s: %s (%s)\n", tool, active.Provider, active.Scope)
			}
			fmt.Println()
		}

		return nil
	},
}

// configPathCmd shows config file paths
var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file paths",
	RunE: func(cmd *cobra.Command, args []string) error {
		scopeManager, err := config.NewScopeManager()
		if err != nil {
			return fmt.Errorf("failed to initialize scope manager: %w", err)
		}

		fmt.Println()
		fmt.Println("Configuration File Paths:")
		fmt.Println()
		fmt.Println("Internal Config:")
		home, _ := os.UserHomeDir()
		fmt.Printf("  %s\n", filepath.Join(home, ".config", "llmconf", "config.json"))
		fmt.Println()

		fmt.Println("Claude Code Settings:")
		fmt.Printf("  Global:  %s\n", scopeManager.GetSettingsPath(config.ScopeGlobal, "claude-code"))
		fmt.Printf("  Project:  %s\n", scopeManager.GetSettingsPath(config.ScopeProject, "claude-code"))
		fmt.Println()

		return nil
	},
}

func init() {
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configPathCmd)
}
