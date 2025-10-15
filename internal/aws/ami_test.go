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

func TestAMISelector_StructureIntegrity(t *testing.T) {
	selector := &AMISelector{
		region: "us-west-2",
	}

	if selector.region != "us-west-2" {
		t.Errorf("Expected region us-west-2, got: %s", selector.region)
	}
}
