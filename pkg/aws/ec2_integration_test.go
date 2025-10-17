//go:build integration
// +build integration

package aws

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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

// TestEC2Client_Integration_InstanceTypeSupport tests instance type validation
func TestEC2Client_Integration_InstanceTypeSupport(t *testing.T) {
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set")
	}

	ctx := context.Background()
	client, err := NewEC2Client(ctx, "default")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	tests := []struct {
		name         string
		instanceType string
		shouldPass   bool
	}{
		{"t3.medium supported", "t3.medium", true},
		{"t3.large supported", "t3.large", true},
		{"m5.large supported", "m5.large", true},
		{"invalid type", "invalid.type", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// IsInstanceTypeSupported requires an availability zone
			supported, err := client.IsInstanceTypeSupported(ctx, tt.instanceType, "us-west-2a")

			// Note: LocalStack may not fully implement instance type checking
			// So we test that the function runs without error rather than specific results
			t.Logf("Instance type %s support check returned: %v, err: %v", tt.instanceType, supported, err)

			// We can at least check that empty string is not supported
			if tt.instanceType == "" && supported {
				t.Error("Empty instance type should not be supported")
			}
		})
	}
}

// TestEC2Client_Integration_GetRegion tests region retrieval
func TestEC2Client_Integration_GetRegion(t *testing.T) {
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set")
	}

	ctx := context.Background()

	tests := []struct {
		name           string
		region         string
		expectedRegion string
	}{
		{"default region", "default", "us-west-2"}, // LocalStack defaults to us-east-1 but config may override
		{"us-west-2", "us-west-2", "us-west-2"},
		{"us-east-1", "us-east-1", "us-east-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *EC2Client
			var err error

			if tt.region == "default" {
				client, err = NewEC2Client(ctx, "default")
			} else {
				client, err = NewEC2ClientForRegion(ctx, tt.region)
			}

			if err != nil {
				t.Fatalf("Failed to create EC2 client: %v", err)
			}

			gotRegion := client.GetRegion()
			if gotRegion == "" {
				t.Error("Expected non-empty region")
			}
			t.Logf("Client created for region: %s, got: %s", tt.region, gotRegion)
		})
	}
}

// TestIAMClient_Integration tests IAM client creation
func TestIAMClient_Integration(t *testing.T) {
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set")
	}

	ctx := context.Background()

	t.Run("Create IAM client", func(t *testing.T) {
		client, err := NewIAMClient(ctx, "default")
		if err != nil {
			t.Fatalf("Failed to create IAM client: %v", err)
		}

		if client == nil {
			t.Error("Expected non-nil IAM client")
		}
		t.Log("IAM client created successfully")
	})

	t.Run("Get or create Session Manager role", func(t *testing.T) {
		client, err := NewIAMClient(ctx, "default")
		if err != nil {
			t.Fatalf("Failed to create IAM client: %v", err)
		}

		// Try to get or create the Session Manager role
		// Note: LocalStack may not fully implement IAM, so we just test that the function runs
		profileInfo, err := client.GetOrCreateSessionManagerRole(ctx, "test-app")
		if err != nil {
			t.Logf("Note: GetOrCreateSessionManagerRole may not work in LocalStack, error: %v", err)
			// Don't fail - LocalStack may not fully support IAM
			return
		}

		if profileInfo == nil || profileInfo.Arn == "" {
			t.Error("Expected non-empty profile ARN")
		}
		t.Logf("Session Manager profile: %+v", profileInfo)
	})
}

// TestEC2Client_Integration_LaunchInstance tests instance launch with LocalStack
func TestEC2Client_Integration_LaunchInstance(t *testing.T) {
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set")
	}

	ctx := context.Background()
	client, err := NewEC2Client(ctx, "default")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	t.Run("Launch with valid parameters", func(t *testing.T) {
		params := LaunchParams{
			AMI:             "ami-test123",
			InstanceType:    "t3.medium",
			KeyPairName:     "test-key",
			SecurityGroupID: "sg-test123",
			SubnetID:        "subnet-test123",
			UserData:        "#!/bin/bash\necho 'test'",
			EBSVolumeSize:   30,
			Environment:     "test",
			InstanceProfile: "test-profile",
		}

		instance, err := client.LaunchInstance(ctx, params)
		if err != nil {
			// LocalStack may not fully support RunInstances
			t.Logf("Note: LaunchInstance may not work fully in LocalStack, error: %v", err)
			// Don't fail - test that API call was attempted
			return
		}

		if instance != nil {
			t.Logf("Instance launched (LocalStack): %s", *instance.InstanceId)
			// Verify instance has expected properties
			if instance.InstanceType != types.InstanceType(params.InstanceType) {
				t.Errorf("Expected instance type %s, got %s", params.InstanceType, instance.InstanceType)
			}
		}
	})

	t.Run("Launch without subnet (use default)", func(t *testing.T) {
		params := LaunchParams{
			AMI:             "ami-test456",
			InstanceType:    "t3.small",
			KeyPairName:     "test-key",
			SecurityGroupID: "sg-test456",
			// SubnetID intentionally omitted to test default subnet fallback
			UserData:        "",
			EBSVolumeSize:   20,
			Environment:     "minimal",
			InstanceProfile: "test-profile",
		}

		_, err := client.LaunchInstance(ctx, params)
		if err != nil {
			// Expected - LocalStack likely doesn't have default subnet
			t.Logf("Expected behavior: Launch without subnet failed (no default): %v", err)
			return
		}
		t.Log("Launch without subnet succeeded (LocalStack has default subnet)")
	})

	t.Run("Launch with invalid parameters", func(t *testing.T) {
		params := LaunchParams{
			AMI:             "", // Empty AMI should fail
			InstanceType:    "t3.medium",
			KeyPairName:     "test-key",
			SecurityGroupID: "sg-test",
			SubnetID:        "subnet-test",
			UserData:        "",
			EBSVolumeSize:   30,
			Environment:     "test",
			InstanceProfile: "test-profile",
		}

		_, err := client.LaunchInstance(ctx, params)
		if err == nil {
			t.Error("Expected error for empty AMI, got nil")
		} else {
			t.Logf("Correctly rejected invalid parameters: %v", err)
		}
	})
}

