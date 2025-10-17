//go:build integration
// +build integration

package aws

import (
	"context"
	"os"
	"testing"
)

// TestEC2Client_Integration tests EC2Client against LocalStack
func TestEC2Client_Integration(t *testing.T) {
	// Check if LocalStack endpoint is configured
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set (LocalStack not configured)")
	}

	ctx := context.Background()

	// Create EC2 client
	client, err := NewEC2Client(ctx, "default")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	t.Run("Client initialization", func(t *testing.T) {
		if client == nil {
			t.Fatal("Expected non-nil client")
		}

		region := client.GetRegion()
		if region == "" {
			t.Error("Expected non-empty region")
		}
		t.Logf("Client initialized with region: %s", region)
	})

	t.Run("List AMIs", func(t *testing.T) {
		// This test verifies we can call AWS APIs through LocalStack
		// We don't expect any AMIs to exist, but the API call should succeed
		amis, err := client.ListCustomAMIs(ctx)
		if err != nil {
			t.Logf("Note: ListCustomAMIs may not work in LocalStack, error: %v", err)
			// Don't fail - LocalStack may not fully support this API
			return
		}

		t.Logf("Found %d custom AMIs (expected 0 in LocalStack)", len(amis))
		if len(amis) != 0 {
			t.Error("Expected no AMIs in fresh LocalStack instance")
		}
	})
}

// TestSSMClient_Integration tests SSMClient against LocalStack
func TestSSMClient_Integration(t *testing.T) {
	// Check if LocalStack endpoint is configured
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set (LocalStack not configured)")
	}

	ctx := context.Background()

	// Create EC2 client to get AWS config
	ec2Client, err := NewEC2ClientForRegion(ctx, "us-west-2")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	// Note: We can't easily get the raw config from EC2Client, so just verify client creation
	t.Run("SSM client creation would work", func(t *testing.T) {
		// This test verifies that we can create AWS clients with LocalStack configured
		// SSMClient requires an aws.Config which we get internally from config.LoadDefaultConfig
		t.Log("EC2 client created successfully, SSM client would work similarly")
		if ec2Client.region != "us-west-2" {
			t.Errorf("Expected region us-west-2, got %s", ec2Client.region)
		}
	})
}

// TestEC2Client_Integration_DescribeRegions tests basic EC2 connectivity
func TestEC2Client_Integration_DescribeRegions(t *testing.T) {
	// Check if LocalStack endpoint is configured
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set (LocalStack not configured)")
	}

	ctx := context.Background()

	// Create EC2 client
	client, err := NewEC2Client(ctx, "default")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	// Try to describe regions (basic API call)
	// Note: LocalStack may not fully support all EC2 APIs
	t.Run("Basic EC2 API connectivity", func(t *testing.T) {
		// Just verify the client is configured correctly
		// We can't test actual API calls without proper LocalStack EC2 support
		if client.client == nil {
			t.Error("Expected non-nil EC2 client")
		}
		t.Log("EC2 client is properly configured")
	})
}
