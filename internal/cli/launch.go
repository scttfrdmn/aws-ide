package cli

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/scttfrdmn/aws-jupyter/internal/aws"
	"github.com/scttfrdmn/aws-jupyter/internal/config"
	"github.com/spf13/cobra"
)

const (
	connectionMethodSSH            = "ssh"
	connectionMethodSessionManager = "session-manager"
	subnetTypePublic               = "public"
	subnetTypePrivate              = "private"
)

// NewLaunchCmd creates the launch command for starting new Jupyter instances
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

	// Load and validate environment configuration
	env, err := loadAndValidateEnvironment(environment, instanceType)
	if err != nil {
		return err
	}

	// Validate launch options
	if err := validateLaunchOptions(connectionMethod, subnetType); err != nil {
		return err
	}

	// Display warnings and information
	displayLaunchWarnings(connectionMethod, subnetType, createNatGateway)

	if dryRun {
		return executeDryRun(ctx, env, profile, region, idleTimeout, connectionMethod, subnetType, createNatGateway)
	}

	return executeLaunch(ctx, env, profile, region, connectionMethod, subnetType, createNatGateway)
}

// loadAndValidateEnvironment loads the environment configuration and applies overrides
func loadAndValidateEnvironment(environment, instanceType string) (*config.Environment, error) {
	env, err := config.LoadEnvironment(environment)
	if err != nil {
		return nil, fmt.Errorf("failed to load environment: %w", err)
	}

	// Override instance type if provided
	if instanceType != "" {
		env.InstanceType = instanceType
	}

	return env, nil
}

// validateLaunchOptions validates connection method and subnet type
func validateLaunchOptions(connectionMethod, subnetType string) error {
	if connectionMethod != connectionMethodSSH && connectionMethod != connectionMethodSessionManager {
		return fmt.Errorf("connection method must be '%s' or '%s'", connectionMethodSSH, connectionMethodSessionManager)
	}
	if subnetType != subnetTypePublic && subnetType != subnetTypePrivate {
		return fmt.Errorf("subnet type must be '%s' or '%s'", subnetTypePublic, subnetTypePrivate)
	}
	return nil
}

// displayLaunchWarnings shows relevant warnings about the selected configuration
func displayLaunchWarnings(connectionMethod, subnetType string, createNatGateway bool) {
	// Warn about private subnet implications
	if subnetType == subnetTypePrivate && !createNatGateway {
		fmt.Println("⚠️  Warning: Private subnet without NAT Gateway means limited internet access")
		fmt.Println("   - Package installations may fail")
		fmt.Println("   - Jupyter extensions may not work")
		fmt.Println("   - Consider using --create-nat-gateway for full functionality")
	}

	// Session Manager information
	if connectionMethod == connectionMethodSessionManager {
		fmt.Println("ℹ️  Using Session Manager connection (no SSH keys needed)")
		if subnetType == subnetTypePublic {
			fmt.Println("   - Instance will be in public subnet but without SSH access")
		}
	}
}

// executeDryRun performs a dry run and displays what would be done
func executeDryRun(ctx context.Context, env *config.Environment, profile, region, idleTimeout, connectionMethod, subnetType string, createNatGateway bool) error {
	ec2Client, err := aws.NewEC2Client(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create AWS client for dry run: %w", err)
	}

	actualRegion := determineRegion(ec2Client, region)
	keyName := aws.DefaultKeyPairStrategy(actualRegion).GetDefaultKeyName()

	printDryRunConfiguration(env, actualRegion, profile, region, idleTimeout, connectionMethod, subnetType, createNatGateway, keyName)
	printDryRunActions(env, connectionMethod, subnetType, createNatGateway, keyName)

	fmt.Println("[DRY RUN] No resources were created")
	return nil
}

