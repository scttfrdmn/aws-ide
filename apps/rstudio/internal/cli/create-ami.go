package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/scttfrdmn/aws-ide/pkg/aws"
	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/spf13/cobra"
)

// NewCreateAMICmd creates the create-ami command for creating AMIs from instances
func NewCreateAMICmd() *cobra.Command {
	var name string
	var noReboot bool

	cmd := &cobra.Command{
		Use:   "create-ami INSTANCE_ID",
		Short: "Create an AMI from an instance",
		Long: `Create an Amazon Machine Image (AMI) from an existing instance.

The AMI will be created in the same region as the instance and can be used
to quickly launch new instances with the same configuration and installed packages.

By default, the instance will be rebooted to ensure filesystem consistency.
Use --no-reboot to create the AMI without rebooting (not recommended for production).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateAMI(args[0], name, noReboot)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "AMI name (default: aws-jupyter-<environment>-<timestamp>)")
	cmd.Flags().BoolVar(&noReboot, "no-reboot", false, "Create AMI without rebooting the instance")

	return cmd
}

func runCreateAMI(instanceID string, name string, noReboot bool) error {
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

	// Generate AMI name if not provided
	if name == "" {
		timestamp := time.Now().Format("20060102-150405")
		name = fmt.Sprintf("aws-jupyter-%s-%s", instance.Environment, timestamp)
	}

	// Create AMI
	fmt.Printf("Creating AMI '%s' from instance %s...\n", name, instanceID)
	if noReboot {
		fmt.Printf("⚠️  Warning: Creating AMI without reboot. Filesystem consistency not guaranteed.\n")
	}

	amiID, err := ec2Client.CreateAMI(ctx, instanceID, name, fmt.Sprintf("aws-jupyter %s environment", instance.Environment), noReboot)
	if err != nil {
		return fmt.Errorf("failed to create AMI: %w", err)
	}

	fmt.Printf("\n✓ AMI creation initiated!\n")
	fmt.Printf("AMI ID: %s\n", amiID)
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Region: %s\n", instance.Region)
	fmt.Printf("\n⏳ The AMI will be available once the creation process completes (usually 5-10 minutes).\n")
	fmt.Printf("You can check the status in the AWS Console or use:\n")
	fmt.Printf("  aws ec2 describe-images --region %s --image-ids %s\n", instance.Region, amiID)

	return nil
}
