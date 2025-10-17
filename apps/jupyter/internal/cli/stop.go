package cli

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/aws-ide/pkg/aws"
	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/spf13/cobra"
)

// NewStopCmd creates the stop command for stopping running instances
func NewStopCmd() *cobra.Command {
	var hibernate bool

	cmd := &cobra.Command{
		Use:   "stop INSTANCE_ID",
		Short: "Stop an instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(args[0], hibernate)
		},
	}

	cmd.Flags().BoolVar(&hibernate, "hibernate", false, "Hibernate instead of stop")
	return cmd
}

func runStop(instanceID string, hibernate bool) error {
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

	// Stop the instance
	action := "Stopping"
	if hibernate {
		action = "Hibernating"
	}
	fmt.Printf("%s instance %s...\n", action, instanceID)

	if err := ec2Client.StopInstance(ctx, instanceID, hibernate); err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	// Kill SSH tunnel if it's running
	if instance.TunnelPID > 0 {
		if err := killProcess(instance.TunnelPID); err != nil {
			fmt.Printf("Warning: Failed to kill SSH tunnel (PID %d): %v\n", instance.TunnelPID, err)
		} else {
			fmt.Printf("SSH tunnel (PID %d) stopped\n", instance.TunnelPID)
			instance.TunnelPID = 0
			if err := state.Save(); err != nil {
				fmt.Printf("Warning: Failed to update state: %v\n", err)
			}
		}
	}

	fmt.Printf("Instance %s stopped successfully\n", instanceID)
	return nil
}
