package cli

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/stuckinforloop/llmconf/internal/config"
	"github.com/stuckinforloop/llmconf/internal/providers"
	"github.com/stuckinforloop/llmconf/internal/secrets"
)

var (
	initProvider      string
	initModelOpus     string
	initModelSonnet   string
	initModelHaiku    string
	skipModelPinning  bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize llmconf configuration",
	Long: `Initialize llmconf with intelligent first-time setup.

This command detects your current state and guides you through:
- Choosing global vs project scope
- Selecting an LLM provider
- Configuring credentials (securely stored)
- Setting up model pinning (recommended)

Examples:
  llmconf init                          # Interactive setup
  llmconf init --provider bedrock       # Pre-select provider
  llmconf init --global                 # Force global scope
  llmconf init --project                # Force project scope`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initProvider, "provider", "", "Skip provider selection")
	initCmd.Flags().StringVar(&initModelOpus, "model-opus", "", "Pin Opus model")
	initCmd.Flags().StringVar(&initModelSonnet, "model-sonnet", "", "Pin Sonnet model")
	initCmd.Flags().StringVar(&initModelHaiku, "model-haiku", "", "Pin Haiku model")
	initCmd.Flags().BoolVar(&skipModelPinning, "skip-model-pinning", false, "Skip model pinning")
}

