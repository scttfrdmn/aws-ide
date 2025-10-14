package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewTerminateCmd creates the terminate command for permanently terminating instances
func NewTerminateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "terminate INSTANCE_ID",
		Short: "Terminate an instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTerminate(args[0])
		},
	}
}

func runTerminate(instanceID string) error {
	// TODO: Implement terminate logic
	fmt.Printf("Terminating instance %s...\n", instanceID)
	return nil
}
