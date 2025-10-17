package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
)

// KeyPairInfo contains information about an AWS key pair
type KeyPairInfo struct {
	Name        string
	Fingerprint string
	PrivateKey  string // Only populated when creating new keys
	Region      string
	CreatedBy   string // "aws-jupyter" or "user"
}

// KeyPairStrategy defines how to handle key pair selection and creation
type KeyPairStrategy struct {
	PreferExisting bool   // Try to reuse existing keys
	DefaultPrefix  string // "aws-jupyter"
	UserSpecified  string // User's custom key name
	Region         string // Current region
	ForceCreate    bool   // Force creation of new key even if exists
}

// DefaultKeyPairStrategy returns the recommended strategy for aws-jupyter
func DefaultKeyPairStrategy(region string) KeyPairStrategy {
	return KeyPairStrategy{
		PreferExisting: true,
		DefaultPrefix:  "aws-jupyter",
		Region:         region,
		ForceCreate:    false,
	}
}

// GetDefaultKeyName returns the default key name for a region
func (s KeyPairStrategy) GetDefaultKeyName() string {
	if s.UserSpecified != "" {
		return s.UserSpecified
	}
	return fmt.Sprintf("%s-%s", s.DefaultPrefix, s.Region)
}

// IsAwsJupyterKey returns true if the key name was created by aws-jupyter
func IsAwsJupyterKey(keyName string) bool {
	return strings.HasPrefix(keyName, "aws-jupyter-")
}

// ListKeyPairs returns all key pairs in the current region
func (e *EC2Client) ListKeyPairs(ctx context.Context) ([]KeyPairInfo, error) {
	result, err := e.client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list key pairs: %w", err)
	}

	var keys []KeyPairInfo
	for _, kp := range result.KeyPairs {
		if kp.KeyName == nil {
			continue
		}

		createdBy := "user"
		if IsAwsJupyterKey(*kp.KeyName) {
			createdBy = "aws-jupyter"
		}

		keys = append(keys, KeyPairInfo{
			Name:        *kp.KeyName,
			Fingerprint: aws.ToString(kp.KeyFingerprint),
			Region:      e.region,
			CreatedBy:   createdBy,
		})
	}

	return keys, nil
}

// KeyPairExists checks if a key pair exists in AWS
func (e *EC2Client) KeyPairExists(ctx context.Context, keyName string) (bool, error) {
	_, err := e.client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
		KeyNames: []string{keyName},
	})
	if err != nil {
		// Check if it's a "not found" error - in AWS SDK v2, this is typically an API error
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "InvalidKeyPair.NotFound" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check key pair existence: %w", err)
	}
	return true, nil
}

// CreateKeyPair creates a new EC2 key pair
func (e *EC2Client) CreateKeyPair(ctx context.Context, keyName string) (*KeyPairInfo, error) {
	result, err := e.client.CreateKeyPair(ctx, &ec2.CreateKeyPairInput{
		KeyName: aws.String(keyName),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeKeyPair,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(keyName)},
					{Key: aws.String("CreatedBy"), Value: aws.String("aws-jupyter-cli")},
					{Key: aws.String("Purpose"), Value: aws.String("Jupyter Lab SSH access")},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create key pair: %w", err)
	}

	return &KeyPairInfo{
		Name:        *result.KeyName,
		Fingerprint: *result.KeyFingerprint,
		PrivateKey:  *result.KeyMaterial,
		Region:      e.region,
		CreatedBy:   "aws-jupyter",
	}, nil
}

// DeleteKeyPair removes a key pair from AWS
func (e *EC2Client) DeleteKeyPair(ctx context.Context, keyName string) error {
	_, err := e.client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete key pair: %w", err)
	}
	return nil
}

// GetOrCreateKeyPair implements the smart key pair strategy
func (e *EC2Client) GetOrCreateKeyPair(ctx context.Context, strategy KeyPairStrategy) (*KeyPairInfo, error) {
	keyName := strategy.GetDefaultKeyName()

	// If user specified a key, just verify it exists
	if strategy.UserSpecified != "" && !strategy.ForceCreate {
		exists, err := e.KeyPairExists(ctx, keyName)
		if err != nil {
			return nil, fmt.Errorf("failed to verify user-specified key pair: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("user-specified key pair '%s' does not exist in region %s", keyName, e.region)
		}

		return &KeyPairInfo{
			Name:      keyName,
			Region:    e.region,
			CreatedBy: "user",
		}, nil
	}

	// Check if our default key exists
	if strategy.PreferExisting && !strategy.ForceCreate {
		exists, err := e.KeyPairExists(ctx, keyName)
		if err != nil {
			return nil, fmt.Errorf("failed to check for existing key pair: %w", err)
		}
		if exists {
			// Get the existing key info
			keys, err := e.ListKeyPairs(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get existing key info: %w", err)
			}

			for _, key := range keys {
				if key.Name == keyName {
					return &key, nil
				}
			}
		}
	}

	// Create new key pair
	fmt.Printf("Creating new SSH key pair: %s\n", keyName)
	return e.CreateKeyPair(ctx, keyName)
}

// ListAwsJupyterKeys returns only key pairs created by aws-jupyter
func (e *EC2Client) ListAwsJupyterKeys(ctx context.Context) ([]KeyPairInfo, error) {
	allKeys, err := e.ListKeyPairs(ctx)
	if err != nil {
		return nil, err
	}

	var jupyterKeys []KeyPairInfo
	for _, key := range allKeys {
		if key.CreatedBy == "aws-jupyter" {
			jupyterKeys = append(jupyterKeys, key)
		}
	}

	return jupyterKeys, nil
}
