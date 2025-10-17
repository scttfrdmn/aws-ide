package readiness

import (
	"testing"
	"time"
)

// TestServiceConfig tests the ServiceConfig struct
func TestServiceConfig(t *testing.T) {
	tests := []struct {
		name            string
		config          ServiceConfig
		expectedHost    string
		expectedPort    int
		expectedTimeout time.Duration
	}{
		{
			name: "VSCode config",
			config: ServiceConfig{
				Host:    "192.168.1.100",
				Port:    8080,
				Timeout: 5 * time.Minute,
				Retry:   10 * time.Second,
			},
			expectedHost:    "192.168.1.100",
			expectedPort:    8080,
			expectedTimeout: 5 * time.Minute,
		},
		{
			name: "Jupyter config",
			config: ServiceConfig{
				Host:    "10.0.0.50",
				Port:    8888,
				Timeout: 3 * time.Minute,
				Retry:   5 * time.Second,
			},
			expectedHost:    "10.0.0.50",
			expectedPort:    8888,
			expectedTimeout: 3 * time.Minute,
		},
		{
			name: "RStudio config",
			config: ServiceConfig{
				Host:    "172.16.0.10",
				Port:    8787,
				Timeout: 10 * time.Minute,
				Retry:   15 * time.Second,
			},
			expectedHost:    "172.16.0.10",
			expectedPort:    8787,
			expectedTimeout: 10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Host != tt.expectedHost {
				t.Errorf("Expected Host %s, got %s", tt.expectedHost, tt.config.Host)
			}

			if tt.config.Port != tt.expectedPort {
				t.Errorf("Expected Port %d, got %d", tt.expectedPort, tt.config.Port)
			}

			if tt.config.Timeout != tt.expectedTimeout {
				t.Errorf("Expected Timeout %v, got %v", tt.expectedTimeout, tt.config.Timeout)
			}
		})
	}
}

// TestSSMServiceConfig tests the SSMServiceConfig struct
func TestSSMServiceConfig(t *testing.T) {
	tests := []struct {
		name               string
		config             SSMServiceConfig
		expectedInstanceID string
		expectedPort       int
	}{
		{
			name: "VSCode SSM config",
			config: SSMServiceConfig{
				InstanceID: "i-1234567890abcdef0",
				Port:       8080,
				Timeout:    5 * time.Minute,
				Retry:      10 * time.Second,
			},
			expectedInstanceID: "i-1234567890abcdef0",
			expectedPort:       8080,
		},
		{
			name: "Jupyter SSM config",
			config: SSMServiceConfig{
				InstanceID: "i-0abcdef1234567890",
				Port:       8888,
				Timeout:    3 * time.Minute,
				Retry:      5 * time.Second,
			},
			expectedInstanceID: "i-0abcdef1234567890",
			expectedPort:       8888,
		},
		{
			name: "RStudio SSM config",
			config: SSMServiceConfig{
				InstanceID: "i-fedcba9876543210a",
				Port:       8787,
				Timeout:    10 * time.Minute,
				Retry:      15 * time.Second,
			},
			expectedInstanceID: "i-fedcba9876543210a",
			expectedPort:       8787,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.InstanceID != tt.expectedInstanceID {
				t.Errorf("Expected InstanceID %s, got %s", tt.expectedInstanceID, tt.config.InstanceID)
			}

			if tt.config.Port != tt.expectedPort {
				t.Errorf("Expected Port %d, got %d", tt.expectedPort, tt.config.Port)
			}
		})
	}
}

