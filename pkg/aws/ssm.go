package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// SSMClient wraps AWS Systems Manager operations
type SSMClient struct {
	client *ssm.Client
	region string
}

// NewSSMClient creates a new SSM client
func NewSSMClient(cfg aws.Config) *SSMClient {
	return &SSMClient{
		client: ssm.NewFromConfig(cfg),
		region: cfg.Region,
	}
}

// CommandResult contains the result of an SSM command execution
type CommandResult struct {
	CommandID    string
	Status       types.CommandInvocationStatus
	Output       string
	ErrorOutput  string
	ResponseCode int32
}

// RunCommand executes a command on an EC2 instance via SSM
// Returns the command ID immediately - use WaitForCommand to get results
func (s *SSMClient) RunCommand(ctx context.Context, instanceID string, command string) (string, error) {
	input := &ssm.SendCommandInput{
		InstanceIds:  []string{instanceID},
		DocumentName: aws.String("AWS-RunShellScript"),
		Parameters: map[string][]string{
			"commands": {command},
		},
		TimeoutSeconds: aws.Int32(30), // Command timeout (not overall timeout)
	}

	result, err := s.client.SendCommand(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to send SSM command: %w", err)
	}

	return *result.Command.CommandId, nil
}

// WaitForCommand waits for an SSM command to complete and returns the result
func (s *SSMClient) WaitForCommand(ctx context.Context, commandID string, instanceID string, timeout time.Duration) (*CommandResult, error) {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for SSM command to complete")
		}

		// Check command status
		invocation, err := s.client.GetCommandInvocation(ctx, &ssm.GetCommandInvocationInput{
			CommandId:  aws.String(commandID),
			InstanceId: aws.String(instanceID),
		})
		if err != nil {
			// Command might not be registered yet, retry
			time.Sleep(1 * time.Second)
			continue
		}

		result := &CommandResult{
			CommandID:    commandID,
			Status:       invocation.Status,
			Output:       aws.ToString(invocation.StandardOutputContent),
			ErrorOutput:  aws.ToString(invocation.StandardErrorContent),
			ResponseCode: invocation.ResponseCode,
		}

		// Check if command completed
		switch invocation.Status {
		case types.CommandInvocationStatusSuccess, types.CommandInvocationStatusFailed,
			types.CommandInvocationStatusCancelled, types.CommandInvocationStatusTimedOut:
			return result, nil
		case types.CommandInvocationStatusInProgress, types.CommandInvocationStatusPending:
			// Still running, wait and retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(1 * time.Second):
				continue
			}
		default:
			// Unknown status
			time.Sleep(1 * time.Second)
			continue
		}
	}
}

// CheckServiceReadiness checks if a service is ready on the instance by curling localhost
// Returns true if the service responds with any HTTP status code (2xx, 3xx, 4xx, etc.)
func (s *SSMClient) CheckServiceReadiness(ctx context.Context, instanceID string, port int) (bool, error) {
	// Use curl to check if service is responding locally
	// -s: silent, -o /dev/null: discard output, -w %{http_code}: print status code
	// --connect-timeout 2: fail fast if service not running
	// --max-time 5: overall timeout
	command := fmt.Sprintf("curl -s -o /dev/null -w '%%{http_code}' --connect-timeout 2 --max-time 5 http://localhost:%d/ 2>/dev/null || echo '000'", port)

	commandID, err := s.RunCommand(ctx, instanceID, command)
	if err != nil {
		return false, fmt.Errorf("failed to run readiness check command: %w", err)
	}

	result, err := s.WaitForCommand(ctx, commandID, instanceID, 15*time.Second)
	if err != nil {
		return false, fmt.Errorf("failed to wait for readiness check: %w", err)
	}

	// Check if we got an HTTP response code (not 000 which means connection failed)
	httpCode := result.Output
	if len(httpCode) >= 3 {
		httpCode = httpCode[0:3] // Get first 3 chars
		if httpCode != "000" {
			// Any HTTP response means service is up (200, 302, 401, etc.)
			return true, nil
		}
	}

	return false, nil
}

// WaitForSSMAgent waits for the SSM agent to become available on an instance
// This should be called before attempting to run commands via SSM
func (s *SSMClient) WaitForSSMAgent(ctx context.Context, instanceID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for SSM agent to become available")
		}

		// Try to get instance information - if SSM agent is running, this will succeed
		input := &ssm.DescribeInstanceInformationInput{
			Filters: []types.InstanceInformationStringFilter{
				{
					Key:    aws.String("InstanceIds"),
					Values: []string{instanceID},
				},
			},
		}

		result, err := s.client.DescribeInstanceInformation(ctx, input)
		if err == nil && len(result.InstanceInformationList) > 0 {
			instance := result.InstanceInformationList[0]
			if instance.PingStatus == types.PingStatusOnline {
				return nil
			}
		}

		// SSM agent not ready yet, wait and retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			continue
		}
	}
}
