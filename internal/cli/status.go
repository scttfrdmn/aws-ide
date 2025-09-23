package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status INSTANCE_ID",
		Short: "Show instance status and logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(args[0])
		},
	}
}

func runStatus(instanceID string) error {
	// TODO: Implement status logic
	fmt.Printf("Status for instance %s:\n", instanceID)
	return nil
}