// executeLaunch performs the actual instance launch
func executeLaunch(ctx context.Context, env *config.Environment, profile, region, connectionMethod, subnetType string, createNatGateway bool) error {
	fmt.Printf("Launching %s environment on %s...\n", env.Name, env.InstanceType)

	// Setup AWS client and determine region
	ec2Client, actualRegion, err := setupAWSClient(ctx, profile, region)
	if err != nil {
		return err
	}

	// Setup authentication (SSH or Session Manager)
	keyInfo, instanceProfile, err := setupAuthentication(ctx, ec2Client, profile, actualRegion, connectionMethod)
	if err != nil {
		return err
	}

	// Setup networking (subnet and NAT gateway)
	subnet, err := setupNetworking(ctx, ec2Client, subnetType, createNatGateway)
	if err != nil {
		return err
	}

	// Setup security group
	securityGroup, err := setupSecurityGroup(ctx, ec2Client, subnet.VpcID, connectionMethod)
	if err != nil {
		return err
	}

	// Select AMI and generate user data
	amiID, userData, err := prepareInstanceImage(ctx, ec2Client, env, actualRegion)
	if err != nil {
		return err
	}

	// Launch and wait for instance
	instance, err := launchAndWaitForInstance(ctx, ec2Client, env, subnet, securityGroup, amiID, userData, keyInfo, instanceProfile, connectionMethod)
	if err != nil {
		return err
	}

	// Display connection information
	return displayInstanceInfo(instance, env, subnet, keyInfo, connectionMethod, subnetType, profile)
}

// determineRegion returns the actual region to use
func determineRegion(ec2Client *aws.EC2Client, regionOverride string) string {
	actualRegion := ec2Client.GetRegion()
	if regionOverride != "" {
		actualRegion = regionOverride
	}
	return actualRegion
}

// setupAWSClient creates and configures the AWS EC2 client
func setupAWSClient(ctx context.Context, profile, region string) (*aws.EC2Client, string, error) {
	ec2Client, err := aws.NewEC2Client(ctx, profile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create AWS client: %w", err)
	}

	actualRegion := ec2Client.GetRegion()
	if region != "" {
		actualRegion = region
		fmt.Printf("Note: Region override (%s) not yet implemented, using profile region (%s)\n", region, actualRegion)
	}

	return ec2Client, actualRegion, nil
}

// setupAuthentication configures SSH or Session Manager authentication
func setupAuthentication(ctx context.Context, ec2Client *aws.EC2Client, profile, region, connectionMethod string) (*aws.KeyPairInfo, *aws.InstanceProfileInfo, error) {
	if connectionMethod == connectionMethodSSH {
		return setupSSHAuthentication(ctx, ec2Client, region)
	}
	return setupSessionManagerAuthentication(ctx, profile)
}

// setupSSHAuthentication configures SSH key pair
func setupSSHAuthentication(ctx context.Context, ec2Client *aws.EC2Client, region string) (*aws.KeyPairInfo, *aws.InstanceProfileInfo, error) {
	fmt.Println("🔑 Setting up SSH key pair...")

	keyStorage, err := config.DefaultKeyStorage()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize key storage: %w", err)
	}

	keyStrategy := aws.DefaultKeyPairStrategy(region)
	keyInfo, err := ec2Client.GetOrCreateKeyPair(ctx, keyStrategy)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup SSH key pair: %w", err)
	}

	fmt.Printf("Using SSH key pair: %s\n", keyInfo.Name)

	if keyInfo.PrivateKey != "" {
		fmt.Println("Saving SSH private key locally...")
		if err := keyStorage.SavePrivateKey(keyInfo); err != nil {
			return nil, nil, fmt.Errorf("failed to save SSH private key: %w", err)
		}
		fmt.Printf("SSH private key saved to: %s\n", keyStorage.GetKeyPath(keyInfo.Name))
	} else {
		if !keyStorage.HasPrivateKey(keyInfo.Name) {
			return nil, nil, fmt.Errorf("SSH key pair '%s' exists in AWS but private key not found locally", keyInfo.Name)
		}
		fmt.Printf("Using existing local private key: %s\n", keyStorage.GetKeyPath(keyInfo.Name))
	}

	return keyInfo, nil, nil
}

// setupSessionManagerAuthentication configures Session Manager IAM role
func setupSessionManagerAuthentication(ctx context.Context, profile string) (*aws.KeyPairInfo, *aws.InstanceProfileInfo, error) {
	fmt.Println("🔐 Setting up Session Manager IAM role...")

	iamClient, err := aws.NewIAMClient(ctx, profile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create IAM client: %w", err)
	}

	instanceProfile, err := iamClient.GetOrCreateSessionManagerRole(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup Session Manager role: %w", err)
	}

	fmt.Printf("Using IAM instance profile: %s\n", instanceProfile.Name)
	return nil, instanceProfile, nil
}

