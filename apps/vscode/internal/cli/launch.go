package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	awssdkconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	vscodeconfig "github.com/scttfrdmn/aws-ide/apps/vscode/internal/config"
	"github.com/scttfrdmn/aws-ide/pkg/aws"
	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/scttfrdmn/aws-ide/pkg/readiness"
	"github.com/spf13/cobra"
)

const (
	connectionMethodSSH            = "ssh"
	connectionMethodSessionManager = "session-manager"
	subnetTypePublic               = "public"
	subnetTypePrivate              = "private"
)

// NewLaunchCmd creates the launch command for starting new VSCode Server instances
func NewLaunchCmd() *cobra.Command {
	var (
		environment      string
		instanceType     string
		customAMI        string
		idleTimeout      string
		profile          string
		region           string
		availabilityZone string
		dryRun           bool
		connectionMethod string
		subnetType       string
		createNatGateway bool
	)

	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Launch a new VSCode Server instance",
		Long: `Launch a new EC2 instance with VSCode Server (code-server) configured and ready to use.

The instance will be configured based on the selected environment preset, which determines:
- System packages to install
- Language runtimes (Node.js, Python, Go)
- VSCode extensions
- Development tools and utilities

Available environments: web-dev, python-dev, go-dev, fullstack`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLaunch(environment, instanceType, customAMI, idleTimeout, profile, region, availabilityZone, dryRun, connectionMethod, subnetType, createNatGateway)
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "web-dev", "Environment configuration to use")
	cmd.Flags().StringVarP(&instanceType, "instance-type", "t", "", "Override instance type")
	cmd.Flags().StringVar(&customAMI, "ami", "", "Use custom AMI instead of base AMI")
	cmd.Flags().StringVarP(&idleTimeout, "idle-timeout", "i", "4h", "Auto-shutdown timeout")
	cmd.Flags().StringVarP(&profile, "profile", "p", "default", "AWS profile to use")
	cmd.Flags().StringVarP(&region, "region", "r", "", "AWS region")
	cmd.Flags().StringVarP(&availabilityZone, "availability-zone", "z", "", "Availability zone (e.g., us-east-1a)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	cmd.Flags().StringVarP(&connectionMethod, "connection", "c", "ssh", "Connection method: ssh or session-manager")
	cmd.Flags().StringVarP(&subnetType, "subnet-type", "s", "public", "Subnet type: public or private")
	cmd.Flags().BoolVar(&createNatGateway, "create-nat-gateway", false, "Create NAT Gateway for private subnet internet access")

	return cmd
}

// parseDuration converts duration strings like "3m", "1h", "4h" to seconds
func parseDuration(s string) (int, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	unit := s[len(s)-1:]
	valueStr := s[:len(s)-1]
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", s)
	}

	switch unit {
	case "s":
		return value, nil
	case "m":
		return value * 60, nil
	case "h":
		return value * 3600, nil
	case "d":
		return value * 86400, nil
	default:
		return 0, fmt.Errorf("invalid duration unit: %s (use s, m, h, or d)", unit)
	}
}

