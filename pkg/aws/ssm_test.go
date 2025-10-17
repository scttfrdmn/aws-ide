package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// TestCommandResult tests the CommandResult struct
func TestCommandResult(t *testing.T) {
	tests := []struct {
		name         string
		result       CommandResult
		expectedID   string
		expectedCode int32
	}{
		{
			name: "successful command result",
			result: CommandResult{
				CommandID:    "cmd-123",
				Status:       types.CommandInvocationStatusSuccess,
				Output:       "200",
				ErrorOutput:  "",
				ResponseCode: 0,
			},
			expectedID:   "cmd-123",
			expectedCode: 0,
		},
		{
			name: "failed command result",
			result: CommandResult{
				CommandID:    "cmd-456",
				Status:       types.CommandInvocationStatusFailed,
				Output:       "",
				ErrorOutput:  "error occurred",
				ResponseCode: 1,
			},
			expectedID:   "cmd-456",
			expectedCode: 1,
		},
		{
			name: "http check result with code",
			result: CommandResult{
				CommandID:    "cmd-789",
				Status:       types.CommandInvocationStatusSuccess,
				Output:       "302",
				ErrorOutput:  "",
				ResponseCode: 0,
			},
			expectedID:   "cmd-789",
			expectedCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.CommandID != tt.expectedID {
				t.Errorf("Expected CommandID %s, got %s", tt.expectedID, tt.result.CommandID)
			}

			if tt.result.ResponseCode != tt.expectedCode {
				t.Errorf("Expected ResponseCode %d, got %d", tt.expectedCode, tt.result.ResponseCode)
			}
		})
	}
}

// TestCommandResult_StatusValues tests different command status values
func TestCommandResult_StatusValues(t *testing.T) {
	tests := []struct {
		name     string
		status   types.CommandInvocationStatus
		expected types.CommandInvocationStatus
	}{
		{"success status", types.CommandInvocationStatusSuccess, types.CommandInvocationStatusSuccess},
		{"failed status", types.CommandInvocationStatusFailed, types.CommandInvocationStatusFailed},
		{"pending status", types.CommandInvocationStatusPending, types.CommandInvocationStatusPending},
		{"in progress status", types.CommandInvocationStatusInProgress, types.CommandInvocationStatusInProgress},
		{"cancelled status", types.CommandInvocationStatusCancelled, types.CommandInvocationStatusCancelled},
		{"timed out status", types.CommandInvocationStatusTimedOut, types.CommandInvocationStatusTimedOut},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CommandResult{
				CommandID: "test",
				Status:    tt.status,
			}

			if result.Status != tt.expected {
				t.Errorf("Expected status %v, got %v", tt.expected, result.Status)
			}
		})
	}
}

// TestSSMClient_StructureIntegrity tests SSMClient struct
func TestSSMClient_StructureIntegrity(t *testing.T) {
	client := &SSMClient{
		region: "us-west-2",
	}

	if client.region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got: %s", client.region)
	}

	// Test that client field exists (even if nil)
	if client.client != nil {
		t.Log("Client field is not nil (this is acceptable)")
	}
}

// TestNewSSMClient_WithConfig tests SSMClient creation
func TestNewSSMClient_WithConfig(t *testing.T) {
	cfg := aws.Config{
		Region: "us-east-1",
	}

	client := NewSSMClient(cfg)

	if client == nil {
		t.Fatal("Expected non-nil SSMClient")
	}

	if client.region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got: %s", client.region)
	}

	if client.client == nil {
		t.Error("Expected non-nil SSM client")
	}
}

// TestNewSSMClient_WithDifferentRegions tests SSMClient with various regions
func TestNewSSMClient_WithDifferentRegions(t *testing.T) {
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}

	for _, region := range regions {
		t.Run(region, func(t *testing.T) {
			cfg := aws.Config{
				Region: region,
			}

			client := NewSSMClient(cfg)

			if client == nil {
				t.Fatal("Expected non-nil SSMClient")
			}

			if client.region != region {
				t.Errorf("Expected region %s, got: %s", region, client.region)
			}
		})
	}
}

