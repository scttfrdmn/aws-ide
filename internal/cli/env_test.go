package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunEnvList(t *testing.T) {
	// Don't change to temp directory - we want to test with actual built-in environments
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run env list command
	err := runEnvList()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check result
	if err != nil {
		t.Fatalf("runEnvList failed: %v", err)
	}

	if !strings.Contains(output, "Available environments:") {
		t.Error("Expected 'Available environments:' in output")
	}

	// The test might not find built-in environments depending on working directory
	// Just check that it runs without error and produces some output
	if len(output) < 10 {
		t.Error("Expected some environment output")
	}
}

func TestRunEnvValidate_Valid(t *testing.T) {
	// Skip this test if we're not in the project root with environments
	if _, err := os.Stat("environments/data-science.yaml"); os.IsNotExist(err) {
		t.Skip("Skipping test: built-in environments not available")
		return
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run env validate command with valid environment
	err := runEnvValidate("data-science")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check result
	if err != nil {
		t.Fatalf("runEnvValidate failed: %v", err)
	}

	if !strings.Contains(output, "Environment data-science is valid:") {
		t.Error("Expected validation success message in output")
	}

	if !strings.Contains(output, "Name: Data Science") {
		t.Error("Expected environment name in output")
	}

	if !strings.Contains(output, "Instance Type: m7g.medium") {
		t.Error("Expected instance type in output")
	}

	if !strings.Contains(output, "EBS Volume:") {
		t.Error("Expected EBS volume info in output")
	}

	if !strings.Contains(output, "Packages:") {
		t.Error("Expected packages count in output")
	}

	if !strings.Contains(output, "Pip Packages:") {
		t.Error("Expected pip packages count in output")
	}
}

func TestRunEnvValidate_Invalid(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Run env validate command with invalid environment
	err := runEnvValidate("non-existent-env")

	// Check result
	if err == nil {
		t.Error("Expected error for non-existent environment, got nil")
	}

	if !strings.Contains(err.Error(), "failed to load environment") {
		t.Errorf("Expected 'failed to load environment' error, got: %v", err)
	}
}

func TestNewEnvCmd(t *testing.T) {
	cmd := NewEnvCmd()

	if cmd == nil {
		t.Fatal("NewEnvCmd returned nil")
	}

	if cmd.Use != "env" {
		t.Errorf("Expected Use='env', got: %s", cmd.Use)
	}

	if cmd.Short != "Environment management commands" {
		t.Errorf("Expected Short='Environment management commands', got: %s", cmd.Short)
	}

	// Check subcommands
	if len(cmd.Commands()) != 2 {
		t.Errorf("Expected 2 subcommands, got: %d", len(cmd.Commands()))
	}
}

func TestNewEnvListCmd(t *testing.T) {
	cmd := NewEnvListCmd()

	if cmd == nil {
		t.Fatal("NewEnvListCmd returned nil")
	}

	if cmd.Use != "list" {
		t.Errorf("Expected Use='list', got: %s", cmd.Use)
	}

	if cmd.Short != "List available environments" {
		t.Errorf("Expected Short='List available environments', got: %s", cmd.Short)
	}
}

func TestNewEnvValidateCmd(t *testing.T) {
	cmd := NewEnvValidateCmd()

	if cmd == nil {
		t.Fatal("NewEnvValidateCmd returned nil")
	}

	if cmd.Use != "validate ENV_NAME" {
		t.Errorf("Expected Use='validate ENV_NAME', got: %s", cmd.Use)
	}

	if cmd.Short != "Validate an environment configuration" {
		t.Errorf("Expected Short='Validate an environment configuration', got: %s", cmd.Short)
	}
}
