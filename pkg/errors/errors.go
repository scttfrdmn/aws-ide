package errors

import (
	"fmt"
	"strings"
)

// ContextualError wraps an error with helpful context and suggestions
type ContextualError struct {
	Operation   string   // What operation was being performed
	Cause       error    // The underlying error
	Context     string   // Additional context about what was happening
	Suggestions []string // Suggested next steps to resolve the issue
}

// Error implements the error interface
func (e *ContextualError) Error() string {
	var sb strings.Builder

	// Main error message
	if e.Operation != "" {
		sb.WriteString(fmt.Sprintf("Failed to %s: %v\n", e.Operation, e.Cause))
	} else {
		sb.WriteString(fmt.Sprintf("Error: %v\n", e.Cause))
	}

	// Add context
	if e.Context != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", e.Context))
	}

	// Add suggestions
	if len(e.Suggestions) > 0 {
		sb.WriteString("\nSuggested actions:\n")
		for _, suggestion := range e.Suggestions {
			sb.WriteString(fmt.Sprintf("  • %s\n", suggestion))
		}
	}

	return sb.String()
}

// Unwrap returns the underlying error
func (e *ContextualError) Unwrap() error {
	return e.Cause
}

// New creates a new contextual error
func New(operation string, cause error) *ContextualError {
	return &ContextualError{
		Operation:   operation,
		Cause:       cause,
		Suggestions: []string{},
	}
}

// WithContext adds context to the error
func (e *ContextualError) WithContext(context string) *ContextualError {
	e.Context = context
	return e
}

// WithSuggestion adds a suggestion to the error
func (e *ContextualError) WithSuggestion(suggestion string) *ContextualError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithSuggestions adds multiple suggestions to the error
func (e *ContextualError) WithSuggestions(suggestions ...string) *ContextualError {
	e.Suggestions = append(e.Suggestions, suggestions...)
	return e
}

// Common error patterns with helpful suggestions

// AWSPermissionError creates an error for AWS permission issues
func AWSPermissionError(operation string, cause error) *ContextualError {
	return New(operation, cause).
		WithContext("This error usually indicates missing AWS permissions.").
		WithSuggestions(
			"Check your AWS IAM permissions for the required actions",
			"Verify your AWS profile with: aws sts get-caller-identity",
			"Ensure your IAM role/user has the necessary EC2, VPC, and SSM permissions",
		)
}

// AWSResourceNotFoundError creates an error for missing AWS resources
func AWSResourceNotFoundError(resource string, resourceID string, cause error) *ContextualError {
	return New(fmt.Sprintf("find %s", resource), cause).
		WithContext(fmt.Sprintf("The %s '%s' does not exist or is not accessible.", resource, resourceID)).
		WithSuggestions(
			"Check if the resource exists in your AWS account",
			fmt.Sprintf("Verify you're using the correct region"),
			"List available resources with the appropriate aws-ide command",
		)
}

// ConfigFileError creates an error for configuration file issues
func ConfigFileError(operation string, cause error) *ContextualError {
	return New(operation, cause).
		WithContext("There was an issue with your configuration file.").
		WithSuggestions(
			"Initialize config with: aws-vscode config init",
			"Check config file at: ~/.aws-ide/config.yaml",
			"Validate YAML syntax if editing manually",
		)
}

// InstanceNotFoundError creates an error when an instance isn't in local state
func InstanceNotFoundError(instanceID string) *ContextualError {
	return New("find instance", fmt.Errorf("instance %s not found in local state", instanceID)).
		WithContext("The instance may have been terminated or was not launched by this tool.").
		WithSuggestions(
			"List your instances with: aws-vscode list",
			"Check AWS Console to see if instance exists",
			"If instance was terminated, the state will be cleaned up automatically",
		)
}

// NetworkError creates an error for network/connectivity issues
func NetworkError(operation string, cause error) *ContextualError {
	return New(operation, cause).
		WithContext("Unable to establish network connection.").
		WithSuggestions(
			"Check your internet connection",
			"Verify the instance is running: aws-vscode status INSTANCE_ID",
			"Check security group rules allow necessary traffic",
			"If using Session Manager, verify SSM agent is running",
		)
}

// EnvironmentNotFoundError creates an error for missing environments
func EnvironmentNotFoundError(envName string) *ContextualError {
	return New("load environment", fmt.Errorf("environment '%s' not found", envName)).
		WithContext("The specified environment configuration does not exist.").
		WithSuggestions(
			"List available environments: aws-vscode env list",
			"Generate an environment from your project: aws-vscode generate",
			"Check if environment name is correct",
		)
}

// AMINotFoundError creates an error for missing AMIs
func AMINotFoundError(amiID string) *ContextualError {
	return New("find AMI", fmt.Errorf("AMI '%s' not found", amiID)).
		WithContext("The specified AMI does not exist or is not accessible.").
		WithSuggestions(
			"List your custom AMIs: aws-vscode list-amis",
			"Verify the AMI ID is correct",
			"Check if AMI exists in the current region",
			"Create a new AMI: aws-vscode create-ami INSTANCE_ID",
		)
}

// ValidationError creates an error for invalid input
func ValidationError(field string, value string, reason string) *ContextualError {
	return New("validate input", fmt.Errorf("invalid %s: %s", field, value)).
		WithContext(reason).
		WithSuggestions(
			"Check the command help for valid values: aws-vscode COMMAND --help",
			"Review the documentation for examples",
		)
}

// QuotaExceededError creates an error for AWS quota limits
func QuotaExceededError(resourceType string, cause error) *ContextualError {
	return New("create "+resourceType, cause).
		WithContext("You've reached an AWS service quota limit.").
		WithSuggestions(
			fmt.Sprintf("Terminate unused %ss to free up quota", resourceType),
			"Request a quota increase through AWS Service Quotas",
			"Check current usage: aws-vscode list",
		)
}

// SessionManagerError creates an error for SSM Session Manager issues
func SessionManagerError(cause error) *ContextualError {
	return New("connect via Session Manager", cause).
		WithContext("Unable to establish Session Manager connection.").
		WithSuggestions(
			"Verify Session Manager plugin is installed",
			"Check instance has SSM agent installed and running",
			"Verify IAM instance profile has AmazonSSMManagedInstanceCore policy",
			"Ensure instance is in a 'running' state",
		)
}

// CostTrackingError creates an error for cost tracking issues
func CostTrackingError(operation string, cause error) *ContextualError {
	return New(operation, cause).
		WithContext("Unable to retrieve cost information.").
		WithSuggestions(
			"Verify AWS Cost Explorer API permissions",
			"Check if Cost Explorer is enabled in your account",
			"Cost data may take 24 hours to appear",
			"Disable cost tracking: aws-vscode config set enable_cost_tracking false",
		)
}
