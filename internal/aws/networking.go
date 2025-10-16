package aws

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
)

// SubnetInfo contains information about a subnet
type SubnetInfo struct {
	ID               string
	VpcID            string
	AvailabilityZone string
	CidrBlock        string
	IsPublic         bool
}

// NATGatewayInfo contains information about a NAT Gateway
type NATGatewayInfo struct {
	ID       string
	SubnetID string
	VpcID    string
	State    string
}

// GetSubnet finds an appropriate subnet based on the subnet type preference and optional availability zone
func (e *EC2Client) GetSubnet(ctx context.Context, subnetType string, availabilityZone string) (*SubnetInfo, error) {
	vpcID, err := e.getDefaultVpcID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default VPC: %w", err)
	}

	// Build filters for subnet query
	filters := []types.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []string{vpcID},
		},
	}

	// Add availability zone filter if specified
	if availabilityZone != "" {
		filters = append(filters, types.Filter{
			Name:   aws.String("availability-zone"),
			Values: []string{availabilityZone},
		})
	}

	// Describe subnets in the VPC
	result, err := e.client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe subnets: %w", err)
	}

	if len(result.Subnets) == 0 {
		if availabilityZone != "" {
			return nil, fmt.Errorf("no subnets found in VPC %s in availability zone %s", vpcID, availabilityZone)
		}
		return nil, fmt.Errorf("no subnets found in VPC %s", vpcID)
	}

	// Find the best subnet based on type preference
	var bestSubnet *types.Subnet
	for _, subnet := range result.Subnets {
		isPublic := aws.ToBool(subnet.MapPublicIpOnLaunch)

		if subnetType == "public" && isPublic {
			bestSubnet = &subnet
			break
		} else if subnetType == "private" && !isPublic {
			bestSubnet = &subnet
			break
		}
	}

	// If no matching subnet found, use the first available
	if bestSubnet == nil {
		bestSubnet = &result.Subnets[0]
		actualType := "private"
		if aws.ToBool(bestSubnet.MapPublicIpOnLaunch) {
			actualType = "public"
		}
		fmt.Printf("⚠️  No %s subnet found, using %s subnet instead\n", subnetType, actualType)
	}

	return &SubnetInfo{
		ID:               aws.ToString(bestSubnet.SubnetId),
		VpcID:            aws.ToString(bestSubnet.VpcId),
		AvailabilityZone: aws.ToString(bestSubnet.AvailabilityZone),
		CidrBlock:        aws.ToString(bestSubnet.CidrBlock),
		IsPublic:         aws.ToBool(bestSubnet.MapPublicIpOnLaunch),
	}, nil
}

// GetOrCreateNATGateway creates a NAT Gateway if it doesn't exist for the VPC
func (e *EC2Client) GetOrCreateNATGateway(ctx context.Context, vpcID string) (*NATGatewayInfo, error) {
	// First, check if a NAT Gateway already exists in this VPC
	existing, err := e.findExistingNATGateway(ctx, vpcID)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing NAT Gateway: %w", err)
	}
	if existing != nil {
		fmt.Printf("Using existing NAT Gateway: %s\n", existing.ID)
		return existing, nil
	}

	// Find a public subnet for the NAT Gateway
	publicSubnet, err := e.GetSubnet(ctx, "public", "")
	if err != nil {
		return nil, fmt.Errorf("failed to find public subnet for NAT Gateway: %w", err)
	}

	if !publicSubnet.IsPublic {
		return nil, fmt.Errorf("no public subnet available for NAT Gateway")
	}

	// Allocate an Elastic IP for the NAT Gateway
	fmt.Println("Allocating Elastic IP for NAT Gateway...")
	eipResult, err := e.client.AllocateAddress(ctx, &ec2.AllocateAddressInput{
		Domain: types.DomainTypeVpc,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeElasticIp,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String("aws-jupyter-nat-gateway-eip")},
					{Key: aws.String("CreatedBy"), Value: aws.String("aws-jupyter-cli")},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to allocate Elastic IP: %w", err)
	}

	// Create the NAT Gateway
	fmt.Printf("Creating NAT Gateway in subnet %s...\n", publicSubnet.ID)
	natResult, err := e.client.CreateNatGateway(ctx, &ec2.CreateNatGatewayInput{
		SubnetId:     aws.String(publicSubnet.ID),
		AllocationId: eipResult.AllocationId,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeNatgateway,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String("aws-jupyter-nat-gateway")},
					{Key: aws.String("CreatedBy"), Value: aws.String("aws-jupyter-cli")},
				},
			},
		},
	})
	if err != nil {
		// Clean up the allocated EIP if NAT Gateway creation fails
		if _, releaseErr := e.client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: eipResult.AllocationId,
		}); releaseErr != nil {
			// Log cleanup failure but return original error
			fmt.Printf("Warning: Failed to release Elastic IP after error: %v\n", releaseErr)
		}
		return nil, fmt.Errorf("failed to create NAT Gateway: %w", err)
	}

	natGateway := &NATGatewayInfo{
		ID:       aws.ToString(natResult.NatGateway.NatGatewayId),
		SubnetID: publicSubnet.ID,
		VpcID:    vpcID,
		State:    string(natResult.NatGateway.State),
	}

	// Wait for NAT Gateway to become available
	fmt.Printf("Waiting for NAT Gateway %s to become available...\n", natGateway.ID)
	if err := e.waitForNATGatewayAvailable(ctx, natGateway.ID); err != nil {
		return nil, fmt.Errorf("NAT Gateway did not become available: %w", err)
	}

	fmt.Printf("✓ NAT Gateway %s is now available\n", natGateway.ID)
	return natGateway, nil
}

