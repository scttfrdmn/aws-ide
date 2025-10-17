package cli

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/aws-ide/pkg/aws"
	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/spf13/cobra"
)

// NewStartCmd creates the start command for starting stopped instances
func NewStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start INSTANCE_ID",
		Short: "Start a stopped instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(args[0])
		},
	}

	return cmd
}

func runStart(instanceID string) error {
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
	ec2Client, err := aws.NewEC2ClientForRegion(ctx, instance.Region)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Check current state
	instanceInfo, err := ec2Client.GetInstanceInfo(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance info: %w", err)
	}

	stateName := string(instanceInfo.State.Name)
	if stateName == "running" {
		fmt.Printf("Instance %s is already running\n", instanceID)
		if instanceInfo.PublicIpAddress != nil {
			fmt.Printf("Public IP: %s\n", *instanceInfo.PublicIpAddress)
		}
		return nil
	}

	if stateName != "stopped" {
		return fmt.Errorf("instance is in state '%s', can only start instances in 'stopped' state", stateName)
	}

	// Start the instance
	fmt.Printf("Starting instance %s...\n", instanceID)

	if err := ec2Client.StartInstance(ctx, instanceID); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	// Wait for instance to be running
	fmt.Printf("⏳ Waiting for instance to be running...\n")
	if err := ec2Client.WaitForInstanceRunning(ctx, instanceID); err != nil {
		return fmt.Errorf("failed waiting for instance to start: %w", err)
	}

	// Get updated instance info (IP may have changed)
	instanceInfo, err = ec2Client.GetInstanceInfo(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get updated instance info: %w", err)
	}

	// Update state with new public IP
	if instanceInfo.PublicIpAddress != nil {
		instance.PublicIP = *instanceInfo.PublicIpAddress
		if err := state.Save(); err != nil {
			fmt.Printf("Warning: Failed to update state: %v\n", err)
		}
	}

	fmt.Printf("\n✓ Instance %s started successfully!\n", instanceID)
	if instanceInfo.PublicIpAddress != nil {
		fmt.Printf("Public IP: %s\n", *instanceInfo.PublicIpAddress)
	}
	fmt.Printf("\n📓 To connect:\n")
	fmt.Printf("aws-jupyter connect %s\n", instanceID)

	return nil
}
