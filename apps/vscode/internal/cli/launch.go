package cli

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	vscodeconfig "github.com/scttfrdmn/aws-ide/apps/vscode/internal/config"
	"github.com/scttfrdmn/aws-ide/pkg/aws"
	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/spf13/cobra"
)

// NewLaunchCmd creates the launch command for starting new VSCode instances
func NewLaunchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Launch a new VSCode Server instance on AWS",
		Long: `Launch a new EC2 instance with VSCode Server (code-server) configured and ready to use.

The instance will be configured based on the selected environment preset, which determines:
- System packages to install
- Language runtimes (Node.js, Python, Go)
- VSCode extensions
- Development tools and utilities

Available environments: web-dev, python-dev, go-dev, fullstack`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("launch command not yet implemented - this is a placeholder for the full implementation")
		},
	}

	// Add flags (to be implemented)
	cmd.Flags().StringP("environment", "e", "web-dev", "Environment preset to use")
	cmd.Flags().StringP("profile", "p", "default", "AWS profile to use")
	cmd.Flags().StringP("region", "r", "", "AWS region (default: from AWS config)")

	return cmd
}

// generateUserData generates the user data script for VSCode Server setup
func generateUserData(env *config.Environment, idleTimeoutSeconds int) (string, error) {
	return vscodeconfig.GenerateUserData(env, idleTimeoutSeconds)
}

// displayVSCodeInfo displays VSCode-specific connection information
func displayVSCodeInfo(instance *types.Instance, env *config.Environment, subnet *aws.SubnetInfo, keyInfo *aws.KeyPairInfo, connectionMethod, subnetType, profile string) error {
	publicIP := "N/A (private subnet)"
	if instance.PublicIpAddress != nil {
		publicIP = *instance.PublicIpAddress
	}
	privateIP := *instance.PrivateIpAddress
	instanceID := *instance.InstanceId

	// Save instance to local state
	if err := saveInstanceToState(instance, env, keyInfo, connectionMethod); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save instance to local state: %v\n", err)
	}

	fmt.Println("\n🎉 VSCode Server launched successfully!")
	fmt.Printf("Instance ID: %s\n", instanceID)
	fmt.Printf("Instance Type: %s\n", env.InstanceType)
	fmt.Printf("Public IP: %s\n", publicIP)
	fmt.Printf("Private IP: %s\n", privateIP)
	fmt.Printf("Subnet: %s (%s)\n", subnet.ID, subnetType)

	if connectionMethod == "ssh" {
		fmt.Printf("SSH Key: %s\n", keyInfo.Name)
		fmt.Println("\n🔗 To connect:")
		if subnet.IsPublic {
			username := "ubuntu"
			if env.AMIBase == "amazonlinux2-arm64" || env.AMIBase == "amazonlinux2-x86_64" {
				username = "ec2-user"
			}
			fmt.Printf("ssh -i ~/.aws-vscode/keys/%s.pem %s@%s\n", keyInfo.Name, username, publicIP)
		} else {
			fmt.Println("Use Session Manager or VPN/bastion to connect to private instance")
		}
	} else {
		fmt.Println("\n🔗 To connect:")
		fmt.Printf("aws ssm start-session --target %s --profile %s\n", instanceID, profile)
	}

	fmt.Println("\n💻 VSCode Server will be available at: http://localhost:8080")
	fmt.Println("⏳ Please wait 2-3 minutes for VSCode Server to complete installation...")
	fmt.Printf("\nTo get the password, SSH into the instance and run:\n")
	fmt.Printf("cat ~/.config/code-server/config.yaml\n")
	fmt.Printf("\nOr use 'aws-vscode connect %s' to get connection details\n", instanceID)

	return nil
}

// saveInstanceToState saves the launched instance to local state
func saveInstanceToState(instance *types.Instance, env *config.Environment, keyInfo *aws.KeyPairInfo, connectionMethod string) error {
	state, err := config.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	publicIP := ""
	if instance.PublicIpAddress != nil {
		publicIP = *instance.PublicIpAddress
	}

	keyPairName := ""
	if keyInfo != nil {
		keyPairName = keyInfo.Name
	}

	// Determine region from placement
	region := ""
	if instance.Placement != nil && instance.Placement.AvailabilityZone != nil {
		az := *instance.Placement.AvailabilityZone
		if len(az) > 0 {
			region = az[:len(az)-1]
		}
	}

	// Get security group
	securityGroup := ""
	if len(instance.SecurityGroups) > 0 && instance.SecurityGroups[0].GroupId != nil {
		securityGroup = *instance.SecurityGroups[0].GroupId
	}

	state.Instances[*instance.InstanceId] = &config.Instance{
		ID:            *instance.InstanceId,
		Environment:   env.Name,
		InstanceType:  env.InstanceType,
		PublicIP:      publicIP,
		KeyPair:       keyPairName,
		LaunchedAt:    *instance.LaunchTime,
		IdleTimeout:   "",
		TunnelPID:     0,
		Region:        region,
		SecurityGroup: securityGroup,
		AMIBase:       env.AMIBase,
	}

	return state.Save()
}
