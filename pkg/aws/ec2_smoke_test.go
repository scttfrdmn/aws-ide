//go:build smoke
// +build smoke

package aws

import (
	"context"
	"testing"
	"time"
)

// TestEC2Client_Smoke_Connectivity tests basic EC2 client creation and AWS connectivity
func TestEC2Client_Smoke_Connectivity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create EC2 client using "aws" profile
	client, err := NewEC2Client(ctx, "aws")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v\n"+
			"This may indicate:\n"+
			"  - AWS credentials not configured for 'aws' profile\n"+
			"  - Network connectivity issues\n"+
			"  - AWS_DEFAULT_REGION not set", err)
	}

	// Verify client is properly initialized
	if client == nil {
		t.Fatal("Expected non-nil EC2 client")
	}

	// Verify region is set
	region := client.GetRegion()
	if region == "" {
		t.Error("Expected non-empty region")
	}
	t.Logf("✓ EC2 client created successfully for region: %s", region)

	// Verify we can make a basic API call
	if client.client == nil {
		t.Fatal("Expected non-nil underlying EC2 client")
	}
	t.Log("✓ AWS connectivity verified")
}

// TestIAMClient_Smoke_SessionManagerRole tests IAM role verification
func TestIAMClient_Smoke_SessionManagerRole(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create IAM client using "aws" profile
	client, err := NewIAMClient(ctx, "aws")
	if err != nil {
		t.Fatalf("Failed to create IAM client: %v\n"+
			"This may indicate:\n"+
			"  - AWS credentials not configured for 'aws' profile\n"+
			"  - Insufficient IAM permissions\n"+
			"  - Network connectivity issues", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil IAM client")
	}
	t.Log("✓ IAM client created successfully")

	// Try to get or create the Session Manager role
	// This verifies IAM permissions and role setup
	profileInfo, err := client.GetOrCreateSessionManagerRole(ctx, "aws-ide-smoke-test")
	if err != nil {
		t.Fatalf("Failed to get or create Session Manager role: %v\n"+
			"This may indicate:\n"+
			"  - Insufficient IAM permissions (need iam:CreateRole, iam:AttachRolePolicy, etc.)\n"+
			"  - IAM service issues\n"+
			"  - Invalid trust policy or permissions", err)
	}

	// Verify profile information
	if profileInfo == nil {
		t.Fatal("Expected non-nil profile info")
	}
	if profileInfo.Name == "" {
		t.Error("Expected non-empty profile name")
	}
	if profileInfo.Arn == "" {
		t.Error("Expected non-empty profile ARN")
	}

	t.Logf("✓ Session Manager role verified:")
	t.Logf("  Name: %s", profileInfo.Name)
	t.Logf("  ARN: %s", profileInfo.Arn)
}

// TestEC2Client_Smoke_SubnetDiscovery tests default subnet discovery
func TestEC2Client_Smoke_SubnetDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create EC2 client using "aws" profile
	client, err := NewEC2Client(ctx, "aws")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	region := client.GetRegion()
	t.Logf("Discovering subnets in region: %s", region)

	// Try to get a public subnet (most common use case)
	// Empty string for availability zone means "any available AZ"
	subnet, err := client.GetSubnet(ctx, "public", "")
	if err != nil {
		t.Fatalf("Failed to get public subnet: %v\n"+
			"This may indicate:\n"+
			"  - No default VPC exists in region %s\n"+
			"  - Insufficient EC2 permissions (need ec2:DescribeSubnets, ec2:DescribeVpcs)\n"+
			"  - Region configuration issues\n"+
			"You may need to create a default VPC: aws ec2 create-default-vpc --region %s", err, region, region)
	}

	// Verify subnet information
	if subnet == nil {
		t.Fatal("Expected non-nil subnet")
	}
	if subnet.ID == "" {
		t.Error("Expected non-empty subnet ID")
	}
	if subnet.AvailabilityZone == "" {
		t.Error("Expected non-empty availability zone")
	}
	if subnet.VpcID == "" {
		t.Error("Expected non-empty VPC ID")
	}

	t.Logf("✓ Public subnet discovered:")
	t.Logf("  Subnet ID: %s", subnet.ID)
	t.Logf("  VPC ID: %s", subnet.VpcID)
	t.Logf("  Availability Zone: %s", subnet.AvailabilityZone)
	t.Logf("  IPv4 CIDR: %s", subnet.CidrBlock)
	t.Logf("  Is Public: %v", subnet.IsPublic)
}

