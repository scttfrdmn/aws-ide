package cli

// Future enhancements to consider:
// 1. Launch throttling - prevent too many concurrent launches or rapid successive launches
//    See: https://github.com/scttfrdmn/lens/issues/49
// 2. System sleep/wake detection - detect when local system goes to sleep/wakes up
//    and optionally hibernate remote Lens instances to save costs when user is away
//    See: https://github.com/scttfrdmn/lens/issues/50

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	awssdkconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	qgisconfig "github.com/scttfrdmn/lens/apps/qgis/internal/config"
	"github.com/scttfrdmn/lens/pkg/aws"
	"github.com/scttfrdmn/lens/pkg/config"
	"github.com/scttfrdmn/lens/pkg/output"
	"github.com/scttfrdmn/lens/pkg/readiness"
	"github.com/spf13/cobra"
)

const (
	connectionMethodSSH            = "ssh"
	connectionMethodSessionManager = "session-manager"
	subnetTypePublic               = "public"
	subnetTypePrivate              = "private"
)

// NewLaunchCmd creates the launch command for starting new QGIS instances
func NewLaunchCmd() *cobra.Command {
	var (
		environment      string
		instanceType     string
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
		Short: "Launch a new QGIS instance with DCV remote desktop",
		Long: `Launch QGIS on AWS EC2 with NICE DCV remote desktop access.

QGIS will be accessible via browser on https://localhost:8443 after
setting up port forwarding with the connect command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLaunch(environment, instanceType, idleTimeout, profile, region, availabilityZone, dryRun, connectionMethod, subnetType, createNatGateway)
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "basic-gis", "QGIS environment to use (basic-gis, advanced-gis, remote-sensing)")
	cmd.Flags().StringVarP(&instanceType, "instance-type", "t", "", "Override instance type")
	cmd.Flags().StringVarP(&idleTimeout, "idle-timeout", "i", "4h", "Auto-shutdown timeout")
	cmd.Flags().StringVarP(&profile, "profile", "p", "default", "AWS profile to use")
	cmd.Flags().StringVarP(&region, "region", "r", "", "AWS region")
	cmd.Flags().StringVarP(&availabilityZone, "availability-zone", "z", "", "Availability zone (e.g., us-east-1a)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	cmd.Flags().StringVarP(&connectionMethod, "connection", "c", "session-manager", "Connection method: ssh or session-manager")
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

func runLaunch(environment, instanceType, idleTimeout, profile, region, availabilityZone string, dryRun bool, connectionMethod, subnetType string, createNatGateway bool) error {
	ctx := context.Background()

	// Load QGIS environment configuration
	env, err := qgisconfig.Get(environment)
	if err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	// Override instance type if provided
	if instanceType != "" {
		env.InstanceType = instanceType
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

	// Display warnings
	displayLaunchWarnings(connectionMethod, subnetType, createNatGateway, env.RequiresGPU)

	if dryRun {
		return executeDryRun(ctx, env, profile, region, idleTimeout, connectionMethod, subnetType, createNatGateway)
	}

	return executeLaunch(ctx, env, profile, region, availabilityZone, idleTimeoutSeconds, connectionMethod, subnetType, createNatGateway)
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
func displayLaunchWarnings(connectionMethod, subnetType string, createNatGateway, requiresGPU bool) {
	out := output.DefaultFormatter()

	// GPU warning
	if requiresGPU {
		out.Info("This environment requires GPU acceleration (g4dn/g5 instances)")
		out.List("Higher hourly cost for GPU instances (~$0.53-$1.01/hr)")
		out.Blank()
	}

	// Warn about private subnet implications
	if subnetType == subnetTypePrivate && !createNatGateway {
		out.Warning("Private network without NAT Gateway means limited internet access")
		out.List("Package installations may fail")
		out.List("DCV remote desktop may not work properly")
		out.List("Consider using --create-nat-gateway for full functionality")
		out.Blank()
	}

	// Session Manager information
	if connectionMethod == connectionMethodSessionManager {
		out.Info("Using Session Manager connection (no SSH keys needed)")
		if subnetType == subnetTypePublic {
			out.List("Environment will be in public network but without SSH access")
		}
		out.Blank()
	}
}

// executeDryRun performs a dry run and displays what would be done
func executeDryRun(ctx context.Context, env *qgisconfig.QGISEnvironment, profile, region, idleTimeout, connectionMethod, subnetType string, createNatGateway bool) error {
	ec2Client, err := aws.NewEC2Client(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create AWS client for dry run: %w", err)
	}

	actualRegion := ec2Client.GetRegion()
	if region != "" {
		actualRegion = region
	}

	keyName := aws.DefaultKeyPairStrategy(actualRegion).GetDefaultKeyName()

	fmt.Printf("[DRY RUN] Would launch %s environment on %s in region %s\n", env.Name, env.InstanceType, actualRegion)
	fmt.Printf("[DRY RUN] Configuration:\n")
	fmt.Printf("  - Environment: %s\n", env.Name)
	fmt.Printf("  - Description: %s\n", env.Description)
	fmt.Printf("  - Instance Type: %s\n", env.InstanceType)
	fmt.Printf("  - GPU Required: %v\n", env.RequiresGPU)
	fmt.Printf("  - EBS Volume: %dGB\n", env.EBSVolumeSize)
	fmt.Printf("  - Packages: %d system packages\n", len(env.Packages))
	fmt.Printf("  - QGIS Plugins: %d plugins\n", len(env.QGISPlugins))
	fmt.Printf("  - DCV Port: %d (HTTPS remote desktop)\n", env.DCVConfig.Port)
	fmt.Printf("  - Idle Timeout: %s\n", idleTimeout)
	fmt.Printf("  - AWS Profile: %s\n", profile)
	fmt.Printf("  - AWS Region: %s\n", actualRegion)
	fmt.Printf("  - Connection Method: %s\n", connectionMethod)
	fmt.Printf("  - Subnet Type: %s\n", subnetType)

	if createNatGateway && subnetType == subnetTypePrivate {
		fmt.Printf("  - NAT Gateway: will be created (additional cost)\n")
	}
	if connectionMethod == connectionMethodSSH {
		fmt.Printf("  - SSH Key Pair: %s\n", keyName)
	}

	fmt.Println("\n[DRY RUN] No resources were created")
	return nil
}

// executeLaunch performs the actual instance launch
func executeLaunch(ctx context.Context, env *qgisconfig.QGISEnvironment, profile, region, availabilityZone string, idleTimeoutSeconds int, connectionMethod, subnetType string, createNatGateway bool) error {
	out := output.DefaultFormatter()
	out.Blank()
	out.Header(fmt.Sprintf("Launching QGIS %s on %s", env.Name, env.InstanceType))
	out.Blank()

	// Setup AWS clients
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

	// Setup networking
	subnet, err := setupNetworking(ctx, ec2Client, env.InstanceType, subnetType, availabilityZone, createNatGateway)
	if err != nil {
		return err
	}

	// Setup security group (needs DCV port 8443)
	securityGroup, err := setupSecurityGroup(ctx, ec2Client, subnet.VpcID, connectionMethod)
	if err != nil {
		return err
	}

	// Select AMI and generate user data
	amiID, userData, err := prepareInstanceImage(ctx, ec2Client, env, actualRegion, idleTimeoutSeconds)
	if err != nil {
		return err
	}

	// Launch and wait for instance
	instance, err := launchAndWaitForInstance(ctx, ec2Client, ssmClient, env, subnet, securityGroup, amiID, userData, keyInfo, instanceProfile)
	if err != nil {
		return err
	}

	// Display connection information
	return displayInstanceInfo(instance, env, subnet, keyInfo, connectionMethod, subnetType, profile)
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
	out := output.DefaultFormatter()
	out.Step("🔐", "Setting up secure access permissions")

	iamClient, err := aws.NewIAMClient(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM client: %w", err)
	}

	instanceProfile, err := iamClient.GetOrCreateSessionManagerRole(ctx, "lens-qgis")
	if err != nil {
		return nil, fmt.Errorf("failed to setup Session Manager role: %w", err)
	}

	out.SuccessWithDetail("IAM profile configured", instanceProfile.Name)
	return instanceProfile, nil
}

// setupSSHKey configures SSH key pair
func setupSSHKey(ctx context.Context, ec2Client *aws.EC2Client, region string) (*aws.KeyPairInfo, error) {
	out := output.DefaultFormatter()
	out.Step("🔑", "Configuring SSH access")

	keyStorage, err := config.DefaultKeyStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize key storage: %w", err)
	}

	keyStrategy := aws.DefaultKeyPairStrategy(region)
	keyInfo, err := ec2Client.GetOrCreateKeyPair(ctx, keyStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to setup SSH key pair: %w", err)
	}

	if keyInfo.PrivateKey != "" {
		if err := keyStorage.SavePrivateKey(keyInfo); err != nil {
			return nil, fmt.Errorf("failed to save SSH private key: %w", err)
		}
		out.SuccessWithDetail("SSH key created and saved", keyStorage.GetKeyPath(keyInfo.Name))
	} else {
		if !keyStorage.HasPrivateKey(keyInfo.Name) {
			return nil, fmt.Errorf("SSH key pair '%s' exists in AWS but private key not found locally", keyInfo.Name)
		}
		out.SuccessWithDetail("Using existing SSH key", keyStorage.GetKeyPath(keyInfo.Name))
	}

	return keyInfo, nil
}

// setupNetworking configures subnet and NAT gateway
func setupNetworking(ctx context.Context, ec2Client *aws.EC2Client, instanceType, subnetType, availabilityZone string, createNatGateway bool) (*aws.SubnetInfo, error) {
	out := output.DefaultFormatter()
	out.Step("🌐", fmt.Sprintf("Configuring network (%s subnet)", subnetType))

	if availabilityZone == "" {
		compatibleAZ, err := ec2Client.FindCompatibleAvailabilityZone(ctx, instanceType, subnetType)
		if err != nil {
			return nil, fmt.Errorf("failed to find compatible availability zone: %w", err)
		}
		availabilityZone = compatibleAZ
	} else {
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
	out.SuccessWithDetail("Network configured", fmt.Sprintf("%s in %s", subnet.CidrBlock, subnet.AvailabilityZone))

	if subnetType == subnetTypePrivate && createNatGateway {
		if err := setupNATGateway(ctx, ec2Client, subnet); err != nil {
			return nil, err
		}
	}

	return subnet, nil
}

// setupNATGateway creates or retrieves NAT gateway
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

// setupSecurityGroup creates or retrieves security group with DCV port 8443
func setupSecurityGroup(ctx context.Context, ec2Client *aws.EC2Client, vpcID, connectionMethod string) (*aws.SecurityGroupInfo, error) {
	out := output.DefaultFormatter()
	out.Step("🔒", "Configuring firewall rules (DCV port 8443)")

	sgStrategy := aws.DefaultSecurityGroupStrategy(vpcID)
	sgStrategy.DefaultName = "lens-qgis-dcv"

	// Note: DCV port 8443 will be accessed via SSM port forwarding
	// so we don't need to open it in the security group

	securityGroup, err := ec2Client.GetOrCreateSecurityGroup(ctx, sgStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to setup security group: %w", err)
	}

	out.SuccessWithDetail("Security configured", securityGroup.Name)
	return securityGroup, nil
}

// prepareInstanceImage selects AMI and generates user data
func prepareInstanceImage(ctx context.Context, ec2Client *aws.EC2Client, env *qgisconfig.QGISEnvironment, region string, idleTimeoutSeconds int) (string, string, error) {
	out := output.DefaultFormatter()
	out.Step("🔍", "Selecting Ubuntu 22.04 LTS base image")

	// QGIS requires Ubuntu 22.04 (jammy) for the QGIS repository
	amiSelector := aws.NewAMISelector(region)
	amiID, err := amiSelector.GetAMI(ctx, ec2Client, "ubuntu-jammy-x86_64")
	if err != nil {
		return "", "", fmt.Errorf("failed to find Ubuntu 22.04 AMI: %w", err)
	}
	out.Success("Ubuntu 22.04 LTS image selected")

	// Generate cloud-init user data with QGIS + DCV setup
	userData, err := qgisconfig.GenerateUserData(env, idleTimeoutSeconds)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate user data: %w", err)
	}

	return amiID, userData, nil
}

// launchAndWaitForInstance launches the EC2 instance and waits for it to be running
func launchAndWaitForInstance(ctx context.Context, ec2Client *aws.EC2Client, ssmClient *aws.SSMClient, env *qgisconfig.QGISEnvironment, subnet *aws.SubnetInfo, securityGroup *aws.SecurityGroupInfo, amiID, userData string, keyInfo *aws.KeyPairInfo, instanceProfile *aws.InstanceProfileInfo) (*types.Instance, error) {
	out := output.DefaultFormatter()
	out.Step("🚀", fmt.Sprintf("Starting QGIS %s environment", env.Name))

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

	if keyInfo != nil {
		launchParams.KeyPairName = keyInfo.Name
	}

	instance, err := ec2Client.LaunchInstance(ctx, launchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to launch instance: %w", err)
	}

	instanceID := *instance.InstanceId
	out.SuccessWithDetail("QGIS environment starting", instanceID)

	out.Status("Waiting for instance to boot")
	if err := ec2Client.WaitForInstanceRunning(ctx, instanceID); err != nil {
		return nil, fmt.Errorf("instance failed to start: %w", err)
	}

	// Get updated instance info
	instance, err = ec2Client.GetInstanceInfo(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance info: %w", err)
	}

	// Poll for DCV readiness
	out.Blank()
	out.Status("Installing QGIS + NICE DCV Desktop")
	out.Info(output.EstimatedTimeMessage(420)) // 7 minutes for QGIS + DCV + desktop
	out.Blank()

	if err := waitForDCVReady(ctx, ssmClient, instance); err != nil {
		out.Blank()
		out.Warning(fmt.Sprintf("%v", err))
		out.Info("You can still try connecting - the service may still be starting up")
	}

	return instance, nil
}

// waitForDCVReady polls DCV server until it's ready on port 8443
func waitForDCVReady(ctx context.Context, ssmClient *aws.SSMClient, instance *types.Instance) error {
	instanceID := *instance.InstanceId

	// DCV runs on port 8443
	config := readiness.SSMServiceConfig{
		InstanceID: instanceID,
		Port:       8443,
		Timeout:    10 * time.Minute, // DCV + Desktop + QGIS takes longer
		Retry:      15 * time.Second,  // Check every 15 seconds
	}

	startTime := time.Now()
	lastUpdate := startTime

	callback := func(message string, elapsed time.Duration) {
		now := time.Now()
		if now.Sub(lastUpdate) >= 30*time.Second || strings.Contains(message, "ready") || strings.Contains(message, "SSM") {
			fmt.Printf("   [%ds] %s\n", int(elapsed.Seconds()), message)
			lastUpdate = now
		}
	}

	result, err := readiness.PollServiceReadinessViaSSM(ctx, config, ssmClient, callback)
	if err != nil {
		return err
	}

	if result.Ready {
		fmt.Printf("✓ NICE DCV Desktop is ready! (took %v)\n", result.ElapsedTime.Round(time.Second))
		return nil
	}

	return fmt.Errorf("%s", result.Message)
}

// displayInstanceInfo shows the launched instance information
func displayInstanceInfo(instance *types.Instance, env *qgisconfig.QGISEnvironment, subnet *aws.SubnetInfo, keyInfo *aws.KeyPairInfo, connectionMethod, subnetType, profile string) error {
	out := output.DefaultFormatter()

	publicIP := "N/A (private subnet)"
	if instance.PublicIpAddress != nil {
		publicIP = *instance.PublicIpAddress
	}
	privateIP := *instance.PrivateIpAddress
	instanceID := *instance.InstanceId

	out.Blank()
	out.Complete("QGIS + DCV Desktop launched successfully!")
	out.Blank()

	out.KeyValue("Instance ID", instanceID)
	out.KeyValue("Instance Type", env.InstanceType)
	out.KeyValue("Public IP", publicIP)
	out.KeyValue("Private IP", privateIP)
	out.KeyValue("Subnet", fmt.Sprintf("%s (%s)", subnet.ID, subnetType))

	if env.RequiresGPU {
		out.KeyValue("GPU", "NVIDIA (enabled)")
	}

	if connectionMethod == connectionMethodSSH && keyInfo != nil {
		out.KeyValue("SSH Key", keyInfo.Name)
	}

	out.Blank()
	out.Subheader("Connection Instructions:")

	out.Info("QGIS Desktop will be available at: https://localhost:8443")
	out.Status("Please wait 5-7 minutes for complete installation")

	out.Blank()
	out.Subheader("Next Steps:")
	out.List(fmt.Sprintf("Use 'lens-qgis connect %s --profile %s' to setup DCV port forwarding", instanceID, profile))
	out.List("Open https://localhost:8443 in your browser")
	out.List("QGIS will auto-launch on the desktop")
	out.Blank()

	return nil
}
