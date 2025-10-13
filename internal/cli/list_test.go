package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/scttfrdmn/aws-jupyter/internal/config"
)

func TestFormatDuration(t *testing.T) {
	now := time.Now()

	tests := []struct {
		start    time.Time
		expected string
	}{
		{now.Add(-2*time.Hour - 30*time.Minute), "2h30m"},
		{now.Add(-1 * time.Hour), "1h0m"},
		{now.Add(-45 * time.Minute), "0h45m"},
		{now.Add(-90 * time.Minute), "1h30m"},
		{now.Add(-3*time.Hour - 15*time.Minute), "3h15m"},
	}

	for _, test := range tests {
		result := formatDuration(test.start)
		if result != test.expected {
			t.Errorf("formatDuration(%v) = %q, expected %q", test.start, result, test.expected)
		}
	}
}

func TestRunList_NoInstances(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run list command
	err := runList()

	// Restore stdout
	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close pipe: %v", err)
	}
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}
	output := buf.String()

	// Check result
	if err != nil {
		t.Fatalf("runList failed: %v", err)
	}

	if !strings.Contains(output, "No instances found") {
		t.Errorf("Expected 'No instances found' in output, got: %s", output)
	}
}

func TestRunList_WithInstances(t *testing.T) {
	setupTestEnvironment(t)
	state := createTestStateWithInstances()

	// Save state
	err := state.Save()
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	output, err := captureListOutput()
	if err != nil {
		t.Fatalf("runList failed: %v", err)
	}

	validateListOutput(t, output)
}

func setupTestEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", originalHome) })

	configDir := config.GetConfigDir()
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
}

func createTestStateWithInstances() *config.LocalState {
	return &config.LocalState{
		Instances: map[string]*config.Instance{
			"i-123456789": {
				ID:           "i-123456789",
				Environment:  "data-science",
				InstanceType: "m7g.medium",
				PublicIP:     "1.2.3.4",
				KeyPair:      "test-key",
				LaunchedAt:   time.Now().Add(-2 * time.Hour),
				IdleTimeout:  "4h",
				TunnelPID:    1234,
				Region:       "us-west-2",
			},
			"i-987654321": {
				ID:           "i-987654321",
				Environment:  "ml-pytorch",
				InstanceType: "m7g.large",
				PublicIP:     "5.6.7.8",
				KeyPair:      "test-key",
				LaunchedAt:   time.Now().Add(-1 * time.Hour),
				IdleTimeout:  "8h",
				TunnelPID:    0, // No tunnel
				Region:       "us-east-1",
			},
		},
		KeyPairs: map[string]string{
			"test-key": "/path/to/key",
		},
	}
}

func captureListOutput() (string, error) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList()

	if closeErr := w.Close(); closeErr != nil {
		return "", closeErr
	}
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		return "", readErr
	}
	return buf.String(), err
}

func validateListOutput(t *testing.T, output string) {
	// Check for header (tabwriter may format tabs differently)
	if !strings.Contains(output, "ID") || !strings.Contains(output, "ENV") || !strings.Contains(output, "TYPE") {
		t.Errorf("Expected header in output, got: %q", output)
	}

	// Check instance 1 details
	validateInstanceInOutput(t, output, "i-123456789", "data-science", "m7g.medium")

	// Check tunnel info for instance with tunnel
	if !strings.Contains(output, ":8888") {
		t.Error("Expected tunnel info :8888 for instance with tunnel")
	}

	// Check instance 2 details
	validateInstanceInOutput(t, output, "i-987654321", "ml-pytorch", "m7g.large")

	// Check that uptime is formatted correctly
	if !strings.Contains(output, "h") || !strings.Contains(output, "m") {
		t.Error("Expected formatted uptime in output")
	}
}

func validateInstanceInOutput(t *testing.T, output, instanceID, environment, instanceType string) {
	if !strings.Contains(output, instanceID) {
		t.Errorf("Expected instance %s in output", instanceID)
	}
	if !strings.Contains(output, environment) {
		t.Errorf("Expected environment %s in output", environment)
	}
	if !strings.Contains(output, instanceType) {
		t.Errorf("Expected instance type %s in output", instanceType)
	}
}

func TestRunList_StateLoadError(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create config directory
	configDir := config.GetConfigDir()
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create invalid state file
	stateFile := configDir + "/state.json"
	err = os.WriteFile(stateFile, []byte("invalid json"), 0600)
	if err != nil {
		t.Fatalf("Failed to create invalid state file: %v", err)
	}

	// Run list command
	err = runList()
	if err == nil {
		t.Error("Expected error when state file is invalid, got nil")
	}

	if !strings.Contains(err.Error(), "failed to load state") {
		t.Errorf("Expected 'failed to load state' error, got: %v", err)
	}
}
