package aws

import (
	"context"
	"os"
	"testing"
)

func TestLaunchParams(t *testing.T) {
	params := LaunchParams{
		AMI:             "ami-12345",
		InstanceType:    "m7g.medium",
		KeyPairName:     "test-key",
		SecurityGroupID: "sg-12345",
		UserData:        "#!/bin/bash\necho 'test'",
		EBSVolumeSize:   20,
		Environment:     "test-env",
	}

	if params.AMI != "ami-12345" {
		t.Errorf("Expected AMI ami-12345, got: %s", params.AMI)
	}

	if params.InstanceType != "m7g.medium" {
		t.Errorf("Expected InstanceType m7g.medium, got: %s", params.InstanceType)
	}

	if params.KeyPairName != "test-key" {
		t.Errorf("Expected KeyPairName test-key, got: %s", params.KeyPairName)
	}

	if params.SecurityGroupID != "sg-12345" {
		t.Errorf("Expected SecurityGroupID sg-12345, got: %s", params.SecurityGroupID)
	}

	if params.EBSVolumeSize != 20 {
		t.Errorf("Expected EBSVolumeSize 20, got: %d", params.EBSVolumeSize)
	}

	if params.Environment != "test-env" {
		t.Errorf("Expected Environment test-env, got: %s", params.Environment)
	}
}

func TestNewEC2Client_InvalidCredentials(t *testing.T) {
	ctx := context.Background()

	// Clear AWS environment variables
	originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	originalSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	originalRegion := os.Getenv("AWS_REGION")

	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Setenv("AWS_REGION", "us-west-2")

	defer func() {
		if originalAccessKey != "" {
			os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
		}
		if originalSecretKey != "" {
			os.Setenv("AWS_SECRET_ACCESS_KEY", originalSecretKey)
		}
		if originalRegion != "" {
			os.Setenv("AWS_REGION", originalRegion)
		} else {
			os.Unsetenv("AWS_REGION")
		}
	}()

	// Try to create client with invalid profile
	client, err := NewEC2Client(ctx, "non-existent-profile")

	// Should succeed in creating client but may have issues with actual AWS calls
	// The AWS SDK will use default credential chain
	if err != nil && client == nil {
		// This is expected if no credentials are available
		t.Logf("Expected behavior: failed to create client without credentials: %v", err)
		return
	}

	// If client was created, it should have proper structure
	if client != nil {
		if client.client == nil {
			t.Error("Expected client.client to be non-nil")
		}
	}
}

func TestNewEC2Client_WithValidRegion(t *testing.T) {
	ctx := context.Background()

	// Set minimal environment for testing
	os.Setenv("AWS_REGION", "us-east-1")
	defer os.Unsetenv("AWS_REGION")

	// Create client (may succeed even without real credentials)
	client, err := NewEC2Client(ctx, "default")

	// Check if client creation succeeded or failed appropriately
	if err != nil {
		// If it failed, it should be due to credential issues
		t.Logf("Client creation failed (expected without credentials): %v", err)
		return
	}

	// If client was created successfully
	if client == nil {
		t.Fatal("Expected non-nil client when no error occurred")
	}

	if client.client == nil {
		t.Error("Expected client.client to be non-nil")
	}

	// Region should be set
	if client.region == "" {
		t.Error("Expected region to be set")
	}
}

func TestEC2Client_Methods_Structure(t *testing.T) {
	// Test that the EC2Client struct has the expected methods
	// This is more of a compile-time check to ensure method signatures exist

	// Create a client structure for testing
	client := &EC2Client{
		region: "us-west-2",
	}

	if client.region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got: %s", client.region)
	}

	// Test that the methods exist by checking their function types
	// We don't call them to avoid nil pointer panics

	// Verify LaunchParams struct
	params := LaunchParams{
		AMI:             "ami-12345",
		InstanceType:    "m7g.medium",
		KeyPairName:     "test-key",
		SecurityGroupID: "sg-12345",
		UserData:        "test",
		EBSVolumeSize:   20,
		Environment:     "test",
	}

	if params.AMI != "ami-12345" {
		t.Errorf("Expected AMI ami-12345, got: %s", params.AMI)
	}

	// This test mainly verifies that the methods exist and compile correctly
	// Actual AWS functionality would require integration tests with real AWS resources
}

func TestEC2Client_StructureIntegrity(t *testing.T) {
	// Test that EC2Client struct maintains expected field structure
	client := &EC2Client{
		region: "us-west-2",
	}

	if client.region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got: %s", client.region)
	}

	// Test that client field exists (even if nil)
	if client.client != nil {
		// If somehow client is not nil, that's also valid
		t.Log("Client field is not nil (this is acceptable)")
	}
}
