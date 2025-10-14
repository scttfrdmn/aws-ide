package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetConfigDir(t *testing.T) {
	dir := GetConfigDir()
	if dir == "" {
		t.Error("GetConfigDir returned empty string")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("GetConfigDir should return absolute path, got: %s", dir)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir failed: %v", err)
	}

	// Check if directories were created
	configDir := GetConfigDir()
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("Config directory was not created: %s", configDir)
	}

	envDir := filepath.Join(configDir, "environments")
	if _, err := os.Stat(envDir); os.IsNotExist(err) {
		t.Errorf("Environments directory was not created: %s", envDir)
	}
}

func TestLoadState_NewState(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	state, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if state == nil {
		t.Fatal("LoadState returned nil state")
	}

	if state.Instances == nil {
		t.Error("State.Instances is nil")
	}

	if state.KeyPairs == nil {
		t.Error("State.KeyPairs is nil")
	}

	if len(state.Instances) != 0 {
		t.Errorf("Expected empty instances, got: %d", len(state.Instances))
	}
}

func TestLoadState_ExistingState(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Create config directory
	configDir := GetConfigDir()
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create test state
	testInstance := &Instance{
		ID:            "i-123456789",
		Environment:   "test",
		InstanceType:  "m7g.medium",
		PublicIP:      "1.2.3.4",
		KeyPair:       "test-key",
		LaunchedAt:    time.Now(),
		IdleTimeout:   "4h",
		TunnelPID:     1234,
		Region:        "us-west-2",
		SecurityGroup: "sg-123",
	}

	testState := &LocalState{
		Instances: map[string]*Instance{
			"i-123456789": testInstance,
		},
		KeyPairs: map[string]string{
			"test-key": "/path/to/key",
		},
	}

	// Write state file
	statePath := filepath.Join(configDir, "state.json")
	data, err := json.MarshalIndent(testState, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test state: %v", err)
	}

	err = os.WriteFile(statePath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Load state
	state, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Verify state
	if len(state.Instances) != 1 {
		t.Errorf("Expected 1 instance, got: %d", len(state.Instances))
	}

	instance := state.Instances["i-123456789"]
	if instance == nil {
		t.Fatal("Expected instance not found")
	}

	if instance.ID != "i-123456789" {
		t.Errorf("Expected instance ID i-123456789, got: %s", instance.ID)
	}

	if instance.Environment != "test" {
		t.Errorf("Expected environment test, got: %s", instance.Environment)
	}
}

func TestLocalState_Save(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Create config directory
	configDir := GetConfigDir()
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create test state
	state := &LocalState{
		Instances: map[string]*Instance{
			"i-test": {
				ID:          "i-test",
				Environment: "test",
			},
		},
		KeyPairs: map[string]string{
			"test": "/test/path",
		},
	}

	// Save state
	err = state.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created
	statePath := filepath.Join(configDir, "state.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("State file was not created")
	}

	// Load and verify content
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	var loadedState LocalState
	err = json.Unmarshal(data, &loadedState)
	if err != nil {
		t.Fatalf("Failed to unmarshal state: %v", err)
	}

	if len(loadedState.Instances) != 1 {
		t.Errorf("Expected 1 instance in saved state, got: %d", len(loadedState.Instances))
	}
}

func TestLoadState_InvalidJSON(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Create config directory
	configDir := GetConfigDir()
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write invalid JSON
	statePath := filepath.Join(configDir, "state.json")
	err = os.WriteFile(statePath, []byte("invalid json"), 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid state file: %v", err)
	}

	// Try to load state
	_, err = LoadState()
	if err == nil {
		t.Error("Expected error when loading invalid JSON, got nil")
	}
}
