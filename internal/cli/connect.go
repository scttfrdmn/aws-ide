package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	awslib "github.com/scttfrdmn/aws-jupyter/internal/aws"
	"github.com/scttfrdmn/aws-jupyter/internal/config"
	"github.com/spf13/cobra"
)

// NewConnectCmd creates the connect command for setting up SSH tunnels to Jupyter instances
func NewConnectCmd() *cobra.Command {
	var localPort int

	cmd := &cobra.Command{
		Use:   "connect INSTANCE_ID",
		Short: "Connect to an existing instance",
		Long:  "Setup SSH tunnel or Session Manager port forwarding to access Jupyter Lab",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnect(args[0], localPort)
		},
	}

	cmd.Flags().IntVarP(&localPort, "port", "p", 8888, "Local port for Jupyter Lab")
	return cmd
}

func runConnect(instanceID string, localPort int) error {
	ctx := context.Background()

	// Load state to get instance details
	state, err := config.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
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
				fmt.Printf("SSH tunnel already running (PID %d)\n", instance.TunnelPID)
				fmt.Printf("Jupyter Lab: http://localhost:%d\n", localPort)
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
		// Use Session Manager port forwarding
		return setupSSMPortForwarding(instanceID, localPort, instance, state)
	}

	// Setup SSH tunnel
	return setupSSHTunnel(instanceID, localPort, publicIP, instance, state)
}

// setupSSHTunnel sets up an SSH tunnel to the instance
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
	remotePort := 8888
	cmd := exec.Command("ssh",
		"-i", keyPath,
		"-N", // No remote command
		"-L", fmt.Sprintf("%d:localhost:%d", localPort, remotePort), // Local port forwarding
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ServerAliveInterval=60",
		"-o", "ServerAliveCountMax=3",
		fmt.Sprintf("ec2-user@%s", publicIP),
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
	fmt.Printf("Jupyter Lab: http://localhost:%d\n", localPort)
	fmt.Println("\nNote: Keep this terminal open or the tunnel will close")
	fmt.Println("To stop the tunnel: aws-jupyter stop " + instanceID)

	return nil
}

// setupSSMPortForwarding sets up Session Manager port forwarding
func setupSSMPortForwarding(instanceID string, localPort int, instance *config.Instance, state *config.LocalState) error {
	fmt.Printf("Setting up Session Manager port forwarding to %s...\n", instanceID)

	// Check if AWS CLI and Session Manager plugin are installed
	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("AWS CLI not found. Please install it first: https://aws.amazon.com/cli/")
	}

	remotePort := 8888
	cmd := exec.Command("aws", "ssm", "start-session",
		"--target", instanceID,
		"--document-name", "AWS-StartPortForwardingSession",
		"--parameters", fmt.Sprintf(`{"portNumber":["%d"],"localPortNumber":["%d"]}`, remotePort, localPort),
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Session Manager port forwarding: %w\nMake sure the Session Manager plugin is installed: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html", err)
	}

	// Save tunnel PID
	instance.TunnelPID = cmd.Process.Pid
	if err := state.Save(); err != nil {
		fmt.Printf("Warning: Failed to save tunnel PID: %v\n", err)
	}

	fmt.Printf("✓ Session Manager port forwarding established (PID %d)\n", instance.TunnelPID)
	fmt.Printf("Jupyter Lab: http://localhost:%d\n", localPort)
	fmt.Println("\nNote: Keep this terminal open or the port forwarding will close")
	fmt.Println("To stop the tunnel: aws-jupyter stop " + instanceID)

	return nil
}
