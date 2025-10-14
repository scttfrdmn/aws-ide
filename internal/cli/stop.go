package cli

import (
	"fmt"

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
	// TODO: Implement stop logic
	action := "Stopping"
	if hibernate {
		action = "Hibernating"
	}
	fmt.Printf("%s instance %s...\n", action, instanceID)
	return nil
}
