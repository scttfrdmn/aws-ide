package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInstanceStruct(t *testing.T) {
	now := time.Now()
	instance := Instance{
		ID:            "i-1234567890abcdef0",
		Environment:   "data-science",
		InstanceType:  "m7g.medium",
		PublicIP:      "1.2.3.4",
		KeyPair:       "aws-jupyter-us-east-1",
		LaunchedAt:    now,
		IdleTimeout:   "4h",
		Region:        "us-east-1",
		SecurityGroup: "sg-123456",
		AMIBase:       "ubuntu24-arm64",
	}

	if instance.ID != "i-1234567890abcdef0" {
		t.Errorf("Expected ID 'i-1234567890abcdef0', got %s", instance.ID)
	}
	if instance.Environment != "data-science" {
		t.Errorf("Expected environment 'data-science', got %s", instance.Environment)
	}
	if !instance.LaunchedAt.Equal(now) {
		t.Errorf("Expected LaunchedAt to equal now")
	}
}

func TestLocalStateStruct(t *testing.T) {
	state := LocalState{
		Instances: make(map[string]*Instance),
		KeyPairs:  make(map[string]string),
	}

	// Add instance
	instance := &Instance{
		ID:          "i-test123",
		Environment: "minimal",
	}
	state.Instances[instance.ID] = instance

	// Add key pair
	state.KeyPairs["test-key"] = "/path/to/key.pem"

	if len(state.Instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(state.Instances))
	}
	if len(state.KeyPairs) != 1 {
		t.Errorf("Expected 1 key pair, got %d", len(state.KeyPairs))
	}
}

func TestGetConfigDir(t *testing.T) {
	configDir := GetConfigDir()

	if configDir == "" {
		t.Error("Config dir should not be empty")
	}

	// Should end with .aws-jupyter
	if filepath.Base(configDir) != ".aws-jupyter" {
		t.Errorf("Expected config dir to end with '.aws-jupyter', got %s", configDir)
	}

	// Should be under home directory
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}

	expected := filepath.Join(home, ".aws-jupyter")
	if configDir != expected {
		t.Errorf("Expected config dir %s, got %s", expected, configDir)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	// Create temporary home directory
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")

	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}
	defer func() {
		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Logf("Warning: failed to restore HOME: %v", err)
		}
	}()

	// Ensure config dir
	if err := EnsureConfigDir(); err != nil {
		t.Fatalf("EnsureConfigDir failed: %v", err)
	}

	// Check directories exist
	configDir := GetConfigDir()
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("Config dir should exist after EnsureConfigDir")
	}

	envDir := filepath.Join(configDir, "environments")
	if _, err := os.Stat(envDir); os.IsNotExist(err) {
		t.Errorf("Environments dir should exist after EnsureConfigDir")
	}

	// Check permissions
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Failed to stat config dir: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("Expected config dir permissions 0755, got %o", info.Mode().Perm())
	}
}

func TestLoadState_NewState(t *testing.T) {
	// Create temporary home directory
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")

	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}
	defer func() {
		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Logf("Warning: failed to restore HOME: %v", err)
		}
	}()

	// Load state (should create new)
	state, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if state == nil {
		t.Fatal("State should not be nil")
	}
	if state.Instances == nil {
		t.Error("Instances map should not be nil")
	}
	if state.KeyPairs == nil {
		t.Error("KeyPairs map should not be nil")
	}
	if len(state.Instances) != 0 {
		t.Errorf("Expected 0 instances in new state, got %d", len(state.Instances))
	}
	if len(state.KeyPairs) != 0 {
		t.Errorf("Expected 0 key pairs in new state, got %d", len(state.KeyPairs))
	}
}

func TestLoadState_ExistingState(t *testing.T) {
	// Create temporary home directory
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")

	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}
	defer func() {
		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Logf("Warning: failed to restore HOME: %v", err)
		}
	}()

	// Create config directory
	configDir := GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create state file
	stateData := map[string]interface{}{
		"instances": map[string]interface{}{
			"i-test123": map[string]interface{}{
				"id":            "i-test123",
				"environment":   "data-science",
				"instance_type": "m7g.medium",
				"public_ip":     "1.2.3.4",
				"key_pair":      "test-key",
				"launched_at":   "2025-01-01T00:00:00Z",
				"region":        "us-east-1",
			},
		},
		"key_pairs": map[string]string{
			"test-key": "/path/to/key.pem",
		},
	}

	data, err := json.MarshalIndent(stateData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}

	statePath := filepath.Join(configDir, "state.json")
	if err := os.WriteFile(statePath, data, 0600); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Load state
	state, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Verify loaded data
	if len(state.Instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(state.Instances))
	}
	if len(state.KeyPairs) != 1 {
		t.Errorf("Expected 1 key pair, got %d", len(state.KeyPairs))
	}

	instance, ok := state.Instances["i-test123"]
	if !ok {
		t.Fatal("Expected instance 'i-test123' to exist")
	}
	if instance.ID != "i-test123" {
		t.Errorf("Expected instance ID 'i-test123', got %s", instance.ID)
	}
	if instance.Environment != "data-science" {
		t.Errorf("Expected environment 'data-science', got %s", instance.Environment)
	}

	keyPath, ok := state.KeyPairs["test-key"]
	if !ok {
		t.Fatal("Expected key pair 'test-key' to exist")
	}
	if keyPath != "/path/to/key.pem" {
		t.Errorf("Expected key path '/path/to/key.pem', got %s", keyPath)
	}
}

