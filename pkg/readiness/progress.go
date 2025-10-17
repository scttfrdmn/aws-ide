package readiness

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// ProgressStep represents a single step in the installation process
type ProgressStep struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Completed bool
}

// ProgressReader reads setup progress from cloud-init logs
type ProgressReader struct {
	ssmClient  *ssm.Client
	instanceID string
	logPath    string
}

// NewProgressReader creates a new progress reader for an instance
func NewProgressReader(ssmClient *ssm.Client, instanceID string) *ProgressReader {
	return &ProgressReader{
		ssmClient:  ssmClient,
		instanceID: instanceID,
		logPath:    "/var/log/setup-progress.log",
	}
}

// StreamProgress streams setup progress updates via SSM
// The callback is called for each progress line received
func (pr *ProgressReader) StreamProgress(ctx context.Context, callback func(line string)) error {
	// Start SSM session to tail the progress log
	sessionInput := &ssm.StartSessionInput{
		Target:       aws.String(pr.instanceID),
		DocumentName: aws.String("AWS-StartInteractiveCommand"),
		Parameters: map[string][]string{
			"command": {fmt.Sprintf("tail -f %s 2>/dev/null || echo 'Log file not found yet...'", pr.logPath)},
		},
	}

	session, err := pr.ssmClient.StartSession(ctx, sessionInput)
	if err != nil {
		return fmt.Errorf("failed to start SSM session: %w", err)
	}

	// Note: This is a simplified version. In practice, you'd need to use
	// the session-manager-plugin to properly stream the output.
	// For now, we'll return the session ID and let the caller handle it.
	_ = session

	return fmt.Errorf("SSM streaming not yet implemented - use session ID: %s", aws.ToString(session.SessionId))
}

// GetProgress retrieves the current progress by reading the log file via SSM
func (pr *ProgressReader) GetProgress(ctx context.Context) ([]ProgressStep, error) {
	// Execute command to read the progress log
	commandInput := &ssm.SendCommandInput{
		InstanceIds:  []string{pr.instanceID},
		DocumentName: aws.String("AWS-RunShellScript"),
		Parameters: map[string][]string{
			"commands": {fmt.Sprintf("cat %s 2>/dev/null || echo ''", pr.logPath)},
		},
	}

	command, err := pr.ssmClient.SendCommand(ctx, commandInput)
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Wait a moment for command to execute
	time.Sleep(2 * time.Second)

	// Get command output
	outputInput := &ssm.GetCommandInvocationInput{
		CommandId:  command.Command.CommandId,
		InstanceId: aws.String(pr.instanceID),
	}

	output, err := pr.ssmClient.GetCommandInvocation(ctx, outputInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get command output: %w", err)
	}

	// Parse the output into steps
	return parseProgressLog(aws.ToString(output.StandardOutputContent))
}

// parseProgressLog parses the progress log content into structured steps
func parseProgressLog(content string) ([]ProgressStep, error) {
	if content == "" {
		return []ProgressStep{}, nil
	}

	var steps []ProgressStep
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "STEP:") {
			stepName := strings.TrimPrefix(line, "STEP:")
			steps = append(steps, ProgressStep{
				Name:      stepName,
				StartTime: time.Now(),
				Completed: false,
			})
		} else if strings.HasPrefix(line, "COMPLETE") {
			// Mark all steps as completed
			for i := range steps {
				steps[i].Completed = true
				steps[i].EndTime = time.Now()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return steps, nil
}

// WaitForCompletion waits for the setup to complete by polling the log
func (pr *ProgressReader) WaitForCompletion(ctx context.Context, callback func(steps []ProgressStep)) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			steps, err := pr.GetProgress(ctx)
			if err != nil {
				// Log file might not exist yet, continue waiting
				continue
			}

			if callback != nil {
				callback(steps)
			}

			// Check if all steps are complete
			if len(steps) > 0 {
				allComplete := true
				for _, step := range steps {
					if !step.Completed {
						allComplete = false
						break
					}
				}
				if allComplete {
					return nil
				}
			}
		}
	}
}

// SimpleProgressReader reads progress from an io.Reader (for local testing or SSH)
type SimpleProgressReader struct {
	reader io.Reader
}

// NewSimpleProgressReader creates a progress reader from an io.Reader
func NewSimpleProgressReader(reader io.Reader) *SimpleProgressReader {
	return &SimpleProgressReader{reader: reader}
}

// Stream reads lines from the reader and calls the callback for each line
func (spr *SimpleProgressReader) Stream(ctx context.Context, callback func(line string)) error {
	scanner := bufio.NewScanner(spr.reader)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line := scanner.Text()
			if callback != nil {
				callback(line)
			}

			// Check for completion
			if strings.Contains(line, "COMPLETE") {
				return nil
			}
		}
	}

	return scanner.Err()
}

// Helper function to check if SSM is available for an instance
func IsSSMAvailable(ctx context.Context, ssmClient *ssm.Client, instanceID string) bool {
	input := &ssm.DescribeInstanceInformationInput{
		Filters: []ssmtypes.InstanceInformationStringFilter{
			{
				Key:    aws.String("InstanceIds"),
				Values: []string{instanceID},
			},
		},
	}

	output, err := ssmClient.DescribeInstanceInformation(ctx, input)
	if err != nil {
		return false
	}

	return len(output.InstanceInformationList) > 0
}
