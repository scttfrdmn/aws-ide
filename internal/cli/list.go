package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

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

		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			instance.ID,
			instance.Environment,
			instance.InstanceType,
			"running", // TODO: get actual state from AWS
			uptime,
			tunnel,
		); err != nil {
			return fmt.Errorf("failed to write instance data: %w", err)
		}
	}

	return w.Flush()
}

func formatDuration(start time.Time) string {
	duration := time.Since(start)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}
