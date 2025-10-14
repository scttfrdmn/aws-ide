package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	awslib "github.com/scttfrdmn/aws-jupyter/internal/aws"
	"github.com/scttfrdmn/aws-jupyter/internal/config"
	"github.com/spf13/cobra"
)

// NewListCmd creates the list command for viewing active instances
func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List running instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList()
		},
	}
}

func runList() error {
	ctx := context.Background()

	state, err := config.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	if len(state.Instances) == 0 {
		fmt.Println("No instances found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tENV\tTYPE\tSTATE\tUPTIME\tTUNNEL"); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	for _, instance := range state.Instances {
		uptime := formatDuration(instance.LaunchedAt)
		tunnel := ""
		if instance.TunnelPID > 0 {
			tunnel = ":8888"
		}

		// Get actual instance state from AWS
		state := getInstanceState(ctx, instance)

		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			instance.ID,
			instance.Environment,
			instance.InstanceType,
			state,
			uptime,
			tunnel,
		); err != nil {
			return fmt.Errorf("failed to write instance data: %w", err)
		}
	}

	return w.Flush()
}

// getInstanceState retrieves the current state of an instance from AWS
func getInstanceState(ctx context.Context, instance *config.Instance) string {
	// Create AWS client for the instance's region
	ec2Client, err := awslib.NewEC2ClientForRegion(ctx, instance.Region)
	if err != nil {
		return "unknown"
	}

	// Get current instance info from AWS
	awsInstance, err := ec2Client.GetInstanceInfo(ctx, instance.ID)
	if err != nil {
		return "unknown"
	}

	return string(awsInstance.State.Name)
}

func formatDuration(start time.Time) string {
	duration := time.Since(start)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}
