package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/scttfrdmn/aws-ide/pkg/aws"
)

func TestNewKeyStorage(t *testing.T) {
	baseDir := "/tmp/test-keys"
	ks := NewKeyStorage(baseDir)

	if ks == nil {
		t.Fatal("Expected non-nil KeyStorage")
	}
	if ks.baseDir != baseDir {
		t.Errorf("Expected baseDir %s, got %s", baseDir, ks.baseDir)
	}
}

func TestDefaultKeyStorage(t *testing.T) {
	ks, err := DefaultKeyStorage()
	if err != nil {
		t.Fatalf("Failed to create default key storage: %v", err)
	}

	if ks == nil {
		t.Fatal("Expected non-nil KeyStorage")
	}

	// Should end with .aws-jupyter/keys
	if !filepath.IsAbs(ks.baseDir) {
		t.Errorf("Expected absolute path, got %s", ks.baseDir)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}

	expected := filepath.Join(home, ".aws-jupyter", "keys")
	if ks.baseDir != expected {
		t.Errorf("Expected baseDir %s, got %s", expected, ks.baseDir)
	}
}

func TestEnsureKeyDir(t *testing.T) {
	tmpDir := t.TempDir()
	keyDir := filepath.Join(tmpDir, "keys")
	ks := NewKeyStorage(keyDir)

	// Ensure directory creation
	if err := ks.EnsureKeyDir(); err != nil {
		t.Fatalf("EnsureKeyDir failed: %v", err)
	}

	// Check directory exists
	info, err := os.Stat(keyDir)
	if err != nil {
		t.Fatalf("Key directory should exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("Key path should be a directory")
	}

	// Check permissions (0700)
	if info.Mode().Perm() != 0700 {
		t.Errorf("Expected permissions 0700, got %o", info.Mode().Perm())
	}

	// Ensure idempotent (calling again should not error)
	if err := ks.EnsureKeyDir(); err != nil {
		t.Errorf("EnsureKeyDir should be idempotent: %v", err)
	}
}

func TestGetKeyPath(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(tmpDir)

	tests := []struct {
		name     string
		keyName  string
		expected string
	}{
		{
			name:     "simple name",
			keyName:  "test-key",
			expected: filepath.Join(tmpDir, "test-key.pem"),
		},
		{
			name:     "sanitize forward slash",
			keyName:  "test/key",
			expected: filepath.Join(tmpDir, "test_key.pem"),
		},
		{
			name:     "sanitize backslash",
			keyName:  "test\\key",
			expected: filepath.Join(tmpDir, "test_key.pem"),
		},
		{
			name:     "multiple slashes",
			keyName:  "path/to/my/key",
			expected: filepath.Join(tmpDir, "path_to_my_key.pem"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ks.GetKeyPath(tt.keyName)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetPublicKeyPath(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(tmpDir)

	tests := []struct {
		name     string
		keyName  string
		expected string
	}{
		{
			name:     "simple name",
			keyName:  "test-key",
			expected: filepath.Join(tmpDir, "test-key.pub"),
		},
		{
			name:     "sanitize slashes",
			keyName:  "test/key",
			expected: filepath.Join(tmpDir, "test_key.pub"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ks.GetPublicKeyPath(tt.keyName)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSavePrivateKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "keys"))

	keyInfo := &aws.KeyPairInfo{
		Name:       "test-key",
		PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\ntest-data\n-----END RSA PRIVATE KEY-----\n",
	}

	// Save key
	if err := ks.SavePrivateKey(keyInfo); err != nil {
		t.Fatalf("Failed to save private key: %v", err)
	}

	// Verify file exists
	keyPath := ks.GetKeyPath("test-key")
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Key file should exist: %v", err)
	}

	// Verify permissions (0600)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected permissions 0600, got %o", info.Mode().Perm())
	}

	// Verify content
	content, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to read key file: %v", err)
	}
	if string(content) != keyInfo.PrivateKey {
		t.Errorf("Key content mismatch")
	}
}

func TestSavePrivateKey_EmptyKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(tmpDir)

	keyInfo := &aws.KeyPairInfo{
		Name:       "test-key",
		PrivateKey: "",
	}

	err := ks.SavePrivateKey(keyInfo)
	if err == nil {
		t.Error("Expected error for empty private key")
	}
	if err.Error() != "no private key data to save" {
		t.Errorf("Expected 'no private key data to save' error, got: %v", err)
	}
}

func TestSavePublicKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "keys"))

	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ... test@example.com"

	// Save public key
	if err := ks.SavePublicKey("test-key", publicKey); err != nil {
		t.Fatalf("Failed to save public key: %v", err)
	}

	// Verify file exists
	pubKeyPath := ks.GetPublicKeyPath("test-key")
	info, err := os.Stat(pubKeyPath)
	if err != nil {
		t.Fatalf("Public key file should exist: %v", err)
	}

	// Verify permissions (0644)
	if info.Mode().Perm() != 0644 {
		t.Errorf("Expected permissions 0644, got %o", info.Mode().Perm())
	}

	// Verify content
	content, err := os.ReadFile(pubKeyPath)
	if err != nil {
		t.Fatalf("Failed to read public key file: %v", err)
	}
	if string(content) != publicKey {
		t.Errorf("Public key content mismatch")
	}
}

