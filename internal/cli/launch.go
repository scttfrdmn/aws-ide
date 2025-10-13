package cli

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/aws-jupyter/internal/aws"
	"github.com/scttfrdmn/aws-jupyter/internal/config"
	"github.com/spf13/cobra"
)

func NewLaunchCmd() *cobra.Command {
	var (
		environment      string
		instanceType     string
		idleTimeout      string
		profile          string
		region           string
		dryRun           bool
		connectionMethod string
		subnetType       string
		createNatGateway bool
	)

	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Launch a new Jupyter instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLaunch(environment, instanceType, idleTimeout, profile, region, dryRun, connectionMethod, subnetType, createNatGateway)
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "data-science", "Environment configuration to use")
	cmd.Flags().StringVarP(&instanceType, "instance-type", "t", "", "Override instance type")
	cmd.Flags().StringVarP(&idleTimeout, "idle-timeout", "i", "4h", "Auto-shutdown timeout")
	cmd.Flags().StringVarP(&profile, "profile", "p", "default", "AWS profile to use")
	cmd.Flags().StringVarP(&region, "region", "r", "", "AWS region")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	cmd.Flags().StringVarP(&connectionMethod, "connection", "c", "ssh", "Connection method: ssh or session-manager")
	cmd.Flags().StringVarP(&subnetType, "subnet-type", "s", "public", "Subnet type: public or private")
	cmd.Flags().BoolVar(&createNatGateway, "create-nat-gateway", false, "Create NAT Gateway for private subnet internet access")

	return cmd
}

