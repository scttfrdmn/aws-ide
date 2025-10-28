package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	awslib "github.com/scttfrdmn/lens/pkg/aws"
	"github.com/scttfrdmn/lens/pkg/config"
	"github.com/spf13/cobra"
)

// NewConnectCmd creates the connect command for setting up port forwarding to QGIS/DCV instances
func NewConnectCmd() *cobra.Command {
	var (
		localPort int
		profile   string
	)

	cmd := &cobra.Command{
		Use:   "connect [INSTANCE_ID]",
		Short: "Connect to a running QGIS instance via DCV remote desktop",
		Long: `Setup port forwarding to access QGIS via NICE DCV remote desktop.

After connecting, open https://localhost:8443 in your browser to access
the QGIS desktop environment.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceID := ""
			if len(args) > 0 {
				instanceID = args[0]
			}
			return runConnect(instanceID, localPort, profile)
		},
	}

	cmd.Flags().IntVarP(&localPort, "port", "p", 8443, "Local port for DCV remote desktop")
	cmd.Flags().StringVar(&profile, "profile", "default", "AWS profile to use")
	return cmd
}

func runConnect(instanceID string, localPort int, profile string) error {
	ctx := context.Background()

	// Load state to get instance details
	state, err := config.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// If no instance ID provided, auto-select
	if instanceID == "" {
		selectedID, err := selectInstance(state)
		if err != nil {
			return err
		}
		instanceID = selectedID
	}

	// Get instance from state
	instance, exists := state.Instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found in local state", instanceID)
	}

	// Check if tunnel is already running
	if instance.TunnelPID > 0 {
		// Verify the process is actually running
		process, err := os.FindProcess(instance.TunnelPID)
		if err == nil {
			// Process exists, check if it's still alive
			if err := process.Signal(os.Signal(nil)); err == nil {
				fmt.Printf("Port forwarding already running (PID %d)\n", instance.TunnelPID)
				fmt.Printf("QGIS Desktop: https://localhost:%d\n", localPort)
				return nil
			}
		}
		// Process doesn't exist anymore, clear the PID
		instance.TunnelPID = 0
	}

	// Create AWS client for the instance's region
	ec2Client, err := awslib.NewEC2ClientForRegion(ctx, instance.Region)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Get current instance info from AWS
	awsInstance, err := ec2Client.GetInstanceInfo(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance info: %w", err)
	}

	// Check if instance is running
	if awsInstance.State.Name != "running" {
		return fmt.Errorf("instance is in state '%s', must be 'running' to connect", awsInstance.State.Name)
	}

	publicIP := ""
	if awsInstance.PublicIpAddress != nil {
		publicIP = *awsInstance.PublicIpAddress
	}

	// Determine connection method
	useSSH := instance.KeyPair != "" && publicIP != ""
	useSSM := publicIP == "" || !useSSH

	if useSSM {
		// Use Session Manager port forwarding (preferred for DCV)
		return setupSSMPortForwarding(instanceID, localPort, instance, state, profile)
	}

	// Setup SSH tunnel
	return setupSSHTunnel(instanceID, localPort, publicIP, instance, state)
}

// setupSSHTunnel sets up an SSH tunnel to the DCV instance
func setupSSHTunnel(instanceID string, localPort int, publicIP string, instance *config.Instance, state *config.LocalState) error {
	fmt.Printf("Setting up SSH tunnel to %s...\n", instanceID)

	keyStorage, err := config.DefaultKeyStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize key storage: %w", err)
	}

	keyPath := keyStorage.GetKeyPath(instance.KeyPair)
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key not found: %s", keyPath)
	}

	// Start SSH tunnel in background
	remotePort := 8443 // DCV server port

	// QGIS uses Ubuntu, so username is always "ubuntu"
	username := "ubuntu"

	cmd := exec.Command("ssh",
		"-i", keyPath,
		"-N",                                                        // No remote command
		"-L", fmt.Sprintf("%d:localhost:%d", localPort, remotePort), // Local port forwarding
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ServerAliveInterval=60",
		"-o", "ServerAliveCountMax=3",
		fmt.Sprintf("%s@%s", username, publicIP),
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start SSH tunnel: %w", err)
	}

	// Save tunnel PID
	instance.TunnelPID = cmd.Process.Pid
	if err := state.Save(); err != nil {
		fmt.Printf("Warning: Failed to save tunnel PID: %v\n", err)
	}

	fmt.Printf("✓ SSH tunnel established (PID %d)\n", instance.TunnelPID)
	fmt.Printf("\n")
	fmt.Printf("QGIS Desktop: https://localhost:%d\n", localPort)
	fmt.Printf("\n")
	fmt.Println("📺 Open the URL above in your browser to access QGIS")
	fmt.Println("   QGIS will auto-launch on the desktop")
	fmt.Println()
	fmt.Println("⚠️  Keep this terminal open or the tunnel will close")
	fmt.Printf("   To stop: lens-qgis stop %s\n", instanceID)

	return nil
}

// setupSSMPortForwarding sets up Session Manager port forwarding for DCV
func setupSSMPortForwarding(instanceID string, localPort int, instance *config.Instance, state *config.LocalState, profile string) error {
	fmt.Printf("Setting up Session Manager port forwarding to %s...\n", instanceID)

	// Check if AWS CLI and Session Manager plugin are installed
	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("AWS CLI not found. Please install it first: https://aws.amazon.com/cli/")
	}

	remotePort := 8443 // DCV server port

	args := []string{
		"ssm", "start-session",
		"--target", instanceID,
		"--document-name", "AWS-StartPortForwardingSession",
		"--parameters", fmt.Sprintf(`{"portNumber":["%d"],"localPortNumber":["%d"]}`, remotePort, localPort),
	}

	// Add profile if not default
	if profile != "" && profile != "default" {
		args = append([]string{"--profile", profile}, args...)
	}

	cmd := exec.Command("aws", args...)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Session Manager port forwarding: %w\nMake sure the Session Manager plugin is installed: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html", err)
	}

	// Save tunnel PID
	instance.TunnelPID = cmd.Process.Pid
	if err := state.Save(); err != nil {
		fmt.Printf("Warning: Failed to save tunnel PID: %v\n", err)
	}

	fmt.Printf("✓ Session Manager port forwarding established (PID %d)\n", instance.TunnelPID)
	fmt.Printf("\n")
	fmt.Printf("QGIS Desktop: https://localhost:%d\n", localPort)
	fmt.Printf("\n")
	fmt.Println("📺 Open the URL above in your browser to access QGIS")
	fmt.Println("   QGIS will auto-launch on the desktop")
	fmt.Println()
	fmt.Println("⚠️  Keep this terminal open or the port forwarding will close")
	fmt.Printf("   To stop: lens-qgis stop %s\n", instanceID)

	return nil
}

// selectInstance automatically selects an instance when none is specified
func selectInstance(state *config.LocalState) (string, error) {
	if len(state.Instances) == 0 {
		return "", fmt.Errorf("no instances found. Launch an instance first with 'lens-qgis launch'")
	}

	if len(state.Instances) == 1 {
		// Only one instance, auto-select it
		for id := range state.Instances {
			fmt.Printf("Auto-selecting instance: %s\n", id)
			return id, nil
		}
	}

	// Multiple instances, show list and prompt
	fmt.Println("Multiple instances found:")
	fmt.Println()
	for id, inst := range state.Instances {
		fmt.Printf("  %s\n", id)
		fmt.Printf("    Environment: %s\n", inst.Environment)
		fmt.Printf("    Type: %s\n", inst.InstanceType)
		if inst.PublicIP != "" {
			fmt.Printf("    Public IP: %s\n", inst.PublicIP)
		}
		fmt.Printf("    Launched: %s\n", inst.LaunchedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
	return "", fmt.Errorf("please specify which instance to connect to: lens-qgis connect INSTANCE_ID")
}
