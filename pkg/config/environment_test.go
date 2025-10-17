package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvironmentStruct(t *testing.T) {
	env := Environment{
		Name:          "test-env",
		InstanceType:  "t3.medium",
		AMIBase:       "ubuntu22-arm64",
		EBSVolumeSize: 30,
		Packages:      []string{"python3", "git"},
		PipPackages:   []string{"jupyter", "pandas"},
		EnvironmentVars: map[string]string{
			"PYTHONPATH": "/home/ubuntu/notebooks",
		},
	}

	if env.Name != "test-env" {
		t.Errorf("Expected name 'test-env', got %s", env.Name)
	}
	if env.InstanceType != "t3.medium" {
		t.Errorf("Expected instance type 't3.medium', got %s", env.InstanceType)
	}
	if len(env.Packages) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(env.Packages))
	}
	if len(env.PipPackages) != 2 {
		t.Errorf("Expected 2 pip packages, got %d", len(env.PipPackages))
	}
}

func TestLoadEnvironment_NotFound(t *testing.T) {
	_, err := LoadEnvironment("non-existent-environment-12345")
	if err == nil {
		t.Error("Expected error for non-existent environment, got nil")
	}
	if err != nil && err.Error() != "environment non-existent-environment-12345 not found" {
		t.Errorf("Expected 'environment not found' error, got: %v", err)
	}
}

func TestLoadEnvironment_FromUserConfig(t *testing.T) {
	// Create temporary config directory
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")

	// Set HOME to temp directory
	if err := os.Setenv("HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}
	defer func() {
		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Logf("Warning: failed to restore HOME: %v", err)
		}
	}()

	// Create config directory structure
	configDir := GetConfigDir()
	envDir := filepath.Join(configDir, "environments")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("Failed to create env dir: %v", err)
	}

	// Create test environment file
	testEnvContent := `name: "test-env"
instance_type: "m7g.medium"
ami_base: "ubuntu22-arm64"
ebs_volume_size: 30
packages:
  - python3-pip
  - git
pip_packages:
  - jupyterlab
  - pandas
jupyter_extensions:
  - jupyterlab
environment_vars:
  PYTHONPATH: "/home/ubuntu/notebooks"
`
	envPath := filepath.Join(envDir, "test-env.yaml")
	if err := os.WriteFile(envPath, []byte(testEnvContent), 0644); err != nil {
		t.Fatalf("Failed to write test environment: %v", err)
	}

	// Test loading
	env, err := LoadEnvironment("test-env")
	if err != nil {
		t.Fatalf("Failed to load environment: %v", err)
	}

	if env.Name != "test-env" {
		t.Errorf("Expected name 'test-env', got %s", env.Name)
	}
	if env.InstanceType != "m7g.medium" {
		t.Errorf("Expected instance type 'm7g.medium', got %s", env.InstanceType)
	}
	if env.EBSVolumeSize != 30 {
		t.Errorf("Expected EBS volume size 30, got %d", env.EBSVolumeSize)
	}
	if len(env.Packages) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(env.Packages))
	}
	if len(env.PipPackages) != 2 {
		t.Errorf("Expected 2 pip packages, got %d", len(env.PipPackages))
	}
	if env.EnvironmentVars["PYTHONPATH"] != "/home/ubuntu/notebooks" {
		t.Errorf("Expected PYTHONPATH env var, got: %v", env.EnvironmentVars)
	}
}

func TestLoadEnvironment_InvalidYAML(t *testing.T) {
	// Create temporary directory
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
	envDir := filepath.Join(configDir, "environments")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("Failed to create env dir: %v", err)
	}

	// Create invalid YAML file
	invalidYAML := `name: test
invalid yaml content [[[
not: proper: structure
`
	envPath := filepath.Join(envDir, "invalid.yaml")
	if err := os.WriteFile(envPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid YAML: %v", err)
	}

	// Test loading invalid YAML
	_, err := LoadEnvironment("invalid")
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestListEnvironments_Empty(t *testing.T) {
	// Create temporary directory with no environments
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

	// Create config directory but no environments
	configDir := GetConfigDir()
	envDir := filepath.Join(configDir, "environments")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("Failed to create env dir: %v", err)
	}

	// List should not error - may find built-in environments
	envs, err := ListEnvironments()
	if err != nil {
		t.Errorf("Expected no error for empty user environments, got: %v", err)
	}
	// Note: May find built-in environments from project structure, so just verify no error
	t.Logf("Found %d environments (may include built-in)", len(envs))
}

func TestListEnvironments_Multiple(t *testing.T) {
	// Create temporary directory
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
	envDir := filepath.Join(configDir, "environments")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatalf("Failed to create env dir: %v", err)
	}

	// Create multiple environment files with unique names to avoid conflicts with built-ins
	envNames := []string{"test-unique-env1", "test-unique-env2", "test-unique-env3"}
	for _, name := range envNames {
		content := `name: "` + name + `"
instance_type: "t3.medium"
ami_base: "ubuntu22-arm64"
ebs_volume_size: 20
packages: []
pip_packages: []
`
		envPath := filepath.Join(envDir, name+".yaml")
		if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write environment %s: %v", name, err)
		}
	}

	// List environments
	envs, err := ListEnvironments()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check our custom environments are present (may also include built-in environments)
	envMap := make(map[string]bool)
	for _, name := range envs {
		envMap[name] = true
	}
	for _, expected := range envNames {
		if !envMap[expected] {
			t.Errorf("Expected environment '%s' not found in list", expected)
		}
	}
}

func TestListEnvironments_Deduplication(t *testing.T) {
	// Create temporary directory
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

	// Create both user and built-in directories
	configDir := GetConfigDir()
	userEnvDir := filepath.Join(configDir, "environments")
	if err := os.MkdirAll(userEnvDir, 0755); err != nil {
		t.Fatalf("Failed to create user env dir: %v", err)
	}

	builtinEnvDir := "environments"
	if err := os.MkdirAll(builtinEnvDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin env dir: %v", err)
	}
	defer os.RemoveAll(builtinEnvDir)

	// Create same environment in both locations
	content := `name: "duplicate"
instance_type: "t3.medium"
ami_base: "ubuntu22-arm64"
ebs_volume_size: 20
packages: []
pip_packages: []
`
	// User version
	userPath := filepath.Join(userEnvDir, "duplicate.yaml")
	if err := os.WriteFile(userPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write user environment: %v", err)
	}

	// Built-in version
	builtinPath := filepath.Join(builtinEnvDir, "duplicate.yaml")
	if err := os.WriteFile(builtinPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write builtin environment: %v", err)
	}

	// List should deduplicate
	envs, err := ListEnvironments()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	count := 0
	for _, name := range envs {
		if name == "duplicate" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 'duplicate' environment (deduplicated), got %d", count)
	}
}
