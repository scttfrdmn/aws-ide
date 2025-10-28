package cli

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/lens/pkg/aws"
	"github.com/scttfrdmn/lens/pkg/config"
	"github.com/spf13/cobra"
)

func NewTerminateCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "terminate INSTANCE_ID",
		Short: "Terminate a QGIS instance",
		Long: `Terminate a QGIS instance permanently.

⚠️  WARNING: This action is irreversible! The instance and all its data
will be permanently deleted.

Use 'lens-qgis stop' if you want to stop the instance temporarily.`,
		Args: cobra.ExactArgs(1),
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

	// Confirm termination unless --force is used
	if !force {
		fmt.Printf("⚠️  WARNING: You are about to terminate instance %s\n", instanceID)
		fmt.Printf("   Environment: %s\n", instance.Environment)
		fmt.Printf("   This action is IRREVERSIBLE!\n\n")
		fmt.Printf("Type 'yes' to confirm: ")

		var confirmation string
		if _, err := fmt.Scanln(&confirmation); err != nil || confirmation != "yes" {
			fmt.Println("Termination cancelled")
			return nil
		}
	}

	// Create AWS client for the instance's region
	ec2Client, err := aws.NewEC2ClientForRegion(ctx, instance.Region)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Terminate the instance
	fmt.Printf("Terminating QGIS instance %s...\n", instanceID)

	if err := ec2Client.TerminateInstance(ctx, instanceID); err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	// Kill DCV port forwarding if it's running
	if instance.TunnelPID > 0 {
		if err := killProcess(instance.TunnelPID); err != nil {
			fmt.Printf("Warning: Failed to kill port forwarding (PID %d): %v\n", instance.TunnelPID, err)
		}
	}

	// Remove from local state
	delete(state.Instances, instanceID)
	if err := state.Save(); err != nil {
		fmt.Printf("Warning: Failed to update local state: %v\n", err)
	}

	fmt.Printf("✓ Instance %s terminated successfully\n", instanceID)
	return nil
}
