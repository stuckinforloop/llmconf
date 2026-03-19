package secrets

import (
	"fmt"
	"strings"
)

// SecretStore defines the interface for credential storage
type SecretStore interface {
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	List(prefix string) ([]string, error)
}

// Key naming convention: llmconf:<provider>:<credential_name>
// e.g., llmconf:bedrock:aws_access_key_id

const keyPrefix = "llmconf:"

// Store wraps a SecretStore with provider-specific methods
type Store struct {
	backend SecretStore
}

// NewStore creates a new Store with the given backend
func NewStore(backend SecretStore) *Store {
	return &Store{backend: backend}
}

// SetCredential stores a credential for a provider
func (s *Store) SetCredential(provider, name, value string) error {
	key := formatKey(provider, name)
	return s.backend.Set(key, value)
}

// GetCredential retrieves a credential for a provider
func (s *Store) GetCredential(provider, name string) (string, error) {
	key := formatKey(provider, name)
	return s.backend.Get(key)
}

// DeleteCredential removes a credential for a provider
func (s *Store) DeleteCredential(provider, name string) error {
	key := formatKey(provider, name)
	return s.backend.Delete(key)
}

// ListCredentials returns all credential names for a provider
func (s *Store) ListCredentials(provider string) ([]string, error) {
	prefix := formatKey(provider, "")
	keys, err := s.backend.List(prefix)
	if err != nil {
		return nil, err
	}

	// Strip prefix and return credential names
	var result []string
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			result = append(result, key[len(prefix):])
		}
	}
	return result, nil
}

// ListProviders returns all providers with stored credentials
func (s *Store) ListProviders() ([]string, error) {
	keys, err := s.backend.List(keyPrefix)
	if err != nil {
		return nil, err
	}

	// Extract unique provider names
	providerMap := make(map[string]bool)
	for _, key := range keys {
		if strings.HasPrefix(key, keyPrefix) {
			parts := strings.Split(key[len(keyPrefix):], ":")
			if len(parts) > 0 && parts[0] != "" {
				providerMap[parts[0]] = true
			}
		}
	}

	var result []string
	for provider := range providerMap {
		result = append(result, provider)
	}
	return result, nil
}

// StoreConfig stores all credentials for a provider configuration
func (s *Store) StoreConfig(provider string, credentials map[string]string) error {
	for name, value := range credentials {
		if value != "" {
			if err := s.SetCredential(provider, name, value); err != nil {
				return fmt.Errorf("failed to store %s: %w", name, err)
			}
		}
	}
	return nil
}

// LoadConfig retrieves all credentials for a provider
func (s *Store) LoadConfig(provider string, credentialNames []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, name := range credentialNames {
		value, err := s.GetCredential(provider, name)
		if err != nil {
			// Credential might not exist, skip
			continue
		}
		result[name] = value
	}
	return result, nil
}

// DeleteProvider removes all credentials for a provider
func (s *Store) DeleteProvider(provider string) error {
	credentials, err := s.ListCredentials(provider)
	if err != nil {
		return err
	}

	for _, name := range credentials {
		if err := s.DeleteCredential(provider, name); err != nil {
			return fmt.Errorf("failed to delete %s: %w", name, err)
		}
	}
	return nil
}

// formatKey creates a storage key for a credential
func formatKey(provider, name string) string {
	if name == "" {
		return fmt.Sprintf("%s%s:", keyPrefix, provider)
	}
	return fmt.Sprintf("%s%s:%s", keyPrefix, provider, name)
}