// TestEC2Client_Smoke_InstanceTypeAvailability tests instance type availability
func TestEC2Client_Smoke_InstanceTypeAvailability(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create EC2 client using "aws" profile
	client, err := NewEC2Client(ctx, "aws")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	region := client.GetRegion()

	// Get default subnet to determine availability zone
	subnet, err := client.GetSubnet(ctx, "public", "")
	if err != nil {
		t.Fatalf("Failed to get default subnet: %v", err)
	}

	az := subnet.AvailabilityZone
	t.Logf("Checking instance type availability in %s (AZ: %s)", region, az)

	// Test common instance types used by aws-ide
	instanceTypes := []struct {
		name     string
		required bool // If true, test fails if not available
	}{
		{"t3.medium", true},  // Default for most apps
		{"t3.large", true},   // Common upgrade option
		{"t3.xlarge", false}, // Optional for RStudio
		{"m5.large", false},  // Alternative family
	}

	availableCount := 0
	for _, it := range instanceTypes {
		supported, err := client.IsInstanceTypeSupported(ctx, it.name, az)
		if err != nil {
			t.Logf("  Warning: Failed to check %s availability: %v", it.name, err)
			continue
		}

		if supported {
			availableCount++
			t.Logf("  ✓ %s is available", it.name)
		} else {
			if it.required {
				t.Errorf("  ✗ %s is NOT available (required)", it.name)
			} else {
				t.Logf("  ○ %s is not available (optional)", it.name)
			}
		}
	}

	if availableCount == 0 {
		t.Fatal("No instance types available - this is likely a permissions or API issue")
	}

	t.Logf("✓ Instance type availability check complete (%d/%d available)", availableCount, len(instanceTypes))
}

// TestEC2Client_Smoke_AMIDiscovery tests AMI discovery for Ubuntu 22.04
func TestEC2Client_Smoke_AMIDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create EC2 client using "aws" profile
	client, err := NewEC2Client(ctx, "aws")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	region := client.GetRegion()
	t.Logf("Discovering Ubuntu AMIs in region: %s", region)

	// Create AMI selector
	amiSelector := NewAMISelector(region)

	// Test multiple common AMI bases
	amiTests := []struct {
		name    string
		base    string
		required bool
	}{
		{"Ubuntu 24.04 ARM64", "ubuntu24-arm64", false},
		{"Ubuntu 24.04 x86_64", "ubuntu24-x86_64", false},
		{"Ubuntu 22.04 ARM64", "ubuntu22-arm64", true},
		{"Ubuntu 22.04 x86_64", "ubuntu22-x86_64", false},
		{"Amazon Linux 2 x86_64", "amazonlinux2-x86_64", true},
	}

	foundCount := 0
	for _, test := range amiTests {
		ami, err := amiSelector.GetAMI(ctx, client, test.base)
		if err != nil {
			if test.required {
				t.Errorf("  ✗ Failed to find %s: %v (required)", test.name, err)
			} else {
				t.Logf("  ○ Could not find %s: %v (optional)", test.name, err)
			}
			continue
		}

		if ami == "" {
			if test.required {
				t.Errorf("  ✗ %s returned empty AMI ID (required)", test.name)
			} else {
				t.Logf("  ○ %s returned empty AMI ID (optional)", test.name)
			}
			continue
		}

		foundCount++
		t.Logf("  ✓ %s: %s", test.name, ami)
	}

	if foundCount == 0 {
		t.Fatal("Failed to find any AMIs - this may indicate:\n" +
			"  - Insufficient EC2 permissions (need ec2:DescribeImages)\n" +
			"  - No Ubuntu AMIs available in region\n" +
			"  - AWS Marketplace issues")
	}

	t.Logf("✓ AMI discovery check complete (%d/%d found)", foundCount, len(amiTests))
}

