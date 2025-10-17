package aws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go"
)

// IAMClient wraps the AWS IAM client with our methods
type IAMClient struct {
	client *iam.Client
}

// NewIAMClient creates a new IAM client using the same profile as EC2Client
func NewIAMClient(ctx context.Context, profile string) (*IAMClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile),
		config.WithRegion("us-east-1"), // IAM is global but requires a region
	)
	if err != nil {
		return nil, err
	}

	return &IAMClient{
		client: iam.NewFromConfig(cfg),
	}, nil
}

// InstanceProfileInfo contains information about an instance profile
type InstanceProfileInfo struct {
	Name string
	Arn  string
	Role string
}

// SessionManagerTrustPolicy is the trust policy for EC2 instances
const SessionManagerTrustPolicy = `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Service": "ec2.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
        }
    ]
}`

// GetOrCreateSessionManagerRole creates or gets the IAM role for Session Manager
func (i *IAMClient) GetOrCreateSessionManagerRole(ctx context.Context) (*InstanceProfileInfo, error) {
	roleName := "aws-jupyter-session-manager-role"
	instanceProfileName := "aws-jupyter-session-manager-profile"

	// Check if role exists
	roleExists, err := i.roleExists(ctx, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to check role existence: %w", err)
	}

	if !roleExists {
		// Create the role
		fmt.Printf("Creating IAM role: %s\n", roleName)
		if err := i.createRole(ctx, roleName); err != nil {
			return nil, fmt.Errorf("failed to create role: %w", err)
		}

		// Attach the Session Manager policy
		if err := i.attachSessionManagerPolicy(ctx, roleName); err != nil {
			return nil, fmt.Errorf("failed to attach Session Manager policy: %w", err)
		}
	} else {
		fmt.Printf("Using existing IAM role: %s\n", roleName)
		// Ensure auto-stop policy is attached to existing role (for backwards compatibility)
		if err := i.ensureAutoStopPolicy(ctx, roleName); err != nil {
			fmt.Printf("Warning: Failed to ensure auto-stop policy: %v\n", err)
		}
	}

	// Check if instance profile exists
	profileExists, profileInfo, err := i.instanceProfileExists(ctx, instanceProfileName)
	if err != nil {
		return nil, fmt.Errorf("failed to check instance profile existence: %w", err)
	}

	if !profileExists {
		// Create instance profile
		fmt.Printf("Creating instance profile: %s\n", instanceProfileName)
		profileInfo, err = i.createInstanceProfile(ctx, instanceProfileName, roleName)
		if err != nil {
			return nil, fmt.Errorf("failed to create instance profile: %w", err)
		}
	} else {
		fmt.Printf("Using existing instance profile: %s\n", instanceProfileName)
	}

	return profileInfo, nil
}

// roleExists checks if an IAM role exists
func (i *IAMClient) roleExists(ctx context.Context, roleName string) (bool, error) {
	_, err := i.client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchEntity" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// createRole creates a new IAM role
func (i *IAMClient) createRole(ctx context.Context, roleName string) error {
	_, err := i.client.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(SessionManagerTrustPolicy),
		Description:              aws.String("IAM role for aws-jupyter instances with Session Manager access"),
		Tags: []types.Tag{
			{
				Key:   aws.String("CreatedBy"),
				Value: aws.String("aws-jupyter-cli"),
			},
			{
				Key:   aws.String("Purpose"),
				Value: aws.String("session-manager-access"),
			},
		},
	})
	return err
}

// attachSessionManagerPolicy attaches the Session Manager policy to the role
func (i *IAMClient) attachSessionManagerPolicy(ctx context.Context, roleName string) error {
	// Attach AWS managed policy for Session Manager
	policyArn := "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"

	_, err := i.client.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyArn),
	})
	if err != nil {
		return fmt.Errorf("failed to attach Session Manager policy: %w", err)
	}

	// Also attach CloudWatch agent policy for better monitoring
	cloudWatchPolicyArn := "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
	_, err = i.client.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(cloudWatchPolicyArn),
	})
	if err != nil {
		fmt.Printf("Warning: Failed to attach CloudWatch policy (non-critical): %v\n", err)
	}

	// Add inline policy for auto-stop (allows instance to stop itself)
	if err := i.attachAutoStopPolicy(ctx, roleName); err != nil {
		fmt.Printf("Warning: Failed to attach auto-stop policy (non-critical): %v\n", err)
	}

	return nil
}

// attachAutoStopPolicy attaches an inline policy allowing the instance to stop itself
func (i *IAMClient) attachAutoStopPolicy(ctx context.Context, roleName string) error {
	policyDoc := `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "ec2:DescribeInstances",
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": "ec2:StopInstances",
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "ec2:ResourceTag/CreatedBy": "aws-jupyter-cli"
                }
            }
        }
    ]
}`

	_, err := i.client.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String("aws-jupyter-auto-stop-policy"),
		PolicyDocument: aws.String(policyDoc),
	})
	if err != nil {
		return fmt.Errorf("failed to attach auto-stop policy: %w", err)
	}

	fmt.Println("✓ Auto-stop policy attached")
	return nil
}

// ensureAutoStopPolicy checks if auto-stop policy exists and attaches it if missing
func (i *IAMClient) ensureAutoStopPolicy(ctx context.Context, roleName string) error {
	// Check if the inline policy already exists
	_, err := i.client.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String("aws-jupyter-auto-stop-policy"),
	})

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchEntity" {
			// Policy doesn't exist, attach it
			return i.attachAutoStopPolicy(ctx, roleName)
		}
		return err
	}

	// Policy already exists
	return nil
}

