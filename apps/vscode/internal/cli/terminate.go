package cli

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/aws-ide/pkg/aws"
	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/spf13/cobra"
)

// NewTerminateCmd creates the terminate command for permanently terminating instances
func NewTerminateCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "terminate INSTANCE_ID",
		Short: "Terminate a VSCode Server instance",
		Long:  "Permanently terminate an EC2 instance and clean up local state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTerminate(args[0], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	return cmd
}

func runTerminate(instanceID string, force bool) error {
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

	// Confirm termination unless force flag is set
	if !force {
		fmt.Printf("WARNING: This will permanently terminate instance %s (%s)\n", instanceID, instance.Environment)
		fmt.Printf("This action cannot be undone. Continue? (yes/no): ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil || (response != "yes" && response != "y") {
			fmt.Println("Termination cancelled")
			return nil
		}
	}

	// Create AWS client for the instance's region
	ec2Client, err := aws.NewEC2ClientForRegion(ctx, instance.Region)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	fmt.Printf("Terminating instance %s...\n", instanceID)

	// Terminate the instance
	if err := ec2Client.TerminateInstance(ctx, instanceID); err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	// Kill tunnel if it's running
	if instance.TunnelPID > 0 {
		if err := killProcess(instance.TunnelPID); err != nil {
			fmt.Printf("Warning: Failed to kill tunnel (PID %d): %v\n", instance.TunnelPID, err)
		} else {
			fmt.Printf("Tunnel (PID %d) stopped\n", instance.TunnelPID)
		}
	}

	// Remove instance from local state
	delete(state.Instances, instanceID)
	if err := state.Save(); err != nil {
		fmt.Printf("Warning: Failed to update local state: %v\n", err)
	}

	fmt.Printf("Instance %s terminated successfully\n", instanceID)
	fmt.Println("Note: The EC2 instance, security groups, and key pairs may take a few moments to fully terminate in AWS")

	return nil
}
