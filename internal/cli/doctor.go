package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/stuckinforloop/llmconf/internal/config"
	"github.com/stuckinforloop/llmconf/internal/providers"
)

var autoFix bool

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and fix configuration issues",
	Long: `Run diagnostics on your llmconf configuration.

Detects and offers to fix:
- Missing credentials
- Conflicting settings
- Incomplete configurations
- Security issues

Examples:
  llmconf doctor         # Run diagnostics
  llmconf doctor --fix   # Auto-fix issues where possible`,
	RunE: runDoctor,
}

func init() {
	doctorCmd.Flags().BoolVar(&autoFix, "fix", false, "Auto-fix issues where possible")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	// Initialize managers
	cfgManager, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	scopeManager, err := config.NewScopeManager()
	if err != nil {
		return fmt.Errorf("failed to initialize scope manager: %w", err)
	}

	// Load current configuration
	cfg, err := cfgManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	fmt.Println()
	color.Cyan("Running diagnostics...")
	fmt.Println()

	issues := []Issue{}
	fixed := []string{}

	// Check 1: Claude Code settings.json validity
	for _, scope := range []config.Scope{config.ScopeGlobal, config.ScopeProject} {
		if scopeManager.HasSettings(scope) {
			_, err := scopeManager.LoadSettings(scope)
			if err != nil {
				issues = append(issues, Issue{
					Type:        "invalid_settings",
					Severity:    "error",
					Description: fmt.Sprintf("%s settings.json is invalid: %v", scope, err),
					AutoFixable: false,
				})
			}
		}
	}

	if len(issues) == 0 {
		printSuccess("Claude Code settings.json is valid")
	}

	// Check 2: Provider configuration completeness
	registry := providers.NewRegistry()
	for _, provider := range registry.List() {
		state, hasState := cfgManager.GetProviderState(cfg, provider.Name())
		if hasState && state.Configured {
			if provider.SupportsModelPinning() && len(state.Models) == 0 {
				issues = append(issues, Issue{
					Type:        "missing_model_pinning",
					Severity:    "warning",
					Description: fmt.Sprintf("%s is missing model pinning", provider.DisplayName()),
					AutoFixable: true,
					Provider:    provider.Name(),
				})
			} else {
				printSuccess(fmt.Sprintf("%s configuration is complete", provider.DisplayName()))
			}
		}
	}

	// Check 3: Scope conflicts
	conflicts, err := scopeManager.DetectConflicts()
	if err != nil {
		printWarning(fmt.Sprintf("Could not detect conflicts: %v", err))
	}

	for _, conflict := range conflicts {
		issues = append(issues, Issue{
			Type:        conflict.Type,
			Severity:    conflict.Severity,
			Description: conflict.Description,
			AutoFixable: true,
			Scope:       string(conflict.Scope),
		})
	}

	// Check 4: Active provider exists
	active, hasActive := cfgManager.GetActiveProvider(cfg, "claude-code")
	if hasActive {
		state, hasState := cfgManager.GetProviderState(cfg, active.Provider)
		if !hasState || !state.Configured {
			issues = append(issues, Issue{
				Type:        "invalid_active_provider",
				Severity:    "error",
				Description: fmt.Sprintf("Active provider '%s' is not configured", active.Provider),
				AutoFixable: false,
			})
		}
	}

	// Report findings
	fmt.Println()
	if len(issues) == 0 {
		printSuccess("All checks passed! No issues found.")
		fmt.Println()
		return nil
	}

	// Categorize issues
	errors := filterIssuesBySeverity(issues, "error")
	warnings := filterIssuesBySeverity(issues, "warning")
	infos := filterIssuesBySeverity(issues, "info")

	if len(errors) > 0 {
		color.Red("Found %d error(s):", len(errors))
		for _, issue := range errors {
			fmt.Printf("  ✗ %s\n", issue.Description)
		}
		fmt.Println()
	}

	if len(warnings) > 0 {
		color.Yellow("Found %d warning(s):", len(warnings))
		for i, issue := range warnings {
			fmt.Printf("  %d. %s\n", i+1, issue.Description)
			if issue.AutoFixable {
				fmt.Printf("     → Can be auto-fixed\n")
			}
		}
		fmt.Println()
	}

	if len(infos) > 0 {
		color.Cyan("Found %d informational issue(s):", len(infos))
		for _, issue := range infos {
			fmt.Printf("  • %s\n", issue.Description)
		}
		fmt.Println()
	}

	// Auto-fix or prompt
	autoFixableIssues := filterAutoFixable(issues)
	if len(autoFixableIssues) > 0 {
		if autoFix {
			for _, issue := range autoFixableIssues {
				if fixIssue(issue, scopeManager) {
					fixed = append(fixed, issue.Description)
				}
			}
		} else if !nonInteractive {
			var fix bool
			fmt.Print("Would you like to fix the auto-fixable issues? (Y/n): ")
			var response string
			fmt.Scanln(&response)
			fix = response == "" || response == "y" || response == "Y"

			if fix {
				for _, issue := range autoFixableIssues {
					if fixIssue(issue, scopeManager) {
						fixed = append(fixed, issue.Description)
					}
				}
			}
		}
	}

	// Report fixed issues
	if len(fixed) > 0 {
		fmt.Println()
		printSuccess(fmt.Sprintf("Fixed %d issue(s):", len(fixed)))
		for _, desc := range fixed {
			fmt.Printf("  ✓ %s\n", desc)
		}
	}

	fmt.Println()
	fmt.Println("Run 'llmconf status' to verify your configuration")
	fmt.Println()

	return nil
}

// Issue represents a detected issue
type Issue struct {
	Type        string
	Severity    string
	Description string
	AutoFixable bool
	Provider    string
	Scope       string
}

func filterIssuesBySeverity(issues []Issue, severity string) []Issue {
	var result []Issue
	for _, issue := range issues {
		if issue.Severity == severity {
			result = append(result, issue)
		}
	}
	return result
}

func filterAutoFixable(issues []Issue) []Issue {
	var result []Issue
	for _, issue := range issues {
		if issue.AutoFixable {
			result = append(result, issue)
		}
	}
	return result
}

func fixIssue(issue Issue, scopeManager *config.ScopeManager) bool {
	switch issue.Type {
	case "api_key_conflict":
		// Remove ANTHROPIC_API_KEY from global
		if err := scopeManager.RemoveEnvVar(config.ScopeGlobal, "ANTHROPIC_API_KEY"); err != nil {
			return false
		}
		return true
	case "missing_model_pinning":
		// User needs to run 'llmconf set <provider>' to add model pinning
		return false // Can't auto-fix, needs user input
	default:
		return false
	}
}