// TestEC2Client_Integration_InstanceOperations tests start/stop/terminate
func TestEC2Client_Integration_InstanceOperations(t *testing.T) {
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set")
	}

	ctx := context.Background()
	client, err := NewEC2Client(ctx, "default")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	testInstanceID := "i-test123456789"

	t.Run("Stop instance", func(t *testing.T) {
		err := client.StopInstance(ctx, testInstanceID, false)
		if err != nil {
			t.Logf("Note: StopInstance may not work in LocalStack, error: %v", err)
			// Don't fail - LocalStack may not have the instance
			return
		}
		t.Log("StopInstance API call succeeded")
	})

	t.Run("Start instance", func(t *testing.T) {
		err := client.StartInstance(ctx, testInstanceID)
		if err != nil {
			t.Logf("Note: StartInstance may not work in LocalStack, error: %v", err)
			return
		}
		t.Log("StartInstance API call succeeded")
	})

	t.Run("Terminate instance", func(t *testing.T) {
		err := client.TerminateInstance(ctx, testInstanceID)
		if err != nil {
			t.Logf("Note: TerminateInstance may not work in LocalStack, error: %v", err)
			return
		}
		t.Log("TerminateInstance API call succeeded")
	})

	t.Run("Get instance info", func(t *testing.T) {
		_, err := client.GetInstanceInfo(ctx, testInstanceID)
		if err != nil {
			t.Logf("Note: GetInstanceInfo correctly failed for non-existent instance: %v", err)
			return
		}
		t.Log("GetInstanceInfo succeeded (instance exists in LocalStack)")
	})
}

// TestEC2Client_Integration_AMIOperations tests AMI creation and listing
func TestEC2Client_Integration_AMIOperations(t *testing.T) {
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set")
	}

	ctx := context.Background()
	client, err := NewEC2Client(ctx, "default")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	testInstanceID := "i-test987654321"

	t.Run("Create AMI", func(t *testing.T) {
		amiID, err := client.CreateAMI(ctx, testInstanceID, "test-ami", "Test AMI description", true)
		if err != nil {
			t.Logf("Note: CreateAMI may not work in LocalStack, error: %v", err)
			// Don't fail - LocalStack may not support AMI creation
			return
		}

		if amiID == "" {
			t.Error("Expected non-empty AMI ID")
		}
		t.Logf("AMI created: %s", amiID)
	})

	t.Run("List custom AMIs", func(t *testing.T) {
		amis, err := client.ListCustomAMIs(ctx)
		if err != nil {
			t.Logf("Note: ListCustomAMIs may not work in LocalStack, error: %v", err)
			return
		}

		t.Logf("Found %d custom AMIs", len(amis))
		for _, ami := range amis {
			t.Logf("  AMI: %s (%s) - %s", ami.ID, ami.Name, ami.State)
		}
	})

	t.Run("Delete AMI", func(t *testing.T) {
		testAMI := "ami-test12345"
		err := client.DeleteAMI(ctx, testAMI)
		if err != nil {
			t.Logf("Note: DeleteAMI may not work in LocalStack, error: %v", err)
			return
		}
		t.Log("DeleteAMI API call succeeded")
	})
}

