//go:build e2e
// +build e2e

package e2etest

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/scttfrdmn/aws-ide/pkg/aws"
	"github.com/scttfrdmn/aws-ide/pkg/config"
	"github.com/scttfrdmn/aws-ide/pkg/readiness"
)

// TestContext holds common resources for E2E tests
type TestContext struct {
	T            *testing.T
	Profile      string
	Region       string
	EC2Client    *aws.EC2Client
	IAMClient    *aws.IAMClient
	SSMClient    *aws.SSMClient
	State        *config.LocalState
	InstanceID   string
	SubnetID     string
	KeyPairName  string
	KeyPairPath  string
	CleanupFuncs []func()
}

// LaunchConfig holds configuration for launching an instance
type LaunchConfig struct {
	AppName       string // "jupyter", "rstudio", "vscode"
	Environment   string // "minimal", "data-science", etc.
	InstanceType  string
	Port          int
	EBSVolumeSize int
	IdleTimeout   string
	UseSessionMgr bool // Use Session Manager instead of SSH
}

// NewTestContext creates a new test context with AWS clients
func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "aws"
	}

	ctx := context.Background()

	// Create clients
	ec2Client, err := aws.NewEC2Client(ctx, profile)
	if err != nil {
		t.Fatalf("Failed to create EC2 client: %v", err)
	}

	iamClient, err := aws.NewIAMClient(ctx, profile)
	if err != nil {
		t.Fatalf("Failed to create IAM client: %v", err)
	}

	ssmClient, err := aws.NewSSMClient(ctx, profile)
	if err != nil {
		t.Fatalf("Failed to create SSM client: %v", err)
	}

	// Load state
	state, err := config.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	return &TestContext{
		T:            t,
		Profile:      profile,
		Region:       ec2Client.GetRegion(),
		EC2Client:    ec2Client,
		IAMClient:    iamClient,
		SSMClient:    ssmClient,
		State:        state,
		CleanupFuncs: make([]func(), 0),
	}
}

// AddCleanup registers a cleanup function to be called at test end
func (tc *TestContext) AddCleanup(f func()) {
	tc.CleanupFuncs = append(tc.CleanupFuncs, f)
}

// Cleanup runs all registered cleanup functions
func (tc *TestContext) Cleanup() {
	tc.T.Helper()

	for i := len(tc.CleanupFuncs) - 1; i >= 0; i-- {
		func() {
			defer func() {
				if r := recover(); r != nil {
					tc.T.Logf("Warning: cleanup function panicked: %v", r)
				}
			}()
			tc.CleanupFuncs[i]()
		}()
	}
}

// LaunchInstance launches an EC2 instance for E2E testing
func (tc *TestContext) LaunchInstance(ctx context.Context, cfg LaunchConfig) (string, error) {
	tc.T.Helper()
	tc.T.Logf("Launching %s instance with config: %+v", cfg.AppName, cfg)

	// Get IAM instance profile
	profileInfo, err := tc.IAMClient.GetOrCreateSessionManagerRole(ctx, fmt.Sprintf("aws-%s-e2e", cfg.AppName))
	if err != nil {
		return "", fmt.Errorf("failed to get IAM profile: %w", err)
	}
	tc.T.Logf("Using IAM profile: %s", profileInfo.Name)

	// Find compatible availability zone and subnet
	tc.T.Logf("Finding compatible AZ for instance type %s", cfg.InstanceType)
	az, err := tc.EC2Client.FindCompatibleAvailabilityZone(ctx, cfg.InstanceType, "public")
	if err != nil {
		return "", fmt.Errorf("failed to find compatible AZ: %w", err)
	}
	tc.T.Logf("Using availability zone: %s", az)

	// Get subnet in the AZ
	subnet, err := tc.EC2Client.GetSubnet(ctx, "public", az)
	if err != nil {
		return "", fmt.Errorf("failed to get subnet: %w", err)
	}
	tc.SubnetID = subnet.ID
	tc.T.Logf("Using subnet: %s", subnet.ID)

	// Get AMI
	amiSelector := aws.NewAMISelector(tc.Region)
	amiBase := "ubuntu24-arm64" // Use ARM64 for cost efficiency
	if cfg.InstanceType[:2] != "t4" && cfg.InstanceType[:2] != "m7" {
		amiBase = "ubuntu24-x86_64" // Fallback for non-Graviton instance types
	}
	ami, err := amiSelector.GetAMI(ctx, tc.EC2Client, amiBase)
	if err != nil {
		return "", fmt.Errorf("failed to get AMI: %w", err)
	}
	tc.T.Logf("Using AMI: %s (%s)", ami, amiBase)

	// Create or get key pair
	tc.KeyPairName = fmt.Sprintf("aws-%s-e2e-%s", cfg.AppName, tc.Region)
	keyPairPath, err := tc.EC2Client.GetOrCreateKeyPair(ctx, tc.KeyPairName)
	if err != nil {
		return "", fmt.Errorf("failed to get key pair: %w", err)
	}
	tc.KeyPairPath = keyPairPath
	tc.T.Logf("Using key pair: %s", tc.KeyPairName)

	// Get or create security group
	sgID, err := tc.EC2Client.GetOrCreateSecurityGroup(ctx, fmt.Sprintf("aws-%s-e2e", cfg.AppName), cfg.Port)
	if err != nil {
		return "", fmt.Errorf("failed to get security group: %w", err)
	}
	tc.T.Logf("Using security group: %s", sgID)

	// Generate user data based on app type
	userData := tc.GenerateUserData(cfg)

	// Launch instance
	params := aws.LaunchParams{
		AMI:             ami,
		InstanceType:    cfg.InstanceType,
		KeyPairName:     tc.KeyPairName,
		SecurityGroupID: sgID,
		SubnetID:        subnet.ID,
		UserData:        userData,
		EBSVolumeSize:   cfg.EBSVolumeSize,
		Environment:     cfg.Environment,
		InstanceProfile: profileInfo.Name,
	}

	tc.T.Log("Launching EC2 instance...")
	instance, err := tc.EC2Client.LaunchInstance(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to launch instance: %w", err)
	}

	instanceID := *instance.InstanceId
	tc.InstanceID = instanceID
	tc.T.Logf("Instance launched: %s", instanceID)

	// Save to state
	tc.State.Instances[instanceID] = &config.Instance{
		ID:           instanceID,
		Environment:  cfg.Environment,
		InstanceType: cfg.InstanceType,
		KeyPair:      tc.KeyPairName,
		LaunchedAt:   time.Now(),
		IdleTimeout:  cfg.IdleTimeout,
		Region:       tc.Region,
	}
	tc.State.KeyPairs[tc.KeyPairName] = keyPairPath
	if err := tc.State.Save(); err != nil {
		tc.T.Logf("Warning: failed to save state: %v", err)
	}

	// Register cleanup
	tc.AddCleanup(func() {
		tc.T.Logf("Cleanup: terminating instance %s", instanceID)
		cleanupCtx := context.Background()
		if err := tc.EC2Client.TerminateInstance(cleanupCtx, instanceID); err != nil {
			tc.T.Logf("Warning: failed to terminate instance %s: %v", instanceID, err)
		}

		// Remove from state
		delete(tc.State.Instances, instanceID)
		if err := tc.State.Save(); err != nil {
			tc.T.Logf("Warning: failed to save state: %v", err)
		}
	})

	return instanceID, nil
}

