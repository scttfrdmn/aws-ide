package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// AMISelector handles AMI selection based on region and architecture
type AMISelector struct {
	region string
}

// NewAMISelector creates a new AMI selector for the given region
func NewAMISelector(region string) *AMISelector {
	return &AMISelector{region: region}
}

// GetAMI returns the appropriate AMI ID for the given base configuration
func (a *AMISelector) GetAMI(ctx context.Context, client *EC2Client, amiBase string) (string, error) {
	// Map of AMI base names to Ubuntu versions and architectures
	switch amiBase {
	case "ubuntu24-arm64":
		return a.findUbuntuAMI(ctx, client, "24.04", "arm64")
	case "ubuntu24-x86_64":
		return a.findUbuntuAMI(ctx, client, "24.04", "x86_64")
	case "ubuntu22-arm64":
		return a.findUbuntuAMI(ctx, client, "22.04", "arm64")
	case "ubuntu22-x86_64":
		return a.findUbuntuAMI(ctx, client, "22.04", "x86_64")
	case "ubuntu20-arm64":
		return a.findUbuntuAMI(ctx, client, "20.04", "arm64")
	case "ubuntu20-x86_64":
		return a.findUbuntuAMI(ctx, client, "20.04", "x86_64")
	case "amazonlinux2-arm64":
		return a.findAmazonLinuxAMI(ctx, client, "2", "arm64")
	case "amazonlinux2-x86_64":
		return a.findAmazonLinuxAMI(ctx, client, "2", "x86_64")
	default:
		// Default to Ubuntu 24.04 ARM64 for Graviton instances (LTS until 2029)
		return a.findUbuntuAMI(ctx, client, "24.04", "arm64")
	}
}

// findUbuntuAMI finds the latest Ubuntu AMI for the given version and architecture
func (a *AMISelector) findUbuntuAMI(ctx context.Context, client *EC2Client, version, arch string) (string, error) {
	// Ubuntu AMI name pattern using codenames
	codenames := map[string]string{
		"24.04": "noble",
		"22.04": "jammy",
		"20.04": "focal",
		"18.04": "bionic",
	}

	codename, ok := codenames[version]
	if !ok {
		return "", fmt.Errorf("unsupported Ubuntu version: %s", version)
	}

	namePattern := fmt.Sprintf("ubuntu/images/hvm-ssd/ubuntu-%s-%s-%s-server-*", codename, version, arch)

	var archType types.ArchitectureValues
	if arch == "arm64" {
		archType = types.ArchitectureValuesArm64
	} else {
		archType = types.ArchitectureValuesX8664
	}

	result, err := client.client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{namePattern},
			},
			{
				Name:   aws.String("architecture"),
				Values: []string{string(archType)},
			},
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
			{
				Name:   aws.String("owner-id"),
				Values: []string{"099720109477"}, // Canonical's AWS account
			},
		},
		Owners: []string{"099720109477"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query Ubuntu AMIs: %w", err)
	}

	if len(result.Images) == 0 {
		return "", fmt.Errorf("no Ubuntu %s %s AMIs found in region %s", version, arch, a.region)
	}

	// Find the most recent AMI by creation date
	var newestAMI *types.Image
	for i := range result.Images {
		img := &result.Images[i]
		if newestAMI == nil || aws.ToString(img.CreationDate) > aws.ToString(newestAMI.CreationDate) {
			newestAMI = img
		}
	}

	amiID := aws.ToString(newestAMI.ImageId)
	fmt.Printf("Selected Ubuntu %s %s AMI: %s (%s)\n", version, arch, amiID, aws.ToString(newestAMI.Name))
	return amiID, nil
}

// findAmazonLinuxAMI finds the latest Amazon Linux AMI
func (a *AMISelector) findAmazonLinuxAMI(ctx context.Context, client *EC2Client, version, arch string) (string, error) {
	// Amazon Linux AMI name pattern based on architecture
	var namePattern string
	var archType types.ArchitectureValues
	if arch == "arm64" {
		archType = types.ArchitectureValuesArm64
		namePattern = "amzn2-ami-hvm-*-arm64-gp2"
	} else {
		archType = types.ArchitectureValuesX8664
		namePattern = "amzn2-ami-hvm-*-x86_64-gp2"
	}

	result, err := client.client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{namePattern},
			},
			{
				Name:   aws.String("architecture"),
				Values: []string{string(archType)},
			},
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
			{
				Name:   aws.String("owner-id"),
				Values: []string{"137112412989"}, // Amazon's AWS account
			},
		},
		Owners: []string{"137112412989"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query Amazon Linux AMIs: %w", err)
	}

	if len(result.Images) == 0 {
		return "", fmt.Errorf("no Amazon Linux %s %s AMIs found in region %s", version, arch, a.region)
	}

	// Find the most recent AMI
	var newestAMI *types.Image
	for i := range result.Images {
		img := &result.Images[i]
		if newestAMI == nil || aws.ToString(img.CreationDate) > aws.ToString(newestAMI.CreationDate) {
			newestAMI = img
		}
	}

	amiID := aws.ToString(newestAMI.ImageId)
	fmt.Printf("Selected Amazon Linux %s %s AMI: %s (%s)\n", version, arch, amiID, aws.ToString(newestAMI.Name))
	return amiID, nil
}
