package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scttfrdmn/aws-ide/pkg/aws"
	"github.com/scttfrdmn/aws-ide/pkg/config"
)

func TestCleanupStateFile_EmptyDeletedAMIs(t *testing.T) {
	// Setup: Create a temporary state file
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")

	// Save original state file path and restore after test
	originalStateFile := os.Getenv("AWS_IDE_STATE_FILE")
	os.Setenv("AWS_IDE_STATE_FILE", stateFile)
	defer func() {
		if originalStateFile != "" {
			os.Setenv("AWS_IDE_STATE_FILE", originalStateFile)
		} else {
			os.Unsetenv("AWS_IDE_STATE_FILE")
		}
	}()

	// Create initial state
	state := &config.LocalState{
		Instances: map[string]*config.Instance{
			"i-123": {
				ID:           "i-123",
				Environment:  "test",
				InstanceType: "t2.micro",
				Region:       "us-east-1",
				LaunchedAt:   time.Now(),
			},
		},
	}
	if err := state.Save(); err != nil {
		t.Fatalf("Failed to save initial state: %v", err)
	}

	// Test: Call cleanupStateFile with empty list
	cleanupStateFile([]aws.AMIInfo{})

	// Verify: State should be unchanged (function currently doesn't modify anything)
	loadedState, err := config.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if len(loadedState.Instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(loadedState.Instances))
	}
}

func TestCleanupStateFile_WithDeletedAMIs(t *testing.T) {
	// Setup: Create a temporary state file
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "state.json")

	originalStateFile := os.Getenv("AWS_IDE_STATE_FILE")
	os.Setenv("AWS_IDE_STATE_FILE", stateFile)
	defer func() {
		if originalStateFile != "" {
			os.Setenv("AWS_IDE_STATE_FILE", originalStateFile)
		} else {
			os.Unsetenv("AWS_IDE_STATE_FILE")
		}
	}()

	// Create initial state with instances
	state := &config.LocalState{
		Instances: map[string]*config.Instance{
			"i-123": {
				ID:           "i-123",
				Environment:  "test",
				InstanceType: "t2.micro",
				Region:       "us-east-1",
				LaunchedAt:   time.Now(),
			},
			"i-456": {
				ID:           "i-456",
				Environment:  "prod",
				InstanceType: "t3.small",
				Region:       "us-west-2",
				LaunchedAt:   time.Now(),
			},
		},
	}
	if err := state.Save(); err != nil {
		t.Fatalf("Failed to save initial state: %v", err)
	}

	// Test: Call cleanupStateFile with some deleted AMIs
	deletedAMIs := []aws.AMIInfo{
		{
			ID:           "ami-123",
			Name:         "test-ami",
			State:        "available",
			CreationDate: time.Now(),
		},
		{
			ID:           "ami-456",
			Name:         "prod-ami",
			State:        "available",
			CreationDate: time.Now(),
		},
	}
	cleanupStateFile(deletedAMIs)

	// Verify: State should be unchanged (function currently doesn't track AMI IDs)
	// This is testing the current behavior; when AMI tracking is added, this test should be updated
	loadedState, err := config.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if len(loadedState.Instances) != 2 {
		t.Errorf("Expected 2 instances, got %d (function doesn't track AMI IDs yet)", len(loadedState.Instances))
	}
}

func TestCleanupStateFile_StateFileNotFound(t *testing.T) {
	// Setup: Point to non-existent state file
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "nonexistent.json")

	originalStateFile := os.Getenv("AWS_IDE_STATE_FILE")
	os.Setenv("AWS_IDE_STATE_FILE", stateFile)
	defer func() {
		if originalStateFile != "" {
			os.Setenv("AWS_IDE_STATE_FILE", originalStateFile)
		} else {
			os.Unsetenv("AWS_IDE_STATE_FILE")
		}
	}()

	// Test: Should not panic when state file doesn't exist
	deletedAMIs := []aws.AMIInfo{
		{
			ID:           "ami-123",
			Name:         "test-ami",
			State:        "available",
			CreationDate: time.Now(),
		},
	}

	// Should not panic
	cleanupStateFile(deletedAMIs)

	// No assertion needed - just verifying no panic
}

func TestCleanupStateFile_InvalidStateFile(t *testing.T) {
	// Setup: Create an invalid JSON state file
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "invalid.json")

	originalStateFile := os.Getenv("AWS_IDE_STATE_FILE")
	os.Setenv("AWS_IDE_STATE_FILE", stateFile)
	defer func() {
		if originalStateFile != "" {
			os.Setenv("AWS_IDE_STATE_FILE", originalStateFile)
		} else {
			os.Unsetenv("AWS_IDE_STATE_FILE")
		}
	}()

	// Write invalid JSON
	if err := os.WriteFile(stateFile, []byte("invalid json{"), 0644); err != nil {
		t.Fatalf("Failed to write invalid state file: %v", err)
	}

	// Test: Should handle invalid JSON gracefully
	deletedAMIs := []aws.AMIInfo{
		{
			ID:           "ami-123",
			Name:         "test-ami",
			State:        "available",
			CreationDate: time.Now(),
		},
	}

	// Should not panic
	cleanupStateFile(deletedAMIs)

	// No assertion needed - just verifying no panic
}
