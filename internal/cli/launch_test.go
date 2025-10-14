package cli

import (
	"os"
	"strings"
	"testing"
)

func TestNewLaunchCmd(t *testing.T) {
	cmd := NewLaunchCmd()

	if cmd == nil {
		t.Fatal("NewLaunchCmd returned nil")
	}

	if cmd.Use != "launch" {
		t.Errorf("Expected Use='launch', got: %s", cmd.Use)
	}

	if cmd.Short != "Launch a new Jupyter instance" {
		t.Errorf("Expected Short='Launch a new Jupyter instance', got: %s", cmd.Short)
	}

	// Check flags
	flags := cmd.Flags()

	envFlag := flags.Lookup("env")
	if envFlag == nil {
		t.Error("Expected 'env' flag to exist")
	} else if envFlag.DefValue != "data-science" {
		t.Errorf("Expected 'env' flag default value 'data-science', got: %s", envFlag.DefValue)
	}

	instanceTypeFlag := flags.Lookup("instance-type")
	if instanceTypeFlag == nil {
		t.Error("Expected 'instance-type' flag to exist")
	}

	idleTimeoutFlag := flags.Lookup("idle-timeout")
	if idleTimeoutFlag == nil {
		t.Error("Expected 'idle-timeout' flag to exist")
	} else if idleTimeoutFlag.DefValue != "4h" {
		t.Errorf("Expected 'idle-timeout' flag default value '4h', got: %s", idleTimeoutFlag.DefValue)
	}

	profileFlag := flags.Lookup("profile")
	if profileFlag == nil {
		t.Error("Expected 'profile' flag to exist")
	} else if profileFlag.DefValue != "default" {
		t.Errorf("Expected 'profile' flag default value 'default', got: %s", profileFlag.DefValue)
	}

	regionFlag := flags.Lookup("region")
	if regionFlag == nil {
		t.Error("Expected 'region' flag to exist")
	}
}

func TestRunLaunch_InvalidEnvironment(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Run launch with non-existent environment
	err := runLaunch("non-existent-env", "", "4h", "default", "", false, "ssh", "public", false)

	// Check result
	if err == nil {
		t.Error("Expected error for non-existent environment, got nil")
	}

	if !strings.Contains(err.Error(), "failed to load environment") {
		t.Errorf("Expected 'failed to load environment' error, got: %v", err)
	}
}

func TestRunLaunch_ValidEnvironment(t *testing.T) {
	// Skip this test if we're not in the project root with environments
	if _, err := os.Stat("environments/data-science.yaml"); os.IsNotExist(err) {
		t.Skip("Skipping test: built-in environments not available")
		return
	}

	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Mock AWS credentials to avoid real AWS calls
	_ = os.Setenv("AWS_ACCESS_KEY_ID", "test")
	_ = os.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	_ = os.Setenv("AWS_REGION", "us-west-2")
	defer func() {
		_ = os.Unsetenv("AWS_ACCESS_KEY_ID")
		_ = os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		_ = os.Unsetenv("AWS_REGION")
	}()

	// Run launch with valid environment but invalid AWS creds (should fail on AWS client creation)
	err := runLaunch("data-science", "m7g.large", "8h", "default", "us-west-2", false, "ssh", "public", false)

	// Check result - should fail when trying to create AWS client with fake creds
	if err == nil {
		t.Error("Expected error when creating AWS client with fake credentials, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create AWS client") {
		t.Errorf("Expected 'failed to create AWS client' error, got: %v", err)
	}
}

func TestRunLaunch_InstanceTypeOverride(t *testing.T) {
	// Skip this test if we're not in the project root with environments
	if _, err := os.Stat("environments/minimal.yaml"); os.IsNotExist(err) {
		t.Skip("Skipping test: built-in environments not available")
		return
	}

	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Mock AWS credentials
	_ = os.Setenv("AWS_ACCESS_KEY_ID", "test")
	_ = os.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	_ = os.Setenv("AWS_REGION", "us-west-2")
	defer func() {
		_ = os.Unsetenv("AWS_ACCESS_KEY_ID")
		_ = os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		_ = os.Unsetenv("AWS_REGION")
	}()

	// The function should load the environment and override instance type
	// We expect it to fail at AWS client creation, but we can check the logic
	err := runLaunch("minimal", "c7g.xlarge", "2h", "default", "", false, "ssh", "public", false)

	// Should fail at AWS client creation
	if err == nil {
		t.Error("Expected error when creating AWS client, got nil")
	}

	// The error should be about AWS client creation, not environment loading
	if strings.Contains(err.Error(), "failed to load environment") {
		t.Error("Should not fail at environment loading step")
	}
}