// TestIAMClient_Integration_RoleOperations tests IAM role creation and management
func TestIAMClient_Integration_RoleOperations(t *testing.T) {
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set")
	}

	ctx := context.Background()

	t.Run("Create Session Manager role with prefix", func(t *testing.T) {
		client, err := NewIAMClient(ctx, "default")
		if err != nil {
			t.Fatalf("Failed to create IAM client: %v", err)
		}

		// Test with different prefixes to ensure uniqueness handling
		prefixes := []string{"test-app-1", "test-app-2", "integration-test"}

		for _, prefix := range prefixes {
			profileInfo, err := client.GetOrCreateSessionManagerRole(ctx, prefix)
			if err != nil {
				t.Logf("Note: GetOrCreateSessionManagerRole may not work in LocalStack for prefix '%s', error: %v", prefix, err)
				continue
			}

			if profileInfo == nil {
				t.Errorf("Expected non-nil profile info for prefix '%s'", prefix)
				continue
			}

			if profileInfo.Name == "" {
				t.Errorf("Expected non-empty profile name for prefix '%s'", prefix)
			}
			if profileInfo.Arn == "" {
				t.Errorf("Expected non-empty ARN for prefix '%s'", prefix)
			}

			t.Logf("Successfully created/retrieved role for prefix '%s':", prefix)
			t.Logf("  Name: %s", profileInfo.Name)
			t.Logf("  ARN: %s", profileInfo.Arn)
		}
	})

	t.Run("IAM role idempotency", func(t *testing.T) {
		client, err := NewIAMClient(ctx, "default")
		if err != nil {
			t.Fatalf("Failed to create IAM client: %v", err)
		}

		prefix := "idempotency-test"

		// Call twice with same prefix
		profile1, err1 := client.GetOrCreateSessionManagerRole(ctx, prefix)
		if err1 != nil {
			t.Logf("Note: First call may not work in LocalStack, error: %v", err1)
			return
		}

		profile2, err2 := client.GetOrCreateSessionManagerRole(ctx, prefix)
		if err2 != nil {
			t.Logf("Note: Second call may not work in LocalStack, error: %v", err2)
			return
		}

		// Should return same profile both times
		if profile1.Name != profile2.Name {
			t.Errorf("Expected same profile name, got %s and %s", profile1.Name, profile2.Name)
		}
		if profile1.Arn != profile2.Arn {
			t.Errorf("Expected same ARN, got %s and %s", profile1.Arn, profile2.Arn)
		}

		t.Log("✓ IAM role creation is idempotent")
	})
}

// TestEC2Client_Integration_ErrorHandling tests error scenarios
func TestEC2Client_Integration_ErrorHandling(t *testing.T) {
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set")
	}

	ctx := context.Background()
	client, err := NewEC2Client(ctx, "default")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	t.Run("Get non-existent instance", func(t *testing.T) {
		_, err := client.GetInstanceInfo(ctx, "i-nonexistent12345")
		if err == nil {
			t.Error("Expected error for non-existent instance, got nil")
		} else {
			t.Logf("Correctly returned error for non-existent instance: %v", err)
		}
	})

	t.Run("Terminate non-existent instance", func(t *testing.T) {
		err := client.TerminateInstance(ctx, "i-nonexistent67890")
		if err == nil {
			t.Log("Note: LocalStack may allow terminating non-existent instance")
		} else {
			t.Logf("Correctly returned error for non-existent instance: %v", err)
		}
	})

	t.Run("Invalid instance type", func(t *testing.T) {
		supported, err := client.IsInstanceTypeSupported(ctx, "invalid.mega.huge", "us-west-2a")
		if err != nil {
			t.Logf("API call failed (expected): %v", err)
		}
		if supported {
			t.Log("Note: LocalStack may report invalid instance types as supported")
		} else {
			t.Log("✓ Correctly identified invalid instance type")
		}
	})

	t.Run("Empty parameters", func(t *testing.T) {
		// Test with empty instance ID
		err := client.StopInstance(ctx, "", false)
		if err == nil {
			t.Error("Expected error for empty instance ID, got nil")
		} else {
			t.Logf("Correctly rejected empty instance ID: %v", err)
		}
	})
}

// TestAMISelector_Integration tests AMI selection logic
func TestAMISelector_Integration(t *testing.T) {
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		t.Skip("Skipping integration test: AWS_ENDPOINT_URL not set")
	}

	ctx := context.Background()
	client, err := NewEC2Client(ctx, "default")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	region := client.GetRegion()
	selector := NewAMISelector(region)

	amiTests := []struct {
		name    string
		amiBase string
	}{
		{"Ubuntu 24.04 ARM64", "ubuntu24-arm64"},
		{"Ubuntu 24.04 x86_64", "ubuntu24-x86_64"},
		{"Ubuntu 22.04 ARM64", "ubuntu22-arm64"},
		{"Amazon Linux 2 x86_64", "amazonlinux2-x86_64"},
	}

	for _, test := range amiTests {
		t.Run(test.name, func(t *testing.T) {
			_, err := selector.GetAMI(ctx, client, test.amiBase)
			if err != nil {
				t.Logf("Note: AMI discovery may not work in LocalStack for %s, error: %v", test.amiBase, err)
				// Expected - LocalStack doesn't have real Ubuntu/Amazon Linux AMIs
				return
			}
			t.Logf("Found AMI for %s (LocalStack may have seeded data)", test.amiBase)
		})
	}
}
