package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	globalFlag    bool
	projectFlag   bool
	nonInteractive bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "llmconf",
	Short: "LLM Provider Configuration CLI",
	Long: `llmconf is a CLI tool for managing LLM provider configurations
across multiple agentic coding tools, starting with Claude Code.

Supports both global (user) and project scopes with secure credential storage.

Get started:
  llmconf init      # First-time setup
  llmconf list      # Show all providers
  llmconf status    # Check current configuration

For more information, visit: https://github.com/stuckinforloop/llmconf`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().BoolVarP(&globalFlag, "global", "g", false, "Use global scope (user-wide)")
	rootCmd.PersistentFlags().BoolVarP(&projectFlag, "project", "p", false, "Use project scope (current directory)")
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "Fail if input needed instead of prompting")

	// Add all subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(rotateCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

// getScope returns the scope based on flags or auto-detection
func getScope() string {
	if globalFlag {
		return "global"
	}
	if projectFlag {
		return "project"
	}
	// Auto-detect: use project scope if in git repo, otherwise global
	if isGitRepo() {
		return "project"
	}
	return "global"
}

// isGitRepo checks if the current directory is in a git repository
func isGitRepo() bool {
	_, err := os.Stat(".git")
	if err == nil {
		return true
	}
	// Check parent directories
	cwd, _ := os.Getwd()
	for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
	}
	return false
}

// printError prints an error message to stderr
func printError(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
}

// printWarning prints a warning message
func printWarning(msg string) {
	fmt.Printf("⚠️  %s\n", msg)
}

// printSuccess prints a success message
func printSuccess(msg string) {
	fmt.Printf("✓ %s\n", msg)
}
