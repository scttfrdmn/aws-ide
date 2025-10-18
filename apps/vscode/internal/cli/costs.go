package cli

import (
	"fmt"
	"sort"

	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/scttfrdmn/aws-ide/pkg/cost"
	"github.com/spf13/cobra"
)

// NewCostsCmd creates the costs command for viewing cost information
func NewCostsCmd() *cobra.Command {
	var showDetails bool

	cmd := &cobra.Command{
		Use:   "costs [INSTANCE_ID]",
		Short: "Show cost information for instances",
		Long: `Display cost breakdown for running instances.

Shows both actual running costs and effective costs (total/elapsed time).
The effective cost demonstrates the true cost savings of cloud infrastructure
by factoring in stop/start cycles.

Without INSTANCE_ID, shows costs for all instances.
With INSTANCE_ID, shows detailed breakdown for that instance.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return runCostsDetail(args[0])
			}
			return runCostsAll(showDetails)
		},
	}

	cmd.Flags().BoolVarP(&showDetails, "details", "d", false, "Show detailed breakdown for each instance")

	return cmd
}

func runCostsAll(showDetails bool) error {
	state, err := config.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	if len(state.Instances) == 0 {
		fmt.Println("No instances found")
		return nil
	}

	// Load config to check if cost tracking is enabled
	cfg, _ := config.LoadUserConfig()
	if cfg != nil && !cfg.EnableCostTracking {
		fmt.Println("Cost tracking is disabled in config")
		fmt.Println("Enable with: aws-vscode config set enable_cost_tracking true")
		return nil
	}

	fmt.Println("Instance Costs Summary")
	fmt.Println("======================")
	fmt.Println()

	// Calculate costs for all instances
	var totalCost, totalRunningHours, totalElapsedHours float64
	var calculations []*instanceCostInfo

	for _, instance := range state.Instances {
		// Convert state changes to cost state changes
		costStateChanges := convertToCostStateChanges(instance.StateChanges)

		calc := cost.CalculateCost(
			instance.InstanceType,
			instance.LaunchedAt,
			costStateChanges,
			instance.EBSSize,
		)

		calculations = append(calculations, &instanceCostInfo{
			instance: instance,
			calc:     calc,
		})

		totalCost += calc.TotalCost
		totalRunningHours += calc.TotalRunningHours
		totalElapsedHours += calc.TotalElapsedHours
	}

	// Sort by cost (highest first)
	sort.Slice(calculations, func(i, j int) bool {
		return calculations[i].calc.TotalCost > calculations[j].calc.TotalCost
	})

	// Display each instance
	for _, info := range calculations {
		fmt.Printf("Instance: %s (%s)\n", info.instance.ID, info.instance.Environment)
		fmt.Printf("  Type: %s\n", info.instance.InstanceType)
		fmt.Printf("  Running: %s / %s (%.0f%% utilization)\n",
			cost.FormatHours(info.calc.TotalRunningHours),
			cost.FormatHours(info.calc.TotalElapsedHours),
			info.calc.GetUtilizationPercentage())
		fmt.Printf("  Total Cost: %s\n", cost.FormatCostShort(info.calc.TotalCost))
		fmt.Printf("  Effective Rate: %s/hour\n", cost.FormatCost(info.calc.EffectiveCostPerHour))

		if showDetails {
			fmt.Printf("    Compute: %s  Storage: %s\n",
				cost.FormatCostShort(info.calc.ComputeCost),
				cost.FormatCostShort(info.calc.StorageCost))
			fmt.Printf("    %s\n", info.calc.OnPremComparison)
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("Overall Summary")
	fmt.Println("---------------")
	fmt.Printf("Total Instances: %d\n", len(calculations))
	fmt.Printf("Total Cost: %s\n", cost.FormatCostShort(totalCost))
	if totalElapsedHours > 0 {
		avgUtilization := (totalRunningHours / totalElapsedHours) * 100
		effectiveRate := totalCost / totalElapsedHours
		fmt.Printf("Average Utilization: %.0f%%\n", avgUtilization)
		fmt.Printf("Effective Rate: %s/hour\n", cost.FormatCost(effectiveRate))
	}

	// Monthly estimate
	if len(calculations) > 0 {
		var monthlyEstimate float64
		for _, info := range calculations {
			monthlyEstimate += info.calc.EstimateMonthly()
		}
		fmt.Printf("\nEstimated Monthly: %s (based on current usage pattern)\n",
			cost.FormatCostShort(monthlyEstimate))

		if cfg != nil && cfg.CostAlertThreshold > 0 {
			if monthlyEstimate > cfg.CostAlertThreshold {
				fmt.Printf("⚠️  Monthly estimate exceeds alert threshold of $%.2f\n", cfg.CostAlertThreshold)
			}
		}
	}

	return nil
}

func runCostsDetail(instanceID string) error {
	state, err := config.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	instance, exists := state.Instances[instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found in local state", instanceID)
	}

	// Convert state changes
	costStateChanges := convertToCostStateChanges(instance.StateChanges)

	calc := cost.CalculateCost(
		instance.InstanceType,
		instance.LaunchedAt,
		costStateChanges,
		instance.EBSSize,
	)

	fmt.Printf("Cost Breakdown for %s\n", instanceID)
	fmt.Println("================================")
	fmt.Println()

	// Instance info
	fmt.Println("Instance Details:")
	fmt.Printf("  Environment:  %s\n", instance.Environment)
	fmt.Printf("  Type:         %s\n", instance.InstanceType)
	fmt.Printf("  Region:       %s\n", instance.Region)
	fmt.Printf("  Launched:     %s\n", instance.LaunchedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  EBS Size:     %d GB\n", instance.EBSSize)
	fmt.Println()

	// Time breakdown
	fmt.Println("Time Breakdown:")
	fmt.Printf("  Elapsed:      %s (%.1f hours)\n",
		cost.FormatHours(calc.TotalElapsedHours), calc.TotalElapsedHours)
	fmt.Printf("  Running:      %s (%.1f hours)\n",
		cost.FormatHours(calc.TotalRunningHours), calc.TotalRunningHours)
	fmt.Printf("  Utilization:  %.1f%%\n", calc.GetUtilizationPercentage())
	fmt.Println()

	// Cost breakdown
	fmt.Println("Cost Breakdown:")
	fmt.Printf("  Compute:      %s (%s/hour × %.1f hours)\n",
		cost.FormatCostShort(calc.ComputeCost),
		cost.FormatCost(cost.InstancePricing[instance.InstanceType]),
		calc.TotalRunningHours)
	fmt.Printf("  Storage:      %s (%d GB × $%.4f/GB-month)\n",
		cost.FormatCostShort(calc.StorageCost),
		instance.EBSSize,
		cost.EBSPricePerGBMonth)
	fmt.Printf("  Total:        %s\n", cost.FormatCostShort(calc.TotalCost))
	fmt.Println()

	// Key metrics
	fmt.Println("Key Metrics:")
	fmt.Printf("  Cost per Running Hour:  %s\n",
		cost.FormatCost(cost.InstancePricing[instance.InstanceType]))
	fmt.Printf("  Effective Cost per Hour: %s\n", cost.FormatCost(calc.EffectiveCostPerHour))
	savings := cost.InstancePricing[instance.InstanceType] - calc.EffectiveCostPerHour
	savingsPercent := (savings / cost.InstancePricing[instance.InstanceType]) * 100
	fmt.Printf("  Savings vs 24/7:         %s/hour (%.0f%%)\n",
		cost.FormatCost(savings), savingsPercent)
	fmt.Println()

	// Monthly estimates
	fmt.Println("Projections:")
	monthlyEstimate := calc.EstimateMonthly()
	fmt.Printf("  Est. Monthly Cost:       %s (at current usage)\n",
		cost.FormatCostShort(monthlyEstimate))

	// 24/7 comparison
	hoursPerMonth := 24.0 * 30.0
	cost247 := (cost.InstancePricing[instance.InstanceType] * hoursPerMonth) +
		(float64(instance.EBSSize) * cost.EBSPricePerGBMonth)
	fmt.Printf("  24/7 Monthly Cost:       %s\n", cost.FormatCostShort(cost247))
	monthlySavings := cost247 - monthlyEstimate
	fmt.Printf("  Monthly Savings:         %s (%.0f%%)\n",
		cost.FormatCostShort(monthlySavings),
		(monthlySavings/cost247)*100)
	fmt.Println()

	// On-prem comparison
	fmt.Println("Cloud vs On-Premise:")
	fmt.Printf("  %s\n", calc.OnPremComparison)
	fmt.Println()

	// State change history
	if len(instance.StateChanges) > 0 {
		fmt.Println("State Change History:")
		for _, change := range instance.StateChanges {
			fmt.Printf("  %s → %s\n",
				change.Timestamp.Format("2006-01-02 15:04:05"),
				change.State)
		}
		fmt.Println()
	}

	return nil
}

type instanceCostInfo struct {
	instance *config.Instance
	calc     *cost.CostCalculation
}

func convertToCostStateChanges(stateChanges []config.StateChange) []cost.StateChange {
	result := make([]cost.StateChange, len(stateChanges))
	for i, sc := range stateChanges {
		result[i] = cost.StateChange{
			State:     sc.State,
			Timestamp: sc.Timestamp,
		}
	}
	return result
}