func runLaunch(environment, instanceType, customAMI, idleTimeout, profile, region, availabilityZone string, dryRun bool, connectionMethod, subnetType string, createNatGateway bool) error {
	ctx := context.Background()

	// Load and validate environment configuration
	env, err := loadAndValidateEnvironment(environment, instanceType)
	if err != nil {
		return err
	}

	// Parse idle timeout
	idleTimeoutSeconds, err := parseDuration(idleTimeout)
	if err != nil {
		return fmt.Errorf("failed to parse idle timeout: %w", err)
	}

	// Validate launch options
	if err := validateLaunchOptions(connectionMethod, subnetType); err != nil {
		return err
	}

	// Display warnings and information
	displayLaunchWarnings(connectionMethod, subnetType, createNatGateway)

	if dryRun {
		return executeDryRun(ctx, env, profile, region, availabilityZone, idleTimeout, connectionMethod, subnetType, createNatGateway)
	}

	return executeLaunch(ctx, env, customAMI, profile, region, availabilityZone, idleTimeoutSeconds, connectionMethod, subnetType, createNatGateway)
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
		fmt.Println("   - VSCode extensions may not install")
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
func executeDryRun(ctx context.Context, env *config.Environment, profile, region, availabilityZone, idleTimeout, connectionMethod, subnetType string, createNatGateway bool) error {
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
func executeLaunch(ctx context.Context, env *config.Environment, customAMI, profile, region, availabilityZone string, idleTimeoutSeconds int, connectionMethod, subnetType string, createNatGateway bool) error {
	fmt.Printf("Launching %s environment on %s...\n", env.Name, env.InstanceType)

	// Setup AWS clients and determine region
	ec2Client, ssmClient, actualRegion, err := setupAWSClient(ctx, profile, region)
	if err != nil {
		return err
	}

	// Setup IAM instance profile (always, for SSM access)
	instanceProfile, err := setupInstanceProfile(ctx, profile)
	if err != nil {
		return err
	}

	// Setup SSH key if needed
	var keyInfo *aws.KeyPairInfo
	if connectionMethod == connectionMethodSSH {
		keyInfo, err = setupSSHKey(ctx, ec2Client, actualRegion)
		if err != nil {
			return err
		}
	}

	// Setup networking (subnet and NAT gateway)
	subnet, err := setupNetworking(ctx, ec2Client, env.InstanceType, subnetType, availabilityZone, createNatGateway)
	if err != nil {
		return err
	}

	// Setup security group
	securityGroup, err := setupSecurityGroup(ctx, ec2Client, subnet.VpcID, connectionMethod)
	if err != nil {
		return err
	}

	// Select AMI and generate user data
	amiID, userData, err := prepareInstanceImage(ctx, ec2Client, env, actualRegion, customAMI, idleTimeoutSeconds)
	if err != nil {
		return err
	}

	// Launch and wait for instance
	instance, err := launchAndWaitForInstance(ctx, ec2Client, ssmClient, env, subnet, securityGroup, amiID, userData, keyInfo, instanceProfile)
	if err != nil {
		return err
	}

	// Display connection information
	return displayVSCodeInfo(instance, env, subnet, keyInfo, connectionMethod, subnetType, profile)
}

// determineRegion returns the actual region to use
func determineRegion(ec2Client *aws.EC2Client, regionOverride string) string {
	actualRegion := ec2Client.GetRegion()
	if regionOverride != "" {
		actualRegion = regionOverride
	}
	return actualRegion
}

// setupAWSClient creates and configures the AWS EC2 and SSM clients
func setupAWSClient(ctx context.Context, profile, region string) (*aws.EC2Client, *aws.SSMClient, string, error) {
	ec2Client, err := aws.NewEC2Client(ctx, profile)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create AWS client: %w", err)
	}

	actualRegion := ec2Client.GetRegion()
	if region != "" {
		actualRegion = region
		fmt.Printf("Note: Region override (%s) not yet implemented, using profile region (%s)\n", region, actualRegion)
	}

	// Create SSM client for readiness checks (reuse config loading)
	cfg, err := awssdkconfig.LoadDefaultConfig(ctx,
		awssdkconfig.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to load AWS config for SSM: %w", err)
	}
	ssmClient := aws.NewSSMClient(cfg)

	return ec2Client, ssmClient, actualRegion, nil
}

// setupInstanceProfile configures IAM instance profile with SSM permissions
func setupInstanceProfile(ctx context.Context, profile string) (*aws.InstanceProfileInfo, error) {
	fmt.Println("🔐 Setting up IAM instance profile with SSM permissions...")

	iamClient, err := aws.NewIAMClient(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM client: %w", err)
	}

	instanceProfile, err := iamClient.GetOrCreateSessionManagerRole(ctx, "aws-vscode")
	if err != nil {
		return nil, fmt.Errorf("failed to setup Session Manager role: %w", err)
	}

	fmt.Printf("Using IAM instance profile: %s\n", instanceProfile.Name)
	return instanceProfile, nil
}

// setupSSHKey configures SSH key pair
func setupSSHKey(ctx context.Context, ec2Client *aws.EC2Client, region string) (*aws.KeyPairInfo, error) {
	fmt.Println("🔑 Setting up SSH key pair...")

	keyStorage, err := config.DefaultKeyStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize key storage: %w", err)
	}

	keyStrategy := aws.DefaultKeyPairStrategy(region)
	keyInfo, err := ec2Client.GetOrCreateKeyPair(ctx, keyStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to setup SSH key pair: %w", err)
	}

	fmt.Printf("Using SSH key pair: %s\n", keyInfo.Name)

	if keyInfo.PrivateKey != "" {
		fmt.Println("Saving SSH private key locally...")
		if err := keyStorage.SavePrivateKey(keyInfo); err != nil {
			return nil, fmt.Errorf("failed to save SSH private key: %w", err)
		}
		fmt.Printf("SSH private key saved to: %s\n", keyStorage.GetKeyPath(keyInfo.Name))
	} else {
		if !keyStorage.HasPrivateKey(keyInfo.Name) {
			return nil, fmt.Errorf("SSH key pair '%s' exists in AWS but private key not found locally", keyInfo.Name)
		}
		fmt.Printf("Using existing local private key: %s\n", keyStorage.GetKeyPath(keyInfo.Name))
	}

	return keyInfo, nil
}

