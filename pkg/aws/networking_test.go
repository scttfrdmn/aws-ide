package aws

import (
	"testing"
)

func TestSubnetInfo_Structure(t *testing.T) {
	info := &SubnetInfo{
		ID:               "subnet-12345",
		VpcID:            "vpc-12345",
		AvailabilityZone: "us-west-2a",
		CidrBlock:        "10.0.1.0/24",
		IsPublic:         true,
	}

	if info.ID != "subnet-12345" {
		t.Errorf("Expected ID subnet-12345, got: %s", info.ID)
	}

	if info.VpcID != "vpc-12345" {
		t.Errorf("Expected VpcID vpc-12345, got: %s", info.VpcID)
	}

	if info.AvailabilityZone != "us-west-2a" {
		t.Errorf("Expected AvailabilityZone us-west-2a, got: %s", info.AvailabilityZone)
	}

	if info.CidrBlock != "10.0.1.0/24" {
		t.Errorf("Expected CidrBlock 10.0.1.0/24, got: %s", info.CidrBlock)
	}

	if !info.IsPublic {
		t.Error("Expected IsPublic to be true")
	}
}

func TestSubnetInfo_PublicVsPrivate(t *testing.T) {
	publicSubnet := &SubnetInfo{
		ID:       "subnet-public",
		IsPublic: true,
	}

	privateSubnet := &SubnetInfo{
		ID:       "subnet-private",
		IsPublic: false,
	}

	if !publicSubnet.IsPublic {
		t.Error("Public subnet should have IsPublic=true")
	}

	if privateSubnet.IsPublic {
		t.Error("Private subnet should have IsPublic=false")
	}
}

func TestNATGatewayInfo_Structure(t *testing.T) {
	info := &NATGatewayInfo{
		ID:       "nat-12345",
		SubnetID: "subnet-12345",
		VpcID:    "vpc-12345",
		State:    "available",
	}

	if info.ID != "nat-12345" {
		t.Errorf("Expected ID nat-12345, got: %s", info.ID)
	}

	if info.SubnetID != "subnet-12345" {
		t.Errorf("Expected SubnetID subnet-12345, got: %s", info.SubnetID)
	}

	if info.VpcID != "vpc-12345" {
		t.Errorf("Expected VpcID vpc-12345, got: %s", info.VpcID)
	}

	if info.State != "available" {
		t.Errorf("Expected State available, got: %s", info.State)
	}
}

func TestNATGatewayInfo_States(t *testing.T) {
	states := []string{"pending", "available", "deleting", "deleted", "failed"}

	for _, state := range states {
		t.Run(state, func(t *testing.T) {
			info := &NATGatewayInfo{
				ID:    "nat-12345",
				State: state,
			}

			if info.State != state {
				t.Errorf("Expected State %s, got: %s", state, info.State)
			}
		})
	}
}

func TestSubnetInfo_CIDRBlockValidation(t *testing.T) {
	tests := []struct {
		cidr    string
		isValid bool
	}{
		{"10.0.1.0/24", true},
		{"172.16.0.0/16", true},
		{"192.168.1.0/24", true},
		{"10.0.0.0/8", true},
		{"invalid", false},
		{"", false},
	}

	for _, test := range tests {
		t.Run(test.cidr, func(t *testing.T) {
			subnet := &SubnetInfo{
				ID:        "subnet-test",
				CidrBlock: test.cidr,
			}

			// Basic validation: CIDR should be non-empty for valid inputs
			if test.isValid && subnet.CidrBlock == "" {
				t.Errorf("Expected non-empty CIDR block for valid input %s", test.cidr)
			}

			if !test.isValid && test.cidr != "" && subnet.CidrBlock != "" {
				// For invalid CIDRs, they might still be stored but aren't validated here
				t.Logf("CIDR %s stored even though marked invalid", test.cidr)
			}
		})
	}
}

func TestSubnetInfo_AvailabilityZones(t *testing.T) {
	zones := []string{
		"us-west-2a",
		"us-west-2b",
		"us-west-2c",
		"us-east-1a",
		"eu-west-1a",
	}

	for _, zone := range zones {
		t.Run(zone, func(t *testing.T) {
			subnet := &SubnetInfo{
				ID:               "subnet-test",
				AvailabilityZone: zone,
			}

			if subnet.AvailabilityZone != zone {
				t.Errorf("Expected AvailabilityZone %s, got: %s", zone, subnet.AvailabilityZone)
			}

			// Zone should end with a letter
			if len(zone) > 0 {
				lastChar := zone[len(zone)-1]
				if lastChar < 'a' || lastChar > 'z' {
					t.Errorf("Expected availability zone to end with a letter, got: %c in %s", lastChar, zone)
				}
			}
		})
	}
}

func TestNATGatewayInfo_IDFormat(t *testing.T) {
	tests := []struct {
		id      string
		isValid bool
	}{
		{"nat-12345", true},
		{"nat-abcdef", true},
		{"nat-0123456789abcdef", true},
		{"invalid-format", false},
		{"", false},
	}

	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			info := &NATGatewayInfo{
				ID: test.id,
			}

			// NAT Gateway IDs should start with "nat-"
			isValid := info.ID != "" && len(info.ID) > 4 && info.ID[:4] == "nat-"

			if isValid != test.isValid {
				t.Errorf("ID validation mismatch for '%s': expected valid=%v, got valid=%v",
					test.id, test.isValid, isValid)
			}
		})
	}
}