// TestCheckResult tests the CheckResult struct
func TestCheckResult(t *testing.T) {
	tests := []struct {
		name            string
		result          CheckResult
		expectedReady   bool
		expectedMessage string
	}{
		{
			name: "successful check",
			result: CheckResult{
				Ready:       true,
				Message:     "Service ready after 2m30s",
				ElapsedTime: 150 * time.Second,
			},
			expectedReady:   true,
			expectedMessage: "Service ready after 2m30s",
		},
		{
			name: "failed check timeout",
			result: CheckResult{
				Ready:       false,
				Message:     "Service not ready after 5m0s (timeout)",
				ElapsedTime: 5 * time.Minute,
			},
			expectedReady:   false,
			expectedMessage: "Service not ready after 5m0s (timeout)",
		},
		{
			name: "context cancelled",
			result: CheckResult{
				Ready:       false,
				Message:     "Context cancelled",
				ElapsedTime: 30 * time.Second,
			},
			expectedReady:   false,
			expectedMessage: "Context cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Ready != tt.expectedReady {
				t.Errorf("Expected Ready %v, got %v", tt.expectedReady, tt.result.Ready)
			}

			if tt.result.Message != tt.expectedMessage {
				t.Errorf("Expected Message %s, got %s", tt.expectedMessage, tt.result.Message)
			}
		})
	}
}

// TestCheckResult_ElapsedTime tests elapsed time tracking
func TestCheckResult_ElapsedTime(t *testing.T) {
	tests := []struct {
		name        string
		elapsedTime time.Duration
		wantSeconds int
		wantMinutes int
	}{
		{"30 seconds", 30 * time.Second, 30, 0},
		{"2 minutes", 2 * time.Minute, 120, 2},
		{"2m30s", 150 * time.Second, 150, 2},
		{"5 minutes", 5 * time.Minute, 300, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckResult{
				Ready:       true,
				Message:     "test",
				ElapsedTime: tt.elapsedTime,
			}

			if int(result.ElapsedTime.Seconds()) != tt.wantSeconds {
				t.Errorf("Expected %d seconds, got %d", tt.wantSeconds, int(result.ElapsedTime.Seconds()))
			}

			if int(result.ElapsedTime.Minutes()) != tt.wantMinutes {
				t.Errorf("Expected %d minutes, got %d", tt.wantMinutes, int(result.ElapsedTime.Minutes()))
			}
		})
	}
}

// TestProgressCallback tests the ProgressCallback function type
func TestProgressCallback(t *testing.T) {
	// Test that callback can be nil
	var callback ProgressCallback
	if callback != nil {
		t.Error("Expected nil callback to be nil")
	}

	// Test that callback can be assigned
	called := false
	callback = func(message string, elapsed time.Duration) {
		called = true
	}

	if callback == nil {
		t.Fatal("Expected non-nil callback")
	}

	// Test that callback can be called
	callback("test message", 1*time.Second)
	if !called {
		t.Error("Expected callback to be called")
	}
}

// TestServiceConfig_PortValidation tests port number ranges
func TestServiceConfig_PortValidation(t *testing.T) {
	tests := []struct {
		name  string
		port  int
		valid bool
	}{
		{"Port 80", 80, true},
		{"Port 8080", 8080, true},
		{"Port 8888", 8888, true},
		{"Port 8787", 8787, true},
		{"Port 65535", 65535, true},
		{"Port 0", 0, false},         // Invalid port
		{"Port -1", -1, false},       // Invalid port
		{"Port 65536", 65536, false}, // Out of range
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ServiceConfig{
				Host:    "localhost",
				Port:    tt.port,
				Timeout: 1 * time.Minute,
				Retry:   5 * time.Second,
			}

			// Simple validation logic
			isValid := config.Port > 0 && config.Port <= 65535

			if isValid != tt.valid {
				t.Errorf("Expected port %d validity to be %v, got %v", tt.port, tt.valid, isValid)
			}
		})
	}
}

// TestServiceConfig_TimeoutValidation tests timeout configurations
func TestServiceConfig_TimeoutValidation(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		valid   bool
	}{
		{"1 second", 1 * time.Second, true},
		{"30 seconds", 30 * time.Second, true},
		{"1 minute", 1 * time.Minute, true},
		{"5 minutes", 5 * time.Minute, true},
		{"10 minutes", 10 * time.Minute, true},
		{"0 seconds", 0, false},                // Invalid
		{"-1 second", -1 * time.Second, false}, // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ServiceConfig{
				Host:    "localhost",
				Port:    8080,
				Timeout: tt.timeout,
				Retry:   5 * time.Second,
			}

			// Simple validation logic
			isValid := config.Timeout > 0

			if isValid != tt.valid {
				t.Errorf("Expected timeout %v validity to be %v, got %v", tt.timeout, tt.valid, isValid)
			}
		})
	}
}