// setupNetworking configures subnet and NAT gateway
func setupNetworking(ctx context.Context, ec2Client *aws.EC2Client, instanceType, subnetType, availabilityZone string, createNatGateway bool) (*aws.SubnetInfo, error) {
	fmt.Printf("🌐 Selecting %s subnet...\n", subnetType)

	// If no availability zone specified, find one that supports the instance type
	if availabilityZone == "" {
		fmt.Printf("Finding availability zone that supports %s...\n", instanceType)
		compatibleAZ, err := ec2Client.FindCompatibleAvailabilityZone(ctx, instanceType, subnetType)
		if err != nil {
			return nil, fmt.Errorf("failed to find compatible availability zone: %w", err)
		}
		availabilityZone = compatibleAZ
		fmt.Printf("Selected availability zone: %s\n", availabilityZone)
	} else {
		// If availability zone specified, validate instance type support
		fmt.Printf("Validating instance type %s in availability zone %s...\n", instanceType, availabilityZone)
		supported, err := ec2Client.IsInstanceTypeSupported(ctx, instanceType, availabilityZone)
		if err != nil {
			return nil, fmt.Errorf("failed to validate instance type: %w", err)
		}
		if !supported {
			return nil, fmt.Errorf("instance type %s is not supported in availability zone %s", instanceType, availabilityZone)
		}
	}

	subnet, err := ec2Client.GetSubnet(ctx, subnetType, availabilityZone)
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
		sgStrategy.DefaultName = "aws-vscode-session-manager"
	} else {
		sgStrategy.DefaultName = "aws-vscode"
	}

	securityGroup, err := ec2Client.GetOrCreateSecurityGroup(ctx, sgStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to setup security group: %w", err)
	}

	fmt.Printf("Using security group: %s (%s)\n", securityGroup.Name, securityGroup.ID)
	return securityGroup, nil
}

// prepareInstanceImage selects AMI and generates user data
func prepareInstanceImage(ctx context.Context, ec2Client *aws.EC2Client, env *config.Environment, region, customAMI string, idleTimeoutSeconds int) (string, string, error) {
	var amiID string

	// Use custom AMI if provided, otherwise select base AMI
	if customAMI != "" {
		fmt.Printf("🔍 Using custom AMI: %s\n", customAMI)
		amiID = customAMI
	} else {
		fmt.Println("🔍 Selecting base AMI for environment...")
		amiSelector := aws.NewAMISelector(region)
		var err error
		amiID, err = amiSelector.GetAMI(ctx, ec2Client, env.AMIBase)
		if err != nil {
			return "", "", fmt.Errorf("failed to find AMI: %w", err)
		}
	}

	fmt.Println("📜 Generating user data script...")
	userData, err := vscodeconfig.GenerateUserData(env, idleTimeoutSeconds)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate user data: %w", err)
	}

	return amiID, userData, nil
}

