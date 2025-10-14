package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awslib "github.com/scttfrdmn/aws-jupyter/internal/aws"
	"github.com/scttfrdmn/aws-jupyter/internal/config"
	"github.com/spf13/cobra"
)

// NewStatusCmd creates the status command for checking instance status
func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status INSTANCE_ID",
		Short: "Show instance status and logs",
		Long:  "Display detailed status information about an EC2 instance including state, uptime, and configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(args[0])
		},
	}
}

func runStatus(instanceID string) error {
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

	// Display status
	fmt.Printf("Instance Status: %s\n", instanceID)
	fmt.Println("=" + string(make([]byte, 60)))
	fmt.Printf("Environment:     %s\n", instance.Environment)
	fmt.Printf("Instance Type:   %s\n", instance.InstanceType)
	fmt.Printf("Region:          %s\n", instance.Region)
	fmt.Printf("State:           %s\n", awsInstance.State.Name)
	fmt.Printf("Public IP:       %s\n", aws.ToString(awsInstance.PublicIpAddress))
	fmt.Printf("Private IP:      %s\n", aws.ToString(awsInstance.PrivateIpAddress))
	fmt.Printf("Availability Zone: %s\n", aws.ToString(awsInstance.Placement.AvailabilityZone))

	// Calculate uptime
	if !instance.LaunchedAt.IsZero() {
		uptime := time.Since(instance.LaunchedAt)
		hours := int(uptime.Hours())
		minutes := int(uptime.Minutes()) % 60
		fmt.Printf("Uptime:          %dh%dm\n", hours, minutes)
	}

	fmt.Printf("Idle Timeout:    %s\n", instance.IdleTimeout)
	fmt.Printf("Key Pair:        %s\n", instance.KeyPair)
	fmt.Printf("Security Group:  %s\n", instance.SecurityGroup)

	// SSH Tunnel status
	if instance.TunnelPID > 0 {
		fmt.Printf("SSH Tunnel:      Active (PID %d)\n", instance.TunnelPID)
		fmt.Printf("Jupyter URL:     http://localhost:8888\n")
	} else {
		fmt.Printf("SSH Tunnel:      Not active\n")
	}

	// Launch time
	fmt.Printf("Launched At:     %s\n", instance.LaunchedAt.Format(time.RFC3339))

	// Instance launch time from AWS (might differ slightly)
	if awsInstance.LaunchTime != nil {
		fmt.Printf("AWS Launch Time: %s\n", awsInstance.LaunchTime.Format(time.RFC3339))
	}

	return nil
}
