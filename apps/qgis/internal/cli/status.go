package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/scttfrdmn/lens/pkg/aws"
	"github.com/scttfrdmn/lens/pkg/config"
	"github.com/scttfrdmn/lens/pkg/output"
	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status INSTANCE_ID",
		Short: "Show detailed status of a QGIS instance",
		Long: `Show detailed status of a QGIS instance.

Displays instance state, configuration, uptime, costs, and connection info.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(args[0])
		},
	}

	return cmd
}

func runStatus(instanceID string) error {
	ctx := context.Background()
	out := output.DefaultFormatter()

	// Load state to get instance details
	state, err := config.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Get instance from state
	instance, exists := state.Instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found in local state", instanceID)
	}

	// Create AWS client for the instance's region
	ec2Client, err := aws.NewEC2ClientForRegion(ctx, instance.Region)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Get current instance info from AWS
	awsInstance, err := ec2Client.GetInstanceInfo(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance info: %w", err)
	}

	// Display status
	out.Header(fmt.Sprintf("QGIS Instance Status: %s", instanceID))
	out.Blank()

	// Instance state
	instanceState := string(awsInstance.State.Name)
	stateDisplay := colorizeState(instanceState)
	out.KeyValue("State", stateDisplay)
	out.Blank()

	// Configuration
	out.Subheader("Configuration")
	out.KeyValue("Environment", instance.Environment)
	out.KeyValue("Instance Type", instance.InstanceType)
	out.KeyValue("Region", instance.Region)
	if awsInstance.Placement != nil && awsInstance.Placement.AvailabilityZone != nil {
		out.KeyValue("Availability Zone", *awsInstance.Placement.AvailabilityZone)
	}
	out.Blank()

	// Network
	out.Subheader("Network")
	if awsInstance.PublicIpAddress != nil {
		out.KeyValue("Public IP", *awsInstance.PublicIpAddress)
	} else {
		out.KeyValue("Public IP", "N/A (private subnet)")
	}
	out.KeyValue("Private IP", *awsInstance.PrivateIpAddress)
	out.Blank()

	// Timing
	out.Subheader("Timing")
	out.KeyValue("Launched", instance.LaunchedAt.Format("2006-01-02 15:04:05"))
	uptime := time.Since(instance.LaunchedAt)
	out.KeyValue("Uptime", formatDuration(uptime))
	out.Blank()

	// Connection
	out.Subheader("Connection")
	if instance.TunnelPID > 0 {
		out.KeyValue("DCV Port Forwarding", fmt.Sprintf("Active (PID %d)", instance.TunnelPID))
		out.KeyValue("QGIS Desktop URL", "https://localhost:8443")
	} else {
		out.KeyValue("DCV Port Forwarding", "Not connected")
		out.Info(fmt.Sprintf("Connect with: lens-qgis connect %s", instanceID))
	}
	out.Blank()

	// Estimated costs
	if instanceState == "running" {
		out.Subheader("Estimated Costs")
		hourlyRate := getHourlyRate(instance.InstanceType)
		if hourlyRate > 0 {
			uptimeHours := uptime.Hours()
			estimatedCost := hourlyRate * uptimeHours
			out.KeyValue("Hourly Rate", fmt.Sprintf("$%.3f/hr", hourlyRate))
			out.KeyValue("Session Cost", fmt.Sprintf("$%.2f (%.1f hours)", estimatedCost, uptimeHours))
		}
		out.Blank()
	}

	return nil
}

func getHourlyRate(instanceType string) float64 {
	rates := map[string]float64{
		"t3.xlarge":    0.1664,
		"t3.2xlarge":   0.3328,
		"m5.xlarge":    0.192,
		"m5.2xlarge":   0.384,
		"g4dn.xlarge":  0.526,
		"g4dn.2xlarge": 0.752,
		"g5.xlarge":    1.006,
	}
	return rates[instanceType]
}
