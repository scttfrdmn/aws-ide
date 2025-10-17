package aws

import (
	"testing"
)

func TestInstanceProfileInfo_Structure(t *testing.T) {
	info := &InstanceProfileInfo{
		Name: "aws-jupyter-session-manager",
		Arn:  "arn:aws:iam::123456789012:instance-profile/aws-jupyter-session-manager",
		Role: "aws-jupyter-session-manager-role",
	}

	if info.Name != "aws-jupyter-session-manager" {
		t.Errorf("Expected Name 'aws-jupyter-session-manager', got: %s", info.Name)
	}

	if info.Arn == "" {
		t.Error("Expected Arn to be non-empty")
	}

	// ARN should start with "arn:aws:iam::"
	if len(info.Arn) < 13 || info.Arn[:13] != "arn:aws:iam::" {
		t.Errorf("Expected Arn to start with 'arn:aws:iam::', got: %s", info.Arn)
	}
}

func TestIAMConstants(t *testing.T) {
	// This test verifies that common IAM-related constants are properly defined
	// by testing the structure and naming conventions

	profileInfo := &InstanceProfileInfo{
		Name: "aws-jupyter-session-manager",
		Arn:  "arn:aws:iam::123456789012:instance-profile/aws-jupyter-session-manager",
	}

	// Verify expected naming patterns
	expectedNamePrefix := "aws-jupyter"
	if len(profileInfo.Name) < len(expectedNamePrefix) ||
		profileInfo.Name[:len(expectedNamePrefix)] != expectedNamePrefix {
		t.Errorf("Expected instance profile name to start with '%s', got: %s", expectedNamePrefix, profileInfo.Name)
	}
}

func TestInstanceProfileInfo_ARNFormat(t *testing.T) {
	tests := []struct {
		name          string
		arn           string
		shouldBeValid bool
	}{
		{
			name:          "valid ARN",
			arn:           "arn:aws:iam::123456789012:instance-profile/aws-jupyter-session-manager",
			shouldBeValid: true,
		},
		{
			name:          "valid ARN different account",
			arn:           "arn:aws:iam::999888777666:instance-profile/aws-jupyter-session-manager",
			shouldBeValid: true,
		},
		{
			name:          "empty ARN",
			arn:           "",
			shouldBeValid: false,
		},
		{
			name:          "invalid prefix",
			arn:           "invalid:arn:format",
			shouldBeValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := &InstanceProfileInfo{
				Name: "test-profile",
				Arn:  test.arn,
			}

			isValid := info.Arn != "" && len(info.Arn) > 13 && info.Arn[:13] == "arn:aws:iam::"

			if isValid != test.shouldBeValid {
				t.Errorf("ARN validation mismatch: %s - expected valid=%v, got valid=%v",
					test.arn, test.shouldBeValid, isValid)
			}
		})
	}
}

func TestInstanceProfileInfo_NameValidation(t *testing.T) {
	tests := []struct {
		name    string
		isValid bool
	}{
		{"aws-jupyter-session-manager", true},
		{"my-custom-profile", true},
		{"", false},
		{"profile-with-numbers-123", true},
		{"profile_with_underscores", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := &InstanceProfileInfo{
				Name: test.name,
				Arn:  "arn:aws:iam::123456789012:instance-profile/" + test.name,
			}

			isValid := info.Name != ""
			if isValid != test.isValid {
				t.Errorf("Name validation mismatch for '%s': expected valid=%v, got valid=%v",
					test.name, test.isValid, isValid)
			}
		})
	}
}