// findExistingNATGateway looks for an existing NAT Gateway in the VPC
func (e *EC2Client) findExistingNATGateway(ctx context.Context, vpcID string) (*NATGatewayInfo, error) {
	result, err := e.client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
		Filter: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(result.NatGateways) == 0 {
		return nil, nil
	}

	// Return the first available NAT Gateway
	ng := result.NatGateways[0]
	return &NATGatewayInfo{
		ID:       aws.ToString(ng.NatGatewayId),
		SubnetID: aws.ToString(ng.SubnetId),
		VpcID:    aws.ToString(ng.VpcId),
		State:    string(ng.State),
	}, nil
}

// waitForNATGatewayAvailable waits for the NAT Gateway to become available
func (e *EC2Client) waitForNATGatewayAvailable(ctx context.Context, natGatewayID string) error {
	maxWaitTime := 5 * time.Minute
	checkInterval := 15 * time.Second
	timeout := time.After(maxWaitTime)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for NAT Gateway to become available")
		case <-ticker.C:
			result, err := e.client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
				NatGatewayIds: []string{natGatewayID},
			})
			if err != nil {
				return fmt.Errorf("failed to check NAT Gateway status: %w", err)
			}

			if len(result.NatGateways) == 0 {
				return fmt.Errorf("NAT Gateway not found")
			}

			state := result.NatGateways[0].State
			switch state {
			case types.NatGatewayStateAvailable:
				return nil
			case types.NatGatewayStateFailed:
				return fmt.Errorf("NAT Gateway creation failed")
			case types.NatGatewayStateDeleted, types.NatGatewayStateDeleting:
				return fmt.Errorf("NAT Gateway was deleted")
			default:
				// Continue waiting for pending states
				fmt.Printf("NAT Gateway state: %s\n", state)
			}
		}
	}
}

// UpdatePrivateSubnetRoutes updates the route table for private subnets to use the NAT Gateway
func (e *EC2Client) UpdatePrivateSubnetRoutes(ctx context.Context, subnetID, natGatewayID string) error {
	// Find the route table associated with this subnet
	routeTables, err := e.client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("association.subnet-id"),
				Values: []string{subnetID},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to find route table for subnet: %w", err)
	}

	var routeTableID string
	if len(routeTables.RouteTables) == 0 {
		// Use the main route table if no specific association
		vpcID, err := e.getVpcIDFromSubnet(ctx, subnetID)
		if err != nil {
			return fmt.Errorf("failed to get VPC ID: %w", err)
		}

		mainRouteTables, err := e.client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: []string{vpcID},
				},
				{
					Name:   aws.String("association.main"),
					Values: []string{"true"},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to find main route table: %w", err)
		}

		if len(mainRouteTables.RouteTables) == 0 {
			return fmt.Errorf("no main route table found")
		}

		routeTableID = aws.ToString(mainRouteTables.RouteTables[0].RouteTableId)
	} else {
		routeTableID = aws.ToString(routeTables.RouteTables[0].RouteTableId)
	}

	// Add route to the NAT Gateway for internet access (0.0.0.0/0)
	fmt.Printf("Adding NAT Gateway route to route table %s\n", routeTableID)
	_, err = e.client.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         aws.String(routeTableID),
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		NatGatewayId:         aws.String(natGatewayID),
	})
	if err != nil {
		// Route might already exist, check if it's just a duplicate
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "RouteAlreadyExists" {
			fmt.Println("Route to NAT Gateway already exists")
			return nil
		}
		return fmt.Errorf("failed to create route to NAT Gateway: %w", err)
	}

	fmt.Println("✓ Route to NAT Gateway created successfully")
	return nil
}

// getVpcIDFromSubnet gets the VPC ID for a given subnet
func (e *EC2Client) getVpcIDFromSubnet(ctx context.Context, subnetID string) (string, error) {
	result, err := e.client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	})
	if err != nil {
		return "", err
	}

	if len(result.Subnets) == 0 {
		return "", fmt.Errorf("subnet not found")
	}

	return aws.ToString(result.Subnets[0].VpcId), nil
}