func TestLoadState_InvalidJSON(t *testing.T) {
	// Create temporary home directory
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")

	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}
	defer func() {
		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Logf("Warning: failed to restore HOME: %v", err)
		}
	}()

	// Create config directory
	configDir := GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write invalid JSON
	statePath := filepath.Join(configDir, "state.json")
	if err := os.WriteFile(statePath, []byte("invalid json {{{"), 0600); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	// Load state should fail
	_, err := LoadState()
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestSaveState(t *testing.T) {
	// Create temporary home directory
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")

	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}
	defer func() {
		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Logf("Warning: failed to restore HOME: %v", err)
		}
	}()

	// Create config directory
	configDir := GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create state
	state := &LocalState{
		Instances: map[string]*Instance{
			"i-abc123": {
				ID:          "i-abc123",
				Environment: "ml-pytorch",
			},
		},
		KeyPairs: map[string]string{
			"my-key": "/keys/my-key.pem",
		},
	}

	// Save state
	if err := state.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	statePath := filepath.Join(configDir, "state.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("State file should exist after Save")
	}

	// Verify permissions
	info, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("Failed to stat state file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected state file permissions 0600, got %o", info.Mode().Perm())
	}

	// Load and verify
	loaded, err := LoadState()
	if err != nil {
		t.Fatalf("Failed to load saved state: %v", err)
	}

	if len(loaded.Instances) != 1 {
		t.Errorf("Expected 1 instance in loaded state, got %d", len(loaded.Instances))
	}
	if len(loaded.KeyPairs) != 1 {
		t.Errorf("Expected 1 key pair in loaded state, got %d", len(loaded.KeyPairs))
	}

	instance, ok := loaded.Instances["i-abc123"]
	if !ok {
		t.Fatal("Expected instance 'i-abc123' in loaded state")
	}
	if instance.Environment != "ml-pytorch" {
		t.Errorf("Expected environment 'ml-pytorch', got %s", instance.Environment)
	}
}

func TestSaveState_RoundTrip(t *testing.T) {
	// Create temporary home directory
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")

	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}
	defer func() {
		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Logf("Warning: failed to restore HOME: %v", err)
		}
	}()

	// Create config directory
	if err := EnsureConfigDir(); err != nil {
		t.Fatalf("Failed to ensure config dir: %v", err)
	}

	// Create complex state
	now := time.Now()
	state := &LocalState{
		Instances: map[string]*Instance{
			"i-123": {
				ID:            "i-123",
				Environment:   "data-science",
				InstanceType:  "m7g.large",
				PublicIP:      "10.0.0.1",
				KeyPair:       "key1",
				LaunchedAt:    now,
				IdleTimeout:   "2h",
				TunnelPID:     12345,
				Region:        "us-west-2",
				SecurityGroup: "sg-abc",
				AMIBase:       "ubuntu24-arm64",
			},
			"i-456": {
				ID:          "i-456",
				Environment: "minimal",
			},
		},
		KeyPairs: map[string]string{
			"key1": "/path/to/key1.pem",
			"key2": "/path/to/key2.pem",
		},
	}

	// Save
	if err := state.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loaded, err := LoadState()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify all data
	if len(loaded.Instances) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(loaded.Instances))
	}
	if len(loaded.KeyPairs) != 2 {
		t.Errorf("Expected 2 key pairs, got %d", len(loaded.KeyPairs))
	}

	inst1 := loaded.Instances["i-123"]
	if inst1 == nil {
		t.Fatal("Instance i-123 should exist")
	}
	if inst1.TunnelPID != 12345 {
		t.Errorf("Expected TunnelPID 12345, got %d", inst1.TunnelPID)
	}
	if inst1.AMIBase != "ubuntu24-arm64" {
		t.Errorf("Expected AMIBase ubuntu24-arm64, got %s", inst1.AMIBase)
	}
	// Time comparison (allow small delta due to serialization)
	if inst1.LaunchedAt.Sub(now).Abs() > time.Second {
		t.Errorf("LaunchedAt time mismatch: %v vs %v", inst1.LaunchedAt, now)
	}
}