// setupNetworking configures subnet and NAT gateway
func setupNetworking(ctx context.Context, ec2Client *aws.EC2Client, subnetType string, createNatGateway bool) (*aws.SubnetInfo, error) {
	fmt.Printf("🌐 Selecting %s subnet...\n", subnetType)

	subnet, err := ec2Client.GetSubnet(ctx, subnetType, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get subnet: %w", err)
	}
	fmt.Printf("Using subnet: %s (%s) in %s\n", subnet.ID, subnet.CidrBlock, subnet.AvailabilityZone)

	if subnetType == subnetTypePrivate && createNatGateway {
		if err := setupNATGateway(ctx, ec2Client, subnet); err != nil {
			return nil, err
		}
	}

	return subnet, nil
}

// setupNATGateway creates or retrieves NAT gateway and updates routes
func setupNATGateway(ctx context.Context, ec2Client *aws.EC2Client, subnet *aws.SubnetInfo) error {
	fmt.Println("🚪 Setting up NAT Gateway for internet access...")

	natGateway, err := ec2Client.GetOrCreateNATGateway(ctx, subnet.VpcID)
	if err != nil {
		return fmt.Errorf("failed to setup NAT Gateway: %w", err)
	}

	if err := ec2Client.UpdatePrivateSubnetRoutes(ctx, subnet.ID, natGateway.ID); err != nil {
		return fmt.Errorf("failed to update subnet routes: %w", err)
	}

	return nil
}

// setupSecurityGroup creates or retrieves the security group
func setupSecurityGroup(ctx context.Context, ec2Client *aws.EC2Client, vpcID, connectionMethod string) (*aws.SecurityGroupInfo, error) {
	fmt.Println("🔒 Setting up security group...")

	sgStrategy := aws.DefaultSecurityGroupStrategy(vpcID)
	if connectionMethod == connectionMethodSessionManager {
		sgStrategy.DefaultName = "aws-jupyter-session-manager"
	}

	securityGroup, err := ec2Client.GetOrCreateSecurityGroup(ctx, sgStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to setup security group: %w", err)
	}

	fmt.Printf("Using security group: %s (%s)\n", securityGroup.Name, securityGroup.ID)
	return securityGroup, nil
}

// prepareInstanceImage selects AMI and generates user data
func prepareInstanceImage(ctx context.Context, ec2Client *aws.EC2Client, env *config.Environment, region string) (string, string, error) {
	fmt.Println("🔍 Selecting AMI for environment...")

	amiSelector := aws.NewAMISelector(region)
	amiID, err := amiSelector.GetAMI(ctx, ec2Client, env.AMIBase)
	if err != nil {
		fmt.Printf("Warning: Could not find latest AMI (%v), using fallback\n", err)
		amiID = amiSelector.GetDefaultAMI()
		fmt.Printf("Using fallback AMI: %s\n", amiID)
	}

	fmt.Println("📜 Generating user data script...")
	userData, err := config.GenerateUserData(env)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate user data: %w", err)
	}

	return amiID, userData, nil
}

// launchAndWaitForInstance launches the EC2 instance and waits for it to be running
func launchAndWaitForInstance(ctx context.Context, ec2Client *aws.EC2Client, env *config.Environment, subnet *aws.SubnetInfo, securityGroup *aws.SecurityGroupInfo, amiID, userData string, keyInfo *aws.KeyPairInfo, instanceProfile *aws.InstanceProfileInfo, connectionMethod string) (*types.Instance, error) {
	fmt.Printf("🚀 Launching EC2 instance (%s)...\n", env.InstanceType)

	launchParams := aws.LaunchParams{
		AMI:             amiID,
		InstanceType:    env.InstanceType,
		SecurityGroupID: securityGroup.ID,
		UserData:        userData,
		EBSVolumeSize:   env.EBSVolumeSize,
		Environment:     env.Name,
		SubnetID:        subnet.ID,
	}

	if connectionMethod == connectionMethodSSH {
		launchParams.KeyPairName = keyInfo.Name
	} else {
		launchParams.InstanceProfile = instanceProfile.Name
	}

	instance, err := ec2Client.LaunchInstance(ctx, launchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to launch instance: %w", err)
	}

	instanceID := *instance.InstanceId
	fmt.Printf("✓ Instance launched: %s\n", instanceID)

	fmt.Println("⏳ Waiting for instance to be running...")
	if err := ec2Client.WaitForInstanceRunning(ctx, instanceID); err != nil {
		return nil, fmt.Errorf("instance failed to start: %w", err)
	}

	return ec2Client.GetInstanceInfo(ctx, instanceID)
}

