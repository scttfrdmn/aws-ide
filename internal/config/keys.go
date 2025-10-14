package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scttfrdmn/aws-jupyter/internal/aws"
)

const (
	// File and directory permissions
	permKeyDir     = 0700 // Owner only (rwx------)
	permPrivateKey = 0600 // Owner read/write only (rw-------)
	permPublicKey  = 0644 // Owner read/write, others read (rw-r--r--)
)

// KeyStorage manages local SSH key storage
type KeyStorage struct {
	baseDir string
}

// NewKeyStorage creates a new key storage manager
func NewKeyStorage(baseDir string) *KeyStorage {
	return &KeyStorage{
		baseDir: baseDir,
	}
}

// DefaultKeyStorage returns the default key storage in ~/.aws-jupyter/keys
func DefaultKeyStorage() (*KeyStorage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".aws-jupyter", "keys")
	return NewKeyStorage(baseDir), nil
}

// EnsureKeyDir creates the key storage directory with secure permissions
func (ks *KeyStorage) EnsureKeyDir() error {
	// Create directory with owner-only permissions
	if err := os.MkdirAll(ks.baseDir, permKeyDir); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}
	return nil
}

// GetKeyPath returns the path for a private key file
func (ks *KeyStorage) GetKeyPath(keyName string) string {
	// Sanitize key name for filesystem safety
	safeName := strings.ReplaceAll(keyName, "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	return filepath.Join(ks.baseDir, safeName+".pem")
}

// GetPublicKeyPath returns the path for a public key file
func (ks *KeyStorage) GetPublicKeyPath(keyName string) string {
	// Sanitize key name for filesystem safety
	safeName := strings.ReplaceAll(keyName, "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	return filepath.Join(ks.baseDir, safeName+".pub")
}

// SavePrivateKey saves a private key with secure permissions (600)
func (ks *KeyStorage) SavePrivateKey(keyInfo *aws.KeyPairInfo) error {
	if keyInfo.PrivateKey == "" {
		return fmt.Errorf("no private key data to save")
	}

	// Ensure key directory exists
	if err := ks.EnsureKeyDir(); err != nil {
		return err
	}

	keyPath := ks.GetKeyPath(keyInfo.Name)

	// Write private key with owner read/write only permissions
	if err := os.WriteFile(keyPath, []byte(keyInfo.PrivateKey), permPrivateKey); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	return nil
}

// SavePublicKey saves a public key with standard permissions (644)
func (ks *KeyStorage) SavePublicKey(keyName, publicKey string) error {
	if publicKey == "" {
		return fmt.Errorf("no public key data to save")
	}

	// Ensure key directory exists
	if err := ks.EnsureKeyDir(); err != nil {
		return err
	}

	pubKeyPath := ks.GetPublicKeyPath(keyName)

	// Write public key with owner read/write, others read permissions
	if err := os.WriteFile(pubKeyPath, []byte(publicKey), permPublicKey); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	return nil
}

// LoadPrivateKey loads a private key from local storage
func (ks *KeyStorage) LoadPrivateKey(keyName string) (string, error) {
	keyPath := ks.GetKeyPath(keyName)

	data, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("private key not found: %s", keyName)
		}
		return "", fmt.Errorf("failed to read private key: %w", err)
	}

	return string(data), nil
}

// HasPrivateKey checks if a private key exists locally
func (ks *KeyStorage) HasPrivateKey(keyName string) bool {
	keyPath := ks.GetKeyPath(keyName)
	_, err := os.Stat(keyPath)
	return err == nil
}

// DeletePrivateKey removes a private key from local storage
func (ks *KeyStorage) DeletePrivateKey(keyName string) error {
	keyPath := ks.GetKeyPath(keyName)

	if err := os.Remove(keyPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted, no error
		}
		return fmt.Errorf("failed to delete private key: %w", err)
	}

	// Also try to delete public key if it exists (best-effort cleanup)
	pubKeyPath := ks.GetPublicKeyPath(keyName)
	if err := os.Remove(pubKeyPath); err != nil && !os.IsNotExist(err) {
		// Log warning but don't fail - public key cleanup is optional
		fmt.Printf("Warning: Failed to delete public key %s: %v\n", pubKeyPath, err)
	}

	return nil
}

// ListStoredKeys returns a list of key names that have private keys stored locally
func (ks *KeyStorage) ListStoredKeys() ([]string, error) {
	entries, err := os.ReadDir(ks.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // No keys stored yet
		}
		return nil, fmt.Errorf("failed to list stored keys: %w", err)
	}

	var keyNames []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Look for .pem files (private keys)
		if strings.HasSuffix(name, ".pem") {
			keyName := strings.TrimSuffix(name, ".pem")
			keyNames = append(keyNames, keyName)
		}
	}

	return keyNames, nil
}

// ValidateKeyPermissions checks that key files have secure permissions
func (ks *KeyStorage) ValidateKeyPermissions(keyName string) error {
	keyPath := ks.GetKeyPath(keyName)

	info, err := os.Stat(keyPath)
	if err != nil {
		return fmt.Errorf("failed to check key permissions: %w", err)
	}

	mode := info.Mode()
	// Check that permissions are owner read/write only
	if mode.Perm() != permPrivateKey {
		return fmt.Errorf("private key has insecure permissions %o, should be %o", mode.Perm(), permPrivateKey)
	}

	return nil
}

// FixKeyPermissions sets correct permissions on a private key file
func (ks *KeyStorage) FixKeyPermissions(keyName string) error {
	keyPath := ks.GetKeyPath(keyName)

	if err := os.Chmod(keyPath, permPrivateKey); err != nil {
		return fmt.Errorf("failed to fix key permissions: %w", err)
	}

	return nil
}

// CleanupOrphanedKeys removes keys that don't exist in AWS
func (ks *KeyStorage) CleanupOrphanedKeys(existingKeys []string) error {
	storedKeys, err := ks.ListStoredKeys()
	if err != nil {
		return err
	}

	// Create a set of existing keys for fast lookup
	existingSet := make(map[string]bool)
	for _, key := range existingKeys {
		existingSet[key] = true
	}

	// Remove keys that don't exist in AWS
	for _, storedKey := range storedKeys {
		if !existingSet[storedKey] {
			fmt.Printf("Removing orphaned key: %s\n", storedKey)
			if err := ks.DeletePrivateKey(storedKey); err != nil {
				fmt.Printf("Warning: failed to delete orphaned key %s: %v\n", storedKey, err)
			}
		}
	}

	return nil
}
