package cli

import (
	"context"
	"fmt"
	"sort"

	"github.com/scttfrdmn/aws-ide/pkg/aws"
	"github.com/spf13/cobra"
)

// NewListAMIsCmd creates the list-amis command for listing custom AMIs
func NewListAMIsCmd() *cobra.Command {
	var region string

	cmd := &cobra.Command{
		Use:   "list-amis",
		Short: "List custom aws-jupyter AMIs",
		Long: `List all AMIs created by aws-jupyter CLI.

Shows AMIs with their IDs, names, creation dates, and states.
Only shows AMIs created in the specified region (defaults to current AWS region).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListAMIs(region)
		},
	}

	cmd.Flags().StringVarP(&region, "region", "r", "", "AWS region (defaults to current region)")

	return cmd
}

func runListAMIs(region string) error {
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

	// List AMIs
	amis, err := ec2Client.ListCustomAMIs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list AMIs: %w", err)
	}

	if len(amis) == 0 {
		fmt.Printf("No custom aws-jupyter AMIs found in region %s\n", ec2Client.GetRegion())
		fmt.Printf("\nCreate an AMI with: aws-jupyter create-ami INSTANCE_ID\n")
		return nil
	}

	// Sort by creation date (newest first)
	sort.Slice(amis, func(i, j int) bool {
		return amis[i].CreationDate.After(amis[j].CreationDate)
	})

	fmt.Printf("Custom aws-jupyter AMIs in region %s:\n\n", ec2Client.GetRegion())
	for _, ami := range amis {
		fmt.Printf("AMI ID: %s\n", ami.ID)
		fmt.Printf("  Name: %s\n", ami.Name)
		fmt.Printf("  State: %s\n", ami.State)
		fmt.Printf("  Created: %s\n", ami.CreationDate.Format("2006-01-02 15:04:05"))
		if ami.Description != "" {
			fmt.Printf("  Description: %s\n", ami.Description)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("To launch an instance using a custom AMI:\n")
	fmt.Printf("  aws-jupyter launch --env ENVIRONMENT --ami AMI_ID\n")

	return nil
}
