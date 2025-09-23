package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect INSTANCE_ID",
		Short: "Connect to an existing instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConnect(args[0])
		},
	}
}

func runConnect(instanceID string) error {
	// TODO: Implement connection logic
	fmt.Printf("Connecting to instance %s...\n", instanceID)
	return nil
}
