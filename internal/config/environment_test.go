package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvironment_BuiltIn(t *testing.T) {
	// Skip this test if we're not in the project root with environments
	if _, err := os.Stat("environments/data-science.yaml"); os.IsNotExist(err) {
		t.Skip("Skipping test: built-in environments not available")
		return
	}

	// Test loading a built-in environment
	env, err := LoadEnvironment("data-science")
	if err != nil {
		t.Fatalf("Failed to load data-science environment: %v", err)
	}

	if env.Name != "Data Science" {
		t.Errorf("Expected name 'Data Science', got: %s", env.Name)
	}

	if env.InstanceType != "m7g.medium" {
		t.Errorf("Expected instance type 'm7g.medium', got: %s", env.InstanceType)
	}

	if env.AMIBase != "ubuntu22-arm64" {
		t.Errorf("Expected AMI base 'ubuntu22-arm64', got: %s", env.AMIBase)
	}

	if len(env.Packages) == 0 {
		t.Error("Expected packages to be present")
	}

	if len(env.PipPackages) == 0 {
		t.Error("Expected pip packages to be present")
	}
}

func TestLoadEnvironment_UserOverride(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create user config directory
	configDir := GetConfigDir()
	envDir := filepath.Join(configDir, "environments")
	err := os.MkdirAll(envDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create user env dir: %v", err)
	}

	// Create user environment file
	userEnvPath := filepath.Join(envDir, "test-env.yaml")
	userEnvContent := `name: "User Test Environment"
instance_type: "c7g.large"
ami_base: "ubuntu22-arm64"
ebs_volume_size: 50
packages:
  - python3-pip
  - git
pip_packages:
  - numpy
jupyter_extensions:
  - jupyterlab
environment_vars:
  TEST_VAR: "test_value"
`
	err = os.WriteFile(userEnvPath, []byte(userEnvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write user env file: %v", err)
	}

	// Load environment
	env, err := LoadEnvironment("test-env")
	if err != nil {
		t.Fatalf("Failed to load user environment: %v", err)
	}

	if env.Name != "User Test Environment" {
		t.Errorf("Expected name 'User Test Environment', got: %s", env.Name)
	}

	if env.InstanceType != "c7g.large" {
		t.Errorf("Expected instance type 'c7g.large', got: %s", env.InstanceType)
	}

	if env.EBSVolumeSize != 50 {
		t.Errorf("Expected EBS volume size 50, got: %d", env.EBSVolumeSize)
	}

	if env.EnvironmentVars["TEST_VAR"] != "test_value" {
		t.Errorf("Expected TEST_VAR=test_value, got: %s", env.EnvironmentVars["TEST_VAR"])
	}
}

func TestLoadEnvironment_NotFound(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Try to load non-existent environment
	_, err := LoadEnvironment("non-existent-env")
	if err == nil {
		t.Error("Expected error when loading non-existent environment, got nil")
	}

	expectedMsg := "environment non-existent-env not found"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestLoadEnvironment_InvalidYAML(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create user config directory
	configDir := GetConfigDir()
	envDir := filepath.Join(configDir, "environments")
	err := os.MkdirAll(envDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create user env dir: %v", err)
	}

	// Create invalid YAML file
	invalidEnvPath := filepath.Join(envDir, "invalid-env.yaml")
	invalidContent := `invalid: yaml: content: [`
	err = os.WriteFile(invalidEnvPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid env file: %v", err)
	}

	// Try to load environment
	_, err = LoadEnvironment("invalid-env")
	if err == nil {
		t.Error("Expected error when loading invalid YAML, got nil")
	}
}

func TestListEnvironments(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// List environments (should work even if empty)
	_, err := ListEnvironments()
	if err != nil {
		t.Fatalf("ListEnvironments failed: %v", err)
	}

	// May or may not have built-in environments depending on working directory
	// But should not fail and should return some list (even if empty)

	// Create user config directory and add user environment
	configDir := GetConfigDir()
	envDir := filepath.Join(configDir, "environments")
	err = os.MkdirAll(envDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create user env dir: %v", err)
	}

	// Create user environment file
	userEnvPath := filepath.Join(envDir, "my-custom-env.yaml")
	userEnvContent := `name: "My Custom Environment"
instance_type: "m7g.medium"
ami_base: "ubuntu22-arm64"
ebs_volume_size: 20
packages: []
pip_packages: []
jupyter_extensions: []
environment_vars: {}
`
	err = os.WriteFile(userEnvPath, []byte(userEnvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write user env file: %v", err)
	}

	// List environments again
	envs, err := ListEnvironments()
	if err != nil {
		t.Fatalf("ListEnvironments failed after adding user env: %v", err)
	}

	// Check for user environment
	found := false
	for _, env := range envs {
		if env == "my-custom-env" {
			found = true
			break
		}
	}
	if !found {
		t.Error("User environment 'my-custom-env' not found in list")
	}
}

func TestListEnvironments_NoDuplicates(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create user config directory
	configDir := GetConfigDir()
	envDir := filepath.Join(configDir, "environments")
	err := os.MkdirAll(envDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create user env dir: %v", err)
	}

	// Create user environment file with same name as built-in
	userEnvPath := filepath.Join(envDir, "data-science.yaml")
	userEnvContent := `name: "User Data Science"
instance_type: "m7g.large"
ami_base: "ubuntu22-arm64"
ebs_volume_size: 30
packages: []
pip_packages: []
jupyter_extensions: []
environment_vars: {}
`
	err = os.WriteFile(userEnvPath, []byte(userEnvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write user env file: %v", err)
	}

	// List environments
	envs, err := ListEnvironments()
	if err != nil {
		t.Fatalf("ListEnvironments failed: %v", err)
	}

	// Count occurrences of "data-science"
	count := 0
	for _, env := range envs {
		if env == "data-science" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 occurrence of 'data-science', got: %d", count)
	}

	// Verify user version takes precedence
	env, err := LoadEnvironment("data-science")
	if err != nil {
		t.Fatalf("Failed to load data-science environment: %v", err)
	}

	if env.Name != "User Data Science" {
		t.Errorf("Expected user environment to override, got name: %s", env.Name)
	}
}
