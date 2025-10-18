package cost

import (
	"fmt"
	"time"
)

// InstancePricing contains hourly pricing for different instance types
// Prices are for on-demand instances in us-east-1 (as of 2025)
var InstancePricing = map[string]float64{
	// Graviton (ARM) instances - t4g
	"t4g.nano":    0.0042,
	"t4g.micro":   0.0084,
	"t4g.small":   0.0168,
	"t4g.medium":  0.0336,
	"t4g.large":   0.0672,
	"t4g.xlarge":  0.1344,
	"t4g.2xlarge": 0.2688,

	// Graviton compute-optimized - c7g
	"c7g.medium":   0.0363,
	"c7g.large":    0.0725,
	"c7g.xlarge":   0.1450,
	"c7g.2xlarge":  0.2900,
	"c7g.4xlarge":  0.5800,
	"c7g.8xlarge":  1.1600,
	"c7g.12xlarge": 1.7400,
	"c7g.16xlarge": 2.3200,

	// Graviton memory-optimized - m7g
	"m7g.medium":   0.0408,
	"m7g.large":    0.0816,
	"m7g.xlarge":   0.1632,
	"m7g.2xlarge":  0.3264,
	"m7g.4xlarge":  0.6528,
	"m7g.8xlarge":  1.3056,
	"m7g.12xlarge": 1.9584,
	"m7g.16xlarge": 2.6112,

	// Graviton memory-optimized - r7g
	"r7g.medium":   0.0504,
	"r7g.large":    0.1008,
	"r7g.xlarge":   0.2016,
	"r7g.2xlarge":  0.4032,
	"r7g.4xlarge":  0.8064,
	"r7g.8xlarge":  1.6128,
	"r7g.12xlarge": 2.4192,
	"r7g.16xlarge": 3.2256,

	// x86 instances for comparison - t3
	"t3.nano":    0.0052,
	"t3.micro":   0.0104,
	"t3.small":   0.0208,
	"t3.medium":  0.0416,
	"t3.large":   0.0832,
	"t3.xlarge":  0.1664,
	"t3.2xlarge": 0.3328,

	// x86 instances for comparison - m6i
	"m6i.large":    0.0960,
	"m6i.xlarge":   0.1920,
	"m6i.2xlarge":  0.3840,
	"m6i.4xlarge":  0.7680,
	"m6i.8xlarge":  1.5360,
	"m6i.12xlarge": 2.3040,
	"m6i.16xlarge": 3.0720,
	"m6i.24xlarge": 4.6080,
	"m6i.32xlarge": 6.1440,
}

// EBS pricing per GB-month (gp3)
const EBSPricePerGBMonth = 0.08

// StateChange represents a change in instance state
type StateChange struct {
	State     string    // "running", "stopped", "terminated"
	Timestamp time.Time
}

// CostCalculation contains cost breakdown for an instance
type CostCalculation struct {
	InstanceType    string
	LaunchedAt      time.Time
	StateChanges    []StateChange
	CurrentState    string
	EBSSize         int // GB

	// Computed costs
	TotalRunningHours float64 // Actual hours in "running" state
	TotalElapsedHours float64 // Total hours since launch
	ComputeCost       float64 // Cost for running hours
	StorageCost       float64 // EBS storage cost
	TotalCost         float64 // Total = Compute + Storage
	EffectiveCostPerHour float64 // Total / Elapsed hours

	// Comparison
	OnPremComparison string // Comparison to on-premise hardware
}

// CalculateCost computes the cost breakdown for an instance
func CalculateCost(instanceType string, launchedAt time.Time, stateChanges []StateChange, ebsSize int) *CostCalculation {
	calc := &CostCalculation{
		InstanceType: instanceType,
		LaunchedAt:   launchedAt,
		StateChanges: stateChanges,
		EBSSize:      ebsSize,
	}

	// Get hourly rate
	hourlyRate, exists := InstancePricing[instanceType]
	if !exists {
		// Default to t4g.medium if unknown
		hourlyRate = InstancePricing["t4g.medium"]
	}

	// Calculate total elapsed time
	now := time.Now()
	calc.TotalElapsedHours = now.Sub(launchedAt).Hours()

	// Calculate running hours by processing state changes
	calc.TotalRunningHours = calculateRunningHours(launchedAt, stateChanges, now)

	// Set current state
	if len(stateChanges) > 0 {
		calc.CurrentState = stateChanges[len(stateChanges)-1].State
	} else {
		calc.CurrentState = "running"
	}

	// Calculate compute cost (only charged when running)
	calc.ComputeCost = calc.TotalRunningHours * hourlyRate

	// Calculate storage cost (charged for all elapsed time)
	storageHours := calc.TotalElapsedHours
	calc.StorageCost = (float64(ebsSize) * EBSPricePerGBMonth * storageHours) / (24 * 30) // Convert month to hours

	// Total cost
	calc.TotalCost = calc.ComputeCost + calc.StorageCost

	// Effective cost per hour
	if calc.TotalElapsedHours > 0 {
		calc.EffectiveCostPerHour = calc.TotalCost / calc.TotalElapsedHours
	}

	// Generate on-prem comparison
	calc.OnPremComparison = generateOnPremComparison(calc.TotalCost, calc.TotalElapsedHours, instanceType)

	return calc
}