// WaitForReady waits for the service to become ready using SSM polling
func (tc *TestContext) WaitForReady(ctx context.Context, instanceID string, port int, timeout time.Duration) error {
	tc.T.Helper()
	tc.T.Logf("Waiting for service on port %d to become ready (timeout: %v)", port, timeout)

	serviceConfig := readiness.ServiceConfig{
		Name: fmt.Sprintf("test-service-%d", port),
		Port: port,
	}

	pollConfig := readiness.PollingConfig{
		Interval:    10 * time.Second,
		Timeout:     timeout,
		MaxAttempts: 0, // Use timeout-based polling
	}

	// Create a channel to receive poll results
	resultCh := make(chan error, 1)

	// Start polling in goroutine
	go func() {
		err := readiness.PollServiceReadinessViaSSM(ctx, tc.SSMClient, instanceID, serviceConfig, pollConfig)
		resultCh <- err
	}()

	// Wait for result or timeout
	select {
	case err := <-resultCh:
		if err != nil {
			return fmt.Errorf("service readiness check failed: %w", err)
		}
		tc.T.Logf("✓ Service is ready on port %d", port)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while waiting for readiness: %w", ctx.Err())
	}
}

// VerifyHTTPResponse verifies the service responds with HTTP 200
func (tc *TestContext) VerifyHTTPResponse(ctx context.Context, url string, expectedCode int) error {
	tc.T.Helper()
	tc.T.Logf("Verifying HTTP response from %s", url)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedCode {
		return fmt.Errorf("expected status code %d, got %d", expectedCode, resp.StatusCode)
	}

	tc.T.Logf("✓ Received expected HTTP %d response", resp.StatusCode)
	return nil
}

// GenerateUserData generates appropriate user data based on app type
func (tc *TestContext) GenerateUserData(cfg LaunchConfig) string {
	tc.T.Helper()

	switch cfg.AppName {
	case "jupyter":
		return tc.generateJupyterUserData(cfg)
	case "rstudio":
		return tc.generateRStudioUserData(cfg)
	case "vscode":
		return tc.generateVSCodeUserData(cfg)
	default:
		tc.T.Fatalf("Unknown app type: %s", cfg.AppName)
		return ""
	}
}

func (tc *TestContext) generateJupyterUserData(cfg LaunchConfig) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

# Install JupyterLab
apt-get update
apt-get install -y python3-pip
pip3 install jupyterlab

# Start JupyterLab
jupyter lab --ip=0.0.0.0 --port=%d --no-browser --allow-root &

echo "JupyterLab E2E test instance ready"
`, cfg.Port)
}

func (tc *TestContext) generateRStudioUserData(cfg LaunchConfig) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

# Install R and RStudio Server
apt-get update
apt-get install -y r-base gdebi-core
wget https://download2.rstudio.org/server/jammy/amd64/rstudio-server-2023.12.1-402-amd64.deb
gdebi -n rstudio-server-2023.12.1-402-amd64.deb

# RStudio runs on port 8787 by default
echo "RStudio Server E2E test instance ready"
`)
}

func (tc *TestContext) generateVSCodeUserData(cfg LaunchConfig) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

# Install code-server
curl -fsSL https://code-server.dev/install.sh | sh

# Start code-server
code-server --bind-addr 0.0.0.0:%d --auth none &

echo "VSCode Server E2E test instance ready"
`, cfg.Port)
}