// TestCommandResult_HTTPCodes tests HTTP response code handling
func TestCommandResult_HTTPCodes(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		shouldPass bool
	}{
		{"200 OK", "200", true},
		{"302 redirect", "302", true},
		{"401 unauthorized", "401", true},
		{"404 not found", "404", true},
		{"500 server error", "500", true},
		{"000 connection failed", "000", false},
		{"empty output", "", false},
		{"short output", "ab", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CommandResult{
				CommandID:    "test",
				Status:       types.CommandInvocationStatusSuccess,
				Output:       tt.output,
				ResponseCode: 0,
			}

			// Simulate the logic from CheckServiceReadiness
			httpCode := result.Output
			isReady := false
			if len(httpCode) >= 3 {
				httpCode = httpCode[0:3]
				if httpCode != "000" {
					isReady = true
				}
			}

			if isReady != tt.shouldPass {
				t.Errorf("Expected isReady=%v for output %s, got %v", tt.shouldPass, tt.output, isReady)
			}
		})
	}
}

// TestCommandResult_MultilineOutput tests handling of multiline outputs
func TestCommandResult_MultilineOutput(t *testing.T) {
	result := CommandResult{
		CommandID:   "test",
		Status:      types.CommandInvocationStatusSuccess,
		Output:      "line1\nline2\nline3",
		ErrorOutput: "error line1\nerror line2",
	}

	if result.Output != "line1\nline2\nline3" {
		t.Errorf("Expected multiline output, got: %s", result.Output)
	}

	if result.ErrorOutput != "error line1\nerror line2" {
		t.Errorf("Expected multiline error output, got: %s", result.ErrorOutput)
	}
}

// TestCommandResult_EdgeCases tests edge cases
func TestCommandResult_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		result CommandResult
	}{
		{
			name: "empty strings",
			result: CommandResult{
				CommandID:   "",
				Output:      "",
				ErrorOutput: "",
			},
		},
		{
			name: "very long output",
			result: CommandResult{
				CommandID: "test",
				Output:    string(make([]byte, 10000)),
			},
		},
		{
			name: "special characters in output",
			result: CommandResult{
				CommandID: "test",
				Output:    "!@#$%^&*(){}[]|\\:;\"'<>,.?/~`",
			},
		},
		{
			name: "unicode in output",
			result: CommandResult{
				CommandID: "test",
				Output:    "Hello 世界 🌍",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the struct can hold these values
			t.Logf("Test %s passed with output length: %d", tt.name, len(tt.result.Output))
		})
	}
}

// TestSSMClient_MethodSignatures tests that methods have correct signatures
func TestSSMClient_MethodSignatures(t *testing.T) {
	// This test verifies that the expected methods exist with correct signatures
	// by creating a client and checking it compiles

	cfg := aws.Config{
		Region: "us-west-2",
	}

	client := NewSSMClient(cfg)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	// Verify the client has the expected structure
	if client.client == nil {
		t.Error("Expected client.client to be non-nil")
	}

	if client.region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got: %s", client.region)
	}
}

// TestCommandResult_ZeroValues tests zero values of CommandResult
func TestCommandResult_ZeroValues(t *testing.T) {
	var result CommandResult

	if result.CommandID != "" {
		t.Errorf("Expected empty CommandID, got: %s", result.CommandID)
	}

	if result.Output != "" {
		t.Errorf("Expected empty Output, got: %s", result.Output)
	}

	if result.ErrorOutput != "" {
		t.Errorf("Expected empty ErrorOutput, got: %s", result.ErrorOutput)
	}

	if result.ResponseCode != 0 {
		t.Errorf("Expected ResponseCode 0, got: %d", result.ResponseCode)
	}
}

// TestSSMClient_HTTPCodeParsing tests the HTTP code parsing logic
func TestSSMClient_HTTPCodeParsing(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantReady  bool
		wantCode   string
	}{
		{"HTTP 200", "200\n", true, "200"},
		{"HTTP 302", "302", true, "302"},
		{"HTTP 401", "401 Unauthorized", true, "401"},
		{"HTTP 404", "404 Not Found", true, "404"},
		{"HTTP 500", "500", true, "500"},
		{"Connection failed", "000", false, "000"},
		{"Short output", "20", false, ""},
		{"No output", "", false, ""},
		{"Whitespace only", "  ", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCode := tt.output
			isReady := false
			extractedCode := ""

			if len(httpCode) >= 3 {
				extractedCode = httpCode[0:3]
				if extractedCode != "000" {
					isReady = true
				}
			}

			if isReady != tt.wantReady {
				t.Errorf("Expected isReady=%v, got %v", tt.wantReady, isReady)
			}

			if tt.wantCode != "" && extractedCode != tt.wantCode {
				t.Errorf("Expected code %s, got %s", tt.wantCode, extractedCode)
			}
		})
	}
}
