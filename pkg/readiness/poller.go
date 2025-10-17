package readiness

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// SSMChecker is an interface for checking service readiness via SSM
type SSMChecker interface {
	WaitForSSMAgent(ctx context.Context, instanceID string, timeout time.Duration) error
	CheckServiceReadiness(ctx context.Context, instanceID string, port int) (bool, error)
}

// ServiceConfig defines the service to check for readiness
type ServiceConfig struct {
	Host    string        // Instance public IP or hostname
	Port    int           // Service port (8888 for Jupyter, 8080 for VSCode, 8787 for RStudio)
	Timeout time.Duration // Overall timeout for readiness check
	Retry   time.Duration // Time between retry attempts
}

// CheckResult contains the result of a readiness check
type CheckResult struct {
	Ready       bool
	Message     string
	ElapsedTime time.Duration
}

// ProgressCallback is called with status updates during polling
type ProgressCallback func(message string, elapsed time.Duration)

// PollServiceReadiness polls a service until it's ready or timeout is reached
func PollServiceReadiness(ctx context.Context, config ServiceConfig, callback ProgressCallback) (*CheckResult, error) {
	startTime := time.Now()
	deadline := startTime.Add(config.Timeout)

	if callback != nil {
		callback(fmt.Sprintf("Waiting for service on %s:%d...", config.Host, config.Port), 0)
	}

	ticker := time.NewTicker(config.Retry)
	defer ticker.Stop()

	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return &CheckResult{
				Ready:       false,
				Message:     "Context cancelled",
				ElapsedTime: time.Since(startTime),
			}, ctx.Err()

		case <-ticker.C:
			attempt++
			elapsed := time.Since(startTime)

			if time.Now().After(deadline) {
				if callback != nil {
					callback(fmt.Sprintf("Timeout after %d attempts", attempt), elapsed)
				}
				return &CheckResult{
					Ready:       false,
					Message:     fmt.Sprintf("Service not ready after %v (timeout)", elapsed.Round(time.Second)),
					ElapsedTime: elapsed,
				}, fmt.Errorf("timeout waiting for service")
			}

			// Try to connect to the service
			ready, err := checkHTTPService(config.Host, config.Port, 5*time.Second)
			if err == nil && ready {
				if callback != nil {
					callback(fmt.Sprintf("Service is ready! (%d attempts)", attempt), elapsed)
				}
				return &CheckResult{
					Ready:       true,
					Message:     fmt.Sprintf("Service ready after %v", elapsed.Round(time.Second)),
					ElapsedTime: elapsed,
				}, nil
			}

			// Not ready yet, update progress
			if callback != nil {
				callback(fmt.Sprintf("Attempt %d: Service not ready yet, retrying...", attempt), elapsed)
			}
		}
	}
}

// checkHTTPService attempts to connect to an HTTP service and verify it's responding
func checkHTTPService(host string, port int, timeout time.Duration) (bool, error) {
	// First check if port is open (faster than HTTP request)
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false, err
	}
	defer func() { _ = conn.Close() }()

	// Port is open, now try HTTP request
	client := &http.Client{
		Timeout: timeout,
	}

	url := fmt.Sprintf("http://%s:%d/", host, port)
	resp, err := client.Get(url)
	if err != nil {
		// Port is open but HTTP not responding yet
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Read a bit of the response to ensure service is actually working
	// (not just accepting connections)
	_, err = io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return false, err
	}

	// Any HTTP response means the service is up
	// (could be 200, 302 redirect to login, 401, etc.)
	return true, nil
}

// QuickCheck performs a single readiness check without retrying
func QuickCheck(host string, port int) bool {
	ready, _ := checkHTTPService(host, port, 3*time.Second)
	return ready
}

// SSMServiceConfig defines the service to check via SSM
type SSMServiceConfig struct {
	InstanceID string        // EC2 instance ID
	Port       int           // Service port (8888 for Jupyter, 8080 for VSCode, 8787 for RStudio)
	Timeout    time.Duration // Overall timeout for readiness check
	Retry      time.Duration // Time between retry attempts
}

// PollServiceReadinessViaSSM polls a service via SSM until it's ready or timeout is reached
// This method works regardless of security group configuration since it checks from inside the instance
func PollServiceReadinessViaSSM(ctx context.Context, config SSMServiceConfig, ssmClient SSMChecker, callback ProgressCallback) (*CheckResult, error) {
	startTime := time.Now()
	deadline := startTime.Add(config.Timeout)

	// First, wait for SSM agent to be ready
	if callback != nil {
		callback("Waiting for SSM agent to be ready...", 0)
	}

	ssmTimeout := 2 * time.Minute // SSM agent usually ready within 1-2 minutes
	if err := ssmClient.WaitForSSMAgent(ctx, config.InstanceID, ssmTimeout); err != nil {
		return &CheckResult{
			Ready:       false,
			Message:     fmt.Sprintf("SSM agent not ready: %v", err),
			ElapsedTime: time.Since(startTime),
		}, fmt.Errorf("SSM agent not available: %w", err)
	}

	ssmReadyTime := time.Since(startTime)
	if callback != nil {
		callback(fmt.Sprintf("SSM agent ready! Now checking service on port %d...", config.Port), ssmReadyTime)
	}

	// Now poll for service readiness
	ticker := time.NewTicker(config.Retry)
	defer ticker.Stop()

	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return &CheckResult{
				Ready:       false,
				Message:     "Context cancelled",
				ElapsedTime: time.Since(startTime),
			}, ctx.Err()

		case <-ticker.C:
			attempt++
			elapsed := time.Since(startTime)

			if time.Now().After(deadline) {
				if callback != nil {
					callback(fmt.Sprintf("Timeout after %d attempts", attempt), elapsed)
				}
				return &CheckResult{
					Ready:       false,
					Message:     fmt.Sprintf("Service not ready after %v (timeout)", elapsed.Round(time.Second)),
					ElapsedTime: elapsed,
				}, fmt.Errorf("timeout waiting for service")
			}

			// Try to check service via SSM
			ready, err := ssmClient.CheckServiceReadiness(ctx, config.InstanceID, config.Port)
			if err != nil {
				// SSM command failed - this could be transient, log and continue
				if callback != nil {
					callback(fmt.Sprintf("Attempt %d: SSM check failed (%v), retrying...", attempt, err), elapsed)
				}
				continue
			}

			if ready {
				if callback != nil {
					callback(fmt.Sprintf("Service is ready! (%d attempts)", attempt), elapsed)
				}
				return &CheckResult{
					Ready:       true,
					Message:     fmt.Sprintf("Service ready after %v", elapsed.Round(time.Second)),
					ElapsedTime: elapsed,
				}, nil
			}

			// Not ready yet, update progress
			if callback != nil {
				callback(fmt.Sprintf("Attempt %d: Service not ready yet, retrying...", attempt), elapsed)
			}
		}
	}
}