func TestSavePublicKey_EmptyKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(tmpDir)

	err := ks.SavePublicKey("test-key", "")
	if err == nil {
		t.Error("Expected error for empty public key")
	}
	if err.Error() != "no public key data to save" {
		t.Errorf("Expected 'no public key data to save' error, got: %v", err)
	}
}

func TestLoadPrivateKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "keys"))

	// First save a key
	keyInfo := &aws.KeyPairInfo{
		Name:       "test-key",
		PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\ntest-content\n-----END RSA PRIVATE KEY-----\n",
	}
	if err := ks.SavePrivateKey(keyInfo); err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Load key
	content, err := ks.LoadPrivateKey("test-key")
	if err != nil {
		t.Fatalf("Failed to load private key: %v", err)
	}

	if content != keyInfo.PrivateKey {
		t.Errorf("Loaded key content doesn't match original")
	}
}

func TestLoadPrivateKey_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(tmpDir)

	_, err := ks.LoadPrivateKey("non-existent-key")
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
	if err.Error() != "private key not found: non-existent-key" {
		t.Errorf("Expected 'private key not found' error, got: %v", err)
	}
}

func TestHasPrivateKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "keys"))

	// Should not exist initially
	if ks.HasPrivateKey("test-key") {
		t.Error("Key should not exist initially")
	}

	// Save a key
	keyInfo := &aws.KeyPairInfo{
		Name:       "test-key",
		PrivateKey: "test-content",
	}
	if err := ks.SavePrivateKey(keyInfo); err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Should exist now
	if !ks.HasPrivateKey("test-key") {
		t.Error("Key should exist after saving")
	}

	// Non-existent key should return false
	if ks.HasPrivateKey("non-existent") {
		t.Error("Non-existent key should return false")
	}
}

func TestDeletePrivateKey(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "keys"))

	// Save a private key
	keyInfo := &aws.KeyPairInfo{
		Name:       "test-key",
		PrivateKey: "test-private-key",
	}
	if err := ks.SavePrivateKey(keyInfo); err != nil {
		t.Fatalf("Failed to save private key: %v", err)
	}

	// Save a public key
	if err := ks.SavePublicKey("test-key", "test-public-key"); err != nil {
		t.Fatalf("Failed to save public key: %v", err)
	}

	// Verify both exist
	if !ks.HasPrivateKey("test-key") {
		t.Fatal("Private key should exist")
	}
	pubKeyPath := ks.GetPublicKeyPath("test-key")
	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		t.Fatal("Public key should exist")
	}

	// Delete key
	if err := ks.DeletePrivateKey("test-key"); err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	// Verify both are deleted
	if ks.HasPrivateKey("test-key") {
		t.Error("Private key should be deleted")
	}
	if _, err := os.Stat(pubKeyPath); !os.IsNotExist(err) {
		t.Error("Public key should also be deleted")
	}
}

func TestDeletePrivateKey_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(tmpDir)

	// Deleting non-existent key should not error
	if err := ks.DeletePrivateKey("non-existent"); err != nil {
		t.Errorf("Deleting non-existent key should not error: %v", err)
	}
}

func TestListStoredKeys(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "keys"))

	// List should be empty initially
	keys, err := ks.ListStoredKeys()
	if err != nil {
		t.Fatalf("ListStoredKeys failed: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}

	// Save multiple keys
	keyNames := []string{"key1", "key2", "key3"}
	for _, name := range keyNames {
		keyInfo := &aws.KeyPairInfo{
			Name:       name,
			PrivateKey: "test-content-" + name,
		}
		if err := ks.SavePrivateKey(keyInfo); err != nil {
			t.Fatalf("Failed to save key %s: %v", name, err)
		}
	}

	// List should return all keys
	keys, err = ks.ListStoredKeys()
	if err != nil {
		t.Fatalf("ListStoredKeys failed: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Check all keys are present
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}
	for _, expected := range keyNames {
		if !keyMap[expected] {
			t.Errorf("Expected key %s not found in list", expected)
		}
	}
}

func TestListStoredKeys_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "nonexistent"))

	// Should return empty list, not error
	keys, err := ks.ListStoredKeys()
	if err != nil {
		t.Errorf("ListStoredKeys should not error on non-existent directory: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys for non-existent directory, got %d", len(keys))
	}
}

