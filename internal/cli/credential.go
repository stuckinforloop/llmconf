package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stuckinforloop/llmconf/internal/secrets"
)

// credentialCmd represents the credential command
var credentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "Manage credentials in the secure store",
	Long: `Manage provider credentials stored in the OS keychain/keyring.

This command is used internally by apiKeyHelper to retrieve credentials
securely without storing them in settings.json files.

Examples:
  llmconf credential get fireworks ANTHROPIC_API_KEY  # Get a credential
  llmconf credential set fireworks ANTHROPIC_API_KEY  # Set a credential
  llmconf credential list fireworks                   # List provider credentials`,
}

// credentialGetCmd retrieves a credential
var credentialGetCmd = &cobra.Command{
	Use:   "get <provider> <name>",
	Short: "Retrieve a credential from the secure store",
	Long: `Retrieve a credential value from the OS keychain/keyring.

This command is used by Claude Code's apiKeyHelper to fetch credentials
securely at runtime.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := args[0]
		name := args[1]

		backend := secrets.NewKeychainStore()
		store := secrets.NewStore(backend)

		value, err := store.GetCredential(provider, name)
		if err != nil {
			return fmt.Errorf("failed to get credential: %w", err)
		}

		fmt.Print(value)
		return nil
	},
}

// credentialSetCmd stores a credential
var credentialSetCmd = &cobra.Command{
	Use:   "set <provider> <name>",
	Short: "Store a credential in the secure store",
	Long:  `Store a credential value in the OS keychain/keyring securely.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := args[0]
		name := args[1]

		fmt.Printf("Enter value for %s/%s: ", provider, name)
		var value string
		fmt.Scanln(&value)

		backend := secrets.NewKeychainStore()
		store := secrets.NewStore(backend)

		if err := store.SetCredential(provider, name, value); err != nil {
			return fmt.Errorf("failed to set credential: %w", err)
		}

		printSuccess(fmt.Sprintf("Credential %s/%s stored securely", provider, name))
		return nil
	},
}

// credentialListCmd lists credentials for a provider
var credentialListCmd = &cobra.Command{
	Use:   "list <provider>",
	Short: "List credentials for a provider",
	Long:  `List all credential names stored for a provider.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := args[0]

		backend := secrets.NewKeychainStore()
		store := secrets.NewStore(backend)

		credentials, err := store.ListCredentials(provider)
		if err != nil {
			return fmt.Errorf("failed to list credentials: %w", err)
		}

		if len(credentials) == 0 {
			fmt.Printf("No credentials found for provider: %s\n", provider)
			return nil
		}

		fmt.Printf("Credentials for %s:\n", provider)
		for _, cred := range credentials {
			fmt.Printf("  - %s\n", cred)
		}

		return nil
	},
}

// credentialDeleteCmd removes a credential
var credentialDeleteCmd = &cobra.Command{
	Use:   "delete <provider> <name>",
	Short: "Delete a credential from the secure store",
	Long:  `Remove a credential from the OS keychain/keyring.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := args[0]
		name := args[1]

		backend := secrets.NewKeychainStore()
		store := secrets.NewStore(backend)

		if err := store.DeleteCredential(provider, name); err != nil {
			return fmt.Errorf("failed to delete credential: %w", err)
		}

		printSuccess(fmt.Sprintf("Credential %s/%s deleted", provider, name))
		return nil
	},
}

func init() {
	credentialCmd.AddCommand(credentialGetCmd)
	credentialCmd.AddCommand(credentialSetCmd)
	credentialCmd.AddCommand(credentialListCmd)
	credentialCmd.AddCommand(credentialDeleteCmd)

	// Add to root
	rootCmd.AddCommand(credentialCmd)
}