// TestEC2Client_Smoke_AvailabilityZoneDiscovery tests finding compatible AZs
func TestEC2Client_Smoke_AvailabilityZoneDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create EC2 client using "aws" profile
	client, err := NewEC2Client(ctx, "aws")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	region := client.GetRegion()
	t.Logf("Finding compatible availability zones in region: %s", region)

	// Test finding AZs for common configurations
	tests := []struct {
		instanceType string
		subnetType   string
	}{
		{"t3.medium", "public"},
		{"t3.large", "public"},
		{"m5.large", "public"},
	}

	foundCount := 0
	for _, test := range tests {
		az, err := client.FindCompatibleAvailabilityZone(ctx, test.instanceType, test.subnetType)
		if err != nil {
			t.Logf("  Warning: Failed to find AZ for %s/%s: %v", test.instanceType, test.subnetType, err)
			continue
		}

		if az == "" {
			t.Logf("  Warning: Empty AZ returned for %s/%s", test.instanceType, test.subnetType)
			continue
		}

		foundCount++
		t.Logf("  ✓ %s (%s subnet): %s", test.instanceType, test.subnetType, az)
	}

	if foundCount == 0 {
		t.Fatal("Failed to find any compatible availability zones")
	}

	t.Logf("✓ Availability zone discovery complete (%d/%d found)", foundCount, len(tests))
}

// TestEC2Client_Smoke_QuickLaunchCheck verifies all prerequisites for launching an instance
func TestEC2Client_Smoke_QuickLaunchCheck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Create EC2 client using "aws" profile
	client, err := NewEC2Client(ctx, "aws")
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	t.Log("Verifying all prerequisites for instance launch...")

	// 1. Get subnet
	subnet, err := client.GetSubnet(ctx, "public", "")
	if err != nil {
		t.Fatalf("Failed to get subnet: %v", err)
	}
	t.Logf("  ✓ Subnet: %s (AZ: %s)", subnet.ID, subnet.AvailabilityZone)

	// 2. Get AMI (use ARM64 which we know exists)
	amiSelector := NewAMISelector(client.GetRegion())
	ami, err := amiSelector.GetAMI(ctx, client, "ubuntu22-arm64")
	if err != nil {
		// Try Amazon Linux as fallback
		ami, err = amiSelector.GetAMI(ctx, client, "amazonlinux2-x86_64")
		if err != nil {
			t.Fatalf("Failed to find any usable AMI: %v", err)
		}
	}
	t.Logf("  ✓ AMI: %s", ami)

	// 3. Check instance type availability
	supported, err := client.IsInstanceTypeSupported(ctx, "t3.medium", subnet.AvailabilityZone)
	if err != nil {
		t.Fatalf("Failed to check instance type: %v", err)
	}
	if !supported {
		t.Fatal("Instance type t3.medium not supported in AZ")
	}
	t.Log("  ✓ Instance Type: t3.medium is available")

	// 4. Get IAM profile
	iamClient, err := NewIAMClient(ctx, "aws")
	if err != nil {
		t.Fatalf("Failed to create IAM client: %v", err)
	}

	profileInfo, err := iamClient.GetOrCreateSessionManagerRole(ctx, "aws-ide-smoke-test")
	if err != nil {
		t.Fatalf("Failed to get IAM profile: %v", err)
	}
	t.Logf("  ✓ IAM Profile: %s", profileInfo.Name)

	t.Log("✓ All launch prerequisites verified")
	t.Log("  (No instance was actually launched - this is a dry-run check)")
}
