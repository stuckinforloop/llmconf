package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/scrypt"
)

// FileStore implements SecretStore using an encrypted file
type FileStore struct {
	filePath string
	key      []byte
	data     map[string]string
}

// NewFileStore creates a new file-based secret store
func NewFileStore(filePath string, password string) (*FileStore, error) {
	// Derive key from password
	key := deriveKey(password)

	store := &FileStore{
		filePath: filePath,
		key:      key,
		data:     make(map[string]string),
	}

	// Load existing data if file exists
	if _, err := os.Stat(filePath); err == nil {
		if err := store.load(); err != nil {
			return nil, fmt.Errorf("failed to load store: %w", err)
		}
	}

	return store, nil
}

// Set stores a secret
func (f *FileStore) Set(key, value string) error {
	f.data[key] = value
	return f.save()
}

// Get retrieves a secret
func (f *FileStore) Get(key string) (string, error) {
	value, ok := f.data[key]
	if !ok {
		return "", fmt.Errorf("secret not found: %s", key)
	}
	return value, nil
}

// Delete removes a secret
func (f *FileStore) Delete(key string) error {
	delete(f.data, key)
	return f.save()
}

// List returns all keys matching the prefix
func (f *FileStore) List(prefix string) ([]string, error) {
	var result []string
	for key := range f.data {
		if strings.HasPrefix(key, prefix) {
			result = append(result, key)
		}
	}
	return result, nil
}

// save encrypts and saves the data to file
func (f *FileStore) save() error {
	// Marshal data
	jsonData, err := json.Marshal(f.data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Encrypt
	encrypted, err := f.encrypt(jsonData)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(f.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(f.filePath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// load reads and decrypts the data from file
func (f *FileStore) load() error {
	// Read file
	encrypted, err := os.ReadFile(f.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Decrypt
	jsonData, err := f.decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt data (wrong password?): %w", err)
	}

	// Unmarshal
	if err := json.Unmarshal(jsonData, &f.data); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

// encrypt encrypts data using AES-GCM
func (f *FileStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

// decrypt decrypts data using AES-GCM
func (f *FileStore) decrypt(ciphertext []byte) ([]byte, error) {
	// Decode base64
	data, err := base64.StdEncoding.DecodeString(string(ciphertext))
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertextBytes, nil)
}

// deriveKey derives a key from password using Argon2id
func deriveKey(password string) []byte {
	salt := make([]byte, 16)
	// Use a fixed salt for simplicity - in production this should be stored
	copy(salt, []byte("llmconf-salt-v1"))

	return argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
}

// deriveKeyScrypt derives a key using scrypt (fallback)
func deriveKeyScrypt(password string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
}

// verifyPassword verifies if the provided password is correct
func (f *FileStore) verifyPassword(password string) bool {
	testKey := deriveKey(password)
	return subtle.ConstantTimeCompare(f.key, testKey) == 1
}

// ChangePassword changes the encryption password
func (f *FileStore) ChangePassword(newPassword string) error {
	f.key = deriveKey(newPassword)
	return f.save()
}
