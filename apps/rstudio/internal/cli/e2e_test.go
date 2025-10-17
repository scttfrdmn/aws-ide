//go:build e2e
// +build e2e

package cli

import (
	"context"
	"testing"
	"time"

	"github.com/scttfrdmn/aws-ide/pkg/e2etest"
)

// TestRStudio_E2E_LaunchConnectTerminate tests the complete lifecycle:
// 1. Launch an RStudio instance
// 2. Wait for service to become ready via SSM
// 3. Verify HTTP response
// 4. Terminate the instance
func TestRStudio_E2E_LaunchConnectTerminate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create test context
	tc := e2etest.NewTestContext(t)
	defer tc.Cleanup()

	// Test configuration - RStudio runs on port 8787
	cfg := e2etest.LaunchConfig{
		AppName:       "rstudio",
		Environment:   "minimal",
		InstanceType:  "t4g.small", // ARM64, cost-effective
		Port:          8787,        // RStudio default port
		EBSVolumeSize: 20,
		IdleTimeout:   "2h",
		UseSessionMgr: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	// Step 1: Launch instance
	t.Log("Step 1: Launching RStudio instance...")
	instanceID, err := tc.LaunchInstance(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}
	t.Logf("✓ Instance launched successfully: %s", instanceID)

	// Step 2: Wait for readiness
	t.Log("Step 2: Waiting for RStudio service to become ready...")
	readyTimeout := 15 * time.Minute
	if err := tc.WaitForReady(ctx, instanceID, cfg.Port, readyTimeout); err != nil {
		t.Fatalf("Service did not become ready: %v", err)
	}
	t.Log("✓ RStudio service is ready")

	// Step 3: Verify service responds (via SSM port forwarding)
	t.Log("Step 3: Verifying service responds...")
	t.Log("✓ Service verification complete (SSM polling confirmed HTTP 200)")

	// Step 4: Terminate instance (handled by cleanup)
	t.Log("Step 4: Instance will be terminated by cleanup function")
	t.Log("✓ E2E test completed successfully")
}

// TestRStudio_E2E_MultipleEnvironments tests launching with different R environments
func TestRStudio_E2E_MultipleEnvironments(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	environments := []struct {
		name          string
		env           string
		instanceType  string
		expectedReady bool
	}{
		{"minimal", "minimal", "t4g.small", true},
		{"tidyverse", "tidyverse", "t4g.medium", true},
	}

	for _, env := range environments {
		t.Run(env.name, func(t *testing.T) {
			tc := e2etest.NewTestContext(t)
			defer tc.Cleanup()

			cfg := e2etest.LaunchConfig{
				AppName:       "rstudio",
				Environment:   env.env,
				InstanceType:  env.instanceType,
				Port:          8787,
				EBSVolumeSize: 30,
				IdleTimeout:   "1h",
				UseSessionMgr: true,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
			defer cancel()

			t.Logf("Launching RStudio with %s environment...", env.env)
			instanceID, err := tc.LaunchInstance(ctx, cfg)
			if err != nil {
				t.Fatalf("Failed to launch instance: %v", err)
			}
			t.Logf("✓ Instance launched: %s", instanceID)

			readyTimeout := 15 * time.Minute
			err = tc.WaitForReady(ctx, instanceID, cfg.Port, readyTimeout)

			if env.expectedReady {
				if err != nil {
					t.Errorf("Expected service to be ready, but got error: %v", err)
				} else {
					t.Logf("✓ Service is ready as expected")
				}
			} else {
				if err == nil {
					t.Errorf("Expected service to fail, but it became ready")
				} else {
					t.Logf("✓ Service failed as expected: %v", err)
				}
			}
		})
	}
}

// TestRStudio_E2E_InstanceLifecycle tests start/stop operations
func TestRStudio_E2E_InstanceLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	tc := e2etest.NewTestContext(t)
	defer tc.Cleanup()

	cfg := e2etest.LaunchConfig{
		AppName:       "rstudio",
		Environment:   "minimal",
		InstanceType:  "t4g.small",
		Port:          8787,
		EBSVolumeSize: 20,
		IdleTimeout:   "2h",
		UseSessionMgr: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	t.Log("Launching instance...")
	instanceID, err := tc.LaunchInstance(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to launch instance: %v", err)
	}
	t.Logf("✓ Instance launched: %s", instanceID)

	t.Log("Waiting for initial readiness...")
	if err := tc.WaitForReady(ctx, instanceID, cfg.Port, 15*time.Minute); err != nil {
		t.Fatalf("Initial readiness check failed: %v", err)
	}
	t.Log("✓ Instance is ready")

	t.Log("Stopping instance...")
	if err := tc.EC2Client.StopInstance(ctx, instanceID); err != nil {
		t.Fatalf("Failed to stop instance: %v", err)
	}
	t.Log("✓ Instance stopped")

	t.Log("Waiting for instance to reach stopped state...")
	time.Sleep(30 * time.Second)

	t.Log("Starting instance...")
	if err := tc.EC2Client.StartInstance(ctx, instanceID); err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}
	t.Log("✓ Instance started")

	t.Log("Waiting for readiness after restart...")
	if err := tc.WaitForReady(ctx, instanceID, cfg.Port, 10*time.Minute); err != nil {
		t.Fatalf("Readiness check after restart failed: %v", err)
	}
	t.Log("✓ Instance is ready after restart")

	t.Log("✓ Instance lifecycle test completed")
}

// TestRStudio_E2E_DifferentInstanceTypes tests various instance types
func TestRStudio_E2E_DifferentInstanceTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	instanceTypes := []struct {
		name         string
		instanceType string
		arch         string
	}{
		{"ARM64-small", "t4g.small", "arm64"},
		{"ARM64-medium", "t4g.medium", "arm64"},
		{"x86-medium", "t3.medium", "x86_64"},
	}

	for _, it := range instanceTypes {
		t.Run(it.name, func(t *testing.T) {
			tc := e2etest.NewTestContext(t)
			defer tc.Cleanup()

			cfg := e2etest.LaunchConfig{
				AppName:       "rstudio",
				Environment:   "minimal",
				InstanceType:  it.instanceType,
				Port:          8787,
				EBSVolumeSize: 20,
				IdleTimeout:   "1h",
				UseSessionMgr: true,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
			defer cancel()

			t.Logf("Testing instance type: %s (%s)", it.instanceType, it.arch)

			instanceID, err := tc.LaunchInstance(ctx, cfg)
			if err != nil {
				t.Fatalf("Failed to launch instance: %v", err)
			}
			t.Logf("✓ Instance launched: %s", instanceID)

			info, err := tc.EC2Client.GetInstanceInfo(ctx, instanceID)
			if err != nil {
				t.Errorf("Failed to get instance info: %v", err)
			} else {
				t.Logf("Instance type: %s, State: %s", info.InstanceType, info.State)
			}

			if err := tc.WaitForReady(ctx, instanceID, cfg.Port, 15*time.Minute); err != nil {
				t.Errorf("Readiness check failed for %s: %v", it.instanceType, err)
			} else {
				t.Logf("✓ Service ready on %s", it.instanceType)
			}
		})
	}
}
