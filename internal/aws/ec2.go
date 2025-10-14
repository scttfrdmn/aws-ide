package aws

import (
	"context"
	"fmt"
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

// GetRegion returns the current region for this client
func (e *EC2Client) GetRegion() string {
	return e.region
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

	result, err := e.client.RunInstances(ctx, runInput)
	if err != nil {
		return nil, err
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

// TerminateInstance permanently terminates an EC2 instance
func (e *EC2Client) TerminateInstance(ctx context.Context, instanceID string) error {
	_, err := e.client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	return err
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
