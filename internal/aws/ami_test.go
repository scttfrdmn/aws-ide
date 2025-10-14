package aws

import (
	"testing"
)

func TestNewAMISelector(t *testing.T) {
	region := "us-west-2"
	selector := NewAMISelector(region)

	if selector == nil {
		t.Fatal("NewAMISelector returned nil")
	}

	if selector.region != region {
		t.Errorf("Expected region %s, got: %s", region, selector.region)
	}
}

func TestGetDefaultAMI(t *testing.T) {
	selector := NewAMISelector("us-west-2")
	ami := selector.GetDefaultAMI()

	if ami == "" {
		t.Error("GetDefaultAMI returned empty string")
	}

	// AMI ID should start with "ami-"
	if len(ami) < 4 || ami[:4] != "ami-" {
		t.Errorf("Expected AMI ID to start with 'ami-', got: %s", ami)
	}
}

func TestGetDefaultAMI_DifferentRegions(t *testing.T) {
	tests := []struct {
		region string
	}{
		{"us-west-2"},
		{"us-east-1"},
		{"eu-west-1"},
		{"ap-southeast-1"},
	}

	for _, test := range tests {
		t.Run(test.region, func(t *testing.T) {
			selector := NewAMISelector(test.region)
			ami := selector.GetDefaultAMI()

			if ami == "" {
				t.Errorf("GetDefaultAMI returned empty string for region %s", test.region)
			}

			if len(ami) < 4 || ami[:4] != "ami-" {
				t.Errorf("Expected AMI ID to start with 'ami-' for region %s, got: %s", test.region, ami)
			}
		})
	}
}

func TestAMISelector_StructureIntegrity(t *testing.T) {
	selector := &AMISelector{
		region: "us-west-2",
	}

	if selector.region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got: %s", selector.region)
	}

	// Test that GetDefaultAMI can be called
	ami := selector.GetDefaultAMI()
	if ami == "" {
		t.Error("GetDefaultAMI returned empty string")
	}
}