// calculateRunningHours processes state changes to determine actual running time
func calculateRunningHours(launchedAt time.Time, stateChanges []StateChange, now time.Time) float64 {
	if len(stateChanges) == 0 {
		// No state changes, assume running since launch
		return now.Sub(launchedAt).Hours()
	}

	var totalRunningHours float64
	currentState := "running" // Instances start in running state
	lastTransition := launchedAt

	for _, change := range stateChanges {
		if currentState == "running" {
			// Add the running time before this state change
			totalRunningHours += change.Timestamp.Sub(lastTransition).Hours()
		}

		currentState = change.State
		lastTransition = change.Timestamp
	}

	// Add time from last state change to now if currently running
	if currentState == "running" {
		totalRunningHours += now.Sub(lastTransition).Hours()
	}

	return totalRunningHours
}

// generateOnPremComparison creates a comparison message to on-premise hardware
func generateOnPremComparison(totalCost float64, elapsedHours float64, instanceType string) string {
	// Rough estimates for comparable hardware
	var onPremCost float64

	// Categorize instance types
	if isSmallInstance(instanceType) {
		onPremCost = 800 // Mini PC or entry workstation
	} else if isMediumInstance(instanceType) {
		onPremCost = 1500 // Mid-range workstation
	} else if isLargeInstance(instanceType) {
		onPremCost = 3000 // High-end workstation
	} else {
		onPremCost = 5000 // Server-class hardware
	}

	elapsedDays := elapsedHours / 24

	if elapsedDays < 1 {
		return fmt.Sprintf("On-prem equivalent: ~$%.0f hardware", onPremCost)
	}

	savings := onPremCost - totalCost
	savingsPercent := (savings / onPremCost) * 100

	if totalCost < onPremCost {
		return fmt.Sprintf("💰 Save $%.2f (%.0f%%) vs ~$%.0f on-prem hardware", savings, savingsPercent, onPremCost)
	}

	breakEvenDays := (onPremCost / totalCost) * elapsedDays
	return fmt.Sprintf("Break-even at ~%.0f days of 24/7 usage (vs $%.0f hardware)", breakEvenDays, onPremCost)
}

func isSmallInstance(instanceType string) bool {
	smallInstances := []string{"nano", "micro", "small"}
	for _, size := range smallInstances {
		if containsSize(instanceType, size) {
			return true
		}
	}
	return false
}

func isMediumInstance(instanceType string) bool {
	return containsSize(instanceType, "medium") || containsSize(instanceType, "large")
}

func isLargeInstance(instanceType string) bool {
	return containsSize(instanceType, "xlarge") && !containsSize(instanceType, "2xlarge")
}

func containsSize(instanceType, size string) bool {
	return len(instanceType) >= len(size) &&
		(instanceType[len(instanceType)-len(size):] == size ||
		 instanceType == "t4g."+size ||
		 instanceType == "t3."+size ||
		 instanceType == "m7g."+size ||
		 instanceType == "m6i."+size ||
		 instanceType == "c7g."+size ||
		 instanceType == "r7g."+size)
}

// FormatCost formats a cost value as currency
func FormatCost(cost float64) string {
	return fmt.Sprintf("$%.4f", cost)
}

// FormatCostShort formats a cost value as currency (shorter format)
func FormatCostShort(cost float64) string {
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}

// FormatHours formats hours in a human-readable way
func FormatHours(hours float64) string {
	if hours < 1 {
		minutes := int(hours * 60)
		return fmt.Sprintf("%dm", minutes)
	} else if hours < 24 {
		return fmt.Sprintf("%.1fh", hours)
	} else {
		days := int(hours / 24)
		remainingHours := int(hours) % 24
		if remainingHours == 0 {
			return fmt.Sprintf("%dd", days)
		}
		return fmt.Sprintf("%dd %dh", days, remainingHours)
	}
}

// EstimateMonthly estimates monthly cost based on current usage pattern
func (c *CostCalculation) EstimateMonthly() float64 {
	if c.TotalElapsedHours == 0 {
		return 0
	}

	// Calculate average usage pattern
	runningRatio := c.TotalRunningHours / c.TotalElapsedHours

	// Estimate for 30 days
	hoursPerMonth := 24.0 * 30.0
	estimatedRunningHours := hoursPerMonth * runningRatio

	hourlyRate, exists := InstancePricing[c.InstanceType]
	if !exists {
		hourlyRate = InstancePricing["t4g.medium"]
	}

	computeCost := estimatedRunningHours * hourlyRate
	storageCost := float64(c.EBSSize) * EBSPricePerGBMonth

	return computeCost + storageCost
}

// GetUtilizationPercentage returns the percentage of time instance was running
func (c *CostCalculation) GetUtilizationPercentage() float64 {
	if c.TotalElapsedHours == 0 {
		return 0
	}
	return (c.TotalRunningHours / c.TotalElapsedHours) * 100
}
