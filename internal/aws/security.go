package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const (
	createdByUser       = "user"
	createdByAwsJupyter = "aws-jupyter"
	defaultSGName       = "aws-jupyter"

	// Port numbers
	portSSH     = 22
	portJupyter = 8888
)

// SecurityGroupStrategy defines the strategy for security group management
type SecurityGroupStrategy struct {
	PreferExisting bool
	DefaultName    string // "aws-jupyter"
	UserSpecified  string
	VpcID          string
	ForceCreate    bool
}

// SecurityGroupInfo contains information about a security group
type SecurityGroupInfo struct {
	ID          string
	Name        string
	Description string
	VpcID       string
	CreatedBy   string
}

// DefaultSecurityGroupStrategy returns the default strategy for security groups
func DefaultSecurityGroupStrategy(vpcID string) SecurityGroupStrategy {
	return SecurityGroupStrategy{
		PreferExisting: true,
		DefaultName:    defaultSGName,
		VpcID:          vpcID,
		ForceCreate:    false,
	}
}

// GetDefaultSecurityGroupName returns the default name that would be used
func (s SecurityGroupStrategy) GetDefaultSecurityGroupName() string {
	if s.UserSpecified != "" {
		return s.UserSpecified
	}
	return s.DefaultName
}

// SecurityGroupExists checks if a security group exists in AWS
func (e *EC2Client) SecurityGroupExists(ctx context.Context, name string) (bool, *SecurityGroupInfo, error) {
	result, err := e.client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []string{name},
			},
		},
	})
	if err != nil {
		return false, nil, fmt.Errorf("failed to check security group existence: %w", err)
	}

	if len(result.SecurityGroups) == 0 {
		return false, nil, nil
	}

	sg := result.SecurityGroups[0]
	createdBy := createdByUser
	if IsAwsJupyterSecurityGroup(aws.ToString(sg.GroupName)) {
		createdBy = createdByAwsJupyter
	}

	info := &SecurityGroupInfo{
		ID:          aws.ToString(sg.GroupId),
		Name:        aws.ToString(sg.GroupName),
		Description: aws.ToString(sg.Description),
		VpcID:       aws.ToString(sg.VpcId),
		CreatedBy:   createdBy,
	}

	return true, info, nil
}

// CreateSecurityGroup creates a new security group with appropriate access rules
func (e *EC2Client) CreateSecurityGroup(ctx context.Context, name, vpcID string) (*SecurityGroupInfo, error) {
	isSessionManager := (name == "aws-jupyter-session-manager")

	var description string
	var rules []types.IpPermission

	if isSessionManager {
		description = "aws-jupyter security group - Session Manager and Jupyter Lab access"
		// For Session Manager, we only need Jupyter access (no SSH)
		rules = []types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(portJupyter),
				ToPort:     aws.Int32(portJupyter),
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String("127.0.0.1/32"),
						Description: aws.String("Jupyter Lab access via port forwarding only"),
					},
				},
			},
		}
	} else {
		description = "aws-jupyter security group - SSH and Jupyter Lab access"

		// Get current public IP for restricted SSH access
		publicIP, err := e.getCurrentPublicIP()
		if err != nil {
			fmt.Printf("Warning: Could not determine public IP, allowing SSH from anywhere: %v\n", err)
			publicIP = "0.0.0.0/0"
		} else {
			publicIP = publicIP + "/32"
			fmt.Printf("Restricting SSH access to your current IP: %s\n", publicIP)
		}

		// Add inbound rules for SSH (22) and Jupyter (8888)
		rules = []types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(portSSH),
				ToPort:     aws.Int32(portSSH),
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String(publicIP),
						Description: aws.String("SSH access from current IP"),
					},
				},
			},
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(portJupyter),
				ToPort:     aws.Int32(portJupyter),
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String("127.0.0.1/32"),
						Description: aws.String("Jupyter Lab access via SSH tunnel only"),
					},
				},
			},
		}
	}

	// Create the security group
	createResult, err := e.client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(name),
		Description: aws.String(description),
		VpcId:       aws.String(vpcID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroup,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(name)},
					{Key: aws.String("CreatedBy"), Value: aws.String("aws-jupyter-cli")},
					{Key: aws.String("Purpose"), Value: aws.String("jupyter-lab-access")},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}

	sgID := aws.ToString(createResult.GroupId)

	_, err = e.client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       aws.String(sgID),
		IpPermissions: rules,
	})
	if err != nil {
		// Clean up the security group if rule addition fails
		if _, deleteErr := e.client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(sgID),
		}); deleteErr != nil {
			// Log cleanup failure but return original error
			fmt.Printf("Warning: Failed to delete security group after error: %v\n", deleteErr)
		}
		return nil, fmt.Errorf("failed to add security group rules: %w", err)
	}

	return &SecurityGroupInfo{
		ID:          sgID,
		Name:        name,
		Description: description,
		VpcID:       vpcID,
		CreatedBy:   createdByAwsJupyter,
	}, nil
}

