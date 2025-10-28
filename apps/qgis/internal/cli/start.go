package cli

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/lens/pkg/aws"
	"github.com/scttfrdmn/lens/pkg/config"
	"github.com/spf13/cobra"
)

func NewStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start INSTANCE_ID",
		Short: "Start a stopped QGIS instance",
		Long: `Start a stopped QGIS instance.

After starting, use 'lens-qgis connect' to access the QGIS desktop.`,
		Args: cobra.ExactArgs(1),
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

	// Start the instance
	fmt.Printf("Starting QGIS instance %s...\n", instanceID)

	if err := ec2Client.StartInstance(ctx, instanceID); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	fmt.Printf("✓ Instance %s started successfully\n", instanceID)
	fmt.Printf("\nWait a few minutes for QGIS + DCV to be ready, then:\n")
	fmt.Printf("  lens-qgis connect %s\n", instanceID)

	return nil
}
