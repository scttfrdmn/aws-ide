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

type EC2Client struct {
	client *ec2.Client
	region string
}

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

func (e *EC2Client) LaunchInstance(ctx context.Context, params LaunchParams) (*types.Instance, error) {
	// Get default VPC and subnet
	subnet, err := e.getDefaultSubnet(ctx)
	if err != nil {
		return nil, err
	}

	runInput := &ec2.RunInstancesInput{
		ImageId:          aws.String(params.AMI),
		InstanceType:     types.InstanceType(params.InstanceType),
		MinCount:         aws.Int32(1),
		MaxCount:         aws.Int32(1),
		KeyName:          aws.String(params.KeyPairName),
		SubnetId:         subnet,
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

func (e *EC2Client) WaitForInstanceRunning(ctx context.Context, instanceID string) error {
	waiter := ec2.NewInstanceRunningWaiter(e.client)
	return waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 5*time.Minute)
}

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

func (e *EC2Client) StopInstance(ctx context.Context, instanceID string, hibernate bool) error {
	input := &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
		Hibernate:   aws.Bool(hibernate),
	}
	_, err := e.client.StopInstances(ctx, input)
	return err
}

func (e *EC2Client) TerminateInstance(ctx context.Context, instanceID string) error {
	_, err := e.client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	return err
}

type LaunchParams struct {
	AMI             string
	InstanceType    string
	KeyPairName     string
	SecurityGroupID string
	UserData        string
	EBSVolumeSize   int
	Environment     string
}