func TestListStoredKeys_IgnoresPublicKeys(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "keys"))

	// Save private key
	keyInfo := &aws.KeyPairInfo{
		Name:       "test-key",
		PrivateKey: "private-content",
	}
	if err := ks.SavePrivateKey(keyInfo); err != nil {
		t.Fatalf("Failed to save private key: %v", err)
	}

	// Save public key
	if err := ks.SavePublicKey("test-key", "public-content"); err != nil {
		t.Fatalf("Failed to save public key: %v", err)
	}

	// List should only return private key name once
	keys, err := ks.ListStoredKeys()
	if err != nil {
		t.Fatalf("ListStoredKeys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("Expected 1 key (private only), got %d", len(keys))
	}
	if keys[0] != "test-key" {
		t.Errorf("Expected key name 'test-key', got %s", keys[0])
	}
}

func TestValidateKeyPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "keys"))

	// Save key with correct permissions
	keyInfo := &aws.KeyPairInfo{
		Name:       "test-key",
		PrivateKey: "test-content",
	}
	if err := ks.SavePrivateKey(keyInfo); err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Validate should pass
	if err := ks.ValidateKeyPermissions("test-key"); err != nil {
		t.Errorf("ValidateKeyPermissions should pass: %v", err)
	}

	// Change permissions to insecure
	keyPath := ks.GetKeyPath("test-key")
	if err := os.Chmod(keyPath, 0644); err != nil {
		t.Fatalf("Failed to change permissions: %v", err)
	}

	// Validate should fail
	err := ks.ValidateKeyPermissions("test-key")
	if err == nil {
		t.Error("Expected error for insecure permissions")
	}
}

func TestValidateKeyPermissions_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(tmpDir)

	err := ks.ValidateKeyPermissions("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
}

func TestFixKeyPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "keys"))

	// Save key
	keyInfo := &aws.KeyPairInfo{
		Name:       "test-key",
		PrivateKey: "test-content",
	}
	if err := ks.SavePrivateKey(keyInfo); err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Change to insecure permissions
	keyPath := ks.GetKeyPath("test-key")
	if err := os.Chmod(keyPath, 0644); err != nil {
		t.Fatalf("Failed to change permissions: %v", err)
	}

	// Verify insecure
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Failed to stat key: %v", err)
	}
	if info.Mode().Perm() == 0600 {
		t.Error("Permissions should be insecure before fix")
	}

	// Fix permissions
	if err := ks.FixKeyPermissions("test-key"); err != nil {
		t.Fatalf("Failed to fix permissions: %v", err)
	}

	// Verify fixed
	info, err = os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Failed to stat key: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected permissions 0600 after fix, got %o", info.Mode().Perm())
	}
}

func TestCleanupOrphanedKeys(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "keys"))

	// Save multiple keys
	allKeys := []string{"key1", "key2", "key3", "key4"}
	for _, name := range allKeys {
		keyInfo := &aws.KeyPairInfo{
			Name:       name,
			PrivateKey: "content-" + name,
		}
		if err := ks.SavePrivateKey(keyInfo); err != nil {
			t.Fatalf("Failed to save key %s: %v", name, err)
		}
	}

	// Verify all exist
	storedKeys, err := ks.ListStoredKeys()
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}
	if len(storedKeys) != 4 {
		t.Fatalf("Expected 4 stored keys, got %d", len(storedKeys))
	}

	// Cleanup with only key1 and key3 existing in AWS
	existingKeys := []string{"key1", "key3"}
	if err := ks.CleanupOrphanedKeys(existingKeys); err != nil {
		t.Fatalf("CleanupOrphanedKeys failed: %v", err)
	}

	// Verify only key1 and key3 remain
	storedKeys, err = ks.ListStoredKeys()
	if err != nil {
		t.Fatalf("Failed to list keys after cleanup: %v", err)
	}
	if len(storedKeys) != 2 {
		t.Errorf("Expected 2 keys after cleanup, got %d", len(storedKeys))
	}

	keyMap := make(map[string]bool)
	for _, key := range storedKeys {
		keyMap[key] = true
	}
	if !keyMap["key1"] || !keyMap["key3"] {
		t.Error("Expected key1 and key3 to remain after cleanup")
	}
	if keyMap["key2"] || keyMap["key4"] {
		t.Error("Expected key2 and key4 to be removed after cleanup")
	}
}

func TestCleanupOrphanedKeys_EmptyExisting(t *testing.T) {
	tmpDir := t.TempDir()
	ks := NewKeyStorage(filepath.Join(tmpDir, "keys"))

	// Save keys
	keyNames := []string{"key1", "key2"}
	for _, name := range keyNames {
		keyInfo := &aws.KeyPairInfo{
			Name:       name,
			PrivateKey: "content",
		}
		if err := ks.SavePrivateKey(keyInfo); err != nil {
			t.Fatalf("Failed to save key: %v", err)
		}
	}

	// Cleanup with no existing keys (all are orphaned)
	if err := ks.CleanupOrphanedKeys([]string{}); err != nil {
		t.Fatalf("CleanupOrphanedKeys failed: %v", err)
	}

	// All keys should be removed
	storedKeys, err := ks.ListStoredKeys()
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}
	if len(storedKeys) != 0 {
		t.Errorf("Expected 0 keys after cleanup with no existing keys, got %d", len(storedKeys))
	}
}
