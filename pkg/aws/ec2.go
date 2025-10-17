package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// EC2Client wraps the AWS EC2 SDK client with convenience methods for managing instances
type EC2Client struct {
	client *ec2.Client
	region string
}

// NewEC2Client creates a new EC2 client using the specified AWS profile
func NewEC2Client(ctx context.Context, profile string) (*EC2Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return nil, err
	}

	return &EC2Client{
		client: ec2.NewFromConfig(cfg),
		region: cfg.Region,
	}, nil
}

// NewEC2ClientForRegion creates a new EC2 client using the default profile for a specific region
func NewEC2ClientForRegion(ctx context.Context, region string) (*EC2Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	return &EC2Client{
		client: ec2.NewFromConfig(cfg),
		region: cfg.Region,
	}, nil
}

// GetRegion returns the current region for this client
func (e *EC2Client) GetRegion() string {
	return e.region
}

// IsInstanceTypeSupported checks if an instance type is available in a specific availability zone
func (e *EC2Client) IsInstanceTypeSupported(ctx context.Context, instanceType, availabilityZone string) (bool, error) {
	result, err := e.client.DescribeInstanceTypeOfferings(ctx, &ec2.DescribeInstanceTypeOfferingsInput{
		LocationType: types.LocationTypeAvailabilityZone,
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-type"),
				Values: []string{instanceType},
			},
			{
				Name:   aws.String("location"),
				Values: []string{availabilityZone},
			},
		},
	})
	if err != nil {
		return false, err
	}

	return len(result.InstanceTypeOfferings) > 0, nil
}

// FindCompatibleAvailabilityZone finds an availability zone that supports the instance type and has the requested subnet type
func (e *EC2Client) FindCompatibleAvailabilityZone(ctx context.Context, instanceType, subnetType string) (string, error) {
	// Get all availability zones that support this instance type
	result, err := e.client.DescribeInstanceTypeOfferings(ctx, &ec2.DescribeInstanceTypeOfferingsInput{
		LocationType: types.LocationTypeAvailabilityZone,
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-type"),
				Values: []string{instanceType},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to query instance type offerings: %w", err)
	}

	if len(result.InstanceTypeOfferings) == 0 {
		return "", fmt.Errorf("instance type %s not available in region %s", instanceType, e.region)
	}

	// Get default VPC to check for subnets
	vpcID, err := e.getDefaultVpcID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get default VPC: %w", err)
	}

	// Try each AZ to find one with a suitable subnet
	for _, offering := range result.InstanceTypeOfferings {
		az := aws.ToString(offering.Location)

		// Check if there's a subnet of the requested type in this AZ
		subnets, err := e.client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: []string{vpcID},
				},
				{
					Name:   aws.String("availability-zone"),
					Values: []string{az},
				},
			},
		})
		if err != nil {
			continue // Try next AZ
		}

		// Check if we have a subnet of the correct type
		for _, subnet := range subnets.Subnets {
			isPublic := aws.ToBool(subnet.MapPublicIpOnLaunch)
			if (subnetType == "public" && isPublic) || (subnetType == "private" && !isPublic) {
				return az, nil
			}
		}
	}

	return "", fmt.Errorf("no availability zone found with both %s support and %s subnet", instanceType, subnetType)
}