// GetOrCreateSecurityGroup gets an existing security group or creates a new one
func (e *EC2Client) GetOrCreateSecurityGroup(ctx context.Context, strategy SecurityGroupStrategy) (*SecurityGroupInfo, error) {
	sgName := strategy.GetDefaultSecurityGroupName()

	// If user specified a security group name, check if it exists
	if strategy.UserSpecified != "" {
		exists, sgInfo, err := e.SecurityGroupExists(ctx, strategy.UserSpecified)
		if err != nil {
			return nil, fmt.Errorf("failed to check user-specified security group: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("user-specified security group '%s' does not exist", strategy.UserSpecified)
		}
		return sgInfo, nil
	}

	// Check if our default security group exists
	if strategy.PreferExisting && !strategy.ForceCreate {
		exists, sgInfo, err := e.SecurityGroupExists(ctx, sgName)
		if err != nil {
			return nil, fmt.Errorf("failed to check for existing security group: %w", err)
		}
		if exists {
			// Validate that the existing security group has the required rules
			if err := e.validateSecurityGroupRules(ctx, sgInfo.ID); err != nil {
				fmt.Printf("Warning: Existing security group may not have optimal rules: %v\n", err)
			}
			return sgInfo, nil
		}
	}

	// Get VPC ID if not provided
	vpcID := strategy.VpcID
	if vpcID == "" {
		defaultVpcID, err := e.getDefaultVpcID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get default VPC: %w", err)
		}
		vpcID = defaultVpcID
	}

	// Create new security group
	fmt.Printf("Creating new security group: %s\n", sgName)
	return e.CreateSecurityGroup(ctx, sgName, vpcID)
}

// getCurrentPublicIP attempts to determine the current public IP address
func (e *EC2Client) getCurrentPublicIP() (string, error) {
	// This is a simplified version - in a real implementation,
	// you'd use an HTTP client to fetch from external services like:
	// - https://checkip.amazonaws.com
	// - https://ifconfig.me/ip
	// - https://ipv4.icanhazip.com
	// For now, we'll return an error to use the fallback

	return "", fmt.Errorf("could not determine public IP from external services")
}

// getDefaultVpcID gets the default VPC ID for the current region
func (e *EC2Client) getDefaultVpcID(ctx context.Context) (string, error) {
	result, err := e.client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("isDefault"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe VPCs: %w", err)
	}

	if len(result.Vpcs) == 0 {
		return "", fmt.Errorf("no default VPC found")
	}

	return aws.ToString(result.Vpcs[0].VpcId), nil
}

// validateSecurityGroupRules checks if a security group has the required rules
func (e *EC2Client) validateSecurityGroupRules(ctx context.Context, sgID string) error {
	result, err := e.client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{sgID},
	})
	if err != nil {
		return fmt.Errorf("failed to describe security group rules: %w", err)
	}

	if len(result.SecurityGroups) == 0 {
		return fmt.Errorf("security group not found")
	}

	sg := result.SecurityGroups[0]
	hasSSH := false
	hasJupyter := false

	for _, rule := range sg.IpPermissions {
		if aws.ToInt32(rule.FromPort) == portSSH && aws.ToInt32(rule.ToPort) == portSSH {
			hasSSH = true
		}
		if aws.ToInt32(rule.FromPort) == portJupyter && aws.ToInt32(rule.ToPort) == portJupyter {
			hasJupyter = true
		}
	}

	if !hasSSH {
		return fmt.Errorf("missing SSH rule (port %d)", portSSH)
	}
	if !hasJupyter {
		return fmt.Errorf("missing Jupyter rule (port %d)", portJupyter)
	}

	return nil
}

// IsAwsJupyterSecurityGroup checks if a security group name follows aws-jupyter naming convention
func IsAwsJupyterSecurityGroup(name string) bool {
	return name == defaultSGName
}

// ListSecurityGroups returns all security groups in the current VPC
func (e *EC2Client) ListSecurityGroups(ctx context.Context) ([]SecurityGroupInfo, error) {
	result, err := e.client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list security groups: %w", err)
	}

	var groups []SecurityGroupInfo
	for _, sg := range result.SecurityGroups {
		if sg.GroupName == nil {
			continue
		}

		createdBy := createdByUser
		if IsAwsJupyterSecurityGroup(*sg.GroupName) {
			createdBy = createdByAwsJupyter
		}

		groups = append(groups, SecurityGroupInfo{
			ID:          aws.ToString(sg.GroupId),
			Name:        aws.ToString(sg.GroupName),
			Description: aws.ToString(sg.Description),
			VpcID:       aws.ToString(sg.VpcId),
			CreatedBy:   createdBy,
		})
	}

	return groups, nil
}
