package cli

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/aws-jupyter/internal/aws"
	"github.com/scttfrdmn/aws-jupyter/internal/config"
	"github.com/spf13/cobra"
)

// NewDeleteAMICmd creates the delete-ami command for deleting custom AMIs
func NewDeleteAMICmd() *cobra.Command {
	var all bool
	var region string

	cmd := &cobra.Command{
		Use:   "delete-ami [AMI_ID]",
		Short: "Delete a custom AMI",
		Long: `Delete one or more custom AMIs created by aws-jupyter CLI.

When deleting an AMI, both the AMI and its associated snapshots are removed.
This operation cannot be undone.

Examples:
  # Delete a specific AMI
  aws-jupyter delete-ami ami-1234567890abcdef0

  # Delete all custom AMIs in current region
  aws-jupyter delete-ami --all

  # Delete all custom AMIs in specific region
  aws-jupyter delete-ami --all --region us-west-2`,
		Args: func(cmd *cobra.Command, args []string) error {
			if all && len(args) > 0 {
				return fmt.Errorf("cannot specify AMI_ID when using --all flag")
			}
			if !all && len(args) != 1 {
				return fmt.Errorf("requires AMI_ID argument or --all flag")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				return runDeleteAllAMIs(region)
			}
			return runDeleteAMI(args[0], region)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Delete all custom aws-jupyter AMIs")
	cmd.Flags().StringVarP(&region, "region", "r", "", "AWS region (defaults to current region)")

	return cmd
}

func runDeleteAMI(amiID, region string) error {
	ctx := context.Background()

	// Create EC2 client
	var ec2Client *aws.EC2Client
	var err error
	if region != "" {
		ec2Client, err = aws.NewEC2ClientForRegion(ctx, region)
	} else {
		ec2Client, err = aws.NewEC2Client(ctx, "default")
	}
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	fmt.Printf("Deleting AMI %s in region %s...\n", amiID, ec2Client.GetRegion())

	// Delete the AMI
	if err := ec2Client.DeleteAMI(ctx, amiID); err != nil {
		return fmt.Errorf("failed to delete AMI: %w", err)
	}

	fmt.Printf("✓ AMI %s deleted successfully\n", amiID)
	return nil
}

func runDeleteAllAMIs(region string) error {
	ctx := context.Background()

	// Create EC2 client
	var ec2Client *aws.EC2Client
	var err error
	if region != "" {
		ec2Client, err = aws.NewEC2ClientForRegion(ctx, region)
	} else {
		ec2Client, err = aws.NewEC2Client(ctx, "default")
	}
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// List all custom AMIs
	amis, err := ec2Client.ListCustomAMIs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list AMIs: %w", err)
	}

	if len(amis) == 0 {
		fmt.Printf("No custom aws-jupyter AMIs found in region %s\n", ec2Client.GetRegion())
		return nil
	}

	// Confirm deletion
	fmt.Printf("Found %d custom aws-jupyter AMI(s) in region %s:\n", len(amis), ec2Client.GetRegion())
	for _, ami := range amis {
		fmt.Printf("  - %s (%s) - %s\n", ami.ID, ami.Name, ami.CreationDate.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("\n⚠️  This will DELETE ALL listed AMIs and their snapshots. This cannot be undone.\n")
	fmt.Printf("Type 'yes' to confirm: ")

	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "yes" {
		fmt.Println("Deletion cancelled")
		return nil
	}

	// Delete each AMI
	fmt.Printf("\nDeleting %d AMI(s)...\n", len(amis))
	successCount := 0
	failCount := 0

	for _, ami := range amis {
		fmt.Printf("Deleting %s (%s)...\n", ami.ID, ami.Name)
		if err := ec2Client.DeleteAMI(ctx, ami.ID); err != nil {
			fmt.Printf("  ✗ Failed: %v\n", err)
			failCount++
		} else {
			fmt.Printf("  ✓ Deleted\n")
			successCount++
		}
	}

	fmt.Printf("\n✓ Deleted %d AMI(s)", successCount)
	if failCount > 0 {
		fmt.Printf(" (%d failed)", failCount)
	}
	fmt.Println()

	// Clean up state file - remove instances that reference deleted AMIs
	if successCount > 0 {
		cleanupStateFile(amis)
	}

	return nil
}

func cleanupStateFile(deletedAMIs []aws.AMIInfo) {
	state, err := config.LoadState()
	if err != nil {
		return
	}

	// Build set of deleted AMI IDs
	deletedSet := make(map[string]bool)
	for _, ami := range deletedAMIs {
		deletedSet[ami.ID] = true
	}

	// Remove instances that were using deleted AMIs
	modified := false
	for id, instance := range state.Instances {
		// Note: state doesn't track AMI ID currently, only AMIBase
		// This is a placeholder for future enhancement
		_ = id
		_ = instance
	}

	if modified {
		state.Save()
	}
}