// LaunchInstance launches a new EC2 instance with the specified parameters
func (e *EC2Client) LaunchInstance(ctx context.Context, params LaunchParams) (*types.Instance, error) {
	// Use provided subnet or get default
	var subnetID *string
	if params.SubnetID != "" {
		subnetID = aws.String(params.SubnetID)
	} else {
		// Fall back to default subnet
		defaultSubnet, err := e.getDefaultSubnet(ctx)
		if err != nil {
			return nil, err
		}
		subnetID = defaultSubnet
	}

	runInput := &ec2.RunInstancesInput{
		ImageId:          aws.String(params.AMI),
		InstanceType:     types.InstanceType(params.InstanceType),
		MinCount:         aws.Int32(1),
		MaxCount:         aws.Int32(1),
		SubnetId:         subnetID,
		SecurityGroupIds: []string{params.SecurityGroupID},
		UserData:         aws.String(params.UserData),
		BlockDeviceMappings: []types.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &types.EbsBlockDevice{
					VolumeSize:          aws.Int32(int32(params.EBSVolumeSize)),
					VolumeType:          types.VolumeTypeGp3,
					DeleteOnTermination: aws.Bool(true),
				},
			},
		},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String("aws-jupyter")},
					{Key: aws.String("CreatedBy"), Value: aws.String("aws-jupyter-cli")},
					{Key: aws.String("Environment"), Value: aws.String(params.Environment)},
				},
			},
		},
	}

	// Set SSH key pair if provided (for SSH connections)
	if params.KeyPairName != "" {
		runInput.KeyName = aws.String(params.KeyPairName)
	}

	// Set IAM instance profile if provided (for Session Manager)
	if params.InstanceProfile != "" {
		runInput.IamInstanceProfile = &types.IamInstanceProfileSpecification{
			Name: aws.String(params.InstanceProfile),
		}
	}

	// Retry logic for IAM propagation delays
	// AWS IAM is eventually consistent, so we need to retry if the instance profile isn't ready
	maxRetries := 5
	var result *ec2.RunInstancesOutput
	var err error
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying (exponential backoff: 2s, 4s, 8s, 16s)
			waitTime := time.Duration(1<<uint(attempt)) * time.Second
			fmt.Printf("Retrying launch in %v (attempt %d/%d)...\n", waitTime, attempt+1, maxRetries)
			time.Sleep(waitTime)
		}

		result, err = e.client.RunInstances(ctx, runInput)
		if err != nil {
			lastErr = err
			errMsg := err.Error()
			// Check if it's an IAM-related error
			if params.InstanceProfile != "" && (
				// Common IAM propagation error messages
				strings.Contains(errMsg, "Invalid IAM Instance Profile") ||
				strings.Contains(errMsg, "iamInstanceProfile") ||
				strings.Contains(errMsg, "not valid") ||
				strings.Contains(errMsg, "does not exist")) {
				// IAM propagation issue, retry
				continue
			}
			// Different error, don't retry
			return nil, err
		}

		// Success!
		break
	}

	if err != nil {
		return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
	}

	if len(result.Instances) == 0 {
		return nil, fmt.Errorf("no instances created")
	}

	return &result.Instances[0], nil
}

func (e *EC2Client) getDefaultSubnet(ctx context.Context) (*string, error) {
	// Get default VPC
	vpcs, err := e.client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("isDefault"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(vpcs.Vpcs) == 0 {
		return nil, fmt.Errorf("no default VPC found")
	}

	// Get a subnet from the default VPC
	subnets, err := e.client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{*vpcs.Vpcs[0].VpcId},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(subnets.Subnets) == 0 {
		return nil, fmt.Errorf("no subnets found in default VPC")
	}

	return subnets.Subnets[0].SubnetId, nil
}

// WaitForInstanceRunning waits for an EC2 instance to reach the running state with a 5 minute timeout
func (e *EC2Client) WaitForInstanceRunning(ctx context.Context, instanceID string) error {
	waiter := ec2.NewInstanceRunningWaiter(e.client)
	return waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 5*time.Minute)
}

// GetInstanceInfo retrieves detailed information about a specific EC2 instance
func (e *EC2Client) GetInstanceInfo(ctx context.Context, instanceID string) (*types.Instance, error) {
	result, err := e.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, err
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance not found")
	}

	return &result.Reservations[0].Instances[0], nil
}

// StopInstance stops a running EC2 instance with optional hibernation support
func (e *EC2Client) StopInstance(ctx context.Context, instanceID string, hibernate bool) error {
	input := &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
		Hibernate:   aws.Bool(hibernate),
	}
	_, err := e.client.StopInstances(ctx, input)
	return err
}