func runInit(cmd *cobra.Command, args []string) error {
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

	// Detect scope
	scope := getScope()
	if nonInteractive {
		// In non-interactive mode, auto-detect based on context
		if !globalFlag && !projectFlag {
			scope = scopeManager.GetCurrentScope().String()
		}
	}

	// Check if this is a first-time user
	hasProviders := len(cfg.Providers) > 0

	// Welcome message
	fmt.Println()
	color.Cyan("Welcome to llmconf!")
	fmt.Println()

	if hasProviders {
		fmt.Println("You have providers configured:")
		for name, state := range cfg.Providers {
			status := "✓ configured"
			if !state.Configured {
				status = "✗ not configured"
			}
			fmt.Printf("  • %s (%s) - %s\n", name, state.Scope, status)
		}
		fmt.Println()

		if nonInteractive {
			fmt.Println("Use 'llmconf set <provider>' to switch providers")
			return nil
		}

		// Ask what to do
		var action string
		huh.NewSelect[string]().
			Title("What would you like to do?").
			Options(
				huh.NewOption("Configure a new provider", "new"),
				huh.NewOption("Switch active provider", "switch"),
				huh.NewOption("View status", "status"),
				huh.NewOption("Exit", "exit"),
			).
			Value(&action).
			Run()

		switch action {
		case "exit":
			return nil
		case "status":
			return runStatus(cmd, args)
		case "switch":
			// Continue to provider selection
		}
	} else {
		fmt.Println("Let's set up your LLM provider for Claude Code.")
		fmt.Println()
	}

	// Provider selection
	provider, err := selectProvider(initProvider, cfgManager, cfg)
	if err != nil {
		return err
	}

	if provider == nil {
		return nil
	}

	fmt.Println()
	color.Cyan("→ Selected: %s", provider.DisplayName())
	fmt.Printf("  %s\n", provider.Description())
	fmt.Printf("  Docs: %s\n", provider.DocumentationURL())
	fmt.Println()

	// Configure the provider
	providerConfig, err := configureProviderInteractive(provider)
	if err != nil {
		return err
	}

	// Store credentials securely
	if err := secretStore.StoreConfig(provider.Name(), providerConfig.Credentials); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	// Handle model pinning
	models := make(map[string]string)
	if provider.SupportsModelPinning() && !skipModelPinning {
		if !nonInteractive {
			fmt.Println()
			printWarning("Model pinning is strongly recommended for " + provider.DisplayName())
			fmt.Println("    Without pinning, Claude Code may break when Anthropic releases new models.")
			fmt.Println()

			var addPinning bool = true
			huh.NewConfirm().
				Title("Would you like to configure model pinning?").
				Value(&addPinning).
				Run()

			if addPinning {
				models = configureModelPinning(provider)
			}
		} else {
			// Non-interactive: use command line flags or defaults
			models = make(map[string]string)
			if initModelOpus != "" {
				models["opus"] = initModelOpus
			}
			if initModelSonnet != "" {
				models["sonnet"] = initModelSonnet
			}
			if initModelHaiku != "" {
				models["haiku"] = initModelHaiku
			}
		}
	}

	// Update provider config with models
	providerConfig.Models = models

	// Update Claude Code settings
	settings, err := scopeManager.LoadSettings(config.Scope(scope))
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Clear only provider-specific env vars, preserve user-defined ones
	clearProviderEnvVars(settings.Env)

	// Generate non-sensitive env vars only
	// Remove credentials - they will be fetched via apiKeyHelper
	secureConfig := &providers.ProviderConfig{
		Credentials: make(map[string]string), // Empty - credentials come via apiKeyHelper
		Models:      models,
		ExtraEnv:    providerConfig.ExtraEnv,
	}

	env, err := provider.GenerateEnv(*secureConfig)
	if err != nil {
		return fmt.Errorf("failed to generate environment: %w", err)
	}

	// Set non-sensitive env vars
	for key, value := range env {
		settings.Env[key] = value
	}

	// Set apiKeyHelper for providers that use API keys
	if provider.Name() == "fireworks" || provider.Name() == "anthropic" || provider.Name() == "litellm" {
		settings.APIKeyHelper = fmt.Sprintf("llmconf credential get %s ANTHROPIC_API_KEY", provider.Name())
	}

	// Save settings
	if err := scopeManager.SaveSettings(config.Scope(scope), settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	// Update llmconf internal config
	state := config.ProviderState{
		Name:        provider.Name(),
		Configured:  true,
		Credentials: getCredentialNames(provider),
		Models:      models,
		AuthMethod:  providerConfig.AuthMethod,
	}
	cfgManager.SetProviderState(cfg, state)
	cfgManager.SetActiveProvider(cfg, "claude-code", provider.Name(), scope)

	if err := cfgManager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Success message
	fmt.Println()
	printSuccess("Configuration complete!")
	fmt.Println()
	fmt.Printf("  Provider: %s\n", provider.DisplayName())
	fmt.Printf("  Scope:    %s\n", scope)
	if providerConfig.AuthMethod != "" {
		fmt.Printf("  Auth:     %s\n", providerConfig.AuthMethod)
	}
	if len(models) > 0 {
		fmt.Printf("  Models:   ")
		first := true
		for modelType, modelID := range models {
			if !first {
				fmt.Printf(", ")
			}
			fmt.Printf("%s=%s", modelType, modelID)
			first = false
		}
		fmt.Println()
	}
	fmt.Println()
	printSuccess(fmt.Sprintf("%s is now active for Claude Code in %s scope.", provider.DisplayName(), scope))
	fmt.Println()

	fmt.Println("Next steps:")
	fmt.Println("  llmconf status       - Check your configuration")
	fmt.Println("  llmconf set <name>   - Switch to another provider")
	fmt.Println()

	return nil
}

// selectProvider handles provider selection
func selectProvider(preselected string, cfgManager *config.ConfigManager, cfg *config.LLMConfConfig) (providers.Provider, error) {
	registry := providers.NewRegistry()

	// If preselected, use that
	if preselected != "" {
		provider, ok := registry.Get(preselected)
		if !ok {
			return nil, fmt.Errorf("unknown provider: %s", preselected)
		}
		return provider, nil
	}

	if nonInteractive {
		return nil, fmt.Errorf("--provider is required in non-interactive mode")
	}

	// Interactive selection
	var selected string

	providerList := registry.List()
	options := make([]huh.Option[string], len(providerList))
	for i, p := range providerList {
		display := fmt.Sprintf("%s - %s", p.DisplayName(), p.Description())
		options[i] = huh.NewOption(display, p.Name())
	}

	err := huh.NewSelect[string]().
		Title("Which LLM provider would you like to use?").
		Options(options...).
		Value(&selected).
		Run()

	if err != nil {
		return nil, err
	}

	provider, ok := registry.Get(selected)
	if !ok {
		return nil, fmt.Errorf("failed to get provider: %s", selected)
	}

	return provider, nil
}

// configureProviderInteractive handles interactive provider configuration
func configureProviderInteractive(provider providers.Provider) (*providers.ProviderConfig, error) {
	config := &providers.ProviderConfig{
		Credentials: make(map[string]string),
	}

	// Get required env vars
	requiredVars := provider.RequiredEnvVars()
	optionalVars := provider.OptionalEnvVars()

	fmt.Printf("Configuring %s:\n\n", provider.DisplayName())

	// Handle authentication method if provider supports multiple
	authMethod := detectAuthMethod(provider)
	if authMethod != "" {
		config.AuthMethod = authMethod
	}

	// Collect required variables
	for _, spec := range requiredVars {
		if spec.Sensitive {
			// For sensitive vars, we'll get from user and store in keychain
			value, err := promptForInput(spec.Description, spec.Sensitive)
			if err != nil {
				return nil, err
			}
			config.Credentials[spec.Name] = value
		} else {
			value, err := promptForInput(spec.Description, spec.Sensitive)
			if err != nil {
				return nil, err
			}
			config.Credentials[spec.Name] = value
		}
	}

	// Collect optional variables
	for _, spec := range optionalVars {
		// Skip if not relevant for current auth method
		if !isRelevantForAuthMethod(spec.Name, config.AuthMethod, provider.Name()) {
			continue
		}

		value, err := promptForOptionalInput(spec.Description, spec.Sensitive)
		if err != nil {
			return nil, err
		}
		if value != "" {
			config.Credentials[spec.Name] = value
		}
	}

	return config, nil
}

// configureModelPinning handles model pinning configuration
func configureModelPinning(provider providers.Provider) map[string]string {
	models := make(map[string]string)
	suggestions := provider.ModelSuggestions()
	defaults := provider.DefaultModels()

	// Configure Sonnet (most commonly used)
	if _, hasSonnet := suggestions["sonnet"]; hasSonnet {
		model := promptForModel("Sonnet", suggestions["sonnet"], defaults["sonnet"])
		if model != "" {
			models["sonnet"] = model
		}
	}

	// Configure Haiku
	if _, hasHaiku := suggestions["haiku"]; hasHaiku {
		model := promptForModel("Haiku", suggestions["haiku"], defaults["haiku"])
		if model != "" {
			models["haiku"] = model
		}
	}

	// Configure Opus (optional)
	if _, hasOpus := suggestions["opus"]; hasOpus {
		model := promptForModelOptional("Opus", suggestions["opus"], defaults["opus"])
		if model != "" {
			models["opus"] = model
		}
	}

	return models
}

// Helper functions

func promptForInput(prompt string, sensitive bool) (string, error) {
	var value string
	input := huh.NewInput().
		Title(prompt).
		Value(&value)

	if sensitive {
		input.EchoMode(huh.EchoModePassword)
	}

	err := input.Run()
	return value, err
}

func promptForOptionalInput(prompt string, sensitive bool) (string, error) {
	var value string
	input := huh.NewInput().
		Title(prompt + " (optional, press Enter to skip)").
		Value(&value)

	if sensitive {
		input.EchoMode(huh.EchoModePassword)
	}

	err := input.Run()
	return value, err
}

func promptForModel(modelType string, suggestions []string, defaultModel string) string {
	if nonInteractive {
		return defaultModel
	}

	options := make([]huh.Option[string], 0, len(suggestions)+2)
	for _, s := range suggestions {
		label := s
		if s == defaultModel {
			label = s + " (Recommended default)"
		}
		options = append(options, huh.NewOption(label, s))
	}
	options = append(options, huh.NewOption("Custom...", "custom"))

	var selected string
	huh.NewSelect[string]().
		Title(modelType + " model [tab for suggestions]").
		Options(options...).
		Value(&selected).
		Run()

	if selected == "custom" {
		huh.NewInput().
			Title("Enter custom " + modelType + " model ID").
			Value(&selected).
			Run()
	}

	return selected
}

func promptForModelOptional(modelType string, suggestions []string, defaultModel string) string {
	if nonInteractive {
		return ""
	}

	options := []huh.Option[string]{
		huh.NewOption("Skip", ""),
	}
	for _, s := range suggestions {
		label := s
		if s == defaultModel {
			label = s + " (Recommended default)"
		}
		options = append(options, huh.NewOption(label, s))
	}
	options = append(options, huh.NewOption("Custom...", "custom"))

	var selected string
	huh.NewSelect[string]().
		Title(modelType + " model [optional]").
		Options(options...).
		Value(&selected).
		Run()

	if selected == "custom" {
		huh.NewInput().
			Title("Enter custom " + modelType + " model ID").
			Value(&selected).
			Run()
	}

	return selected
}

func detectAuthMethod(provider providers.Provider) string {
	// For providers with multiple auth methods, detect which to use
	switch provider.Name() {
	case "bedrock":
		// Default to SSO for organizations
		return "sso"
	}
	return ""
}

func isRelevantForAuthMethod(varName, authMethod, provider string) bool {
	// Filter env vars based on auth method
	if provider == "bedrock" {
		switch authMethod {
		case "sso":
			return varName != "AWS_ACCESS_KEY_ID" && varName != "AWS_SECRET_ACCESS_KEY"
		case "api_key":
			return varName != "AWS_PROFILE"
		}
	}
	return true
}

func getCredentialNames(provider providers.Provider) []string {
	var names []string
	for _, spec := range provider.RequiredEnvVars() {
		if spec.Sensitive {
			names = append(names, spec.Name)
		}
	}
	for _, spec := range provider.OptionalEnvVars() {
		if spec.Sensitive {
			names = append(names, spec.Name)
		}
	}
	return names
}
