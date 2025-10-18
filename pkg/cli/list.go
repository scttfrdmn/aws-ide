package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	awslib "github.com/scttfrdmn/aws-ide/pkg/aws"
	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/spf13/cobra"
)

var (
	filterState       string
	filterEnvironment string
	filterIDE         string
	olderThan         string
	newerThan         string
	sortBy            string
	outputFormat      string
	noColor           bool
)

// NewListCmd creates the list command for viewing active instances
func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List running instances with filtering and sorting options",
		Long: `List all tracked instances with optional filtering and sorting.

Filter Options:
  --state         Filter by instance state (running, stopped, terminated)
  --env           Filter by environment name
  --ide           Filter by IDE type (jupyter, rstudio, vscode)
  --older-than    Show instances older than duration (e.g., 2h, 1d)
  --newer-than    Show instances newer than duration (e.g., 30m, 1h)

Sort Options:
  --sort-by       Sort by: uptime, type, env, state (default: uptime)

Output Options:
  --format        Output format: table, json, csv (default: table)
  --no-color      Disable color-coded output`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList()
		},
	}

	cmd.Flags().StringVar(&filterState, "state", "", "Filter by instance state (running, stopped, terminated)")
	cmd.Flags().StringVar(&filterEnvironment, "env", "", "Filter by environment name")
	cmd.Flags().StringVar(&filterIDE, "ide", "", "Filter by IDE type (jupyter, rstudio, vscode)")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "Show instances older than duration (e.g., 2h, 1d)")
	cmd.Flags().StringVar(&newerThan, "newer-than", "", "Show instances newer than duration (e.g., 30m, 1h)")
	cmd.Flags().StringVar(&sortBy, "sort-by", "uptime", "Sort by: uptime, type, env, state")
	cmd.Flags().StringVar(&outputFormat, "format", "table", "Output format: table, json, csv")
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

	if len(state.Instances) == 0 {
		fmt.Println("No instances found")
		return nil
	}

	// Gather instance information
	var instances []instanceInfo
	for _, instance := range state.Instances {
		instanceState := getInstanceState(ctx, instance)
		uptime := time.Since(instance.LaunchedAt)

		instances = append(instances, instanceInfo{
			Instance: instance,
			State:    instanceState,
			Uptime:   uptime,
		})
	}

	// Apply filters
	instances = applyFilters(instances)

	if len(instances) == 0 {
		fmt.Println("No instances match the specified filters")
		return nil
	}

	// Sort instances
	sortInstances(instances)

	// Output in requested format
	switch outputFormat {
	case "json":
		return outputJSON(instances)
	case "csv":
		return outputCSV(instances)
	default:
		return outputTable(instances)
	}
}

func applyFilters(instances []instanceInfo) []instanceInfo {
	var filtered []instanceInfo

	for _, info := range instances {
		// Filter by state
		if filterState != "" && !strings.EqualFold(info.State, filterState) {
			continue
		}

		// Filter by environment
		if filterEnvironment != "" && !strings.EqualFold(info.Instance.Environment, filterEnvironment) {
			continue
		}

		// Filter by IDE type (check AMIBase field)
		if filterIDE != "" {
			ideType := strings.ToLower(info.Instance.AMIBase)
			if !strings.Contains(ideType, strings.ToLower(filterIDE)) {
				continue
			}
		}

		// Filter by age
		if olderThan != "" {
			duration, err := parseDuration(olderThan)
			if err == nil && info.Uptime < duration {
				continue
			}
		}

		if newerThan != "" {
			duration, err := parseDuration(newerThan)
			if err == nil && info.Uptime > duration {
				continue
			}
		}

		filtered = append(filtered, info)
	}

	return filtered
}

func sortInstances(instances []instanceInfo) {
	sort.Slice(instances, func(i, j int) bool {
		switch strings.ToLower(sortBy) {
		case "type":
			return instances[i].Instance.InstanceType < instances[j].Instance.InstanceType
		case "env":
			return instances[i].Instance.Environment < instances[j].Instance.Environment
		case "state":
			return instances[i].State < instances[j].State
		case "uptime":
			fallthrough
		default:
			return instances[i].Uptime > instances[j].Uptime // Oldest first
		}
	})
}

func outputTable(instances []instanceInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tENV\tTYPE\tSTATE\tUPTIME\tTUNNEL"); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	for _, info := range instances {
		uptime := formatDuration(info.Instance.LaunchedAt)
		tunnel := ""
		if info.Instance.TunnelPID > 0 {
			tunnel = ":8888"
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
			tunnel,
		); err != nil {
			return fmt.Errorf("failed to write instance data: %w", err)
		}
	}

	return w.Flush()
}

func outputJSON(instances []instanceInfo) error {
	output := make([]map[string]interface{}, 0, len(instances))

	for _, info := range instances {
		output = append(output, map[string]interface{}{
			"id":            info.Instance.ID,
			"environment":   info.Instance.Environment,
			"instance_type": info.Instance.InstanceType,
			"state":         info.State,
			"uptime":        formatDuration(info.Instance.LaunchedAt),
			"uptime_hours":  info.Uptime.Hours(),
			"tunnel_pid":    info.Instance.TunnelPID,
			"region":        info.Instance.Region,
			"public_ip":     info.Instance.PublicIP,
			"launched_at":   info.Instance.LaunchedAt,
			"idle_timeout":  info.Instance.IdleTimeout,
		})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputCSV(instances []instanceInfo) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Write header
	if err := w.Write([]string{"ID", "Environment", "InstanceType", "State", "Uptime", "Region", "PublicIP", "TunnelPID"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data
	for _, info := range instances {
		record := []string{
			info.Instance.ID,
			info.Instance.Environment,
			info.Instance.InstanceType,
			info.State,
			formatDuration(info.Instance.LaunchedAt),
			info.Instance.Region,
			info.Instance.PublicIP,
			fmt.Sprintf("%d", info.Instance.TunnelPID),
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}

func colorizeState(state string) string {
	switch strings.ToLower(state) {
	case "running":
		return "\033[32m" + state + "\033[0m" // Green
	case "stopped":
		return "\033[31m" + state + "\033[0m" // Red
	case "stopping":
		return "\033[33m" + state + "\033[0m" // Yellow
	case "pending":
		return "\033[36m" + state + "\033[0m" // Cyan
	case "terminated", "terminating":
		return "\033[90m" + state + "\033[0m" // Gray
	default:
		return state
	}
}

func parseDuration(s string) (time.Duration, error) {
	// Support simple duration formats like "2h", "30m", "1d"
	if strings.HasSuffix(s, "d") {
		days := s[:len(s)-1]
		var d float64
		if _, err := fmt.Sscanf(days, "%f", &d); err != nil {
			return 0, err
		}
		return time.Duration(d * 24 * float64(time.Hour)), nil
	}
	return time.ParseDuration(s)
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