func runLaunch(environment, instanceType, idleTimeout, profile, region string, dryRun bool, connectionMethod, subnetType string, createNatGateway bool) error {
	ctx := context.Background()

	// Load environment configuration
	env, err := config.LoadEnvironment(environment)
	if err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	// Override instance type if provided
	if instanceType != "" {
		env.InstanceType = instanceType
	}

	// Validate options
	if connectionMethod != "ssh" && connectionMethod != "session-manager" {
		return fmt.Errorf("connection method must be 'ssh' or 'session-manager'")
	}
	if subnetType != "public" && subnetType != "private" {
		return fmt.Errorf("subnet type must be 'public' or 'private'")
	}

	// Warn about private subnet implications
	if subnetType == "private" && !createNatGateway {
		fmt.Println("⚠️  Warning: Private subnet without NAT Gateway means limited internet access")
		fmt.Println("   - Package installations may fail")
		fmt.Println("   - Jupyter extensions may not work")
		fmt.Println("   - Consider using --create-nat-gateway for full functionality")
	}

	// Session Manager implications
	if connectionMethod == "session-manager" {
		fmt.Println("ℹ️  Using Session Manager connection (no SSH keys needed)")
		if subnetType == "public" {
			fmt.Println("   - Instance will be in public subnet but without SSH access")
		}
	}

	if dryRun {
		// Create AWS client to determine actual region and key name
		ec2Client, err := aws.NewEC2Client(ctx, profile)
		if err != nil {
			return fmt.Errorf("failed to create AWS client for dry run: %w", err)
		}

		// Determine actual region (override or from AWS config)
		actualRegion := ec2Client.GetRegion()
		if region != "" {
			actualRegion = region
		}

		// Get the key name that would be used
		keyStrategy := aws.DefaultKeyPairStrategy(actualRegion)
		keyName := keyStrategy.GetDefaultKeyName()

		fmt.Printf("[DRY RUN] Would launch %s environment on %s in region %s\n", env.Name, env.InstanceType, actualRegion)
		fmt.Printf("[DRY RUN] Configuration:\n")
		fmt.Printf("  - Environment: %s\n", env.Name)
		fmt.Printf("  - Instance Type: %s\n", env.InstanceType)
		fmt.Printf("  - AMI Base: %s\n", env.AMIBase)
		fmt.Printf("  - EBS Volume: %dGB\n", env.EBSVolumeSize)
		fmt.Printf("  - Packages: %d system packages\n", len(env.Packages))
		fmt.Printf("  - Pip Packages: %d python packages\n", len(env.PipPackages))
		fmt.Printf("  - Jupyter Extensions: %d extensions\n", len(env.JupyterExtensions))
		fmt.Printf("  - Idle Timeout: %s\n", idleTimeout)
		fmt.Printf("  - AWS Profile: %s\n", profile)
		fmt.Printf("  - AWS Region: %s\n", actualRegion)
		if region != "" {
			fmt.Printf("  - Region Override: %s\n", region)
		}
		fmt.Printf("  - Connection Method: %s\n", connectionMethod)
		fmt.Printf("  - Subnet Type: %s\n", subnetType)
		if createNatGateway && subnetType == "private" {
			fmt.Printf("  - NAT Gateway: will be created (additional cost)\n")
		}
		if connectionMethod == "ssh" {
			fmt.Printf("  - SSH Key Pair: %s (economical reuse)\n", keyName)
		} else {
			fmt.Printf("  - Session Manager: IAM role will be created/attached\n")
		}

		fmt.Printf("[DRY RUN] Would perform these actions:\n")
		actionNum := 1
		if connectionMethod == "ssh" {
			fmt.Printf("  %d. Create/verify SSH key pair (%s)\n", actionNum, keyName)
			actionNum++
		} else {
			fmt.Printf("  %d. Create/verify IAM role for Session Manager\n", actionNum)
			actionNum++
		}

		if connectionMethod == "ssh" {
			fmt.Printf("  %d. Create/verify security group (SSH + Jupyter access)\n", actionNum)
		} else {
			fmt.Printf("  %d. Create/verify security group (Jupyter access only)\n", actionNum)
		}
		actionNum++

		if subnetType == "private" && createNatGateway {
			fmt.Printf("  %d. Create/verify NAT Gateway for internet access\n", actionNum)
			actionNum++
		}

		fmt.Printf("  %d. Generate user data script for environment setup\n", actionNum)
		actionNum++
		fmt.Printf("  %d. Launch EC2 instance (%s) in %s subnet\n", actionNum, env.InstanceType, subnetType)
		actionNum++
		fmt.Printf("  %d. Wait for instance to be running\n", actionNum)
		actionNum++

		if connectionMethod == "ssh" {
			fmt.Printf("  %d. Setup SSH tunnel (port 8888)\n", actionNum)
		} else {
			fmt.Printf("  %d. Setup Session Manager port forwarding (port 8888)\n", actionNum)
		}
		actionNum++

		fmt.Printf("  %d. Save instance state locally\n", actionNum)
		actionNum++
		fmt.Printf("  %d. Display connection information\n", actionNum)

		fmt.Println("[DRY RUN] No resources were created")
		return nil
	}

	fmt.Printf("Launching %s environment on %s...\n", env.Name, env.InstanceType)

	// Create AWS client
	ec2Client, err := aws.NewEC2Client(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Determine actual region (override or from AWS config)
	actualRegion := ec2Client.GetRegion()
	if region != "" {
		actualRegion = region
		// TODO: Handle region override by creating new client with specific region
		fmt.Printf("Note: Region override (%s) not yet implemented, using profile region (%s)\n", region, actualRegion)
	}

	var keyInfo *aws.KeyPairInfo
	var instanceProfile *aws.InstanceProfileInfo

	// Handle connection method setup
	if connectionMethod == "ssh" {
		// Setup SSH key pair
		fmt.Println("🔑 Setting up SSH key pair...")
		keyStorage, err := config.DefaultKeyStorage()
		if err != nil {
			return fmt.Errorf("failed to initialize key storage: %w", err)
		}

		keyStrategy := aws.DefaultKeyPairStrategy(actualRegion)
		keyInfo, err = ec2Client.GetOrCreateKeyPair(ctx, keyStrategy)
		if err != nil {
			return fmt.Errorf("failed to setup SSH key pair: %w", err)
		}

		fmt.Printf("Using SSH key pair: %s\n", keyInfo.Name)

		// Save private key locally if it was newly created
		if keyInfo.PrivateKey != "" {
			fmt.Println("Saving SSH private key locally...")
			if err := keyStorage.SavePrivateKey(keyInfo); err != nil {
				return fmt.Errorf("failed to save SSH private key: %w", err)
			}
			fmt.Printf("SSH private key saved to: %s\n", keyStorage.GetKeyPath(keyInfo.Name))
		} else {
			// For existing keys, verify we have the private key locally
			if !keyStorage.HasPrivateKey(keyInfo.Name) {
				return fmt.Errorf("SSH key pair '%s' exists in AWS but private key not found locally", keyInfo.Name)
			}
			fmt.Printf("Using existing local private key: %s\n", keyStorage.GetKeyPath(keyInfo.Name))
		}
	} else {
		// Setup Session Manager IAM role
		fmt.Println("🔐 Setting up Session Manager IAM role...")
		iamClient, err := aws.NewIAMClient(ctx, profile)
		if err != nil {
			return fmt.Errorf("failed to create IAM client: %w", err)
		}

		instanceProfile, err = iamClient.GetOrCreateSessionManagerRole(ctx)
		if err != nil {
			return fmt.Errorf("failed to setup Session Manager role: %w", err)
		}

		fmt.Printf("Using IAM instance profile: %s\n", instanceProfile.Name)
	}

	// Get or create appropriate subnet
	fmt.Printf("🌐 Selecting %s subnet...\n", subnetType)
	subnet, err := ec2Client.GetSubnet(ctx, subnetType, "")
	if err != nil {
		return fmt.Errorf("failed to get subnet: %w", err)
	}
	fmt.Printf("Using subnet: %s (%s) in %s\n", subnet.ID, subnet.CidrBlock, subnet.AvailabilityZone)

	// Handle NAT Gateway for private subnets
	var natGateway *aws.NATGatewayInfo
	if subnetType == "private" && createNatGateway {
		fmt.Println("🚪 Setting up NAT Gateway for internet access...")
		natGateway, err = ec2Client.GetOrCreateNATGateway(ctx, subnet.VpcId)
		if err != nil {
			return fmt.Errorf("failed to setup NAT Gateway: %w", err)
		}

		// Update route tables for the private subnet
		if err := ec2Client.UpdatePrivateSubnetRoutes(ctx, subnet.ID, natGateway.ID); err != nil {
			return fmt.Errorf("failed to update subnet routes: %w", err)
		}
	}

	// Setup security group
	fmt.Println("🔒 Setting up security group...")
	sgStrategy := aws.DefaultSecurityGroupStrategy(subnet.VpcId)
	if connectionMethod == "session-manager" {
		// For Session Manager, we don't need SSH access
		sgStrategy.DefaultName = "aws-jupyter-session-manager"
	}

	securityGroup, err := ec2Client.GetOrCreateSecurityGroup(ctx, sgStrategy)
	if err != nil {
		return fmt.Errorf("failed to setup security group: %w", err)
	}
	fmt.Printf("Using security group: %s (%s)\n", securityGroup.Name, securityGroup.ID)

	// Select AMI based on environment configuration
	fmt.Println("🔍 Selecting AMI for environment...")
	amiSelector := aws.NewAMISelector(actualRegion)
	amiID, err := amiSelector.GetAMI(ctx, ec2Client, env.AMIBase)
	if err != nil {
		fmt.Printf("Warning: Could not find latest AMI (%v), using fallback\n", err)
		amiID = amiSelector.GetDefaultAMI()
		fmt.Printf("Using fallback AMI: %s\n", amiID)
	}

	// Generate user data script
	fmt.Println("📜 Generating user data script...")
	userData, err := config.GenerateUserData(env)
	if err != nil {
		return fmt.Errorf("failed to generate user data: %w", err)
	}

	// Launch EC2 instance
	fmt.Printf("🚀 Launching EC2 instance (%s)...\n", env.InstanceType)

	launchParams := aws.LaunchParams{
		AMI:             amiID,
		InstanceType:    env.InstanceType,
		SecurityGroupID: securityGroup.ID,
		UserData:        userData,
		EBSVolumeSize:   env.EBSVolumeSize,
		Environment:     env.Name,
	}

	// Set subnet explicitly
	launchParams.SubnetId = subnet.ID

	// Set connection-specific parameters
	if connectionMethod == "ssh" {
		launchParams.KeyPairName = keyInfo.Name
	} else {
		launchParams.InstanceProfile = instanceProfile.Name
	}

	instance, err := ec2Client.LaunchInstance(ctx, launchParams)
	if err != nil {
		return fmt.Errorf("failed to launch instance: %w", err)
	}

	instanceId := *instance.InstanceId
	fmt.Printf("✓ Instance launched: %s\n", instanceId)

	// Wait for instance to be running
	fmt.Println("⏳ Waiting for instance to be running...")
	if err := ec2Client.WaitForInstanceRunning(ctx, instanceId); err != nil {
		return fmt.Errorf("instance failed to start: %w", err)
	}

	// Get instance details
	instanceInfo, err := ec2Client.GetInstanceInfo(ctx, instanceId)
	if err != nil {
		return fmt.Errorf("failed to get instance info: %w", err)
	}

	publicIP := "N/A (private subnet)"
	if instanceInfo.PublicIpAddress != nil {
		publicIP = *instanceInfo.PublicIpAddress
	}
	privateIP := *instanceInfo.PrivateIpAddress

	fmt.Println("\n🎉 Instance launched successfully!")
	fmt.Printf("Instance ID: %s\n", instanceId)
	fmt.Printf("Instance Type: %s\n", env.InstanceType)
	fmt.Printf("Public IP: %s\n", publicIP)
	fmt.Printf("Private IP: %s\n", privateIP)
	fmt.Printf("Subnet: %s (%s)\n", subnet.ID, subnetType)

	if connectionMethod == "ssh" {
		fmt.Printf("SSH Key: %s\n", keyInfo.Name)
		fmt.Println("\n🔗 To connect:")
		if subnet.IsPublic {
			fmt.Printf("ssh -i ~/.aws-jupyter/keys/%s.pem ec2-user@%s\n", keyInfo.Name, publicIP)
		} else {
			fmt.Println("Use Session Manager or VPN/bastion to connect to private instance")
		}
	} else {
		fmt.Println("\n🔗 To connect:")
		fmt.Printf("aws ssm start-session --target %s --profile %s\n", instanceId, profile)
	}

	fmt.Println("\n📓 Jupyter Lab will be available at: http://localhost:8888")
	fmt.Println("Use 'aws-jupyter connect' to setup port forwarding")

	// TODO: Save instance state locally
	// TODO: Setup port forwarding

	return nil
}
