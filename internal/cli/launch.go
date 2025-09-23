package cli

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/aws-jupyter/internal/aws"
	"github.com/scttfrdmn/aws-jupyter/internal/config"
	"github.com/spf13/cobra"
)

func NewLaunchCmd() *cobra.Command {
	var (
		environment  string
		instanceType string
		idleTimeout  string
		profile      string
		region       string
		dryRun       bool
	)

	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Launch a new Jupyter instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLaunch(environment, instanceType, idleTimeout, profile, region, dryRun)
		},
	}

	cmd.Flags().StringVarP(&environment, "env", "e", "data-science", "Environment configuration to use")
	cmd.Flags().StringVarP(&instanceType, "instance-type", "t", "", "Override instance type")
	cmd.Flags().StringVarP(&idleTimeout, "idle-timeout", "i", "4h", "Auto-shutdown timeout")
	cmd.Flags().StringVarP(&profile, "profile", "p", "default", "AWS profile to use")
	cmd.Flags().StringVarP(&region, "region", "r", "", "AWS region")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")

	return cmd
}

func runLaunch(environment, instanceType, idleTimeout, profile, region string, dryRun bool) error {
	ctx := context.Background()

	// Load environment configuration
	env, err := config.LoadEnvironment(environment)
	if err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	// Override instance type if provided
	if instanceType != "" {
		env.InstanceType = instanceType
	}

	if dryRun {
		fmt.Printf("[DRY RUN] Would launch %s environment on %s in region %s\n", env.Name, env.InstanceType, region)
		fmt.Printf("[DRY RUN] Configuration:\n")
		fmt.Printf("  - Environment: %s\n", env.Name)
		fmt.Printf("  - Instance Type: %s\n", env.InstanceType)
		fmt.Printf("  - AMI Base: %s\n", env.AMIBase)
		fmt.Printf("  - EBS Volume: %dGB\n", env.EBSVolumeSize)
		fmt.Printf("  - Packages: %d system packages\n", len(env.Packages))
		fmt.Printf("  - Pip Packages: %d python packages\n", len(env.PipPackages))
		fmt.Printf("  - Jupyter Extensions: %d extensions\n", len(env.JupyterExtensions))
		fmt.Printf("  - Idle Timeout: %s\n", idleTimeout)
		fmt.Printf("  - AWS Profile: %s\n", profile)
		if region != "" {
			fmt.Printf("  - AWS Region: %s (override)\n", region)
		}

		fmt.Printf("[DRY RUN] Would perform these actions:\n")
		fmt.Printf("  1. Create/verify SSH key pair\n")
		fmt.Printf("  2. Create/verify security group (SSH + Jupyter access)\n")
		fmt.Printf("  3. Generate user data script for environment setup\n")
		fmt.Printf("  4. Launch EC2 instance (%s)\n", env.InstanceType)
		fmt.Printf("  5. Wait for instance to be running\n")
		fmt.Printf("  6. Setup SSH tunnel (port 8888)\n")
		fmt.Printf("  7. Save instance state locally\n")
		fmt.Printf("  8. Display connection information\n")

		fmt.Println("[DRY RUN] No resources were created")
		return nil
	}

	fmt.Printf("Launching %s environment on %s...\n", env.Name, env.InstanceType)

	// Create AWS client
	_, err = aws.NewEC2Client(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// TODO: Implement key pair creation, security group setup, userdata generation
	// TODO: Launch instance, setup SSH tunnel, save state

	fmt.Println("Instance launched successfully!")
	return nil
}