// StartInstance starts a stopped EC2 instance
func (e *EC2Client) StartInstance(ctx context.Context, instanceID string) error {
	_, err := e.client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	return err
}

// TerminateInstance permanently terminates an EC2 instance
func (e *EC2Client) TerminateInstance(ctx context.Context, instanceID string) error {
	_, err := e.client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	return err
}

// CreateAMI creates an Amazon Machine Image from an EC2 instance
func (e *EC2Client) CreateAMI(ctx context.Context, instanceID, name, description string, noReboot bool) (string, error) {
	input := &ec2.CreateImageInput{
		InstanceId:  aws.String(instanceID),
		Name:        aws.String(name),
		Description: aws.String(description),
		NoReboot:    aws.Bool(noReboot),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeImage,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(name)},
					{Key: aws.String("CreatedBy"), Value: aws.String("aws-jupyter-cli")},
				},
			},
		},
	}

	result, err := e.client.CreateImage(ctx, input)
	if err != nil {
		return "", err
	}

	return *result.ImageId, nil
}

// AMIInfo contains information about a custom AMI
type AMIInfo struct {
	ID           string
	Name         string
	Description  string
	State        string
	CreationDate time.Time
}

// ListCustomAMIs lists all AMIs created by aws-jupyter CLI
func (e *EC2Client) ListCustomAMIs(ctx context.Context) ([]AMIInfo, error) {
	input := &ec2.DescribeImagesInput{
		Owners: []string{"self"},
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:CreatedBy"),
				Values: []string{"aws-jupyter-cli"},
			},
		},
	}

	result, err := e.client.DescribeImages(ctx, input)
	if err != nil {
		return nil, err
	}

	amis := make([]AMIInfo, 0, len(result.Images))
	for _, image := range result.Images {
		creationDate, _ := time.Parse(time.RFC3339, aws.ToString(image.CreationDate))
		amis = append(amis, AMIInfo{
			ID:           aws.ToString(image.ImageId),
			Name:         aws.ToString(image.Name),
			Description:  aws.ToString(image.Description),
			State:        string(image.State),
			CreationDate: creationDate,
		})
	}

	return amis, nil
}

// DeleteAMI deletes an AMI and its associated snapshots
func (e *EC2Client) DeleteAMI(ctx context.Context, amiID string) error {
	// First, get the AMI details to find associated snapshots
	describeInput := &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	}

	describeResult, err := e.client.DescribeImages(ctx, describeInput)
	if err != nil {
		return fmt.Errorf("failed to describe AMI: %w", err)
	}

	if len(describeResult.Images) == 0 {
		return fmt.Errorf("AMI %s not found", amiID)
	}

	image := describeResult.Images[0]

	// Deregister the AMI
	deregisterInput := &ec2.DeregisterImageInput{
		ImageId: aws.String(amiID),
	}

	_, err = e.client.DeregisterImage(ctx, deregisterInput)
	if err != nil {
		return fmt.Errorf("failed to deregister AMI: %w", err)
	}

	// Delete associated snapshots
	for _, blockDevice := range image.BlockDeviceMappings {
		if blockDevice.Ebs != nil && blockDevice.Ebs.SnapshotId != nil {
			snapshotID := *blockDevice.Ebs.SnapshotId
			deleteSnapshotInput := &ec2.DeleteSnapshotInput{
				SnapshotId: aws.String(snapshotID),
			}

			_, err := e.client.DeleteSnapshot(ctx, deleteSnapshotInput)
			if err != nil {
				// Log but don't fail - snapshot might be in use or already deleted
				fmt.Printf("Warning: Failed to delete snapshot %s: %v\n", snapshotID, err)
			}
		}
	}

	return nil
}

// LaunchParams contains all parameters needed to launch a new EC2 instance
type LaunchParams struct {
	AMI             string
	InstanceType    string
	KeyPairName     string
	SecurityGroupID string
	UserData        string
	EBSVolumeSize   int
	Environment     string
	SubnetID        string
	InstanceProfile string
}