// launchAndWaitForInstance launches the EC2 instance and waits for it to be running
func launchAndWaitForInstance(ctx context.Context, ec2Client *aws.EC2Client, ssmClient *aws.SSMClient, env *config.Environment, subnet *aws.SubnetInfo, securityGroup *aws.SecurityGroupInfo, amiID, userData string, keyInfo *aws.KeyPairInfo, instanceProfile *aws.InstanceProfileInfo) (*types.Instance, error) {
	fmt.Printf("🚀 Launching EC2 instance (%s)...\n", env.InstanceType)

	launchParams := aws.LaunchParams{
		AMI:             amiID,
		InstanceType:    env.InstanceType,
		SecurityGroupID: securityGroup.ID,
		UserData:        userData,
		EBSVolumeSize:   env.EBSVolumeSize,
		Environment:     env.Name,
		SubnetID:        subnet.ID,
		InstanceProfile: instanceProfile.Name,
	}

	// Add SSH key if provided
	if keyInfo != nil {
		launchParams.KeyPairName = keyInfo.Name
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

	// Get updated instance info with IP address
	instance, err = ec2Client.GetInstanceInfo(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance info: %w", err)
	}

	// Poll for VSCode Server readiness with progress streaming
	fmt.Println("⏳ Instance is running, waiting for VSCode Server to be ready...")
	fmt.Println("   (This typically takes 2-3 minutes for code-server installation)")
	fmt.Println()

	// Try to stream progress if we have SSH access
	progressDone := make(chan bool)
	go streamSetupProgress(ctx, instance, keyInfo, progressDone)

	if err := waitForVSCodeReady(ctx, ssmClient, instance); err != nil {
		close(progressDone) // Stop progress streaming
		fmt.Printf("\n⚠️  Warning: %v\n", err)
		fmt.Println("   You can still try connecting - the service may still be starting up.")
	} else {
		close(progressDone) // Stop progress streaming
	}

	return instance, nil
}

// displayVSCodeInfo displays VSCode Server-specific connection information
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

	if connectionMethod == connectionMethodSSH {
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
	fmt.Printf("\nOr use 'aws-vscode connect %s' to setup port forwarding\n", instanceID)

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

// printDryRunConfiguration displays the dry run configuration
func printDryRunConfiguration(env *config.Environment, actualRegion, profile, region, idleTimeout, connectionMethod, subnetType string, createNatGateway bool, keyName string) {
	fmt.Printf("[DRY RUN] Would launch %s environment on %s in region %s\n", env.Name, env.InstanceType, actualRegion)
	fmt.Printf("[DRY RUN] Configuration:\n")
	fmt.Printf("  - Environment: %s\n", env.Name)
	fmt.Printf("  - Instance Type: %s\n", env.InstanceType)
	fmt.Printf("  - AMI Base: %s\n", env.AMIBase)
	fmt.Printf("  - EBS Volume: %dGB\n", env.EBSVolumeSize)
	fmt.Printf("  - Packages: %d system packages\n", len(env.Packages))
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
		fmt.Printf("  %d. Create/verify security group (SSH + VSCode access)\n", actionNum)
	} else {
		fmt.Printf("  %d. Create/verify security group (VSCode access only)\n", actionNum)
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

	fmt.Printf("  %d. Save instance state locally\n", actionNum)
	actionNum++
	fmt.Printf("  %d. Display connection information\n", actionNum)
}

// streamSetupProgress streams setup progress by tailing the progress log via SSH
func streamSetupProgress(ctx context.Context, instance *types.Instance, keyInfo *aws.KeyPairInfo, done chan bool) {
	// Can only stream progress if we have SSH access
	if keyInfo == nil || instance.PublicIpAddress == nil {
		return
	}

	host := *instance.PublicIpAddress
	keyStorage, err := config.DefaultKeyStorage()
	if err != nil {
		return
	}

	keyPath := keyStorage.GetKeyPath(keyInfo.Name)

	// Wait a bit for instance to be accessible and log file to be created
	time.Sleep(30 * time.Second)

	// Tail the progress log
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastLine := ""
	seenSteps := make(map[string]bool)

	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Try to read the progress log via SSH
			cmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o ConnectTimeout=3 -o BatchMode=yes ubuntu@%s 'tail -20 /var/log/setup-progress.log 2>/dev/null || echo \"\"' 2>/dev/null",
				keyPath, host)

			output, err := exec.CommandContext(ctx, "bash", "-c", cmd).Output()
			if err != nil {
				continue
			}

			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || line == lastLine {
					continue
				}

				if strings.HasPrefix(line, "STEP:") {
					step := strings.TrimPrefix(line, "STEP:")
					if !seenSteps[step] {
						fmt.Printf("   📋 %s\n", step)
						seenSteps[step] = true
					}
				} else if line == "COMPLETE" {
					fmt.Println("   ✅ Setup complete!")
					return
				}
				lastLine = line
			}
		}
	}
}

// waitForVSCodeReady polls the VSCode Server until it's ready to accept connections via SSM
func waitForVSCodeReady(ctx context.Context, ssmClient *aws.SSMClient, instance *types.Instance) error {
	instanceID := *instance.InstanceId

	// VSCode Server (code-server) runs on port 8080
	config := readiness.SSMServiceConfig{
		InstanceID: instanceID,
		Port:       8080,
		Timeout:    5 * time.Minute,  // 5 minutes should be enough for installation
		Retry:      10 * time.Second, // Check every 10 seconds
	}

	// Track elapsed time for progress updates
	startTime := time.Now()
	lastUpdate := startTime

	callback := func(message string, elapsed time.Duration) {
		// Only print updates every 20 seconds to avoid spam
		now := time.Now()
		if now.Sub(lastUpdate) >= 20*time.Second || strings.Contains(message, "ready") || strings.Contains(message, "SSM") {
			fmt.Printf("   [%ds] %s\n", int(elapsed.Seconds()), message)
			lastUpdate = now
		}
	}

	result, err := readiness.PollServiceReadinessViaSSM(ctx, config, ssmClient, callback)
	if err != nil {
		return err
	}

	if result.Ready {
		fmt.Printf("✓ VSCode Server is ready! (took %v)\n", result.ElapsedTime.Round(time.Second))
		return nil
	}

	return fmt.Errorf("%s", result.Message)
}
