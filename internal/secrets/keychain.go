package secrets

import (
	"github.com/zalando/go-keyring"
)

// KeychainStore implements SecretStore using the OS keychain/keyring
type KeychainStore struct {
	serviceName string
}

// NewKeychainStore creates a new keychain-based secret store
func NewKeychainStore() *KeychainStore {
	return &KeychainStore{
		serviceName: "llmconf",
	}
}

// Set stores a secret in the keychain
func (k *KeychainStore) Set(key, value string) error {
	return keyring.Set(k.serviceName, key, value)
}

// Get retrieves a secret from the keychain
func (k *KeychainStore) Get(key string) (string, error) {
	return keyring.Get(k.serviceName, key)
}

// Delete removes a secret from the keychain
func (k *KeychainStore) Delete(key string) error {
	return keyring.Delete(k.serviceName, key)
}

// List returns all keys matching the prefix
// Note: OS keychains don't support prefix listing natively
// This is a limitation - we return empty list
func (k *KeychainStore) List(prefix string) ([]string, error) {
	// OS keychains don't support listing by prefix
	// Return empty list - this is a known limitation
	return []string{}, nil
}