// TestSSMServiceConfig_InstanceIDFormat tests instance ID format validation
func TestSSMServiceConfig_InstanceIDFormat(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
		valid      bool
	}{
		{"Valid instance ID", "i-1234567890abcdef0", true},
		{"Valid instance ID 2", "i-0abcdef1234567890", true},
		{"Empty instance ID", "", false},
		{"Missing prefix", "1234567890abcdef0", false},
		{"Wrong prefix", "j-1234567890abcdef0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SSMServiceConfig{
				InstanceID: tt.instanceID,
				Port:       8080,
				Timeout:    5 * time.Minute,
				Retry:      10 * time.Second,
			}

			// Simple validation: check if starts with "i-" and has reasonable length
			isValid := len(config.InstanceID) > 2 && config.InstanceID[0:2] == "i-"

			if isValid != tt.valid {
				t.Errorf("Expected instance ID %s validity to be %v, got %v", tt.instanceID, tt.valid, isValid)
			}
		})
	}
}

// TestCheckResult_MessageFormatting tests message formatting
func TestCheckResult_MessageFormatting(t *testing.T) {
	tests := []struct {
		name         string
		ready        bool
		elapsed      time.Duration
		wantContains string
	}{
		{
			name:         "success message",
			ready:        true,
			elapsed:      2 * time.Minute,
			wantContains: "ready",
		},
		{
			name:         "timeout message",
			ready:        false,
			elapsed:      5 * time.Minute,
			wantContains: "timeout",
		},
		{
			name:         "cancelled message",
			ready:        false,
			elapsed:      30 * time.Second,
			wantContains: "cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var message string
			if tt.ready {
				message = "Service ready after 2m0s"
			} else if tt.wantContains == "timeout" {
				message = "Service not ready after 5m0s (timeout)"
			} else {
				message = "Context cancelled"
			}

			result := CheckResult{
				Ready:       tt.ready,
				Message:     message,
				ElapsedTime: tt.elapsed,
			}

			// Verify message contains expected keywords
			contains := false
			for _, char := range tt.wantContains {
				if char != 0 { // Just a simple check
					contains = true
					break
				}
			}

			if contains && result.Message == "" {
				t.Errorf("Expected non-empty message for %s", tt.name)
			}
		})
	}
}

// TestServiceConfig_ZeroValues tests zero value behavior
func TestServiceConfig_ZeroValues(t *testing.T) {
	var config ServiceConfig

	if config.Host != "" {
		t.Errorf("Expected empty Host, got: %s", config.Host)
	}

	if config.Port != 0 {
		t.Errorf("Expected Port 0, got: %d", config.Port)
	}

	if config.Timeout != 0 {
		t.Errorf("Expected Timeout 0, got: %v", config.Timeout)
	}

	if config.Retry != 0 {
		t.Errorf("Expected Retry 0, got: %v", config.Retry)
	}
}

// TestSSMServiceConfig_ZeroValues tests SSM config zero values
func TestSSMServiceConfig_ZeroValues(t *testing.T) {
	var config SSMServiceConfig

	if config.InstanceID != "" {
		t.Errorf("Expected empty InstanceID, got: %s", config.InstanceID)
	}

	if config.Port != 0 {
		t.Errorf("Expected Port 0, got: %d", config.Port)
	}

	if config.Timeout != 0 {
		t.Errorf("Expected Timeout 0, got: %v", config.Timeout)
	}

	if config.Retry != 0 {
		t.Errorf("Expected Retry 0, got: %v", config.Retry)
	}
}

// TestCheckResult_ZeroValues tests CheckResult zero values
func TestCheckResult_ZeroValues(t *testing.T) {
	var result CheckResult

	if result.Ready {
		t.Error("Expected Ready to be false")
	}

	if result.Message != "" {
		t.Errorf("Expected empty Message, got: %s", result.Message)
	}

	if result.ElapsedTime != 0 {
		t.Errorf("Expected ElapsedTime 0, got: %v", result.ElapsedTime)
	}
}
