package aws

import (
	"testing"
)

func TestDefaultKeyPairStrategy(t *testing.T) {
	region := "us-west-2"
	strategy := DefaultKeyPairStrategy(region)

	if strategy.PreferExisting != true {
		t.Error("Expected PreferExisting to be true")
	}

	if strategy.DefaultPrefix != "aws-jupyter" {
		t.Errorf("Expected DefaultPrefix 'aws-jupyter', got: %s", strategy.DefaultPrefix)
	}

	if strategy.Region != region {
		t.Errorf("Expected Region %s, got: %s", region, strategy.Region)
	}

	if strategy.ForceCreate != false {
		t.Error("Expected ForceCreate to be false")
	}
}

func TestKeyPairStrategy_GetDefaultKeyName(t *testing.T) {
	tests := []struct {
		name          string
		strategy      KeyPairStrategy
		expectedName  string
	}{
		{
			name: "default strategy",
			strategy: KeyPairStrategy{
				DefaultPrefix: "aws-jupyter",
				Region:        "us-west-2",
			},
			expectedName: "aws-jupyter-us-west-2",
		},
		{
			name: "user specified key",
			strategy: KeyPairStrategy{
				DefaultPrefix: "aws-jupyter",
				Region:        "us-west-2",
				UserSpecified: "my-custom-key",
			},
			expectedName: "my-custom-key",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.strategy.GetDefaultKeyName()
			if result != test.expectedName {
				t.Errorf("Expected %s, got: %s", test.expectedName, result)
			}
		})
	}
}

func TestIsAwsJupyterKey(t *testing.T) {
	tests := []struct {
		keyName  string
		expected bool
	}{
		{"aws-jupyter-us-west-2", true},
		{"aws-jupyter-us-east-1", true},
		{"aws-jupyter-eu-west-1", true},
		{"my-custom-key", false},
		{"aws-jupyter", false}, // Must have region suffix
		{"", false},
	}

	for _, test := range tests {
		t.Run(test.keyName, func(t *testing.T) {
			result := IsAwsJupyterKey(test.keyName)
			if result != test.expected {
				t.Errorf("IsAwsJupyterKey(%s) = %v, expected %v", test.keyName, result, test.expected)
			}
		})
	}
}

func TestKeyPairInfo_Structure(t *testing.T) {
	info := &KeyPairInfo{
		Name:       "test-key",
		PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
		CreatedBy:  "aws-jupyter",
	}

	if info.Name != "test-key" {
		t.Errorf("Expected Name test-key, got: %s", info.Name)
	}

	if info.PrivateKey == "" {
		t.Error("Expected PrivateKey to be non-empty")
	}

	if info.CreatedBy != "aws-jupyter" {
		t.Errorf("Expected CreatedBy aws-jupyter, got: %s", info.CreatedBy)
	}
}

func TestKeyPairStrategy_PreferExistingBehavior(t *testing.T) {
	strategy := KeyPairStrategy{
		PreferExisting: true,
		DefaultPrefix:  "aws-jupyter",
		Region:         "us-west-2",
		ForceCreate:    false,
	}

	if !strategy.PreferExisting {
		t.Error("Expected PreferExisting to be true")
	}

	if strategy.ForceCreate {
		t.Error("Expected ForceCreate to be false when PreferExisting is true")
	}
}

func TestKeyPairStrategy_ForceCreateBehavior(t *testing.T) {
	strategy := KeyPairStrategy{
		PreferExisting: false,
		DefaultPrefix:  "test-key",
		Region:         "us-west-2",
		ForceCreate:    true,
	}

	if strategy.PreferExisting {
		t.Error("Expected PreferExisting to be false")
	}

	if !strategy.ForceCreate {
		t.Error("Expected ForceCreate to be true")
	}
}