// displayInstanceInfo shows the launched instance information
func displayInstanceInfo(instance *types.Instance, env *config.Environment, subnet *aws.SubnetInfo, keyInfo *aws.KeyPairInfo, connectionMethod, subnetType, profile string) error {
	publicIP := "N/A (private subnet)"
	if instance.PublicIpAddress != nil {
		publicIP = *instance.PublicIpAddress
	}
	privateIP := *instance.PrivateIpAddress
	instanceID := *instance.InstanceId

	fmt.Println("\n🎉 Instance launched successfully!")
	fmt.Printf("Instance ID: %s\n", instanceID)
	fmt.Printf("Instance Type: %s\n", env.InstanceType)
	fmt.Printf("Public IP: %s\n", publicIP)
	fmt.Printf("Private IP: %s\n", privateIP)
	fmt.Printf("Subnet: %s (%s)\n", subnet.ID, subnetType)

	if connectionMethod == connectionMethodSSH {
		fmt.Printf("SSH Key: %s\n", keyInfo.Name)
		fmt.Println("\n🔗 To connect:")
		if subnet.IsPublic {
			fmt.Printf("ssh -i ~/.aws-jupyter/keys/%s.pem ec2-user@%s\n", keyInfo.Name, publicIP)
		} else {
			fmt.Println("Use Session Manager or VPN/bastion to connect to private instance")
		}
	} else {
		fmt.Println("\n🔗 To connect:")
		fmt.Printf("aws ssm start-session --target %s --profile %s\n", instanceID, profile)
	}

	fmt.Println("\n📓 Jupyter Lab will be available at: http://localhost:8888")
	fmt.Println("Use 'aws-jupyter connect' to setup port forwarding")

	return nil
}

// printDryRunConfiguration displays the dry run configuration
func printDryRunConfiguration(env *config.Environment, actualRegion, profile, region, idleTimeout, connectionMethod, subnetType string, createNatGateway bool, keyName string) {
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
	if createNatGateway && subnetType == subnetTypePrivate {
		fmt.Printf("  - NAT Gateway: will be created (additional cost)\n")
	}
	if connectionMethod == connectionMethodSSH {
		fmt.Printf("  - SSH Key Pair: %s (economical reuse)\n", keyName)
	} else {
		fmt.Printf("  - Session Manager: IAM role will be created/attached\n")
	}
}

// printDryRunActions displays the actions that would be performed
func printDryRunActions(env *config.Environment, connectionMethod, subnetType string, createNatGateway bool, keyName string) {
	fmt.Printf("[DRY RUN] Would perform these actions:\n")
	actionNum := 1

	if connectionMethod == connectionMethodSSH {
		fmt.Printf("  %d. Create/verify SSH key pair (%s)\n", actionNum, keyName)
	} else {
		fmt.Printf("  %d. Create/verify IAM role for Session Manager\n", actionNum)
	}
	actionNum++

	if connectionMethod == connectionMethodSSH {
		fmt.Printf("  %d. Create/verify security group (SSH + Jupyter access)\n", actionNum)
	} else {
		fmt.Printf("  %d. Create/verify security group (Jupyter access only)\n", actionNum)
	}
	actionNum++

	if subnetType == subnetTypePrivate && createNatGateway {
		fmt.Printf("  %d. Create/verify NAT Gateway for internet access\n", actionNum)
		actionNum++
	}

	fmt.Printf("  %d. Generate user data script for environment setup\n", actionNum)
	actionNum++
	fmt.Printf("  %d. Launch EC2 instance (%s) in %s subnet\n", actionNum, env.InstanceType, subnetType)
	actionNum++
	fmt.Printf("  %d. Wait for instance to be running\n", actionNum)
	actionNum++

	if connectionMethod == connectionMethodSSH {
		fmt.Printf("  %d. Setup SSH tunnel (port 8888)\n", actionNum)
	} else {
		fmt.Printf("  %d. Setup Session Manager port forwarding (port 8888)\n", actionNum)
	}
	actionNum++

	fmt.Printf("  %d. Save instance state locally\n", actionNum)
	actionNum++
	fmt.Printf("  %d. Display connection information\n", actionNum)
}