// instanceProfileExists checks if an instance profile exists
func (i *IAMClient) instanceProfileExists(ctx context.Context, profileName string) (bool, *InstanceProfileInfo, error) {
	result, err := i.client.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchEntity" {
			return false, nil, nil
		}
		return false, nil, err
	}

	profile := result.InstanceProfile
	roleName := ""
	if len(profile.Roles) > 0 {
		roleName = aws.ToString(profile.Roles[0].RoleName)
	}

	info := &InstanceProfileInfo{
		Name: aws.ToString(profile.InstanceProfileName),
		Arn:  aws.ToString(profile.Arn),
		Role: roleName,
	}

	return true, info, nil
}

// createInstanceProfile creates a new instance profile and adds the role to it
func (i *IAMClient) createInstanceProfile(ctx context.Context, profileName, roleName string) (*InstanceProfileInfo, error) {
	// Create the instance profile
	result, err := i.client.CreateInstanceProfile(ctx, &iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
		Tags: []types.Tag{
			{
				Key:   aws.String("CreatedBy"),
				Value: aws.String("aws-jupyter-cli"),
			},
			{
				Key:   aws.String("Purpose"),
				Value: aws.String("session-manager-access"),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance profile: %w", err)
	}

	// Add the role to the instance profile
	_, err = i.client.AddRoleToInstanceProfile(ctx, &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
		RoleName:            aws.String(roleName),
	})
	if err != nil {
		// Clean up the instance profile if adding role fails
		if _, deleteErr := i.client.DeleteInstanceProfile(ctx, &iam.DeleteInstanceProfileInput{
			InstanceProfileName: aws.String(profileName),
		}); deleteErr != nil {
			// Log cleanup failure but return original error
			fmt.Printf("Warning: Failed to cleanup instance profile after error: %v\n", deleteErr)
		}
		return nil, fmt.Errorf("failed to add role to instance profile: %w", err)
	}

	// Wait for the instance profile to propagate (IAM is eventually consistent)
	fmt.Println("Waiting for IAM instance profile to propagate...")
	if err := i.waitForInstanceProfileReady(ctx, profileName); err != nil {
		return nil, fmt.Errorf("instance profile not ready: %w", err)
	}

	return &InstanceProfileInfo{
		Name: profileName,
		Arn:  aws.ToString(result.InstanceProfile.Arn),
		Role: roleName,
	}, nil
}

// waitForInstanceProfileReady polls until the instance profile is fully propagated
func (i *IAMClient) waitForInstanceProfileReady(ctx context.Context, profileName string) error {
	maxWaitTime := 30 * time.Second
	checkInterval := 2 * time.Second
	timeout := time.After(maxWaitTime)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for instance profile to be ready")
		case <-ticker.C:
			// Try to get the instance profile with the role attached
			result, err := i.client.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{
				InstanceProfileName: aws.String(profileName),
			})
			if err != nil {
				continue // Not ready yet
			}

			// Check if role is attached
			if len(result.InstanceProfile.Roles) > 0 {
				fmt.Println("✓ IAM instance profile ready")
				return nil
			}
		}
	}
}

// ValidateSessionManagerRole checks if the role has the correct policies attached
func (i *IAMClient) ValidateSessionManagerRole(ctx context.Context, roleName string) error {
	// List attached policies
	result, err := i.client.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to list role policies: %w", err)
	}

	// Check for required Session Manager policy
	hasSessionManagerPolicy := false
	for _, policy := range result.AttachedPolicies {
		if aws.ToString(policy.PolicyArn) == "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore" {
			hasSessionManagerPolicy = true
			break
		}
	}

	if !hasSessionManagerPolicy {
		return fmt.Errorf("role missing AmazonSSMManagedInstanceCore policy")
	}

	return nil
}

// CleanupSessionManagerResources removes Session Manager IAM resources (use with caution)
func (i *IAMClient) CleanupSessionManagerResources(ctx context.Context) error {
	roleName := "aws-jupyter-session-manager-role"
	instanceProfileName := "aws-jupyter-session-manager-profile"

	fmt.Println("⚠️  Cleaning up Session Manager IAM resources...")

	// Remove role from instance profile
	_, err := i.client.RemoveRoleFromInstanceProfile(ctx, &iam.RemoveRoleFromInstanceProfileInput{
		InstanceProfileName: aws.String(instanceProfileName),
		RoleName:            aws.String(roleName),
	})
	if err != nil {
		fmt.Printf("Warning: Failed to remove role from instance profile: %v\n", err)
	}

	// Delete instance profile
	_, err = i.client.DeleteInstanceProfile(ctx, &iam.DeleteInstanceProfileInput{
		InstanceProfileName: aws.String(instanceProfileName),
	})
	if err != nil {
		fmt.Printf("Warning: Failed to delete instance profile: %v\n", err)
	}

	// Detach policies from role
	policies := []string{
		"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
		"arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy",
	}

	for _, policyArn := range policies {
		_, err = i.client.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
			RoleName:  aws.String(roleName),
			PolicyArn: aws.String(policyArn),
		})
		if err != nil {
			fmt.Printf("Warning: Failed to detach policy %s: %v\n", policyArn, err)
		}
	}

	// Delete role
	_, err = i.client.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	fmt.Println("✓ Cleaned up Session Manager IAM resources")
	return nil
}
