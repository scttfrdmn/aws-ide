package cli

import (
	"fmt"
	"sort"

	"github.com/scttfrdmn/lens/apps/qgis/internal/config"
	"github.com/scttfrdmn/lens/pkg/output"
	"github.com/spf13/cobra"
)

func NewEnvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "env",
		Short: "List available QGIS environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnvList()
		},
	}
}

func runEnvList() error {
	out := output.DefaultFormatter()

	out.Header("Available QGIS Environments")
	out.Blank()

	envs := config.GetDefaultEnvironments()

	// Sort environment names for consistent output
	names := make([]string, 0, len(envs))
	for name := range envs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		env := envs[name]

		out.Subheader(env.Name)
		out.KeyValue("Description", env.Description)
		out.KeyValue("Instance Type", env.InstanceType)

		// Show GPU requirement
		if env.RequiresGPU {
			out.KeyValue("GPU", "Required (NVIDIA)")
		} else {
			out.KeyValue("GPU", "Not required")
		}

		out.KeyValue("Storage", fmt.Sprintf("%d GB EBS", env.EBSVolumeSize))

		// Show package count
		out.KeyValue("Packages", fmt.Sprintf("%d system packages", len(env.Packages)))

		// Show plugins
		if len(env.QGISPlugins) > 0 {
			out.KeyValue("QGIS Plugins", fmt.Sprintf("%d plugins", len(env.QGISPlugins)))
		}

		// Show estimated cost
		cost := getEstimatedCost(env.InstanceType)
		if cost != "" {
			out.KeyValue("Est. Cost", cost)
		}

		out.Blank()
	}

	out.Info("Launch an environment with: lens-qgis launch --env <name>")
	out.Blank()

	return nil
}

// getEstimatedCost returns estimated hourly cost for common instance types
func getEstimatedCost(instanceType string) string {
	costs := map[string]string{
		"t3.xlarge":    "~$0.17/hr",
		"t3.2xlarge":   "~$0.33/hr",
		"m5.xlarge":    "~$0.19/hr",
		"m5.2xlarge":   "~$0.38/hr",
		"g4dn.xlarge":  "~$0.53/hr",
		"g4dn.2xlarge": "~$0.75/hr",
		"g5.xlarge":    "~$1.01/hr",
	}
	return costs[instanceType]
}
