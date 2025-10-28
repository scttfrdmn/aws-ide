package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	awslib "github.com/scttfrdmn/lens/pkg/aws"
	"github.com/scttfrdmn/lens/pkg/config"
	"github.com/spf13/cobra"
)

var (
	filterState string
	noColor     bool
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all QGIS instances",
		Long: `List all QGIS instances with their current status.

Shows instance ID, environment, type, state, uptime, and DCV connection status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList()
		},
	}

	cmd.Flags().StringVar(&filterState, "state", "", "Filter by instance state (running, stopped, terminated)")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "Disable color-coded output")

	return cmd
}

type instanceInfo struct {
	Instance *config.Instance
	State    string
	Uptime   time.Duration
}

func runList() error {
	ctx := context.Background()

	state, err := config.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Filter to only QGIS instances (check environment names)
	qgisInstances := make(map[string]*config.Instance)
	qgisEnvs := map[string]bool{
		"basic-gis":      true,
		"advanced-gis":   true,
		"remote-sensing": true,
	}

	for id, inst := range state.Instances {
		if qgisEnvs[inst.Environment] {
			qgisInstances[id] = inst
		}
	}

	if len(qgisInstances) == 0 {
		fmt.Println("No QGIS instances found")
		fmt.Println()
		fmt.Println("Launch a QGIS instance with: lens-qgis launch")
		return nil
	}

	// Gather instance information
	var instances []instanceInfo
	for _, instance := range qgisInstances {
		instanceState := getInstanceState(ctx, instance)
		uptime := time.Since(instance.LaunchedAt)

		// Apply state filter
		if filterState != "" && !strings.EqualFold(instanceState, filterState) {
			continue
		}

		instances = append(instances, instanceInfo{
			Instance: instance,
			State:    instanceState,
			Uptime:   uptime,
		})
	}

	if len(instances) == 0 {
		fmt.Printf("No QGIS instances found with state: %s\n", filterState)
		return nil
	}

	// Sort by uptime (oldest first)
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Uptime > instances[j].Uptime
	})

	// Output table
	return outputTable(instances)
}

func getInstanceState(ctx context.Context, instance *config.Instance) string {
	// Try to get current state from AWS
	ec2Client, err := awslib.NewEC2ClientForRegion(ctx, instance.Region)
	if err != nil {
		return "unknown"
	}

	awsInstance, err := ec2Client.GetInstanceInfo(ctx, instance.ID)
	if err != nil {
		return "unknown"
	}

	return string(awsInstance.State.Name)
}

func outputTable(instances []instanceInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tENV\tTYPE\tSTATE\tUPTIME\tDCV"); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	for _, info := range instances {
		uptime := formatDuration(info.Uptime)
		dcvStatus := ""
		if info.Instance.TunnelPID > 0 {
			dcvStatus = ":8443"
		}

		state := info.State
		if !noColor {
			state = colorizeState(state)
		}

		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			info.Instance.ID,
			info.Instance.Environment,
			info.Instance.InstanceType,
			state,
			uptime,
			dcvStatus,
		); err != nil {
			return fmt.Errorf("failed to write instance data: %w", err)
		}
	}

	return w.Flush()
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func colorizeState(state string) string {
	switch strings.ToLower(state) {
	case "running":
		return "\033[32m" + state + "\033[0m" // Green
	case "stopped":
		return "\033[33m" + state + "\033[0m" // Yellow
	case "terminated", "terminating":
		return "\033[31m" + state + "\033[0m" // Red
	case "pending", "stopping":
		return "\033[36m" + state + "\033[0m" // Cyan
	default:
		return state
	}
}
